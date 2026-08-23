<script setup lang="ts">
import { ref, watch } from "vue";
import { Copy, Check } from "@lucide/vue";

const props = defineProps<{
  bbox: [number, number, number, number]; // [minLng, minLat, maxLng, maxLat]
}>();

const emit = defineEmits<{
  (e: "update:bbox", bbox: [number, number, number, number]): void;
}>();

const inputRef = ref<HTMLInputElement | null>(null);
const rawInput = ref("");
const inputError = ref(false);
const copied = ref(false);
let copyTimeout: ReturnType<typeof setTimeout> | null = null;

const formatBbox = (b: [number, number, number, number]) => {
  return b.map((n) => parseFloat(n.toFixed(6))).join(", ");
};

watch(
  () => props.bbox,
  (newBbox) => {
    if (document.activeElement !== inputRef.value) {
      rawInput.value = formatBbox(newBbox);
      inputError.value = false;
    }
  },
  { immediate: true, deep: true },
);

const handleInput = () => {
  const text = rawInput.value.trim();
  if (!text) {
    inputError.value = false;
    return;
  }

  const parts = text
    .split(/[\s,]+/)
    .map((p) => parseFloat(p))
    .filter((n) => !isNaN(n));

  if (parts.length === 4) {
    const [minLng, minLat, maxLng, maxLat] = parts;
    if (
      minLng >= -180 &&
      minLng <= 180 &&
      maxLng >= -180 &&
      maxLng <= 180 &&
      minLat >= -90 &&
      minLat <= 90 &&
      maxLat >= -90 &&
      maxLat <= 90 &&
      minLng <= maxLng &&
      minLat <= maxLat
    ) {
      inputError.value = false;
      emit("update:bbox", [minLng, minLat, maxLng, maxLat]);
      return;
    }
  }

  inputError.value = true;
};

const handleBlur = () => {
  rawInput.value = formatBbox(props.bbox);
  inputError.value = false;
};

const copyToClipboard = async () => {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(rawInput.value);
    } else {
      const textarea = document.createElement("textarea");
      textarea.value = rawInput.value;
      textarea.setAttribute("readonly", "true");
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      textarea.style.pointerEvents = "none";
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
    }
    copied.value = true;
    if (copyTimeout) clearTimeout(copyTimeout);
    copyTimeout = setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch (err) {
    console.error("Failed to copy coordinates:", err);
  }
};
</script>

<template>
  <div class="bbox-coords">
    <div class="coords-title">Bounds</div>
    <div class="coords-grid">
      <div class="coords-col">
        <div class="coords-row">
          <span class="coord-label">W:</span>
          <span class="coord-value">{{ bbox[0].toFixed(4) }}°</span>
        </div>
        <div class="coords-row">
          <span class="coord-label">E:</span>
          <span class="coord-value">{{ bbox[2].toFixed(4) }}°</span>
        </div>
      </div>
      <div class="coords-col">
        <div class="coords-row">
          <span class="coord-label">S:</span>
          <span class="coord-value">{{ bbox[1].toFixed(4) }}°</span>
        </div>
        <div class="coords-row">
          <span class="coord-label">N:</span>
          <span class="coord-value">{{ bbox[3].toFixed(4) }}°</span>
        </div>
      </div>
    </div>

    <div class="coords-input-section">
      <div class="coords-input-wrapper">
        <input
          ref="inputRef"
          v-model="rawInput"
          type="text"
          class="coords-input"
          :class="{ 'coords-input-error': inputError }"
          placeholder="minLng, minLat, maxLng, maxLat"
          title="Bounding Box coordinates (W, S, E, N)"
          @input="handleInput"
          @blur="handleBlur"
        />
        <button
          type="button"
          class="coords-copy-btn"
          :title="copied ? 'Copied!' : 'Copy to clipboard'"
          @click="copyToClipboard"
        >
          <Check v-if="copied" class="copy-icon text-success" :size="14" />
          <Copy v-else class="copy-icon" :size="14" />
        </button>
      </div>
      <div v-if="inputError" class="coords-error-text">
        Invalid format. Use: minLng, minLat, maxLng, maxLat
      </div>
    </div>
  </div>
</template>

<style scoped>
@reference "tailwindcss";

.bbox-coords {
  @apply mt-3 rounded border border-white/10 bg-black/25 px-2.5 py-2;
}

.coords-title {
  @apply mb-1.5 text-xs font-semibold uppercase text-zinc-500;
  letter-spacing: 0.05em;
}

.coords-grid {
  @apply grid grid-cols-2 gap-3;
}

.coords-col {
  @apply flex flex-col gap-1;
}

.coords-row {
  @apply flex justify-between font-mono text-[0.8rem];
}

.coord-label {
  @apply mr-1 text-zinc-400;
}

.coord-value {
  @apply font-semibold text-amber-500;
}

.coords-input-section {
  @apply mt-2.5 flex flex-col gap-1;
}

.coords-input-wrapper {
  @apply relative flex w-full items-center;
}

.coords-input {
  @apply w-full rounded border border-zinc-700 bg-zinc-800 px-2 py-1.5 pr-8 font-mono text-[0.72rem] text-white;
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}

.coords-input:focus {
  @apply border-amber-500 outline-none ring-2 ring-amber-500/20;
}

.coords-input-error {
  @apply border-red-500 ring-2 ring-red-500/20;
}

.coords-copy-btn {
  @apply absolute right-1 flex items-center justify-center border-none bg-transparent p-1 text-zinc-400 transition-colors;
}

.coords-copy-btn:hover {
  @apply text-amber-500;
}

.coords-copy-btn .copy-icon {
  @apply h-[14px] w-[14px];
}

.coords-copy-btn .text-success {
  @apply text-green-500;
}

.coords-error-text {
  @apply text-[0.68rem] text-red-500;
}
</style>
