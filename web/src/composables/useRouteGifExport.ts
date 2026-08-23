import { computed, onUnmounted, ref } from "vue";
import { useLocalStorage } from "@vueuse/core";
import type { BBox, GeoJSONFeatureCollection } from "../lib/activity";
import { buildRouteAnimation, isMovieExportSupported, isNativeMp4Supported, type RouteGifProgress } from "../lib/routeGif";

const DEFAULT_EXPORT_SIZE = 768;
const DEFAULT_PREVIEW_SIZE = 420;
const DEFAULT_FRAME_DELAY_MS = 45;
const DEFAULT_ROUTE_COLOR = "#ff8c00";
const DEFAULT_FLASH_COLOR = "#ffffff";
const HEX_COLOR_PATTERN = /^#(?:[0-9a-fA-F]{6})$/;

interface RouteGifOptions {
  geoJSON: GeoJSONFeatureCollection | null;
  bbox: BBox;
  cityName: string;
  routeLabel: string;
  datasetName?: string | null;
}

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

const slugify = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");

const buildFileName = ({ cityName, routeLabel, datasetName }: RouteGifOptions, extension = "gif") => {
  const parts = [datasetName, routeLabel, cityName]
    .map((part) => (part ? slugify(part) : ""))
    .filter(Boolean);

  return `${parts.join("-") || "route-animation"}.${extension}`;
};

const downloadBlob = (blob: Blob, fileName: string) => {
  const href = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = href;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.setTimeout(() => URL.revokeObjectURL(href), 1000);
};

const normalizeRouteColor = (value: string) => {
  const normalized = value.trim().toLowerCase();
  return HEX_COLOR_PATTERN.test(normalized) ? normalized : DEFAULT_ROUTE_COLOR;
};

const normalizeFlashColor = (value: string) => {
  const normalized = value.trim().toLowerCase();
  return HEX_COLOR_PATTERN.test(normalized) ? normalized : DEFAULT_FLASH_COLOR;
};

export function useRouteGifExport() {
  const frameDelayMs = ref(DEFAULT_FRAME_DELAY_MS);
  const routeColor = useLocalStorage("map-route-color", DEFAULT_ROUTE_COLOR);
  const flashColor = useLocalStorage("map-flash-color", DEFAULT_FLASH_COLOR);
  const previewUrl = ref<string | null>(null);
  const isPreparingPreview = ref(false);
  const isDownloading = ref(false);
  const exportError = ref<string | null>(null);
  const exportStatus = ref<string | null>(null);
  const progress = ref<RouteGifProgress | null>(null);
  const progressMode = ref<"preview" | "download" | null>(null);
  const exportFormat = ref<"gif" | "webm" | "mp4">("gif");

  const isTranscodeAvailable = ref(false);

  fetch("/api/transcode/check")
    .then((res) => res.json())
    .then((data: any) => {
      isTranscodeAvailable.value = !!data.available;
    })
    .catch((err) => {
      console.error("Failed to check transcode availability", err);
    });
  const showCityName = ref(true);
  const cityFont = ref("serif");
  const cityPosition = ref("top-left");
  const cityNameOverlay = ref("");
  const showDistance = ref(true);
  const distancePosition = ref("bottom-right");
  const distanceUnit = useLocalStorage("map-distance-unit", "miles");
  const distanceFont = ref("monospace");
  const showDate = ref(true);
  const datePosition = ref("bottom-left");
  const dateFont = ref("serif");
  const dateFormat = useLocalStorage<"month-day-year" | "month-year">("map-date-format", "month-day-year");
  let previewBuildToken = 0;

  const revokePreviewUrl = () => {
    if (!previewUrl.value) {
      return;
    }

    URL.revokeObjectURL(previewUrl.value);
    previewUrl.value = null;
  };

  onUnmounted(() => {
    revokePreviewUrl();
  });

  const statusMessage = computed(() => {
    const formatLabel = exportFormat.value === "gif" ? "GIF" : "movie";
    if (progress.value && progressMode.value === "preview") {
      return `Rendering preview ${progress.value.completedRoutes} of ${progress.value.totalRoutes} rides…`;
    }

    if (progress.value && progressMode.value === "download") {
      return `Encoding download ${progress.value.completedRoutes} of ${progress.value.totalRoutes} rides…`;
    }

    return exportStatus.value;
  });

  const updateFrameDelayMs = (value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }

    frameDelayMs.value = clamp(Math.round(value), 20, 5000);
  };

  const updateRouteColor = (value: string) => {
    routeColor.value = normalizeRouteColor(value);
  };

  const updateFlashColor = (value: string) => {
    flashColor.value = normalizeFlashColor(value);
  };

  const resetState = () => {
    previewBuildToken += 1;
    frameDelayMs.value = DEFAULT_FRAME_DELAY_MS;
    routeColor.value = DEFAULT_ROUTE_COLOR;
    flashColor.value = DEFAULT_FLASH_COLOR;
    exportFormat.value = "gif";
    isPreparingPreview.value = false;
    isDownloading.value = false;
    exportError.value = null;
    exportStatus.value = null;
    progress.value = null;
    progressMode.value = null;
    revokePreviewUrl();
    showCityName.value = true;
    cityFont.value = "serif";
    cityPosition.value = "top-left";
    cityNameOverlay.value = "";
    showDistance.value = true;
    distancePosition.value = "bottom-right";
    distanceUnit.value = "miles";
    distanceFont.value = "monospace";
    showDate.value = true;
    datePosition.value = "bottom-left";
    dateFont.value = "serif";
    dateFormat.value = "month-day-year";
  };

  const validateRoutes = (geoJSON: GeoJSONFeatureCollection | null) => {
    if (!geoJSON || geoJSON.features.length === 0) {
      exportError.value = "Load at least one ride before preparing an animated export.";
      exportStatus.value = null;
      revokePreviewUrl();
      return false;
    }

    return true;
  };

  const preparePreview = async (options: RouteGifOptions) => {
    if (!validateRoutes(options.geoJSON)) {
      return false;
    }

    const currentToken = ++previewBuildToken;
    isPreparingPreview.value = true;
    exportError.value = null;
    exportStatus.value = null;
    progressMode.value = "preview";
    progress.value = { completedRoutes: 0, totalRoutes: options.geoJSON.features.length };

    try {
      const blob = await buildRouteAnimation({
        geoJSON: options.geoJSON,
        bbox: options.bbox,
        size: DEFAULT_PREVIEW_SIZE,
        frameDelayMs: frameDelayMs.value,
        routeColor: routeColor.value,
        flashColor: flashColor.value,
        cityName: cityNameOverlay.value || options.cityName,
        showCityName: showCityName.value,
        cityFont: cityFont.value,
        cityPosition: cityPosition.value,
        showDistance: showDistance.value,
        distancePosition: distancePosition.value,
        distanceUnit: distanceUnit.value,
        distanceFont: distanceFont.value,
        showDate: showDate.value,
        datePosition: datePosition.value,
        dateFont: dateFont.value,
        dateFormat: dateFormat.value,
        format: (exportFormat.value === "mp4" && !isNativeMp4Supported()) ? "webm" : exportFormat.value,
        onProgress: (nextProgress) => {
          if (currentToken === previewBuildToken) {
            progress.value = nextProgress;
          }
        },
      });

      if (currentToken !== previewBuildToken) {
        return false;
      }

      revokePreviewUrl();
      previewUrl.value = URL.createObjectURL(blob);
      exportStatus.value = `Preview ready for ${options.geoJSON.features.length} rides.`;
      return true;
    } catch (error) {
      if (currentToken !== previewBuildToken) {
        return false;
      }

      console.error(error);
      exportError.value =
        error instanceof Error ? error.message : "Failed to prepare the animated preview.";
      return false;
    } finally {
      if (currentToken === previewBuildToken) {
        isPreparingPreview.value = false;
        progress.value = null;
        progressMode.value = null;
      }
    }
  };

  const downloadAnimation = async (options: RouteGifOptions) => {
    if (!validateRoutes(options.geoJSON)) {
      return false;
    }

    isDownloading.value = true;
    exportError.value = null;
    exportStatus.value = null;
    progressMode.value = "download";
    progress.value = { completedRoutes: 0, totalRoutes: options.geoJSON.features.length };

    try {
      const requestedFormat = exportFormat.value;
      const buildFormat = (requestedFormat === "mp4" && !isNativeMp4Supported()) ? "webm" : requestedFormat;

      let blob = await buildRouteAnimation({
        geoJSON: options.geoJSON,
        bbox: options.bbox,
        size: DEFAULT_EXPORT_SIZE,
        frameDelayMs: frameDelayMs.value,
        routeColor: routeColor.value,
        flashColor: flashColor.value,
        cityName: cityNameOverlay.value || options.cityName,
        showCityName: showCityName.value,
        cityFont: cityFont.value,
        cityPosition: cityPosition.value,
        showDistance: showDistance.value,
        distancePosition: distancePosition.value,
        distanceUnit: distanceUnit.value,
        distanceFont: distanceFont.value,
        showDate: showDate.value,
        datePosition: datePosition.value,
        dateFont: dateFont.value,
        dateFormat: dateFormat.value,
        format: buildFormat,
        onProgress: (nextProgress) => {
          progress.value = nextProgress;
        },
      });

      if (requestedFormat === "mp4" && !isNativeMp4Supported()) {
        exportStatus.value = "Converting video to MP4 on server…";
        const response = await fetch("/api/transcode", {
          method: "POST",
          headers: {
            "Content-Type": blob.type || "video/webm",
          },
          body: blob,
        });

        if (!response.ok) {
          const errMsg = await response.text();
          throw new Error(`Transcode failed: ${errMsg || response.statusText}`);
        }

        blob = await response.blob();
      }

      const extension = blob.type.includes("mp4") ? "mp4" : blob.type.includes("webm") ? "webm" : "gif";
      downloadBlob(blob, buildFileName(options, extension));
      exportStatus.value = `Downloaded ${options.geoJSON.features.length}-ride ${
        requestedFormat === "gif" ? "GIF" : requestedFormat === "mp4" ? "MP4 Video" : "WebM Video"
      }.`;
      return true;
    } catch (error) {
      console.error(error);
      exportError.value =
        error instanceof Error ? error.message : "Failed to export the animation.";
      return false;
    } finally {
      isDownloading.value = false;
      progress.value = null;
      progressMode.value = null;
    }
  };

  return {
    frameDelayMs,
    routeColor,
    flashColor,
    exportFormat,
    isMovieExportSupported: isMovieExportSupported(),
    isNativeMp4Supported: isNativeMp4Supported(),
    isTranscodeAvailable,
    previewUrl,
    isPreparingPreview,
    isDownloading,
    exportError,
    statusMessage,
    showCityName,
    cityFont,
    cityPosition,
    cityNameOverlay,
    showDistance,
    distancePosition,
    distanceUnit,
    distanceFont,
    showDate,
    datePosition,
    dateFont,
    dateFormat,
    updateFrameDelayMs,
    updateRouteColor,
    updateFlashColor,
    preparePreview,
    downloadAnimation,
    resetState,
  };
}
