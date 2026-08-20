package strava

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/parquet-go/parquet-go"
)

// geoMetadata is the GeoParquet file-level metadata value for the "geo" key.
const geoMetadata = `{"version":"1.1.0","primary_column":"geometry","columns":{"geometry":{"encoding":"WKB","geometry_types":["LineString"]}}}`

// simplifiedGeoMetadata allows simplified rows to split pauses into MultiLineString geometries.
const simplifiedGeoMetadata = `{"version":"1.1.0","primary_column":"geometry","columns":{"geometry":{"encoding":"WKB","geometry_types":["LineString","MultiLineString"]}}}`

// IngestResult holds counts of activities processed during ingest.
type IngestResult struct {
	Total     int
	Parsed    int
	RideCount int
}

// IngestZip reads a Strava bulk-export zip, processes all supported GPS
// activity formats (.fit.gz, .gpx), and writes a geoparquet file to outPath
// plus a companion simplified geoparquet file.
// The zip is read in-place; no temporary extraction is performed.
func IngestZip(zipPath, outPath string) (IngestResult, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return IngestResult{}, fmt.Errorf("open zip %q: %w", zipPath, err)
	}
	defer r.Close()

	fileMap := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	var csvEntry *zip.File
	for _, f := range r.File {
		if filepath.Base(f.Name) == "activities.csv" {
			csvEntry = f
			break
		}
	}
	if csvEntry == nil {
		return IngestResult{}, fmt.Errorf("activities.csv not found in %s", zipPath)
	}

	prefix := strings.TrimSuffix(csvEntry.Name, "activities.csv")

	rc, err := csvEntry.Open()
	if err != nil {
		return IngestResult{}, fmt.Errorf("open activities.csv in zip: %w", err)
	}
	activities, err := ParseActivities(rc)
	rc.Close()
	if err != nil {
		return IngestResult{}, fmt.Errorf("parse activities.csv: %w", err)
	}

	opener := func(filename string) ([]trackPoint, error) {
		name := prefix + filename
		zf, ok := fileMap[name]
		if !ok {
			return nil, fmt.Errorf("not found in zip: %s", name)
		}
		rc, err := zf.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return routeTrack(filename, rc)
	}

	return processActivities(activities, opener, outPath)
}

// IngestDir reads an already-extracted Strava export directory and writes
// a geoparquet file to outPath plus a companion simplified geoparquet file.
// Useful for faster iteration during development.
func IngestDir(dir, outPath string) (IngestResult, error) {
	f, err := os.Open(filepath.Join(dir, "activities.csv"))
	if err != nil {
		return IngestResult{}, fmt.Errorf("open activities.csv: %w", err)
	}
	activities, err := ParseActivities(f)
	f.Close()
	if err != nil {
		return IngestResult{}, fmt.Errorf("parse activities.csv: %w", err)
	}

	opener := func(filename string) ([]trackPoint, error) {
		path := filepath.Join(dir, filename)
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return routeTrack(filename, f)
	}

	return processActivities(activities, opener, outPath)
}

// processActivities is the shared core: filters to supported GPS activity
// formats, parses tracks, and writes one geoparquet row per activity.
func processActivities(
	activities []Activity,
	openTrack func(filename string) ([]trackPoint, error),
	outPath string,
) (IngestResult, error) {
	var result IngestResult
	result.Total = len(activities)

	out, err := os.Create(outPath)
	if err != nil {
		return result, fmt.Errorf("create %q: %w", outPath, err)
	}
	defer out.Close()

	simplifiedOutPath := SimplifiedParquetPath(outPath)
	simplifiedOut, err := os.Create(simplifiedOutPath)
	if err != nil {
		return result, fmt.Errorf("create %q: %w", simplifiedOutPath, err)
	}
	defer simplifiedOut.Close()

	writer := parquet.NewGenericWriter[ActivityRow](out,
		parquet.KeyValueMetadata("geo", geoMetadata),
	)
	simplifiedWriter := parquet.NewGenericWriter[ActivityRow](simplifiedOut,
		parquet.KeyValueMetadata("geo", simplifiedGeoMetadata),
	)

	type job struct {
		index int
		act   Activity
	}

	type parseResult struct {
		index         int
		row           *ActivityRow
		simplifiedRow *ActivityRow
	}

	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	var filterCount int
	for _, act := range activities {
		if act.Filename != nil && isSupportedTrack(*act.Filename) {
			filterCount++
		}
	}
	if numWorkers > filterCount {
		numWorkers = filterCount
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	jobs := make(chan job, filterCount)
	results := make(chan parseResult, filterCount)

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				points, err := openTrack(*j.act.Filename)
				if err != nil {
					slog.Warn("skipping: track read error", "id", j.act.ActivityID, "file", *j.act.Filename, "err", err)
					results <- parseResult{index: j.index}
					continue
				}

				wkb := LineStringWKB(trackPointsToCoords(points))
				if wkb == nil {
					slog.Warn("skipping: no GPS data", "id", j.act.ActivityID, "file", *j.act.Filename)
					results <- parseResult{index: j.index}
					continue
				}

				simplifiedWKB := simplifiedGeometryWKB(points)
				if simplifiedWKB == nil {
					simplifiedWKB = wkb
				}

				row := activityToRow(j.act, wkb)
				simplifiedRow := activityToRow(j.act, simplifiedWKB)
				results <- parseResult{index: j.index, row: &row, simplifiedRow: &simplifiedRow}
			}
		}()
	}

	for i, act := range activities {
		if act.Filename == nil || !isSupportedTrack(*act.Filename) {
			continue
		}
		jobs <- job{index: i, act: act}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	parsedRows := make([]*ActivityRow, len(activities))
	simplifiedRows := make([]*ActivityRow, len(activities))
	var skipped int
	for res := range results {
		if res.row != nil {
			parsedRows[res.index] = res.row
			simplifiedRows[res.index] = res.simplifiedRow
		} else {
			skipped++
		}
	}

	var written int
	for i, row := range parsedRows {
		if row == nil {
			continue
		}
		if _, err := writer.Write([]ActivityRow{*row}); err != nil {
			writer.Close()
			simplifiedWriter.Close()
			return result, fmt.Errorf("write row for activity %d: %w", row.ActivityID, err)
		}

		simplifiedRow := simplifiedRows[i]
		if simplifiedRow == nil {
			simplifiedRow = row
		}
		if _, err := simplifiedWriter.Write([]ActivityRow{*simplifiedRow}); err != nil {
			writer.Close()
			simplifiedWriter.Close()
			return result, fmt.Errorf("write simplified row for activity %d: %w", row.ActivityID, err)
		}

		written++
		if row.ActivityType == "Ride" {
			result.RideCount++
		}
	}

	if err := writer.Close(); err != nil {
		simplifiedWriter.Close()
		return result, fmt.Errorf("close parquet writer: %w", err)
	}
	if err := simplifiedWriter.Close(); err != nil {
		return result, fmt.Errorf("close simplified parquet writer: %w", err)
	}

	result.Parsed = written

	slog.Info("geoparquet written",
		"path", outPath,
		"rows", written,
		"skipped", skipped,
	)
	slog.Info("simplified geoparquet written",
		"path", simplifiedOutPath,
		"rows", written,
	)
	return result, nil
}

// isSupportedTrack reports whether a filename has a supported GPS format.
func isSupportedTrack(filename string) bool {
	return strings.HasSuffix(filename, ".fit.gz") ||
		strings.HasSuffix(filename, ".fit") ||
		strings.HasSuffix(filename, ".gpx.gz") ||
		strings.HasSuffix(filename, ".gpx") ||
		strings.HasSuffix(filename, ".tcx.gz")
}

// routeTrack dispatches to the correct parser based on file extension.
func routeTrack(filename string, r io.Reader) ([]trackPoint, error) {
	switch {
	case strings.HasSuffix(filename, ".fit.gz"):
		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("gzip: %w", err)
		}
		defer gr.Close()
		return readFITTrack(gr)
	case strings.HasSuffix(filename, ".fit"):
		return readFITTrack(r)
	case strings.HasSuffix(filename, ".gpx.gz"):
		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("gzip: %w", err)
		}
		defer gr.Close()
		return readGPXTrack(gr)
	case strings.HasSuffix(filename, ".gpx"):
		return readGPXTrack(r)
	case strings.HasSuffix(filename, ".tcx.gz"):
		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("gzip: %w", err)
		}
		defer gr.Close()
		return readTCXTrack(gr)
	default:
		return nil, fmt.Errorf("unsupported track format: %s", filename)
	}
}

func activityToRow(act Activity, wkb []byte) ActivityRow {
	row := ActivityRow{
		ActivityID:   act.ActivityID,
		ActivityDate: act.ActivityDate,
		ActivityName: act.ActivityName,
		ActivityType: act.ActivityType,
		Filename:     *act.Filename,
		Geometry:     wkb,
	}
	if act.ElapsedTime != nil {
		row.ElapsedTime = *act.ElapsedTime
	}
	if act.Distance != nil {
		row.Distance = *act.Distance
	}
	if act.MovingTime != nil {
		row.MovingTime = *act.MovingTime
	}
	if act.MaxSpeed != nil {
		row.MaxSpeed = *act.MaxSpeed
	}
	if act.AverageSpeed != nil {
		row.AverageSpeed = *act.AverageSpeed
	}
	if act.ElevationGain != nil {
		row.ElevationGain = *act.ElevationGain
	}
	if act.ElevationLoss != nil {
		row.ElevationLoss = *act.ElevationLoss
	}
	if act.AverageHeartRate != nil {
		row.AverageHeartRate = *act.AverageHeartRate
	}
	return row
}
