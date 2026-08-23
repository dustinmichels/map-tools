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
@reference "tailwindcss";

.search-container {
  @apply relative z-10 mx-auto w-full max-w-[600px];
}

.input-wrapper {
  @apply relative flex items-center;
}

.search-input {
  @apply w-full rounded-lg border border-zinc-700 bg-zinc-800 px-4 py-3 pr-12 text-base text-white shadow-md shadow-black/30;
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}

.search-input:focus {
  @apply border-amber-500 outline-none ring-2 ring-amber-500/20;
}

.input-actions {
  @apply absolute right-3.5 flex h-full items-center justify-center;
}

.clear-btn {
  @apply cursor-pointer border-none bg-transparent p-0 text-[20px] leading-none text-zinc-500 transition-colors;
}

.clear-btn:hover {
  @apply text-white;
}

.spinner {
  @apply h-[18px] w-[18px] rounded-full border-2 border-zinc-600 border-t-amber-500;
  animation: spin 0.8s linear infinite;
}

.suggestions-list {
  @apply absolute inset-x-0 top-full z-50 mt-1 max-h-[250px] list-none overflow-y-auto rounded-lg border border-zinc-700 bg-zinc-800 p-0 shadow-xl shadow-black/50;
}

.suggestion-item {
  @apply flex cursor-pointer items-start border-b border-zinc-800 px-4 py-3 transition-colors;
}

.suggestion-item:last-child {
  @apply border-b-0;
}

.suggestion-item:hover {
  @apply bg-zinc-700;
}

.pin-icon {
  @apply mr-2 shrink-0;
}

.display-name {
  @apply break-words text-sm leading-snug text-zinc-200;
}

.error-message {
  @apply mt-2 text-red-500;
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
