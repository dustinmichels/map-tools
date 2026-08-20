package strava

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// ReadGPXGZ decompresses a gzip stream then parses the GPX data inside.
// Returns [lon, lat] pairs (X, Y) for all track points.
func ReadGPXGZ(r io.Reader) ([][2]float64, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	return ReadGPX(gr)
}

// ReadGPX parses a GPX file and returns [lon, lat] pairs (X, Y) for all
// track points across all tracks and segments.
func ReadGPX(r io.Reader) ([][2]float64, error) {
	points, err := readGPXTrack(r)
	if err != nil {
		return nil, err
	}
	return trackPointsToCoords(points), nil
}

func readGPXTrack(r io.Reader) ([]trackPoint, error) {
	type trackData struct {
		Time string `xml:"time"`
	}

	dec := xml.NewDecoder(r)
	points := make([]trackPoint, 0)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gpx decode: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "trkpt" {
			continue
		}

		var lat, lon float64
		var hasLat, hasLon bool
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "lat":
				if v, err := strconv.ParseFloat(attr.Value, 64); err == nil {
					lat = v
					hasLat = true
				}
			case "lon":
				if v, err := strconv.ParseFloat(attr.Value, 64); err == nil {
					lon = v
					hasLon = true
				}
			}
		}

		var data trackData
		if err := dec.DecodeElement(&data, &se); err != nil {
			return nil, fmt.Errorf("gpx track point: %w", err)
		}
		if hasLat && hasLon {
			points = append(points, trackPoint{
				Lon:       lon,
				Lat:       lat,
				Timestamp: parseTrackTimestamp(data.Time),
			})
		}
	}

	return points, nil
}
