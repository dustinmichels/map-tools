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
} = useRouteGifExport();

watch(
  cityName,
  (newCity) => {
    cityNameOverlay.value = formatCityName(newCity);
  },
  { immediate: true },
);

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
    <FlowStepper
      :current-step="currentStep"
      :steps="steps"
      @step-click="currentStep = $event"
    >
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
      <MapView v-model:bbox="bbox" :center="center" :show-b-box="true" :routes="[]" />
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
            <strong>{{ usingSimplifiedGeometry ? "simplified" : "original" }}</strong> geometry.
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
            <code class="block"
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
.flow-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.centered-banner {
  text-align: center;
}

.compact-lead {
  margin-bottom: 0;
}

.simplify-prompt {
  display: flex;
  flex-direction: column;
  gap: 12px;
  border: 1px solid #8a5a12;
  background: rgba(255, 153, 0, 0.1);
}

.simplify-copy {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.simplify-copy h3,
.simplify-copy p {
  margin: 0;
}

.simplify-copy h3 {
  color: #ffd180;
}

.simplify-copy p {
  color: #ffcc80;
}

.simplify-actions {
  display: flex;
  justify-content: flex-start;
}

.geometry-panel {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  border-radius: 12px;
  border: 1px solid #2f2f2f;
  background: #181818;
}

.geometry-panel-copy {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.geometry-toggle {
  display: inline-flex;
  gap: 8px;
}

.geometry-toggle-button {
  border: 1px solid #444;
  background: #111;
  color: #d0d0d0;
  border-radius: 999px;
  padding: 8px 12px;
  cursor: pointer;
  transition:
    border-color 0.15s,
    background 0.15s,
    color 0.15s;
}

.geometry-toggle-button.active {
  border-color: #ff9900;
  background: rgba(255, 153, 0, 0.15);
  color: #fff;
}

.geometry-toggle-button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.geometry-note {
  margin: 0;
  color: #a8a8a8;
  font-size: 0.92rem;
}

.geometry-status {
  border: 1px solid #8a5a12;
  border-radius: 999px;
  padding: 6px 10px;
  background: rgba(255, 153, 0, 0.12);
  color: #ffd180;
  font-size: 0.82rem;
  white-space: nowrap;
}
</style>
