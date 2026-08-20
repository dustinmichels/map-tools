package strava

import (
	"compress/gzip"
	"fmt"
	"io"
	"math"

	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

// ReadFIT parses a raw (uncompressed) FIT file and returns [lon, lat] pairs.
func ReadFIT(r io.Reader) ([][2]float64, error) {
	points, err := readFITTrack(r)
	if err != nil {
		return nil, err
	}
	return trackPointsToCoords(points), nil
}

// ReadFITGZ decompresses a gzip stream then parses the FIT data inside.
// Returns [lon, lat] pairs (X, Y) for all valid GPS records.
func ReadFITGZ(r io.Reader) ([][2]float64, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	points, err := readFITTrack(gr)
	if err != nil {
		return nil, err
	}
	return trackPointsToCoords(points), nil
}

func readFITTrack(r io.Reader) ([]trackPoint, error) {
	dec := decoder.New(r)

	fit, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	points := make([]trackPoint, 0)
	for i := range fit.Messages {
		msg := &fit.Messages[i]
		if msg.Num != typedef.MesgNumRecord {
			continue
		}
		rec := mesgdef.NewRecord(msg)
		lat := rec.PositionLatDegrees()
		lon := rec.PositionLongDegrees()
		if math.IsNaN(lat) || math.IsNaN(lon) {
			continue
		}

		timestamp := rec.Timestamp
		if !timestamp.IsZero() {
			timestamp = timestamp.UTC()
		}
		points = append(points, trackPoint{Lon: lon, Lat: lat, Timestamp: timestamp})
	}
	return points, nil
}
