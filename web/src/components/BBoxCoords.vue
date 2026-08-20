<script setup lang="ts">
import { ref, watch } from "vue";
import { Copy, Check } from "lucide-vue-next";

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
  { immediate: true, deep: true }
);

const handleInput = () => {
  const text = rawInput.value.trim();
  if (!text) {
    inputError.value = false;
    return;
  }

  const parts = text.split(/[\s,]+/).map((p) => parseFloat(p)).filter((n) => !isNaN(n));

  if (parts.length === 4) {
    const [minLng, minLat, maxLng, maxLat] = parts;
    if (
      minLng >= -180 && minLng <= 180 &&
      maxLng >= -180 && maxLng <= 180 &&
      minLat >= -90 && minLat <= 90 &&
      maxLat >= -90 && maxLat <= 90 &&
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
      <div v-slot:default v-if="inputError" class="coords-error-text">
        Invalid format. Use: minLng, minLat, maxLng, maxLat
      </div>
    </div>
  </div>
</template>

<style scoped>
.bbox-coords {
  margin-top: 12px;
  padding: 8px 10px;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}

.coords-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: #888;
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.coords-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.coords-col {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.coords-row {
  display: flex;
  justify-content: space-between;
  font-family: monospace;
  font-size: 0.8rem;
}

.coord-label {
  color: #aaa;
  margin-right: 4px;
}

.coord-value {
  color: #ff9900;
  font-weight: 600;
}

.coords-input-section {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.coords-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  width: 100%;
}

.coords-input {
  width: 100%;
  padding: 6px 30px 6px 8px;
  font-family: monospace;
  font-size: 0.72rem;
  background: #1e1e1e;
  border: 1px solid #333;
  border-radius: 4px;
  color: #fff;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.coords-input:focus {
  outline: none;
  border-color: #ff9900;
  box-shadow: 0 0 0 2px rgba(255, 153, 0, 0.2);
}

.coords-input-error {
  border-color: #ff4444 !important;
  box-shadow: 0 0 0 2px rgba(255, 68, 68, 0.2) !important;
}

.coords-copy-btn {
  position: absolute;
  right: 4px;
  background: none;
  border: none;
  color: #aaa;
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.15s;
}

.coords-copy-btn:hover {
  color: #ff9900;
}

.coords-copy-btn .copy-icon {
  width: 14px;
  height: 14px;
}

.coords-copy-btn .text-success {
  color: #22c55e;
}

.coords-error-text {
  font-size: 0.68rem;
  color: #ff4444;
}
</style>

