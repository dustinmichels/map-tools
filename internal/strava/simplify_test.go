package strava

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"
)

func TestSimplifiedParquetPathUsesDoubleUnderscoreLowercaseSuffix(t *testing.T) {
	got := SimplifiedParquetPath(filepath.Join("tmp", "rides.parquet"))
	want := filepath.Join("tmp", "rides__simplified.parquet")
	if got != want {
		t.Fatalf("expected simplified parquet path %q, got %q", want, got)
	}
}

func TestSimplifyMapDataSplitsPauseAndCollapsesSlowCluster(t *testing.T) {
	base := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
	points := []trackPoint{
		{Lon: -71.1000, Lat: 42.3600, Timestamp: base},
		{Lon: -71.0996, Lat: 42.3600, Timestamp: base.Add(10 * time.Second)},
		{Lon: -71.09958, Lat: 42.36001, Timestamp: base.Add(20 * time.Second)},
		{Lon: -71.09957, Lat: 42.36002, Timestamp: base.Add(30 * time.Second)},
		{Lon: -71.09956, Lat: 42.36001, Timestamp: base.Add(40 * time.Second)},
		{Lon: -71.0992, Lat: 42.3600, Timestamp: base.Add(50 * time.Second)},
		{Lon: -71.0950, Lat: 42.3600, Timestamp: base.Add(7 * time.Minute)},
		{Lon: -71.0946, Lat: 42.3601, Timestamp: base.Add(7*time.Minute + 10*time.Second)},
	}

	parts := simplifyMapData(points)
	if len(parts) != 2 {
		t.Fatalf("expected 2 line parts, got %d", len(parts))
	}

	totalPoints := 0
	for i, part := range parts {
		if len(part) < 2 {
			t.Fatalf("part %d has %d points, want at least 2", i, len(part))
		}
		totalPoints += len(part)
	}

	if totalPoints >= len(points) {
		t.Fatalf("expected simplification to reduce points, got %d from %d", totalPoints, len(points))
	}
}

func TestProcessActivitiesWritesSimplifiedParquet(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "rides.parquet")
	base := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
	filename := "ride.fit"

	activities := []Activity{{
		ActivityID:   42,
		ActivityDate: "2026-08-19",
		ActivityName: "Commute",
		ActivityType: "Ride",
		Filename:     &filename,
	}}

	track := []trackPoint{
		{Lon: -71.1000, Lat: 42.3600, Timestamp: base},
		{Lon: -71.0996, Lat: 42.3600, Timestamp: base.Add(10 * time.Second)},
		{Lon: -71.09958, Lat: 42.36001, Timestamp: base.Add(20 * time.Second)},
		{Lon: -71.09957, Lat: 42.36002, Timestamp: base.Add(30 * time.Second)},
		{Lon: -71.09956, Lat: 42.36001, Timestamp: base.Add(40 * time.Second)},
		{Lon: -71.0992, Lat: 42.3600, Timestamp: base.Add(50 * time.Second)},
		{Lon: -71.0950, Lat: 42.3600, Timestamp: base.Add(7 * time.Minute)},
		{Lon: -71.0946, Lat: 42.3601, Timestamp: base.Add(7*time.Minute + 10*time.Second)},
	}

	result, err := processActivities(activities, func(string) ([]trackPoint, error) {
		return track, nil
	}, outPath)
	if err != nil {
		t.Fatalf("processActivities: %v", err)
	}
	if result.Parsed != 1 {
		t.Fatalf("expected 1 parsed activity, got %d", result.Parsed)
	}

	simplifiedPath := SimplifiedParquetPath(outPath)
	for _, path := range []string{outPath, simplifiedPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected parquet output %q: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("expected parquet output %q to be non-empty", path)
		}
	}

	originalRow := readSingleActivityRow(t, outPath)
	simplifiedRow := readSingleActivityRow(t, simplifiedPath)
	if len(simplifiedRow.Geometry) >= len(originalRow.Geometry) {
		t.Fatalf("expected simplified geometry to shrink, got original=%d simplified=%d", len(originalRow.Geometry), len(simplifiedRow.Geometry))
	}
}

func readSingleActivityRow(t *testing.T, path string) ActivityRow {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open parquet %q: %v", path, err)
	}
	defer f.Close()

	reader := parquet.NewGenericReader[ActivityRow](f)
	defer reader.Close()

	rows := make([]ActivityRow, 1)
	n, err := reader.Read(rows)
	if err != nil && err != io.EOF {
		t.Fatalf("read parquet %q: %v", path, err)
	}
	if n != 1 {
		t.Fatalf("expected one row in %q, got %d", path, n)
	}

	return rows[0]
}
