<script setup lang="ts">
import { ref, useSlots } from "vue";
import { useDropZone } from "@vueuse/core";
import { formatFileSize } from "../../lib/activity";

const props = withDefaults(
  defineProps<{
    title: string;
    description: string;
    selectedFile: File | null;
    uploadError: string | null;
    isUploading: boolean;
    uploadSuccess: boolean;
    totalCount: number | null;
    parsedCount: number | null;
    rideCount: number | null;
    activeDatasetName?: string | null;
    usingExistingDataset?: boolean;
    color?: string;
    colorLabel?: string;
    showColorPicker?: boolean;
    uploadLabel?: string;
  }>(),
  {
    activeDatasetName: null,
    usingExistingDataset: false,
    color: "#ff9900",
    colorLabel: "Route color",
    showColorPicker: false,
    uploadLabel: "Process ZIP",
  },
);

const emit = defineEmits<{
  selectFile: [file: File | null];
  upload: [];
  updateColor: [color: string];
}>();

const slots = useSlots();
const dropZoneRef = ref<HTMLDivElement>();
const { isOverDropZone } = useDropZone(dropZoneRef, (files) => {
  if (files) {
    emit("selectFile", files[0] ?? null);
  }
});

const fileListChanged = (event: Event) => {
  const target = event.target as HTMLInputElement;
  emit("selectFile", target.files?.[0] ?? null);
};
</script>

<template>
  <section class="card upload-card">
    <div class="upload-card-head">
      <div class="upload-card-copy">
        <h3>{{ title }}</h3>
        <p>{{ description }}</p>
      </div>
      <label v-if="showColorPicker" class="color-control">
        <span>{{ colorLabel }}</span>
        <input
          :value="color"
          type="color"
          @input="emit('updateColor', ($event.target as HTMLInputElement).value)"
        />
      </label>
    </div>

    <div v-if="slots.sourceSelection" class="source-selection">
      <slot name="sourceSelection" />
    </div>

    <div ref="dropZoneRef" class="upload-zone" :class="{ dragging: isOverDropZone }">
      <input type="file" accept=".zip" class="file-input" @change="fileListChanged" />
      <label class="upload-label">
        <span class="upload-icon">📦</span>
        <span v-if="selectedFile" class="file-name"
          >{{ selectedFile.name }} ({{ formatFileSize(selectedFile.size) }})</span
        >
        <span v-else>Choose one ZIP or drop it here</span>
      </label>
    </div>

    <div v-if="uploadError" class="error-banner">⚠️ {{ uploadError }}</div>

    <div v-if="isUploading" class="progress-container">
      <div class="progress-spinner"></div>
      <span>Processing the ZIP into a saved GeoParquet dataset…</span>
    </div>

    <div v-if="uploadSuccess" class="success-banner upload-success">
      <strong>{{ usingExistingDataset ? "Saved upload ready." : "ZIP processed." }}</strong>
      <span v-if="usingExistingDataset">
        Using {{ activeDatasetName ?? "the selected GeoParquet file" }} from the local library.
      </span>
      <span v-else>
        Parsed {{ parsedCount }} / {{ totalCount }} activities, kept {{ rideCount }} rides, and
        saved simplified geometry for map creation.
      </span>
    </div>

    <button
      class="btn btn-primary upload-action"
      :disabled="!selectedFile || isUploading || uploadSuccess"
      @click="emit('upload')"
    >
      {{ uploadLabel }}
    </button>
  </section>
</template>

<style scoped>
@reference "tailwindcss";

.upload-card {
  @apply flex flex-col gap-3.5;
}

.upload-card-head {
  @apply flex items-start justify-between gap-3.5;
}

.upload-card-copy {
  @apply flex flex-col gap-2;
}

.upload-card-copy h3,
.upload-card-copy p {
  @apply m-0;
}

.source-selection {
  @apply flex flex-col gap-2.5;
}

.color-control {
  @apply flex flex-col gap-1.5 text-[0.72rem] font-bold uppercase text-zinc-500;
  letter-spacing: 0.08em;
}

.color-control input {
  @apply h-8 w-[46px] rounded-lg border border-zinc-600 bg-zinc-950 p-1;
}

.upload-success {
  @apply flex flex-col gap-1;
}

.upload-action {
  @apply self-start;
}

@media (max-width: 700px) {
  .upload-card-head {
    @apply flex-col;
  }

  .upload-action {
    @apply w-full;
  }
}
</style>
