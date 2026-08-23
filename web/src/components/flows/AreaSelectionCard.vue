<script setup lang="ts">
import SearchCity from "../SearchCity.vue";
import BBoxCoords from "../BBoxCoords.vue";
import type { BBox, SelectedCity } from "../../lib/activity";

const props = withDefaults(
  defineProps<{
    title: string;
    description: string;
    cityName: string;
    bbox: BBox;
    backLabel?: string;
    showCurrentArea?: boolean;
  }>(),
  {
    backLabel: "Back",
    showCurrentArea: true,
  },
);

const emit = defineEmits<{
  back: [];
  resetArea: [];
  selectCity: [payload: SelectedCity];
  "update:bbox": [bbox: BBox];
}>();
</script>

<template>
  <div class="card-group">
    <section class="card flow-card area-card">
      <div class="area-copy">
        <h2 class="area-title">{{ props.title }}</h2>
        <p class="area-description">{{ props.description }}</p>
      </div>

      <div class="area-toolbar">
        <div class="control-panel area-search">
          <div class="control-header">
            <label class="control-label">Search for a city</label>
            <button
              type="button"
              class="btn btn-secondary area-reset-btn"
              @click="emit('resetArea')"
            >
              Reset to default
            </button>
          </div>
          <SearchCity :selected-name="props.cityName" @select-city="emit('selectCity', $event)" />
        </div>

        <div v-if="props.showCurrentArea" class="city-box area-current">
          <h4>Current area</h4>
          <div class="city-name">{{ props.cityName }}</div>
          <BBoxCoords :bbox="props.bbox" @update:bbox="emit('update:bbox', $event)" />
        </div>
      </div>
      <slot name="details" />

      <div class="card-actions mt-auto">
        <button class="btn btn-secondary" @click="emit('back')">{{ props.backLabel }}</button>
      </div>
    </section>

    <div class="map-container-wrapper">
      <slot />
    </div>
  </div>
</template>

<style scoped>
@reference "tailwindcss";

.area-card {
  @apply gap-4;
}

.area-copy {
  @apply flex flex-col gap-2;
}

.area-title,
.area-description {
  @apply m-0;
}

.area-toolbar {
  @apply grid grid-cols-1 items-start gap-3;
}

.control-header {
  @apply flex flex-wrap items-center justify-between gap-2;
}

.area-search {
  min-width: 0;
}

.area-reset-btn {
  @apply px-3 py-1.5 text-[0.72rem];
}

.area-current {
  @apply mt-0;
}

@media (max-width: 720px) {
  .control-header {
    @apply items-stretch;
  }

  .area-reset-btn {
    @apply w-full;
  }
}
</style>
