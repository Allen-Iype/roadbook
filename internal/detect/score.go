package detect

// Confidence scoring (BRIEF §1.6): a candidate scores 0–100 from named
// weighted components. Scoring is a ranking concern — "rank, don't filter"
// — so it lives in the pure detection core and never excludes a candidate.
// The score is decision support for the confirm step, never a trigger for
// auto-confirmation.

// ScoreParams names every scoring threshold (invariant 3). Weights need not
// sum to 1: the score divides by the sum of the weights actually present, so
// only their ratios matter — and a component that cannot be computed simply
// drops out of numerator and denominator, redistributing its weight across
// the rest proportionally. Absent data must never read as low confidence.
//
// Each *Full* anchor is the raw value at which its component saturates at
// 1.0; below it the component scales linearly from zero.
type ScoreParams struct {
	WeightDistance float64 `json:"W_DISTANCE"` // how far from home the destination is
	WeightDwell    float64 `json:"W_DWELL"`    // how long was spent at the destination
	WeightDensity  float64 `json:"W_DENSITY"`  // how well-observed the span is
	WeightDuration float64 `json:"W_DURATION"` // how long the span lasted

	DistanceFullKm    float64 `json:"DISTANCE_FULL_KM"`     // destination distance saturating the component
	DwellFullHrs      float64 `json:"DWELL_FULL_HRS"`       // destination dwell saturating the component
	DensityFullPerDay float64 `json:"DENSITY_FULL_PER_DAY"` // observations/day saturating the component
	DurationFullDays  float64 `json:"DURATION_FULL_DAYS"`   // span days saturating the component

	DestRadiusKm float64 `json:"DEST_RADIUS_KM"` // dwells within this of the destination count as destination dwell
}

// DefaultScoreParams: distance and dwell carry the most weight because they
// are the two halves of the detection rule itself (far from home, with a
// real destination); density and duration corroborate.
func DefaultScoreParams() ScoreParams {
	return ScoreParams{
		WeightDistance: 0.35, WeightDwell: 0.30, WeightDensity: 0.20, WeightDuration: 0.15,
		DistanceFullKm: 1000, DwellFullHrs: 24, DensityFullPerDay: 48, DurationFullDays: 7,
		DestRadiusKm: 60,
	}
}

// ScoreComponent is one line of a stored breakdown: enough to reproduce the
// arithmetic by hand. Contribution is the component's share of the final
// 0–100 score; contributions sum to the score.
type ScoreComponent struct {
	Name         string  `json:"name"`
	Present      bool    `json:"present"`
	Weight       float64 `json:"weight"` // configured weight
	Raw          float64 `json:"raw"`    // measured value, 0 when absent
	Unit         string  `json:"unit"`
	Normalized   float64 `json:"normalized"`   // raw scaled against its Full anchor, clamped to [0,1]
	Contribution float64 `json:"contribution"` // weight/Σ(present weights) × normalized × 100
}

// scoreInput is what one component measured, or didn't: Known=false means the
// value could not be computed from the available data, which is different
// from a value of zero.
type scoreInput struct {
	name  string
	unit  string
	raw   float64
	known bool
	w     float64
	full  float64
}

// scoreCandidate turns measured inputs into a 0–100 score with its breakdown.
// Pure arithmetic; rounding matches the project's Python-compatible rounding
// so stored scores are stable across re-runs.
func scoreCandidate(inputs []scoreInput) (float64, []ScoreComponent) {
	var wSum float64
	for _, in := range inputs {
		if in.known {
			wSum += in.w
		}
	}
	comps := make([]ScoreComponent, 0, len(inputs))
	var score float64
	for _, in := range inputs {
		c := ScoreComponent{Name: in.name, Unit: in.unit, Weight: in.w, Present: in.known}
		if in.known && wSum > 0 {
			c.Raw = pyRound(in.raw, 4)
			n := 0.0
			if in.full > 0 {
				n = in.raw / in.full
			}
			if n > 1 {
				n = 1
			}
			if n < 0 {
				n = 0
			}
			c.Normalized = pyRound(n, 4)
			c.Contribution = pyRound(in.w/wSum*n*100, 1)
			score += in.w / wSum * n * 100
		}
		comps = append(comps, c)
	}
	return pyRound(score, 1), comps
}
