package api

import (
	"context"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/store"
)

// StopNames runs the naming pipeline for one journey (phase 18): the pure
// choice of a position per stop, one nearest-place query for all of them,
// and the pure rule for which answers are close enough to say. Exported and
// kept apart from the handler because `roadbook journey -candidate` calls
// the very same function — the page and the command that reproduces it
// cannot name a stop differently (invariant 13).
//
// Both results are index-aligned with j.Stops. names[i] is nil for a stop
// that goes unnamed: no position to look up, nothing within the limit, or
// an unloaded places table.
func StopNames(ctx context.Context, st *store.Store, j journey.Journey, visits []domain.Visit, p journey.PlaceParams) ([]journey.StopPosition, []*journey.StopName, error) {
	positions := journey.StopPositions(j, visits)

	// Only stops that have a position go to the database; idx remembers
	// which stop each query point belongs to.
	var pts []domain.LatLng
	var idx []int
	for i, pos := range positions {
		if pos.OK {
			pts = append(pts, pos.Loc)
			idx = append(idx, i)
		}
	}
	found, err := st.PlacesForPoints(ctx, pts)
	if err != nil {
		return nil, nil, err
	}
	nearest := make([]journey.NearestPlace, len(positions))
	for k, i := range idx {
		nearest[i] = found[k]
	}
	return positions, journey.NameStops(nearest, p), nil
}

// Regions attributes each leg and each stop of a journey to admin-1 regions
// (phase 18, #17) — what lets a day say "Vesturland → Vestfirðir" where the
// cover could only list the whole trip's regions. Like StopNames it is
// shared by the handler and `roadbook journey -candidate`.
//
// legs[i] is the regions leg i's points fall in, each once, in order of
// first appearance; stops[i] is the region of the position stop i was
// looked up at (positions comes from StopNames, so a stop with no points of
// its own is attributed at its visit position, the same one that named it),
// or nil. All of it is empty when the states table is not loaded.
//
// Only the leg's own timestamped points are asked — for a gap, its two
// endpoints — never routed road geometry: inference does not testify to
// where a person was (the rule the trip-level list already follows).
func Regions(ctx context.Context, st *store.Store, j journey.Journey, positions []journey.StopPosition) (legs [][]store.StateRef, stops []*store.StateRef, err error) {
	// Every leg point, then every stop position, in one query.
	var pts []domain.LatLng
	for _, l := range j.Legs {
		for _, p := range l.Points {
			pts = append(pts, p.Loc)
		}
	}
	nLeg := len(pts)
	var stopIdx []int
	for i, pos := range positions {
		if pos.OK {
			pts = append(pts, pos.Loc)
			stopIdx = append(stopIdx, i)
		}
	}
	each, err := st.StateForEachPoint(ctx, pts)
	if err != nil {
		return nil, nil, err
	}

	legs = make([][]store.StateRef, len(j.Legs))
	k := 0
	for i, l := range j.Legs {
		seen := map[string]bool{}
		for range l.Points {
			if r := each[k]; r != nil && !seen[r.Code] {
				seen[r.Code] = true
				legs[i] = append(legs[i], *r)
			}
			k++
		}
	}
	stops = make([]*store.StateRef, len(positions))
	for n, i := range stopIdx {
		stops[i] = each[nLeg+n]
	}
	return legs, stops, nil
}
