<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { UploadSummary } from "../../lib/activity";
import { uploadArchive } from "../../lib/uploads";
import { useUploadedDatasets } from "../../composables/useUploadedDatasets";
import BulkUploadCard from "./BulkUploadCard.vue";
import UploadedDatasetList from "./UploadedDatasetList.vue";

const uploadLibrary = useUploadedDatasets();
const selectedFiles = ref<File[]>([]);
const isUploading = ref(false);
const uploadError = ref<string | null>(null);
const uploadResults = ref<UploadSummary[]>([]);

const uploadSummary = computed(() => {
  if (!uploadResults.value.length) {
    return null;
  }

  const uploadedArchives = uploadResults.value.length;
  const totalRides = uploadResults.value.reduce((sum, upload) => sum + upload.rideCount, 0);

  return `Processed ${uploadedArchives} archive${uploadedArchives === 1 ? "" : "s"} and saved ${totalRides} ride${
    totalRides === 1 ? "" : "s"
  } locally.`;
});

const guideSteps = [
  "Log into Strava on your computer.",
  "Open Settings → My Account.",
  "Go to Download or Delete Your Account.",
  "Choose Request Your Archive.",
  "Download the ZIP link from the email.",
];

const setSelectedFiles = (files: File[]) => {
  selectedFiles.value = files;
  uploadError.value = null;
  uploadResults.value = [];
};

const uploadSelectedFiles = async () => {
  if (!selectedFiles.value.length) {
    return;
  }

  const invalidFiles = selectedFiles.value.filter(
    (file) => !file.name.toLowerCase().endsWith(".zip"),
  );
  if (invalidFiles.length) {
    uploadError.value = `Only .zip files are supported. Invalid selection: ${invalidFiles.map((file) => file.name).join(", ")}`;
    return;
  }

  isUploading.value = true;
  uploadError.value = null;
  uploadResults.value = [];

  try {
    const results: UploadSummary[] = [];

    for (const file of selectedFiles.value) {
      results.push(await uploadArchive(file));
    }

    uploadResults.value = results;
    selectedFiles.value = [];
    await uploadLibrary.loadUploads();
  } catch (err) {
    console.error(err);
    uploadError.value =
      err instanceof Error ? err.message : "Failed to process the selected archives.";
    await uploadLibrary.loadUploads();
  } finally {
    isUploading.value = false;
  }
};

onMounted(() => {
  void uploadLibrary.loadUploads();
});
</script>

<template>
  <section class="upload-page">
    <BulkUploadCard
      :selected-files="selectedFiles"
      :is-uploading="isUploading"
      :upload-error="uploadError"
      :upload-summary="uploadSummary"
      @select-files="setSelectedFiles"
      @upload="uploadSelectedFiles"
    />

    <section class="card upload-library-card">
      <UploadedDatasetList
        title="Saved uploads"
        description="Rename, delete, open, and reuse the GeoParquet files stored in your local Map Tools library."
        :uploads="uploadLibrary.uploads.value"
        :limit="3"
        :manageable="true"
        :openable="true"
        :busy-dataset-id="uploadLibrary.busyDatasetId.value"
        empty-message="No GeoParquet uploads found in the local library yet."
        @rename="uploadLibrary.renameUpload($event.datasetId, $event.name)"
        @simplify="uploadLibrary.simplifyUpload"
        @open="uploadLibrary.openUpload"
        @delete="uploadLibrary.deleteUpload"
      />
    </section>

    <section class="card export-guide">
      <div class="guide-head">
        <div>
          <span class="hero-kicker">Strava export</span>
          <h3>Get the ZIP</h3>
        </div>
        <a
          href="https://support.strava.com/en-us/articles/15401919-exporting-your-data-and-bulk-export"
          target="_blank"
          class="link"
          >Open guide</a
        >
      </div>

      <p class="guide-copy">Request the archive in Strava, then download the ZIP from the email.</p>

      <ol class="guide-steps">
        <li v-for="step in guideSteps" :key="step">{{ step }}</li>
      </ol>
    </section>

    <div v-if="uploadLibrary.error.value" class="error-banner">
      ⚠️ {{ uploadLibrary.error.value }}
    </div>
  </section>
</template>

<style scoped>
@reference "tailwindcss";

.upload-page {
  @apply flex flex-col gap-4;
}

.hero-kicker {
  @apply text-[0.72rem] font-bold uppercase text-amber-500;
  letter-spacing: 0.12em;
}

.export-guide,
.upload-library-card {
  @apply flex flex-col gap-3;
}

.guide-head {
  @apply flex items-start justify-between gap-3;
}

.guide-head h3,
.guide-copy {
  @apply m-0;
}

.guide-steps {
  @apply m-0 pl-[18px] leading-relaxed;
}

.guide-steps li + li {
  @apply mt-[7px];
}

@media (max-width: 720px) {
  .guide-head {
    @apply flex-col;
  }
}
</style>
