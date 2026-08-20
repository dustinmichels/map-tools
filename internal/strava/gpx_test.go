package strava_test

import (
	"math"
	"os"
	"testing"

	"github.com/dustinmichels/map-tools/internal/strava"
)

// sampleGPX is a real .gpx file from the test export.
const sampleGPX = "../../data/strava_export/activities/9977767493.gpx"

func TestReadGPX(t *testing.T) {
	f, err := os.Open(sampleGPX)
	if os.IsNotExist(err) {
		t.Skipf("sample GPX not found: %s", sampleGPX)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	coords, err := strava.ReadGPX(f)
	if err != nil {
		t.Fatalf("ReadGPX: %v", err)
	}
	if len(coords) == 0 {
		t.Fatal("ReadGPX returned no coordinates")
	}

	// First trkpt in the file: lat="42.3883080" lon="-71.0919540"
	// Coords are [lon, lat].
	wantLon, wantLat := -71.091954, 42.388308
	got := coords[0]
	if math.Abs(got[0]-wantLon) > 1e-5 || math.Abs(got[1]-wantLat) > 1e-5 {
		t.Errorf("first coord: got [%.6f, %.6f], want [%.6f, %.6f]",
			got[0], got[1], wantLon, wantLat)
	}

	t.Logf("parsed %d track points", len(coords))
}
