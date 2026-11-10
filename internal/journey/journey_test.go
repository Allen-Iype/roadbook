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

	t.Run("a run of mislocated fixes is rejected as a teleport cluster", func(t *testing.T) {
		// The measured failure (candidate 62): the device in one place, a
		// wrong geolocation-database entry repeatedly placing it ~300 km
		// away. The run's interior is self-consistent, so point-wise speed
		// rules pass it; the cluster rule kills it because both edges are
		// impossible and bridging the neighbours is plausible.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(60), Loc: loc(10.001, 70)},
				{Time: at(120), Loc: loc(10.002, 70)},
				{Time: at(180), Loc: loc(12.7, 70)}, // ~300 km in 60 s
				{Time: at(240), Loc: loc(12.7, 70)},
				{Time: at(300), Loc: loc(12.7, 70)},
				{Time: at(360), Loc: loc(10.003, 70)},
				{Time: at(420), Loc: loc(10.004, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.RejectedSpeed != 3 {
			t.Fatalf("rejected = %d, want the whole 3-point run", j.RejectedSpeed)
		}
		if len(j.Legs) != 1 || j.Legs[0].Kind != journey.LegObserved {
			t.Fatalf("legs = %d, want 1 observed leg with the teleport gone", len(j.Legs))
		}
		if j.TotalKm > 1 {
			t.Errorf("total = %.1f km, want < 1 (no 300 km fiction)", j.TotalKm)
		}
	})

	t.Run("a long silence before the teleport does not launder it", func(t *testing.T) {
		// Overnight silence makes the run's ENTRY speed plausible (300 km
		// over 10 quiet hours); the exit is still impossible and the bridge
		// still plausible — rejected. This is the case that defeats
		// last-accepted-anchor speed rules.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(10 * 3600), Loc: loc(12.7, 70)},
				{Time: at(10*3600 + 60), Loc: loc(12.7, 70)},
				{Time: at(10*3600 + 120), Loc: loc(10.001, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(11*3600), p)
		if j.RejectedSpeed != 2 {
			t.Fatalf("rejected = %d, want 2", j.RejectedSpeed)
		}
		if j.TotalKm > 1 {
			t.Errorf("total = %.1f km, want < 1", j.TotalKm)
		}
	})

	t.Run("a real flight is never evaluated, let alone rejected", func(t *testing.T) {
		// 1000 km in 2 h is 500 km/h — no impossible edge, so the rule
		// does not fire and the gap classifies as air downstream.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(2 * 3600), Loc: loc(19, 70)},
				{Time: at(2*3600 + 60), Loc: loc(19.001, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(3*3600), p)
		if j.RejectedSpeed != 0 {
			t.Fatalf("rejected = %d, want 0", j.RejectedSpeed)
		}
		if got := j.Legs[1].GapKind; got != journey.GapAir {
			t.Errorf("gap kind = %q, want air", got)
		}
	})

	t.Run("an impossible edge without a plausible bridge is kept, conservatively", func(t *testing.T) {
		// If removing the cluster would NOT restore plausibility, the data
		// is contradictory and the rule declines to choose a side.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(60), Loc: loc(12.7, 70)},   // impossible hop
				{Time: at(120), Loc: loc(12.75, 70)}, // stays there
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.RejectedSpeed != 0 {
			t.Errorf("rejected = %d, want 0 (no bridge to judge by)", j.RejectedSpeed)
		}
	})

	t.Run("same-minute quantization is not a teleport", func(t *testing.T) {
		// Trace timestamps are minute-truncated: two points in the same
		// minute 130 m apart divide to infinite speed without any
		// mislocation. The edge duration is floored at ThinSpacingSeconds
		// (the stream's stated resolution), so this must survive — the
		// false positive found on the 30 Apr fixture during implementation.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10.0, 70.0)},
				{Time: at(240), Loc: loc(10.018, 70.0)},  // ~2 km on, 30 km/h
				{Time: at(360), Loc: loc(10.036, 70.0)},  // next cluster
				{Time: at(360), Loc: loc(10.0372, 70.0)}, // same minute, ~130 m
				{Time: at(600), Loc: loc(10.054, 70.0)},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.RejectedSpeed != 0 {
			t.Errorf("rejected = %d, want 0 (quantization noise, not teleportation)", j.RejectedSpeed)
		}
	})

	t.Run("google ground excludes FLYING activities; divergence compares ground to ground", func(t *testing.T) {
		obs := domain.Observations{
			Activities: []domain.Activity{
				{Start: at(0), End: at(1000), DistanceM: 100_000, Mode: "IN_PASSENGER_VEHICLE"},
				{Start: at(1100), End: at(5000), DistanceM: 1_000_000, Mode: "FLYING"},
			},
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(1000), Loc: loc(10.9, 70)}, // ~100 km observed
			},
		}
		j := journey.Assemble(obs, t0, at(5000), p)
		if math.Abs(j.GoogleDistanceKm-1100) > 1e-9 || math.Abs(j.GoogleGroundKm-100) > 1e-9 {
			t.Fatalf("google = %.1f / ground %.1f, want 1100 / 100", j.GoogleDistanceKm, j.GoogleGroundKm)
		}
		// Ground reconstruction ~100.1 km vs Google ground 100: within the
		// 15%% default, not flagged.
		pct, ok := j.DivergencePct()
		if !ok {
			t.Fatal("divergence not computable despite ground activities")
		}
		if math.Abs(pct) > 1 || j.DivergenceFlagged() {
			t.Errorf("divergence = %.2f%% flagged=%v, want ~0%% unflagged", pct, j.DivergenceFlagged())
		}
	})

	t.Run("divergence beyond the named threshold is flagged, either direction", func(t *testing.T) {
		obs := domain.Observations{
			Activities: []domain.Activity{
				{Start: at(0), End: at(1000), DistanceM: 200_000, Mode: "IN_BUS"},
			},
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(1000), Loc: loc(10.9, 70)}, // ~100 km reconstructed vs 200 claimed
			},
		}
		j := journey.Assemble(obs, t0, at(5000), p)
		pct, ok := j.DivergencePct()
		if !ok || pct > -49 || pct < -51 {
			t.Fatalf("divergence = %.1f%% ok=%v, want ~-50%%", pct, ok)
		}
		if !j.DivergenceFlagged() {
			t.Error("a 50%% shortfall must flag at the 15%% default")
		}
		off := p
		off.DivergenceWarnPct = 0
		j2 := journey.Assemble(obs, t0, at(5000), off)
		if j2.DivergenceFlagged() {
			t.Error("DivergenceWarnPct 0 disables the flag")
		}
	})

	t.Run("no ground activities means nothing to compare, not zero divergence", func(t *testing.T) {
		obs := domain.Observations{
			Activities: []domain.Activity{
				{Start: at(0), End: at(1000), DistanceM: 500_000, Mode: "FLYING"},
			},
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(1000), Loc: loc(10.1, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(5000), p)
		if _, ok := j.DivergencePct(); ok {
			t.Error("divergence computable with zero google ground — absence must stay absent")
		}
		if j.DivergenceFlagged() {
			t.Error("flag raised with nothing to compare")
		}
	})

	t.Run("MaxSpeedKmh 0 disables teleport rejection", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(60), Loc: loc(10.001, 70)},
				{Time: at(120), Loc: loc(12.7, 70)},
				{Time: at(180), Loc: loc(10.002, 70)},
			},
		}
		off := p
		off.MaxSpeedKmh = 0
		j := journey.Assemble(obs, t0, at(3600), off)
		if j.RejectedSpeed != 0 {
			t.Errorf("rejected = %d, want 0 with the filter off", j.RejectedSpeed)
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

// Excursion rejection (phase 17, STOP 1b): the iPhone export's fallback
// positions are needles — one place tens of km off, visited and left
// within minutes, with nothing recorded on the way. 0.35° of latitude is
// about 39 km, the measured typical distance.
func TestExcursionRejection(t *testing.T) {
	p := journey.DefaultParams()

	t.Run("a single phantom point out and straight back is rejected", func(t *testing.T) {
		// 39 km in 6 minutes is about 390 km/h: under MaxSpeedKmh, so the
		// teleport pass keeps it; the shape is what gives it away.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(6 * 60), Loc: loc(10.35, 70)},
				{Time: at(12 * 60), Loc: loc(10.001, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.RejectedSpeed != 0 {
			t.Fatalf("teleport pass rejected %d, want 0 (the case it cannot see)", j.RejectedSpeed)
		}
		if j.RejectedExcursion != 1 {
			t.Fatalf("rejected = %d, want 1", j.RejectedExcursion)
		}
		if j.TotalKm > 1 {
			t.Errorf("total = %.1f km, want < 1 (no 78 km needle)", j.TotalKm)
		}
	})

	t.Run("a slow phantom is rejected too — speed does not separate them", func(t *testing.T) {
		// 39 km out over 50 minutes reads as an ordinary 47 km/h drive.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(50 * 60), Loc: loc(10.35, 70)},
				{Time: at(100 * 60), Loc: loc(10.001, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(3*3600), p)
		if j.RejectedExcursion != 1 {
			t.Fatalf("rejected = %d, want 1", j.RejectedExcursion)
		}
	})

	t.Run("a run of phantoms at one place falls together", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(5 * 60), Loc: loc(10.35, 70)},
				{Time: at(10 * 60), Loc: loc(10.35, 70)},
				{Time: at(15 * 60), Loc: loc(10.351, 70)},
				{Time: at(20 * 60), Loc: loc(10.35, 70)},
				{Time: at(25 * 60), Loc: loc(10.002, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.RejectedExcursion != 4 {
			t.Fatalf("rejected = %d, want the whole 4-point run", j.RejectedExcursion)
		}
	})

	t.Run("a real out-and-back with a trail of points is kept", func(t *testing.T) {
		// The same 39 km and back, but recorded along the way: every hop is
		// a few km, so no cluster is far from both its neighbours.
		var pts []domain.PathPoint
		for i := 0; i <= 10; i++ {
			pts = append(pts, domain.PathPoint{Time: at(i * 5 * 60), Loc: loc(10+0.035*float64(i), 70)})
		}
		for i := 9; i >= 0; i-- {
			pts = append(pts, domain.PathPoint{Time: at((20 - i) * 5 * 60), Loc: loc(10+0.035*float64(i), 70)})
		}
		j := journey.Assemble(domain.Observations{Points: pts}, t0, at(3*3600), p)
		if j.RejectedExcursion != 0 {
			t.Errorf("rejected = %d, want 0 (a recorded road out and back)", j.RejectedExcursion)
		}
	})

	t.Run("an out-and-back longer than the window is kept", func(t *testing.T) {
		// Away three hours with nothing recorded on the way: plausibly a
		// real errand, and outside what the rule claims to know.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(90 * 60), Loc: loc(10.35, 70)},
				{Time: at(180 * 60), Loc: loc(10.001, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(4*3600), p)
		if j.RejectedExcursion != 0 {
			t.Errorf("rejected = %d, want 0 (beyond ExcursionMaxMinutes)", j.RejectedExcursion)
		}
	})

	t.Run("going somewhere and staying there is kept", func(t *testing.T) {
		// Out 39 km and on another 39 km: the neighbours are not close
		// together, so this is travel, not a needle.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(30 * 60), Loc: loc(10.35, 70)},
				{Time: at(60 * 60), Loc: loc(10.70, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(2*3600), p)
		if j.RejectedExcursion != 0 {
			t.Errorf("rejected = %d, want 0", j.RejectedExcursion)
		}
	})

	t.Run("a short hop out and back stays under ExcursionMinKm", func(t *testing.T) {
		// About 11 km: the neighbourhood errand, never judged.
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(15 * 60), Loc: loc(10.1, 70)},
				{Time: at(30 * 60), Loc: loc(10.001, 70)},
			},
		}
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.RejectedExcursion != 0 {
			t.Errorf("rejected = %d, want 0", j.RejectedExcursion)
		}
	})

	t.Run("ExcursionMinKm 0 disables excursion rejection", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(6 * 60), Loc: loc(10.35, 70)},
				{Time: at(12 * 60), Loc: loc(10.001, 70)},
			},
		}
		off := p
		off.ExcursionMinKm = 0
		j := journey.Assemble(obs, t0, at(3600), off)
		if j.RejectedExcursion != 0 {
			t.Errorf("rejected = %d, want 0 with the filter off", j.RejectedExcursion)
		}
	})

	t.Run("the observations are not modified", func(t *testing.T) {
		obs := domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(6 * 60), Loc: loc(10.35, 70)},
				{Time: at(12 * 60), Loc: loc(10.001, 70)},
			},
		}
		before := *obs.Points[1].Loc
		j := journey.Assemble(obs, t0, at(3600), p)
		if j.RejectedExcursion != 1 || len(obs.Points) != 3 || *obs.Points[1].Loc != before {
			t.Errorf("observations changed or rule did not fire: %+v", obs.Points)
		}
	})
}
