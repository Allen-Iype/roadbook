package store_test

import (
	"context"
	"testing"

	"roadbook/internal/domain"
	"roadbook/internal/states"
	"roadbook/internal/store/storetest"
)

// Load only one country's regions: the full 4,589-polygon load is what
// `roadbook states` does once per instance, not what a unit test needs to
// prove the replace, count, and attribution semantics.
func icelandStates(t *testing.T) []states.State {
	t.Helper()
	all, err := states.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	var out []states.State
	for _, s := range all {
		if s.CountryCode == "IS" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		t.Fatal("no Icelandic regions in the bundled file")
	}
	return out
}

// The -if-empty startup path: a fresh schema counts zero; a double load
// counts once (replace, not append).
func TestCountStates(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	n, err := s.CountStates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("fresh schema: want 0 states, got %d", n)
	}
	list := icelandStates(t)
	if err := s.ReplaceStates(ctx, list); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceStates(ctx, list); err != nil {
		t.Fatal(err)
	}
	n, err = s.CountStates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(list)) {
		t.Fatalf("after double load: want %d states, got %d", len(list), n)
	}
}

// Attribution reads in journey order: a synthetic drive from the Westfjords
// to the south coast names Vestfirðir first and Suðurland last, and the
// reverse drive reverses the list. A point far out at sea attributes to
// nothing. Before any load, the answer is empty, not an error.
func TestStatesForPoints(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	westfjords := domain.LatLng{Lat: 66.07, Lon: -23.12} // Ísafjörður
	// Inland, deliberately: a shoreline town (Vík was the first try) can
	// fall a few hundred metres outside a 1:10m coastline polygon.
	south := domain.LatLng{Lat: 63.93, Lon: -21.00} // Selfoss
	sea := domain.LatLng{Lat: 60.0, Lon: -30.0}

	got, err := s.StatesForPoints(ctx, []domain.LatLng{westfjords, south})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("unloaded table: want no attribution, got %+v", got)
	}

	if err := s.ReplaceStates(ctx, icelandStates(t)); err != nil {
		t.Fatal(err)
	}
	names := func(pts ...domain.LatLng) []string {
		refs, err := s.StatesForPoints(ctx, pts)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, len(refs))
		for i, r := range refs {
			out[i] = r.Name
			if r.CountryCode != "IS" {
				t.Errorf("%s country = %q, want IS", r.Name, r.CountryCode)
			}
		}
		return out
	}
	if got := names(westfjords, sea, south); len(got) != 2 || got[0] != "Vestfirðir" || got[1] != "Suðurland" {
		t.Errorf("westfjords→south = %v, want [Vestfirðir Suðurland]", got)
	}
	if got := names(south, westfjords); len(got) != 2 || got[0] != "Suðurland" || got[1] != "Vestfirðir" {
		t.Errorf("south→westfjords = %v, want [Suðurland Vestfirðir]", got)
	}
	if got := names(sea); len(got) != 0 {
		t.Errorf("open sea = %v, want nothing", got)
	}
}
