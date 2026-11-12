package suggest

import (
	"context"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
)

// PlaceLookup is the one question Places asks: the nearest listed place to
// each point. *store.Store answers it (PlacesForPoints); the interface keeps
// this package free of the database, and lets the test answer from memory.
type PlaceLookup interface {
	PlacesForPoints(ctx context.Context, pts []domain.LatLng) ([]journey.NearestPlace, error)
}

// SourcePlaces is the source reported when the places table had nothing to
// offer; a suggestion that does carry a name reports the list the name came
// from ("osm").
const SourcePlaces = "places"

// Places suggests the nearest listed town to the destination, from the
// places table (phase 18) — the same rows and the same distance limit that
// name a stop, so a trip is never suggested a name its own page would not
// print beside the stop. It is offline: the table is loaded from the list
// embedded in the binary, and nothing here reaches the network. That is why
// it can be the default where Nominatim never could.
//
// The destination is where the traveller dwelt farthest from home
// (detection's rule), so its nearest town is the natural short name for the
// trip: "Munnar", "Ísafjörður". Beyond MaxKm — or with the table not loaded
// — it suggests nothing, and says so, exactly like Null.
type Places struct {
	Lookup PlaceLookup
	// MaxKm is journey.PlaceParams.MaxKm: one limit for names everywhere.
	MaxKm float64
}

// NewPlaces builds the default suggester with the naming limit stops use.
func NewPlaces(lookup PlaceLookup) Places {
	return Places{Lookup: lookup, MaxKm: journey.DefaultPlaceParams().MaxKm}
}

func (p Places) Suggest(ctx context.Context, dest domain.LatLng) (Suggestion, error) {
	found, err := p.Lookup.PlacesForPoints(ctx, []domain.LatLng{dest})
	if err != nil {
		return Suggestion{}, err
	}
	if len(found) == 0 || !found[0].Found || found[0].DistanceKm > p.MaxKm {
		return Suggestion{Source: SourcePlaces}, nil
	}
	return Suggestion{Name: found[0].Name, Source: found[0].Source}, nil
}
