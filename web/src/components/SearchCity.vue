<script setup lang="ts">
import { onUnmounted, ref, watch } from "vue";

interface NominatimResult {
  place_id: number;
  boundingbox: string[];
  lat: string;
  lon: string;
  display_name: string;
}

const emit = defineEmits<{
  (
    event: "select-city",
    payload: {
      name: string;
      bbox: [number, number, number, number];
      lat: number;
      lon: number;
    },
  ): void;
}>();

const query = ref("");
const suggestions = ref<NominatimResult[]>([]);
const loading = ref(false);
const showError = ref(false);
let debounceTimeout: ReturnType<typeof setTimeout> | null = null;
let latestRequestId = 0;
let suppressNextWatch = false;

const searchCities = async (value: string) => {
  if (!value.trim()) {
    suggestions.value = [];
    return;
  }

  const currentRequestId = ++latestRequestId;
  loading.value = true;
  showError.value = false;

  try {
    const url = `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(value)}&limit=6`;
    const res = await fetch(url, {
      headers: {
        "User-Agent": "MapTools-App/1.0",
      },
    });
    if (!res.ok) {
      throw new Error("Search failed");
    }

    const data = (await res.json()) as NominatimResult[];
    if (currentRequestId === latestRequestId) {
      suggestions.value = data;
    }
  } catch (error) {
    if (currentRequestId === latestRequestId) {
      console.error(error);
      showError.value = true;
    }
  } finally {
    if (currentRequestId === latestRequestId) {
      loading.value = false;
    }
  }
};

watch(query, (value) => {
  if (debounceTimeout) {
    clearTimeout(debounceTimeout);
  }
  if (suppressNextWatch) {
    suppressNextWatch = false;
    return;
  }
  if (!value.trim()) {
    suggestions.value = [];
    return;
  }

  debounceTimeout = setTimeout(() => {
    void searchCities(value);
  }, 400);
});

onUnmounted(() => {
  if (debounceTimeout) {
    clearTimeout(debounceTimeout);
  }
});

const selectSuggestion = (item: NominatimResult) => {
  latestRequestId += 1;
  suppressNextWatch = true;
  query.value = item.display_name;
  suggestions.value = [];

  const minLat = parseFloat(item.boundingbox[0]);
  const maxLat = parseFloat(item.boundingbox[1]);
  const minLng = parseFloat(item.boundingbox[2]);
  const maxLng = parseFloat(item.boundingbox[3]);

  emit("select-city", {
    name: item.display_name,
    bbox: [minLng, minLat, maxLng, maxLat],
    lat: parseFloat(item.lat),
    lon: parseFloat(item.lon),
  });
};
const handleEnter = (event: KeyboardEvent) => {
  if (suggestions.value.length > 0) {
    event.preventDefault();
    event.stopPropagation();
    selectSuggestion(suggestions.value[0]);
  }
};

const clearInput = () => {
  latestRequestId += 1;
  query.value = "";
  suggestions.value = [];
};
</script>

<template>
  <div class="search-container">
    <div class="input-wrapper">
      <input
        v-model="query"
        type="text"
        placeholder="Type a city name (e.g. San Francisco, Amsterdam)..."
        class="search-input"
        @keydown.enter="handleEnter"
      />
      <div class="input-actions">
        <span v-if="loading" class="spinner"></span>
        <button v-else-if="query" class="clear-btn" title="Clear input" @click="clearInput">
          &times;
        </button>
      </div>
    </div>

    <ul v-if="suggestions.length > 0" class="suggestions-list">
      <li
        v-for="item in suggestions"
        :key="item.place_id"
        class="suggestion-item"
        @click="selectSuggestion(item)"
      >
        <span class="pin-icon">📍</span>
        <span class="display-name">{{ item.display_name }}</span>
      </li>
    </ul>

    <div v-if="showError" class="error-message">
      Could not fetch search results. Please try again.
    </div>
  </div>
</template>

<style scoped>
.search-container {
  position: relative;
  width: 100%;
  max-width: 600px;
  margin: 0 auto;
  z-index: 10;
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.search-input {
  width: 100%;
  padding: 12px 48px 12px 16px;
  font-size: 16px;
  background: #1e1e1e;
  border: 1px solid #333;
  border-radius: 8px;
  color: #fff;
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}

.search-input:focus {
  outline: none;
  border-color: #ff9900;
  box-shadow:
    0 0 0 2px rgba(255, 153, 0, 0.2),
    0 4px 6px rgba(0, 0, 0, 0.3);
}

.input-actions {
  position: absolute;
  right: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.clear-btn {
  background: none;
  border: none;
  color: #888;
  font-size: 20px;
  cursor: pointer;
  padding: 0;
  line-height: 1;
}

.clear-btn:hover {
  color: #fff;
}

.spinner {
  width: 18px;
  height: 18px;
  border: 2px solid #555;
  border-top-color: #ff9900;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.suggestions-list {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin: 4px 0 0;
  padding: 0;
  background: #1e1e1e;
  border: 1px solid #333;
  border-radius: 8px;
  list-style: none;
  max-height: 250px;
  overflow-y: auto;
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.5);
  z-index: 100;
}

.suggestion-item {
  display: flex;
  align-items: flex-start;
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid #2a2a2a;
  transition: background 0.15s;
}

.suggestion-item:last-child {
  border-bottom: none;
}

.suggestion-item:hover {
  background: #2a2a2a;
}

.pin-icon {
  margin-right: 8px;
  flex-shrink: 0;
}

.display-name {
  color: #e0e0e0;
  font-size: 14px;
  line-height: 1.4;
  word-break: break-word;
}

.error-message {
  margin-top: 8px;
  color: #ff4444;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
