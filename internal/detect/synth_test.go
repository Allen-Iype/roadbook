package detect

import (
	"math"
	"testing"
	"time"

	"roadbook/internal/domain"
)

// Table tests for stay-point visit synthesis (phase 11 CP1). The pass is pure:
// these tables are its specification. Geometry note: at the test latitude
// (64°N), 0.001° of latitude ≈ 111 m; longitude deltas are scaled accordingly
// in the helpers so distances read as intended.

var synthT0 = time.Date(2026, 4, 10, 18, 0, 0, 0, time.FixedZone("", 0))

func fix(minAfter float64, lat, lon float64) domain.RawPosition {
	return domain.RawPosition{
		Time:   synthT0.Add(time.Duration(minAfter * float64(time.Minute))),
		Loc:    &domain.LatLng{Lat: lat, Lon: lon},
		Source: domain.SourcePhoto,
	}
}

func defaultSynth() SynthParams { return DefaultSynthParams() }

func TestSynthesizeStays(t *testing.T) {
	base := domain.LatLng{Lat: 64.1466, Lon: -21.9426} // Reykjavík-ish; only geometry matters

	cases := []struct {
		name  string
		fixes []domain.RawPosition
		p     SynthParams
		want  []struct {
			startMin, endMin float64 // minutes after synthT0
		}
	}{
		{
			name:  "empty input yields no stays",
			fixes: nil,
			p:     defaultSynth(),
		},
		{
			name:  "single fix has zero duration and never qualifies",
			fixes: []domain.RawPosition{fix(0, base.Lat, base.Lon)},
			p:     defaultSynth(),
		},
		{
			name: "burst within radius spanning the minimum qualifies",
			fixes: []domain.RawPosition{
				fix(0, base.Lat, base.Lon),
				fix(20, base.Lat+0.0005, base.Lon), // ~55 m north
				fix(40, base.Lat, base.Lon+0.001),  // ~49 m east at 64°N
			},
			p: defaultSynth(),
			want: []struct{ startMin, endMin float64 }{
				{0, 40},
			},
		},
		{
			name: "burst under the minimum duration is discarded",
			fixes: []domain.RawPosition{
				fix(0, base.Lat, base.Lon),
				fix(10, base.Lat+0.0005, base.Lon),
				fix(20, base.Lat, base.Lon),
			},
			p: defaultSynth(), // 20 min < StayMinMin 30
		},
		{
			name: "a fix outside the radius closes the stay and opens another",
			fixes: []domain.RawPosition{
				fix(0, base.Lat, base.Lon),
				fix(35, base.Lat, base.Lon),
				fix(40, base.Lat+0.01, base.Lon), // ~1.1 km away
				fix(80, base.Lat+0.01, base.Lon),
			},
			p: defaultSynth(),
			want: []struct{ startMin, endMin float64 }{
				{0, 35},
				{40, 80},
			},
		},
		{
			name: "a time gap above the maximum splits an otherwise-continuous place",
			fixes: []domain.RawPosition{
				fix(0, base.Lat, base.Lon),
				fix(35, base.Lat, base.Lon),
				fix(35+300, base.Lat, base.Lon), // 5 h gap > StayMaxGapMin 240
				fix(35+340, base.Lat, base.Lon),
			},
			p: defaultSynth(),
			want: []struct{ startMin, endMin float64 }{
				{0, 35},
				{335, 375},
			},
		},
		{
			name: "unsorted input is sorted before the pass",
			fixes: []domain.RawPosition{
				fix(40, base.Lat, base.Lon),
				fix(0, base.Lat, base.Lon),
				fix(20, base.Lat, base.Lon),
			},
			p: defaultSynth(),
			want: []struct{ startMin, endMin float64 }{
				{0, 40},
			},
		},
		{
			name: "nil locations are skipped",
			fixes: []domain.RawPosition{
				{Time: synthT0, Source: domain.SourcePhoto}, // no Loc
				fix(0, base.Lat, base.Lon),
				fix(35, base.Lat, base.Lon),
			},
			p: defaultSynth(),
			want: []struct{ startMin, endMin float64 }{
				{0, 35},
			},
		},
		{
			name: "running-mean centroid anchors against slow drift",
			// Nine fixes each stepping ~55 m: consecutive fixes are always
			// near each other, but the walk leaves the first fix ~500 m
			// behind. The running mean must eventually refuse — one stay
			// must not stretch over the whole drift.
			fixes: []domain.RawPosition{
				fix(0, base.Lat+0.0000, base.Lon),
				fix(10, base.Lat+0.0005, base.Lon),
				fix(20, base.Lat+0.0010, base.Lon),
				fix(30, base.Lat+0.0015, base.Lon),
				fix(40, base.Lat+0.0020, base.Lon),
				fix(50, base.Lat+0.0025, base.Lon),
				fix(60, base.Lat+0.0030, base.Lon),
				fix(70, base.Lat+0.0035, base.Lon),
				fix(80, base.Lat+0.0040, base.Lon),
			},
			p: defaultSynth(),
			want: []struct{ startMin, endMin float64 }{
				// With radius 200 m the running mean trails the walk; the
				// first stay covers the fixes it can hold, the remainder
				// forms a second. The exact split is pinned: fix 7 (~390 m
				// from the mean of fixes 0–6) is the first refusal.
				{0, 60},
				{70, 80}, // 10 min < StayMinMin — discarded below
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := synthesizeStays(tc.fixes, tc.p)
			// Drop expected stays that don't clear the minimum duration —
			// the table states geometry; qualification is the pass's job.
			var want []struct{ startMin, endMin float64 }
			for _, w := range tc.want {
				if w.endMin-w.startMin >= tc.p.StayMinMin {
					want = append(want, w)
				}
			}
			if len(got) != len(want) {
				t.Fatalf("got %d stays, want %d: %+v", len(got), len(want), got)
			}
			for i, w := range want {
				ws, we := synthT0.Add(time.Duration(w.startMin*float64(time.Minute))), synthT0.Add(time.Duration(w.endMin*float64(time.Minute)))
				if !got[i].Start.Equal(ws) || !got[i].End.Equal(we) {
					t.Errorf("stay %d: got [%v, %v], want [%v, %v]", i, got[i].Start, got[i].End, ws, we)
				}
				if got[i].Loc == nil {
					t.Fatalf("stay %d: nil centroid", i)
				}
				if got[i].SemanticType != "" {
					t.Errorf("stay %d: synthetic visit must not assert a semantic type, got %q", i, got[i].SemanticType)
				}
			}
		})
	}
}

func TestSynthesizeStaysCentroidIsRunningMean(t *testing.T) {
	// Three fixes; centroid must be the arithmetic mean of all member
	// coordinates, not the first fix.
	f := []domain.RawPosition{
		fix(0, 64.1000, -21.9000),
		fix(20, 64.1010, -21.9000),
		fix(40, 64.1020, -21.9000),
	}
	got := synthesizeStays(f, defaultSynth())
	if len(got) != 1 {
		t.Fatalf("got %d stays, want 1", len(got))
	}
	if math.Abs(got[0].Loc.Lat-64.1010) > 1e-9 || math.Abs(got[0].Loc.Lon-(-21.9000)) > 1e-9 {
		t.Errorf("centroid = %v, want mean (64.1010, -21.9000)", *got[0].Loc)
	}
}

func TestPhotoFixSelection(t *testing.T) {
	loc := &domain.LatLng{Lat: 1, Lon: 1}
	raws := []domain.RawPosition{
		{Time: synthT0, Loc: loc, Source: domain.SourcePhoto},
		{Time: synthT0, Loc: loc, Source: "WIFI"},
		{Time: synthT0, Loc: loc, Source: "GPS"},
		{Time: synthT0, Loc: nil, Source: domain.SourcePhoto}, // located only
		{Time: synthT0, Loc: loc, Source: ""},
	}
	got := photoFixes(raws)
	if len(got) != 1 {
		t.Fatalf("got %d photo fixes, want 1 (source-scoped, located only)", len(got))
	}
}

func TestMergeVisitsByStart(t *testing.T) {
	v := func(min float64, sem string) domain.Visit {
		return domain.Visit{Start: synthT0.Add(time.Duration(min * float64(time.Minute))),
			End: synthT0.Add(time.Duration((min + 10) * float64(time.Minute))), SemanticType: sem}
	}
	a := []domain.Visit{v(0, "INFERRED_HOME"), v(60, "UNKNOWN")}
	b := []domain.Visit{v(30, ""), v(60, "")}
	got := mergeVisitsByStart(a, b)
	if len(got) != 4 {
		t.Fatalf("got %d visits, want 4", len(got))
	}
	order := []string{"INFERRED_HOME", "", "UNKNOWN", ""}
	for i, want := range order {
		if got[i].SemanticType != want {
			t.Errorf("merged[%d].SemanticType = %q, want %q (real visits win ties)", i, got[i].SemanticType, want)
		}
	}
	// Inputs untouched (invariant 2).
	if len(a) != 2 || len(b) != 2 {
		t.Error("merge mutated an input slice")
	}
}

func TestDeriveSyntheticBases(t *testing.T) {
	home := domain.LatLng{Lat: 64.1466, Lon: -21.9426}
	away := domain.LatLng{Lat: 64.2539, Lon: -15.2082} // ~330 km east

	stay := func(day int, at domain.LatLng) domain.Visit {
		s := time.Date(2026, 4, day, 19, 0, 0, 0, time.UTC)
		return domain.Visit{Start: s, End: s.Add(45 * time.Minute), Loc: &at}
	}

	p := DefaultParams()

	t.Run("recurring cluster across enough days becomes a base", func(t *testing.T) {
		var stays []domain.Visit
		for d := 1; d <= 9; d++ { // 9 distinct days ≥ HomeMinDays 8
			stays = append(stays, stay(d, home))
		}
		stays = append(stays, stay(10, away), stay(11, away), stay(12, away))
		bases := deriveSyntheticBases(stays, p)
		if len(bases) != 1 {
			t.Fatalf("got %d bases, want 1", len(bases))
		}
		if bases[0].N != 9 {
			t.Errorf("base N = %d, want 9", bases[0].N)
		}
	})

	t.Run("recurrence without day spread is not a home", func(t *testing.T) {
		var stays []domain.Visit
		for h := 8; h < 20; h++ { // 12 stays, all on one day — a hotel, not a home
			s := time.Date(2026, 4, 5, h, 0, 0, 0, time.UTC)
			stays = append(stays, domain.Visit{Start: s, End: s.Add(45 * time.Minute), Loc: &home})
		}
		if bases := deriveSyntheticBases(stays, p); len(bases) != 0 {
			t.Fatalf("got %d bases, want 0 (HomeMinDays guard)", len(bases))
		}
	})
}
