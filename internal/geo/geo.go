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
