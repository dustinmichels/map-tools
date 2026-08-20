<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import AreaSelectionCard from "./AreaSelectionCard.vue";
import DatasetUploadCard from "./DatasetUploadCard.vue";
import FlowStepper from "./FlowStepper.vue";
import MapView from "../MapView.vue";
import UploadedDatasetList from "../uploads/UploadedDatasetList.vue";
import { useActivityDataset } from "../../composables/useActivityDataset";
import { useUploadedDatasets } from "../../composables/useUploadedDatasets";
import {
  DEFAULT_BOSTON_BBOX,
  DEFAULT_BOSTON_CENTER,
  bboxCenter,
  getFeatureCollectionBounds,
  type BBox,
  type GeoJSONFeatureCollection,
  type LngLat,
  type RouteLayer,
  type SelectedCity,
} from "../../lib/activity";

const DEFAULT_PERSON_ONE_NAME = "Person 1";
const DEFAULT_PERSON_TWO_NAME = "Person 2";

const steps = [
  { number: 1, label: "Upload" },
  { number: 2, label: "Area" },
  { number: 3, label: "Map" },
];

const currentStep = ref(1);
const cityName = ref("Boston, MA, USA");
const bbox = ref<BBox>([...DEFAULT_BOSTON_BBOX]);
const center = ref<LngLat>([...DEFAULT_BOSTON_CENTER]);
const personOneName = ref(DEFAULT_PERSON_ONE_NAME);
const personTwoName = ref(DEFAULT_PERSON_TWO_NAME);
const personOneLabel = computed(() => personOneName.value.trim() || DEFAULT_PERSON_ONE_NAME);
const personTwoLabel = computed(() => personTwoName.value.trim() || DEFAULT_PERSON_TWO_NAME);
const personOneColor = ref("#ff8c00");
const personTwoColor = ref("#2563eb");
const compareAllRides = ref(false);
const personOne = useActivityDataset();
const personTwo = useActivityDataset();
const uploadLibrary = useUploadedDatasets();

const compareRoutes = computed<RouteLayer[]>(() => [
  {
    id: "person-one",
    label: personOneLabel.value,
    color: personOneColor.value,
    data: personOne.activitiesGeoJSON.value,
  },
  {
    id: "person-two",
    label: personTwoLabel.value,
    color: personTwoColor.value,
    data: personTwo.activitiesGeoJSON.value,
  },
]);
const totalComparedActivities = computed(
  () => (personOne.activitiesCount.value ?? 0) + (personTwo.activitiesCount.value ?? 0),
);
const hasBothUploads = computed(
  () => personOne.uploadSuccess.value && personTwo.uploadSuccess.value,
);
const isFiltering = computed(() => personOne.isFiltering.value || personTwo.isFiltering.value);
const hasResults = computed(
  () => personOne.activitiesCount.value !== null && personTwo.activitiesCount.value !== null,
);

const combinedCompareGeoJSON = computed<GeoJSONFeatureCollection>(() => ({
  type: "FeatureCollection",
  features: compareRoutes.value.flatMap((route) => route.data?.features ?? []),
}));
const compareBounds = computed(() => getFeatureCollectionBounds(combinedCompareGeoJSON.value));
const compareMapBBox = computed<BBox>(() =>
  compareAllRides.value && compareBounds.value ? compareBounds.value : bbox.value,
);
const compareMapCenter = computed<LngLat>(() =>
  compareAllRides.value && compareBounds.value ? bboxCenter(compareBounds.value) : center.value,
);
const areaStepTitle = computed(() =>
  compareAllRides.value ? "Step 2: Compare every ride" : "Step 2: Frame one shared area",
);
const areaStepDescription = computed(() =>
  compareAllRides.value
    ? "Skip the bounding box and compare every Ride activity from both uploads."
    : "Search a city and drag the box around the area both people should share.",
);
const filterErrors = computed(() =>
  [personOne.filterError.value, personTwo.filterError.value].filter((message): message is string =>
    Boolean(message),
  ),
);

const handleSelectCity = (payload: SelectedCity) => {
  cityName.value = payload.name;
  bbox.value = payload.bbox;
  center.value = [payload.lon, payload.lat];
};

const submitPersonOneArchive = async () => {
  const upload = await personOne.submitZip();
  if (upload) {
    await uploadLibrary.loadUploads();
  }
};

const submitPersonTwoArchive = async () => {
  const upload = await personTwo.submitZip();
  if (upload) {
    await uploadLibrary.loadUploads();
  }
};

const personOneNeedsSimplifiedGeometry = computed(
  () =>
    personOne.usingExistingDataset.value &&
    personOne.activeDataset.value !== null &&
    !personOne.activeDataset.value.hasSimplified,
);
const personTwoNeedsSimplifiedGeometry = computed(
  () =>
    personTwo.usingExistingDataset.value &&
    personTwo.activeDataset.value !== null &&
    !personTwo.activeDataset.value.hasSimplified,
);

const simplifySelectedDataset = async (person: typeof personOne) => {
  const activeDataset = person.activeDataset.value;
  if (!activeDataset || activeDataset.hasSimplified) {
    return;
  }

  const updatedUpload = await uploadLibrary.simplifyUpload(activeDataset.datasetId);
  if (updatedUpload) {
    person.useExistingDataset(updatedUpload);
  }
};

const runCompare = async () => {
  const selectedBBox = compareAllRides.value ? null : bbox.value;

  await Promise.all([
    personOne.filterActivities({ bbox: selectedBBox }),
    personTwo.filterActivities({ bbox: selectedBBox }),
  ]);
};

watch(currentStep, (step) => {
  if (step === 3 && hasBothUploads.value) {
    void runCompare();
  }
});

onMounted(async () => {
  await uploadLibrary.loadUploads();
  if (uploadLibrary.uploads.value.length > 0) {
    if (!personOne.activeDataset.value) {
      personOne.useExistingDataset(uploadLibrary.uploads.value[0]);
    }
    if (uploadLibrary.uploads.value.length > 1 && !personTwo.activeDataset.value) {
      personTwo.useExistingDataset(uploadLibrary.uploads.value[1]);
    }
  }
});

const nextButton = computed(() => {
  if (currentStep.value === 1) {
    return {
      label: "Next: Choose scope",
      disabled: !hasBothUploads.value,
      action: () => {
        currentStep.value = 2;
      },
    };
  } else if (currentStep.value === 2) {
    return {
      label: "Next: Build compare map",
      disabled: false,
      action: () => {
        currentStep.value = 3;
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
  compareAllRides.value = false;
  personOneName.value = DEFAULT_PERSON_ONE_NAME;
  personTwoName.value = DEFAULT_PERSON_TWO_NAME;
  personOneColor.value = "#ff8c00";
  personTwoColor.value = "#2563eb";
  personOne.reset();
  personTwo.reset();
  if (uploadLibrary.uploads.value.length > 0) {
    personOne.useExistingDataset(uploadLibrary.uploads.value[0]);
  }
  if (uploadLibrary.uploads.value.length > 1) {
    personTwo.useExistingDataset(uploadLibrary.uploads.value[1]);
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
    <FlowStepper :current-step="currentStep" :steps="steps">
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
      <div class="compare-upload-grid">
        <div class="compare-upload-column">
          <div class="compare-name-field">
            <label class="compare-name-label" for="compare-person-one-name">Display name</label>
            <input
              id="compare-person-one-name"
              v-model="personOneName"
              type="text"
              class="compare-name-input"
            />
          </div>
          <DatasetUploadCard
            :title="personOneLabel"
            description="Pick a saved GeoParquet dataset or process the first Strava ZIP."
            :selected-file="personOne.selectedFile.value"
            :upload-error="personOne.uploadError.value"
            :is-uploading="personOne.isUploading.value"
            :upload-success="personOne.uploadSuccess.value"
            :total-count="personOne.totalCount.value"
            :parsed-count="personOne.parsedCount.value"
            :ride-count="personOne.rideCount.value"
            :active-dataset-name="personOne.activeDataset.value?.displayName ?? null"
            :using-existing-dataset="personOne.usingExistingDataset.value"
            :color="personOneColor"
            color-label="Route color"
            :show-color-picker="true"
            @select-file="personOne.setSelectedFile"
            @upload="submitPersonOneArchive"
            @update-color="personOneColor = $event"
          >
            <template #sourceSelection>
              <UploadedDatasetList
                v-if="uploadLibrary.uploads.value.length"
                title="Saved uploads"
                :description="`Pick an existing dataset for ${personOneLabel}.`"
                :uploads="uploadLibrary.uploads.value"
                :selected-dataset-id="personOne.activeDataset.value?.datasetId ?? null"
                :selectable="true"
                action-label="Use upload"
                @select="personOne.useExistingDataset"
              />
            </template>
          </DatasetUploadCard>
          <div v-if="personOneNeedsSimplifiedGeometry" class="card simplify-prompt">
            <div class="simplify-copy">
              <h3>Simplify {{ personOneLabel }} geometry</h3>
              <p>
                {{ personOne.activeDataset.value?.displayName }} can add a simplified GeoParquet
                companion now to speed this compare map.
              </p>
            </div>
            <div class="simplify-actions">
              <button
                class="btn btn-primary"
                :disabled="
                  uploadLibrary.busyDatasetId.value === personOne.activeDataset.value?.datasetId
                "
                @click="simplifySelectedDataset(personOne)"
              >
                {{
                  uploadLibrary.busyDatasetId.value === personOne.activeDataset.value?.datasetId
                    ? "Simplifying geometry…"
                    : "Simplify geometry"
                }}
              </button>
            </div>
          </div>
        </div>
        <div class="compare-upload-column">
          <div class="compare-name-field">
            <label class="compare-name-label" for="compare-person-two-name">Display name</label>
            <input
              id="compare-person-two-name"
              v-model="personTwoName"
              type="text"
              class="compare-name-input"
            />
          </div>
          <DatasetUploadCard
            :title="personTwoLabel"
            description="Pick another saved dataset or process the second Strava ZIP."
            :selected-file="personTwo.selectedFile.value"
            :upload-error="personTwo.uploadError.value"
            :is-uploading="personTwo.isUploading.value"
            :upload-success="personTwo.uploadSuccess.value"
            :total-count="personTwo.totalCount.value"
            :parsed-count="personTwo.parsedCount.value"
            :ride-count="personTwo.rideCount.value"
            :active-dataset-name="personTwo.activeDataset.value?.displayName ?? null"
            :using-existing-dataset="personTwo.usingExistingDataset.value"
            :color="personTwoColor"
            color-label="Route color"
            :show-color-picker="true"
            @select-file="personTwo.setSelectedFile"
            @upload="submitPersonTwoArchive"
            @update-color="personTwoColor = $event"
          >
            <template #sourceSelection>
              <UploadedDatasetList
                v-if="uploadLibrary.uploads.value.length"
                title="Saved uploads"
                :description="`Pick an existing dataset for ${personTwoLabel}.`"
                :uploads="uploadLibrary.uploads.value"
                :selected-dataset-id="personTwo.activeDataset.value?.datasetId ?? null"
                :selectable="true"
                :show-manage-link="true"
                action-label="Use upload"
                @select="personTwo.useExistingDataset"
              />
            </template>
          </DatasetUploadCard>
          <div v-if="personTwoNeedsSimplifiedGeometry" class="card simplify-prompt">
            <div class="simplify-copy">
              <h3>Simplify {{ personTwoLabel }} geometry</h3>
              <p>
                {{ personTwo.activeDataset.value?.displayName }} can add a simplified GeoParquet
                companion now to speed this compare map.
              </p>
            </div>
            <div class="simplify-actions">
              <button
                class="btn btn-primary"
                :disabled="
                  uploadLibrary.busyDatasetId.value === personTwo.activeDataset.value?.datasetId
                "
                @click="simplifySelectedDataset(personTwo)"
              >
                {{
                  uploadLibrary.busyDatasetId.value === personTwo.activeDataset.value?.datasetId
                    ? "Simplifying geometry…"
                    : "Simplify geometry"
                }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="uploadLibrary.error.value" class="error-banner">
        ⚠️ {{ uploadLibrary.error.value }}
      </div>

      <p class="compare-note">Load both people, then choose a shared box or compare every Ride.</p>

    </div>

    <AreaSelectionCard
      v-else-if="currentStep === 2"
      :title="areaStepTitle"
      :description="areaStepDescription"
      :city-name="cityName"
      :bbox="bbox"
      :show-current-area="!compareAllRides"
      @back="currentStep = 1"
      @select-city="handleSelectCity"
    >
      <template #details>
        <label class="compare-scope-toggle">
          <input v-model="compareAllRides" type="checkbox" class="compare-scope-checkbox" />
          <span class="compare-scope-copy">
            <strong>Compare all rides</strong>
            <span>Skip the box and include every Ride activity from both uploads.</span>
          </span>
        </label>
      </template>
      <MapView v-model:bbox="bbox" :center="center" :show-b-box="!compareAllRides" :routes="[]" />
    </AreaSelectionCard>

    <div v-else class="card-group compare-results-layout">
      <section class="card flow-card final-card">
        <h2>Compare map</h2>

        <div v-if="isFiltering" class="processing-indicator">
          <div class="processing-ring"></div>
          <h3>Processing both people…</h3>
          <p v-if="compareAllRides">Loading every Ride activity from both uploaded datasets.</p>
          <p v-else>Running the same area filter for {{ personOneLabel }} and {{ personTwoLabel }}.</p>
        </div>

        <div v-else-if="filterErrors.length" class="error-stack">
          <div v-for="message in filterErrors" :key="message" class="error-banner">
            ⚠️ {{ message }}
          </div>
          <div class="mt-4 retry-actions">
            <button class="btn btn-primary" @click="runCompare">Retry both</button>
          </div>
        </div>

        <template v-else>
          <p>
            Overlaying <strong>{{ totalComparedActivities }}</strong> rides from both people
            <template v-if="compareAllRides">across both uploaded datasets.</template>
            <template v-else
              >inside <strong>{{ cityName }}</strong
              >.</template
            >
          </p>

          <div class="compare-legend">
            <div class="legend-row">
              <span class="legend-swatch" :style="{ backgroundColor: personOneColor }"></span>
              <div class="legend-copy">
                <strong>{{ personOneLabel }}</strong>
                <span>{{ personOne.activitiesCount.value }} rides</span>
              </div>
              <input
                v-model="personOneColor"
                type="color"
                class="legend-picker"
                :aria-label="`${personOneLabel} color`"
              />
            </div>
            <div class="legend-row">
              <span class="legend-swatch" :style="{ backgroundColor: personTwoColor }"></span>
              <div class="legend-copy">
                <strong>{{ personTwoLabel }}</strong>
                <span>{{ personTwo.activitiesCount.value }} rides</span>
              </div>
              <input
                v-model="personTwoColor"
                type="color"
                class="legend-picker"
                :aria-label="`${personTwoLabel} color`"
              />
            </div>
          </div>

          <div class="export-summary">
            <h4>Scope</h4>
            <p>{{ compareAllRides ? "All uploaded rides" : cityName }}</p>
            <template v-if="!compareAllRides">
              <h4>Bounding Box</h4>
              <code class="block"
                >{{ bbox[0].toFixed(4) }}, {{ bbox[1].toFixed(4) }}, {{ bbox[2].toFixed(4) }},
                {{ bbox[3].toFixed(4) }}</code
              >
            </template>
          </div>
        </template>

        <div class="card-actions mt-auto">
          <button class="btn btn-secondary" @click="currentStep = 2">Back</button>
          <button v-if="!isFiltering" class="btn btn-secondary" @click="resetFlow">Start over</button>
        </div>
      </section>

      <div class="map-container-wrapper">
        <MapView
          :bbox="compareMapBBox"
          :center="compareMapCenter"
          :show-b-box="false"
          :routes="compareRoutes"
        />
      </div>
    </div>
  </section>
</template>

<style scoped>
.flow-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.compare-upload-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.compare-upload-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.compare-name-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.compare-name-label {
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #888;
}

.compare-name-input {
  background: #111;
  border: 1px solid #444;
  border-radius: 8px;
  color: #fff;
  padding: 8px 10px;
}

.compare-note {
  margin: 0;
  color: #9c9c9c;
}

.error-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.retry-actions {
  display: flex;
  justify-content: center;
}

.compare-results-layout {
  align-items: stretch;
}

.compare-legend {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 16px;
}

.legend-row {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 10px;
  align-items: center;
  padding: 10px 12px;
  border-radius: 10px;
  background: #202020;
  border: 1px solid #333;
}

.legend-swatch {
  width: 14px;
  height: 14px;
  border-radius: 999px;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.08);
}

.legend-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: #d0d0d0;
}

.legend-copy strong {
  color: #fff;
}

.legend-copy span {
  font-size: 0.88rem;
  color: #a0a0a0;
}

.legend-picker {
  width: 42px;
  height: 30px;
  border: 1px solid #444;
  border-radius: 8px;
  background: #111;
  padding: 4px;
}

.compare-scope-toggle {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  border-radius: 10px;
  background: #202020;
  border: 1px solid #333;
}

.compare-scope-checkbox {
  margin: 3px 0 0;
  accent-color: #ff9900;
}

.compare-scope-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.compare-scope-copy strong {
  color: #fff;
}

.compare-scope-copy span {
  color: #a0a0a0;
  font-size: 0.9rem;
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

@media (max-width: 900px) {
  .compare-upload-grid {
    grid-template-columns: 1fr;
  }
}
</style>
