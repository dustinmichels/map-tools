<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import AreaSelectionCard from "./AreaSelectionCard.vue";
import DatasetUploadCard from "./DatasetUploadCard.vue";
import FlowStepper from "./FlowStepper.vue";
import RouteGifExportCard from "./RouteGifExportCard.vue";
import MapView from "../MapView.vue";
import UploadedDatasetList from "../uploads/UploadedDatasetList.vue";
import { useActivityDataset } from "../../composables/useActivityDataset";
import { useRouteGifExport } from "../../composables/useRouteGifExport";
import { useUploadedDatasets } from "../../composables/useUploadedDatasets";
import {
  DEFAULT_BOSTON_BBOX,
  DEFAULT_BOSTON_CENTER,
  type BBox,
  type GeoJSONFeatureCollection,
  type GeometryMode,
  type LngLat,
  type RouteLayer,
  type SelectedCity,
  formatCityName,
} from "../../lib/activity";

const props = withDefaults(
  defineProps<{
    toolTitle?: string;
    routeId?: string;
  }>(),
  {
    toolTitle: "Lightning Map",
    routeId: "lightning-map",
  },
);

const steps = [
  { number: 1, label: "Upload" },
  { number: 2, label: "Area" },
  { number: 3, label: "Map" },
  { number: 4, label: "Prepare Export" },
];

const currentStep = ref(1);
const cityName = ref("Boston, MA, USA");
const bbox = ref<BBox>([...DEFAULT_BOSTON_BBOX]);
const center = ref<LngLat>([...DEFAULT_BOSTON_CENTER]);
const geometryMode = ref<GeometryMode>("simplified");
const dataset = useActivityDataset();
const uploadLibrary = useUploadedDatasets();
const {
  frameDelayMs,
  routeColor,
  flashColor,
  exportFormat,
  isMovieExportSupported,
  isNativeMp4Supported,
  isTranscodeAvailable,
  previewUrl,
  previewRouteCount,
  previewTargetCount,
  previewProgressCount,
  isPreparingPreview,
  isDownloading,
  exportError,
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
} = useRouteGifExport();

watch(
  cityName,
  (newCity) => {
    cityNameOverlay.value = formatCityName(newCity);
  },
  { immediate: true },
);

const AREA_PREVIEW_DELAY_MS = 300;
const AREA_PREVIEW_ROUTE_LIMIT = 200;
const AREA_PREVIEW_ROUTE_COLOR = "#a1a1aa";

const previewViewportBBox = ref<BBox | null>(null);
const previewGeoJSON = ref<GeoJSONFeatureCollection | null>(null);
const isLoadingAreaPreview = ref(false);

const previewRoutes = computed<RouteLayer[]>(() => {
  const geoJSON = previewGeoJSON.value;
  if (!geoJSON) {
    return [];
  }

  return [
    {
      id: `${props.routeId}-preview`,
      label: `${props.toolTitle} preview`,
      color: AREA_PREVIEW_ROUTE_COLOR,
      data: geoJSON,
      opacity: 0.35,
      width: 1.5,
      interactive: false,
    },
  ];
});

const displayRoutes = computed<RouteLayer[]>(() => [
  {
    id: props.routeId,
    label: props.toolTitle,
    color: routeColor.value,
    data: dataset.activitiesGeoJSON.value,
  },
]);
const availableRouteCount = computed(() => dataset.activitiesGeoJSON.value?.features.length ?? 0);
const hasAvailableRoutes = computed(() => availableRouteCount.value > 0);
const usingSimplifiedGeometry = computed(() => geometryMode.value === "simplified");
const showSimplifyPrompt = computed(
  () =>
    usingSimplifiedGeometry.value &&
    dataset.usingExistingDataset.value &&
    dataset.activeDataset.value !== null &&
    !dataset.activeDataset.value.hasSimplified,
);
const geometryModeDescription = computed(() =>
  usingSimplifiedGeometry.value
    ? "Simplified geometry removes extra points for faster redraws."
    : "Original geometry keeps every recorded point from the saved dataset.",
);

const handleSelectCity = (payload: SelectedCity) => {
  cityName.value = payload.name;
  bbox.value = payload.bbox;
  center.value = [payload.lon, payload.lat];
};

const submitSelectedArchive = async () => {
  const upload = await dataset.submitZip();
  if (upload) {
    await uploadLibrary.loadUploads();
  }
};

const simplifySelectedUpload = async () => {
  const activeDataset = dataset.activeDataset.value;
  if (!activeDataset || activeDataset.hasSimplified) {
    return;
  }

  const updatedUpload = await uploadLibrary.simplifyUpload(activeDataset.datasetId);
  if (updatedUpload) {
    dataset.useExistingDataset(updatedUpload);
  }
};

const loadMap = async (preserveResults = false) => {
  await dataset.filterActivities({
    bbox: bbox.value,
    geometryMode: geometryMode.value,
    preserveResults,
  });
};

const handlePreviewViewportChange = (viewportBBox: BBox) => {
  previewViewportBBox.value = [...viewportBBox];
};

watch(
  [currentStep, () => dataset.sessionId.value, previewViewportBBox],
  ([step, sessionId, viewportBBox], _, onCleanup) => {
    if (step !== 2 || !sessionId || !viewportBBox) {
      isLoadingAreaPreview.value = false;
      if (step !== 2 || !sessionId) {
        previewGeoJSON.value = null;
      }
      return;
    }

    const controller = new AbortController();
    const timeoutId = window.setTimeout(async () => {
      isLoadingAreaPreview.value = true;

      try {
        const res = await fetch("/api/filter", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            sessionId,
            bbox: viewportBBox,
            geometryMode: "simplified",
            limit: AREA_PREVIEW_ROUTE_LIMIT,
          }),
          signal: controller.signal,
        });

        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || `Server returned status ${res.status}`);
        }

        previewGeoJSON.value = (await res.json()) as GeoJSONFeatureCollection;
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") {
          return;
        }
        console.error("Failed to load area preview:", error);
      } finally {
        if (!controller.signal.aborted) {
          isLoadingAreaPreview.value = false;
        }
      }
    }, AREA_PREVIEW_DELAY_MS);

    onCleanup(() => {
      controller.abort();
      window.clearTimeout(timeoutId);
    });
  },
);

const prepareRouteGifPreview = async () => {
  await preparePreview({
    geoJSON: dataset.activitiesGeoJSON.value,
    bbox: bbox.value,
    cityName: cityName.value,
    routeLabel: props.toolTitle,
    datasetName: dataset.activeDataset.value?.displayName ?? null,
  });
};

const downloadRouteAnimation = async () => {
  await downloadAnimation({
    geoJSON: dataset.activitiesGeoJSON.value,
    bbox: bbox.value,
    cityName: cityName.value,
    routeLabel: props.toolTitle,
    datasetName: dataset.activeDataset.value?.displayName ?? null,
  });
};

watch(currentStep, (step) => {
  if (step !== 2) {
    previewViewportBBox.value = null;
    previewGeoJSON.value = null;
    isLoadingAreaPreview.value = false;
  }

  if (step === 3 && dataset.readyToFilter.value) {
    void loadMap();
    return;
  }

  if (step === 4 && hasAvailableRoutes.value && !dataset.isFiltering.value) {
    const count = availableRouteCount.value;
    if (count > 1) {
      const defaultDelay = Math.round((12000 - 1200) / (count - 1));
      updateFrameDelayMs(defaultDelay);
    }
    void prepareRouteGifPreview();
  }
});

watch(geometryMode, () => {
  if (currentStep.value >= 3 && dataset.readyToFilter.value) {
    void loadMap(currentStep.value === 3);
  }
});

watch(
  [
    routeColor,
    flashColor,
    frameDelayMs,
    exportFormat,
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
  ],
  (_, __, onCleanup) => {
    if (currentStep.value !== 4 || !hasAvailableRoutes.value || dataset.isFiltering.value) {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      void prepareRouteGifPreview();
    }, 200);
    onCleanup(() => window.clearTimeout(timeoutId));
  },
);

onMounted(async () => {
  await uploadLibrary.loadUploads();
  if (uploadLibrary.uploads.value.length > 0 && !dataset.activeDataset.value) {
    dataset.useExistingDataset(uploadLibrary.uploads.value[0]);
  }
});

const nextButton = computed(() => {
  if (currentStep.value === 1) {
    return {
      label: "Next: Frame area",
      disabled: !dataset.uploadSuccess.value,
      action: () => {
        currentStep.value = 2;
      },
    };
  }

  if (currentStep.value === 2) {
    return {
      label: "Next: Build map",
      disabled: false,
      action: () => {
        currentStep.value = 3;
      },
    };
  }

  if (currentStep.value === 3) {
    return {
      label: "Next: Prepare Export",
      disabled:
        dataset.isFiltering.value ||
        dataset.filterError.value !== null ||
        !hasAvailableRoutes.value,
      action: () => {
        currentStep.value = 4;
      },
    };
  }

  return null;
});

const resetFlow = () => {
  currentStep.value = 1;
  cityName.value = "Boston, MA, USA";
  bbox.value = [...DEFAULT_BOSTON_BBOX];
  center.value = [...DEFAULT_BOSTON_CENTER];
  geometryMode.value = "simplified";
  previewViewportBBox.value = null;
  previewGeoJSON.value = null;
  isLoadingAreaPreview.value = false;
  dataset.reset();
  resetState();
  cityNameOverlay.value = formatCityName(cityName.value);
  if (uploadLibrary.uploads.value.length > 0) {
    dataset.useExistingDataset(uploadLibrary.uploads.value[0]);
  }
};

const handleGlobalKeydown = (event: KeyboardEvent) => {
  if (event.key === "Enter") {
    const activeEl = document.activeElement as HTMLElement | null;
    if (activeEl) {
      if (activeEl.tagName === "TEXTAREA") {
        return;
      }
      if (activeEl.tagName === "INPUT") {
        const inputEl = activeEl as HTMLInputElement;
        if (
          inputEl.type === "text" ||
          inputEl.type === "password" ||
          inputEl.type === "email" ||
          inputEl.type === "number"
        ) {
          if (activeEl.classList.contains("search-input")) {
            const suggestionsList = document.querySelector(".suggestions-list");
            if (suggestionsList) {
              return;
            }
          } else {
            return;
          }
        }
      }
    }

    if (nextButton.value && !nextButton.value.disabled) {
      event.preventDefault();
      nextButton.value.action();
    }
  }
};

onMounted(() => {
  window.addEventListener("keydown", handleGlobalKeydown);
});

onUnmounted(() => {
  window.removeEventListener("keydown", handleGlobalKeydown);
});
</script>

<template>
  <section class="flow-layout">
    <FlowStepper :current-step="currentStep" :steps="steps" @step-click="currentStep = $event">
      <template #actions>
        <button
          v-if="nextButton"
          class="btn btn-primary"
          :disabled="nextButton.disabled"
          @click="nextButton.action"
        >
          {{ nextButton.label }}
        </button>
      </template>
    </FlowStepper>

    <div v-if="currentStep === 1" class="flow-layout">
      <DatasetUploadCard
        title="Step 1: Load a saved upload or ZIP"
        description="Pick a saved GeoParquet dataset or process one Strava ZIP."
        :selected-file="dataset.selectedFile.value"
        :upload-error="dataset.uploadError.value"
        :is-uploading="dataset.isUploading.value"
        :upload-success="dataset.uploadSuccess.value"
        :total-count="dataset.totalCount.value"
        :parsed-count="dataset.parsedCount.value"
        :ride-count="dataset.rideCount.value"
        :active-dataset-name="dataset.activeDataset.value?.displayName ?? null"
        :using-existing-dataset="dataset.usingExistingDataset.value"
        @select-file="dataset.setSelectedFile"
        @upload="submitSelectedArchive"
      >
        <template #sourceSelection>
          <UploadedDatasetList
            v-if="uploadLibrary.uploads.value.length"
            title="Saved uploads"
            description="Pick one to skip ZIP processing."
            :uploads="uploadLibrary.uploads.value"
            :selected-dataset-id="dataset.activeDataset.value?.datasetId ?? null"
            :selectable="true"
            :show-manage-link="true"
            action-label="Use upload"
            @select="dataset.useExistingDataset"
          />
        </template>
      </DatasetUploadCard>
      <div v-if="showSimplifyPrompt" class="card simplify-prompt">
        <div class="simplify-copy">
          <h3>Simplify geometry for faster map loads</h3>
          <p>
            {{ dataset.activeDataset.value?.displayName }} does not have a simplified GeoParquet
            companion yet. Build it now for the simplified view, or switch to original geometry in
            the map viewer.
          </p>
        </div>
        <div class="simplify-actions">
          <button
            class="btn btn-primary"
            :disabled="uploadLibrary.busyDatasetId.value === dataset.activeDataset.value?.datasetId"
            @click="simplifySelectedUpload"
          >
            {{
              uploadLibrary.busyDatasetId.value === dataset.activeDataset.value?.datasetId
                ? "Simplifying geometry…"
                : "Simplify geometry"
            }}
          </button>
        </div>
      </div>
      <div v-if="uploadLibrary.error.value" class="error-banner">
        ⚠️ {{ uploadLibrary.error.value }}
      </div>
    </div>

    <AreaSelectionCard
      v-else-if="currentStep === 2"
      title="Step 2: Frame the map area"
      description="Search a city and drag the box around the routes you want to keep."
      :city-name="cityName"
      v-model:bbox="bbox"
      @back="currentStep = 1"
      @select-city="handleSelectCity"
    >
      <template #details>
        <p v-if="dataset.readyToFilter.value" class="geometry-note">
          Showing up to {{ AREA_PREVIEW_ROUTE_LIMIT }} simplified rides from the visible map area.
          <span v-if="isLoadingAreaPreview"> Refreshing preview…</span>
        </p>
      </template>
      <MapView
        v-model:bbox="bbox"
        :center="center"
        :show-b-box="true"
        :routes="previewRoutes"
        @viewport-change="handlePreviewViewportChange"
      />
    </AreaSelectionCard>

    <div v-else-if="currentStep === 3" class="card-group">
      <section class="card flow-card final-card">
        <h2>{{ props.toolTitle }}</h2>

        <div v-if="dataset.isFiltering.value" class="processing-indicator">
          <div class="processing-ring"></div>
          <h3>Running the filter…</h3>
          <p>Querying the saved dataset inside the selected area.</p>
        </div>

        <div v-else-if="dataset.filterError.value" class="error-banner">
          ⚠️ {{ dataset.filterError.value }}
          <div class="mt-4">
            <button class="btn btn-primary" @click="loadMap()">Retry</button>
          </div>
        </div>

        <template v-else>
          <p>
            Showing <strong>{{ dataset.activitiesCount.value }}</strong> rides from
            <strong>{{ cityName }}</strong> in one route layer using
            <strong>{{ usingSimplifiedGeometry ? "simplified" : "original" }}</strong>
            geometry.
          </p>

          <div class="geometry-panel">
            <div class="geometry-panel-copy">
              <span class="control-label">Geometry</span>
              <div class="geometry-toggle" role="group" aria-label="Geometry mode">
                <button
                  class="geometry-toggle-button"
                  :class="{ active: usingSimplifiedGeometry }"
                  :disabled="dataset.isFiltering.value"
                  @click="geometryMode = 'simplified'"
                >
                  Simplified
                </button>
                <button
                  class="geometry-toggle-button"
                  :class="{ active: !usingSimplifiedGeometry }"
                  :disabled="dataset.isFiltering.value"
                  @click="geometryMode = 'original'"
                >
                  Original
                </button>
              </div>
              <p class="geometry-note">{{ geometryModeDescription }}</p>
            </div>
            <span v-if="dataset.isFiltering.value" class="geometry-status">Updating map…</span>
          </div>

          <div class="export-summary">
            <h4>Location</h4>
            <p>{{ cityName }}</p>
            <h4>Bounding Box</h4>
            <code class="code-block"
              >{{ bbox[0].toFixed(4) }}, {{ bbox[1].toFixed(4) }}, {{ bbox[2].toFixed(4) }},
              {{ bbox[3].toFixed(4) }}</code
            >
          </div>
        </template>

        <div class="card-actions mt-auto">
          <button class="btn btn-secondary" @click="currentStep = 2">Back</button>
          <button v-if="!dataset.isFiltering.value" class="btn btn-secondary" @click="resetFlow">
            Start over
          </button>
        </div>
      </section>

      <div class="map-container-wrapper">
        <MapView v-model:bbox="bbox" :center="center" :show-b-box="false" :routes="displayRoutes" />
      </div>
    </div>

    <section v-else class="card flow-card final-card">
      <RouteGifExportCard
        :route-count="availableRouteCount"
        :preview-route-count="previewRouteCount"
        :preview-target-count="previewTargetCount"
        :preview-progress-count="previewProgressCount"
        :is-filtering="dataset.isFiltering.value"
        :is-preparing-preview="isPreparingPreview"
        :is-downloading="isDownloading"
        :route-color="routeColor"
        :flash-color="flashColor"
        :frame-delay-ms="frameDelayMs"
        :preview-url="previewUrl"
        :export-error="exportError"
        :status-message="statusMessage"
        v-model:show-city-name="showCityName"
        v-model:city-font="cityFont"
        v-model:city-position="cityPosition"
        v-model:city-name-overlay="cityNameOverlay"
        v-model:show-distance="showDistance"
        v-model:distance-position="distancePosition"
        v-model:distance-unit="distanceUnit"
        v-model:distance-font="distanceFont"
        v-model:show-date="showDate"
        v-model:date-position="datePosition"
        v-model:date-font="dateFont"
        v-model:date-format="dateFormat"
        v-model:export-format="exportFormat"
        :is-movie-export-supported="isMovieExportSupported"
        :is-native-mp4-supported="isNativeMp4Supported"
        :is-transcode-available="isTranscodeAvailable"
        @update:route-color="updateRouteColor"
        @update:flash-color="updateFlashColor"
        @update:frame-delay-ms="updateFrameDelayMs"
        @export="downloadRouteAnimation"
      />

      <div class="card-actions mt-auto">
        <button class="btn btn-secondary" @click="currentStep = 3">Back</button>
        <button class="btn btn-secondary" @click="resetFlow">Start over</button>
      </div>
    </section>
  </section>
</template>

<style scoped>
@reference "tailwindcss";

.flow-layout {
  @apply flex flex-col gap-4;
}

.centered-banner {
  @apply text-center;
}

.compact-lead {
  @apply mb-0;
}

.simplify-prompt {
  @apply flex flex-col gap-3 border border-amber-800 bg-amber-500/10;
}

.simplify-copy {
  @apply flex flex-col gap-1.5;
}

.simplify-copy h3,
.simplify-copy p {
  @apply m-0;
}

.simplify-copy h3 {
  @apply text-amber-200;
}

.simplify-copy p {
  @apply text-amber-300;
}

.simplify-actions {
  @apply flex justify-start;
}

.geometry-panel {
  @apply flex flex-wrap items-start justify-between gap-3 rounded-xl border border-zinc-800 bg-zinc-900 p-3.5;
}

.geometry-panel-copy {
  @apply flex flex-col gap-2.5;
}

.geometry-toggle {
  @apply inline-flex gap-2;
}

.geometry-toggle-button {
  @apply cursor-pointer rounded-full border border-zinc-600 bg-zinc-950 px-3 py-2 text-zinc-300 transition-colors;
}

.geometry-toggle-button.active {
  @apply border-amber-500 bg-amber-500/15 text-white;
}

.geometry-toggle-button:disabled {
  @apply cursor-not-allowed opacity-60;
}

.geometry-note {
  @apply m-0 text-[0.92rem] text-zinc-400;
}

.geometry-status {
  @apply whitespace-nowrap rounded-full border border-amber-800 bg-amber-500/10 px-2.5 py-1.5 text-[0.82rem] text-amber-200;
}
</style>
