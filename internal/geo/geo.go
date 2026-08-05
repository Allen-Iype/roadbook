// Package geo provides the small amount of spherical geometry the pipeline needs.
package geo

import (
	"math"

	"roadbook/internal/domain"
)

// meanEarthRadiusM matches the reference detector exactly; changing it changes
// every distance and therefore detection output.
const meanEarthRadiusM = 6371008.8

// HaversineM returns the great-circle distance between a and b in metres.
func HaversineM(a, b domain.LatLng) float64 {
	p1 := a.Lat * math.Pi / 180
	p2 := b.Lat * math.Pi / 180
	dp := p2 - p1
	dl := (b.Lon - a.Lon) * math.Pi / 180
	h := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * meanEarthRadiusM * math.Asin(math.Sqrt(h))
}

// PointToPolylineM returns the distance in metres from p to the nearest point
// on the polyline, which may be a single point (distance to it) — never an
// empty slice by contract.
//
// Segments are measured in a local equirectangular frame centred on p:
// exact enough for the kilometre scales the far-from-route flag cares about
// (the error is second-order in span/Earth-radius), and honest about what it
// is — a flag threshold, not a survey.
func PointToPolylineM(p domain.LatLng, line []domain.LatLng) float64 {
	if len(line) == 1 {
		return HaversineM(p, line[0])
	}
	min := math.Inf(1)
	for i := 0; i+1 < len(line); i++ {
		if d := pointToSegmentM(p, line[i], line[i+1]); d < min {
			min = d
		}
	}
	return min
}

func pointToSegmentM(p, a, b domain.LatLng) float64 {
	// Project into metres around p; p sits at the origin.
	const rad = math.Pi / 180
	cos := math.Cos(p.Lat * rad)
	ax := (a.Lon - p.Lon) * rad * cos * meanEarthRadiusM
	ay := (a.Lat - p.Lat) * rad * meanEarthRadiusM
	bx := (b.Lon - p.Lon) * rad * cos * meanEarthRadiusM
	by := (b.Lat - p.Lat) * rad * meanEarthRadiusM

	dx, dy := bx-ax, by-ay
	lenSq := dx*dx + dy*dy
	if lenSq == 0 {
		return math.Hypot(ax, ay) // degenerate segment: a point
	}
	// t is the projection of the origin onto the segment, clamped to it.
	t := -(ax*dx + ay*dy) / lenSq
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(ax+t*dx, ay+t*dy)
}
