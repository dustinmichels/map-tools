# Frontend Architecture Reorganization and Library Proposal

This document outlines structural and library improvements for the Map Tools Vue frontend.

---

## 1. State Management Optimization (Singleton Pattern)

### Current Architecture
State within composables (e.g., `useUploadedDatasets.ts`, `useActivityDataset.ts`) is instantiated inside the returned factory functions. When multiple pages or components invoke these composables, they receive separate, isolated reactive state instances:
```ts
// web/src/composables/useUploadedDatasets.ts
export function useUploadedDatasets() {
  const uploads = ref<UploadedDataset[]>([]);
  const isLoading = ref(false);
  // ...
}
```
* **Risk:** Data changes on one screen (e.g., renaming, deleting, or simplifying a dataset on `UploadPage.vue`) do not propagate automatically to selection menus on other screens (e.g., `LightningMapFlow.vue`, `CompareFlow.vue`) until a manual reload is triggered.

### Proposed Improvement
Elevate the reactive references outside the exported composable function to convert the module into a shared singleton. This allows all importing components to share the same synchronized state without introducing Pinia boilerplate.

```ts
// web/src/composables/useUploadedDatasets.ts
import { ref } from "vue";
import type { UploadedDataset } from "../lib/activity";

// Global references shared by all components importing this composable
const uploads = ref<UploadedDataset[]>([]);
const isLoading = ref(false);
const error = ref<string | null>(null);
const busyDatasetId = ref<string | null>(null);

export function useUploadedDatasets() {
  const loadUploads = async () => {
    isLoading.value = true;
    error.value = null;
    try {
      uploads.value = await listUploadedDatasets();
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to load uploads";
    } finally {
      isLoading.value = false;
    }
  };

  return {
    uploads,
    isLoading,
    error,
    busyDatasetId,
    loadUploads,
  };
}
```

---

## 2. Directory Reorganization

### Current Directory Structure
Shared component dependencies (like `MapView.vue`, `BBoxCoords.vue`, and `SearchCity.vue`) sit at the flat root of `web/src/components/`, mixed with feature-specific subdirectories.

### Proposed Directory Structure
Migrate to a domain-driven structure, separating generic UI primitives, shared domain components, and page-specific feature modules:
```
web/src/
├── components/
│   ├── ui/               # Generic UI primitives
│   │   ├── FlowStepper.vue
│   │   └── BBoxCoords.vue
│   ├── shared/           # Shared map-specific domain components
│   │   ├── MapView.vue
│   │   └── SearchCity.vue
│   └── features/         # Feature modules containing pages and specific sub-cards
│       ├── uploads/
│       │   ├── UploadPage.vue
│       │   ├── UploadedDatasetList.vue
│       │   └── BulkUploadCard.vue
│       └── flows/
│           ├── LightningMapFlow.vue
│           ├── CompareFlow.vue
│           ├── RouteGifExportCard.vue
│           ├── AreaSelectionCard.vue
│           └── DatasetUploadCard.vue
```

---

## 3. Styling Refactoring (Tailwind CSS Integration)

### Current Architecture
Styling relies on `web/src/main.css` containing over 700 lines of custom CSS (`.btn`, `.btn-primary`, `.card`, `.upload-zone`, etc.), layouts, and media queries.
* **Risk:** High visual maintenance overhead. Any tweak to layout or color variables requires writing custom CSS classes, leading to class duplication and styling conflicts.

### Proposed Improvement
Install Tailwind CSS to replace custom component styles with utility classes. This eliminates `main.css` class bloat, ensures consistent spacing, and enables easy responsive classes (e.g., `md:grid-cols-2`) and focus/hover transitions.

```html
<!-- Before (Requires custom css for .card, .hero-card, .btn) -->
<div class="card hero-card">
  <button class="btn btn-primary" :disabled="busy">Action</button>
</div>

<!-- After (Tailwind CSS utility styling) -->
<div class="p-6 bg-slate-900 border border-slate-800 rounded-lg shadow-sm flex flex-col">
  <button 
    class="px-4 py-2 bg-orange-500 hover:bg-orange-400 disabled:opacity-50 text-white font-medium rounded-md transition-colors"
    :disabled="busy"
  >
    Action
  </button>
</div>
```

---

## 4. Recommended Library Additions

### 1. VueUse (`@vueuse/core`)
A library containing Composition API utilities that reduces boilerplate:

* **File Drag-and-Drop:** `BulkUploadCard.vue` and `DatasetUploadCard.vue` write custom drag-and-drop listeners. Replace them with the `useDropZone` utility:
  ```ts
  import { useDropZone } from "@vueuse/core";

  const dropZoneRef = ref<HTMLDivElement>();
  const { isOverDropZone } = useDropZone(dropZoneRef, (files) => {
    if (files) emit("select-files", Array.from(files));
  });
  ```
* **Setting Persistence:** User preferences (e.g., `routeColor`, `flashColor`, `distanceUnit`, and `dateFormat` in `useRouteGifExport.ts`) reset on page reload. Bind these settings to `useLocalStorage` to persist them automatically across sessions:
  ```ts
  import { useLocalStorage } from "@vueuse/core";

  const routeColor = useLocalStorage("map-route-color", "#ff8c00");
  const flashColor = useLocalStorage("map-flash-color", "#ffffff");
  ```
* **Map Sizing & Debouncing:** Use `useResizeObserver` for MapLibre container resizing and `useDebounceFn` to rate-limit search queries or bounding box changes.

### 2. Radix Vue / PrimeVue
Replacing custom elements with accessible UI library primitives improves keyboard navigation and ARIA support.
* **Stepper:** Replaces `FlowStepper.vue` with a pre-built component that handles mobile responsive steps and focus management.
* **Color Picker & Sliders:** Replaces raw browser input elements (`<input type="color">` and `<input type="range">`) with styled, customizable overlay components matching the application theme.

---

## 5. Modularizing `MapView.vue`

### Current Architecture
`MapView.vue` has grown to ~850 lines. It mixes three separate concerns:
1. MapLibre initialization, sizing, and lifecycle cleanup.
2. Layer definition, source updates, and styling.
3. Interaction logic (bounding box coordinate marker drags, route line hovering, popup overlays).

### Proposed Improvement
Extract the MapLibre engine logic into a composable named `useMapLibre.ts`. This isolates MapLibre configuration, makes map interactions unit-testable, and keeps the template of `MapView.vue` clean.

```ts
// web/src/composables/useMapLibre.ts
import { shallowRef } from "vue";
import { Map as MapLibreMap } from "maplibre-gl";

export function useMapLibre() {
  const map = shallowRef<MapLibreMap | null>(null);

  const initMap = (container: HTMLElement, options: any) => {
    map.value = new MapLibreMap({ container, ...options });
    return map.value;
  };

  const updateSource = (sourceId: string, data: any) => {
    if (!map.value) return;
    const source = map.value.getSource(sourceId) as any;
    if (source) {
      source.setData(data);
    }
  };

  return {
    map,
    initMap,
    updateSource,
  };
}
```
