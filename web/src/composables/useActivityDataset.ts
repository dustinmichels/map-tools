import { computed, ref } from "vue";
import type {
  BBox,
  GeoJSONFeatureCollection,
  GeometryMode,
  UploadSummary,
  UploadedDataset,
} from "../lib/activity";
import { uploadArchive } from "../lib/uploads";

const ZIP_ERROR_MESSAGE = "Please select a .zip archive.";

interface FilterActivitiesOptions {
  bbox?: BBox | null;
  geometryMode?: GeometryMode;
  preserveResults?: boolean;
}

export function useActivityDataset() {
  const selectedFile = ref<File | null>(null);
  const isUploading = ref(false);
  const uploadError = ref<string | null>(null);
  const uploadSuccess = ref(false);
  const sessionId = ref<string | null>(null);
  const totalCount = ref<number | null>(null);
  const parsedCount = ref<number | null>(null);
  const rideCount = ref<number | null>(null);
  const activeDataset = ref<UploadedDataset | null>(null);
  const usingExistingDataset = ref(false);

  const isFiltering = ref(false);
  const filterError = ref<string | null>(null);
  const activitiesCount = ref<number | null>(null);
  const activitiesGeoJSON = ref<GeoJSONFeatureCollection | null>(null);

  const clearFilterState = () => {
    filterError.value = null;
    activitiesCount.value = null;
    activitiesGeoJSON.value = null;
  };

  const clearUploadState = () => {
    uploadSuccess.value = false;
    sessionId.value = null;
    totalCount.value = null;
    parsedCount.value = null;
    rideCount.value = null;
    activeDataset.value = null;
    usingExistingDataset.value = false;
    clearFilterState();
  };

  const applyUploadedDataset = (upload: UploadedDataset, fromExisting: boolean) => {
    sessionId.value = upload.datasetId;
    totalCount.value = upload.total ?? null;
    parsedCount.value = upload.parsed ?? null;
    rideCount.value = upload.rideCount ?? null;
    activeDataset.value = upload;
    usingExistingDataset.value = fromExisting;
    uploadSuccess.value = true;
    clearFilterState();
  };

  const setSelectedFile = (file: File | null) => {
    if (!file) {
      selectedFile.value = null;
      uploadError.value = null;
      clearUploadState();
      return;
    }

    if (!file.name.toLowerCase().endsWith(".zip")) {
      uploadError.value = ZIP_ERROR_MESSAGE;
      return;
    }

    selectedFile.value = file;
    uploadError.value = null;
    clearUploadState();
  };

  const submitZip = async () => {
    if (!selectedFile.value) {
      return null;
    }

    isUploading.value = true;
    uploadError.value = null;
    clearUploadState();

    try {
      const data = (await uploadArchive(selectedFile.value)) as UploadSummary;
      applyUploadedDataset(data.dataset, false);
      totalCount.value = data.total;
      parsedCount.value = data.parsed;
      rideCount.value = data.rideCount;
      return data;
    } catch (error) {
      console.error(error);
      uploadError.value =
        error instanceof Error ? error.message : "An error occurred during upload.";
      return null;
    } finally {
      isUploading.value = false;
    }
  };

  const useExistingDataset = (upload: UploadedDataset) => {
    selectedFile.value = null;
    uploadError.value = null;
    clearUploadState();
    applyUploadedDataset(upload, true);
  };

  const filterActivities = async (options: FilterActivitiesOptions = {}) => {
    if (!sessionId.value) {
      return;
    }

    const { bbox = null, geometryMode = "simplified", preserveResults = false } = options;

    isFiltering.value = true;
    if (preserveResults) {
      filterError.value = null;
    } else {
      clearFilterState();
    }

    try {
      const res = await fetch("/api/filter", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          sessionId: sessionId.value,
          ...(bbox ? { bbox } : {}),
          geometryMode,
        }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `Server returned status ${res.status}`);
      }

      const geoJSON = (await res.json()) as GeoJSONFeatureCollection;
      filterError.value = null;
      activitiesGeoJSON.value = geoJSON;
      activitiesCount.value = geoJSON.features.length;
    } catch (error) {
      console.error(error);
      filterError.value =
        error instanceof Error ? error.message : "An error occurred while filtering activities.";
    } finally {
      isFiltering.value = false;
    }
  };

  const reset = () => {
    selectedFile.value = null;
    uploadError.value = null;
    isUploading.value = false;
    isFiltering.value = false;
    clearUploadState();
  };

  return {
    selectedFile,
    isUploading,
    uploadError,
    uploadSuccess,
    sessionId,
    totalCount,
    parsedCount,
    rideCount,
    activeDataset,
    usingExistingDataset,
    isFiltering,
    filterError,
    activitiesCount,
    activitiesGeoJSON,
    readyToFilter: computed(() => uploadSuccess.value && sessionId.value !== null),
    setSelectedFile,
    submitZip,
    useExistingDataset,
    filterActivities,
    reset,
  };
}
