package journey_test

// How a trip's nights fall, on the private datasets (phase 19, #14). The
// inputs live in data/, private and gitignored; on a clone without them
// these tests skip. Only counts are asserted and printed — never a date, a
// coordinate or a place — so nothing real enters committed code. This is
// the reproducible source of the night figures the phase-19 documents
// quote (CLAUDE.md invariant 13). The demo's nights are pinned through
// the API in internal/api/nights_journey_test.go.

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"roadbook/internal/detect"
	"roadbook/internal/journey"
	"roadbook/internal/timeline"
)

type nightCount struct {
	trips, tripsWithNights     int
	nights                     int
	atStop, moving, unknown    int
	atStopWithoutOwnPosition   int // the holding stop has no recorded points; phase 18 names it from the visit record
	civilDaysMinusTripsCounted int // sum over trips of (civil days - 1): must equal nights
}

func nightsOn(t *testing.T, srcPath string) nightCount {
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
	var c nightCount
	for _, cand := range res.Candidates {
		c.trips++
		j := journey.Assemble(obs, cand.Start, cand.End, journey.DefaultParams())
		nights := journey.Nights(j, journey.DefaultNightParams())
		c.civilDaysMinusTripsCounted += journey.Summarize(j).CivilDays - 1
		if len(nights) > 0 {
			c.tripsWithNights++
		}
		for _, n := range nights {
			c.nights++
			switch n.Kind {
			case journey.NightStop:
				c.atStop++
				if s := j.Stops[n.StopIndex]; s.Loc.Lat == 0 && s.Loc.Lon == 0 {
					c.atStopWithoutOwnPosition++
				}
			case journey.NightMoving:
				c.moving++
			default:
				c.unknown++
			}
		}
	}
	t.Logf("%s: %d trips (%d with a night), %d nights: %d at a stop (%d of them a stop with no points of its own), %d on the move, %d not known",
		srcPath, c.trips, c.tripsWithNights, c.nights, c.atStop, c.atStopWithoutOwnPosition, c.moving, c.unknown)
	return c
}

func TestNightsOnTheFixture(t *testing.T) {
	if c := nightsOn(t, "../../data/fixture-2025-04.timeline.json"); c != fixtureNights {
		t.Errorf("fixture nights moved: %+v, want %+v", c, fixtureNights)
	}
}

func TestNightsOnTheArchive(t *testing.T) {
	if c := nightsOn(t, "../../data/Timeline.json"); c != archiveNights {
		t.Errorf("archive nights moved: %+v, want %+v", c, archiveNights)
	}
}

func TestNightsOnTheIPhoneExport(t *testing.T) {
	if c := nightsOn(t, "../../data/ios-timeline.json"); c != iphoneNights {
		t.Errorf("iPhone nights moved: %+v, want %+v", c, iphoneNights)
	}
}

// Measured 2026-11-13, default parameters (22:00–06:00, at least 3 h).
var (
	fixtureNights = nightCount{trips: 18, tripsWithNights: 15, nights: 123,
		atStop: 91, moving: 22, unknown: 10, atStopWithoutOwnPosition: 3, civilDaysMinusTripsCounted: 123}
	archiveNights = nightCount{trips: 66, tripsWithNights: 58, nights: 568,
		atStop: 444, moving: 45, unknown: 79, atStopWithoutOwnPosition: 18, civilDaysMinusTripsCounted: 568}
	iphoneNights = nightCount{trips: 60, tripsWithNights: 53, nights: 264,
		atStop: 193, moving: 9, unknown: 62, atStopWithoutOwnPosition: 17, civilDaysMinusTripsCounted: 264}
)
