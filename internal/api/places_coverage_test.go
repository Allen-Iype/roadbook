package api_test

// How many stops get a name, on the private datasets (phase 18, #16). The
// inputs live in data/, private and gitignored; on a clone without them
// these tests skip. Only counts are asserted and printed — never a
// coordinate or a place name — so nothing real enters committed code. This
// is the reproducible source of every naming figure the phase-18 documents
// quote for these datasets (CLAUDE.md invariant 13); the demo's figures
// come from TestDemoStopsAreNamed.
//
// Nothing is imported into the database: the export is parsed in memory,
// detection and assembly run as the pure functions they are, and only the
// two reference tables are loaded (the embedded public lists of places and
// regions), so the nearest-place and region queries are the real ones.

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"

	"roadbook/internal/api"
	"roadbook/internal/detect"
	"roadbook/internal/journey"
	"roadbook/internal/places"
	"roadbook/internal/states"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
	"roadbook/internal/timeline"
)

type nameCount struct {
	trips, stops                int
	recorded, visit, noPosition int // which position each stop is looked up at
	near, from, unnamed         int // of the stops that have a position
	tripsWithUnnamed            int
	// Regions (#17): legs and positioned stops whose points fall in no
	// admin-1 polygon, and legs that cross from one region into another.
	legs, legsNoRegion, legsCrossing int
	stopsNoRegion                    int
}

func namesOn(t *testing.T, s *store.Store, srcPath string) nameCount {
	t.Helper()
	src, err := os.Open(srcPath)
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("private dataset %s not present; skipping", srcPath)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	obs, _, err := timeline.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res := detect.Run(obs, detect.DefaultParams())

	ctx := context.Background()
	var c nameCount
	for _, cand := range res.Candidates {
		c.trips++
		j := journey.Assemble(obs, cand.Start, cand.End, journey.DefaultParams())
		// The whole export's visits, where the handler passes the window's:
		// StopPositions only counts visits that overlap a stop, so the
		// answer is the same.
		positions, names, err := api.StopNames(ctx, s, j, obs.Visits, journey.DefaultPlaceParams())
		if err != nil {
			t.Fatal(err)
		}
		legStates, stopStates, err := api.Regions(ctx, s, j, positions)
		if err != nil {
			t.Fatal(err)
		}
		for _, ls := range legStates {
			c.legs++
			switch {
			case len(ls) == 0:
				c.legsNoRegion++
			case len(ls) > 1:
				c.legsCrossing++
			}
		}
		for i := range j.Stops {
			if positions[i].OK && stopStates[i] == nil {
				c.stopsNoRegion++
			}
		}
		unnamedHere := false
		for i := range j.Stops {
			c.stops++
			switch {
			case !positions[i].OK:
				c.noPosition++
				continue
			case positions[i].Kind == journey.PositionVisit:
				c.visit++
			default:
				c.recorded++
			}
			switch {
			case names[i] == nil:
				c.unnamed++
				unnamedHere = true
			case names[i].Near:
				c.near++
			default:
				c.from++
			}
		}
		if unnamedHere {
			c.tripsWithUnnamed++
		}
	}
	t.Logf("%s: regions: %d legs, %d in no region, %d crossing a border; %d positioned stops in no region",
		srcPath, c.legs, c.legsNoRegion, c.legsCrossing, c.stopsNoRegion)
	t.Logf("%s: %d trips, %d stops; position: %d recorded, %d from a visit, %d none; named: %d near, %d \"N km from\", %d beyond the limit (on %d trips)",
		srcPath, c.trips, c.stops, c.recorded, c.visit, c.noPosition, c.near, c.from, c.unnamed, c.tripsWithUnnamed)
	return c
}

func placesStore(t *testing.T) *store.Store {
	t.Helper()
	s := storetest.Open(t)
	list, err := places.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplacePlaces(context.Background(), list, places.SourceOSM); err != nil {
		t.Fatal(err)
	}
	regions, err := states.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceStates(context.Background(), regions); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStopNamesOnTheFixture(t *testing.T) {
	c := namesOn(t, placesStore(t), "../../data/fixture-2025-04.timeline.json")
	if c != fixtureNames {
		t.Errorf("fixture naming moved: %+v, want %+v", c, fixtureNames)
	}
}

func TestStopNamesOnTheArchive(t *testing.T) {
	c := namesOn(t, placesStore(t), "../../data/Timeline.json")
	if c != archiveNames {
		t.Errorf("archive naming moved: %+v, want %+v", c, archiveNames)
	}
}

func TestStopNamesOnTheIPhoneExport(t *testing.T) {
	c := namesOn(t, placesStore(t), "../../data/ios-timeline.json")
	if c != iphoneNames {
		t.Errorf("iPhone naming moved: %+v, want %+v", c, iphoneNames)
	}
}

// Measured 2026-11-11 against the embedded list, default parameters (near
// within 10 km, named within 25 km).
var (
	fixtureNames = nameCount{trips: 18, stops: 574, recorded: 485, visit: 89, noPosition: 0,
		near: 506, from: 67, unnamed: 1, tripsWithUnnamed: 1,
		legs: 2594, legsNoRegion: 28, legsCrossing: 61, stopsNoRegion: 15}
	archiveNames = nameCount{trips: 66, stops: 2227, recorded: 1875, visit: 340, noPosition: 12,
		near: 1912, from: 302, unnamed: 1, tripsWithUnnamed: 1,
		legs: 9622, legsNoRegion: 47, legsCrossing: 176, stopsNoRegion: 23}
	iphoneNames = nameCount{trips: 60, stops: 824, recorded: 744, visit: 77, noPosition: 3,
		near: 735, from: 84, unnamed: 2, tripsWithUnnamed: 1,
		legs: 3872, legsNoRegion: 22, legsCrossing: 128, stopsNoRegion: 15}
)
