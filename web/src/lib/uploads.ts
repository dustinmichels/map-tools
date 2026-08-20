import type { UploadedDataset, UploadSummary } from "./activity";

const readError = async (res: Response) => {
  const text = await res.text();
  return text || `Server returned status ${res.status}`;
};

export const uploadArchive = async (file: File) => {
  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch("/api/upload", {
    method: "POST",
    body: formData,
  });

  if (!res.ok) {
    throw new Error(await readError(res));
  }

  return (await res.json()) as UploadSummary;
};

export const listUploadedDatasets = async () => {
  const res = await fetch("/api/uploads");

  if (!res.ok) {
    throw new Error(await readError(res));
  }

  const data = (await res.json()) as { uploads: UploadedDataset[] };
  return data.uploads;
};

export const renameUploadedDataset = async (datasetId: string, name: string) => {
  const res = await fetch(`/api/uploads/${encodeURIComponent(datasetId)}`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ name }),
  });

  if (!res.ok) {
    throw new Error(await readError(res));
  }

  return (await res.json()) as UploadedDataset;
};
export const simplifyUploadedDataset = async (datasetId: string) => {
  const res = await fetch(`/api/uploads/${encodeURIComponent(datasetId)}/simplify`, {
    method: "POST",
  });

  if (!res.ok) {
    throw new Error(await readError(res));
  }

  return (await res.json()) as UploadedDataset;
};

export const openUploadedDataset = async (datasetId: string) => {
  const res = await fetch(`/api/uploads/${encodeURIComponent(datasetId)}/open`, {
    method: "POST",
  });

  if (!res.ok) {
    throw new Error(await readError(res));
  }
};

export const deleteUploadedDataset = async (datasetId: string) => {
  const res = await fetch(`/api/uploads/${encodeURIComponent(datasetId)}`, {
    method: "DELETE",
  });

  if (!res.ok) {
    throw new Error(await readError(res));
  }
};
