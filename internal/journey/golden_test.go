package journey_test

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"testing"
	"time"

	"roadbook/internal/geo"
	"roadbook/internal/journey"
	"roadbook/internal/timeline"
)

// TestGoldenFixture pins journey assembly to the committed measurement of the
// 27 Jul 2026 overnight bus journey, the way the reference detector pins
// detection. The fixture is anonymised (longitudes shifted by a constant), so
// every distance, duration, and leg boundary is identical to the real data and
// this test runs everywhere — no data/ gate. Semantics under test:
// testdata/journey-27jul2026.CONTRACT.md.
func TestGoldenFixture(t *testing.T) {
	f, err := os.Open("../../testdata/journey-27jul2026.anon.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	obs, _, err := timeline.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	var exp struct {
		Params struct {
			GapThresholdMinutes float64 `json:"gapThresholdMinutes"`
			ThinSpacingSeconds  float64 `json:"thinSpacingSeconds"`
		} `json:"params"`
		Window struct {
			StartTime       string  `json:"startTime"`
			EndTime         string  `json:"endTime"`
			DurationSeconds float64 `json:"durationSeconds"`
		} `json:"window"`
		Join struct {
			TracePoints                int `json:"tracePoints"`
			MergedPointsWithRawSignals int `json:"mergedPointsWithRawSignals"`
			MergedComposition          struct {
				FromTrace      int `json:"fromTrace"`
				FromRawSignals int `json:"fromRawSignals"`
			} `json:"mergedComposition"`
		} `json:"join"`
		DistanceKm struct {
			ReconstructedChordSum float64 `json:"reconstructedChordSum"`
			TraceOnlyChordSum     float64 `json:"traceOnlyChordSum"`
			GoogleActivitySum     float64 `json:"googleActivitySum"`
			DisagreementPct       float64 `json:"disagreementPct"`
		} `json:"distanceKm"`
		Gaps struct {
			Count                 int     `json:"count"`
			InferredKm            float64 `json:"inferredKm"`
			InferredPctOfDistance float64 `json:"inferredPctOfDistance"`
			WorstGapSeconds       []int   `json:"worstGapSeconds"`
		} `json:"gaps"`
		Legs struct {
			Observed int `json:"observed"`
			Gap      int `json:"gap"`
		} `json:"legs"`
		StopDetection struct {
			RestHaltStart  string  `json:"restHaltStart"`
			RestHaltEnd    string  `json:"restHaltEnd"`
			DisplacementKm float64 `json:"displacementKm"`
		} `json:"stopDetection"`
	}
	raw, err := os.ReadFile("../../testdata/journey-27jul2026.expected.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &exp); err != nil {
		t.Fatal(err)
	}

	winStart, err := time.Parse(time.RFC3339, exp.Window.StartTime)
	if err != nil {
		t.Fatal(err)
	}
	winEnd, err := time.Parse(time.RFC3339, exp.Window.EndTime)
	if err != nil {
		t.Fatal(err)
	}
	if got := winEnd.Sub(winStart).Seconds(); got != exp.Window.DurationSeconds {
		t.Errorf("window duration = %.0f s, want %.0f", got, exp.Window.DurationSeconds)
	}

	p := journey.DefaultParams()
	p.GapThresholdMinutes = exp.Params.GapThresholdMinutes
	p.ThinSpacingSeconds = exp.Params.ThinSpacingSeconds
	j := journey.Assemble(obs, winStart, winEnd, p)

	// The expected file stores distances at 4 decimal places; 1e-4 absorbs
	// that rounding and float summation order, nothing more.
	const tol = 1e-4
	closeTo := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > tol {
			t.Errorf("%s = %.6f, want %.4f", name, got, want)
		}
	}

	if j.TracePointsInWindow != exp.Join.TracePoints {
		t.Errorf("trace points in window = %d, want %d", j.TracePointsInWindow, exp.Join.TracePoints)
	}
	if j.MergedPoints() != exp.Join.MergedPointsWithRawSignals {
		t.Errorf("merged points = %d, want %d", j.MergedPoints(), exp.Join.MergedPointsWithRawSignals)
	}
	if j.TracePointsKept != exp.Join.MergedComposition.FromTrace ||
		j.RawPointsKept != exp.Join.MergedComposition.FromRawSignals {
		t.Errorf("merged composition = %d trace + %d raw, want %d + %d",
			j.TracePointsKept, j.RawPointsKept,
			exp.Join.MergedComposition.FromTrace, exp.Join.MergedComposition.FromRawSignals)
	}

	closeTo("total chord", j.TotalKm, exp.DistanceKm.ReconstructedChordSum)
	closeTo("google sum", j.GoogleDistanceKm, exp.DistanceKm.GoogleActivitySum)
	closeTo("disagreement pct",
		(j.TotalKm-j.GoogleDistanceKm)/j.GoogleDistanceKm*100, exp.DistanceKm.DisagreementPct)
	closeTo("inferred km", j.InferredKm, exp.Gaps.InferredKm)
	closeTo("inferred pct", j.InferredKm/j.TotalKm*100, exp.Gaps.InferredPctOfDistance)

	// The trace-only chord is a property of the fixture, not a Journey field:
	// recompute it here from the parsed points.
	var traceOnly float64
	var prev *journey.TimedPoint
	for _, pp := range obs.Points {
		if pp.Loc == nil || pp.Time.Before(winStart) || pp.Time.After(winEnd) {
			continue
		}
		cur := journey.TimedPoint{Time: pp.Time, Loc: *pp.Loc}
		if prev != nil {
			traceOnly += geo.HaversineM(prev.Loc, cur.Loc) / 1000
		}
		prev = &cur
	}
	closeTo("trace-only chord", traceOnly, exp.DistanceKm.TraceOnlyChordSum)

	var observed, gaps int
	var gapSeconds []int
	for _, l := range j.Legs {
		switch l.Kind {
		case journey.LegObserved:
			observed++
		case journey.LegGap:
			gaps++
			gapSeconds = append(gapSeconds, int(l.End().Sub(l.Start()).Seconds()))
			if l.GapKind != journey.GapUnknown {
				t.Errorf("gap leg kind = %q, want unknown (phase 2 never classifies)", l.GapKind)
			}
			if len(l.Points) != 2 {
				t.Errorf("gap leg carries %d points, want exactly 2", len(l.Points))
			}
		}
	}
	if observed != exp.Legs.Observed || gaps != exp.Legs.Gap {
		t.Errorf("legs = %d observed + %d gap, want %d + %d",
			observed, gaps, exp.Legs.Observed, exp.Legs.Gap)
	}
	if gaps != exp.Gaps.Count {
		t.Errorf("gap count = %d, want %d", gaps, exp.Gaps.Count)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(gapSeconds)))
	if len(gapSeconds) != len(exp.Gaps.WorstGapSeconds) {
		t.Fatalf("gap durations %v, want %v", gapSeconds, exp.Gaps.WorstGapSeconds)
	}
	for i := range gapSeconds {
		if gapSeconds[i] != exp.Gaps.WorstGapSeconds[i] {
			t.Errorf("worst gaps %v, want %v", gapSeconds, exp.Gaps.WorstGapSeconds)
			break
		}
	}

	haltStart, _ := time.Parse(time.RFC3339, exp.StopDetection.RestHaltStart)
	haltEnd, _ := time.Parse(time.RFC3339, exp.StopDetection.RestHaltEnd)
	if len(j.Stops) != 1 {
		t.Fatalf("stops = %d, want exactly 1 (the rest halt)", len(j.Stops))
	}
	s := j.Stops[0]
	if !s.Start.Equal(haltStart) || !s.End.Equal(haltEnd) {
		t.Errorf("rest halt %v..%v, want %v..%v", s.Start, s.End, haltStart, haltEnd)
	}
	closeTo("halt displacement", s.DisplacementKm, exp.StopDetection.DisplacementKm)
}
