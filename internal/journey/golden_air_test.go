package journey_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"roadbook/internal/journey"
	"roadbook/internal/timeline"
)

// TestGoldenAirFixture pins air-gap classification to the committed 30 Apr
// 2026 flight adventure (out and back by air), the way journey-27jul2026 pins
// ground assembly. The fixture is anonymised (longitudes shifted by a
// constant, placeIds redacted), so every distance and duration is identical
// to the real data and the test runs everywhere — no data/ gate.
//
// The case this fixture exists to hold: Google's activities label only ONE of
// the two flights FLYING — a mode-based classifier would draw the return
// flight as a ground gap. The implied-speed rule (BRIEF §3D) classifies both.
func TestGoldenAirFixture(t *testing.T) {
	f, err := os.Open("../../testdata/journey-30apr2026.anon.json")
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
			AirSpeedMinKmh      float64 `json:"airSpeedMinKmh"`
		} `json:"params"`
		Window struct {
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
		} `json:"window"`
		Join struct {
			TracePointsInWindow int `json:"tracePointsInWindow"`
			MergedPoints        int `json:"mergedPoints"`
		} `json:"join"`
		Legs struct {
			Observed int `json:"observed"`
			Gap      int `json:"gap"`
		} `json:"legs"`
		Stops      int `json:"stops"`
		DistanceKm struct {
			Total             float64 `json:"total"`
			Observed          float64 `json:"observed"`
			Inferred          float64 `json:"inferred"`
			Air               float64 `json:"air"`
			GoogleActivitySum float64 `json:"googleActivitySum"`
			GoogleFlyingSum   float64 `json:"googleFlyingSum"`
		} `json:"distanceKm"`
		AirLegs []struct {
			Index      int     `json:"index"`
			DistanceKm float64 `json:"distanceKm"`
			Seconds    float64 `json:"seconds"`
		} `json:"airLegs"`
		GoogleFlyingActivities int `json:"googleFlyingActivities"`
	}
	raw, err := os.ReadFile("../../testdata/journey-30apr2026.expected.json")
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

	p := journey.DefaultParams()
	p.GapThresholdMinutes = exp.Params.GapThresholdMinutes
	p.ThinSpacingSeconds = exp.Params.ThinSpacingSeconds
	p.AirSpeedMinKmh = exp.Params.AirSpeedMinKmh
	j := journey.Assemble(obs, winStart, winEnd, p)

	// 4-dp expected values; 1e-4 absorbs that rounding and float summation
	// order, nothing more.
	const tol = 1e-4
	closeTo := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > tol {
			t.Errorf("%s = %.6f, want %.4f", name, got, want)
		}
	}

	if j.TracePointsInWindow != exp.Join.TracePointsInWindow {
		t.Errorf("trace points in window = %d, want %d", j.TracePointsInWindow, exp.Join.TracePointsInWindow)
	}
	if j.MergedPoints() != exp.Join.MergedPoints {
		t.Errorf("merged points = %d, want %d", j.MergedPoints(), exp.Join.MergedPoints)
	}

	closeTo("total km", j.TotalKm, exp.DistanceKm.Total)
	closeTo("observed km", j.ObservedKm, exp.DistanceKm.Observed)
	closeTo("inferred km", j.InferredKm, exp.DistanceKm.Inferred)
	closeTo("air km", j.AirKm, exp.DistanceKm.Air)
	closeTo("google sum", j.GoogleDistanceKm, exp.DistanceKm.GoogleActivitySum)
	// Google's ground figure is the activity sum minus its FLYING share —
	// both already pinned by this file, so the assertion derives from them.
	closeTo("google ground km", j.GoogleGroundKm,
		exp.DistanceKm.GoogleActivitySum-exp.DistanceKm.GoogleFlyingSum)

	// The air legs sit exactly where the measurement put them; every other
	// gap in this journey is ground-speed and stays unknown.
	airByIndex := map[int]struct {
		km  float64
		sec float64
	}{}
	for _, a := range exp.AirLegs {
		airByIndex[a.Index] = struct {
			km  float64
			sec float64
		}{a.DistanceKm, a.Seconds}
	}
	observed, gaps := 0, 0
	for i, l := range j.Legs {
		if l.Kind == journey.LegObserved {
			observed++
			continue
		}
		gaps++
		want, isAir := airByIndex[i]
		if isAir {
			if l.GapKind != journey.GapAir {
				t.Errorf("leg %d kind = %q, want air", i, l.GapKind)
			}
			if len(l.Points) != 2 {
				t.Errorf("air leg %d carries %d points, want exactly 2 (the arc is presentation, not data)", i, len(l.Points))
			}
			closeTo("air leg km", l.DistanceKm, want.km)
			if got := l.End().Sub(l.Start()).Seconds(); got != want.sec {
				t.Errorf("air leg %d duration = %.0f s, want %.0f", i, got, want.sec)
			}
		} else if l.GapKind != journey.GapUnknown {
			t.Errorf("leg %d kind = %q, want unknown (ground-speed gap)", i, l.GapKind)
		}
	}
	if observed != exp.Legs.Observed || gaps != exp.Legs.Gap {
		t.Errorf("legs = %d observed + %d gap, want %d + %d", observed, gaps, exp.Legs.Observed, exp.Legs.Gap)
	}
	if len(j.Stops) != exp.Stops {
		t.Errorf("stops = %d, want %d", len(j.Stops), exp.Stops)
	}

	// The mode cross-check this fixture pins: one FLYING activity, two
	// flights. Mode is a guess (CLAUDE.md source hazards); speed is the rule.
	var flyingKm float64
	var flyingN int
	for _, a := range obs.Activities {
		if a.Mode == "FLYING" && !a.End.Before(winStart) && !a.Start.After(winEnd) {
			flyingKm += a.DistanceM / 1000
			flyingN++
		}
	}
	if flyingN != exp.GoogleFlyingActivities {
		t.Errorf("FLYING activities in window = %d, want %d", flyingN, exp.GoogleFlyingActivities)
	}
	closeTo("google flying km", flyingKm, exp.DistanceKm.GoogleFlyingSum)
}
