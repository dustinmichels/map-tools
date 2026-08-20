<script setup lang="ts">
import { ref, useSlots } from "vue";
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
const isDragging = ref(false);

const fileListChanged = (event: Event) => {
  const target = event.target as HTMLInputElement;
  emit("selectFile", target.files?.[0] ?? null);
};

const droppedFile = (event: DragEvent) => {
  isDragging.value = false;
  emit("selectFile", event.dataTransfer?.files?.[0] ?? null);
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

    <div
      class="upload-zone"
      :class="{ dragging: isDragging }"
      @dragover.prevent="isDragging = true"
      @dragleave.prevent="isDragging = false"
      @drop.prevent="droppedFile"
    >
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
.upload-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.upload-card-head {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  align-items: flex-start;
}

.upload-card-copy {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.upload-card-copy h3,
.upload-card-copy p {
  margin: 0;
}

.source-selection {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.color-control {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #888;
}

.color-control input {
  width: 46px;
  height: 32px;
  border: 1px solid #444;
  border-radius: 8px;
  background: #111;
  padding: 4px;
}

.upload-success {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.upload-action {
  align-self: flex-start;
}

@media (max-width: 700px) {
  .upload-card-head {
    flex-direction: column;
  }

  .upload-action {
    width: 100%;
  }
}
</style>
