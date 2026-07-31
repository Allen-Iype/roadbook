package detect

import (
	"testing"
	"time"

	"roadbook/internal/domain"
)

// All data in this file is synthetic. Coordinates are fabricated (a "home" at
// 10°N 20°E); nothing here derives from any real export. The real-data
// regression tests live in regression_test.go and skip when data/ is absent.
//
// Scale cheat-sheet for reading the cases: 1° of latitude ≈ 111.2 km. NEAR is
// 25 km (≈0.225°), FAR is 100 km (≈0.9°), so latitude 11.0 is comfortably
// "far" from home at 10.0, and 10.1 is comfortably "near".

func at(day, hour, min int) time.Time {
	return time.Date(2025, 1, day, hour, min, 0, 0, time.UTC)
}

func loc(lat, lon float64) *domain.LatLng { return &domain.LatLng{Lat: lat, Lon: lon} }

func homeVisit(day int, l *domain.LatLng) domain.Visit {
	return domain.Visit{Start: at(day, 8, 0), End: at(day, 9, 0), Loc: l, SemanticType: "INFERRED_HOME"}
}

// homeDays emits one INFERRED_HOME visit per day — eight is the clustering
// minimum for a base to exist.
func homeDays(from, to int, l *domain.LatLng) []domain.Visit {
	var vs []domain.Visit
	for d := from; d <= to; d++ {
		vs = append(vs, homeVisit(d, l))
	}
	return vs
}

// trip is a compact away-journey: a dwelt visit at dest starting on day at
// 10:00 for dwellMin minutes, plus four path points near dest at 10:30–12:00.
// With the visit that is five observations spanning two hours — enough to pass
// MIN_OBS and MIN_HRS.
func trip(day int, dest *domain.LatLng, dwellMin int) ([]domain.Visit, []domain.PathPoint) {
	v := []domain.Visit{{
		Start:        at(day, 10, 0),
		End:          at(day, 10, 0).Add(time.Duration(dwellMin) * time.Minute),
		Loc:          dest,
		SemanticType: "UNKNOWN",
	}}
	var pts []domain.PathPoint
	for i := range 4 {
		pts = append(pts, domain.PathPoint{
			Time: at(day, 10, 30+i*30),
			Loc:  loc(dest.Lat+0.001, dest.Lon),
		})
	}
	return v, pts
}

func TestRun(t *testing.T) {
	home := loc(10, 20)

	cases := []struct {
		name         string
		obs          domain.Observations
		wantCands    int
		wantOutliers int // -1: don't check
		check        func(t *testing.T, r Result)
	}{
		{
			name:         "quiet weeks at home produce nothing",
			obs:          domain.Observations{Visits: homeDays(1, 10, home)},
			wantCands:    0,
			wantOutliers: 0,
		},
		{
			name: "far trip with a dwelt destination is one candidate",
			obs: func() domain.Observations {
				tv, tp := trip(10, loc(11, 20), 600)
				visits := append(homeDays(1, 8, home), tv...)
				visits = append(visits, homeVisit(11, home))
				return domain.Observations{Visits: visits, Points: tp}
			}(),
			wantCands:    1,
			wantOutliers: 0,
			check: func(t *testing.T, r Result) {
				c := r.Candidates[0]
				if c.DestKm != 111 { // 1° of latitude
					t.Errorf("DestKm = %d, want 111", c.DestKm)
				}
				if c.Stops != 1 {
					t.Errorf("Stops = %d, want 1", c.Stops)
				}
				if c.ObsCount != 5 {
					t.Errorf("ObsCount = %d, want 5", c.ObsCount)
				}
				if c.StartTruncated || c.EndTruncated {
					t.Errorf("unexpected truncation flags: start=%v end=%v", c.StartTruncated, c.EndTruncated)
				}
			},
		},
		{
			name: "passing through without dwelling is not an adventure (bug 3)",
			obs: func() domain.Observations {
				tv, tp := trip(10, loc(11, 20), 30) // 30 min < MIN_DWELL_MIN
				visits := append(homeDays(1, 8, home), tv...)
				visits = append(visits, homeVisit(11, home))
				return domain.Observations{Visits: visits, Points: tp}
			}(),
			wantCands:    0,
			wantOutliers: 0,
		},
		{
			name: "a dwelt destination inside FAR_KM is not far enough",
			obs: func() domain.Observations {
				tv, tp := trip(10, loc(10.5, 20), 600) // ≈56 km: away, but not far
				visits := append(homeDays(1, 8, home), tv...)
				visits = append(visits, homeVisit(11, home))
				return domain.Observations{Visits: visits, Points: tp}
			}(),
			wantCands:    0,
			wantOutliers: 0,
		},
		{
			name: "fewer than MIN_OBS observations rejects the span",
			obs: func() domain.Observations {
				tv, tp := trip(10, loc(11, 20), 600)
				visits := append(homeDays(1, 8, home), tv...)
				visits = append(visits, homeVisit(11, home))
				return domain.Observations{Visits: visits, Points: tp[:1]} // visit + 1 point = 2 obs
			}(),
			wantCands:    0,
			wantOutliers: 0,
		},
		{
			name: "a span shorter than MIN_HRS rejects the span",
			obs: func() domain.Observations {
				// Dwelt visit qualifies (90 min) but the last away observation
				// is only 40 minutes after the first.
				visits := append(homeDays(1, 8, home), domain.Visit{
					Start: at(10, 10, 0), End: at(10, 11, 30), Loc: loc(11, 20), SemanticType: "UNKNOWN",
				})
				visits = append(visits, homeVisit(11, home))
				var pts []domain.PathPoint
				for i := range 4 {
					pts = append(pts, domain.PathPoint{Time: at(10, 10, 10+i*10), Loc: loc(11.001, 20)})
				}
				return domain.Observations{Visits: visits, Points: pts}
			}(),
			wantCands:    0,
			wantOutliers: 0,
		},
		{
			name: "impossible from both neighbours is dropped as an outlier",
			obs: func() domain.Observations {
				// The closing home visit matters: an observation at the very
				// edge of the window has only one neighbour, and being fast
				// from all one of them would drop it too.
				visits := append(homeDays(1, 8, home), homeVisit(11, home))
				pts := []domain.PathPoint{
					{Time: at(10, 10, 0), Loc: loc(10.001, 20)},
					{Time: at(10, 10, 30), Loc: loc(30, 20)}, // ≈2224 km in 30 min, both directions
					{Time: at(10, 11, 0), Loc: loc(10.002, 20)},
				}
				return domain.Observations{Visits: visits, Points: pts}
			}(),
			wantCands:    0,
			wantOutliers: 1,
		},
		{
			name: "fast from one side only is kept — real flights look like this",
			obs: func() domain.Observations {
				visits := homeDays(1, 8, home)
				pts := []domain.PathPoint{
					{Time: at(10, 10, 0), Loc: loc(10.001, 20)},
					{Time: at(10, 10, 30), Loc: loc(14.1, 20)}, // ≈912 km/h from previous, slow to next
					{Time: at(10, 11, 30), Loc: loc(14.2, 20)},
				}
				return domain.Observations{Visits: visits, Points: pts}
			}(),
			wantCands:    0, // span exists but has no dwelt destination
			wantOutliers: 0,
		},
		{
			name: "home is a set: near either concurrent base is not away (bug 2)",
			obs: func() domain.Observations {
				second := loc(12, 20) // ≈222 km from the first base
				var visits []domain.Visit
				for d := 1; d <= 8; d++ { // interleaved so both eras are concurrent
					visits = append(visits, homeVisit(d, home))
					visits = append(visits, domain.Visit{
						Start: at(d, 10, 0), End: at(d, 11, 0), Loc: second, SemanticType: "INFERRED_HOME",
					})
				}
				pts := []domain.PathPoint{{Time: at(5, 12, 0), Loc: loc(12.01, 20)}} // ≈1 km from second base
				return domain.Observations{Visits: visits, Points: pts}
			}(),
			wantCands:    0,
			wantOutliers: 0,
			check: func(t *testing.T, r Result) {
				if len(r.Bases) != 2 {
					t.Errorf("bases = %d, want 2", len(r.Bases))
				}
			},
		},
		{
			name: "the corridor between two homes is transit, not adventure (bug 3)",
			obs: func() domain.Observations {
				far := loc(14, 20) // ≈445 km away; midpoint is >100 km from both
				var visits []domain.Visit
				for d := 1; d <= 8; d++ {
					visits = append(visits, homeVisit(d, home))
					visits = append(visits, domain.Visit{
						Start: at(d, 18, 0), End: at(d, 19, 0), Loc: far, SemanticType: "INFERRED_HOME",
					})
				}
				pts := []domain.PathPoint{ // driving between them, never dwelling
					{Time: at(5, 10, 0), Loc: loc(11, 20)},
					{Time: at(5, 11, 0), Loc: loc(12, 20)},
					{Time: at(5, 12, 0), Loc: loc(12.5, 20)},
					{Time: at(5, 13, 0), Loc: loc(13, 20)},
					{Time: at(5, 14, 0), Loc: loc(13.5, 20)},
				}
				return domain.Observations{Visits: visits, Points: pts}
			}(),
			wantCands:    0,
			wantOutliers: 0,
		},
		{
			name: "outside every era, distance falls back to all bases",
			obs: func() domain.Observations {
				// Base era ends day 8 + 45; day 100 is beyond it, yet the trip
				// must still measure against the only home that ever existed.
				tv, tp := trip(100, loc(11, 20), 600)
				visits := append(homeDays(1, 8, home), tv...)
				visits = append(visits, domain.Visit{
					Start: at(101, 8, 0), End: at(101, 9, 0), Loc: home, SemanticType: "UNKNOWN",
				})
				return domain.Observations{Visits: visits, Points: tp}
			}(),
			wantCands:    1,
			wantOutliers: 0,
		},
		{
			name: "a window starting mid-journey marks the candidate start-truncated (bug 4)",
			obs: func() domain.Observations {
				tv, tp := trip(1, loc(11, 20), 600) // the very first observations are away
				visits := append(tv, homeDays(3, 10, home)...)
				return domain.Observations{Visits: visits, Points: tp}
			}(),
			wantCands:    1,
			wantOutliers: 0,
			check: func(t *testing.T, r Result) {
				c := r.Candidates[0]
				if !c.StartTruncated {
					t.Error("StartTruncated = false, want true")
				}
				if c.EndTruncated {
					t.Error("EndTruncated = true, want false")
				}
			},
		},
		{
			name: "a journey still in progress at the window edge is end-truncated (bug 4)",
			obs: func() domain.Observations {
				tv, tp := trip(10, loc(11, 20), 600) // never returns home
				visits := append(homeDays(1, 8, home), tv...)
				return domain.Observations{Visits: visits, Points: tp}
			}(),
			wantCands:    1,
			wantOutliers: 0,
			check: func(t *testing.T, r Result) {
				if !r.Candidates[0].EndTruncated {
					t.Error("EndTruncated = false, want true")
				}
			},
		},
		{
			name: "a second trip to the same destination counts as a repeat",
			obs: func() domain.Observations {
				t1v, t1p := trip(10, loc(11, 20), 600)
				t2v, t2p := trip(20, loc(11.001, 20), 600)
				visits := append(homeDays(1, 8, home), t1v...)
				visits = append(visits, homeVisit(11, home))
				visits = append(visits, t2v...)
				visits = append(visits, homeVisit(21, home))
				return domain.Observations{Visits: visits, Points: append(t1p, t2p...)}
			}(),
			wantCands:    2,
			wantOutliers: 0,
			check: func(t *testing.T, r Result) {
				if r.Candidates[0].Repeat != 0 || r.Candidates[1].Repeat != 1 {
					t.Errorf("Repeat = %d,%d, want 0,1", r.Candidates[0].Repeat, r.Candidates[1].Repeat)
				}
			},
		},
		{
			name: "no derivable home base yields no candidates rather than a crash",
			obs: domain.Observations{Points: []domain.PathPoint{
				{Time: at(1, 10, 0), Loc: loc(11, 20)},
				{Time: at(1, 11, 0), Loc: loc(11.1, 20)},
			}},
			wantCands:    0,
			wantOutliers: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Run(tc.obs, DefaultParams())
			if len(r.Candidates) != tc.wantCands {
				t.Fatalf("candidates = %d, want %d", len(r.Candidates), tc.wantCands)
			}
			if tc.wantOutliers >= 0 && r.OutliersDropped != tc.wantOutliers {
				t.Errorf("outliers dropped = %d, want %d", r.OutliersDropped, tc.wantOutliers)
			}
			if tc.check != nil {
				tc.check(t, r)
			}
		})
	}
}
