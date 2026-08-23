import { GIFEncoder } from "gifenc";
import type { BBox, GeoJSONFeature, GeoJSONFeatureCollection } from "./activity";

const DEFAULT_EXPORT_SIZE = 768;
const DEFAULT_FRAME_DELAY_MS = 45;
const DEFAULT_FINAL_FRAME_DELAY_MS = 1200;
const DEFAULT_PADDING = 24;
const DEFAULT_ROUTE_COLOR = "#ff8c00";
const DEFAULT_FLASH_COLOR = "#ffffff";
const SHADE_COUNT = 24;
const MAX_MERCATOR_LATITUDE = 85.05112878;

type Coordinate = [number, number];

type Segment = Coordinate[];

export interface RouteGifProgress {
  completedRoutes: number;
  totalRoutes: number;
}

export interface BuildRouteAnimationGifOptions {
  geoJSON: GeoJSONFeatureCollection;
  bbox: BBox;
  size?: number;
  frameDelayMs?: number;
  finalFrameDelayMs?: number;
  routeColor?: string;
  flashColor?: string;
  onProgress?: (progress: RouteGifProgress) => void;
  cityName?: string;
  showCityName?: boolean;
  cityFont?: string;
  cityPosition?: string;
  showDistance?: boolean;
  distancePosition?: string;
  distanceUnit?: string;
  distanceFont?: string;
  showDate?: boolean;
  datePosition?: string;
  dateFont?: string;
  dateFormat?: "month-day-year" | "month-year";
}

interface Projector {
  clipX: number;
  clipY: number;
  clipWidth: number;
  clipHeight: number;
  project: (coordinate: Coordinate) => Coordinate;
}

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

const clampLatitude = (latitude: number) =>
  clamp(latitude, -MAX_MERCATOR_LATITUDE, MAX_MERCATOR_LATITUDE);

const mercatorX = (longitude: number) => (longitude * Math.PI) / 180;

const mercatorY = (latitude: number) => {
  const radians = (clampLatitude(latitude) * Math.PI) / 180;
  return Math.log(Math.tan(Math.PI / 4 + radians / 2));
};

const nextPaint = () =>
  new Promise<void>((resolve) => {
    window.requestAnimationFrame(() => resolve());
  });

const normalizeSize = (value: number | undefined) => {
  if (!Number.isFinite(value)) {
    return DEFAULT_EXPORT_SIZE;
  }

  return clamp(Math.round(value ?? DEFAULT_EXPORT_SIZE), 256, 1400);
};

const normalizeDelay = (value: number | undefined, fallback: number) => {
  if (!Number.isFinite(value)) {
    return fallback;
  }

  return clamp(Math.round(value ?? fallback), 20, 5000);
};

const parseHexColor = (value: string | undefined): Coordinate & [number] => {
  const trimmed = value?.trim() ?? DEFAULT_ROUTE_COLOR;
  const normalized = trimmed.startsWith("#") ? trimmed.slice(1) : trimmed;

  if (/^[0-9a-fA-F]{6}$/.test(normalized)) {
    return [
      Number.parseInt(normalized.slice(0, 2), 16),
      Number.parseInt(normalized.slice(2, 4), 16),
      Number.parseInt(normalized.slice(4, 6), 16),
    ];
  }

  if (/^[0-9a-fA-F]{3}$/.test(normalized)) {
    return normalized.split("").map((part) => Number.parseInt(part.repeat(2), 16)) as Coordinate &
      [number];
  }

  return [255, 140, 0];
};

const getStringProperty = (feature: GeoJSONFeature, primaryKey: string, fallbackKey?: string) => {
  const value =
    feature.properties?.[primaryKey] ??
    (fallbackKey ? feature.properties?.[fallbackKey] : undefined);

  if (value === undefined || value === null || value === "") {
    return null;
  }

  return String(value);
};

const getRouteId = (feature: GeoJSONFeature) =>
  getStringProperty(feature, "route_id", "activity_id") ?? "unknown-route";

const getRouteDate = (feature: GeoJSONFeature) =>
  getStringProperty(feature, "route_date", "activity_date");

const formatDate = (dateStr: string | undefined, format?: "month-day-year" | "month-year"): string => {
  if (!dateStr) return "";
  const match = dateStr.match(/^(\d{4})-(\d{2})-(\d{2})/);
  const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  if (match) {
    const year = match[1];
    const monthIndex = parseInt(match[2], 10) - 1;
    const day = parseInt(match[3], 10);
    if (monthIndex >= 0 && monthIndex < 12) {
      if (format === "month-year") {
        return `${months[monthIndex]}, ${year}`;
      }
      return `${months[monthIndex]} ${day}, ${year}`;
    }
  }
  const date = new Date(dateStr);
  if (!isNaN(date.getTime())) {
    const hasTime = dateStr.includes("T") || dateStr.includes(" ");
    const month = hasTime ? date.getMonth() : date.getUTCMonth();
    const day = hasTime ? date.getDate() : date.getUTCDate();
    const year = hasTime ? date.getFullYear() : date.getUTCFullYear();
    if (format === "month-year") {
      return `${months[month]}, ${year}`;
    }
    return `${months[month]} ${day}, ${year}`;
  }
  return dateStr;
};

const getRouteTimestamp = (feature: GeoJSONFeature) => {
  const routeDate = getRouteDate(feature);
  if (!routeDate) {
    return null;
  }

  const timestamp = Date.parse(routeDate);
  return Number.isNaN(timestamp) ? null : timestamp;
};

const compareChronologically = (left: GeoJSONFeature, right: GeoJSONFeature) => {
  const leftTimestamp = getRouteTimestamp(left);
  const rightTimestamp = getRouteTimestamp(right);

  if (leftTimestamp !== null && rightTimestamp !== null && leftTimestamp !== rightTimestamp) {
    return leftTimestamp - rightTimestamp;
  }

  if (leftTimestamp !== null && rightTimestamp === null) {
    return -1;
  }

  if (leftTimestamp === null && rightTimestamp !== null) {
    return 1;
  }

  const leftDate = getRouteDate(left) ?? "";
  const rightDate = getRouteDate(right) ?? "";
  const dateCompare = leftDate.localeCompare(rightDate);
  if (dateCompare !== 0) {
    return dateCompare;
  }

  return getRouteId(left).localeCompare(getRouteId(right), undefined, { numeric: true });
};

const isCoordinate = (value: unknown): value is Coordinate =>
  Array.isArray(value) &&
  value.length >= 2 &&
  typeof value[0] === "number" &&
  Number.isFinite(value[0]) &&
  typeof value[1] === "number" &&
  Number.isFinite(value[1]);

const collectSegments = (feature: GeoJSONFeature): Segment[] => {
  const geometry = feature.geometry;
  if (!geometry) {
    return [];
  }

  if (geometry.type === "LineString" && Array.isArray(geometry.coordinates)) {
    const segment = geometry.coordinates.filter(isCoordinate);
    return segment.length > 1 ? [segment] : [];
  }

  if (geometry.type === "MultiLineString" && Array.isArray(geometry.coordinates)) {
    return geometry.coordinates
      .map((segment) => (Array.isArray(segment) ? segment.filter(isCoordinate) : []))
      .filter((segment): segment is Segment => segment.length > 1);
  }

  return [];
};

const hasRenderableGeometry = (feature: GeoJSONFeature) => collectSegments(feature).length > 0;

const createPalette = (
  routeColor: [number, number, number],
  flashColor: [number, number, number],
  includeWhite = false,
) => {
  const palette: number[][] = [[0, 0, 0]];

  for (let index = 1; index <= SHADE_COUNT; index += 1) {
    const ratio = index / SHADE_COUNT;
    palette.push([
      Math.round(routeColor[0] * ratio),
      Math.round(routeColor[1] * ratio),
      Math.round(routeColor[2] * ratio),
    ]);
  }

  for (let index = 1; index <= SHADE_COUNT; index += 1) {
    const ratio = index / SHADE_COUNT;
    palette.push([
      Math.round(flashColor[0] * ratio),
      Math.round(flashColor[1] * ratio),
      Math.round(flashColor[2] * ratio),
    ]);
  }

  const flashIsWhite = flashColor[0] === 255 && flashColor[1] === 255 && flashColor[2] === 255;
  const needTertiaryWhite = includeWhite && !flashIsWhite;

  if (needTertiaryWhite) {
    for (let index = 1; index <= SHADE_COUNT; index += 1) {
      const ratio = index / SHADE_COUNT;
      const val = Math.round(255 * ratio);
      palette.push([val, val, val]);
    }
  }

  return { palette, needTertiaryWhite };
};

const createProjector = (bbox: BBox, size: number): Projector => {
  const minX = mercatorX(bbox[0]);
  const maxX = mercatorX(bbox[2]);
  const minY = mercatorY(bbox[1]);
  const maxY = mercatorY(bbox[3]);

  const spanX = Math.max(maxX - minX, Number.EPSILON);
  const spanY = Math.max(maxY - minY, Number.EPSILON);
  const drawableSize = Math.max(size - DEFAULT_PADDING * 2, 1);
  const scale = drawableSize / Math.max(spanX, spanY);
  const clipWidth = spanX * scale;
  const clipHeight = spanY * scale;
  const clipX = (size - clipWidth) / 2;
  const clipY = (size - clipHeight) / 2;

  return {
    clipX,
    clipY,
    clipWidth,
    clipHeight,
    project: ([longitude, latitude]) => [
      clipX + (mercatorX(longitude) - minX) * scale,
      clipY + (maxY - mercatorY(latitude)) * scale,
    ],
  };
};

const drawRoute = (
  ctx: CanvasRenderingContext2D,
  feature: GeoJSONFeature,
  projector: Projector,
) => {
  const segments = collectSegments(feature);
  if (segments.length === 0) {
    return;
  }

  ctx.beginPath();
  for (const segment of segments) {
    const [firstCoordinate, ...remainingCoordinates] = segment;
    const [firstX, firstY] = projector.project(firstCoordinate);
    ctx.moveTo(firstX, firstY);

    for (const coordinate of remainingCoordinates) {
      const [x, y] = projector.project(coordinate);
      ctx.lineTo(x, y);
    }
  }
  ctx.stroke();
};

const getBestScaleAndError = (r: number, g: number, b: number, base: [number, number, number]) => {
  const dot = r * base[0] + g * base[1] + b * base[2];
  const denom = base[0] * base[0] + base[1] * base[1] + base[2] * base[2];
  if (denom === 0) return { scale: 0, error: Infinity };
  const scale = clamp(dot / denom, 0, 1);
  const dr = r - scale * base[0];
  const dg = g - scale * base[1];
  const db = b - scale * base[2];
  const error = dr * dr + dg * dg + db * db;
  return { scale, error };
};

const createIndexedFrame = (
  imageData: Uint8ClampedArray,
  routeColor: [number, number, number],
  flashColor: [number, number, number],
  needTertiaryWhite: boolean,
) => {
  const frame = new Uint8Array(imageData.length / 4);

  for (let pixelIndex = 0; pixelIndex < frame.length; pixelIndex += 1) {
    const dataOffset = pixelIndex * 4;
    const r = imageData[dataOffset];
    const g = imageData[dataOffset + 1];
    const b = imageData[dataOffset + 2];

    if (r === 0 && g === 0 && b === 0) {
      frame[pixelIndex] = 0;
      continue;
    }

    const routeRes = getBestScaleAndError(r, g, b, routeColor);
    const flashRes = getBestScaleAndError(r, g, b, flashColor);

    let minError = routeRes.error;
    let bestColorSet = 0; // 0 = route, 1 = flash, 2 = white
    let bestScale = routeRes.scale;

    if (flashRes.error < minError) {
      minError = flashRes.error;
      bestColorSet = 1;
      bestScale = flashRes.scale;
    }

    if (needTertiaryWhite) {
      const whiteRes = getBestScaleAndError(r, g, b, [255, 255, 255]);
      if (whiteRes.error < minError) {
        minError = whiteRes.error;
        bestColorSet = 2;
        bestScale = whiteRes.scale;
      }
    }

    const shadeVal = clamp(Math.round(bestScale * SHADE_COUNT), 1, SHADE_COUNT);
    if (bestColorSet === 0) {
      frame[pixelIndex] = shadeVal;
    } else if (bestColorSet === 1) {
      frame[pixelIndex] = SHADE_COUNT + shadeVal;
    } else {
      frame[pixelIndex] = 2 * SHADE_COUNT + shadeVal;
    }
  }

  return frame;
};

export async function buildRouteAnimationGif({
  geoJSON,
  bbox,
  size,
  frameDelayMs,
  finalFrameDelayMs,
  routeColor,
  flashColor,
  onProgress,
  cityName,
  showCityName = false,
  cityFont = "serif",
  cityPosition = "bottom-left",
  showDistance = false,
  distancePosition = "bottom-right",
  distanceUnit = "km",
  distanceFont = "monospace",
  showDate = false,
  datePosition = "bottom-right",
  dateFont = "monospace",
  dateFormat = "month-day-year",
}: BuildRouteAnimationGifOptions) {
  if (typeof document === "undefined") {
    throw new Error("Animated GIF export is only available in the browser.");
  }

  const routes = geoJSON.features
    .filter(hasRenderableGeometry)
    .slice()
    .sort(compareChronologically);
  if (routes.length === 0) {
    throw new Error("No ride geometry is available for GIF export.");
  }

  const distanceFactor = distanceUnit === "miles" ? 0.62137119 : 1.0;
  const unitLabel = distanceUnit === "miles" ? " mi" : " km";

  const getRouteDistance = (feature: GeoJSONFeature): number => {
    const dist = feature.properties?.distance;
    let rawDist = 0;
    if (typeof dist === "number") {
      rawDist = dist;
    } else if (typeof dist === "string") {
      const parsed = parseFloat(dist);
      rawDist = isNaN(parsed) ? 0 : parsed;
    }
    return rawDist * distanceFactor;
  };

  const totalAccumulatedDistance = routes.reduce((sum, r) => sum + getRouteDistance(r), 0);
  const finalTotalRounded = Math.round(totalAccumulatedDistance);
  const padLength = Math.max(4, String(finalTotalRounded).length);
  const squareSize = normalizeSize(size);
  const perFrameDelay = normalizeDelay(frameDelayMs, DEFAULT_FRAME_DELAY_MS);
  const endingDelay = normalizeDelay(finalFrameDelayMs, DEFAULT_FINAL_FRAME_DELAY_MS);
  const routeRgb = parseHexColor(routeColor);
  const flashRgb = parseHexColor(flashColor ?? DEFAULT_FLASH_COLOR);
  const includeWhite = !!((showCityName && cityName) || showDistance || showDate);
  const { palette, needTertiaryWhite } = createPalette(routeRgb, flashRgb, includeWhite);
  const projector = createProjector(bbox, squareSize);

  const drawOverlays = (
    context2d: CanvasRenderingContext2D,
    currentDistance: number,
    currentDateStr: string | undefined
  ) => {
    const hasCity = !!(showCityName && cityName);
    const hasDist = !!showDistance;
    const hasDate = !!(showDate && currentDateStr);

    if (!hasCity && !hasDist && !hasDate) {
      return;
    }

    context2d.save();
    const fontSize = Math.max(16, Math.round(squareSize * 0.05));
    const padding = Math.max(16, Math.round(squareSize * 0.05));
    const lineSpacing = fontSize * 1.25;

    const getFontString = (style: string) => {
      let family = "Georgia, serif";
      if (style === "sans-serif") {
        family = "system-ui, -apple-system, sans-serif";
      } else if (style === "monospace") {
        family = `ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace`;
      }
      return `bold ${fontSize}px ${family}`;
    };

    context2d.fillStyle = "#ffffff";
    context2d.textBaseline = "alphabetic";

    const getCoords = (pos: string) => {
      let x = padding;
      let y = padding;
      let align: CanvasTextAlign = "left";

      if (pos === "bottom-left") {
        x = padding;
        y = squareSize - padding;
        align = "left";
      } else if (pos === "bottom-right") {
        x = squareSize - padding;
        y = squareSize - padding;
        align = "right";
      } else if (pos === "top-left") {
        x = padding;
        y = padding + fontSize;
        align = "left";
      } else if (pos === "top-right") {
        x = squareSize - padding;
        y = padding + fontSize;
        align = "right";
      }
      return { x, y, align };
    };

    const itemsByPosition: Record<string, Array<{ text: string; fontStyle: string }>> = {
      "bottom-left": [],
      "bottom-right": [],
      "top-left": [],
      "top-right": [],
    };

    if (hasCity) {
      itemsByPosition[cityPosition].push({
        text: cityName || "",
        fontStyle: cityFont,
      });
    }

    if (hasDate) {
      itemsByPosition[datePosition].push({
        text: formatDate(currentDateStr, dateFormat),
        fontStyle: dateFont,
      });
    }

    if (hasDist) {
      const rounded = Math.round(currentDistance);
      const distStr = String(rounded).padStart(padLength, "0") + unitLabel;
      itemsByPosition[distancePosition].push({
        text: distStr,
        fontStyle: distanceFont,
      });
    }

    const positions = ["bottom-left", "bottom-right", "top-left", "top-right"];
    for (const pos of positions) {
      const items = itemsByPosition[pos];
      if (items.length === 0) {
        continue;
      }

      const { x, y, align } = getCoords(pos);
      context2d.textAlign = align;

      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        let itemY = y;
        if (pos.startsWith("bottom-")) {
          itemY = y - (items.length - 1 - i) * lineSpacing;
        } else {
          itemY = y + i * lineSpacing;
        }

        context2d.font = getFontString(item.fontStyle);
        context2d.fillText(item.text, x, itemY);
      }
    }

    context2d.restore();
  };

  // Separate canvas to accumulate only route paths (without text)
  const routesCanvas = document.createElement("canvas");
  routesCanvas.width = squareSize;
  routesCanvas.height = squareSize;
  const routesCtx = routesCanvas.getContext("2d");
  if (!routesCtx) {
    throw new Error("Canvas rendering is not available in this browser.");
  }

  routesCtx.fillStyle = "#000000";
  routesCtx.fillRect(0, 0, squareSize, squareSize);
  routesCtx.lineCap = "round";
  routesCtx.lineJoin = "round";
  routesCtx.lineWidth = Math.max(1.75, squareSize / 320);
  routesCtx.strokeStyle = `rgb(${routeRgb[0]}, ${routeRgb[1]}, ${routeRgb[2]})`;

  // Main canvas to draw the final composite frame (routes + text)
  const canvas = document.createElement("canvas");
  canvas.width = squareSize;
  canvas.height = squareSize;
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  if (!ctx) {
    throw new Error("Canvas rendering is not available in this browser.");
  }

  ctx.lineCap = "round";
  ctx.lineJoin = "round";
  ctx.lineWidth = Math.max(1.75, squareSize / 320);
  ctx.strokeStyle = `rgb(${flashRgb[0]}, ${flashRgb[1]}, ${flashRgb[2]})`;

  const gif = GIFEncoder();

  try {
    const routeDistances = routes.map(getRouteDistance);
    const distancePrefixSums = [0];
    let runningSum = 0;
    for (let i = 0; i < routeDistances.length; i++) {
      runningSum += routeDistances[i];
      distancePrefixSums.push(runningSum);
    }

    for (let index = 0; index < routes.length; index += 1) {
      if (index > 0) {
        routesCtx.save();
        routesCtx.beginPath();
        routesCtx.rect(projector.clipX, projector.clipY, projector.clipWidth, projector.clipHeight);
        routesCtx.clip();
        drawRoute(routesCtx, routes[index - 1], projector);
        routesCtx.restore();
      }

      // Clear the composite canvas and copy the accumulated routes
      ctx.fillStyle = "#000000";
      ctx.fillRect(0, 0, squareSize, squareSize);
      ctx.drawImage(routesCanvas, 0, 0);

      // Draw current route in flash color on the composite canvas
      ctx.save();
      ctx.beginPath();
      ctx.rect(projector.clipX, projector.clipY, projector.clipWidth, projector.clipHeight);
      ctx.clip();
      drawRoute(ctx, routes[index], projector);
      ctx.restore();

      // Draw city name, distance, and date overlays
      drawOverlays(ctx, distancePrefixSums[index], getRouteDate(routes[index]));

      const frame = createIndexedFrame(
        ctx.getImageData(0, 0, squareSize, squareSize).data,
        routeRgb,
        flashRgb,
        needTertiaryWhite,
      );
      gif.writeFrame(frame, squareSize, squareSize, {
        palette: index === 0 ? palette : undefined,
        repeat: index === 0 ? 0 : undefined,
        delay: perFrameDelay,
      });

      onProgress?.({ completedRoutes: index + 1, totalRoutes: routes.length });
      if ((index + 1) % 10 === 0 || index === routes.length - 1) {
        await nextPaint();
      }
    }

    // Draw the final route in route color onto routesCanvas
    routesCtx.save();
    routesCtx.beginPath();
    routesCtx.rect(projector.clipX, projector.clipY, projector.clipWidth, projector.clipHeight);
    routesCtx.clip();
    drawRoute(routesCtx, routes[routes.length - 1], projector);
    routesCtx.restore();

    // Copy all final routes onto composite canvas
    ctx.fillStyle = "#000000";
    ctx.fillRect(0, 0, squareSize, squareSize);
    ctx.drawImage(routesCanvas, 0, 0);

    // Draw overlays with the final total accumulated distance and date
    drawOverlays(ctx, totalAccumulatedDistance, getRouteDate(routes[routes.length - 1]));

    const finalFrame = createIndexedFrame(
      ctx.getImageData(0, 0, squareSize, squareSize).data,
      routeRgb,
      flashRgb,
      needTertiaryWhite,
    );
    gif.writeFrame(finalFrame, squareSize, squareSize, {
      delay: endingDelay,
    });
  } finally {
    // Clean context state is handled per-frame
  }

  gif.finish();
  return new Blob([gif.bytesView()], { type: "image/gif" });
}
