package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dustinmichels/map-tools/internal/strava"
	"github.com/parquet-go/parquet-go"
)

func TestHealth(t *testing.T) {
	router := apiRouter()
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
	if len(resp) != 1 {
		t.Errorf("expected health response to contain only status, got %v", resp)
	}
}

func TestListRenameOpenAndDeleteUploads(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	parquetPath := filepath.Join(dataDir, "activities-existing.parquet")
	if err := os.WriteFile(parquetPath, []byte("parquet"), 0644); err != nil {
		t.Fatal(err)
	}

	simplifiedParquetPath := strava.SimplifiedParquetPath(parquetPath)
	if err := os.WriteFile(simplifiedParquetPath, []byte("simplified"), 0644); err != nil {
		t.Fatal(err)
	}

	total := 12
	parsed := 11
	rideCount := 9
	createdAt := time.Date(2026, time.January, 15, 12, 30, 0, 0, time.UTC)
	if err := writeUploadMetadata(uploadMetadataPath(parquetPath), uploadMetadata{
		DatasetID:   "dataset-123",
		DisplayName: "Existing Upload",
		CreatedAt:   createdAt,
		Total:       &total,
		Parsed:      &parsed,
		RideCount:   &rideCount,
	}); err != nil {
		t.Fatal(err)
	}

	router := apiRouter()

	reqList := httptest.NewRequest("GET", "/uploads", nil)
	rrList := httptest.NewRecorder()
	router.ServeHTTP(rrList, reqList)

	if rrList.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d", rrList.Code)
	}

	var listResp listUploadsResponse
	if err := json.NewDecoder(rrList.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}

	if len(listResp.Uploads) != 1 {
		t.Fatalf("expected one upload, got %d", len(listResp.Uploads))
	}

	upload := listResp.Uploads[0]
	if upload.DatasetID != "dataset-123" {
		t.Fatalf("expected datasetId dataset-123, got %q", upload.DatasetID)
	}
	if upload.DisplayName != "Existing Upload" {
		t.Fatalf("expected display name Existing Upload, got %q", upload.DisplayName)
	}
	if upload.Total == nil || *upload.Total != total {
		t.Fatalf("expected total %d, got %#v", total, upload.Total)
	}

	if !upload.HasSimplified {
		t.Fatal("expected listed upload to report simplified companion")
	}

	renameBody := bytes.NewBufferString(`{"name":"Boston Routes"}`)
	reqRename := httptest.NewRequest("PATCH", "/uploads/dataset-123", renameBody)
	reqRename.Header.Set("Content-Type", "application/json")
	rrRename := httptest.NewRecorder()
	router.ServeHTTP(rrRename, reqRename)

	if rrRename.Code != http.StatusOK {
		t.Fatalf("expected rename status 200, got %d", rrRename.Code)
	}

	var renamed UploadedDataset
	if err := json.NewDecoder(rrRename.Body).Decode(&renamed); err != nil {
		t.Fatalf("failed to decode rename response: %v", err)
	}

	renamedParquetPath := filepath.Join(dataDir, "Boston Routes.parquet")
	if _, err := os.Stat(renamedParquetPath); err != nil {
		t.Fatalf("expected renamed parquet file to exist: %v", err)
	}
	if renamed.FileName != "Boston Routes.parquet" {
		t.Fatalf("expected renamed file name to be Boston Routes.parquet, got %q", renamed.FileName)
	}
	if renamed.DisplayName != "Boston Routes" {
		t.Fatalf("expected renamed display name to be Boston Routes, got %q", renamed.DisplayName)
	}
	if !renamed.HasSimplified {
		t.Fatal("expected renamed upload to report simplified companion")
	}

	renamedSimplifiedPath := strava.SimplifiedParquetPath(renamedParquetPath)
	if _, err := os.Stat(renamedSimplifiedPath); err != nil {
		t.Fatalf("expected renamed simplified parquet file to exist: %v", err)
	}
	openedPath := ""
	restoreOpenUploadPath := openUploadPath
	openUploadPath = func(path string) error {
		openedPath = path
		return nil
	}
	defer func() {
		openUploadPath = restoreOpenUploadPath
	}()

	reqOpen := httptest.NewRequest("POST", "/uploads/dataset-123/open", nil)
	rrOpen := httptest.NewRecorder()
	router.ServeHTTP(rrOpen, reqOpen)

	if rrOpen.Code != http.StatusNoContent {
		t.Fatalf("expected open status 204, got %d", rrOpen.Code)
	}
	if openedPath != renamedParquetPath {
		t.Fatalf("expected open path %q, got %q", renamedParquetPath, openedPath)
	}

	reqDelete := httptest.NewRequest("DELETE", "/uploads/dataset-123", nil)
	rrDelete := httptest.NewRecorder()
	router.ServeHTTP(rrDelete, reqDelete)

	if rrDelete.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d", rrDelete.Code)
	}

	if _, err := os.Stat(renamedParquetPath); !os.IsNotExist(err) {
		t.Fatalf("expected parquet file to be deleted, stat err = %v", err)
	}
	if _, err := os.Stat(uploadMetadataPath(renamedParquetPath)); !os.IsNotExist(err) {
		t.Fatalf("expected metadata file to be deleted, stat err = %v", err)
	}
	if _, err := os.Stat(renamedSimplifiedPath); !os.IsNotExist(err) {
		t.Fatalf("expected simplified parquet file to be deleted, stat err = %v", err)
	}
}

func TestSimplifyUploadCreatesCompanionParquet(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	parquetPath := filepath.Join(dataDir, "legacy.parquet")
	writeUploadActivityParquet(t, parquetPath)

	if err := writeUploadMetadata(uploadMetadataPath(parquetPath), uploadMetadata{
		DatasetID:   "dataset-legacy",
		DisplayName: "Legacy Upload",
		CreatedAt:   time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	router := apiRouter()
	req := httptest.NewRequest("POST", "/uploads/dataset-legacy/simplify", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected simplify status 200, got %d", rr.Code)
	}

	var simplified UploadedDataset
	if err := json.NewDecoder(rr.Body).Decode(&simplified); err != nil {
		t.Fatalf("failed to decode simplify response: %v", err)
	}
	if !simplified.HasSimplified {
		t.Fatal("expected simplify response to report simplified companion")
	}

	simplifiedPath := strava.SimplifiedParquetPath(parquetPath)
	if _, err := os.Stat(simplifiedPath); err != nil {
		t.Fatalf("expected simplified parquet file to exist: %v", err)
	}

	originalRow := readUploadActivityRow(t, parquetPath)
	simplifiedRow := readUploadActivityRow(t, simplifiedPath)
	if len(simplifiedRow.Geometry) >= len(originalRow.Geometry) {
		t.Fatalf("expected simplified geometry to shrink, got original=%d simplified=%d", len(originalRow.Geometry), len(simplifiedRow.Geometry))
	}
}

func TestNextBulkUploadBaseNameAppendsDailySuffix(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	for _, fileName := range []string{
		"bulk_upload_2026-08-19.parquet",
		"bulk_upload_2026-08-19_2.parquet",
	} {
		if err := os.WriteFile(filepath.Join(dataDir, fileName), []byte("parquet"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	baseName, err := nextBulkUploadBaseName(time.Date(2026, time.August, 19, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if baseName != "bulk_upload_2026-08-19_3" {
		t.Fatalf("expected suffixed base name, got %q", baseName)
	}
}

func TestFilterPrefersSimplifiedCompanion(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	parquetPath := filepath.Join(dataDir, "legacy.parquet")
	writeUploadActivityParquetWithCoords(t, parquetPath, [][2]float64{
		{-71.1000, 42.3600},
		{-71.0999, 42.3601},
		{-71.0998, 42.3602},
		{-71.0997, 42.3603},
		{-71.0996, 42.3604},
		{-71.0995, 42.3605},
	})

	simplifiedPath := strava.SimplifiedParquetPath(parquetPath)
	writeUploadActivityParquetWithCoords(t, simplifiedPath, [][2]float64{
		{-71.1000, 42.3600},
		{-71.0995, 42.3605},
	})

	router := apiRouter()
	body, err := json.Marshal(FilterRequest{SessionId: "legacy"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/filter", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		respBody, _ := io.ReadAll(rr.Body)
		t.Fatalf("filter failed with status %d: %s", rr.Code, respBody)
	}

	var geojson map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&geojson); err != nil {
		t.Fatalf("decode filter geojson: %v", err)
	}

	features, ok := geojson["features"].([]any)
	if !ok || len(features) != 1 {
		t.Fatalf("expected one feature, got %v", geojson["features"])
	}

	feature, ok := features[0].(map[string]any)
	if !ok {
		t.Fatalf("expected feature object, got %T", features[0])
	}

	properties, ok := feature["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties object, got %T", feature["properties"])
	}
	if properties["route_id"] != float64(99) {
		t.Fatalf("expected route_id 99, got %v", properties["route_id"])
	}
	if properties["route_date"] != "2026-08-19" {
		t.Fatalf("expected route_date 2026-08-19, got %v", properties["route_date"])
	}

	geometry, ok := feature["geometry"].(map[string]any)
	if !ok {
		t.Fatalf("expected geometry object, got %T", feature["geometry"])
	}

	coordinates, ok := geometry["coordinates"].([]any)
	if !ok {
		t.Fatalf("expected line string coordinates, got %T", geometry["coordinates"])
	}
	if len(coordinates) != 2 {
		t.Fatalf("expected filter to use simplified parquet coordinates, got %d points", len(coordinates))
	}
}
func TestFilterUsesOriginalGeometryWhenRequested(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	parquetPath := filepath.Join(dataDir, "legacy.parquet")
	writeUploadActivityParquetWithCoords(t, parquetPath, [][2]float64{
		{-71.1000, 42.3600},
		{-71.0999, 42.3601},
		{-71.0998, 42.3602},
		{-71.0997, 42.3603},
		{-71.0996, 42.3604},
		{-71.0995, 42.3605},
	})

	simplifiedPath := strava.SimplifiedParquetPath(parquetPath)
	writeUploadActivityParquetWithCoords(t, simplifiedPath, [][2]float64{
		{-71.1000, 42.3600},
		{-71.0995, 42.3605},
	})

	router := apiRouter()
	body, err := json.Marshal(FilterRequest{SessionId: "legacy", GeometryMode: "original"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/filter", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		respBody, _ := io.ReadAll(rr.Body)
		t.Fatalf("filter failed with status %d: %s", rr.Code, respBody)
	}

	var geojson map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&geojson); err != nil {
		t.Fatalf("decode filter geojson: %v", err)
	}

	features, ok := geojson["features"].([]any)
	if !ok || len(features) != 1 {
		t.Fatalf("expected one feature, got %v", geojson["features"])
	}

	feature, ok := features[0].(map[string]any)
	if !ok {
		t.Fatalf("expected feature object, got %T", features[0])
	}

	properties, ok := feature["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties object, got %T", feature["properties"])
	}
	if properties["route_id"] != float64(99) {
		t.Fatalf("expected route_id 99, got %v", properties["route_id"])
	}
	if properties["route_date"] != "2026-08-19" {
		t.Fatalf("expected route_date 2026-08-19, got %v", properties["route_date"])
	}

	geometry, ok := feature["geometry"].(map[string]any)
	if !ok {
		t.Fatalf("expected geometry object, got %T", feature["geometry"])
	}

	coordinates, ok := geometry["coordinates"].([]any)
	if !ok {
		t.Fatalf("expected line string coordinates, got %T", geometry["coordinates"])
	}
	if len(coordinates) != 6 {
		t.Fatalf("expected filter to use original parquet coordinates, got %d points", len(coordinates))
	}
}

func TestFilterClipsMultilineGeometryToBBox(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	parquetPath := filepath.Join(dataDir, "legacy.parquet")
	writeUploadActivityParquetWithParts(t, parquetPath, [][][2]float64{
		{
			{-71.1000, 42.3600},
			{-71.0900, 42.3700},
		},
		{
			{-71.3000, 42.5000},
			{-71.2500, 42.5500},
		},
	})

	router := apiRouter()
	bbox := [4]float64{-71.1500, 42.3500, -71.0500, 42.4500}
	body, err := json.Marshal(FilterRequest{SessionId: "legacy", BBox: &bbox, GeometryMode: "original"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/filter", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		respBody, _ := io.ReadAll(rr.Body)
		t.Fatalf("filter failed with status %d: %s", rr.Code, respBody)
	}

	var geojson map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&geojson); err != nil {
		t.Fatalf("decode filter geojson: %v", err)
	}

	features, ok := geojson["features"].([]any)
	if !ok || len(features) != 1 {
		t.Fatalf("expected one feature, got %v", geojson["features"])
	}

	feature, ok := features[0].(map[string]any)
	if !ok {
		t.Fatalf("expected feature object, got %T", features[0])
	}

	lineParts := decodeGeoJSONLineParts(t, feature["geometry"])
	if len(lineParts) != 1 {
		t.Fatalf("expected one clipped line part, got %d", len(lineParts))
	}
	if len(lineParts[0]) != 2 {
		t.Fatalf("expected clipped line to keep 2 coordinates, got %d", len(lineParts[0]))
	}

	for index, coord := range lineParts[0] {
		if coord[0] < bbox[0] || coord[0] > bbox[2] || coord[1] < bbox[1] || coord[1] > bbox[3] {
			t.Fatalf("coordinate %d outside bbox: %v", index, coord)
		}
	}
}

func TestFilterCreatesSimplifiedCompanionOnDemand(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	parquetPath := filepath.Join(dataDir, "legacy.parquet")
	writeUploadActivityParquet(t, parquetPath)

	router := apiRouter()
	body, err := json.Marshal(FilterRequest{SessionId: "legacy"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/filter", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		respBody, _ := io.ReadAll(rr.Body)
		t.Fatalf("filter failed with status %d: %s", rr.Code, respBody)
	}

	simplifiedPath := strava.SimplifiedParquetPath(parquetPath)
	if _, err := os.Stat(simplifiedPath); err != nil {
		t.Fatalf("expected filter to create simplified parquet file: %v", err)
	}

	originalRow := readUploadActivityRow(t, parquetPath)
	simplifiedRow := readUploadActivityRow(t, simplifiedPath)
	if len(simplifiedRow.Geometry) >= len(originalRow.Geometry) {
		t.Fatalf("expected on-demand simplified geometry to shrink, got original=%d simplified=%d", len(originalRow.Geometry), len(simplifiedRow.Geometry))
	}
}

func TestUploadNamesBulkUploadsByDay(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)

	createdAt := time.Date(2026, time.August, 19, 8, 0, 0, 0, time.UTC)
	restoreNowUTC := nowUTC
	nowUTC = func() time.Time {
		return createdAt
	}
	defer func() {
		nowUTC = restoreNowUTC
	}()

	restoreIngestZip := ingestZip
	ingestZip = func(zipPath, outPath string) (strava.IngestResult, error) {
		if _, err := os.Stat(zipPath); err != nil {
			return strava.IngestResult{}, err
		}
		if err := os.WriteFile(outPath, []byte("parquet"), 0644); err != nil {
			return strava.IngestResult{}, err
		}
		if err := os.WriteFile(strava.SimplifiedParquetPath(outPath), []byte("simplified"), 0644); err != nil {
			return strava.IngestResult{}, err
		}
		return strava.IngestResult{Total: 2, Parsed: 2, RideCount: 1}, nil
	}
	defer func() {
		ingestZip = restoreIngestZip
	}()

	router := apiRouter()

	upload := func(fileName string) UploadResponse {
		t.Helper()

		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, err := mw.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte("zip-bytes")); err != nil {
			t.Fatal(err)
		}
		if err := mw.Close(); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			body, _ := io.ReadAll(rr.Body)
			t.Fatalf("upload failed with status %d: %s", rr.Code, body)
		}

		var resp UploadResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		return resp
	}

	first := upload("strava-export.zip")
	if first.Dataset.DisplayName != "bulk_upload_2026-08-19" {
		t.Fatalf("expected first display name, got %q", first.Dataset.DisplayName)
	}
	if first.Dataset.FileName != "bulk_upload_2026-08-19.parquet" {
		t.Fatalf("expected first file name, got %q", first.Dataset.FileName)
	}
	if _, err := os.Stat(filepath.Join(dataDir, first.Dataset.FileName)); err != nil {
		t.Fatalf("expected first parquet file to exist: %v", err)
	}
	if !first.Dataset.HasSimplified {
		t.Fatal("expected first upload to report simplified companion")
	}

	second := upload("strava-export-again.zip")
	if second.Dataset.DisplayName != "bulk_upload_2026-08-19_2" {
		t.Fatalf("expected suffixed display name, got %q", second.Dataset.DisplayName)
	}
	if second.Dataset.FileName != "bulk_upload_2026-08-19_2.parquet" {
		t.Fatalf("expected suffixed file name, got %q", second.Dataset.FileName)
	}
	if _, err := os.Stat(filepath.Join(dataDir, second.Dataset.FileName)); err != nil {
		t.Fatalf("expected second parquet file to exist: %v", err)
	}
	if !second.Dataset.HasSimplified {
		t.Fatal("expected second upload to report simplified companion")
	}

}

func TestUploadAndFilter(t *testing.T) {
	zipPath := "../../data/strava_export.zip"
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		t.Skip("skipping integration test, test zip file not found")
	}

	dataDir := filepath.Join(t.TempDir(), "uploads")
	t.Setenv("MAPTOOLS_DATA_DIR", dataDir)

	createdAt := time.Date(2026, time.August, 19, 8, 0, 0, 0, time.UTC)
	restoreNowUTC := nowUTC
	nowUTC = func() time.Time {
		return createdAt
	}
	defer func() {
		nowUTC = restoreNowUTC
	}()

	router := apiRouter()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filepath.Base(zipPath))
	if err != nil {
		t.Fatal(err)
	}

	zipFile, err := os.Open(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zipFile.Close()

	if _, err := io.Copy(fw, zipFile); err != nil {
		t.Fatal(err)
	}
	mw.Close()

	reqUpload := httptest.NewRequest("POST", "/upload", &buf)
	reqUpload.Header.Set("Content-Type", mw.FormDataContentType())
	rrUpload := httptest.NewRecorder()
	router.ServeHTTP(rrUpload, reqUpload)

	if rrUpload.Code != http.StatusOK {
		body, _ := io.ReadAll(rrUpload.Body)
		t.Fatalf("upload failed with status %d: %s", rrUpload.Code, body)
	}

	var uploadResp map[string]any
	if err := json.NewDecoder(rrUpload.Body).Decode(&uploadResp); err != nil {
		t.Fatal(err)
	}

	sessionId, _ := uploadResp["sessionId"].(string)
	if sessionId == "" {
		t.Fatal("expected sessionId, got empty string")
	}

	dataset, ok := uploadResp["dataset"].(map[string]any)
	if !ok {
		t.Fatalf("expected dataset payload, got %v", uploadResp["dataset"])
	}
	if dataset["datasetId"] != sessionId {
		t.Fatalf("expected datasetId %q, got %v", sessionId, dataset["datasetId"])
	}
	if dataset["displayName"] != "bulk_upload_2026-08-19" {
		t.Fatalf("expected upload display name to be bulk_upload_2026-08-19, got %v", dataset["displayName"])
	}
	if dataset["fileName"] != "bulk_upload_2026-08-19.parquet" {
		t.Fatalf("expected upload file name to be bulk_upload_2026-08-19.parquet, got %v", dataset["fileName"])
	}
	if dataset["hasSimplified"] != true {
		t.Fatalf("expected upload to report simplified companion, got %v", dataset["hasSimplified"])
	}

	if uploadResp["total"] == nil || uploadResp["parsed"] == nil || uploadResp["rideCount"] == nil || uploadResp["summary"] == nil {
		t.Errorf("missing statistics in upload response: %v", uploadResp)
	} else {
		total := uploadResp["total"].(float64)
		parsed := uploadResp["parsed"].(float64)
		rideCount := uploadResp["rideCount"].(float64)
		summary := uploadResp["summary"].(string)

		if parsed != 1146 {
			t.Errorf("expected 1146 parsed activities, got %.0f", parsed)
		}
		if total <= 0 {
			t.Errorf("expected total > 0, got %.0f", total)
		}
		if rideCount <= 0 {
			t.Errorf("expected rideCount > 0, got %.0f", rideCount)
		}
		expectedSummary := fmt.Sprintf("Succesfully parsed %.0f / %.0f  activities. %.0f are type = Ride.", parsed, total, rideCount)
		if summary != expectedSummary {
			t.Errorf("expected summary %q, got %q", expectedSummary, summary)
		}
	}

	parquetPath := filepath.Join(dataDir, "bulk_upload_2026-08-19.parquet")
	simplifiedParquetPath := strava.SimplifiedParquetPath(parquetPath)
	defer os.Remove(parquetPath)
	defer os.Remove(simplifiedParquetPath)
	defer os.Remove(uploadMetadataPath(parquetPath))
	if _, err := os.Stat(parquetPath); os.IsNotExist(err) {
		t.Fatalf("expected parquet file to exist at %s, but it does not", parquetPath)
	}
	if _, err := os.Stat(simplifiedParquetPath); os.IsNotExist(err) {
		t.Fatalf("expected simplified parquet file to exist at %s, but it does not", simplifiedParquetPath)
	}

	bbox := [4]float64{-71.1912, 42.2279, -70.9227, 42.3969}
	filterReqBody := FilterRequest{
		SessionId: sessionId,
		BBox:      &bbox,
	}
	bodyBuf, err := json.Marshal(filterReqBody)
	if err != nil {
		t.Fatal(err)
	}

	decodeFilterResponse := func(rr *httptest.ResponseRecorder) []interface{} {
		if rr.Code != http.StatusOK {
			body, _ := io.ReadAll(rr.Body)
			t.Fatalf("filter failed with status %d: %s", rr.Code, body)
		}

		var geojson map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&geojson); err != nil {
			t.Fatalf("failed to decode geojson output: %v", err)
		}

		if geojson["type"] != "FeatureCollection" {
			t.Errorf("expected type FeatureCollection, got %v", geojson["type"])
		}

		features, ok := geojson["features"].([]interface{})
		if !ok {
			t.Fatal("geojson features is not a slice")
		}

		for index, feature := range features {
			featureMap, ok := feature.(map[string]interface{})
			if !ok {
				t.Fatalf("feature %d is not an object", index)
			}

			properties, ok := featureMap["properties"].(map[string]interface{})
			if !ok {
				t.Fatalf("feature %d properties missing", index)
			}

			if properties["activity_type"] != "Ride" {
				t.Fatalf("feature %d activity_type = %v, want Ride", index, properties["activity_type"])
			}
			if properties["route_id"] == nil {
				t.Fatalf("feature %d route_id missing", index)
			}
			if properties["route_date"] == nil {
				t.Fatalf("feature %d route_date missing", index)
			}
		}

		return features
	}

	reqFilter := httptest.NewRequest("POST", "/filter", bytes.NewReader(bodyBuf))
	reqFilter.Header.Set("Content-Type", "application/json")
	rrFilter := httptest.NewRecorder()
	router.ServeHTTP(rrFilter, reqFilter)

	features := decodeFilterResponse(rrFilter)
	if len(features) == 0 {
		t.Error("expected at least some rides to be returned, got 0")
	} else {
		t.Logf("found %d rides matching search criteria", len(features))
	}

	filterAllReqBody := FilterRequest{SessionId: sessionId}
	allBodyBuf, err := json.Marshal(filterAllReqBody)
	if err != nil {
		t.Fatal(err)
	}

	reqFilterAll := httptest.NewRequest("POST", "/filter", bytes.NewReader(allBodyBuf))
	reqFilterAll.Header.Set("Content-Type", "application/json")
	rrFilterAll := httptest.NewRecorder()
	router.ServeHTTP(rrFilterAll, reqFilterAll)

	allFeatures := decodeFilterResponse(rrFilterAll)
	if len(allFeatures) < len(features) {
		t.Fatalf("expected all-rides filter to return at least %d rides, got %d", len(features), len(allFeatures))
	}
}

func writeUploadActivityParquet(t *testing.T, path string) {
	t.Helper()
	writeUploadActivityParquetWithCoords(t, path, [][2]float64{
		{-71.1000, 42.3600},
		{-71.0998, 42.3601},
		{-71.0996, 42.3602},
		{-71.0994, 42.3603},
		{-71.0992, 42.3604},
		{-71.0990, 42.3605},
	})
}

func writeUploadActivityParquetWithCoords(t *testing.T, path string, coords [][2]float64) {
	t.Helper()
	writeUploadActivityParquetWithGeometry(
		t,
		path,
		`{"version":"1.1.0","primary_column":"geometry","columns":{"geometry":{"encoding":"WKB","geometry_types":["LineString"]}}}`,
		strava.LineStringWKB(coords),
	)
}

func writeUploadActivityParquetWithParts(t *testing.T, path string, parts [][][2]float64) {
	t.Helper()

	geometry := strava.MultiLineStringWKB(parts)
	if len(parts) == 1 {
		geometry = strava.LineStringWKB(parts[0])
	}

	writeUploadActivityParquetWithGeometry(
		t,
		path,
		`{"version":"1.1.0","primary_column":"geometry","columns":{"geometry":{"encoding":"WKB","geometry_types":["LineString","MultiLineString"]}}}`,
		geometry,
	)
}

func writeUploadActivityParquetWithGeometry(t *testing.T, path, geoMetadata string, geometry []byte) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create parquet %q: %v", path, err)
	}

	writer := parquet.NewGenericWriter[strava.ActivityRow](file,
		parquet.KeyValueMetadata("geo", geoMetadata),
	)
	rows := []strava.ActivityRow{{
		ActivityID:   99,
		ActivityDate: "2026-08-19",
		ActivityName: "Legacy Ride",
		ActivityType: "Ride",
		Filename:     "legacy.fit",
		Geometry:     geometry,
	}}
	if _, err := writer.Write(rows); err != nil {
		_ = file.Close()
		t.Fatalf("write parquet %q: %v", path, err)
	}
	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatalf("close parquet writer %q: %v", path, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close parquet %q: %v", path, err)
	}
}

func decodeGeoJSONLineParts(t *testing.T, geometryValue any) [][][2]float64 {
	t.Helper()

	geometry, ok := geometryValue.(map[string]any)
	if !ok {
		t.Fatalf("expected geometry object, got %T", geometryValue)
	}

	switch geometry["type"] {
	case "LineString":
		return [][][2]float64{decodeGeoJSONLine(t, geometry["coordinates"])}
	case "MultiLineString":
		rawParts, ok := geometry["coordinates"].([]any)
		if !ok {
			t.Fatalf("expected multiline coordinates, got %T", geometry["coordinates"])
		}

		parts := make([][][2]float64, 0, len(rawParts))
		for _, rawPart := range rawParts {
			parts = append(parts, decodeGeoJSONLine(t, rawPart))
		}
		return parts
	default:
		t.Fatalf("expected line geometry, got %v", geometry["type"])
		return nil
	}
}

func decodeGeoJSONLine(t *testing.T, coordinatesValue any) [][2]float64 {
	t.Helper()

	rawCoordinates, ok := coordinatesValue.([]any)
	if !ok {
		t.Fatalf("expected line coordinates, got %T", coordinatesValue)
	}

	coordinates := make([][2]float64, 0, len(rawCoordinates))
	for _, rawCoordinate := range rawCoordinates {
		pair, ok := rawCoordinate.([]any)
		if !ok || len(pair) != 2 {
			t.Fatalf("expected coordinate pair, got %T", rawCoordinate)
		}

		lng, lngOK := pair[0].(float64)
		lat, latOK := pair[1].(float64)
		if !lngOK || !latOK {
			t.Fatalf("expected numeric coordinate pair, got %v", pair)
		}

		coordinates = append(coordinates, [2]float64{lng, lat})
	}

	return coordinates
}

func readUploadActivityRow(t *testing.T, path string) strava.ActivityRow {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open parquet %q: %v", path, err)
	}
	defer file.Close()

	reader := parquet.NewGenericReader[strava.ActivityRow](file)
	defer reader.Close()

	rows := make([]strava.ActivityRow, 1)
	n, err := reader.Read(rows)
	if err != nil && err != io.EOF {
		t.Fatalf("read parquet %q: %v", path, err)
	}
	if n != 1 {
		t.Fatalf("expected one row in %q, got %d", path, n)
	}

	return rows[0]
}
func TestGetURL(t *testing.T) {
	tests := []struct {
		addr     string
		expected string
	}{
		{":8080", "http://localhost:8080"},
		{"localhost:8080", "http://localhost:8080"},
		{"127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"0.0.0.0:8080", "http://localhost:8080"},
		{"[::]:8080", "http://localhost:8080"},
		{":80", "http://localhost:80"},
		{"invalid-addr", "http://localhostinvalid-addr"},
	}

	for _, tt := range tests {
		actual := getURL(tt.addr)
		if actual != tt.expected {
			t.Errorf("getURL(%q) = %q, expected %q", tt.addr, actual, tt.expected)
		}
	}
}
