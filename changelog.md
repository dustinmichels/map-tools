# Changelog

## 2026-08-26

- Upgraded Go backend to leverage Go 1.27 features:
  - Replaced `github.com/google/uuid` dependency with the new standard library `uuid` package (`uuid.New()`).
  - Implemented a generic `decode[T]` method on `requestDecoder` to parse incoming HTTP request bodies, utilizing Go 1.27's new support for generic methods on structs.
  - Benefited automatically from the new underlying v2 engine for standard `encoding/json` serialization/deserialization.

## 2026-08-23

- Added `@vueuse/core` to the frontend dependencies.
  - Replaced custom ZIP drag-and-drop handling in `BulkUploadCard.vue` and `DatasetUploadCard.vue` with `useDropZone`.
  - Persisted GIF export settings in `useRouteGifExport.ts` with `useLocalStorage` for `routeColor`, `flashColor`, `distanceUnit`, and `dateFormat`.
  - Added reset-to-default color buttons in `RouteGifExportCard.vue` for non-default route and flash colors.
  - Switched `MapView.vue` to `useResizeObserver` for MapLibre container resizing and debounced bbox updates with `useDebounceFn`.
  - Switched `SearchCity.vue` to `useDebounceFn` for debounced city search requests.
