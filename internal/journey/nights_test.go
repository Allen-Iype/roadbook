package journey

import (
	"reflect"
	"testing"
)

func TestNights(t *testing.T) {
	p := DefaultNightParams()
	type want struct {
		date  string
		kind  NightKind
		stop  int
		hours float64
	}
	cases := []struct {
		name string
		j    Journey
		want []want
	}{
		{
			name: "a day trip has no nights",
			j: Journey{
				WindowStart: ts("2026-05-22T08:00:00+00:00"), WindowEnd: ts("2026-05-22T20:00:00+00:00"),
			},
		},
		{
			name: "one stop across the whole window holds the night for eight hours",
			j: Journey{
				WindowStart: ts("2026-05-22T09:00:00+00:00"), WindowEnd: ts("2026-05-23T18:00:00+00:00"),
				Stops: []Stop{stopAt("2026-05-22T19:00:00+00:00", "2026-05-23T08:00:00+00:00")},
			},
			want: []want{{"2026-05-22", NightStop, 0, 8}},
		},
		{
			name: "a late arrival still holds the night: 02:00 to 09:00 is four hours of it",
			j: Journey{
				WindowStart: ts("2026-05-22T09:00:00+00:00"), WindowEnd: ts("2026-05-23T18:00:00+00:00"),
				Stops: []Stop{stopAt("2026-05-23T02:00:00+00:00", "2026-05-23T09:00:00+00:00")},
			},
			want: []want{{"2026-05-22", NightStop, 0, 4}},
		},
		{
			name: "exactly the minimum holds; a minute under does not",
			j: Journey{
				WindowStart: ts("2026-05-22T09:00:00+00:00"), WindowEnd: ts("2026-05-24T18:00:00+00:00"),
				Stops: []Stop{
					stopAt("2026-05-22T20:00:00+00:00", "2026-05-23T01:00:00+00:00"), // 22:00–01:00 = 3 h
					stopAt("2026-05-23T20:00:00+00:00", "2026-05-24T00:59:00+00:00"), // 2 h 59
				},
			},
			want: []want{{"2026-05-22", NightStop, 0, 3}, {"2026-05-23", NightUnknown, -1, 0}},
		},
		{
			name: "two stops in one window: the one covering more holds it",
			j: Journey{
				WindowStart: ts("2026-05-22T09:00:00+00:00"), WindowEnd: ts("2026-05-23T18:00:00+00:00"),
				Stops: []Stop{
					stopAt("2026-05-22T21:00:00+00:00", "2026-05-23T01:00:00+00:00"), // 3 h
					stopAt("2026-05-23T01:30:00+00:00", "2026-05-23T07:00:00+00:00"), // 4.5 h
				},
			},
			want: []want{{"2026-05-22", NightStop, 1, 4.5}},
		},
		{
			name: "a half-hour halt across midnight is not a night",
			j: Journey{
				WindowStart: ts("2026-05-22T09:00:00+00:00"), WindowEnd: ts("2026-05-23T18:00:00+00:00"),
				Stops: []Stop{stopAt("2026-05-22T23:45:00+00:00", "2026-05-23T00:15:00+00:00")},
			},
			want: []want{{"2026-05-22", NightUnknown, -1, 0}},
		},
		{
			name: "an overnight bus: recorded travel through the window is a night on the move",
			j: Journey{
				WindowStart: ts("2026-05-22T21:00:00+00:00"), WindowEnd: ts("2026-05-23T09:00:00+00:00"),
				Legs: []Leg{obsLeg("2026-05-22T21:00:00+00:00", "2026-05-23T05:00:00+00:00", 420, 30)},
			},
			want: []want{{"2026-05-22", NightMoving, -1, 7}},
		},
		{
			name: "a stationary recorded run and a gap are not travel",
			j: Journey{
				WindowStart: ts("2026-05-22T21:00:00+00:00"), WindowEnd: ts("2026-05-23T09:00:00+00:00"),
				Legs: []Leg{
					obsLeg("2026-05-22T21:00:00+00:00", "2026-05-23T01:00:00+00:00", 0, 2),
					gapLegAt(GapUnknown, "2026-05-23T01:00:00+00:00", "2026-05-23T06:00:00+00:00", 300),
				},
			},
			want: []want{{"2026-05-22", NightUnknown, -1, 0}},
		},
		{
			name: "a stop wins over travel when both reach the minimum",
			j: Journey{
				WindowStart: ts("2026-05-22T09:00:00+00:00"), WindowEnd: ts("2026-05-23T18:00:00+00:00"),
				Legs:  []Leg{obsLeg("2026-05-22T22:00:00+00:00", "2026-05-23T02:00:00+00:00", 200, 10)},
				Stops: []Stop{stopAt("2026-05-23T02:00:00+00:00", "2026-05-23T08:00:00+00:00")},
			},
			want: []want{{"2026-05-22", NightStop, 0, 4}},
		},
		{
			name: "one stop over three days holds both nights",
			j: Journey{
				WindowStart: ts("2026-05-22T20:00:00+00:00"), WindowEnd: ts("2026-05-24T10:00:00+00:00"),
				Stops: []Stop{stopAt("2026-05-22T22:00:00+00:00", "2026-05-24T08:00:00+00:00")},
			},
			want: []want{{"2026-05-22", NightStop, 0, 8}, {"2026-05-23", NightStop, 0, 8}},
		},
		{
			// The window is read on the traveller's clock, not UTC: 22:00
			// at +05:30 is 16:30 UTC. Read in UTC this stop would cover
			// two hours of the window and lose the night.
			name: "the window is on the journey's own offset",
			j: Journey{
				WindowStart: ts("2026-05-22T09:00:00+05:30"), WindowEnd: ts("2026-05-23T18:00:00+05:30"),
				Stops: []Stop{stopAt("2026-05-22T21:00:00+05:30", "2026-05-23T05:30:00+05:30")},
			},
			want: []want{{"2026-05-22", NightStop, 0, 7.5}},
		},
		{
			// A flight east on day two: the first night is read at +00:00,
			// the second at +05:30, where the traveller then was.
			name: "after a change of offset, later nights use the new clock",
			j: Journey{
				WindowStart: ts("2026-05-22T09:00:00+00:00"), WindowEnd: ts("2026-05-24T12:00:00+05:30"),
				Stops: []Stop{
					stopAt("2026-05-22T22:00:00+00:00", "2026-05-23T06:00:00+00:00"),
					stopAt("2026-05-23T22:00:00+05:30", "2026-05-24T06:00:00+05:30"),
				},
				Legs: []Leg{obsLeg("2026-05-23T08:00:00+00:00", "2026-05-23T20:00:00+05:30", 7000, 2)},
			},
			want: []want{{"2026-05-22", NightStop, 0, 8}, {"2026-05-23", NightStop, 1, 8}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Nights(c.j, p)
			var have []want
			for _, n := range got {
				have = append(have, want{n.Date, n.Kind, n.StopIndex, n.Hours})
			}
			if !reflect.DeepEqual(have, c.want) {
				t.Errorf("nights = %+v, want %+v", have, c.want)
			}
			if days := Summarize(c.j).CivilDays; len(got) != days-1 {
				t.Errorf("%d nights for %d civil days; want days-1", len(got), days)
			}
		})
	}
}

// The window's hours are parameters, not literals in the pass.
func TestNightsUsesItsParameters(t *testing.T) {
	j := Journey{
		WindowStart: ts("2026-05-22T09:00:00+00:00"), WindowEnd: ts("2026-05-23T18:00:00+00:00"),
		Stops: []Stop{stopAt("2026-05-22T20:00:00+00:00", "2026-05-22T23:30:00+00:00")}, // 1.5 h of 22–06
	}
	if n := Nights(j, DefaultNightParams())[0]; n.Kind != NightUnknown {
		t.Errorf("default: %+v, want unknown", n)
	}
	if n := Nights(j, NightParams{StartHour: 22, EndHour: 6, MinHours: 1})[0]; n.Kind != NightStop || n.Hours != 1.5 {
		t.Errorf("minimum 1 h: %+v, want the stop for 1.5 h", n)
	}
	if n := Nights(j, NightParams{StartHour: 20, EndHour: 6, MinHours: 3})[0]; n.Kind != NightStop || n.Hours != 3.5 {
		t.Errorf("window from 20:00: %+v, want the stop for 3.5 h", n)
	}
}
