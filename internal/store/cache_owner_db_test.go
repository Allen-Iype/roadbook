package store_test

// Phase 23 (#9), decided 2026-11-25: the two road caches belong to an
// account. Written before the store code. One account's cached road is
// never read by another, even for the same question, and each keeps its
// own answer.

import (
	"context"
	"testing"

	"roadbook/internal/domain"
	"roadbook/internal/route"
	"roadbook/internal/store/storetest"
)

func TestRoadCachesAreOwned(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	seedTenant(t, s, userA)
	seedTenant(t, s, userB)

	gap := route.KeyFor(domain.LatLng{Lat: 10, Lon: 70}, domain.LatLng{Lat: 10.1, Lon: 70}, "driving")
	road := []domain.LatLng{{Lat: 10, Lon: 70}, {Lat: 10.05, Lon: 70.01}, {Lat: 10.1, Lon: 70}}
	stretch := matchTrace("same-points", 3)

	if err := s.SaveRoute(ctx, userA, gap, route.Cached{Status: route.StatusRouted, Points: road, DistanceM: 12000}, "test", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMatch(ctx, userA, stretch, route.CachedMatch{
		Status: route.StatusMatched, Points: road, DistanceM: 12000, EndsPlaced: true,
	}, "test", ""); err != nil {
		t.Fatal(err)
	}

	// B asks the same two questions and finds nothing.
	if got, err := s.LookupRoutes(ctx, userB, []route.Key{gap}); err != nil || len(got) != 0 {
		t.Errorf("B read %d of A's routes (err %v), want 0", len(got), err)
	}
	if got, err := s.LookupMatches(ctx, userB, []route.MatchKey{stretch.Key}); err != nil || len(got) != 0 {
		t.Errorf("B read %d of A's matches (err %v), want 0", len(got), err)
	}

	// B's own answer to the same question sits beside A's, not over it.
	if err := s.SaveRoute(ctx, userB, gap, route.Cached{Status: route.StatusNoRoute}, "test", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMatch(ctx, userB, stretch, route.CachedMatch{Status: route.StatusNoMatch}, "test", ""); err != nil {
		t.Fatal(err)
	}
	ra, _ := s.LookupRoutes(ctx, userA, []route.Key{gap})
	rb, _ := s.LookupRoutes(ctx, userB, []route.Key{gap})
	if ra[gap].Status != route.StatusRouted || rb[gap].Status != route.StatusNoRoute {
		t.Errorf("routes: A %q, B %q; want routed and no_route", ra[gap].Status, rb[gap].Status)
	}
	ma, _ := s.LookupMatches(ctx, userA, []route.MatchKey{stretch.Key})
	mb, _ := s.LookupMatches(ctx, userB, []route.MatchKey{stretch.Key})
	if ma[stretch.Key].Status != route.StatusMatched || mb[stretch.Key].Status != route.StatusNoMatch {
		t.Errorf("matches: A %q, B %q; want matched and no_match", ma[stretch.Key].Status, mb[stretch.Key].Status)
	}
}
