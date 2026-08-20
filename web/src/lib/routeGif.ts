import { GIFEncoder } from "gifenc";
import type { BBox, GeoJSONFeature, GeoJSONFeatureCollection } from "./activity";

const DEFAULT_EXPORT_SIZE = 768;
const DEFAULT_FRAME_DELAY_MS = 45;
const DEFAULT_FINAL_FRAME_DELAY_MS = 1200;
const DEFAULT_PADDING = 24;
const DEFAULT_ROUTE_COLOR = "#ff8c00";
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
  onProgress?: (progress: RouteGifProgress) => void;
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
    return normalized.split("").map((part) => Number.parseInt(part.repeat(2), 16)) as Coordinate & [number];
  }

  return [255, 140, 0];
};

const getStringProperty = (
  feature: GeoJSONFeature,
  primaryKey: string,
  fallbackKey?: string,
) => {
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

const createPalette = (routeColor: [number, number, number]) => {
  const palette: number[][] = [[0, 0, 0]];

  for (let index = 1; index <= SHADE_COUNT; index += 1) {
    const ratio = index / SHADE_COUNT;
    palette.push([
      Math.round(routeColor[0] * ratio),
      Math.round(routeColor[1] * ratio),
      Math.round(routeColor[2] * ratio),
    ]);
  }

  return palette;
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

const drawRoute = (ctx: CanvasRenderingContext2D, feature: GeoJSONFeature, projector: Projector) => {
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

const createIndexedFrame = (
  imageData: Uint8ClampedArray,
  routeColor: [number, number, number],
) => {
  const frame = new Uint8Array(imageData.length / 4);
  const dominantChannel = Math.max(routeColor[0], routeColor[1], routeColor[2], 1);

  for (let pixelIndex = 0; pixelIndex < frame.length; pixelIndex += 1) {
    const dataOffset = pixelIndex * 4;
    const red = imageData[dataOffset];
    const green = imageData[dataOffset + 1];
    const blue = imageData[dataOffset + 2];

    if (red === 0 && green === 0 && blue === 0) {
      frame[pixelIndex] = 0;
      continue;
    }

    const shade = Math.max(red, green, blue) / dominantChannel;
    frame[pixelIndex] = clamp(Math.round(shade * SHADE_COUNT), 1, SHADE_COUNT);
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
  onProgress,
}: BuildRouteAnimationGifOptions) {
  if (typeof document === "undefined") {
    throw new Error("Animated GIF export is only available in the browser.");
  }

  const routes = geoJSON.features.filter(hasRenderableGeometry).slice().sort(compareChronologically);
  if (routes.length === 0) {
    throw new Error("No ride geometry is available for GIF export.");
  }

  const squareSize = normalizeSize(size);
  const perFrameDelay = normalizeDelay(frameDelayMs, DEFAULT_FRAME_DELAY_MS);
  const endingDelay = normalizeDelay(finalFrameDelayMs, DEFAULT_FINAL_FRAME_DELAY_MS);
  const routeRgb = parseHexColor(routeColor);
  const palette = createPalette(routeRgb);
  const projector = createProjector(bbox, squareSize);

  const canvas = document.createElement("canvas");
  canvas.width = squareSize;
  canvas.height = squareSize;

  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  if (!ctx) {
    throw new Error("Canvas rendering is not available in this browser.");
  }

  ctx.fillStyle = "#000000";
  ctx.fillRect(0, 0, squareSize, squareSize);
  ctx.lineCap = "round";
  ctx.lineJoin = "round";
  ctx.lineWidth = Math.max(1.75, squareSize / 320);
  ctx.strokeStyle = `rgb(${routeRgb[0]}, ${routeRgb[1]}, ${routeRgb[2]})`;

  ctx.save();
  ctx.beginPath();
  ctx.rect(projector.clipX, projector.clipY, projector.clipWidth, projector.clipHeight);
  ctx.clip();

  const gif = GIFEncoder();

  try {
    for (let index = 0; index < routes.length; index += 1) {
      drawRoute(ctx, routes[index], projector);

      const frame = createIndexedFrame(ctx.getImageData(0, 0, squareSize, squareSize).data, routeRgb);
      gif.writeFrame(frame, squareSize, squareSize, {
        palette: index === 0 ? palette : undefined,
        repeat: index === 0 ? 0 : undefined,
        delay: index === routes.length - 1 ? endingDelay : perFrameDelay,
      });

      onProgress?.({ completedRoutes: index + 1, totalRoutes: routes.length });
      if ((index + 1) % 10 === 0 || index === routes.length - 1) {
        await nextPaint();
      }
    }
  } finally {
    ctx.restore();
  }

  gif.finish();
  return new Blob([gif.bytesView()], { type: "image/gif" });
}
