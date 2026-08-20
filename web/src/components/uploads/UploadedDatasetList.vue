<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { formatCreatedAt, formatFileSize, type UploadedDataset } from "../../lib/activity";

const props = withDefaults(
  defineProps<{
    title: string;
    description?: string;
    uploads: UploadedDataset[];
    selectedDatasetId?: string | null;
    selectable?: boolean;
    manageable?: boolean;
    openable?: boolean;
    emptyMessage?: string;
    actionLabel?: string;
    showManageLink?: boolean;
    busyDatasetId?: string | null;
    limit?: number;
  }>(),
  {
    description: "",
    selectedDatasetId: null,
    selectable: false,
    manageable: false,
    openable: false,
    emptyMessage: "No uploaded GeoParquet files yet.",
    actionLabel: "Use upload",
    showManageLink: false,
    busyDatasetId: null,
  },
);

const emit = defineEmits<{
  select: [upload: UploadedDataset];
  rename: [payload: { datasetId: string; name: string }];
  simplify: [datasetId: string];
  open: [datasetId: string];
  delete: [datasetId: string];
}>();

const sortDirection = ref<"desc" | "asc">("desc");
const editingDatasetId = ref<string | null>(null);
const draftNames = reactive<Record<string, string>>({});

const createdAtValue = (createdAt: string) => {
  const parsed = new Date(createdAt);
  return Number.isNaN(parsed.getTime()) ? 0 : parsed.getTime();
};

const sortedUploads = computed(() =>
  [...props.uploads].sort((left, right) => {
    const delta = createdAtValue(left.createdAt) - createdAtValue(right.createdAt);
    return sortDirection.value === "asc" ? delta : -delta;
  }),
);

const isExpanded = ref(false);

const displayedUploads = computed(() => {
  if (props.limit && !isExpanded.value) {
    return sortedUploads.value.slice(0, props.limit);
  }
  return sortedUploads.value;
});

const startEditing = (upload: UploadedDataset) => {
  draftNames[upload.datasetId] = upload.displayName;
  editingDatasetId.value = upload.datasetId;
};

const stopEditing = () => {
  editingDatasetId.value = null;
};

const submitRename = (datasetId: string) => {
  const name = draftNames[datasetId]?.trim();
  if (!name) {
    return;
  }

  emit("rename", { datasetId, name });
  stopEditing();
};
</script>

<template>
  <div class="upload-library">
    <div class="upload-library-head">
      <div>
        <h3>{{ props.title }}</h3>
        <p v-if="props.description">{{ props.description }}</p>
      </div>
      <label v-if="props.uploads.length > 1" class="sort-control">
        <span>Sort</span>
        <select v-model="sortDirection">
          <option value="desc">Newest first</option>
          <option value="asc">Oldest first</option>
        </select>
      </label>
    </div>

    <p v-if="!props.uploads.length" class="upload-empty">{{ props.emptyMessage }}</p>

    <div v-else class="upload-list">
      <article
        v-for="upload in displayedUploads"
        :key="upload.datasetId"
        class="upload-row"
        :class="{ selected: props.selectedDatasetId === upload.datasetId }"
      >
        <div class="upload-copy">
          <div class="upload-title-row">
            <strong>{{ upload.displayName }}</strong>
            <span v-if="props.selectedDatasetId === upload.datasetId" class="selected-pill"
              >Selected</span
            >
          </div>
          <code>{{ upload.fileName }}</code>
          <div class="upload-meta">
            <span>Created {{ formatCreatedAt(upload.createdAt) }}</span>
            <span>{{ formatFileSize(upload.sizeBytes) }}</span>
            <span v-if="upload.rideCount != null">{{ upload.rideCount }} rides</span>
            <span
              :class="[
                'upload-geometry-status',
                upload.hasSimplified
                  ? 'upload-geometry-status-ready'
                  : 'upload-geometry-status-pending',
              ]"
            >
              {{ upload.hasSimplified ? "Simplified geom ready" : "Needs geom simplification" }}
            </span>
          </div>
        </div>

        <div v-if="editingDatasetId === upload.datasetId" class="upload-actions edit-actions">
          <input
            v-model="draftNames[upload.datasetId]"
            class="rename-input"
            type="text"
            :disabled="props.busyDatasetId === upload.datasetId"
          />
          <button
            class="btn btn-primary"
            :disabled="
              !draftNames[upload.datasetId]?.trim() || props.busyDatasetId === upload.datasetId
            "
            @click="submitRename(upload.datasetId)"
          >
            Save
          </button>
          <button
            class="btn btn-secondary"
            :disabled="props.busyDatasetId === upload.datasetId"
            @click="stopEditing"
          >
            Cancel
          </button>
        </div>

        <div v-else class="upload-actions">
          <button
            v-if="props.selectable"
            class="btn btn-secondary"
            :disabled="props.busyDatasetId === upload.datasetId"
            @click="emit('select', upload)"
          >
            {{
              props.selectedDatasetId === upload.datasetId ? "Using this upload" : props.actionLabel
            }}
          </button>
          <button
            v-if="props.openable"
            class="btn btn-secondary"
            :disabled="props.busyDatasetId === upload.datasetId"
            @click="emit('open', upload.datasetId)"
          >
            Open File
          </button>
          <button
            v-if="props.manageable"
            class="btn btn-secondary"
            :disabled="props.busyDatasetId === upload.datasetId || upload.hasSimplified"
            @click="emit('simplify', upload.datasetId)"
          >
            {{ upload.hasSimplified ? "Geom simplified" : "Simplify geom" }}
          </button>
          <button
            v-if="props.manageable"
            class="btn btn-secondary"
            :disabled="props.busyDatasetId === upload.datasetId"
            @click="startEditing(upload)"
          >
            Rename
          </button>
          <button
            v-if="props.manageable"
            class="btn btn-secondary btn-danger"
            :disabled="props.busyDatasetId === upload.datasetId"
            @click="emit('delete', upload.datasetId)"
          >
            Delete
          </button>
        </div>
      </article>
    </div>
    <div v-if="props.limit && props.uploads.length > props.limit" class="show-more-container">
      <button class="btn btn-secondary" @click="isExpanded = !isExpanded">
        {{ isExpanded ? "Show less" : "Show more" }}
      </button>
    </div>

    <p v-if="props.showManageLink && props.uploads.length" class="manage-link-copy">
      Need to rename or delete files?
      <RouterLink to="/upload" class="link">Manage uploads</RouterLink>.
    </p>
  </div>
</template>

<style scoped>
.upload-library {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.upload-library-head {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  align-items: flex-start;
}

.upload-library-head h3 {
  margin: 0 0 6px;
  color: #fff;
}

.upload-library-head p {
  margin: 0;
  color: #b0b0b0;
  line-height: 1.45;
}

.sort-control {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #888;
}

.sort-control select,
.rename-input {
  background: #111;
  border: 1px solid #444;
  border-radius: 8px;
  color: #fff;
  padding: 8px 10px;
}

.upload-empty {
  margin: 0;
  color: #888;
}
.upload-geometry-status {
  font-size: 0.8rem;
}

.upload-geometry-status-ready {
  color: #81c784;
}

.upload-geometry-status-pending {
  color: #ffb74d;
}

.upload-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 380px;
  overflow-y: auto;
}

.upload-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 14px;
  padding: 12px 14px;
  border-radius: 10px;
  background: #202020;
  border: 1px solid #333;
}

.upload-row.selected {
  border-color: #ff9900;
  box-shadow: inset 0 0 0 1px rgba(255, 153, 0, 0.2);
}

.upload-copy {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.upload-title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.upload-title-row strong {
  color: #fff;
}

.selected-pill {
  border-radius: 999px;
  padding: 3px 9px;
  background: rgba(255, 153, 0, 0.12);
  border: 1px solid rgba(255, 153, 0, 0.4);
  color: #ffb347;
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.upload-copy code {
  width: fit-content;
  max-width: 100%;
  overflow-wrap: anywhere;
}

.upload-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  color: #a0a0a0;
  font-size: 0.88rem;
}

.upload-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.edit-actions {
  min-width: min(320px, 100%);
}

.rename-input {
  min-width: 220px;
}

.btn-danger {
  border-color: rgba(211, 47, 47, 0.55);
  color: #ef5350;
}

.btn-danger:hover:not(:disabled) {
  background: rgba(211, 47, 47, 0.12);
  border-color: #d32f2f;
}

.manage-link-copy {
  margin: 0;
  color: #888;
}

.show-more-container {
  display: flex;
  justify-content: center;
}

@media (max-width: 900px) {
  .upload-row {
    grid-template-columns: 1fr;
  }

  .upload-actions,
  .edit-actions {
    justify-content: flex-start;
  }

  .rename-input {
    min-width: 0;
    width: 100%;
  }
}
</style>
