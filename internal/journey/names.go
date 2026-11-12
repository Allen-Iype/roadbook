package journey

import (
	"time"

	"roadbook/internal/domain"
)

// Naming a stop (phase 18). Three small pure steps around one database
// question, all of them outside Assemble — the place route.Apply and
// PlacePhoto take — so the golden files and the reference pins do not move:
//
//	StopPositions   which position to look up for each stop        (pure)
//	  … the store finds the nearest listed place to each position  (PostGIS)
//	NameStops       which of those answers are close enough to say (pure)
//
// A name is an inference, like a routed line, and is served with what it
// rests on: how far away the named place is, which list named it, and which
// kind of position was looked up.

// PlaceParams are the naming thresholds — named parameters, echoed with
// every journey response (invariant 3). Deliberately NOT part of Params,
// like DefaultPhotoFarWarnM: they parameterise a step that runs after
// assembly, and the golden fixtures pin assembly's parameters alone.
type PlaceParams struct {
	// NearKm: a place within this distance of the stop is said plainly —
	// "near Ísafjörður". Nothing here knows a town's outline, so "in" is
	// never said.
	NearKm float64 `json:"near_km"`
	// MaxKm: between NearKm and this, the distance is said with the name —
	// "16 km from Ísafjörður". Beyond it the stop goes unnamed: a town
	// farther off than this is not where the stop was.
	MaxKm float64 `json:"max_km"`
}

// DefaultPlaceParams are starting values, to be read against real trips at
// the phase's first review rather than measured optima. What they hold, on
// the brief's measurements against the embedded list (docs/phase-18/
// COMPARISON.md §2): 86–89 % of stops have a listed place within 10 km, and
// every measured stop has one within 25 km.
func DefaultPlaceParams() PlaceParams {
	return PlaceParams{NearKm: 10, MaxKm: 25}
}

// PositionKind says whose coordinate a stop was looked up at.
type PositionKind string

const (
	// PositionRecorded: the stop's own Loc — the mean of the points the
	// phone recorded during the halt.
	PositionRecorded PositionKind = "recorded"
	// PositionVisit: the stop has no points of its own, and the coordinate
	// comes from Google's visit record covering the same time. Google's
	// statement, not a traced position; served as such.
	PositionVisit PositionKind = "visit"
)

// StopPosition is the position a stop is named (and attributed to a region)
// from. OK is false when there is none: no points of its own and no located
// visit overlapping it.
type StopPosition struct {
	Loc  domain.LatLng
	Kind PositionKind
	OK   bool
}

// StopPositions gives, for each stop of j in order, the position to look
// up. A stop with recorded points uses its own Loc. A stop without — a halt
// between two activities during which no path point fell — borrows the
// coordinate of the located visit that overlaps it longest; ties go to the
// earlier visit, so the answer does not depend on input order.
//
// Why a visit is trusted for this: on the brief's measurements 506 of 521
// position-less stops overlap a located visit, and where a stop has both
// its own position and a visit the two sit a median 0.06–0.11 km apart
// (docs/phase-18/BRIEF.md §2.3). Why only for this: the stop's own Loc, the
// stop list and the map are untouched — whether visits belong in assembly
// is a different decision.
//
// Pure, and visits are only read (invariant 2).
func StopPositions(j Journey, visits []domain.Visit) []StopPosition {
	out := make([]StopPosition, len(j.Stops))
	for i, s := range j.Stops {
		if s.Points > 0 {
			out[i] = StopPosition{Loc: s.Loc, Kind: PositionRecorded, OK: true}
			continue
		}
		var best time.Duration
		var bestStart time.Time
		for _, v := range visits {
			if v.Loc == nil {
				continue
			}
			start, end := s.Start, s.End
			if v.Start.After(start) {
				start = v.Start
			}
			if v.End.Before(end) {
				end = v.End
			}
			ov := end.Sub(start)
			if ov <= 0 {
				continue
			}
			if ov > best || (ov == best && v.Start.Before(bestStart)) {
				best, bestStart = ov, v.Start
				out[i] = StopPosition{Loc: *v.Loc, Kind: PositionVisit, OK: true}
			}
		}
	}
	return out
}

// NearestPlace is the store's answer for one position: the nearest row of
// the places table and the distance to it. Found is false when the table is
// empty (`roadbook places` has not run) — the page then shows no names, the
// same visible degradation as regions before `roadbook states`.
type NearestPlace struct {
	Found      bool
	Name       string
	Source     string
	DistanceKm float64
}

// StopName is a name fit to print beside a stop.
type StopName struct {
	Name       string
	DistanceKm float64
	// Source is the list the name came from ("osm").
	Source string
	// Near is true within PlaceParams.NearKm: the wording is "near X".
	// False means "N km from X". Served so that every surface — page, CLI,
	// any later one — words the same stop the same way.
	Near bool
}

// NameStops keeps the answers that are close enough to say: nearest[i] is
// the store's answer for stop i (the zero value for a stop with no
// position), and the result is nil for a stop that goes unnamed — nothing
// found, or the nearest place beyond MaxKm.
func NameStops(nearest []NearestPlace, p PlaceParams) []*StopName {
	out := make([]*StopName, len(nearest))
	for i, n := range nearest {
		if !n.Found || n.DistanceKm > p.MaxKm {
			continue
		}
		out[i] = &StopName{Name: n.Name, DistanceKm: n.DistanceKm, Source: n.Source, Near: n.DistanceKm <= p.NearKm}
	}
	return out
}
