package strava

import (
	"encoding/binary"
	"math"
)

// LineStringWKB encodes coords as a WKB LineString (ISO, little-endian).
// coords are [lon, lat] pairs (X, Y order per GeoParquet spec).
// Returns nil if fewer than 2 points.
func LineStringWKB(coords [][2]float64) []byte {
	if len(coords) < 2 {
		return nil
	}
	buf := make([]byte, 9+len(coords)*16)
	buf[0] = 1                                 // byte order: little-endian
	binary.LittleEndian.PutUint32(buf[1:5], 2) // geometry type: LineString
	binary.LittleEndian.PutUint32(buf[5:9], uint32(len(coords)))
	for i, c := range coords {
		off := 9 + i*16
		binary.LittleEndian.PutUint64(buf[off:off+8], math.Float64bits(c[0]))    // X = lon
		binary.LittleEndian.PutUint64(buf[off+8:off+16], math.Float64bits(c[1])) // Y = lat
	}
	return buf
}

// MultiLineStringWKB encodes multiple line parts as a WKB MultiLineString.
// Returns nil unless at least two valid line parts are present.
func MultiLineStringWKB(parts [][][2]float64) []byte {
	validParts := make([][][2]float64, 0, len(parts))
	totalBytes := 9
	for _, part := range parts {
		if len(part) < 2 {
			continue
		}
		validParts = append(validParts, part)
		totalBytes += 9 + len(part)*16
	}
	if len(validParts) < 2 {
		return nil
	}

	buf := make([]byte, totalBytes)
	buf[0] = 1                                 // byte order: little-endian
	binary.LittleEndian.PutUint32(buf[1:5], 5) // geometry type: MultiLineString
	binary.LittleEndian.PutUint32(buf[5:9], uint32(len(validParts)))

	offset := 9
	for _, part := range validParts {
		buf[offset] = 1
		binary.LittleEndian.PutUint32(buf[offset+1:offset+5], 2)
		binary.LittleEndian.PutUint32(buf[offset+5:offset+9], uint32(len(part)))
		offset += 9
		for _, coord := range part {
			binary.LittleEndian.PutUint64(buf[offset:offset+8], math.Float64bits(coord[0]))
			binary.LittleEndian.PutUint64(buf[offset+8:offset+16], math.Float64bits(coord[1]))
			offset += 16
		}
	}

	return buf
}
