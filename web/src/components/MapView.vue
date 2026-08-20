<script setup lang="ts">
import { Loader2 } from "lucide-vue-next";
import { onMounted, onUnmounted, ref, shallowRef, watch } from "vue";
import {
  AttributionControl,
  Map as MapLibreMap,
  Marker,
  NavigationControl,
  Popup,
  setWorkerUrl,
} from "maplibre-gl";
import type { GeoJSONSource } from "maplibre-gl";
import type { BBox, LngLat, RouteLayer } from "../lib/activity";
import { EMPTY_FEATURE_COLLECTION } from "../lib/activity";
import "maplibre-gl/dist/maplibre-gl.css";
import maplibreWorkerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url";

setWorkerUrl(maplibreWorkerUrl);

const props = withDefaults(
  defineProps<{
    bbox: BBox;
    center: LngLat;
    routes?: RouteLayer[];
    showBBox?: boolean;
  }>(),
  {
    routes: () => [],
    showBBox: true,
  },
);

const emit = defineEmits<{
  (event: "update:bbox", bbox: BBox): void;
}>();

interface RouteHoverEvent {
  lngLat: { lng: number; lat: number };
  features?: Array<{ properties?: Record<string, unknown> }>;
}

interface RouteTooltipHandlers {
  mouseenter: (event: RouteHoverEvent) => void;
  mousemove: (event: RouteHoverEvent) => void;
  mouseleave: () => void;
  click: (event: RouteHoverEvent) => void;
}

const mapContainer = ref<HTMLElement | null>(null);
const map = shallowRef<MapLibreMap | null>(null);
const isDragging = ref(false);
const mapReady = ref(false);
const mapError = ref<string | null>(null);
const renderedRouteIds = new Set<string>();
const routeTooltipHandlers = new Map<string, RouteTooltipHandlers>();
const routeTooltip = shallowRef<Popup | null>(null);
let loadTimeout: ReturnType<typeof setTimeout> | null = null;
let markers: {
  sw: Marker;
  nw: Marker;
  ne: Marker;
  se: Marker;
} | null = null;

const getBBoxGeoJSON = (bboxValue: BBox) => {
  const [minLng, minLat, maxLng, maxLat] = bboxValue;
  return {
    type: "Feature" as const,
    properties: {},
    geometry: {
      type: "Polygon" as const,
      coordinates: [
        [
          [minLng, minLat],
          [minLng, maxLat],
          [maxLng, maxLat],
          [maxLng, minLat],
          [minLng, minLat],
        ],
      ],
    },
  };
};

const routeSourceId = (routeId: string) => `route-source-${routeId}`;
const routeLayerId = (routeId: string) => `route-layer-${routeId}`;

const createTooltipRow = (label: string, value: string) => {
  const row = document.createElement("div");
  row.className = "route-tooltip-row";

  const heading = document.createElement("span");
  heading.className = "route-tooltip-label";
  heading.textContent = label;

  const content = document.createElement("span");
  content.className = "route-tooltip-value";
  content.textContent = value;

  row.append(heading, content);
  return row;
};

const tooltipValue = (
  properties: Record<string, unknown> | undefined,
  primaryKey: string,
  fallbackKey?: string,
) => {
  if (!properties) {
    return null;
  }

  const value = properties[primaryKey] ?? (fallbackKey ? properties[fallbackKey] : undefined);
  if (value === undefined || value === null || value === "") {
    return null;
  }

  return String(value);
};
const routeIdValue = (properties: Record<string, unknown> | undefined) =>
  tooltipValue(properties, "route_id", "activity_id");

const copyTextToClipboard = async (value: string) => {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value);
    return;
  }

  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  textarea.style.pointerEvents = "none";
  document.body.appendChild(textarea);
  textarea.select();

  const copied = document.execCommand("copy");
  document.body.removeChild(textarea);
  if (!copied) {
    throw new Error("Clipboard copy failed.");
  }
};

const copyRouteIdToClipboard = async (properties: Record<string, unknown> | undefined) => {
  const routeId = routeIdValue(properties);
  if (!routeId) {
    return;
  }

  await copyTextToClipboard(routeId);
};

const buildRouteTooltip = (properties: Record<string, unknown> | undefined) => {
  const routeId = routeIdValue(properties);
  const routeDate = tooltipValue(properties, "route_date", "activity_date");
  if (!routeId && !routeDate) {
    return null;
  }

  const card = document.createElement("div");
  card.className = "route-tooltip-card";
  if (routeId) {
    card.appendChild(createTooltipRow("Route", routeId));
  }
  if (routeDate) {
    card.appendChild(createTooltipRow("Date", routeDate));
  }
  return card;
};

const hideRouteTooltip = () => {
  routeTooltip.value?.remove();
  if (map.value) {
    map.value.getCanvas().style.cursor = "";
  }
};

const showRouteTooltip = (event: RouteHoverEvent) => {
  if (!map.value || !routeTooltip.value) {
    return;
  }

  const feature = event.features?.[0];
  const content = buildRouteTooltip(feature?.properties);
  if (!content) {
    hideRouteTooltip();
    return;
  }

  map.value.getCanvas().style.cursor = "pointer";
  routeTooltip.value.setDOMContent(content).setLngLat([event.lngLat.lng, event.lngLat.lat]).addTo(
    map.value,
  );
};

const detachRouteTooltip = (routeId: string) => {
  if (!map.value) {
    routeTooltipHandlers.delete(routeId);
    return;
  }

  const handlers = routeTooltipHandlers.get(routeId);
  if (!handlers) {
    return;
  }

  const layerId = routeLayerId(routeId);
  map.value.off("mouseenter", layerId, handlers.mouseenter);
  map.value.off("mousemove", layerId, handlers.mousemove);
  map.value.off("mouseleave", layerId, handlers.mouseleave);
  map.value.off("click", layerId, handlers.click);
  routeTooltipHandlers.delete(routeId);
  hideRouteTooltip();
};

const attachRouteTooltip = (routeId: string) => {
  if (!map.value || routeTooltipHandlers.has(routeId)) {
    return;
  }

  const layerId = routeLayerId(routeId);
  const handlers: RouteTooltipHandlers = {
    mouseenter: (event) => {
      showRouteTooltip(event);
    },
    mousemove: (event) => {
      showRouteTooltip(event);
    },
    mouseleave: () => {
      hideRouteTooltip();
    },
    click: (event) => {
      showRouteTooltip(event);
      void copyRouteIdToClipboard(event.features?.[0]?.properties).catch((error: unknown) => {
        console.error("Failed to copy route ID:", error);
      });
    },
  };

  map.value.on("mouseenter", layerId, handlers.mouseenter);
  map.value.on("mousemove", layerId, handlers.mousemove);
  map.value.on("mouseleave", layerId, handlers.mouseleave);
  map.value.on("click", layerId, handlers.click);
  routeTooltipHandlers.set(routeId, handlers);
};

const syncRoutes = () => {
  if (!map.value || !mapReady.value) {
    return;
  }

  const nextRouteIds = new Set<string>();

  for (const route of props.routes) {
    nextRouteIds.add(route.id);

    const sourceId = routeSourceId(route.id);
    const layerId = routeLayerId(route.id);
    const data = route.data ?? EMPTY_FEATURE_COLLECTION;
    const source = map.value.getSource(sourceId) as GeoJSONSource | undefined;

    if (source) {
      source.setData(data);
    } else {
      map.value.addSource(sourceId, {
        type: "geojson",
        data,
      });
    }

    if (map.value.getLayer(layerId)) {
      map.value.setPaintProperty(layerId, "line-color", route.color);
    } else {
      map.value.addLayer({
        id: layerId,
        type: "line",
        source: sourceId,
        layout: {
          "line-join": "round",
          "line-cap": "round",
        },
        paint: {
          "line-color": route.color,
          "line-width": 2.5,
          "line-opacity": 0.95,
        },
      });
    }

    attachRouteTooltip(route.id);
  }

  for (const routeId of renderedRouteIds) {
    if (nextRouteIds.has(routeId)) {
      continue;
    }

    detachRouteTooltip(routeId);

    const layerId = routeLayerId(routeId);
    const sourceId = routeSourceId(routeId);

    if (map.value.getLayer(layerId)) {
      map.value.removeLayer(layerId);
    }
    if (map.value.getSource(sourceId)) {
      map.value.removeSource(sourceId);
    }
  }

  renderedRouteIds.clear();
  for (const routeId of nextRouteIds) {
    renderedRouteIds.add(routeId);
  }
  hideRouteTooltip();
};

const createMarkerEl = (label: string) => {
  const wrapper = document.createElement("div");
  wrapper.style.width = "18px";
  wrapper.style.height = "18px";

  const inner = document.createElement("div");
  inner.style.width = "100%";
  inner.style.height = "100%";
  inner.style.borderRadius = "50%";
  inner.style.backgroundColor = "#ff9900";
  inner.style.border = "3px solid #ffffff";
  inner.style.boxShadow = "0 2px 6px rgba(0,0,0,0.4)";
  inner.style.cursor = "move";
  inner.style.transition = "transform 0.1s, background-color 0.1s";
  inner.title = `Drag to resize ${label} corner`;

  inner.addEventListener("mouseenter", () => {
    inner.style.transform = "scale(1.25)";
    inner.style.backgroundColor = "#ffb347";
  });
  inner.addEventListener("mouseleave", () => {
    inner.style.transform = "scale(1)";
    inner.style.backgroundColor = "#ff9900";
  });

  wrapper.appendChild(inner);
  return wrapper;
};

const handleDrag = (corner: "sw" | "nw" | "ne" | "se") => {
  if (!markers || !map.value) {
    return;
  }

  const swLngLat = markers.sw.getLngLat();
  const nwLngLat = markers.nw.getLngLat();
  const neLngLat = markers.ne.getLngLat();
  const seLngLat = markers.se.getLngLat();

  let [minLng, minLat, maxLng, maxLat] = props.bbox;
  const epsilon = 0.0002;

  if (corner === "sw") {
    minLng = Math.min(swLngLat.lng, maxLng - epsilon);
    minLat = Math.min(swLngLat.lat, maxLat - epsilon);
    markers.sw.setLngLat([minLng, minLat]);
    markers.nw.setLngLat([minLng, maxLat]);
    markers.se.setLngLat([maxLng, minLat]);
  } else if (corner === "nw") {
    minLng = Math.min(nwLngLat.lng, maxLng - epsilon);
    maxLat = Math.max(nwLngLat.lat, minLat + epsilon);
    markers.nw.setLngLat([minLng, maxLat]);
    markers.sw.setLngLat([minLng, minLat]);
    markers.ne.setLngLat([maxLng, maxLat]);
  } else if (corner === "ne") {
    maxLng = Math.max(neLngLat.lng, minLng + epsilon);
    maxLat = Math.max(neLngLat.lat, minLat + epsilon);
    markers.ne.setLngLat([maxLng, maxLat]);
    markers.nw.setLngLat([minLng, maxLat]);
    markers.se.setLngLat([maxLng, minLat]);
  } else {
    maxLng = Math.max(seLngLat.lng, minLng + epsilon);
    minLat = Math.min(seLngLat.lat, maxLat - epsilon);
    markers.se.setLngLat([maxLng, minLat]);
    markers.sw.setLngLat([minLng, minLat]);
    markers.ne.setLngLat([maxLng, maxLat]);
  }

  const updatedBBox: BBox = [minLng, minLat, maxLng, maxLat];
  emit("update:bbox", updatedBBox);

  const source = map.value.getSource("bbox-source") as GeoJSONSource | undefined;
  if (source) {
    source.setData(getBBoxGeoJSON(updatedBBox));
  }
};

const setupMarkers = () => {
  if (!map.value) {
    return;
  }

  if (markers) {
    markers.sw.remove();
    markers.nw.remove();
    markers.ne.remove();
    markers.se.remove();
  }

  const [minLng, minLat, maxLng, maxLat] = props.bbox;

  const sw = new Marker({ element: createMarkerEl("Southwest"), draggable: true })
    .setLngLat([minLng, minLat])
    .addTo(map.value);
  const nw = new Marker({ element: createMarkerEl("Northwest"), draggable: true })
    .setLngLat([minLng, maxLat])
    .addTo(map.value);
  const ne = new Marker({ element: createMarkerEl("Northeast"), draggable: true })
    .setLngLat([maxLng, maxLat])
    .addTo(map.value);
  const se = new Marker({ element: createMarkerEl("Southeast"), draggable: true })
    .setLngLat([maxLng, minLat])
    .addTo(map.value);

  markers = { sw, nw, ne, se };

  sw.on("dragstart", () => {
    isDragging.value = true;
  });
  sw.on("drag", () => {
    handleDrag("sw");
  });
  sw.on("dragend", () => {
    isDragging.value = false;
  });

  nw.on("dragstart", () => {
    isDragging.value = true;
  });
  nw.on("drag", () => {
    handleDrag("nw");
  });
  nw.on("dragend", () => {
    isDragging.value = false;
  });

  ne.on("dragstart", () => {
    isDragging.value = true;
  });
  ne.on("drag", () => {
    handleDrag("ne");
  });
  ne.on("dragend", () => {
    isDragging.value = false;
  });

  se.on("dragstart", () => {
    isDragging.value = true;
  });
  se.on("drag", () => {
    handleDrag("se");
  });
  se.on("dragend", () => {
    isDragging.value = false;
  });
};

const fitToBBox = (duration = 1000) => {
  if (!map.value) {
    return;
  }

  const [minLng, minLat, maxLng, maxLat] = props.bbox;
  map.value.fitBounds([minLng, minLat, maxLng, maxLat], {
    padding: 60,
    maxZoom: 14,
    duration,
  });
};

watch(
  () => props.bbox,
  (newBBox) => {
    if (isDragging.value || !map.value) {
      return;
    }

    if (markers) {
      const [minLng, minLat, maxLng, maxLat] = newBBox;
      markers.sw.setLngLat([minLng, minLat]);
      markers.nw.setLngLat([minLng, maxLat]);
      markers.ne.setLngLat([maxLng, maxLat]);
      markers.se.setLngLat([maxLng, minLat]);
    }

    const source = map.value.getSource("bbox-source") as GeoJSONSource | undefined;
    if (source) {
      source.setData(getBBoxGeoJSON(newBBox));
    }
  },
  { deep: true },
);

watch(
  () => props.center,
  () => {
    if (map.value) {
      fitToBBox();
    }
  },
  { deep: true },
);

watch(
  () => props.routes,
  () => {
    syncRoutes();
  },
  { deep: true, immediate: true },
);

watch(
  () => props.showBBox,
  (showBBox) => {
    if (!map.value || !mapReady.value) {
      return;
    }

    for (const layerId of ["bbox-fill", "bbox-line"]) {
      if (map.value.getLayer(layerId)) {
        map.value.setLayoutProperty(layerId, "visibility", showBBox ? "visible" : "none");
      }
    }

    if (markers) {
      const display = showBBox ? "block" : "none";
      markers.sw.getElement().style.display = display;
      markers.nw.getElement().style.display = display;
      markers.ne.getElement().style.display = display;
      markers.se.getElement().style.display = display;
    }
  },
  { immediate: true },
);

const MAP_STYLE = "https://tiles.openfreemap.org/styles/dark";

const initMap = () => {
  if (!mapContainer.value) {
    return;
  }

  if (map.value) {
    map.value.remove();
    map.value = null;
  }
  routeTooltipHandlers.clear();
  routeTooltip.value?.remove();
  routeTooltip.value = null;

  try {
    map.value = new MapLibreMap({
      container: mapContainer.value,
      style: MAP_STYLE,
      center: props.center,
      zoom: 12,
      attributionControl: false,
    });
  } catch (error) {
    console.error("Failed to initialize MapLibre GL:", error);
    mapError.value =
      error instanceof Error ? error.message : "WebGL is not supported or failed to initialize.";
    return;
  }
  map.value.on("error", (event) => {
    console.error("MapLibre error:", event);
  });

  map.value.addControl(new NavigationControl(), "top-right");
  map.value.addControl(new AttributionControl({ compact: true }), "bottom-right");

  map.value.on("load", () => {
    if (loadTimeout) {
      clearTimeout(loadTimeout);
      loadTimeout = null;
    }
    if (!map.value) {
      return;
    }

    routeTooltip.value = new Popup({
      closeButton: false,
      closeOnClick: false,
      className: "route-tooltip-popup",
      offset: 12,
    });

    map.value.addSource("bbox-source", {
      type: "geojson",
      data: getBBoxGeoJSON(props.bbox),
    });

    map.value.addLayer({
      id: "bbox-fill",
      type: "fill",
      source: "bbox-source",
      paint: {
        "fill-color": "#ff9900",
        "fill-opacity": 0.35,
      },
    });

    map.value.addLayer({
      id: "bbox-line",
      type: "line",
      source: "bbox-source",
      paint: {
        "line-color": "#ff9900",
        "line-width": 3,
      },
    });

    setupMarkers();
    mapReady.value = true;
    syncRoutes();

    for (const layerId of ["bbox-fill", "bbox-line"]) {
      if (map.value.getLayer(layerId)) {
        map.value.setLayoutProperty(layerId, "visibility", props.showBBox ? "visible" : "none");
      }
    }

    if (markers) {
      const display = props.showBBox ? "block" : "none";
      markers.sw.getElement().style.display = display;
      markers.nw.getElement().style.display = display;
      markers.ne.getElement().style.display = display;
      markers.se.getElement().style.display = display;
    }

    map.value.resize();
    fitToBBox(0);
  });
};

onMounted(() => {
  if (!mapContainer.value) {
    return;
  }

  loadTimeout = setTimeout(() => {
    if (!mapReady.value && !mapError.value) {
      mapError.value =
        "Map loading timed out. The basemap tiles may be blocked by your network or firewall.";
    }
  }, 15000);

  initMap();
});

onUnmounted(() => {
  if (loadTimeout) {
    clearTimeout(loadTimeout);
  }

  hideRouteTooltip();
  routeTooltipHandlers.clear();
  routeTooltip.value?.remove();
  routeTooltip.value = null;

  if (map.value) {
    map.value.remove();
    map.value = null;
  }
});
</script>

<template>
  <div class="map-wrapper">
    <div ref="mapContainer" class="map-container"></div>
    <Transition name="map-fade">
      <div v-if="mapError" class="map-error-overlay">
        <span class="error-icon">⚠️</span>
        <span class="error-title">Map Loading Failed</span>
        <span class="error-msg">{{ mapError }}</span>
      </div>
      <div v-else-if="!mapReady" class="map-loading">
        <Loader2 class="spinner" :size="36" />
        <span>Loading map…</span>
      </div>
    </Transition>
    <div v-if="mapReady" class="map-actions">
      <button class="fit-btn" title="Refit map to current bounding box" @click="fitToBBox()">
        🔍 Recenter Box
      </button>
    </div>
  </div>
</template>

<style scoped>
.map-wrapper {
  position: relative;
  width: 100%;
  height: 550px;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #333;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}

.map-container {
  width: 100%;
  height: 100%;
}

.map-loading {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: #111;
  color: #888;
  font-size: 13px;
  z-index: 10;
}

.map-error-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: #1a1515;
  color: #ff9999;
  font-size: 13px;
  z-index: 10;
  padding: 24px;
  text-align: center;
}

.error-icon {
  font-size: 32px;
  margin-bottom: 4px;
}

.error-title {
  font-size: 16px;
  font-weight: 600;
  color: #ff5555;
}

.error-msg {
  color: #b39999;
  max-width: 400px;
  line-height: 1.5;
}

.spinner {
  color: #ff9900;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.map-fade-leave-active {
  transition: opacity 0.3s ease;
}

.map-fade-leave-to {
  opacity: 0;
}

.map-actions {
  position: absolute;
  bottom: 12px;
  left: 12px;
  z-index: 5;
}

.fit-btn {
  background: #1e1e1e;
  color: #fff;
  border: 1px solid #444;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  gap: 6px;
  transition:
    background 0.15s,
    border-color 0.15s;
}

.fit-btn:hover {
  background: #2a2a2a;
  border-color: #ff9900;
}

:deep(.route-tooltip-popup .maplibregl-popup-content) {
  padding: 0;
  border: 1px solid #333;
  border-radius: 10px;
  background: #111;
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.45);
}

:deep(.route-tooltip-popup .maplibregl-popup-tip) {
  border-top-color: #333;
}

:deep(.route-tooltip-card) {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 170px;
  padding: 10px 12px;
}

:deep(.route-tooltip-row) {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

:deep(.route-tooltip-label) {
  color: #9a9a9a;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

:deep(.route-tooltip-value) {
  color: #fff;
  font-size: 0.9rem;
  font-weight: 600;
}

:deep(.mapboxgl-marker) {
  z-index: 10;
}
</style>
