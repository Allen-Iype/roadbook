package journey

import (
	"math"
	"os"
	"testing"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/timeline"
)

// Fixture builders: timestamps as RFC 3339 strings so each carries its own
// offset, exactly as the export and the store hand them over.
func ts(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func obsLeg(start, end string, km float64, points int) Leg {
	pts := make([]TimedPoint, points)
	for i := range pts {
		t := ts(end)
		if i == 0 {
			t = ts(start)
		}
		pts[i] = TimedPoint{Time: t, Loc: domain.LatLng{Lat: 65, Lon: -21}}
	}
	return Leg{Kind: LegObserved, Points: pts, DistanceKm: km}
}

func gapLegAt(kind GapKind, start, end string, km float64) Leg {
	return Leg{Kind: LegGap, GapKind: kind, DistanceKm: km,
		Points: []TimedPoint{{Time: ts(start)}, {Time: ts(end)}}}
}

func stopAt(start, end string) Stop { return Stop{Start: ts(start), End: ts(end)} }

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// The civil-day rule, ported case for case from the web's sliceDays tests
// (web/lib/slice-days.test.ts) so the served count and the narrative's day
// count cannot drift: the parity test on the web side pins the other
// direction.
func TestCivilDays(t *testing.T) {
	cases := []struct {
		name string
		j    Journey
		want int
	}{
		{
			// The acceptance journey's shape: Friday evening departure,
			// a dwell over Saturday, Sunday return — three civil days.
			name: "Westfjords: Fri 20:00 → Sun 10:00 is three days",
			j: Journey{
				WindowStart: ts("2026-05-22T20:00:00+00:00"), WindowEnd: ts("2026-05-24T10:00:00+00:00"),
				Stops: []Stop{stopAt("2026-05-22T22:00:00+00:00", "2026-05-24T08:00:00+00:00")},
			},
			want: 3,
		},
		{
			name: "a transit crossing midnight touches two days",
			j: Journey{
				WindowStart: ts("2026-05-01T22:00:00+00:00"), WindowEnd: ts("2026-05-02T02:00:00+00:00"),
				Legs: []Leg{obsLeg("2026-05-01T22:00:00+00:00", "2026-05-02T02:00:00+00:00", 100, 5)},
			},
			want: 2,
		},
		{
			name: "ending exactly at 00:00 is the next civil day",
			j: Journey{
				WindowStart: ts("2026-05-01T22:00:00+00:00"), WindowEnd: ts("2026-05-02T00:00:00+00:00"),
			},
			want: 2,
		},
		{
			// 23:30 UTC on the 1st is 05:00 on the 2nd at +05:30; the
			// window ends the same evening local time: one day, not two.
			name: "civil days follow the recorded offset, not UTC",
			j: Journey{
				WindowStart: ts("2026-05-02T05:00:00+05:30"), WindowEnd: ts("2026-05-02T21:00:00+05:30"),
			},
			want: 1,
		},
		{
			// A westward overnight flight ends on an EARLIER civil date
			// than its departure; the range still runs min to max.
			name: "westward flight: range is min to max of civil dates",
			j: Journey{
				WindowStart: ts("2026-05-02T01:00:00+09:00"), WindowEnd: ts("2026-05-01T20:00:00-07:00"),
				Legs: []Leg{gapLegAt(GapAir, "2026-05-02T01:00:00+09:00", "2026-05-01T20:00:00-07:00", 9000)},
			},
			want: 2,
		},
		{
			// A halt between two activities that straddle the window can
			// reach past its edges; the day range includes it, as the
			// narrative's does.
			name: "a stop reaching past the window edge extends the range",
			j: Journey{
				WindowStart: ts("2026-05-02T08:00:00+00:00"), WindowEnd: ts("2026-05-02T18:00:00+00:00"),
				Stops: []Stop{stopAt("2026-05-01T23:00:00+00:00", "2026-05-02T09:00:00+00:00")},
			},
			want: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := civilDays(tc.j); got != tc.want {
				t.Errorf("civilDays = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	j := Journey{
		WindowStart: ts("2026-05-22T20:00:00+00:00"), WindowEnd: ts("2026-05-24T10:00:00+00:00"),
		Legs: []Leg{
			obsLeg("2026-05-22T20:00:00+00:00", "2026-05-22T22:00:00+00:00", 120, 30), // 60 km/h
			gapLegAt(GapRoad, "2026-05-22T22:00:00+00:00", "2026-05-24T08:00:00+00:00", 5),
			obsLeg("2026-05-24T08:00:00+00:00", "2026-05-24T08:00:00+00:00", 0, 1),   // a fix: no duration
			obsLeg("2026-05-24T09:00:00+00:00", "2026-05-24T10:00:00+00:00", 40, 12), // 40 km/h
		},
		Stops: []Stop{
			stopAt("2026-05-22T22:00:00+00:00", "2026-05-24T08:00:00+00:00"), // 34 h
			stopAt("2026-05-24T08:00:00+00:00", "2026-05-24T09:00:00+00:00"), // 1 h
		},
	}
	s := Summarize(j)
	if !near(s.SpanHours, 38) {
		t.Errorf("SpanHours = %v, want 38", s.SpanHours)
	}
	if s.CivilDays != 3 {
		t.Errorf("CivilDays = %d, want 3", s.CivilDays)
	}
	if s.Stops != 2 || !near(s.DwellHours, 35) {
		t.Errorf("Stops/DwellHours = %d/%v, want 2/35", s.Stops, s.DwellHours)
	}
	// Pace: 160 km over 3 observed hours; the fix and the road gap
	// contribute nothing to either sum.
	if !s.PaceOK || !near(s.ObservedHours, 3) || !near(s.ObservedPaceKmh, 160.0/3) {
		t.Errorf("pace = %v over %v h (ok %v), want %v over 3 h", s.ObservedPaceKmh, s.ObservedHours, s.PaceOK, 160.0/3)
	}

	// A fix-only journey has no pace: absent, never zero. The second leg
	// is the demo's Westfjords case — a stationary one-minute run with two
	// points and zero km, which is a fix, not a stretch, and must not yield
	// "0 km/h".
	fixOnly := Journey{
		WindowStart: j.WindowStart, WindowEnd: j.WindowEnd,
		Legs: []Leg{
			obsLeg("2026-05-23T08:00:00+00:00", "2026-05-23T08:00:00+00:00", 0, 1),
			obsLeg("2026-05-24T13:00:00+00:00", "2026-05-24T13:01:00+00:00", 0, 2),
		},
	}
	if fs := Summarize(fixOnly); fs.PaceOK || fs.ObservedPaceKmh != 0 {
		t.Errorf("fix-only pace = %v (ok %v), want absent", fs.ObservedPaceKmh, fs.PaceOK)
	}
}

// The golden overnight bus journey pins the summary against real,
// anonymised data: the window runs 19:46 to 07:12 next morning at +05:30 —
// two civil days — and every observed leg has a duration.
func TestSummarizeGolden(t *testing.T) {
	f, err := os.Open("../../testdata/journey-27jul2026.anon.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	obs, _, err := timeline.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	j := Assemble(obs, ts("2026-07-27T19:46:35+05:30"), ts("2026-07-28T07:12:22+05:30"), DefaultParams())
	s := Summarize(j)
	if s.CivilDays != 2 {
		t.Errorf("golden CivilDays = %d, want 2", s.CivilDays)
	}
	if !near(s.SpanHours, ts("2026-07-28T07:12:22+05:30").Sub(ts("2026-07-27T19:46:35+05:30")).Hours()) {
		t.Errorf("golden SpanHours = %v", s.SpanHours)
	}
	if !s.PaceOK || s.ObservedPaceKmh <= 0 {
		t.Errorf("golden pace absent: %+v", s)
	}
}
