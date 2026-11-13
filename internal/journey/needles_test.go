package journey_test

// Needle counts on the private datasets (phase 17, STOP 1b). The inputs live
// in data/, which is private and gitignored; on a clone without them these
// tests skip. Only counts are asserted, never a coordinate, so no real value
// enters committed code. These are the reproducible source of every figure the
// phase-17 documents quote about excursion rejection (CLAUDE.md invariant 13):
// how many needles the map pipeline drew on each dataset's trips before the
// pass existed, and how many it draws now.

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"roadbook/internal/detect"
	"roadbook/internal/geo"
	"roadbook/internal/journey"
	"roadbook/internal/timeline"
)

// needle shape, as measured: consecutive drawn points A→B→C with both hops
// over needleKm, A and C within a quarter of the shorter hop, inside 3 hours.
// Deliberately written apart from the rule under test, so the count is an
// independent check of what the map shows rather than an echo of the filter.
const needleKm = 20

func countNeedles(j journey.Journey) int {
	var pts []journey.TimedPoint
	for _, l := range j.Legs {
		if l.Kind == journey.LegObserved {
			pts = append(pts, l.Points...)
		}
	}
	n := 0
	for i := 1; i+1 < len(pts); i++ {
		a, b, c := pts[i-1], pts[i], pts[i+1]
		ab := geo.HaversineM(a.Loc, b.Loc) / 1000
		bc := geo.HaversineM(b.Loc, c.Loc) / 1000
		ac := geo.HaversineM(a.Loc, c.Loc) / 1000
		if ab > needleKm && bc > needleKm && ac < 0.25*min(ab, bc) && c.Time.Sub(a.Time).Hours() <= 3 {
			n++
		}
	}
	return n
}

type needleCount struct {
	trips, before, after, tripsBefore, tripsAfter, rejected int
}

func needlesOn(t *testing.T, srcPath string) needleCount {
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

	on := journey.DefaultParams()
	off := on
	off.ExcursionMinKm = 0
	var c needleCount
	for _, cand := range res.Candidates {
		c.trips++
		jb := journey.Assemble(obs, cand.Start, cand.End, off)
		ja := journey.Assemble(obs, cand.Start, cand.End, on)
		nb, na := countNeedles(jb), countNeedles(ja)
		c.before += nb
		c.after += na
		c.rejected += ja.RejectedExcursion
		if nb > 0 {
			c.tripsBefore++
		}
		if na > 0 {
			c.tripsAfter++
		}
	}
	t.Logf("%s: %d trips; needles %d on %d trips before, %d on %d trips after; %d points set aside",
		srcPath, c.trips, c.before, c.tripsBefore, c.after, c.tripsAfter, c.rejected)
	return c
}

func checkNeedles(t *testing.T, got, want needleCount) {
	t.Helper()
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestNeedlesIOS(t *testing.T) {
	checkNeedles(t, needlesOn(t, "../../data/ios-timeline.json"),
		// Re-pinned 2026-11-13 (phase 20): 22 more trips, reached without a
		// long stop. Before: 60 trips, 67 needles on 7, 4 left on 4, 63 set aside.
		needleCount{trips: 82, before: 100, tripsBefore: 11, after: 8, tripsAfter: 5, rejected: 89})
}

func TestNeedlesFixture(t *testing.T) {
	checkNeedles(t, needlesOn(t, "../../data/fixture-2025-04.segments.timeline.json"),
		needleCount{trips: 18}) // Android: nothing to find, nothing set aside
}

func TestNeedlesArchive(t *testing.T) {
	checkNeedles(t, needlesOn(t, "../../data/Timeline.json"),
		needleCount{trips: 68, before: 3, tripsBefore: 2, rejected: 3}) // 66 trips before phase 20
}
