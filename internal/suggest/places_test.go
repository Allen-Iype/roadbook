package suggest

import (
	"context"
	"errors"
	"testing"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
)

type fakeLookup struct {
	answer journey.NearestPlace
	err    error
	asked  []domain.LatLng
}

func (f *fakeLookup) PlacesForPoints(_ context.Context, pts []domain.LatLng) ([]journey.NearestPlace, error) {
	f.asked = append(f.asked, pts...)
	if f.err != nil {
		return nil, f.err
	}
	out := make([]journey.NearestPlace, len(pts))
	for i := range out {
		out[i] = f.answer
	}
	return out, nil
}

func TestPlacesSuggest(t *testing.T) {
	ctx := context.Background()
	dest := domain.LatLng{Lat: 10.09, Lon: 77.06}

	cases := []struct {
		name   string
		answer journey.NearestPlace
		want   Suggestion
	}{
		{"a town beside the destination", journey.NearestPlace{Found: true, Name: "Munnar", Source: "osm", DistanceKm: 0.4}, Suggestion{Name: "Munnar", Source: "osm"}},
		{"a town at the limit is still the name", journey.NearestPlace{Found: true, Name: "Munnar", Source: "osm", DistanceKm: 25}, Suggestion{Name: "Munnar", Source: "osm"}},
		{"a town past the limit is not", journey.NearestPlace{Found: true, Name: "Munnar", Source: "osm", DistanceKm: 25.1}, Suggestion{Source: SourcePlaces}},
		{"an unloaded table suggests nothing", journey.NearestPlace{}, Suggestion{Source: SourcePlaces}},
	}
	for _, c := range cases {
		f := &fakeLookup{answer: c.answer}
		got, err := NewPlaces(f).Suggest(ctx, dest)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
		if len(f.asked) != 1 || f.asked[0] != dest {
			t.Errorf("%s: asked %v, want exactly the destination", c.name, f.asked)
		}
	}

	// A database failure is an error the handler reports, not an empty name.
	if _, err := NewPlaces(&fakeLookup{err: errors.New("boom")}).Suggest(ctx, dest); err == nil {
		t.Error("lookup error was swallowed")
	}
	if NewPlaces(&fakeLookup{}).MaxKm != journey.DefaultPlaceParams().MaxKm {
		t.Error("the suggester's limit is not the stops' limit")
	}
}
