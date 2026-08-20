package strava_test

import (
	"os"
	"testing"

	"github.com/dustinmichels/map-tools/internal/strava"
	"github.com/parquet-go/parquet-go"
)

const (
	zipPath    = "../../data/strava_export.zip"
	exportDir  = "../../data/strava_export"
	outputPath = "../../data/activities.parquet"
)

// TestIngestZip processes the full Strava export zip and produces a geoparquet
// file at data/activities.parquet.  Run with -v to see row/skip counts.
func TestIngestZip(t *testing.T) {
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		t.Skipf("zip not found: %s", zipPath)
	}
	if _, err := strava.IngestZip(zipPath, outputPath); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file missing after ingest: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
	simplifiedPath := strava.SimplifiedParquetPath(outputPath)
	simplifiedInfo, err := os.Stat(simplifiedPath)
	if err != nil {
		t.Fatalf("simplified output file missing after ingest: %v", err)
	}
	if simplifiedInfo.Size() == 0 {
		t.Fatal("simplified output file is empty")
	}
	t.Logf("wrote %s (%d bytes)", outputPath, info.Size())
}

// TestIngestDir uses the already-extracted directory for faster iteration.
func TestIngestDir(t *testing.T) {
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		t.Skipf("export dir not found: %s", exportDir)
	}
	if _, err := strava.IngestDir(exportDir, outputPath); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("output file missing after ingest: %v", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	pf, err := parquet.OpenFile(f, info.Size())
	if err != nil {
		t.Fatalf("open parquet: %v", err)
	}

	simplifiedPath := strava.SimplifiedParquetPath(outputPath)
	simplifiedFile, err := os.Open(simplifiedPath)
	if err != nil {
		t.Fatalf("simplified output file missing after ingest: %v", err)
	}
	defer simplifiedFile.Close()

	simplifiedInfo, err := simplifiedFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	simplifiedParquet, err := parquet.OpenFile(simplifiedFile, simplifiedInfo.Size())
	if err != nil {
		t.Fatalf("open simplified parquet: %v", err)
	}

	// 944 .fit.gz + 208 .gpx + 12 .fit + 5 .gpx.gz + 3 .tcx.gz = 1172 total supported;
	// 26 have no GPS data → expect 1146 rows.
	const wantRows = 1146
	if got := pf.NumRows(); got != wantRows {
		t.Errorf("row count: got %d, want %d", got, wantRows)
	}
	if got := simplifiedParquet.NumRows(); got != wantRows {
		t.Errorf("simplified row count: got %d, want %d", got, wantRows)
	}
	t.Logf("wrote %s (%d bytes, %d rows)", outputPath, info.Size(), pf.NumRows())
}
