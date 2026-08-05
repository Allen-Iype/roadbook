package journey

import (
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
)

// DefaultPhotoFarWarnM is the far-from-route threshold (phase 4 BRIEF §3G):
// an order of magnitude above the source data's positioning error (~100 m
// WiFi p90), so measurement noise cannot fire it, and well below the
// kilometres-scale divergence of a genuinely wrong route. A named parameter
// echoed with every photo-list response; deliberately NOT part of Params —
// it parameterises placement, which runs after assembly, and the golden
// fixtures pin assembly alone.
const DefaultPhotoFarWarnM = 1000.0

// PlaceKind says which element of the journey held the photo's instant —
// and therefore which drawn geometry its distance was measured against
// (invariants 5 and 8: the flag and the map can never disagree, because
// they read the same geometry).
type PlaceKind string

const (
	PlaceObserved PlaceKind = "observed" // the leg's measured points
	PlaceRoad     PlaceKind = "road"     // the leg's routed polyline
	PlaceUnknown  PlaceKind = "unknown"  // the gap's endpoint chord
	PlaceAir      PlaceKind = "air"      // excluded from flagging (§3G)
	PlaceStop     PlaceKind = "stop"     // distance to the stop's location
)

// Placement is where a journey claims to have been at a photo's instant,
// and how far the photo's own position sits from that claim.
type Placement struct {
	Kind      PlaceKind
	LegIndex  int // valid when Kind is a leg kind; -1 otherwise
	StopIndex int // valid when Kind is PlaceStop; -1 otherwise

	// DistanceM is point-to-polyline against the drawn geometry. Not
	// meaningful for PlaceAir: a photo out an aircraft window can sit
	// anywhere relative to the great-circle arc, which is presentation
	// between two endpoints, not a claimed path — HasDistance is false and
	// the photo is never flagged.
	DistanceM   float64
	HasDistance bool
	Flagged     bool // DistanceM > farWarnM
}

// PlacePhoto finds the journey element whose time span contains t and
// measures pos against its drawn geometry. Returns ok=false when the
// instant falls outside every stop and leg — the photo is then unplaced on
// the timeline, rendered as absence, never guessed.
//
// Pure: a function of the assembled (and route-applied) journey and the
// photo's two coordinates — derived at read time, never stored (BRIEF §3B),
// so re-detection that shifts a span re-places photos automatically.
//
// Stops are checked before legs: a stop's span typically lies inside a gap
// leg's span, and "the device dwelt here" is the more specific claim — the
// honest comparison for a photo taken during a halt is the halt's location,
// not the chord passing through it.
func PlacePhoto(j Journey, t time.Time, pos domain.LatLng, farWarnM float64) (Placement, bool) {
	within := func(start, end time.Time) bool {
		return !t.Before(start) && !t.After(end)
	}

	for i, s := range j.Stops {
		if within(s.Start, s.End) {
			d := geo.HaversineM(pos, s.Loc)
			return Placement{
				Kind: PlaceStop, LegIndex: -1, StopIndex: i,
				DistanceM: d, HasDistance: true, Flagged: d > farWarnM,
			}, true
		}
	}

	for i, l := range j.Legs {
		if !within(l.Start(), l.End()) {
			continue
		}
		p := Placement{LegIndex: i, StopIndex: -1}
		switch {
		case l.Kind == LegObserved:
			p.Kind = PlaceObserved
			p.DistanceM = geo.PointToPolylineM(pos, legLocs(l.Points))
			p.HasDistance = true
		case l.GapKind == GapAir:
			p.Kind = PlaceAir // shown at its position, never flagged (§3G)
		case l.GapKind == GapRoad:
			p.Kind = PlaceRoad
			p.DistanceM = geo.PointToPolylineM(pos, l.RoutedPoints)
			p.HasDistance = true
		default:
			p.Kind = PlaceUnknown
			p.DistanceM = geo.PointToPolylineM(pos, legLocs(l.Points))
			p.HasDistance = true
		}
		p.Flagged = p.HasDistance && p.DistanceM > farWarnM
		return p, true
	}

	return Placement{LegIndex: -1, StopIndex: -1}, false
}

func legLocs(pts []TimedPoint) []domain.LatLng {
	out := make([]domain.LatLng, len(pts))
	for i, pt := range pts {
		out[i] = pt.Loc
	}
	return out
}
