package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"

	"uuid"

	"github.com/dustinmichels/map-tools/internal/strava"
	"github.com/go-chi/chi/v5"
)

func apiRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", handleHealth)
	r.Get("/uploads", handleListUploads)
	r.Post("/upload", handleUpload)
	r.Post("/uploads/{datasetId}/open", handleOpenUpload)
	r.Post("/uploads/{datasetId}/simplify", handleSimplifyUpload)
	r.Patch("/uploads/{datasetId}", handleRenameUpload)
	r.Delete("/uploads/{datasetId}", handleDeleteUpload)
	r.Post("/filter", handleFilter)
	r.Post("/transcode", handleTranscode)
	r.Get("/transcode/check", handleTranscodeCheck)
	return r
}

type requestDecoder struct{}

func (requestDecoder) decode[T any](r *http.Request) (T, error) {
	var val T
	err := json.NewDecoder(r.Body).Decode(&val)
	return val, err
}

type UploadedDataset struct {
	DatasetID     string    `json:"datasetId"`
	FileName      string    `json:"fileName"`
	DisplayName   string    `json:"displayName"`
	CreatedAt     time.Time `json:"createdAt"`
	SizeBytes     int64     `json:"sizeBytes"`
	HasSimplified bool      `json:"hasSimplified"`
	Total         *int      `json:"total,omitempty"`
	Parsed        *int      `json:"parsed,omitempty"`
	RideCount     *int      `json:"rideCount,omitempty"`
}

type uploadMetadata struct {
	DatasetID   string    `json:"datasetId"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
	Total       *int      `json:"total,omitempty"`
	Parsed      *int      `json:"parsed,omitempty"`
	RideCount   *int      `json:"rideCount,omitempty"`
}

type uploadRecord struct {
	dataset      UploadedDataset
	parquetPath  string
	metadataPath string
}

var openUploadPath = revealPathInFileManager
var nowUTC = func() time.Time {
	return time.Now().UTC()
}
var ingestZip = strava.IngestZip

type UploadResponse struct {
	Status    string          `json:"status"`
	SessionID string          `json:"sessionId"`
	Total     int             `json:"total"`
	Parsed    int             `json:"parsed"`
	RideCount int             `json:"rideCount"`
	Summary   string          `json:"summary"`
	Dataset   UploadedDataset `json:"dataset"`
}

type listUploadsResponse struct {
	Uploads []UploadedDataset `json:"uploads"`
}

type renameUploadRequest struct {
	Name string `json:"name"`
}

func handleListUploads(w http.ResponseWriter, r *http.Request) {
	uploads, err := listUploadedDatasets()
	if err != nil {
		slog.Error("failed to list uploads", "err", err)
		http.Error(w, "failed to list uploads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(listUploadsResponse{Uploads: uploads})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	slog.Info("received upload request")

	if err := r.ParseMultipartForm(500 << 20); err != nil {
		slog.Error("failed to parse multipart form", "err", err)
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		slog.Error("failed to get file from form", "err", err)
		http.Error(w, "invalid file key 'file' in form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	response, err := processUploadedArchive(file)
	if err != nil {
		slog.Error("ingest failed", "sessionId", response.SessionID, "err", err)
		http.Error(w, fmt.Sprintf("ingest failed: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info(response.Summary, "sessionId", response.SessionID, "parquetPath", response.Dataset.FileName)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func handleRenameUpload(w http.ResponseWriter, r *http.Request) {
	datasetID := strings.TrimSpace(chi.URLParam(r, "datasetId"))
	if datasetID == "" {
		http.Error(w, "datasetId is required", http.StatusBadRequest)
		return
	}

	var reqDecoder requestDecoder
	req, err := reqDecoder.decode[renameUploadRequest](r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	displayName := strings.TrimSpace(req.Name)
	targetBaseName := sanitizeUploadFileName(displayName)
	if targetBaseName == "" {
		http.Error(w, "name must contain letters or numbers", http.StatusBadRequest)
		return
	}

	record, err := resolveUploadRecord(datasetID)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}

		slog.Error("failed to resolve upload for rename", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to rename upload", http.StatusInternalServerError)
		return
	}

	targetDir, err := uploadDataDir()
	if err != nil {
		slog.Error("failed to resolve upload data directory", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to rename upload", http.StatusInternalServerError)
		return
	}

	targetParquetPath := filepath.Join(targetDir, targetBaseName+".parquet")
	targetSimplifiedPath := strava.SimplifiedParquetPath(targetParquetPath)
	sourceSimplifiedPath := strava.SimplifiedParquetPath(record.parquetPath)
	targetMetadataPath := uploadMetadataPath(targetParquetPath)
	if targetParquetPath != record.parquetPath {
		if _, err := os.Stat(targetParquetPath); err == nil {
			http.Error(w, "a parquet file with that name already exists", http.StatusConflict)
			return
		}
		if targetSimplifiedPath != sourceSimplifiedPath {
			if _, err := os.Stat(targetSimplifiedPath); err == nil {
				http.Error(w, "a simplified parquet file with that name already exists", http.StatusConflict)
				return
			}
		}

		if err := os.Rename(record.parquetPath, targetParquetPath); err != nil {
			slog.Error("failed to rename parquet file", "datasetId", datasetID, "err", err)
			http.Error(w, "failed to rename upload", http.StatusInternalServerError)
			return
		}
		if targetSimplifiedPath != sourceSimplifiedPath && fileExists(sourceSimplifiedPath) {
			if err := os.Rename(sourceSimplifiedPath, targetSimplifiedPath); err != nil {
				slog.Error("failed to rename simplified parquet file", "datasetId", datasetID, "err", err)
				http.Error(w, "failed to rename upload", http.StatusInternalServerError)
				return
			}
		}
		if targetMetadataPath != record.metadataPath && fileExists(record.metadataPath) {
			if err := os.Rename(record.metadataPath, targetMetadataPath); err != nil {
				slog.Error("failed to rename upload metadata", "datasetId", datasetID, "err", err)
				http.Error(w, "failed to rename upload metadata", http.StatusInternalServerError)
				return
			}
		}
	}

	metadata := uploadMetadata{
		DatasetID:   record.dataset.DatasetID,
		DisplayName: displayName,
		CreatedAt:   record.dataset.CreatedAt,
		Total:       record.dataset.Total,
		Parsed:      record.dataset.Parsed,
		RideCount:   record.dataset.RideCount,
	}
	if err := writeUploadMetadata(targetMetadataPath, metadata); err != nil {
		slog.Error("failed to write upload metadata", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to rename upload", http.StatusInternalServerError)
		return
	}

	info, err := os.Stat(targetParquetPath)
	if err != nil {
		slog.Error("failed to stat renamed upload", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to rename upload", http.StatusInternalServerError)
		return
	}

	updated := UploadedDataset{
		DatasetID:     record.dataset.DatasetID,
		FileName:      filepath.Base(targetParquetPath),
		DisplayName:   displayName,
		CreatedAt:     record.dataset.CreatedAt,
		SizeBytes:     info.Size(),
		HasSimplified: fileExists(targetSimplifiedPath),
		Total:         record.dataset.Total,
		Parsed:        record.dataset.Parsed,
		RideCount:     record.dataset.RideCount,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func handleSimplifyUpload(w http.ResponseWriter, r *http.Request) {
	datasetID := strings.TrimSpace(chi.URLParam(r, "datasetId"))
	if datasetID == "" {
		http.Error(w, "datasetId is required", http.StatusBadRequest)
		return
	}

	record, err := resolveUploadRecord(datasetID)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}

		slog.Error("failed to resolve upload for simplify", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to simplify upload", http.StatusInternalServerError)
		return
	}

	simplifiedPath, err := ensureSimplifiedUploadParquet(record)
	if err != nil {
		slog.Error("failed to write simplified parquet file", "datasetId", datasetID, "parquetPath", record.parquetPath, "simplifiedPath", strava.SimplifiedParquetPath(record.parquetPath), "err", err)
		http.Error(w, "failed to simplify upload", http.StatusInternalServerError)
		return
	}

	updated := record.dataset
	updated.HasSimplified = fileExists(simplifiedPath)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func handleOpenUpload(w http.ResponseWriter, r *http.Request) {
	datasetID := strings.TrimSpace(chi.URLParam(r, "datasetId"))
	if datasetID == "" {
		http.Error(w, "datasetId is required", http.StatusBadRequest)
		return
	}

	record, err := resolveUploadRecord(datasetID)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}

		slog.Error("failed to resolve upload for open", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to open upload", http.StatusInternalServerError)
		return
	}

	if err := openUploadPath(record.parquetPath); err != nil {
		slog.Error("failed to open upload in file manager", "datasetId", datasetID, "parquetPath", record.parquetPath, "err", err)
		http.Error(w, "failed to open upload", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleDeleteUpload(w http.ResponseWriter, r *http.Request) {
	datasetID := strings.TrimSpace(chi.URLParam(r, "datasetId"))
	if datasetID == "" {
		http.Error(w, "datasetId is required", http.StatusBadRequest)
		return
	}

	record, err := resolveUploadRecord(datasetID)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}

		slog.Error("failed to resolve upload for delete", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to delete upload", http.StatusInternalServerError)
		return
	}

	if err := os.Remove(record.parquetPath); err != nil && !os.IsNotExist(err) {
		slog.Error("failed to remove parquet file", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to delete upload", http.StatusInternalServerError)
		return
	}
	if err := os.Remove(strava.SimplifiedParquetPath(record.parquetPath)); err != nil && !os.IsNotExist(err) {
		slog.Error("failed to remove simplified parquet file", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to delete upload", http.StatusInternalServerError)
		return
	}

	if err := os.Remove(record.metadataPath); err != nil && !os.IsNotExist(err) {
		slog.Error("failed to remove upload metadata", "datasetId", datasetID, "err", err)
		http.Error(w, "failed to delete upload metadata", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type FilterRequest struct {
	SessionId    string      `json:"sessionId"`
	BBox         *[4]float64 `json:"bbox,omitempty"`
	GeometryMode string      `json:"geometryMode,omitempty"`
	Limit        int         `json:"limit,omitempty"`
}

func buildBBoxEnvelope(bbox *[4]float64) string {
	if bbox == nil {
		return ""
	}

	return fmt.Sprintf(
		"ST_MakeEnvelope(%f, %f, %f, %f)",
		bbox[0], bbox[1], bbox[2], bbox[3],
	)
}

func buildRideFilterWhereClause(bbox *[4]float64) string {
	conditions := []string{"activity_type = 'Ride'"}
	if bboxEnvelope := buildBBoxEnvelope(bbox); bboxEnvelope != "" {
		conditions = append(conditions, fmt.Sprintf("ST_Intersects(geometry, %s)", bboxEnvelope))
	}

	return "WHERE " + strings.Join(conditions, " AND ")
}

func buildRouteGeometrySelect(bbox *[4]float64) string {
	if bboxEnvelope := buildBBoxEnvelope(bbox); bboxEnvelope != "" {
		return fmt.Sprintf("ST_Intersection(geometry, %s) AS geometry", bboxEnvelope)
	}

	return "geometry"
}

func buildFilterFromClause(parquetPath, whereClause string, limit int) string {
	quotedParquetPath := quoteDuckDBPath(parquetPath)
	if limit <= 0 {
		return fmt.Sprintf("FROM read_parquet('%s')\n  %s", quotedParquetPath, whereClause)
	}

	return fmt.Sprintf(
		"FROM (SELECT * FROM read_parquet('%s')\n    %s) AS filtered\n  USING SAMPLE %d ROWS (reservoir)",
		quotedParquetPath,
		whereClause,
		limit,
	)
}

func resolveFilterParquetPath(record uploadRecord, geometryMode string) (string, error) {
	if geometryMode == "original" {
		return record.parquetPath, nil
	}

	return ensureSimplifiedUploadParquet(record)
}

func handleFilter(w http.ResponseWriter, r *http.Request) {
	slog.Info("received filter request")

	var reqDecoder requestDecoder
	req, err := reqDecoder.decode[FilterRequest](r)
	if err != nil {
		slog.Error("failed to decode filter request", "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Limit < 0 {
		http.Error(w, "limit must be non-negative", http.StatusBadRequest)
		return
	}

	record, err := resolveUploadRecord(req.SessionId)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Error("session parquet file not found", "sessionId", req.SessionId)
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		slog.Error("failed to resolve session parquet file", "sessionId", req.SessionId, "err", err)
		http.Error(w, "failed to resolve session", http.StatusInternalServerError)
		return
	}

	slog.Info(
		"filtering activities with duckdb",
		"sessionId", req.SessionId,
		"bbox", req.BBox,
		"geometryMode", req.GeometryMode,
		"limit", req.Limit,
	)

	whereClause := buildRideFilterWhereClause(req.BBox)
	parquetPath, err := resolveFilterParquetPath(record, req.GeometryMode)
	if err != nil {
		slog.Error(
			"failed to prepare parquet for filter",
			"sessionId", req.SessionId,
			"geometryMode", req.GeometryMode,
			"parquetPath", record.parquetPath,
			"err", err,
		)
		http.Error(w, "failed to prepare geometry", http.StatusInternalServerError)
		return
	}

	geoJSONData, err := exportGeoJSONFromParquet(parquetPath, whereClause, req.BBox, req.Limit)
	if err != nil {
		slog.Error("duckdb filter query failed", "err", err)
		http.Error(w, fmt.Sprintf("filter query failed: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("filtering completed successfully", "sessionId", req.SessionId, "bytes", len(geoJSONData))
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(geoJSONData)
}

func exportGeoJSONFromParquet(parquetPath, whereClause string, bbox *[4]float64, limit int) ([]byte, error) {
	outputFile, err := os.CreateTemp("", "maptools-output-*.geojson")
	if err != nil {
		return nil, err
	}

	outputGeoJSON := outputFile.Name()
	if err := outputFile.Close(); err != nil {
		return nil, err
	}
	if err := os.Remove(outputGeoJSON); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`INSTALL spatial;
LOAD spatial;
SET geometry_always_xy = true;
COPY (
  SELECT
    activity_id AS route_id,
    activity_date AS route_date,
    activity_id,
    activity_date,
    activity_name,
    activity_type,
    distance_km AS distance,
    %s
  %s
  ORDER BY activity_date ASC
) TO '%s' WITH (FORMAT 'GDAL', DRIVER 'GeoJSON');`,
		buildRouteGeometrySelect(bbox),
		buildFilterFromClause(parquetPath, whereClause, limit),
		quoteDuckDBPath(outputGeoJSON),
	)

	cmd := exec.Command("duckdb", "-c", query)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("duckdb query failed: %w, stderr: %s", err, stderr.String())
	}
	defer os.Remove(outputGeoJSON)

	return os.ReadFile(outputGeoJSON)
}

func processUploadedArchive(file io.Reader) (UploadResponse, error) {
	response := UploadResponse{}

	uploadDir, err := uploadDataDir()
	if err != nil {
		return response, err
	}

	createdAt := nowUTC()
	fileBaseName, err := nextBulkUploadBaseName(createdAt)
	if err != nil {
		return response, err
	}

	sessionID := uuid.New().String()
	parquetPath := filepath.Join(uploadDir, fileBaseName+".parquet")
	zipFile, err := os.CreateTemp("", "maptools-upload-*.zip")
	if err != nil {
		return response, err
	}

	zipPath := zipFile.Name()
	if _, err := io.Copy(zipFile, file); err != nil {
		zipFile.Close()
		_ = os.Remove(zipPath)
		return response, err
	}
	if err := zipFile.Close(); err != nil {
		_ = os.Remove(zipPath)
		return response, err
	}

	slog.Info("processing upload zip", "sessionId", sessionID, "zipPath", zipPath)
	res, err := ingestZip(zipPath, parquetPath)
	_ = os.Remove(zipPath)
	if err != nil {
		_ = os.Remove(parquetPath)
		_ = os.Remove(strava.SimplifiedParquetPath(parquetPath))
		return UploadResponse{SessionID: sessionID}, err
	}

	total := res.Total
	parsed := res.Parsed
	rideCount := res.RideCount
	displayName := fileBaseName
	metadata := uploadMetadata{
		DatasetID:   sessionID,
		DisplayName: displayName,
		CreatedAt:   createdAt,
		Total:       &total,
		Parsed:      &parsed,
		RideCount:   &rideCount,
	}
	if err := writeUploadMetadata(uploadMetadataPath(parquetPath), metadata); err != nil {
		return UploadResponse{SessionID: sessionID}, err
	}

	info, err := os.Stat(parquetPath)
	if err != nil {
		return UploadResponse{SessionID: sessionID}, err
	}

	summary := fmt.Sprintf("Succesfully parsed %d / %d  activities. %d are type = Ride.", res.Parsed, res.Total, res.RideCount)
	return UploadResponse{
		Status:    "success",
		SessionID: sessionID,
		Total:     res.Total,
		Parsed:    res.Parsed,
		RideCount: res.RideCount,
		Summary:   summary,
		Dataset: UploadedDataset{
			DatasetID:     sessionID,
			FileName:      filepath.Base(parquetPath),
			DisplayName:   displayName,
			CreatedAt:     createdAt,
			SizeBytes:     info.Size(),
			HasSimplified: fileExists(strava.SimplifiedParquetPath(parquetPath)),
			Total:         &total,
			Parsed:        &parsed,
			RideCount:     &rideCount,
		},
	}, nil
}

func listUploadedDatasets() ([]UploadedDataset, error) {
	records, err := listUploadRecords()
	if err != nil {
		return nil, err
	}

	uploads := make([]UploadedDataset, 0, len(records))
	for _, record := range records {
		uploads = append(uploads, record.dataset)
	}

	return uploads, nil
}

func resolveUploadRecord(datasetID string) (uploadRecord, error) {
	if strings.TrimSpace(datasetID) == "" {
		return uploadRecord{}, os.ErrNotExist
	}

	records, err := listUploadRecords()
	if err != nil {
		return uploadRecord{}, err
	}

	for _, record := range records {
		if record.dataset.DatasetID == datasetID {
			return record, nil
		}
	}

	return uploadRecord{}, os.ErrNotExist
}

func listUploadRecords() ([]uploadRecord, error) {
	uploadDir, err := uploadDataDir()
	if err != nil {
		return nil, err
	}

	records := make([]uploadRecord, 0)
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			return records, nil
		}
		return nil, err
	}

	seenDatasetIDs := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".parquet" || isSimplifiedParquetPath(entry.Name()) {
			continue
		}

		record, err := buildUploadRecord(uploadDir, entry)
		if err != nil {
			return nil, err
		}
		if _, seen := seenDatasetIDs[record.dataset.DatasetID]; seen {
			continue
		}

		seenDatasetIDs[record.dataset.DatasetID] = struct{}{}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].dataset.CreatedAt.After(records[j].dataset.CreatedAt)
	})

	return records, nil
}

func uploadDataDir() (string, error) {
	overrideDir := strings.TrimSpace(os.Getenv("MAPTOOLS_DATA_DIR"))
	if overrideDir != "" {
		if err := os.MkdirAll(overrideDir, 0755); err != nil {
			return "", err
		}
		return overrideDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	uploadDir := ""
	switch runtime.GOOS {
	case "darwin":
		uploadDir = filepath.Join(homeDir, "Library", "Application Support", "MapTools", "data")
	case "windows":
		appDataDir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		uploadDir = filepath.Join(appDataDir, "MapTools", "data")
	default:
		dataHomeDir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
		if dataHomeDir == "" {
			dataHomeDir = filepath.Join(homeDir, ".local", "share")
		}
		uploadDir = filepath.Join(dataHomeDir, "MapTools", "data")
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	return uploadDir, nil
}

func buildUploadRecord(uploadDir string, entry os.DirEntry) (uploadRecord, error) {
	parquetPath := filepath.Join(uploadDir, entry.Name())
	info, err := entry.Info()
	if err != nil {
		return uploadRecord{}, err
	}

	record := uploadRecord{
		dataset: UploadedDataset{
			DatasetID:     strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			FileName:      entry.Name(),
			DisplayName:   strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			CreatedAt:     info.ModTime(),
			SizeBytes:     info.Size(),
			HasSimplified: fileExists(strava.SimplifiedParquetPath(parquetPath)),
		},
		parquetPath:  parquetPath,
		metadataPath: uploadMetadataPath(parquetPath),
	}

	if metadata, err := readUploadMetadata(record.metadataPath); err == nil {
		if metadata.DatasetID != "" {
			record.dataset.DatasetID = metadata.DatasetID
		}
		if strings.TrimSpace(metadata.DisplayName) != "" {
			record.dataset.DisplayName = metadata.DisplayName
		}
		if !metadata.CreatedAt.IsZero() {
			record.dataset.CreatedAt = metadata.CreatedAt
		}
		record.dataset.Total = metadata.Total
		record.dataset.Parsed = metadata.Parsed
		record.dataset.RideCount = metadata.RideCount
	} else if !os.IsNotExist(err) {
		return uploadRecord{}, err
	}

	return record, nil
}

func readUploadMetadata(path string) (uploadMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return uploadMetadata{}, err
	}

	var metadata uploadMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return uploadMetadata{}, err
	}

	return metadata, nil
}

func writeUploadMetadata(path string, metadata uploadMetadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func uploadMetadataPath(parquetPath string) string {
	return strings.TrimSuffix(parquetPath, filepath.Ext(parquetPath)) + ".upload.json"
}

func isSimplifiedParquetPath(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strings.HasSuffix(base, "__simplified")
}

func ensureSimplifiedUploadParquet(record uploadRecord) (string, error) {
	simplifiedPath := strava.SimplifiedParquetPath(record.parquetPath)
	if fileExists(simplifiedPath) {
		return simplifiedPath, nil
	}

	if err := strava.SimplifyParquet(record.parquetPath, simplifiedPath); err != nil {
		return "", err
	}

	return simplifiedPath, nil
}

func quoteDuckDBPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), "'", "''")
}

func revealPathInFileManager(path string) error {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", absolutePath).Run()
	case "windows":
		return exec.Command("explorer.exe", "/select,", absolutePath).Run()
	default:
		return exec.Command("xdg-open", filepath.Dir(absolutePath)).Run()
	}
}

func nextBulkUploadBaseName(createdAt time.Time) (string, error) {
	baseName := "bulk_upload_" + createdAt.UTC().Format(time.DateOnly)
	uploadDir, err := uploadDataDir()
	if err != nil {
		return "", err
	}

	for suffix := 1; ; suffix++ {
		candidate := baseName
		if suffix > 1 {
			candidate = fmt.Sprintf("%s_%d", baseName, suffix)
		}

		path := filepath.Join(uploadDir, candidate+".parquet")
		if !fileExists(path) {
			return candidate, nil
		}
	}
}

func sanitizeUploadFileName(name string) string {
	name = strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	if name == "" {
		return ""
	}

	var builder strings.Builder
	lastSeparator := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastSeparator = false
		case r == ' ' || r == '-' || r == '_':
			if !lastSeparator {
				builder.WriteRune(r)
				lastSeparator = true
			}
		default:
			if !lastSeparator {
				builder.WriteRune('-')
				lastSeparator = true
			}
		}
	}

	return strings.Trim(builder.String(), " -_")
}

type HealthResponse struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func handleTranscodeCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := exec.LookPath("ffmpeg")
	available := err == nil
	_ = json.NewEncoder(w).Encode(map[string]bool{"available": available})
}

func handleTranscode(w http.ResponseWriter, r *http.Request) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		http.Error(w, "ffmpeg not found on server", http.StatusServiceUnavailable)
		return
	}

	tmpIn, err := os.CreateTemp("", "transcode-*.webm")
	if err != nil {
		http.Error(w, "failed to create temp file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpIn.Name())
	defer tmpIn.Close()

	_, err = io.Copy(tmpIn, r.Body)
	if err != nil {
		http.Error(w, "failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	tmpIn.Close()

	tmpOutName := tmpIn.Name() + ".mp4"
	defer os.Remove(tmpOutName)

	cmd := exec.Command(ffmpegPath, "-y", "-i", tmpIn.Name(), "-c:v", "libx264", "-pix_fmt", "yuv420p", "-crf", "23", tmpOutName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("ffmpeg failed: %v\nstderr: %s", err, stderr.String()), http.StatusInternalServerError)
		return
	}

	outFile, err := os.Open(tmpOutName)
	if err != nil {
		http.Error(w, "failed to open transcoded file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	w.Header().Set("Content-Type", "video/mp4")
	_, _ = io.Copy(w, outFile)
}
