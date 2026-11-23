package store_test

// Phase 23 (#9): the match cache. Written before the store code. A row is
// one answered question — "which road do these points lie on?" — keyed by
// the request itself, read back exactly, replaced by a later answer, and
// absent for a question nobody asked.

import (
	"context"
	"testing"

	"roadbook/internal/domain"
	"roadbook/internal/route"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

func matchTrace(hash string, n int) route.Trace {
	return route.Trace{
		Key:    route.MatchKey{PointsHash: hash, Profile: "driving", RadiusM: 100},
		Points: make([]route.TracePoint, n),
	}
}

func TestMatchCacheRoundTrip(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	a, b, never := matchTrace("aaaa", 4), matchTrace("bbbb", 2), matchTrace("cccc", 3)
	keys := []route.MatchKey{a.Key, b.Key, never.Key}

	got, err := s.LookupMatches(ctx, keys)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("empty cache answered %d keys", len(got))
	}

	road := []domain.LatLng{{Lat: 10.0001, Lon: 70.0002}, {Lat: 10.015, Lon: 70.003}}
	if err := s.SaveMatch(ctx, a, route.CachedMatch{
		Status: route.StatusMatched, Points: road, DistanceM: 3601.5, Unplaced: 1, EndsPlaced: true,
	}, "osrm http://localhost:5000", "test-extract"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMatch(ctx, b, route.CachedMatch{Status: route.StatusNoMatch}, "osrm http://localhost:5000", ""); err != nil {
		t.Fatal(err)
	}

	got, err = s.LookupMatches(ctx, keys)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("lookup answered %d keys, want 2 (the third was never asked)", len(got))
	}
	ma := got[a.Key]
	if ma.Status != route.StatusMatched || ma.DistanceM != 3601.5 || ma.Unplaced != 1 || !ma.EndsPlaced {
		t.Errorf("matched row = %+v", ma)
	}
	if len(ma.Points) != 2 || ma.Points[0] != road[0] || ma.Points[1] != road[1] {
		t.Errorf("geometry = %+v, want %+v exactly, lat first", ma.Points, road)
	}
	if mb := got[b.Key]; mb.Status != route.StatusNoMatch || mb.Points != nil || mb.DistanceM != 0 {
		t.Errorf("no_match row = %+v, want the bare remembered negative", mb)
	}

	// A different radius or profile is a different question.
	other := a.Key
	other.RadiusM = 50
	cyc := a.Key
	cyc.Profile = "cycling"
	got, err = s.LookupMatches(ctx, []route.MatchKey{other, cyc})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("radius 50 / cycling answered from the radius-100 driving row: %+v", got)
	}

	// -refresh: a later answer replaces the row.
	if err := s.SaveMatch(ctx, a, route.CachedMatch{Status: route.StatusNoMatch}, "osrm http://localhost:5000", "newer-extract"); err != nil {
		t.Fatal(err)
	}
	got, err = s.LookupMatches(ctx, []route.MatchKey{a.Key})
	if err != nil {
		t.Fatal(err)
	}
	if ma := got[a.Key]; ma.Status != route.StatusNoMatch || ma.Points != nil {
		t.Errorf("after replacement = %+v, want no_match with no geometry", ma)
	}
}

// A batch records what it did for matching beside what it did for routing.
func TestRouteRunRecordsMatching(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	id, err := s.InsertRouteRun(ctx, "osrm http://localhost:5000", "test-extract",
		map[string]any{"match_radius_m": 100},
		store.RouteRunCounts{
			GapsFound: 5, CacheHits: 1, Routed: 3, NoRoute: 1, Failures: 0,
			StretchesFound: 9, MatchCacheHits: 2, Matched: 5, NoMatch: 1, MatchFailures: 1,
		})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.RouteRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	want := store.RouteRunCounts{
		GapsFound: 5, CacheHits: 1, Routed: 3, NoRoute: 1, Failures: 0,
		StretchesFound: 9, MatchCacheHits: 2, Matched: 5, NoMatch: 1, MatchFailures: 1,
	}
	if got != want {
		t.Errorf("route run %d = %+v, want %+v", id, got, want)
	}
}
