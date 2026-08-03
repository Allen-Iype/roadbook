package detect

import "testing"

func inputs() []scoreInput {
	return []scoreInput{
		{name: "distance_from_home", unit: "km", raw: 500, known: true, w: 0.35, full: 1000},
		{name: "destination_dwell", unit: "h", raw: 12, known: true, w: 0.30, full: 24},
		{name: "observation_density", unit: "obs/day", raw: 24, known: true, w: 0.20, full: 48},
		{name: "span_duration", unit: "days", raw: 3.5, known: true, w: 0.15, full: 7},
	}
}

func TestScoreAllPresent(t *testing.T) {
	// Every component at exactly half its anchor: the weighted mean is 0.5
	// regardless of the weights, so the score must be 50.
	score, comps := scoreCandidate(inputs())
	if score != 50.0 {
		t.Fatalf("score = %v, want 50.0", score)
	}
	var sum float64
	for _, c := range comps {
		if !c.Present {
			t.Errorf("component %s not present", c.Name)
		}
		if c.Normalized != 0.5 {
			t.Errorf("component %s normalized = %v, want 0.5", c.Name, c.Normalized)
		}
		sum += c.Contribution
	}
	// Contributions are individually rounded to 1 decimal; they must still
	// reproduce the score to within stacking error.
	if sum < 49.8 || sum > 50.2 {
		t.Errorf("contributions sum to %v, want ~50", sum)
	}
}

func TestScoreClampsAtFull(t *testing.T) {
	in := inputs()
	in[0].raw = 5000 // five times the anchor still normalizes to 1, not 5
	score, comps := scoreCandidate(in)
	if comps[0].Normalized != 1.0 {
		t.Fatalf("normalized = %v, want clamped 1.0", comps[0].Normalized)
	}
	want := pyRound(0.35/1.0*100+0.30*0.5*100+0.20*0.5*100+0.15*0.5*100, 1)
	if score != want {
		t.Fatalf("score = %v, want %v", score, want)
	}
}

// The redistribution rule (BRIEF §1.6): a component that cannot be computed
// drops out of numerator and denominator, so the others' ratios decide the
// score — absent data must never read as low confidence.
func TestScoreRedistributesMissingWeight(t *testing.T) {
	in := inputs()
	in[1].known = false // dwell unmeasurable, weight 0.30 redistributes
	score, comps := scoreCandidate(in)
	// Remaining components all sit at 0.5, so the score must still be 50 —
	// not 50 minus the dwell share.
	if score != 50.0 {
		t.Fatalf("score with missing component = %v, want 50.0", score)
	}
	if comps[1].Present || comps[1].Contribution != 0 || comps[1].Raw != 0 {
		t.Errorf("missing component should be present=false with zero contribution, got %+v", comps[1])
	}
	// The missing component's configured weight stays visible in the
	// breakdown — the reader can see what was redistributed.
	if comps[1].Weight != 0.30 {
		t.Errorf("missing component weight = %v, want 0.30 (configured)", comps[1].Weight)
	}
}

func TestScoreNothingKnown(t *testing.T) {
	in := inputs()
	for i := range in {
		in[i].known = false
	}
	score, comps := scoreCandidate(in)
	if score != 0 {
		t.Fatalf("score with nothing known = %v, want 0", score)
	}
	for _, c := range comps {
		if c.Present {
			t.Errorf("component %s claims present", c.Name)
		}
	}
}
