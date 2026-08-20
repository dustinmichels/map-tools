package strava

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/parquet-go/parquet-go"
)

func TestSimplifyParquetShrinksStoredGeometry(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "rides.parquet")
	outputPath := SimplifiedParquetPath(inputPath)

	rows := []ActivityRow{{
		ActivityID:   42,
		ActivityDate: "2026-08-19",
		ActivityName: "Morning Ride",
		ActivityType: "Ride",
		Filename:     "ride.fit",
		Geometry: LineStringWKB([][2]float64{
			{-71.1000, 42.3600},
			{-71.0998, 42.3601},
			{-71.0996, 42.3602},
			{-71.0994, 42.3603},
			{-71.0992, 42.3604},
			{-71.0990, 42.3605},
		}),
	}}
	writeActivityParquet(t, inputPath, rows)

	if err := SimplifyParquet(inputPath, outputPath); err != nil {
		t.Fatalf("SimplifyParquet: %v", err)
	}

	original := readSingleActivityRow(t, inputPath)
	simplified := readSingleActivityRow(t, outputPath)

	if simplified.ActivityID != original.ActivityID {
		t.Fatalf("expected activity id %d, got %d", original.ActivityID, simplified.ActivityID)
	}
	if simplified.ActivityName != original.ActivityName {
		t.Fatalf("expected activity name %q, got %q", original.ActivityName, simplified.ActivityName)
	}
	if simplified.Filename != original.Filename {
		t.Fatalf("expected filename %q, got %q", original.Filename, simplified.Filename)
	}
	if len(simplified.Geometry) >= len(original.Geometry) {
		t.Fatalf("expected simplified geometry to shrink, got original=%d simplified=%d", len(original.Geometry), len(simplified.Geometry))
	}
}

func writeActivityParquet(t *testing.T, path string, rows []ActivityRow) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create parquet %q: %v", path, err)
	}

	writer := parquet.NewGenericWriter[ActivityRow](file,
		parquet.KeyValueMetadata("geo", geoMetadata),
	)
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
