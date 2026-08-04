// Package route is the routing seam (CLAUDE.md invariant 7, phase 3 BRIEF
// §1.1): one narrow interface, a null implementation as the default, an OSRM
// client as the network implementation. Routing is batch-only — the serve
// binary never dials a router; it reads the route_cache table that
// `roadbook route` fills, and Apply decorates an assembled journey from
// those cached answers. Only unknown gaps are ever routed: observed legs are
// measurement (invariant 6) and air legs are flights.
package route

import (
	"context"
	"errors"
	"math"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
	"roadbook/internal/journey"
)

// ErrNoRoute is a data answer, not a failure: the road network cannot
// connect the two points (patchy OSM coverage — the designed, visible case).
// It is cached so re-runs don't re-ask. Operational errors (network down,
// timeout) are anything else and are never cached.
var ErrNoRoute = errors.New("no route between these points")

// Route is a road answer in domain terms — nothing OSRM-shaped escapes the
// client (invariant 4's discipline applied to this seam).
type Route struct {
	Points    []domain.LatLng
	DistanceM float64
	DurationS float64
}

// Router answers one question: what road connects these two points? Batch
// orchestration, caching, and classification are deliberately not its job.
type Router interface {
	Route(ctx context.Context, from, to domain.LatLng) (Route, error)
}

// Null fills nothing: every gap stays unknown and renders as the dashed
// straight line phase 2 already draws. Invariant 7's "draws straight lines
// and says so" is satisfied by the product's degraded rendering — a null
// router that fabricated `road` results would label uninspected geometry as
// inference that never ran (BRIEF §3B).
type Null struct{}

func (Null) Route(context.Context, domain.LatLng, domain.LatLng) (Route, error) {
	return Route{}, ErrNoRoute
}

// Key identifies one cached routing request: the endpoint pair rounded to
// 4 decimals (~11 m — below the source data's accuracy) plus the profile.
// Fixed-point integers, not floats: the key must match by equality, and
// 4-decimal values are not exactly representable in binary floating point.
// The rounded coordinate is itself what gets routed, so the key IS the
// request and a hit is exact, never approximate (BRIEF §3C). Directional —
// one-way roads exist.
type Key struct {
	FromLatE4 int32
	FromLonE4 int32
	ToLatE4   int32
	ToLonE4   int32
	Profile   string
}

func e4(deg float64) int32 { return int32(math.Round(deg * 1e4)) }

func fromE4(v int32) float64 { return float64(v) / 1e4 }

func KeyFor(from, to domain.LatLng, profile string) Key {
	return Key{
		FromLatE4: e4(from.Lat), FromLonE4: e4(from.Lon),
		ToLatE4: e4(to.Lat), ToLonE4: e4(to.Lon),
		Profile: profile,
	}
}

// From and To are the rounded coordinates — the exact request a Router is
// asked and the cache stores.
func (k Key) From() domain.LatLng {
	return domain.LatLng{Lat: fromE4(k.FromLatE4), Lon: fromE4(k.FromLonE4)}
}

func (k Key) To() domain.LatLng {
	return domain.LatLng{Lat: fromE4(k.ToLatE4), Lon: fromE4(k.ToLonE4)}
}

// Cache statuses. A row exists only for questions a router actually
// answered; StatusNoRoute is the remembered negative.
const (
	StatusRouted  = "routed"
	StatusNoRoute = "no_route"
)

// Cached is one route_cache row in domain terms.
type Cached struct {
	Status    string
	Points    []domain.LatLng
	DistanceM float64
	DurationS float64
}

// UnknownKeys collects the cache keys for a journey's unroutable gaps —
// unknown gaps only: air is a flight and observed is measurement. Deduped,
// in first-appearance order.
func UnknownKeys(j journey.Journey, profile string) []Key {
	seen := map[Key]bool{}
	var keys []Key
	for _, l := range j.Legs {
		if l.Kind != journey.LegGap || l.GapKind != journey.GapUnknown {
			continue
		}
		k := KeyFor(l.Points[0].Loc, l.Points[1].Loc, profile)
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	return keys
}

// Apply decorates an assembled journey from cached answers: an unknown gap
// whose key has a routed cache entry becomes a road leg carrying the routed
// geometry and distance. Everything else — observed legs, air legs, unknown
// gaps without an answer — passes through untouched, and the leg still
// carries exactly its two timestamped endpoints either way (the routed
// geometry rides alongside, in RoutedPoints). Pure: a function of its
// arguments, no I/O — the lookup map is the only bridge to the cache, so
// this is testable with a literal. The input journey is not mutated; legs
// are copied before decoration.
func Apply(j journey.Journey, profile string, lookup map[Key]Cached) journey.Journey {
	legs := make([]journey.Leg, len(j.Legs))
	copy(legs, j.Legs)
	j.Legs = legs

	j.RoutedKm = 0
	j.UnknownKm = 0
	for i, l := range j.Legs {
		if l.Kind != journey.LegGap || l.GapKind != journey.GapUnknown {
			continue
		}
		c, ok := lookup[KeyFor(l.Points[0].Loc, l.Points[1].Loc, profile)]
		if !ok || c.Status != StatusRouted {
			j.UnknownKm += l.DistanceKm
			continue
		}
		l.GapKind = journey.GapRoad
		l.RoutedPoints = c.Points
		l.RoutedKm = c.DistanceM / 1000
		j.Legs[i] = l
		j.RoutedKm += l.RoutedKm
	}
	return j
}

// ChordKm is the great-circle distance between a key's two endpoints — the
// figure a routed distance replaces, kept for reporting.
func ChordKm(k Key) float64 {
	return geo.HaversineM(k.From(), k.To()) / 1000
}
