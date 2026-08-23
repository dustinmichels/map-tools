import { computed } from "vue";
import { useLocalStorage } from "@vueuse/core";
import { DEFAULT_BOSTON_BBOX, bboxCenter, type BBox } from "../lib/activity";

const STORAGE_CITY_KEY = "map-last-city-name";
const STORAGE_BBOX_KEY = "map-last-bbox";
export const DEFAULT_AREA_CITY_NAME = "Boston, MA, USA";

const isValidBBox = (value: unknown): value is BBox =>
  Array.isArray(value) &&
  value.length === 4 &&
  value.every((part) => typeof part === "number" && Number.isFinite(part)) &&
  value[0] >= -180 &&
  value[0] <= 180 &&
  value[2] >= -180 &&
  value[2] <= 180 &&
  value[1] >= -90 &&
  value[1] <= 90 &&
  value[3] >= -90 &&
  value[3] <= 90 &&
  value[0] <= value[2] &&
  value[1] <= value[3];

const normalizeCityName = (value: unknown) => {
  const normalized = typeof value === "string" ? value.trim() : "";
  return normalized || DEFAULT_AREA_CITY_NAME;
};

const normalizeBBox = (value: unknown): BBox =>
  isValidBBox(value) ? [...value] : [...DEFAULT_BOSTON_BBOX];

export function useStoredAreaSelection() {
  const cityName = useLocalStorage(STORAGE_CITY_KEY, DEFAULT_AREA_CITY_NAME);
  const storedBBox = useLocalStorage<BBox>(STORAGE_BBOX_KEY, [...DEFAULT_BOSTON_BBOX]);

  cityName.value = normalizeCityName(cityName.value);
  storedBBox.value = normalizeBBox(storedBBox.value);

  const bbox = computed<BBox>({
    get: () => normalizeBBox(storedBBox.value),
    set: (value) => {
      storedBBox.value = normalizeBBox(value);
    },
  });
  const center = computed(() => bboxCenter(bbox.value));

  const selectCity = (payload: { name: string; bbox: BBox }) => {
    cityName.value = normalizeCityName(payload.name);
    bbox.value = payload.bbox;
  };

  const resetArea = () => {
    cityName.value = DEFAULT_AREA_CITY_NAME;
    bbox.value = [...DEFAULT_BOSTON_BBOX];
  };

  return {
    cityName,
    bbox,
    center,
    resetArea,
    selectCity,
  };
}
