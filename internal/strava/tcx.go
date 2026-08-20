package strava

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
)

// ReadTCXGZ decompresses a gzip stream then parses the TCX data inside.
// Returns [lon, lat] pairs (X, Y) for all trackpoints that carry position data.
func ReadTCXGZ(r io.Reader) ([][2]float64, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	return ReadTCX(gr)
}

// ReadTCX parses a TCX file and returns [lon, lat] pairs (X, Y) for all
// trackpoints that carry a <Position> element.
func ReadTCX(r io.Reader) ([][2]float64, error) {
	points, err := readTCXTrack(r)
	if err != nil {
		return nil, err
	}
	return trackPointsToCoords(points), nil
}

func readTCXTrack(r io.Reader) ([]trackPoint, error) {
	type position struct {
		Lat float64 `xml:"LatitudeDegrees"`
		Lon float64 `xml:"LongitudeDegrees"`
	}
	type trackpoint struct {
		Position *position `xml:"Position"`
		Time     string    `xml:"Time"`
	}

	dec := xml.NewDecoder(r)
	points := make([]trackPoint, 0)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tcx decode: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Trackpoint" {
			continue
		}

		var tp trackpoint
		if err := dec.DecodeElement(&tp, &se); err != nil {
			return nil, fmt.Errorf("tcx trackpoint: %w", err)
		}
		if tp.Position == nil {
			continue
		}

		points = append(points, trackPoint{
			Lon:       tp.Position.Lon,
			Lat:       tp.Position.Lat,
			Timestamp: parseTrackTimestamp(tp.Time),
		})
	}

	return points, nil
}
