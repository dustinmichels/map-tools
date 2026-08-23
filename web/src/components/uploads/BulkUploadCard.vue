<script setup lang="ts">
import { computed, ref } from "vue";
import { useDropZone } from "@vueuse/core";
import { formatFileSize } from "../../lib/activity";

const props = defineProps<{
  selectedFiles: File[];
  isUploading: boolean;
  uploadError: string | null;
  uploadSummary: string | null;
}>();

const emit = defineEmits<{
  selectFiles: [files: File[]];
  upload: [];
}>();

const dropZoneRef = ref<HTMLDivElement>();
const { isOverDropZone } = useDropZone(dropZoneRef, (files) => {
  if (files) {
    selectFiles(files);
  }
});

const selectedFilesLabel = computed(() => {
  if (!props.selectedFiles.length) {
    return "Choose one or more ZIPs, or drop them here.";
  }

  if (props.selectedFiles.length === 1) {
    const [file] = props.selectedFiles;
    return `${file.name} (${formatFileSize(file.size)})`;
  }

  return `${props.selectedFiles.length} ZIP files selected`;
});

const selectFiles = (files: FileList | File[] | null) => {
  emit("selectFiles", files ? Array.from(files) : []);
};

const fileListChanged = (event: Event) => {
  const target = event.target as HTMLInputElement;
  selectFiles(target.files);
};
</script>

<template>
  <section class="card upload-card">
    <div class="upload-card-head">
      <div class="upload-card-copy">
        <h2>Process Strava ZIPs</h2>
        <p>Select one or more bulk export ZIPs and save them as reusable GeoParquet datasets.</p>
      </div>
    </div>

    <div ref="dropZoneRef" class="upload-zone" :class="{ dragging: isOverDropZone }">
      <input type="file" accept=".zip" multiple class="file-input" @change="fileListChanged" />
      <label class="upload-label">
        <span class="upload-icon">📦</span>
        <span class="file-name">{{ selectedFilesLabel }}</span>
      </label>
    </div>

    <ul v-if="selectedFiles.length > 1" class="selected-file-list">
      <li v-for="file in selectedFiles" :key="`${file.name}-${file.size}`">
        <span>{{ file.name }}</span>
        <span>{{ formatFileSize(file.size) }}</span>
      </li>
    </ul>

    <div v-if="uploadError" class="error-banner">⚠️ {{ uploadError }}</div>

    <div v-if="isUploading" class="progress-container">
      <div class="progress-spinner"></div>
      <span>Processing ZIPs and writing saved GeoParquet datasets…</span>
    </div>

    <div v-if="uploadSummary" class="success-banner upload-success">
      <strong>Done.</strong>
      <span>{{ uploadSummary }}</span>
    </div>

    <button
      class="btn btn-primary upload-action"
      :disabled="!selectedFiles.length || isUploading"
      @click="emit('upload')"
    >
      Process selected ZIPs
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

.upload-card-copy h2,
.upload-card-copy p {
  @apply m-0;
}

.selected-file-list {
  @apply m-0 flex list-none flex-col gap-2 p-0;
}

.selected-file-list li {
  @apply flex justify-between gap-3 rounded-xl border border-zinc-700 bg-zinc-800 px-3 py-2.5 text-[0.92rem] text-zinc-300;
}

.upload-success {
  @apply flex flex-col gap-1;
}

.upload-action {
  @apply self-start;
}

@media (max-width: 700px) {
  .upload-action {
    @apply w-full;
  }
}
</style>
