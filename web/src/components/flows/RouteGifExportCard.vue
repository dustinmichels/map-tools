<script setup lang="ts">
import { computed, ref, watch } from "vue";

const props = defineProps<{
  routeCount: number;
  isFiltering: boolean;
  isPreparingPreview: boolean;
  isDownloading: boolean;
  routeColor: string;
  flashColor: string;
  frameDelayMs: number;
  previewUrl: string | null;
  exportError: string | null;
  statusMessage: string | null;
  showCityName: boolean;
  cityFont: string;
  cityPosition: string;
  cityNameOverlay: string;
  showDistance: boolean;
  distancePosition: string;
  distanceUnit: string;
  distanceFont: string;
  showDate: boolean;
  datePosition: string;
  dateFont: string;
  dateFormat: "month-day-year" | "month-year";
  exportFormat: "gif" | "webm" | "mp4";
  isMovieExportSupported: boolean;
  isNativeMp4Supported: boolean;
  isTranscodeAvailable: boolean;
}>();
const emit = defineEmits<{
  (event: "update:routeColor", value: string): void;
  (event: "update:flashColor", value: string): void;
  (event: "update:frameDelayMs", value: number): void;
  (event: "update:showCityName", value: boolean): void;
  (event: "update:cityFont", value: string): void;
  (event: "update:cityPosition", value: string): void;
  (event: "update:cityNameOverlay", value: string): void;
  (event: "update:showDistance", value: boolean): void;
  (event: "update:distancePosition", value: string): void;
  (event: "update:distanceUnit", value: string): void;
  (event: "update:distanceFont", value: string): void;
  (event: "update:showDate", value: boolean): void;
  (event: "update:datePosition", value: string): void;
  (event: "update:dateFont", value: string): void;
  (event: "update:dateFormat", value: "month-day-year" | "month-year"): void;
  (event: "update:exportFormat", value: "gif" | "webm" | "mp4"): void;
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

const downloadButtonLabel = computed(() => {
  if (props.isDownloading) {
    if (props.exportFormat === "gif") return "Downloading GIF…";
    if (props.exportFormat === "webm") return "Downloading WebM…";
    return "Downloading MP4…";
  }
  if (props.exportFormat === "gif") return "Download GIF";
  if (props.exportFormat === "webm") return "Download WebM";
  return "Download MP4";
});

const isDelayFocused = ref(false);
const isDurationFocused = ref(false);

const localFrameDelay = ref("");
const localDuration = ref("");
const activeTab = ref<"city" | "distance" | "date">("city");

const formatDuration = (delay: number) => {
  if (props.routeCount === 0) {
    return "0";
  }
  if (props.routeCount === 1) {
    return "1.2";
  }
  return (((props.routeCount - 1) * delay + 1200) / 1000).toFixed(1);
};

const syncFromProps = () => {
  if (!isDelayFocused.value) {
    localFrameDelay.value = props.frameDelayMs.toString();
  }
  if (!isDurationFocused.value) {
    localDuration.value = formatDuration(props.frameDelayMs);
  }
};

watch(() => props.frameDelayMs, syncFromProps, { immediate: true });
watch(() => props.routeCount, syncFromProps);

const onFrameDelayInput = (event: Event) => {
  const rawValue = (event.target as HTMLInputElement).value;
  localFrameDelay.value = rawValue;

  const numericValue = Number(rawValue);
  if (Number.isFinite(numericValue) && rawValue.trim() !== "") {
    emit("update:frameDelayMs", numericValue);

    if (!isDurationFocused.value) {
      localDuration.value = formatDuration(numericValue);
    }
  }
};

const onFrameDelayFocus = () => {
  isDelayFocused.value = true;
};

const toastMessage = ref<string | null>(null);
let toastTimeoutId: number | null = null;

const showToast = (message: string) => {
  if (toastTimeoutId) {
    window.clearTimeout(toastTimeoutId);
  }
  toastMessage.value = message;
  toastTimeoutId = window.setTimeout(() => {
    toastMessage.value = null;
    toastTimeoutId = null;
  }, 4000);
};

const onFrameDelayBlur = () => {
  isDelayFocused.value = false;
  const numericValue = Number(localFrameDelay.value);
  if (Number.isFinite(numericValue) && numericValue < 20) {
    showToast(`The export must be at least ${minDuration.value} seconds long, to maintain reliable rendering.`);
  }
  syncFromProps();
};

const minDuration = computed(() => {
  if (props.routeCount === 0) {
    return 0;
  }
  if (props.routeCount === 1) {
    return 1.2;
  }
  return Number((((props.routeCount - 1) * 20 + 1200) / 1000).toFixed(1));
});

const maxDuration = computed(() => {
  if (props.routeCount === 0) {
    return 0;
  }
  if (props.routeCount === 1) {
    return 1.2;
  }
  return Number((((props.routeCount - 1) * 5000 + 1200) / 1000).toFixed(1));
});

const onDurationInput = (event: Event) => {
  const rawValue = (event.target as HTMLInputElement).value;
  localDuration.value = rawValue;

  const numericValue = Number(rawValue);
  if (Number.isFinite(numericValue) && numericValue > 0 && rawValue.trim() !== "") {
    if (props.routeCount > 1) {
      const calculatedDelay = (numericValue * 1000 - 1200) / (props.routeCount - 1);
      const clampedDelay = Math.min(5000, Math.max(20, calculatedDelay));
      emit("update:frameDelayMs", clampedDelay);

      if (!isDelayFocused.value) {
        localFrameDelay.value = Math.round(clampedDelay).toString();
      }
    }
  }
};

const onDurationFocus = () => {
  isDurationFocused.value = true;
};

const onDurationBlur = () => {
  isDurationFocused.value = false;
  const numericValue = Number(localDuration.value);
  if (Number.isFinite(numericValue) && numericValue < minDuration.value) {
    showToast(`The export must be at least ${minDuration.value} seconds long, to maintain reliable rendering.`);
  }
  syncFromProps();
};

const emitRouteColor = (event: Event) => {
  emit("update:routeColor", (event.target as HTMLInputElement).value);
};

const emitFlashColor = (event: Event) => {
  emit("update:flashColor", (event.target as HTMLInputElement).value);
};

const toggleOverlay = (type: "city" | "distance" | "date", currentVal: boolean) => {
  const newVal = !currentVal;
  if (type === "city") {
    emit("update:showCityName", newVal);
  } else if (type === "distance") {
    emit("update:showDistance", newVal);
  } else if (type === "date") {
    emit("update:showDate", newVal);
  }
  if (newVal) {
    activeTab.value = type;
  }
};
</script>

<template>
  <section class="gif-export-card">
    <div class="gif-export-copy">
      <h3 class="gif-export-title">Step 4: Prepare Export</h3>
      <p class="gif-export-description">
        Preview the animated export on a black background before downloading it.
      </p>
      <p class="gif-export-meta">{{ routeCountLabel }}</p>
    </div>

    <div class="gif-preview-shell">
      <div class="gif-preview-stage">
        <template v-if="previewUrl">
          <video
            v-if="exportFormat === 'webm' || exportFormat === 'mp4'"
            class="gif-preview-image"
            :src="previewUrl"
            autoplay
            loop
            muted
            playsinline
          ></video>
          <img
            v-else
            class="gif-preview-image"
            :src="previewUrl"
            alt="Animated route preview"
          />
        </template>
        <div v-else class="gif-preview-empty">Preview appears here after the export is prepared.</div>
        <div v-if="isPreparingPreview" class="gif-preview-overlay">Rendering preview…</div>
      </div>
    </div>

    <div class="general-gif-settings">
      <h4 class="settings-section-title">General Settings</h4>
      <div class="general-settings-grid">
        <label class="gif-field" v-if="isMovieExportSupported">
          <span class="gif-field-label">Export format</span>
          <select
            class="gif-field-select"
            :disabled="controlsDisabled"
            :value="exportFormat"
            @change="emit('update:exportFormat', ($event.target as HTMLSelectElement).value as 'gif' | 'webm' | 'mp4')"
          >
            <option value="gif">GIF (.gif)</option>
            <option value="webm">WebM Video (.webm)</option>
            <option value="mp4" v-if="isNativeMp4Supported || isTranscodeAvailable">MP4 Video (.mp4)</option>
          </select>
          <span class="gif-field-help">Output file container type.</span>
        </label>

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
          <span class="gif-field-label">Flash color</span>
          <div class="gif-color-control">
            <input
              class="gif-color-input"
              type="color"
              :disabled="controlsDisabled"
              :value="flashColor"
              @input="emitFlashColor"
            />
            <span class="gif-color-value">{{ flashColor.toUpperCase() }}</span>
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
            :value="localFrameDelay"
            @input="onFrameDelayInput"
            @focus="onFrameDelayFocus"
            @blur="onFrameDelayBlur"
          />
          <span class="gif-field-help">Lower values add rides faster.</span>
        </label>

        <label class="gif-field">
          <span class="gif-field-label">Total duration (s)</span>
          <input
            class="gif-field-input"
            type="number"
            :min="minDuration"
            :max="maxDuration"
            step="0.1"
            :disabled="controlsDisabled || routeCount <= 1"
            :value="localDuration"
            @input="onDurationInput"
            @focus="onDurationFocus"
            @blur="onDurationBlur"
          />
          <span class="gif-field-help">Time to draw all routes.</span>
        </label>
      </div>
    </div>

    <div class="overlay-settings-layout">
      <div class="overlay-selector-column">
        <h4 class="settings-section-title">Overlays</h4>
        <div class="overlay-selector-list">
          <div
            class="overlay-selector-item"
            :class="{ active: activeTab === 'city' }"
            @click="activeTab = 'city'"
          >
            <span class="overlay-item-name">City Label</span>
            <button
              type="button"
              class="toggle-switch-btn"
              :class="{ on: showCityName }"
              :disabled="controlsDisabled"
              @click.stop="toggleOverlay('city', showCityName)"
            >
              <span class="toggle-switch-slider"></span>
            </button>
          </div>

          <div
            class="overlay-selector-item"
            :class="{ active: activeTab === 'date' }"
            @click="activeTab = 'date'"
          >
            <span class="overlay-item-name">Date</span>
            <button
              type="button"
              class="toggle-switch-btn"
              :class="{ on: showDate }"
              :disabled="controlsDisabled"
              @click.stop="toggleOverlay('date', showDate)"
            >
              <span class="toggle-switch-slider"></span>
            </button>
          </div>

          <div
            class="overlay-selector-item"
            :class="{ active: activeTab === 'distance' }"
            @click="activeTab = 'distance'"
          >
            <span class="overlay-item-name">Total Distance</span>
            <button
              type="button"
              class="toggle-switch-btn"
              :class="{ on: showDistance }"
              :disabled="controlsDisabled"
              @click.stop="toggleOverlay('distance', showDistance)"
            >
              <span class="toggle-switch-slider"></span>
            </button>
          </div>
        </div>
      </div>

      <div class="overlay-details-column">
        <h4 class="settings-section-title">
          {{ activeTab === 'city' ? 'City Label Settings' : activeTab === 'date' ? 'Date Settings' : 'Distance Settings' }}
        </h4>
        <div class="overlay-details-panel">
          <!-- City Label Config -->
          <div v-show="activeTab === 'city'" class="details-section">
            <div class="details-grid">
              <label class="gif-field">
                <span class="gif-field-label">City label text</span>
                <input
                  class="gif-field-input"
                  type="text"
                  :disabled="controlsDisabled"
                  :value="cityNameOverlay"
                  @input="emit('update:cityNameOverlay', ($event.target as HTMLInputElement).value)"
                />
              </label>

              <label class="gif-field">
                <span class="gif-field-label">Font style</span>
                <select
                  class="gif-field-select"
                  :disabled="controlsDisabled"
                  :value="cityFont"
                  @change="emit('update:cityFont', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="serif">Serif (Georgia)</option>
                  <option value="sans-serif">Sans-Serif</option>
                  <option value="monospace">Monospace</option>
                </select>
              </label>

              <label class="gif-field">
                <span class="gif-field-label">Position</span>
                <select
                  class="gif-field-select"
                  :disabled="controlsDisabled"
                  :value="cityPosition"
                  @change="emit('update:cityPosition', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="bottom-left">Bottom Left</option>
                  <option value="bottom-right">Bottom Right</option>
                  <option value="top-left">Top Left</option>
                  <option value="top-right">Top Right</option>
                </select>
              </label>
            </div>
          </div>

          <!-- Date Config -->
          <div v-show="activeTab === 'date'" class="details-section">
            <div class="details-grid">
              <label class="gif-field">
                <span class="gif-field-label">Date format</span>
                <select
                  class="gif-field-select"
                  :disabled="controlsDisabled"
                  :value="dateFormat"
                  @change="emit('update:dateFormat', ($event.target as HTMLSelectElement).value as any)"
                >
                  <option value="month-day-year">Month day year (e.g. Jan 12, 2026)</option>
                  <option value="month-year">Month, Year (e.g. Jan, 2026)</option>
                </select>
              </label>

              <label class="gif-field">
                <span class="gif-field-label">Font style</span>
                <select
                  class="gif-field-select"
                  :disabled="controlsDisabled"
                  :value="dateFont"
                  @change="emit('update:dateFont', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="serif">Serif (Georgia)</option>
                  <option value="sans-serif">Sans-Serif</option>
                  <option value="monospace">Monospace</option>
                </select>
              </label>

              <label class="gif-field">
                <span class="gif-field-label">Position</span>
                <select
                  class="gif-field-select"
                  :disabled="controlsDisabled"
                  :value="datePosition"
                  @change="emit('update:datePosition', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="bottom-left">Bottom Left</option>
                  <option value="bottom-right">Bottom Right</option>
                  <option value="top-left">Top Left</option>
                  <option value="top-right">Top Right</option>
                </select>
              </label>
            </div>
          </div>
          <!-- Total Distance Config -->
          <div v-show="activeTab === 'distance'" class="details-section">
            <div class="details-grid">
              <label class="gif-field">
                <span class="gif-field-label">Distance unit</span>
                <select
                  class="gif-field-select"
                  :disabled="controlsDisabled"
                  :value="distanceUnit"
                  @change="emit('update:distanceUnit', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="km">Kilometers (km)</option>
                  <option value="miles">Miles (mi)</option>
                </select>
              </label>

              <label class="gif-field">
                <span class="gif-field-label">Font style</span>
                <select
                  class="gif-field-select"
                  :disabled="controlsDisabled"
                  :value="distanceFont"
                  @change="emit('update:distanceFont', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="serif">Serif (Georgia)</option>
                  <option value="sans-serif">Sans-Serif</option>
                  <option value="monospace">Monospace</option>
                </select>
              </label>

              <label class="gif-field">
                <span class="gif-field-label">Position</span>
                <select
                  class="gif-field-select"
                  :disabled="controlsDisabled"
                  :value="distancePosition"
                  @change="emit('update:distancePosition', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="bottom-left">Bottom Left</option>
                  <option value="bottom-right">Bottom Right</option>
                  <option value="top-left">Top Left</option>
                  <option value="top-right">Top Right</option>
                </select>
              </label>
            </div>
          </div>

        </div>
      </div>
    </div>
    <div class="gif-export-actions">
      <button class="btn btn-primary" :disabled="downloadDisabled" @click="emit('export')">
        {{ downloadButtonLabel }}
      </button>
      <span v-if="statusMessage" class="gif-export-status">{{ statusMessage }}</span>
    </div>

    <div v-if="exportError" class="error-banner gif-export-error">⚠️ {{ exportError }}</div>

    <Transition name="toast">
      <div v-if="toastMessage" class="toast-warning">
        <span class="toast-icon">⚠️</span>
        <span class="toast-text">{{ toastMessage }}</span>
      </div>
    </Transition>
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

.settings-section-title {
  margin: 0 0 10px 0;
  font-size: 0.82rem;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-weight: 700;
}

.general-gif-settings {
  border: 1px solid #2f2f2f;
  background: #0c0c0c;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 4px;
}

.general-settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
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
  font-family:
    ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
    "Courier New", monospace;
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

.overlay-settings-layout {
  display: flex;
  gap: 20px;
  border: 1px solid #2f2f2f;
  background: #0c0c0c;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 4px;
}

.overlay-selector-column {
  flex: 1;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #2f2f2f;
  padding-right: 20px;
}

.overlay-details-column {
  flex: 1.5;
  min-width: 250px;
  display: flex;
  flex-direction: column;
}

.overlay-selector-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.overlay-selector-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid #1f1f1f;
  background: #121212;
  cursor: pointer;
  transition: background-color 0.2s, border-color 0.2s;
}

.overlay-selector-item:hover {
  background: #181818;
  border-color: #333;
}

.overlay-selector-item.active {
  background: #1e1e1e;
  border-color: #ff8c00;
}

.overlay-item-name {
  color: #fff;
  font-weight: 500;
  font-size: 0.92rem;
}

.toggle-switch-btn {
  position: relative;
  width: 44px;
  height: 24px;
  background-color: #2a2a2a;
  border-radius: 999px;
  border: 1px solid #444;
  cursor: pointer;
  transition: background-color 0.2s, border-color 0.2s;
  padding: 0;
  display: flex;
  align-items: center;
}

.toggle-switch-btn.on {
  background-color: #ff8c00;
  border-color: #ffb347;
}

.toggle-switch-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toggle-switch-slider {
  position: absolute;
  left: 2px;
  width: 18px;
  height: 18px;
  background-color: #fff;
  border-radius: 50%;
  transition: transform 0.2s;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
}

.toggle-switch-btn.on .toggle-switch-slider {
  transform: translateX(20px);
}

.overlay-details-panel {
  background: #121212;
  border: 1px solid #1f1f1f;
  border-radius: 8px;
  padding: 16px;
  flex: 1;
}

.details-section {
  height: 100%;
}

.details-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

@media (max-width: 640px) {
  .overlay-settings-layout {
    flex-direction: column;
    gap: 16px;
  }
  .overlay-selector-column {
    border-right: none;
    border-bottom: 1px solid #2f2f2f;
    padding-right: 0;
    padding-bottom: 16px;
  }
}

.gif-field-select {
  border: 1px solid #444;
  border-radius: 8px;
  background: #0f0f0f;
  color: #fff;
  padding: 10px 12px;
  font-size: 0.95rem;
  appearance: none;
  background-image: url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e");
  background-repeat: no-repeat;
  background-position: right 12px center;
  background-size: 16px;
  padding-right: 36px;
}

.gif-field-select:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toast-warning {
  position: fixed;
  bottom: 24px;
  right: 24px;
  background-color: #1a0f02;
  border: 1px solid #ff8c00;
  color: #ffb74d;
  padding: 12px 18px;
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.6);
  font-weight: 500;
  font-size: 0.92rem;
  z-index: 9999;
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: 420px;
}

.toast-icon {
  font-size: 1.1rem;
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-enter-from,
.toast-leave-to {
  transform: translateY(20px) scale(0.95);
  opacity: 0;
}

</style>
