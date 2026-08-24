# Frontend Architecture Recommendations

This document proposes the next frontend improvements for the Map Tools Vue application.

---

## 1. Share uploaded dataset state across screens

Keep upload-library state synchronized anywhere the app can rename, delete, simplify, or reopen datasets.

### Recommendation

- Move the reactive state in `web/src/composables/useUploadedDatasets.ts` to module scope so every consumer shares the same `uploads`, `isLoading`, `error`, and `busyDatasetId` refs.
- Keep `web/src/composables/useActivityDataset.ts` page-local unless the product needs one active dataset session to survive route changes or be shared between flows.
- Do not add Pinia unless state coordination grows beyond a few composables with obvious shared ownership.

### Example direction

```ts
import { ref } from "vue";
import type { UploadedDataset } from "../lib/activity";

const uploads = ref<UploadedDataset[]>([]);
const isLoading = ref(false);
const error = ref<string | null>(null);
const busyDatasetId = ref<string | null>(null);

export function useUploadedDatasets() {
  // actions operate on shared refs
  return {
    uploads,
    isLoading,
    error,
    busyDatasetId,
  };
}
```

---

## 2. Keep the component tree feature-oriented

Use a simple directory structure that matches feature boundaries instead of forcing a deep architecture taxonomy.

### Recommendation

- Keep feature-specific UI grouped under:
  - `web/src/components/flows/`
  - `web/src/components/uploads/`
  - `web/src/components/home/`
- Reserve the root of `web/src/components/` for broadly shared primitives only.
- If shared root-level components continue to grow, move them into `web/src/components/shared/`.
- Avoid a full directory rewrite unless it directly improves navigation or reduces coupling in active work.

### Suggested target

```text
web/src/components/
├── flows/
├── uploads/
├── home/
└── shared/
    ├── MapView.vue
    ├── SearchCity.vue
    └── BBoxCoords.vue
```

---

## 3. Keep Tailwind as the styling base

Use Tailwind for layout, spacing, color, and responsive behavior. Keep a thin semantic layer only where it improves reuse.

### Recommendation

- Continue importing Tailwind through `web/src/main.css`.
- Prefer utility classes for component-local layout and one-off styling decisions.
- Keep shared semantic classes such as buttons, cards, and upload zones only when they remove repetition across multiple screens.
- Treat `main.css` as a small design-token and shared-pattern layer, not a second styling system.
- If a class is used in one place and maps directly to a readable utility stack, inline it.

---

## 4. Keep using VueUse for browser-facing composables

VueUse is the right default for common reactive browser utilities.

### Recommendation

- Use `useDropZone` for file drag-and-drop interactions.
- Use `useLocalStorage` for user-facing preferences that should survive reloads.
- Use `useResizeObserver` for map and panel resizing concerns.
- Use `useDebounceFn` for search and viewport-driven updates.
- Prefer VueUse before writing custom wrappers around DOM events, persistence, or timers.

---

## 5. Do not add a full UI component library yet

Avoid pulling in a heavy UI dependency to replace small custom controls.

### Recommendation

- Keep `FlowStepper.vue` custom unless it becomes a maintenance or accessibility burden.
- Keep native `<input type="color">` and `<input type="range">` controls unless product requirements demand richer styling or interaction.
- Revisit a library such as Radix Vue or PrimeVue only if there is a concrete need:
  - keyboard-navigation bugs
  - accessibility gaps
  - repeated complex overlay components
  - a broader design-system push
- If that need appears, prefer the smallest dependency that solves the specific problem.

---

## 6. Break up `MapView.vue` by responsibility

`MapView.vue` should stop owning every map concern directly.

### Recommendation

- Extract MapLibre setup, teardown, and resize handling into a composable or helper module.
- Extract bounding-box marker creation, drag behavior, and bbox event wiring into a focused module.
- Extract route source/layer synchronization, tooltip rendering, and route interaction handlers into separate helpers.
- Keep `MapView.vue` responsible for orchestration and template wiring, not low-level map mechanics.

### Suggested split

1. `useMapLibre.ts`
   - create map
   - destroy map
   - resize handling
   - viewport event hookup
2. `mapBBox.ts`
   - bbox polygon source updates
   - draggable marker lifecycle
   - bbox drag math
3. `mapRoutes.ts`
   - source/layer registration
   - feature updates
   - hover/click handlers
   - tooltip management

This split is more important than the exact filenames. The goal is to reduce the size and responsibility surface of `MapView.vue`.

---

## Priority order

1. Extract responsibilities out of `MapView.vue`.
2. Share `useUploadedDatasets()` state across upload and flow screens.
3. Keep the current feature-based component structure; only move shared root components if it clarifies ownership.
4. Continue the Tailwind + light semantic CSS approach.
5. Continue using VueUse instead of custom browser utility code.
6. Postpone a full UI component library until a specific accessibility or design-system need exists.
