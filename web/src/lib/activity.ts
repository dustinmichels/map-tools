export type BBox = [number, number, number, number];
export type LngLat = [number, number];

export type Geometry =
  | { type: "LineString"; coordinates: [number, number][] }
  | { type: "MultiLineString"; coordinates: [number, number][][] }
  | { type: "Point"; coordinates: [number, number] }
  | { type: "MultiPoint"; coordinates: [number, number][] }
  | { type: string; coordinates?: unknown };

export interface GeoJSONFeature {
  geometry?: Geometry | null;
  properties?: Record<string, unknown>;
}

export interface GeoJSONFeatureCollection {
  type: "FeatureCollection";
  features: GeoJSONFeature[];
}

export interface RouteLayer {
  id: string;
  label: string;
  color: string;
  data: GeoJSONFeatureCollection | null;
}

export interface SelectedCity {
  name: string;
  bbox: BBox;
  lat: number;
  lon: number;
}

export interface UploadedDataset {
  datasetId: string;
  fileName: string;
  displayName: string;
  createdAt: string;
  sizeBytes: number;
  hasSimplified: boolean;
  total?: number | null;
  parsed?: number | null;
  rideCount?: number | null;
}

export interface UploadSummary {
  sessionId: string;
  total: number;
  parsed: number;
  rideCount: number;
  dataset: UploadedDataset;
}

export const DEFAULT_BOSTON_BBOX: BBox = [-71.1912, 42.2279, -70.9227, 42.3969];
export const DEFAULT_BOSTON_CENTER: LngLat = [-71.0589, 42.3601];
export const EMPTY_FEATURE_COLLECTION: GeoJSONFeatureCollection = {
  type: "FeatureCollection",
  features: [],
};

export const bboxCenter = (bbox: BBox): LngLat => [
  (bbox[0] + bbox[2]) / 2,
  (bbox[1] + bbox[3]) / 2,
];

const includeCoordinate = (
  coords: [number, number],
  bounds: { minLng: number; minLat: number; maxLng: number; maxLat: number },
) => {
  bounds.minLng = Math.min(bounds.minLng, coords[0]);
  bounds.minLat = Math.min(bounds.minLat, coords[1]);
  bounds.maxLng = Math.max(bounds.maxLng, coords[0]);
  bounds.maxLat = Math.max(bounds.maxLat, coords[1]);
};

export const getFeatureCollectionBounds = (geoJSON: GeoJSONFeatureCollection): BBox | null => {
  const bounds = {
    minLng: Number.POSITIVE_INFINITY,
    minLat: Number.POSITIVE_INFINITY,
    maxLng: Number.NEGATIVE_INFINITY,
    maxLat: Number.NEGATIVE_INFINITY,
  };

  for (const feature of geoJSON.features) {
    const geometry = feature.geometry;
    if (!geometry) {
      continue;
    }

    if (geometry.type === "LineString") {
      for (const coords of geometry.coordinates) {
        includeCoordinate(coords, bounds);
      }
      continue;
    }

    if (geometry.type === "MultiLineString") {
      for (const line of geometry.coordinates) {
        for (const coords of line) {
          includeCoordinate(coords, bounds);
        }
      }
      continue;
    }

    if (geometry.type === "Point") {
      includeCoordinate(geometry.coordinates, bounds);
      continue;
    }

    if (geometry.type === "MultiPoint") {
      for (const coords of geometry.coordinates) {
        includeCoordinate(coords, bounds);
      }
    }
  }

  if (!Number.isFinite(bounds.minLng)) {
    return null;
  }

  return [bounds.minLng, bounds.minLat, bounds.maxLng, bounds.maxLat];
};

export const formatFileSize = (bytes: number) => {
  if (bytes === 0) return "0 Bytes";
  const unit = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB"];
  const index = Math.floor(Math.log(bytes) / Math.log(unit));
  return `${parseFloat((bytes / Math.pow(unit, index)).toFixed(2))} ${sizes[index]}`;
};

export const formatCreatedAt = (value: string) => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
};
