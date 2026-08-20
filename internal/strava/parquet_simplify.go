package strava

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/parquet-go/parquet-go"
)

const (
	wkbLittleEndian    = 1
	wkbLineString      = 2
	wkbMultiLineString = 5
)

// SimplifyParquet rewrites a geoparquet file with simplified route geometries.
func SimplifyParquet(inPath, outPath string) error {
	input, err := os.Open(inPath)
	if err != nil {
		return fmt.Errorf("open %q: %w", inPath, err)
	}
	defer input.Close()

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output dir for %q: %w", outPath, err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(outPath), filepath.Base(outPath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp parquet for %q: %w", outPath, err)
	}

	tmpPath := tmpFile.Name()
	removeTmp := true
	defer func() {
		_ = tmpFile.Close()
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	reader := parquet.NewGenericReader[ActivityRow](input)
	defer reader.Close()

	writer := parquet.NewGenericWriter[ActivityRow](tmpFile,
		parquet.KeyValueMetadata("geo", simplifiedGeoMetadata),
	)

	rows := make([]ActivityRow, 256)
	for {
		n, readErr := reader.Read(rows)
		if readErr != nil && readErr != io.EOF {
			_ = writer.Close()
			return fmt.Errorf("read %q: %w", inPath, readErr)
		}

		batch := rows[:n]
		for i := range batch {
			simplifiedGeometry, err := simplifyStoredGeometryWKB(batch[i].Geometry)
			if err != nil {
				_ = writer.Close()
				return fmt.Errorf("simplify activity %d geometry: %w", batch[i].ActivityID, err)
			}
			batch[i].Geometry = simplifiedGeometry
		}

		if len(batch) > 0 {
			if _, err := writer.Write(batch); err != nil {
				_ = writer.Close()
				return fmt.Errorf("write %q: %w", outPath, err)
			}
		}

		if readErr == io.EOF {
			break
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close writer for %q: %w", outPath, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		return fmt.Errorf("rename %q to %q: %w", tmpPath, outPath, err)
	}

	removeTmp = false
	return nil
}

func simplifyStoredGeometryWKB(geometry []byte) ([]byte, error) {
	if len(geometry) == 0 {
		return nil, nil
	}

	parts, err := decodePolylineWKB(geometry)
	if err != nil {
		return nil, err
	}

	simplifiedParts := make([][][2]float64, 0, len(parts))
	for _, part := range parts {
		simplifiedPart := simplifyStoredCoords(part)
		if len(simplifiedPart) >= 2 {
			simplifiedParts = append(simplifiedParts, simplifiedPart)
		}
	}
	if len(simplifiedParts) == 0 {
		return geometry, nil
	}

	simplifiedGeometry := polylineWKB(simplifiedParts)
	if len(simplifiedGeometry) == 0 {
		return geometry, nil
	}
	return simplifiedGeometry, nil
}

func simplifyStoredCoords(coords [][2]float64) [][2]float64 {
	if len(coords) < 2 {
		return coords
	}

	points := make([]trackPoint, 0, len(coords))
	for _, coord := range coords {
		points = append(points, trackPoint{Lon: coord[0], Lat: coord[1]})
	}

	deduped := dedupeConsecutivePoints(points)
	if len(deduped) < 2 {
		return coords
	}

	simplified := simplifySegmentRDP(deduped, routeSimplifyEpsilonMeters)
	simplified = dedupeConsecutivePoints(simplified)
	if len(simplified) < 2 {
		simplified = deduped
	}
	return trackPointsToCoords(simplified)
}

func decodePolylineWKB(data []byte) ([][][2]float64, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("wkb too short: %d bytes", len(data))
	}
	if data[0] != wkbLittleEndian {
		return nil, fmt.Errorf("unsupported wkb byte order %d", data[0])
	}

	switch binary.LittleEndian.Uint32(data[1:5]) {
	case wkbLineString:
		coords, _, err := decodeLineStringWKBAt(data, 0)
		if err != nil {
			return nil, err
		}
		return [][][2]float64{coords}, nil
	case wkbMultiLineString:
		return decodeMultiLineStringWKB(data)
	default:
		return nil, fmt.Errorf("unsupported wkb geometry type %d", binary.LittleEndian.Uint32(data[1:5]))
	}
}

func decodeMultiLineStringWKB(data []byte) ([][][2]float64, error) {
	if len(data) < 9 {
		return nil, fmt.Errorf("multilinestring wkb too short: %d bytes", len(data))
	}

	partCount := int(binary.LittleEndian.Uint32(data[5:9]))
	parts := make([][][2]float64, 0, partCount)
	offset := 9
	for i := range partCount {
		coords, nextOffset, err := decodeLineStringWKBAt(data, offset)
		if err != nil {
			return nil, fmt.Errorf("decode multiline part %d: %w", i, err)
		}
		parts = append(parts, coords)
		offset = nextOffset
	}
	if offset != len(data) {
		return nil, fmt.Errorf("unexpected trailing bytes in multiline wkb: %d", len(data)-offset)
	}
	return parts, nil
}

func decodeLineStringWKBAt(data []byte, offset int) ([][2]float64, int, error) {
	if offset < 0 || offset+9 > len(data) {
		return nil, 0, fmt.Errorf("linestring header out of range at offset %d", offset)
	}
	if data[offset] != wkbLittleEndian {
		return nil, 0, fmt.Errorf("unsupported nested wkb byte order %d", data[offset])
	}
	if geometryType := binary.LittleEndian.Uint32(data[offset+1 : offset+5]); geometryType != wkbLineString {
		return nil, 0, fmt.Errorf("unsupported nested wkb geometry type %d", geometryType)
	}

	pointCount := int(binary.LittleEndian.Uint32(data[offset+5 : offset+9]))
	end := offset + 9 + pointCount*16
	if end > len(data) {
		return nil, 0, fmt.Errorf("linestring payload out of range: need %d bytes, have %d", end-offset, len(data)-offset)
	}

	coords := make([][2]float64, 0, pointCount)
	cursor := offset + 9
	for range pointCount {
		lon := math.Float64frombits(binary.LittleEndian.Uint64(data[cursor : cursor+8]))
		lat := math.Float64frombits(binary.LittleEndian.Uint64(data[cursor+8 : cursor+16]))
		coords = append(coords, [2]float64{lon, lat})
		cursor += 16
	}
	return coords, end, nil
}
