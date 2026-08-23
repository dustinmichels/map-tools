<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { RotateCcw } from "lucide-vue-next";

const DEFAULT_ROUTE_COLOR = "#ff8c00";
const DEFAULT_FLASH_COLOR = "#ffffff";

const props = defineProps<{
  routeCount: number;
  previewRouteCount: number;
  previewTargetCount: number;
  previewProgressCount: number;
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
const previewSummary = computed(() => {
  if (props.routeCount === 0) {
    return null;
  }

  if (props.previewRouteCount >= props.routeCount) {
    return `Showing all ${props.routeCount} rides in the browser preview.`;
  }

  if (!props.previewUrl) {
    return `Rendering first ${props.previewTargetCount} of ${props.routeCount} rides for the preview.`;
  }

  if (props.isPreparingPreview) {
    return `Playing first ${props.previewRouteCount} of ${props.routeCount} rides while rendering the rest of the preview in the background.`;
  }

  return `Showing first ${props.previewRouteCount} of ${props.routeCount} rides in the browser preview.`;
});
const previewReadyPercent = computed(() => {
  if (props.routeCount === 0) {
    return 0;
  }

  return (props.previewRouteCount / props.routeCount) * 100;
});

const previewTargetPercent = computed(() => {
  if (props.routeCount === 0) {
    return 0;
  }

  return (Math.max(props.previewRouteCount, props.previewTargetCount) / props.routeCount) * 100;
});

const previewProgressPercent = computed(() => {
  if (props.routeCount === 0) {
    return 0;
  }

  return (Math.max(props.previewRouteCount, props.previewProgressCount) / props.routeCount) * 100;
});

const previewProgressLabel = computed(() => {
  if (props.routeCount === 0) {
    return null;
  }

  if (props.previewRouteCount >= props.routeCount) {
    return `${props.routeCount} of ${props.routeCount} rides ready`;
  }

  if (!props.previewUrl) {
    return `Preparing first ${props.previewTargetCount} rides`;
  }

  if (props.isPreparingPreview) {
    return `${Math.max(props.previewRouteCount, props.previewProgressCount)} of ${props.routeCount} rides rendered`;
  }

  return `${props.previewRouteCount} of ${props.routeCount} rides ready`;
});

const previewProgressDetail = computed(() => {
  if (
    props.routeCount === 0 ||
    !props.isPreparingPreview ||
    props.previewRouteCount >= props.routeCount
  ) {
    return null;
  }

  if (!props.previewUrl) {
    return "Building the first playable preview.";
  }

  return `Current preview: ${props.previewRouteCount} rides · Rendering the remaining ${props.routeCount - props.previewRouteCount} rides`;
});

const isDelayFocused = ref(false);
const isDurationFocused = ref(false);

const localFrameDelay = ref("");
const localDuration = ref("");
const activeTab = ref<"city" | "distance" | "date">("city");

const getPlaybackDurationMs = (delay: number) => {
  if (props.routeCount === 0) {
    return 0;
  }

  return (props.routeCount + 1) * delay + 1200;
};

const formatDuration = (delay: number) => {
  if (props.routeCount === 0) {
    return "0";
  }

  return (getPlaybackDurationMs(delay) / 1000).toFixed(1);
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
    showToast(
      `The export must be at least ${minDuration.value} seconds long, to maintain reliable rendering.`,
    );
  }
  syncFromProps();
};

const minDuration = computed(() => {
  if (props.routeCount === 0) {
    return 0;
  }

  return Number((getPlaybackDurationMs(20) / 1000).toFixed(1));
});

const maxDuration = computed(() => {
  if (props.routeCount === 0) {
    return 0;
  }

  return Number((getPlaybackDurationMs(5000) / 1000).toFixed(1));
});

const onDurationInput = (event: Event) => {
  const rawValue = (event.target as HTMLInputElement).value;
  localDuration.value = rawValue;
  const numericValue = Number(rawValue);

  if (Number.isFinite(numericValue) && numericValue > 0 && rawValue.trim() !== "") {
    if (props.routeCount > 0) {
      const calculatedDelay = (numericValue * 1000 - 1200) / (props.routeCount + 1);
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
    showToast(
      `The export must be at least ${minDuration.value} seconds long, to maintain reliable rendering.`,
    );
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
            :key="previewUrl"
            class="gif-preview-image"
            :src="previewUrl"
            autoplay
            loop
            muted
            playsinline
          ></video>
          <img
            v-else
            :key="previewUrl"
            class="gif-preview-image"
            :src="previewUrl"
            alt="Animated route preview"
          />
        </template>
        <div v-else class="gif-preview-empty">
          Preview appears here after the export is prepared.
        </div>
        <div v-if="isPreparingPreview && !previewUrl" class="gif-preview-overlay">
          Rendering preview…
        </div>
      </div>
    </div>
    <div v-if="routeCount > 0" class="gif-preview-progress">
      <div class="gif-preview-progress-copy">
        <span class="gif-preview-progress-label">{{ previewProgressLabel }}</span>
        <span v-if="previewProgressDetail" class="gif-preview-progress-detail">{{
          previewProgressDetail
        }}</span>
      </div>
      <div class="gif-preview-progress-bar" aria-hidden="true">
        <span
          class="gif-preview-progress-target"
          :style="{ width: `${previewTargetPercent}%` }"
        ></span>
        <span
          class="gif-preview-progress-ready"
          :style="{ width: `${previewReadyPercent}%` }"
        ></span>
        <span
          v-if="isPreparingPreview"
          class="gif-preview-progress-active"
          :style="{ width: `${previewProgressPercent}%` }"
        ></span>
      </div>
    </div>
    <div v-if="previewSummary" class="gif-preview-summary">
      <span>{{ previewSummary }}</span>
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
            @change="
              emit(
                'update:exportFormat',
                ($event.target as HTMLSelectElement).value as 'gif' | 'webm' | 'mp4',
              )
            "
          >
            <option value="gif">GIF (.gif)</option>
            <option value="webm">WebM Video (.webm)</option>
            <option value="mp4" v-if="isNativeMp4Supported || isTranscodeAvailable">
              MP4 Video (.mp4)
            </option>
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
            <button
              v-if="routeColor.toLowerCase() !== DEFAULT_ROUTE_COLOR"
              type="button"
              class="gif-color-reset-btn"
              title="Reset to default"
              :disabled="controlsDisabled"
              @click="emit('update:routeColor', DEFAULT_ROUTE_COLOR)"
            >
              <RotateCcw :size="14" />
            </button>
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
            <button
              v-if="flashColor.toLowerCase() !== DEFAULT_FLASH_COLOR"
              type="button"
              class="gif-color-reset-btn"
              title="Reset to default"
              :disabled="controlsDisabled"
              @click="emit('update:flashColor', DEFAULT_FLASH_COLOR)"
            >
              <RotateCcw :size="14" />
            </button>
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
          <span class="gif-field-help">Time to draw all routes in the export.</span>
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
          {{
            activeTab === "city"
              ? "City Label Settings"
              : activeTab === "date"
                ? "Date Settings"
                : "Distance Settings"
          }}
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
                  @change="
                    emit('update:dateFormat', ($event.target as HTMLSelectElement).value as any)
                  "
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
                  @change="
                    emit('update:distancePosition', ($event.target as HTMLSelectElement).value)
                  "
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
@reference "tailwindcss";

.gif-export-card {
  @apply flex flex-col gap-4;
}

.gif-export-copy {
  @apply flex flex-col gap-1.5;
}

.gif-export-title,
.gif-export-description,
.gif-export-meta {
  @apply m-0;
}

.gif-export-title {
  @apply text-base text-white;
}

.gif-export-description {
  @apply text-[0.92rem] leading-relaxed text-zinc-400;
}

.gif-export-meta {
  @apply text-[0.82rem] font-semibold uppercase text-amber-300;
  letter-spacing: 0.04em;
}

.gif-preview-shell {
  @apply rounded-2xl border border-zinc-800 bg-black/95 p-3.5;
}

.gif-preview-stage {
  position: relative;
  width: min(100%, 420px);
  aspect-ratio: 1;
  @apply mx-auto overflow-hidden rounded-xl border border-zinc-900 bg-black;
}

.gif-preview-image,
.gif-preview-empty,
.gif-preview-overlay {
  @apply absolute inset-0;
}

.gif-preview-image {
  @apply h-full w-full object-contain;
  image-rendering: auto;
}

.gif-preview-empty {
  @apply flex items-center justify-center px-6 text-center text-[0.92rem] text-zinc-500;
}

.gif-preview-overlay {
  @apply flex items-center justify-center bg-black/65 text-[0.95rem] font-semibold text-white;
  letter-spacing: 0.02em;
}

.gif-preview-progress {
  @apply flex flex-col gap-2;
}

.gif-preview-progress-copy {
  @apply flex flex-wrap items-center justify-between gap-2 text-[0.82rem] text-zinc-400;
}

.gif-preview-progress-label {
  @apply font-semibold text-zinc-200;
}

.gif-preview-progress-detail {
  @apply text-zinc-500;
}

.gif-preview-progress-bar {
  position: relative;
  height: 0.75rem;
  @apply overflow-hidden rounded-full bg-zinc-900;
}

.gif-preview-progress-target,
.gif-preview-progress-ready,
.gif-preview-progress-active {
  position: absolute;
  inset: 0 auto 0 0;
  @apply rounded-full;
  transition: width 160ms ease;
}

.gif-preview-progress-target {
  @apply bg-amber-950/70;
}

.gif-preview-progress-ready {
  @apply bg-amber-700/65;
}

.gif-preview-progress-active {
  @apply bg-amber-400;
}

.gif-preview-summary {
  @apply rounded-xl border border-zinc-800 bg-zinc-950/70 px-4 py-3 text-[0.9rem] text-zinc-300;
}

.settings-section-title {
  @apply mb-2.5 mt-0 text-[0.82rem] font-bold uppercase text-zinc-500;
  letter-spacing: 0.08em;
}

.general-gif-settings {
  @apply mb-1 rounded-xl border border-zinc-800 bg-black/95 p-4;
}

.general-settings-grid {
  @apply grid gap-3;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
}

.gif-field {
  @apply flex flex-col gap-1.5;
}

.gif-field-label {
  @apply text-[0.78rem] font-bold uppercase text-zinc-400;
  letter-spacing: 0.08em;
}

.gif-color-control {
  @apply flex items-center gap-3;
}

.gif-color-input {
  @apply h-[42px] w-[54px] rounded-[10px] border border-zinc-600 bg-zinc-950 p-1;
}

.gif-color-value {
  @apply text-[0.92rem] text-zinc-100;
  font-family:
    ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New",
    monospace;
}

.gif-color-reset-btn {
  @apply flex h-7 w-7 items-center justify-center rounded-md border border-zinc-700 bg-zinc-900 text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-zinc-200 disabled:cursor-not-allowed disabled:opacity-40;
}

.gif-field-input {
  @apply rounded-lg border border-zinc-600 bg-zinc-950 px-3 py-2.5 text-[0.95rem] text-white;
}

.gif-color-input:disabled,
.gif-field-input:disabled,
.gif-field-select:disabled {
  @apply cursor-not-allowed opacity-60;
}

.gif-field-help {
  @apply text-[0.82rem] text-zinc-500;
}

.gif-export-actions {
  @apply flex flex-wrap items-center gap-3;
}

.gif-export-status {
  @apply text-[0.9rem] text-zinc-200;
}

.gif-export-error {
  @apply mt-0;
}

.overlay-settings-layout {
  @apply mb-1 flex gap-5 rounded-xl border border-zinc-800 bg-black/95 p-4;
}

.overlay-selector-column {
  @apply flex min-w-[200px] flex-1 flex-col border-r border-zinc-800 pr-5;
}

.overlay-details-column {
  @apply flex min-w-[250px] flex-[1.5] flex-col;
}

.overlay-selector-list {
  @apply flex flex-col gap-2;
}

.overlay-selector-item {
  @apply flex cursor-pointer items-center justify-between rounded-lg border border-zinc-900 bg-zinc-950 px-3 py-2.5 transition-colors;
}

.overlay-selector-item:hover {
  @apply border-zinc-700 bg-zinc-900;
}

.overlay-selector-item.active {
  @apply border-amber-500 bg-zinc-800;
}

.overlay-item-name {
  @apply text-[0.92rem] font-medium text-white;
}

.toggle-switch-btn {
  @apply relative flex h-6 w-11 items-center rounded-full border border-zinc-600 bg-zinc-800 p-0 transition-colors;
}

.toggle-switch-btn.on {
  @apply border-amber-300 bg-amber-500;
}

.toggle-switch-btn:disabled {
  @apply cursor-not-allowed opacity-50;
}

.toggle-switch-slider {
  @apply absolute left-0.5 h-[18px] w-[18px] rounded-full bg-white shadow-sm shadow-black/40 transition-transform;
}

.toggle-switch-btn.on .toggle-switch-slider {
  transform: translateX(20px);
}

.overlay-details-panel {
  @apply flex-1 rounded-lg border border-zinc-900 bg-zinc-950 p-4;
}

.details-section {
  @apply h-full;
}

.details-grid {
  @apply flex flex-col gap-3;
}

.gif-field-select {
  @apply rounded-lg border border-zinc-600 bg-zinc-950 px-3 py-2.5 pr-9 text-[0.95rem] text-white;
  appearance: none;
  background-image: url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e");
  background-repeat: no-repeat;
  background-position: right 12px center;
  background-size: 16px;
}

@media (max-width: 640px) {
  .overlay-settings-layout {
    @apply flex-col gap-4;
  }

  .overlay-selector-column {
    @apply border-b border-r-0 border-zinc-800 pb-4 pr-0;
  }
}

.toast-warning {
  @apply fixed bottom-6 right-6 z-[9999] flex max-w-[420px] items-center gap-2.5 rounded-lg border border-amber-500 bg-amber-950/80 px-[18px] py-3 text-[0.92rem] font-medium text-amber-300 shadow-2xl shadow-black/60;
}

.toast-icon {
  @apply text-[1.1rem];
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
