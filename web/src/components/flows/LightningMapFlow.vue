<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from "vue";
import AreaSelectionCard from "./AreaSelectionCard.vue";
import DatasetUploadCard from "./DatasetUploadCard.vue";
import FlowStepper from "./FlowStepper.vue";
import UploadedDatasetList from "../uploads/UploadedDatasetList.vue";
import { useActivityDataset } from "../../composables/useActivityDataset";
import { useUploadedDatasets } from "../../composables/useUploadedDatasets";
import {
  DEFAULT_BOSTON_BBOX,
  DEFAULT_BOSTON_CENTER,
  type BBox,
  type LngLat,
  type RouteLayer,
  type SelectedCity,
} from "../../lib/activity";

const MapView = defineAsyncComponent(() => import("../MapView.vue"));

const steps = [
  { number: 1, label: "Upload" },
  { number: 2, label: "Area" },
  { number: 3, label: "Process" },
  { number: 4, label: "Map" },
];

const currentStep = ref(1);
const cityName = ref("Boston, MA, USA");
const bbox = ref<BBox>([...DEFAULT_BOSTON_BBOX]);
const center = ref<LngLat>([...DEFAULT_BOSTON_CENTER]);
const dataset = useActivityDataset();
const uploadLibrary = useUploadedDatasets();

const lightningRoutes = computed<RouteLayer[]>(() => [
  {
    id: "lightning-map",
    label: "Lightning Map",
    color: "#ff8c00",
    data: dataset.activitiesGeoJSON.value,
  },
]);

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

const needsSimplifiedGeometry = computed(
  () =>
    dataset.usingExistingDataset.value &&
    dataset.activeDataset.value !== null &&
    !dataset.activeDataset.value.hasSimplified,
);

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

watch(currentStep, (step) => {
  if (step === 3 && dataset.readyToFilter.value) {
    void dataset.filterActivities(bbox.value);
  }
});

onMounted(() => {
  void uploadLibrary.loadUploads();
});

const resetFlow = () => {
  currentStep.value = 1;
  cityName.value = "Boston, MA, USA";
  bbox.value = [...DEFAULT_BOSTON_BBOX];
  center.value = [...DEFAULT_BOSTON_CENTER];
  dataset.reset();
};
</script>

<template>
  <section class="flow-layout">
    <FlowStepper :current-step="currentStep" :steps="steps" />

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
      <div v-if="needsSimplifiedGeometry" class="card simplify-prompt">
        <div class="simplify-copy">
          <h3>Simplify geometry before building the map</h3>
          <p>
            {{ dataset.activeDataset.value?.displayName }} was saved before simplified geometry was
            available. Create the simplified GeoParquet companion now so the lightning map uses it.
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
      <div class="card-actions">
        <button
          class="btn btn-primary"
          :disabled="!dataset.uploadSuccess.value"
          @click="currentStep = 2"
        >
          Frame area
        </button>
      </div>
    </div>

    <AreaSelectionCard
      v-else-if="currentStep === 2"
      title="Step 2: Frame the map area"
      description="Search a city and drag the box around the routes you want to keep."
      :city-name="cityName"
      :bbox="bbox"
      next-label="Build map"
      @back="currentStep = 1"
      @next="currentStep = 3"
      @select-city="handleSelectCity"
    >
      <MapView v-model:bbox="bbox" :center="center" :show-b-box="true" :routes="[]" />
    </AreaSelectionCard>

    <div v-else-if="currentStep === 3" class="card flow-card text-center">
      <h2>Step 3: Build the map</h2>
      <p>Filtering saved rides inside the selected bounding box.</p>

      <div v-if="dataset.isFiltering.value" class="processing-indicator">
        <div class="processing-ring"></div>
        <h3>Running the filter…</h3>
        <p>Querying the saved dataset inside the selected area.</p>
      </div>

      <div v-else-if="dataset.filterError.value" class="error-banner">
        ⚠️ {{ dataset.filterError.value }}
        <div class="mt-4">
          <button class="btn btn-primary" @click="dataset.filterActivities(bbox)">Retry</button>
        </div>
      </div>

      <div
        v-else-if="dataset.activitiesCount.value !== null"
        class="success-banner centered-banner"
      >
        <h3>Ready</h3>
        <p class="lead-text compact-lead">
          Found <strong>{{ dataset.activitiesCount.value }}</strong> rides in
          <strong>{{ cityName }}</strong
          >.
        </p>
      </div>

      <div class="card-actions">
        <button
          class="btn btn-secondary"
          :disabled="dataset.isFiltering.value"
          @click="currentStep = 2"
        >
          Back
        </button>
        <button
          class="btn btn-primary"
          :disabled="dataset.activitiesCount.value === null || dataset.isFiltering.value"
          @click="currentStep = 4"
        >
          Open map
        </button>
      </div>
    </div>

    <div v-else class="card-group">
      <section class="card flow-card final-card">
        <h2>Map preview</h2>
        <p>
          Showing <strong>{{ dataset.activitiesCount.value }}</strong> rides from
          <strong>{{ cityName }}</strong> in one route layer.
        </p>

        <div class="export-summary">
          <h4>Location</h4>
          <p>{{ cityName }}</p>
          <h4>Bounding Box</h4>
          <code class="block"
            >{{ bbox[0].toFixed(4) }}, {{ bbox[1].toFixed(4) }}, {{ bbox[2].toFixed(4) }},
            {{ bbox[3].toFixed(4) }}</code
          >
        </div>

        <div class="card-actions mt-auto">
          <button class="btn btn-secondary" @click="currentStep = 3">Back</button>
          <button class="btn btn-secondary" @click="resetFlow">Start over</button>
        </div>
      </section>

      <div class="map-container-wrapper">
        <MapView
          v-model:bbox="bbox"
          :center="center"
          :show-b-box="false"
          :routes="lightningRoutes"
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
</style>
