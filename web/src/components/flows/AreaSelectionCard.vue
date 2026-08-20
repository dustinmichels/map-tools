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
  selectCity: [payload: SelectedCity];
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
          <label class="control-label">Search for a city</label>
          <SearchCity @select-city="emit('selectCity', $event)" />
        </div>

        <div v-if="props.showCurrentArea" class="city-box area-current">
          <h4>Current area</h4>
          <div class="city-name">{{ props.cityName }}</div>
          <BBoxCoords :bbox="props.bbox" />
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
.area-card {
  gap: 16px;
}

.area-copy {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.area-title,
.area-description {
  margin: 0;
}

.area-toolbar {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
  align-items: start;
}

.area-search {
  min-width: 0;
}

.area-current {
  margin-top: 0;
}
</style>
