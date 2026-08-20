package strava

import (
	"math"
	"path/filepath"
	"strings"
	"time"
)

const (
	earthRadiusMeters                = 6371000.0
	pauseSplitMinDuration            = 4 * time.Minute
	pauseSplitMinDistanceMeters      = 150.0
	slowClusterMinPoints             = 3
	slowClusterMaxStepDuration       = 20 * time.Second
	slowClusterMaxStepDistanceMeters = 12.0
	slowClusterMaxRadiusMeters       = 30.0
	slowClusterMaxDuration           = 90 * time.Second
	slowClusterMaxSpeedMetersPerSec  = 1.8
	routeSimplifyEpsilonMeters       = 8.0
	consecutivePointDedupMeters      = 0.75
)

type trackPoint struct {
	Lon       float64
	Lat       float64
	Timestamp time.Time
}

// SimplifiedParquetPath returns the companion parquet path used for simplified
// route geometries.
func SimplifiedParquetPath(parquetPath string) string {
	ext := filepath.Ext(parquetPath)
	base := strings.TrimSuffix(parquetPath, ext)
	return base + "_Simplified" + ext
}

func simplifyMapData(points []trackPoint) [][][2]float64 {
	segments := splitTrackOnPauses(points)
	if len(segments) == 0 {
		return nil
	}

	parts := make([][][2]float64, 0, len(segments))
	for _, segment := range segments {
		original := dedupeConsecutivePoints(segment)
		if len(original) < 2 {
			continue
		}

		collapsed := collapseSlowClusters(original)
		collapsed = dedupeConsecutivePoints(collapsed)
		if len(collapsed) < 2 {
			collapsed = original
		}

		simplified := simplifySegmentRDP(collapsed, routeSimplifyEpsilonMeters)
		simplified = dedupeConsecutivePoints(simplified)
		if len(simplified) < 2 {
			simplified = original
		}

		coords := trackPointsToCoords(simplified)
		if len(coords) < 2 {
			continue
		}
		parts = append(parts, coords)
	}

	if len(parts) == 0 {
		return nil
	}
	return parts
}

func simplifiedGeometryWKB(points []trackPoint) []byte {
	parts := simplifyMapData(points)
	return polylineWKB(parts)
}

func polylineWKB(parts [][][2]float64) []byte {
	validParts := make([][][2]float64, 0, len(parts))
	for _, part := range parts {
		if len(part) >= 2 {
			validParts = append(validParts, part)
		}
	}

	switch len(validParts) {
	case 0:
		return nil
	case 1:
		return LineStringWKB(validParts[0])
	default:
		return MultiLineStringWKB(validParts)
	}
}

func splitTrackOnPauses(points []trackPoint) [][]trackPoint {
	points = dedupeConsecutivePoints(points)
	if len(points) < 2 {
		return nil
	}

	segments := make([][]trackPoint, 0, 1)
	start := 0
	for i := 1; i < len(points); i++ {
		if !shouldSplitTrack(points[i-1], points[i]) {
			continue
		}
		if i-start >= 2 {
			segments = append(segments, append([]trackPoint(nil), points[start:i]...))
		}
		start = i
	}

	if len(points)-start >= 2 {
		segments = append(segments, append([]trackPoint(nil), points[start:]...))
	}
	return segments
}

func shouldSplitTrack(prev, next trackPoint) bool {
	if prev.Timestamp.IsZero() || next.Timestamp.IsZero() {
		return false
	}

	delta := next.Timestamp.Sub(prev.Timestamp)
	if delta < pauseSplitMinDuration {
		return false
	}

	return pointDistanceMeters(prev, next) >= pauseSplitMinDistanceMeters
}

func collapseSlowClusters(points []trackPoint) []trackPoint {
	if len(points) < slowClusterMinPoints {
		return append([]trackPoint(nil), points...)
	}

	collapsed := make([]trackPoint, 0, len(points))
	for i := 0; i < len(points); {
		end := slowClusterEnd(points, i)
		if end-i >= slowClusterMinPoints {
			collapsed = append(collapsed, points[i], points[end-1])
			i = end
			continue
		}

		collapsed = append(collapsed, points[i])
		i++
	}

	return collapsed
}

func slowClusterEnd(points []trackPoint, start int) int {
	if start+slowClusterMinPoints > len(points) {
		return start + 1
	}
	if points[start].Timestamp.IsZero() {
		return start + 1
	}

	end := start + 1
	for end < len(points) {
		prev := points[end-1]
		curr := points[end]
		if curr.Timestamp.IsZero() {
			break
		}

		stepDuration := curr.Timestamp.Sub(prev.Timestamp)
		if stepDuration <= 0 || stepDuration > slowClusterMaxStepDuration {
			break
		}

		stepDistance := pointDistanceMeters(prev, curr)
		if stepDistance > slowClusterMaxStepDistanceMeters {
			break
		}
		if stepDistance/stepDuration.Seconds() > slowClusterMaxSpeedMetersPerSec {
			break
		}

		clusterDuration := curr.Timestamp.Sub(points[start].Timestamp)
		if clusterDuration > slowClusterMaxDuration {
			break
		}
		if pointDistanceMeters(points[start], curr) > slowClusterMaxRadiusMeters {
			break
		}

		end++
	}

	return end
}

func simplifySegmentRDP(points []trackPoint, epsilonMeters float64) []trackPoint {
	if len(points) <= 2 {
		return append([]trackPoint(nil), points...)
	}

	keep := make([]bool, len(points))
	keep[0] = true
	keep[len(points)-1] = true
	markRDPPoints(points, 0, len(points)-1, epsilonMeters, keep)

	simplified := make([]trackPoint, 0, len(points))
	for i, point := range points {
		if keep[i] {
			simplified = append(simplified, point)
		}
	}
	return simplified
}

func markRDPPoints(points []trackPoint, start, end int, epsilonMeters float64, keep []bool) {
	if end-start <= 1 {
		return
	}

	maxDistance := 0.0
	furthest := -1
	for i := start + 1; i < end; i++ {
		distance := perpendicularDistanceMeters(points[i], points[start], points[end])
		if distance > maxDistance {
			maxDistance = distance
			furthest = i
		}
	}

	if furthest == -1 || maxDistance <= epsilonMeters {
		return
	}

	keep[furthest] = true
	markRDPPoints(points, start, furthest, epsilonMeters, keep)
	markRDPPoints(points, furthest, end, epsilonMeters, keep)
}

func perpendicularDistanceMeters(point, start, end trackPoint) float64 {
	if nearlySamePoint(start, end) {
		return pointDistanceMeters(point, start)
	}

	refLatRad := ((start.Lat + end.Lat + point.Lat) / 3) * math.Pi / 180
	px, py := projectMeters(point, refLatRad)
	sx, sy := projectMeters(start, refLatRad)
	ex, ey := projectMeters(end, refLatRad)

	dx := ex - sx
	dy := ey - sy
	denominator := dx*dx + dy*dy
	if denominator == 0 {
		return math.Hypot(px-sx, py-sy)
	}

	t := ((px-sx)*dx + (py-sy)*dy) / denominator
	closestX := sx + t*dx
	closestY := sy + t*dy
	return math.Hypot(px-closestX, py-closestY)
}

func projectMeters(point trackPoint, refLatRad float64) (float64, float64) {
	lonRad := point.Lon * math.Pi / 180
	latRad := point.Lat * math.Pi / 180
	return earthRadiusMeters * lonRad * math.Cos(refLatRad), earthRadiusMeters * latRad
}

func pointDistanceMeters(a, b trackPoint) float64 {
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := lat2 - lat1
	dLon := (b.Lon - a.Lon) * math.Pi / 180

	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

func dedupeConsecutivePoints(points []trackPoint) []trackPoint {
	if len(points) < 2 {
		return append([]trackPoint(nil), points...)
	}

	deduped := make([]trackPoint, 0, len(points))
	for _, point := range points {
		if len(deduped) == 0 || pointDistanceMeters(deduped[len(deduped)-1], point) >= consecutivePointDedupMeters {
			deduped = append(deduped, point)
		}
	}
	return deduped
}

func nearlySamePoint(a, b trackPoint) bool {
	return pointDistanceMeters(a, b) < consecutivePointDedupMeters
}

func trackPointsToCoords(points []trackPoint) [][2]float64 {
	coords := make([][2]float64, 0, len(points))
	for _, point := range points {
		coords = append(coords, [2]float64{point.Lon, point.Lat})
	}
	return coords
}

func parseTrackTimestamp(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}

	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return ts.UTC()
}
