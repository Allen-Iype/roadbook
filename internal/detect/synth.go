package detect

// Stay-point visit synthesis (phase 11). A source with no visit segments —
// photos today; Records.json, Semantic History, continuous GPX when their
// parsers arrive — produces neither home bases nor destination dwells, because
// both consume visits (bases.go, detect.go). This pass derives visit-shaped
// stays from bare timestamped fixes so such sources join the identical
// pipeline. It is pure and parameterised (invariants 1 and 3) and its output
// is never persisted (invariant 2): synthetic visits are derived at detection
// time, recomputed free on every run, and the run's params record how.

import (
	"sort"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
)

// SynthParams govern stay-point visit synthesis over photo-sourced fixes.
// Every threshold is named and recorded per run (invariant 3); defaults are
// the phase 11 brief's starting values (§4E), tuned against the committed
// corpus.
type SynthParams struct {
	// StayRadiusM: a fix within this of the open stay's running-mean
	// centroid extends the stay. Above photo-GPS scatter at one venue,
	// below "a different place".
	StayRadiusM float64 `json:"STAY_RADIUS_M"`
	// StayMinMin: a closed stay shorter than this is discarded. Deliberately
	// below MinDwellMin — synthesis finds stays, detection judges which are
	// destinations.
	StayMinMin float64 `json:"STAY_MIN_MIN"`
	// StayMaxGapMin: a silence longer than this closes the open stay even at
	// the same place. Photos are bursty; dinner and next morning's breakfast
	// at one hotel must not fuse into a single overnight dwell the evidence
	// never observed.
	StayMaxGapMin float64 `json:"STAY_MAX_GAP_MIN"`
	// HomeMinDays: synthetic home evidence must recur across at least this
	// many distinct civil days before a cluster can be a home base. People
	// photograph trips — a week of stays at one hotel is recurrence, but not
	// residence; day spread is what separates the two.
	HomeMinDays int `json:"HOME_MIN_DAYS"`
}

// DefaultSynthParams are the phase 11 brief's proposed starting values.
func DefaultSynthParams() SynthParams {
	return SynthParams{StayRadiusM: 200, StayMinMin: 30, StayMaxGapMin: 240, HomeMinDays: 8}
}

// photoFixes selects the located photo-sourced fixes from raw positions. The
// scoping by source class is what keeps every non-photo run byte-identical:
// Timeline rawSignals (WIFI/CELL/GPS) never enter detection, exactly as
// before this pass existed.
func photoFixes(raws []domain.RawPosition) []domain.RawPosition {
	var out []domain.RawPosition
	for _, rp := range raws {
		if rp.Source == domain.SourcePhoto && rp.Loc != nil {
			out = append(out, rp)
		}
	}
	return out
}

// synthesizeStays runs the single-forward-pass stay-point sweep: an open stay
// holds a running-mean centroid; a fix within StayRadiusM of it (and within
// StayMaxGapMin of the last member) extends the stay, anything else closes it
// and opens a new one. Closed stays lasting at least StayMinMin become
// synthetic visits — centroid location, first-to-last member span, and no
// semantic type, because synthesis asserts presence, never meaning.
func synthesizeStays(fixes []domain.RawPosition, p SynthParams) []domain.Visit {
	var located []domain.RawPosition
	for _, f := range fixes {
		if f.Loc != nil {
			located = append(located, f)
		}
	}
	if len(located) == 0 {
		return nil
	}
	// Sort a copy: photo batches arrive in file order, which is arbitrary,
	// and the input slice is immutable (invariant 2).
	sorted := make([]domain.RawPosition, len(located))
	copy(sorted, located)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Time.Before(sorted[j].Time) })

	type openStay struct {
		sumLat, sumLon float64
		n              int
		start, end     time.Time
	}
	var stays []domain.Visit
	var cur *openStay

	closeStay := func() {
		if cur == nil {
			return
		}
		if cur.end.Sub(cur.start).Minutes() >= p.StayMinMin {
			stays = append(stays, domain.Visit{
				Start: cur.start,
				End:   cur.end,
				Loc:   &domain.LatLng{Lat: cur.sumLat / float64(cur.n), Lon: cur.sumLon / float64(cur.n)},
			})
		}
		cur = nil
	}

	for _, f := range sorted {
		if cur != nil {
			centroid := domain.LatLng{Lat: cur.sumLat / float64(cur.n), Lon: cur.sumLon / float64(cur.n)}
			gapMin := f.Time.Sub(cur.end).Minutes()
			if gapMin > p.StayMaxGapMin || geo.HaversineM(centroid, *f.Loc) > p.StayRadiusM {
				closeStay()
			}
		}
		if cur == nil {
			cur = &openStay{start: f.Time, end: f.Time}
		}
		cur.sumLat += f.Loc.Lat
		cur.sumLon += f.Loc.Lon
		cur.n++
		if f.Time.After(cur.end) {
			cur.end = f.Time
		}
	}
	closeStay()
	return stays
}

// mergeVisitsByStart merges two start-sorted visit slices into a new slice,
// real visits winning ties. Neither input is touched (invariant 2). The
// result feeds the same bisect the file-order path uses, so it must be — and
// is — sorted by start.
func mergeVisitsByStart(real, synth []domain.Visit) []domain.Visit {
	out := make([]domain.Visit, 0, len(real)+len(synth))
	i, j := 0, 0
	for i < len(real) && j < len(synth) {
		if synth[j].Start.Before(real[i].Start) {
			out = append(out, synth[j])
			j++
		} else {
			out = append(out, real[i])
			i++
		}
	}
	out = append(out, real[i:]...)
	out = append(out, synth[j:]...)
	return out
}

// deriveSyntheticBases derives home bases from synthetic stays — the home
// evidence of last resort, consulted only when INFERRED_HOME evidence yields
// nothing (strict precedence; see Run). Same clustering as deriveBases, plus
// the HomeMinDays day-spread guard, because synthetic stays carry no semantic
// assertion of home and recurrence alone cannot tell a residence from a
// week's hotel.
func deriveSyntheticBases(stays []domain.Visit, p Params) []Base {
	ev := make([]baseEvidence, 0, len(stays))
	for _, s := range stays {
		if s.Loc != nil {
			ev = append(ev, baseEvidence{date: civilDate(s.Start), loc: *s.Loc})
		}
	}
	return clusterBases(ev, p.Bases, p.Synth.HomeMinDays)
}
