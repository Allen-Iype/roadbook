package journey_test

import (
	"testing"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
)

// Phase 23 BRIEF §7: a gap the export labels a flight is drawn as one even
// when the silence around it (time at the airport) pulls the speed between
// its ends under AIR_SPEED_MIN_KMH.
func TestGapLabelledFlying(t *testing.T) {
	// One recorded run, a silence of `hours` over `deg` degrees of
	// latitude (111 km each), one recorded run.
	obsFor := func(deg float64, hours int, acts ...domain.Activity) domain.Observations {
		end := 60 + hours*3600
		return domain.Observations{
			Points: []domain.PathPoint{
				{Time: at(0), Loc: loc(10, 70)},
				{Time: at(60), Loc: loc(10.001, 70)},
				{Time: at(end), Loc: loc(10.001+deg, 70)},
				{Time: at(end + 60), Loc: loc(10.002+deg, 70)},
			},
			Activities: acts,
		}
	}
	act := func(mode string, fromSec, toSec int) domain.Activity {
		return domain.Activity{Start: at(fromSec), End: at(toSec), Mode: mode}
	}
	def := journey.DefaultParams()
	off := def
	off.AirLabelMinKm = 0
	short := def
	short.AirLabelMinKm = 10

	cases := []struct {
		name string
		obs  domain.Observations
		p    journey.Params
		want journey.GapKind
	}{
		// 5° ≈ 556 km in 4 h is 139 km/h: under the speed line.
		{"slow and unlabelled: not a flight", obsFor(5, 4), def, journey.GapUnknown},
		{"slow, labelled flying, long: a flight", obsFor(5, 4, act("FLYING", 3600, 3*3600)), def, journey.GapAir},
		{"the same with the label rule switched off", obsFor(5, 4, act("FLYING", 3600, 3*3600)), off, journey.GapUnknown},
		{"labelled a train: not a flight", obsFor(5, 4, act("IN_TRAIN", 3600, 3*3600)), def, journey.GapUnknown},
		{
			"flying is not the longest label over the gap: not a flight",
			obsFor(5, 4, act("FLYING", 60, 3600), act("IN_PASSENGER_VEHICLE", 3600, 4*3600)), def, journey.GapUnknown,
		},
		// 0.5° ≈ 56 km: the ground at one end of a flight.
		{"labelled flying but under the minimum length", obsFor(0.5, 1, act("FLYING", 60, 3600)), def, journey.GapUnknown},
		{"the same with the minimum lowered", obsFor(0.5, 1, act("FLYING", 60, 3600)), short, journey.GapAir},
		// 10° ≈ 1,111 km in 2 h: the speed rule, whatever the label says.
		{"fast, labelled as road travel: still a flight", obsFor(10, 2, act("IN_PASSENGER_VEHICLE", 60, 2*3600)), def, journey.GapAir},
	}
	for _, tc := range cases {
		j := journey.Assemble(tc.obs, t0, at(6*3600), tc.p)
		var got []journey.GapKind
		for _, l := range j.Legs {
			if l.Kind == journey.LegGap {
				got = append(got, l.GapKind)
			}
		}
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("%s: gap kinds = %v, want [%s]", tc.name, got, tc.want)
		}
	}
	if def.AirLabelMinKm != 100 {
		t.Errorf("AIR_LABEL_MIN_KM default = %v, want 100 (phase 23 BRIEF §7 G)", def.AirLabelMinKm)
	}
}
