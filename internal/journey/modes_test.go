package journey

import (
	"reflect"
	"testing"
	"time"

	"roadbook/internal/domain"
)

func TestModeBreakdown(t *testing.T) {
	t0 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	at := func(h int) time.Time { return t0.Add(time.Duration(h) * time.Hour) }
	act := func(startH, endH int, mode string, km float64) domain.Activity {
		return domain.Activity{Start: at(startH), End: at(endH), Mode: mode, DistanceM: km * 1000}
	}

	cases := []struct {
		name     string
		acts     []domain.Activity
		from, to time.Time
		want     []ModeKm
	}{
		{
			name: "sums by mode, km-descending",
			acts: []domain.Activity{
				act(1, 2, "IN_BUS", 50),
				act(3, 4, "IN_PASSENGER_VEHICLE", 120),
				act(5, 6, "IN_BUS", 30),
			},
			from: at(0), to: at(10),
			want: []ModeKm{{"IN_PASSENGER_VEHICLE", 120}, {"IN_BUS", 80}},
		},
		{
			name: "overlap in full, never pro-rated; outside dropped",
			acts: []domain.Activity{
				act(-2, 1, "FLYING", 500),  // straddles the start: counts whole
				act(9, 12, "IN_BUS", 40),   // straddles the end: counts whole
				act(-5, -3, "WALKING", 10), // entirely before: dropped
				act(11, 13, "WALKING", 10), // entirely after: dropped
			},
			from: at(0), to: at(10),
			want: []ModeKm{{"FLYING", 500}, {"IN_BUS", 40}},
		},
		{
			name: "zero distance contributes nothing; empty mode buckets as UNKNOWN",
			acts: []domain.Activity{
				act(1, 2, "CYCLING", 0),
				act(3, 4, "", 15),
			},
			from: at(0), to: at(10),
			want: []ModeKm{{"UNKNOWN", 15}},
		},
		{
			name: "equal km ties break by mode name",
			acts: []domain.Activity{
				act(1, 2, "IN_TRAIN", 25),
				act(3, 4, "IN_BUS", 25),
			},
			from: at(0), to: at(10),
			want: []ModeKm{{"IN_BUS", 25}, {"IN_TRAIN", 25}},
		},
		{
			name: "no activities (a photo-sourced journey): empty, not zeros",
			acts: nil,
			from: at(0), to: at(10),
			want: []ModeKm{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ModeBreakdown(tc.acts, tc.from, tc.to)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
