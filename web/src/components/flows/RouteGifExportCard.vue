<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  routeCount: number;
  isFiltering: boolean;
  isPreparingPreview: boolean;
  isDownloading: boolean;
  routeColor: string;
  frameDelayMs: number;
  previewUrl: string | null;
  exportError: string | null;
  statusMessage: string | null;
}>();

const emit = defineEmits<{
  (event: "update:routeColor", value: string): void;
  (event: "update:frameDelayMs", value: number): void;
  (event: "export"): void;
}>();

const routeCountLabel = computed(() =>
  props.routeCount === 1 ? "1 ride ready" : `${props.routeCount} rides ready`,
);

const controlsDisabled = computed(
  () => props.isFiltering || props.isDownloading || props.routeCount === 0,
);

const downloadDisabled = computed(
  () => controlsDisabled.value || props.isPreparingPreview || props.previewUrl === null,
);

const downloadButtonLabel = computed(() =>
  props.isDownloading ? "Downloading GIF…" : "Download GIF",
);

const emitFrameDelay = (event: Event) => {
  const nextValue = Number((event.target as HTMLInputElement).value);
  if (!Number.isFinite(nextValue)) {
    return;
  }

  emit("update:frameDelayMs", nextValue);
};

const emitRouteColor = (event: Event) => {
  emit("update:routeColor", (event.target as HTMLInputElement).value);
};
</script>

<template>
  <section class="gif-export-card">
    <div class="gif-export-copy">
      <h3 class="gif-export-title">Step 4: Prepare GIF</h3>
      <p class="gif-export-description">
        Preview the animated export on a black background before downloading it.
      </p>
      <p class="gif-export-meta">{{ routeCountLabel }}</p>
    </div>

    <div class="gif-preview-shell">
      <div class="gif-preview-stage">
        <img
          v-if="previewUrl"
          class="gif-preview-image"
          :src="previewUrl"
          alt="Animated route GIF preview"
        />
        <div v-else class="gif-preview-empty">Preview appears here after the GIF is prepared.</div>
        <div v-if="isPreparingPreview" class="gif-preview-overlay">Rendering preview…</div>
      </div>
    </div>

    <div class="gif-export-controls">
      <label class="gif-field">
        <span class="gif-field-label">Line color</span>
        <div class="gif-color-control">
          <input
            class="gif-color-input"
            type="color"
            :disabled="controlsDisabled"
            :value="routeColor"
            @input="emitRouteColor"
          />
          <span class="gif-color-value">{{ routeColor.toUpperCase() }}</span>
        </div>
      </label>

      <label class="gif-field">
        <span class="gif-field-label">Frame delay (ms)</span>
        <input
          class="gif-field-input"
          type="number"
          min="20"
          max="5000"
          step="5"
          :disabled="controlsDisabled"
          :value="frameDelayMs"
          @input="emitFrameDelay"
        />
        <span class="gif-field-help">Lower values add rides faster.</span>
      </label>
    </div>

    <div class="gif-export-actions">
      <button class="btn btn-primary" :disabled="downloadDisabled" @click="emit('export')">
        {{ downloadButtonLabel }}
      </button>
      <span v-if="statusMessage" class="gif-export-status">{{ statusMessage }}</span>
    </div>

    <div v-if="exportError" class="error-banner gif-export-error">⚠️ {{ exportError }}</div>
  </section>
</template>

<style scoped>
.gif-export-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.gif-export-copy {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.gif-export-title,
.gif-export-description,
.gif-export-meta {
  margin: 0;
}

.gif-export-title {
  color: #fff;
  font-size: 1rem;
}

.gif-export-description {
  color: #b4b4b4;
  font-size: 0.92rem;
  line-height: 1.45;
}

.gif-export-meta {
  color: #ffb347;
  font-size: 0.82rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.gif-preview-shell {
  border-radius: 16px;
  border: 1px solid #2f2f2f;
  background: #0c0c0c;
  padding: 14px;
}

.gif-preview-stage {
  position: relative;
  width: min(100%, 420px);
  aspect-ratio: 1;
  margin: 0 auto;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #1f1f1f;
  background: #000;
}

.gif-preview-image,
.gif-preview-empty,
.gif-preview-overlay {
  position: absolute;
  inset: 0;
}

.gif-preview-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
  image-rendering: auto;
}

.gif-preview-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  color: #8d8d8d;
  font-size: 0.92rem;
  text-align: center;
}

.gif-preview-overlay {
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.66);
  color: #fff;
  font-size: 0.95rem;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.gif-export-controls {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.gif-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.gif-field-label {
  color: #9a9a9a;
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.gif-color-control {
  display: flex;
  align-items: center;
  gap: 12px;
}

.gif-color-input {
  width: 54px;
  height: 42px;
  border: 1px solid #444;
  border-radius: 10px;
  background: #0f0f0f;
  padding: 4px;
}

.gif-color-value {
  color: #e6e6e6;
  font-family: ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.92rem;
}

.gif-field-input {
  border: 1px solid #444;
  border-radius: 8px;
  background: #0f0f0f;
  color: #fff;
  padding: 10px 12px;
  font-size: 0.95rem;
}

.gif-color-input:disabled,
.gif-field-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.gif-field-help {
  color: #888;
  font-size: 0.82rem;
}

.gif-export-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.gif-export-status {
  color: #d7d7d7;
  font-size: 0.9rem;
}

.gif-export-error {
  margin-top: 0;
}
</style>
