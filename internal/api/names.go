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
