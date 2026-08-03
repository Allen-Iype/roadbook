// Package detect implements adventure-candidate detection as a pure function
// (CLAUDE.md invariant 1): no I/O, no clock, no randomness. Identical inputs
// yield identical output, which is what makes the reference detector's results a
// regression test.
//
// This is a faithful port of prototype/detect_fixture.py. Where Go and Python
// differ in defaults — sort stability, rounding semantics, binary-search
// behaviour — this file replicates Python's behaviour deliberately, so the two
// implementations agree observation-for-observation. Those spots are marked.
package detect

import (
	"sort"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
)

// Params holds every detection threshold; nothing in the algorithm body is a
// literal (CLAUDE.md invariant 3). Each run's params are recorded with its
// results.
type Params struct {
	NearM       float64 `json:"NEAR_M"`        // "away" means farther than this from every active home base
	FarKm       float64 `json:"FAR_KM"`        // a destination must be dwelt farther than this
	MinObs      int     `json:"MIN_OBS"`       // reject spans with fewer observations
	MinHrs      float64 `json:"MIN_HRS"`       // reject spans shorter than this
	MinDwellMin float64 `json:"MIN_DWELL_MIN"` // a stop counts as dwelling only at or above this
	MaxKmh      float64 `json:"MAX_KMH"`       // implied-speed outlier threshold; must clear real air travel

	Score ScoreParams `json:"SCORE"` // confidence-scoring weights and anchors (score.go)
}

// DefaultParams are the measured defaults from the exploration phase.
func DefaultParams() Params {
	return Params{NearM: 25000, FarKm: 100, MinObs: 5, MinHrs: 1.0, MinDwellMin: 60, MaxKmh: 900,
		Score: DefaultScoreParams()}
}

// Base is a derived home base: a cluster of INFERRED_HOME visits with the era in
// which it was active. From/To are civil dates (midnight UTC markers).
type Base struct {
	Center domain.LatLng // median coordinate, rounded to 6 decimals
	From   time.Time
	To     time.Time
	N      int // number of home visits in the cluster
}

// ModeCount is one travel mode and its activity count within a candidate. Order
// in Candidate.Modes is significant: most frequent first, ties by first
// occurrence (replicating Python's Counter.most_common).
type ModeCount struct {
	Mode string `json:"mode"`
	N    int    `json:"n"`
}

// Candidate is one detected adventure candidate. Derived and disposable: every
// detection run regenerates all of them.
type Candidate struct {
	Start          time.Time
	End            time.Time
	Days           float64       // duration, rounded to 1 decimal
	DestKm         int           // distance home → farthest place dwelt (never merely passed through)
	TrackKm        int           // sum of activity distances within the span
	Stops          int           // dwelt visits within the span
	Dest           domain.LatLng // farthest dwelt place, rounded to 4 decimals
	Modes          []ModeCount   // top 3 modes
	Repeat         int           // earlier candidates with a destination within 60 km
	ObsCount       int           // observations in the span
	StartTruncated bool          // span begins at the first observation of the window (bug 4)
	EndTruncated   bool          // span ends at the last observation of the window

	// Confidence score, 0–100, with the per-component breakdown that
	// reproduces it (BRIEF §1.6). Ranking support only: no threshold on it
	// ever filters a candidate or confirms one automatically.
	Score          float64
	ScoreBreakdown []ScoreComponent
}

// Result is everything one detection run produces.
type Result struct {
	Bases           []Base
	Candidates      []Candidate
	OutliersDropped int
}

// obs is a flattened observation: something known to be at a place at a time.
type obs struct {
	start time.Time
	end   time.Time
	loc   domain.LatLng
}

// Run detects adventure candidates. See the package comment: the structure
// mirrors the reference detector stage for stage.
func Run(o domain.Observations, p Params) Result {
	all := flatten(o)
	all, dropped := rejectOutliers(all, p)
	bases := deriveBases(o.Visits)
	if len(bases) == 0 {
		// No INFERRED_HOME cluster ⇒ "away from home" is undefined. The
		// reference detector would crash here; returning no candidates is the
		// deliberate behaviour (the importer should surface it).
		return Result{OutliersDropped: dropped}
	}

	dist := make([]float64, len(all))
	for i, ob := range all {
		dist[i] = homeDistM(bases, ob.start, ob.loc)
	}

	spans := findSpans(dist, p.NearM)

	// File-order key slices for the same binary searches the prototype does.
	// The export is chronological; the search replicates bisect exactly either
	// way, so both implementations agree even if that assumption ever breaks.
	visitStarts := make([]time.Time, len(o.Visits))
	for i, v := range o.Visits {
		visitStarts[i] = v.Start
	}
	actStarts := make([]time.Time, len(o.Activities))
	for i, a := range o.Activities {
		actStarts[i] = a.Start
	}

	var cands []Candidate
	for _, sp := range spans {
		t0, t1 := all[sp.s].start, all[sp.e].end

		// A destination is somewhere you dwelt, never a point passed through:
		// this is what keeps home-to-home corridor transit from qualifying.
		type dwell struct {
			distM float64
			loc   domain.LatLng
			hrs   float64
		}
		var dwells []dwell
		for j := bisectLeft(visitStarts, t0); j < bisectRight(visitStarts, t1); j++ {
			v := o.Visits[j]
			if v.Loc != nil && v.End.Sub(v.Start).Minutes() >= p.MinDwellMin {
				dwells = append(dwells, dwell{homeDistM(bases, v.Start, *v.Loc), *v.Loc, v.End.Sub(v.Start).Hours()})
			}
		}
		if len(dwells) == 0 {
			continue
		}
		best := dwells[0]
		for _, d := range dwells[1:] {
			// Python max() on (dist, [lat, lon]) tuples: farthest wins, exact
			// distance ties break by coordinate.
			if d.distM > best.distM ||
				(d.distM == best.distM && (d.loc.Lat > best.loc.Lat ||
					(d.loc.Lat == best.loc.Lat && d.loc.Lon > best.loc.Lon))) {
				best = d
			}
		}
		durHrs := t1.Sub(t0).Hours()
		if best.distM/1000 <= p.FarKm || sp.e-sp.s+1 < p.MinObs || durHrs < p.MinHrs {
			continue
		}

		var trackM float64
		modeCounts := map[string]int{}
		var modeOrder []string
		for j := bisectLeft(actStarts, t0); j < bisectRight(actStarts, t1); j++ {
			a := o.Activities[j]
			trackM += a.DistanceM
			if a.Mode != "" {
				if _, seen := modeCounts[a.Mode]; !seen {
					modeOrder = append(modeOrder, a.Mode)
				}
				modeCounts[a.Mode]++
			}
		}

		// Destination dwell for scoring: time dwelt within DestRadiusKm of
		// the chosen destination, so days spent *at* the place count and a
		// lunch stop en route does not.
		var destDwellHrs float64
		for _, d := range dwells {
			if geo.HaversineM(best.loc, d.loc)/1000 <= p.Score.DestRadiusKm {
				destDwellHrs += d.hrs
			}
		}
		days := durHrs / 24
		score, breakdown := scoreCandidate([]scoreInput{
			{name: "distance_from_home", unit: "km", raw: best.distM / 1000, known: true,
				w: p.Score.WeightDistance, full: p.Score.DistanceFullKm},
			{name: "destination_dwell", unit: "h", raw: destDwellHrs, known: true,
				w: p.Score.WeightDwell, full: p.Score.DwellFullHrs},
			{name: "observation_density", unit: "obs/day", raw: float64(sp.e-sp.s+1) / days, known: days > 0,
				w: p.Score.WeightDensity, full: p.Score.DensityFullPerDay},
			{name: "span_duration", unit: "days", raw: days, known: true,
				w: p.Score.WeightDuration, full: p.Score.DurationFullDays},
		})

		cands = append(cands, Candidate{
			Start:    t0,
			End:      t1,
			Days:     pyRound(durHrs/24, 1),
			DestKm:   int(pyRound(best.distM/1000, 0)),
			TrackKm:  int(pyRound(trackM/1000, 0)),
			Stops:    len(dwells),
			Dest:     domain.LatLng{Lat: pyRound(best.loc.Lat, 4), Lon: pyRound(best.loc.Lon, 4)},
			Modes:    topModes(modeCounts, modeOrder, 3),
			ObsCount: sp.e - sp.s + 1,
			// Bug 4: a span touching the edge of the observed window was cut by
			// the import boundary, not ended by a return home. Mark, don't guess.
			StartTruncated: sp.s == 0,
			EndTruncated:   sp.e == len(all)-1,
			Score:          score,
			ScoreBreakdown: breakdown,
		})
	}

	// Repeat counts use the rounded destinations, as the prototype does.
	for i := range cands {
		for j := range i {
			if geo.HaversineM(cands[j].Dest, cands[i].Dest) < 60000 {
				cands[i].Repeat++
			}
		}
	}

	return Result{Bases: bases, Candidates: cands, OutliersDropped: dropped}
}

// flatten turns visits, activity endpoints, and path points into one
// time-ordered observation list. The sort must be stable and key only on start
// time: Python's sort is stable and the prototype relies on it, so equal
// timestamps keep insertion order (visits, then activities, then points).
func flatten(o domain.Observations) []obs {
	var all []obs
	for _, v := range o.Visits {
		if v.Loc != nil {
			all = append(all, obs{v.Start, v.End, *v.Loc})
		}
	}
	for _, a := range o.Activities {
		if a.From != nil {
			all = append(all, obs{a.Start, a.Start, *a.From})
		}
		if a.To != nil {
			all = append(all, obs{a.End, a.End, *a.To})
		}
	}
	for _, pt := range o.Points {
		if pt.Loc != nil {
			all = append(all, obs{pt.Time, pt.Time, *pt.Loc})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].start.Before(all[j].start) })
	return all
}

// rejectOutliers drops a point only if it implies more than MaxKmh from *both*
// neighbours — a genuinely fast leg looks wrong from one side only, so requiring
// both prevents discarding real flights. Flags are computed against the original
// list, then filtered in one pass (not iteratively).
func rejectOutliers(all []obs, p Params) ([]obs, int) {
	bad := make([]bool, len(all))
	for i := range all {
		considered, fast := 0, 0
		for _, j := range []int{i - 1, i + 1} {
			if j < 0 || j >= len(all) {
				continue
			}
			dtHrs := all[j].start.Sub(all[i].start).Abs().Hours()
			if dtHrs <= 0 {
				continue
			}
			considered++
			if geo.HaversineM(all[i].loc, all[j].loc)/1000/dtHrs > p.MaxKmh {
				fast++
			}
		}
		bad[i] = considered > 0 && fast == considered
	}
	kept := all[:0]
	dropped := 0
	for i, ob := range all {
		if bad[i] {
			dropped++
		} else {
			kept = append(kept, ob)
		}
	}
	return kept, dropped
}

type span struct{ s, e int }

// findSpans returns maximal runs of observations farther than nearM from every
// active home base.
func findSpans(dist []float64, nearM float64) []span {
	var spans []span
	cur := -1
	last := -1
	for i := range dist {
		if dist[i] > nearM {
			if cur < 0 {
				cur = i
			}
			last = i
		} else if cur >= 0 {
			spans = append(spans, span{cur, last})
			cur = -1
		}
	}
	if cur >= 0 {
		spans = append(spans, span{cur, last})
	}
	return spans
}

// homeDistM is distance to the nearest base active at t; if no base is active
// (outside every era), it falls back to all bases. Home is a set over time, not
// a point: "away" means far from every base active at that moment.
func homeDistM(bases []Base, t time.Time, loc domain.LatLng) float64 {
	d := civilDate(t)
	min := -1.0
	for _, b := range bases {
		if d.Before(b.From) || d.After(b.To) {
			continue
		}
		if m := geo.HaversineM(b.Center, loc); min < 0 || m < min {
			min = m
		}
	}
	if min >= 0 {
		return min
	}
	for _, b := range bases {
		if m := geo.HaversineM(b.Center, loc); min < 0 || m < min {
			min = m
		}
	}
	return min
}

func topModes(counts map[string]int, order []string, n int) []ModeCount {
	out := make([]ModeCount, 0, len(order))
	for _, m := range order {
		out = append(out, ModeCount{m, counts[m]})
	}
	// Stable by descending count over first-seen order = Counter.most_common.
	sort.SliceStable(out, func(i, j int) bool { return out[i].N > out[j].N })
	if len(out) > n {
		out = out[:n]
	}
	return out
}
