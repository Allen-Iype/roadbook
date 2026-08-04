package journey_test

import (
	"math"
	"testing"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
)

var t0 = time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

func at(sec int) time.Time { return t0.Add(time.Duration(sec) * time.Second) }

func loc(lat, lon float64) *domain.LatLng { return &domain.LatLng{Lat: lat, Lon: lon} }

func TestAssemble(t *testing.T) {
	p := journey.DefaultParams()

	t.Run("empty observations produce an empty journey, not a panic", func(t *testing.T) {
		j := journey.Assemble(domain.Observations{}, t0, at(3600), p)
		if len(j.Legs) != 0 || len(j.Stops) != 0 || j.TotalKm != 0 {
			t.Errorf("got %+v, want empty", j)
		}
	})

	t.Run("points outside the window and points without a location are excluded", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(-60), Loc: loc(10, 70)}, // before window
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(60), Loc: nil},           // no location
				{Time: at(4000), Loc: loc(10, 71)}, // after window
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.TracePointsInWindow != 1 {
			t.Errorf("trace points in window = %d, want 1", j.TracePointsInWindow)
		}
	})

	t.Run("a silence over the threshold becomes a gap leg with exactly two endpoints", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10.0, 70.0)},
				{Time: at(60), Loc: loc(10.1, 70.0)},
				{Time: at(60 + 21*60), Loc: loc(10.5, 70.0)}, // 21 min silence
				{Time: at(120 + 21*60), Loc: loc(10.6, 70.0)},
			},
		}
		j := journey.Assemble(obs, t0, at(7200), p)
		kinds := []journey.LegKind{}
		for _, l := range j.Legs {
			kinds = append(kinds, l.Kind)
		}
		want := []journey.LegKind{journey.LegObserved, journey.LegGap, journey.LegObserved}
		if len(kinds) != len(want) {
			t.Fatalf("legs = %v, want %v", kinds, want)
		}
		for i := range want {
			if kinds[i] != want[i] {
				t.Fatalf("legs = %v, want %v", kinds, want)
			}
		}
		gap := j.Legs[1]
		if gap.GapKind != journey.GapUnknown {
			t.Errorf("gap kind = %q, want unknown", gap.GapKind)
		}
		if len(gap.Points) != 2 {
			t.Errorf("gap points = %d, want 2", len(gap.Points))
		}
		// Observed and gap distances partition the full chord sum.
		if math.Abs(j.TotalKm-(j.ObservedKm+j.InferredKm)) > 1e-12 {
			t.Errorf("total %.6f != observed %.6f + inferred %.6f", j.TotalKm, j.ObservedKm, j.InferredKm)
		}
	})

	t.Run("at an identical timestamp the trace point wins over the raw position", func(t *testing.T) {
		obs := domain.Observations{
			Points:       []domain.PathPoint{{Time: at(0), Loc: loc(10, 70)}},
			RawPositions: []domain.RawPosition{{Time: at(0), Loc: loc(20, 80)}},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.TracePointsKept != 1 || j.RawPointsKept != 0 {
			t.Errorf("kept %d trace + %d raw, want the trace point (CONTRACT.md §4)",
				j.TracePointsKept, j.RawPointsKept)
		}
	})

	t.Run("thinning keeps the earliest of any cluster, regardless of source", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{{Time: at(10), Loc: loc(10, 70)}},
			RawPositions: []domain.RawPosition{
				{Time: at(0), Loc: loc(10, 70)},  // earliest: kept
				{Time: at(29), Loc: loc(10, 70)}, // 29 s after last kept: dropped
				{Time: at(30), Loc: loc(10, 70)}, // exactly the spacing: kept
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.TracePointsKept != 0 || j.RawPointsKept != 2 {
			t.Errorf("kept %d trace + %d raw, want 0 + 2 (raw@0 and raw@30)",
				j.TracePointsKept, j.RawPointsKept)
		}
	})

	t.Run("exact (0,0) is rejected and counted, never assembled", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(60), Loc: loc(0, 0)}, // Null Island: an export defect
			},
			RawPositions: []domain.RawPosition{{Time: at(120), Loc: loc(0, 0)}},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.RejectedNullIsland != 2 {
			t.Errorf("null island rejected = %d, want 2", j.RejectedNullIsland)
		}
		if j.MergedPoints() != 1 {
			t.Errorf("merged = %d, want 1", j.MergedPoints())
		}
	})

	t.Run("the accuracy filter is off by default and excludes only when set", func(t *testing.T) {
		obs := domain.Observations{
			RawPositions: []domain.RawPosition{
				{Time: at(0), Loc: loc(10, 70), AccuracyM: 1500},
				{Time: at(60), Loc: loc(10.1, 70), AccuracyM: 15},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p) // default: filter off
		if j.RawPointsKept != 2 || j.RejectedAccuracy != 0 {
			t.Errorf("filter off: kept %d, rejected %d; want 2, 0", j.RawPointsKept, j.RejectedAccuracy)
		}
		strict := p
		strict.MaxAccuracyM = 100
		j = journey.Assemble(obs, t0, at(3600), strict)
		if j.RawPointsKept != 1 || j.RejectedAccuracy != 1 {
			t.Errorf("filter at 100 m: kept %d, rejected %d; want 1, 1", j.RawPointsKept, j.RejectedAccuracy)
		}
	})

	t.Run("a pause between activities is a stop; a short one is not", func(t *testing.T) {
		obs := domain.Observations{
			Activities: []domain.Activity{
				// Deliberately out of order: findStops must sort a copy.
				{Start: at(2000), End: at(3000), DistanceM: 2000},
				{Start: at(0), End: at(1400), DistanceM: 1000},
				{Start: at(3100), End: at(3600), DistanceM: 500}, // 100 s pause: below threshold
			},
			Points: []domain.PathPoint{
				{Time: at(1400), Loc: loc(10.0, 70.0)},
				{Time: at(1700), Loc: loc(10.001, 70.0)},
				{Time: at(2000), Loc: loc(10.002, 70.0)},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if len(j.Stops) != 1 {
			t.Fatalf("stops = %d, want 1 (the 600 s pause)", len(j.Stops))
		}
		s := j.Stops[0]
		if !s.Start.Equal(at(1400)) || !s.End.Equal(at(2000)) {
			t.Errorf("stop %v..%v, want %v..%v", s.Start, s.End, at(1400), at(2000))
		}
		if s.Points != 3 {
			t.Errorf("stop points = %d, want 3", s.Points)
		}
		// Displacement is first->last, not the path sum.
		wantKm := 0.002 * 111.2 // ~2 latitude millidegrees
		if math.Abs(s.DisplacementKm-wantKm) > 0.01 {
			t.Errorf("displacement = %.4f km, want ~%.4f", s.DisplacementKm, wantKm)
		}
		if j.GoogleDistanceKm != 3.5 {
			t.Errorf("google sum = %.2f km, want 3.5", j.GoogleDistanceKm)
		}
		// The input slice was not reordered (invariant 2).
		if !obs.Activities[0].Start.Equal(at(2000)) {
			t.Error("Assemble reordered the caller's activity slice")
		}
	})

	t.Run("a gap whose implied speed meets AirSpeedMinKmh classifies as air", func(t *testing.T) {
		// One hour of silence covering ~5° of latitude ≈ 556 km → ~556 km/h.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(3600), Loc: loc(15, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if len(j.Legs) != 3 {
			t.Fatalf("legs = %d, want 3 (observed, gap, observed)", len(j.Legs))
		}
		gap := j.Legs[1]
		if gap.GapKind != journey.GapAir {
			t.Errorf("gap kind = %q, want air", gap.GapKind)
		}
		if len(gap.Points) != 2 {
			t.Errorf("air gap carries %d points, want exactly 2 (the arc is presentation, not data)", len(gap.Points))
		}
		if math.Abs(j.AirKm-gap.DistanceKm) > 1e-12 || math.Abs(j.AirKm-j.InferredKm) > 1e-12 {
			t.Errorf("air km = %.4f, want the gap's %.4f and all of inferred %.4f",
				j.AirKm, gap.DistanceKm, j.InferredKm)
		}
	})

	t.Run("a ground-speed gap stays unknown", func(t *testing.T) {
		// The same hour of silence over ~1° ≈ 111 km → ~111 km/h: fast ground
		// transport, below the threshold.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(3600), Loc: loc(11, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if got := j.Legs[1].GapKind; got != journey.GapUnknown {
			t.Errorf("gap kind = %q, want unknown", got)
		}
		if j.AirKm != 0 {
			t.Errorf("air km = %.4f, want 0", j.AirKm)
		}
	})

	t.Run("AirSpeedMinKmh 0 disables classification entirely", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(3600), Loc: loc(15, 70)}, // supersonic-fast gap
			},
		}
		off := p
		off.AirSpeedMinKmh = 0
		j := journey.Assemble(obs, t0, at(3600), off)
		if got := j.Legs[1].GapKind; got != journey.GapUnknown {
			t.Errorf("gap kind = %q, want unknown with classification off", got)
		}
	})
}
