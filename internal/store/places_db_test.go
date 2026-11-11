package store_test

import (
	"context"
	"testing"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
	"roadbook/internal/journey"
	"roadbook/internal/places"
	"roadbook/internal/store/storetest"
)

// The -if-empty startup path rests on the same two store facts as countries
// and states: a fresh schema counts zero, and a loaded table counts exactly
// what was loaded — twice over, because ReplacePlaces replaces. This loads
// the WHOLE embedded list, unlike the states test's Iceland subset: the
// single-statement load is the thing under test, and its time is printed
// because every fresh instance pays it once at first start.
func TestReplaceAndCountPlaces(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	n, err := s.CountPlaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("fresh schema: want 0 places, got %d", n)
	}

	list, err := places.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := s.ReplacePlaces(ctx, list, places.SourceOSM); err != nil {
		t.Fatal(err)
	}
	t.Logf("loaded %d places in %s", len(list), time.Since(start).Round(time.Millisecond))
	if err := s.ReplacePlaces(ctx, list, places.SourceOSM); err != nil { // replace, not append
		t.Fatal(err)
	}

	n, err = s.CountPlaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(list)) {
		t.Fatalf("after double load: want %d places, got %d", len(list), n)
	}

	// A smaller list replaces the larger one outright.
	if err := s.ReplacePlaces(ctx, list[:3], "test"); err != nil {
		t.Fatal(err)
	}
	if n, _ = s.CountPlaces(ctx); n != 3 {
		t.Fatalf("after replacing with 3: got %d", n)
	}
}

// The nearest-place query, against the whole embedded list. The cases are
// public geography, not anyone's data: a point beside a town, a point in
// mid-ocean, a town's own node, the high-latitude case the geography column
// exists for, and the order of answers.
func TestPlacesForPoints(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	pts := []domain.LatLng{
		{Lat: 66.07, Lon: -23.10},   // beside Ísafjörður
		{Lat: 10.09, Lon: 77.06},    // beside Munnar
		{Lat: -48.0, Lon: -123.0},   // the South Pacific, far from any town
		{Lat: 64.146, Lon: -21.942}, // Reykjavík's own node, to the rounding
	}

	// Unloaded table: an answer per point, none found, no error.
	got, err := s.PlacesForPoints(ctx, pts)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(pts) {
		t.Fatalf("unloaded: %d answers for %d points", len(got), len(pts))
	}
	for i, n := range got {
		if n.Found {
			t.Fatalf("unloaded table named point %d: %+v", i, n)
		}
	}
	if got, err := s.PlacesForPoints(ctx, nil); err != nil || len(got) != 0 {
		t.Fatalf("no points: %v, %d answers", err, len(got))
	}

	list, err := places.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplacePlaces(ctx, list, places.SourceOSM); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	got, err = s.PlacesForPoints(ctx, pts)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d lookups in %s", len(pts), time.Since(start).Round(time.Microsecond))

	for i, w := range []struct {
		name  string
		maxKm float64
	}{
		{"Ísafjörður", 2},
		{"Munnar", 2},
		{"", 0}, // checked below: found, but far
		{"Reykjavík", 0.1},
	} {
		n := got[i]
		if !n.Found || n.Source != places.SourceOSM {
			t.Fatalf("point %d: %+v", i, n)
		}
		if w.name == "" {
			continue
		}
		if n.Name != w.name || n.DistanceKm > w.maxKm {
			t.Errorf("point %d: got %q at %.2f km, want %q within %.1f km", i, n.Name, n.DistanceKm, w.name, w.maxKm)
		}
	}
	// Mid-ocean: the table always has a nearest row; it is the naming rule
	// (journey.NameStops, MaxKm) that declines to print it.
	if got[2].DistanceKm < 1000 {
		t.Errorf("mid-ocean point is %.0f km from %q — expected over 1,000 km", got[2].DistanceKm, got[2].Name)
	}
	if named := journey.NameStops(got, journey.DefaultPlaceParams()); named[2] != nil || named[0] == nil {
		t.Errorf("naming rule: ocean %v, town %v", named[2], named[0])
	}

	// Nearest on the globe, not in degrees: a degree of longitude is short
	// at 66° north, so a flat-coordinate search can pick the wrong town. The
	// check is independent of the query: no row of the list is closer than
	// the one it answered.
	p := domain.LatLng{Lat: 65.9, Lon: -22.0}
	ans, err := s.PlacesForPoints(ctx, []domain.LatLng{p})
	if err != nil {
		t.Fatal(err)
	}
	best := 1e18
	for _, pl := range list {
		if d := geo.HaversineM(p, domain.LatLng{Lat: pl.Lat, Lon: pl.Lon}) / 1000; d < best {
			best = d
		}
	}
	if ans[0].DistanceKm > best+0.05 {
		t.Errorf("high latitude: query answered %q at %.2f km, but a place lies %.2f km away", ans[0].Name, ans[0].DistanceKm, best)
	}
}
