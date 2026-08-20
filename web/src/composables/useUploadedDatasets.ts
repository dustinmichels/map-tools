import { ref } from "vue";
import type { UploadedDataset } from "../lib/activity";
import {
  deleteUploadedDataset,
  listUploadedDatasets,
  openUploadedDataset,
  renameUploadedDataset,
  simplifyUploadedDataset,
} from "../lib/uploads";

export function useUploadedDatasets() {
  const uploads = ref<UploadedDataset[]>([]);
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  const busyDatasetId = ref<string | null>(null);

  const loadUploads = async () => {
    isLoading.value = true;
    error.value = null;

    try {
      uploads.value = await listUploadedDatasets();
    } catch (err) {
      console.error(err);
      error.value = err instanceof Error ? err.message : "Failed to load uploaded datasets.";
    } finally {
      isLoading.value = false;
    }
  };

  const renameUpload = async (datasetId: string, name: string) => {
    busyDatasetId.value = datasetId;
    error.value = null;

    try {
      const updatedUpload = await renameUploadedDataset(datasetId, name);
      uploads.value = uploads.value.map((upload) =>
        upload.datasetId === datasetId ? updatedUpload : upload,
      );
    } catch (err) {
      console.error(err);
      error.value = err instanceof Error ? err.message : "Failed to rename uploaded dataset.";
    } finally {
      busyDatasetId.value = null;
    }
  };
  const simplifyUpload = async (datasetId: string) => {
    busyDatasetId.value = datasetId;
    error.value = null;

    try {
      const updatedUpload = await simplifyUploadedDataset(datasetId);
      uploads.value = uploads.value.map((upload) =>
        upload.datasetId === datasetId ? updatedUpload : upload,
      );
      return updatedUpload;
    } catch (err) {
      console.error(err);
      error.value = err instanceof Error ? err.message : "Failed to simplify uploaded dataset.";
      return null;
    } finally {
      busyDatasetId.value = null;
    }
  };

  const openUpload = async (datasetId: string) => {
    busyDatasetId.value = datasetId;
    error.value = null;

    try {
      await openUploadedDataset(datasetId);
    } catch (err) {
      console.error(err);
      error.value = err instanceof Error ? err.message : "Failed to open uploaded dataset.";
    } finally {
      busyDatasetId.value = null;
    }
  };

  const deleteUpload = async (datasetId: string) => {
    busyDatasetId.value = datasetId;
    error.value = null;

    try {
      await deleteUploadedDataset(datasetId);
      uploads.value = uploads.value.filter((upload) => upload.datasetId !== datasetId);
    } catch (err) {
      console.error(err);
      error.value = err instanceof Error ? err.message : "Failed to delete uploaded dataset.";
    } finally {
      busyDatasetId.value = null;
    }
  };

  return {
    uploads,
    isLoading,
    error,
    busyDatasetId,
    loadUploads,
    renameUpload,
    simplifyUpload,
    openUpload,
    deleteUpload,
  };
}
