package detect

// Candidate↔decision matching: the identity scheme from BRIEF §3.1. Decisions
// carry an anchor (the span and destination of the candidate as the user saw it
// when deciding); each detection run's candidates are re-associated with
// decisions by this pure, deterministic function. Nothing stores the
// association — it is recomputed, so re-detection can never corrupt it.

import (
	"sort"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
)

// SpanRef identifies a candidate for matching: its row id, time span, and
// destination.
type SpanRef struct {
	ID    int64
	Start time.Time
	End   time.Time
	Dest  domain.LatLng
}

// Anchor is a decision's stored snapshot of what the user decided on.
type Anchor struct {
	ID        int64
	Start     time.Time
	End       time.Time
	Dest      domain.LatLng
	CreatedAt time.Time
}

// MatchParams holds the matching thresholds — named parameters like every
// other threshold (CLAUDE.md invariant 3).
type MatchParams struct {
	MatchKm float64 // maximum destination separation for a candidate and decision to be compatible
}

func DefaultMatchParams() MatchParams { return MatchParams{MatchKm: 50} }

// Match associates candidates with decisions. A pair is compatible if the time
// spans overlap at all and the destinations are within MatchKm. Compatible
// pairs are ranked by overlap duration (time overlap is the load-bearing
// signal: spans are days long and parameter changes move boundaries by hours)
// and assigned greedily, each candidate and each decision used at most once.
// Ties break by earlier candidate start, then earlier decision creation, then
// ids — fully deterministic: identical inputs give identical matchings.
//
// The returned map is candidate id → decision id. Decisions absent from the
// values are orphans; the caller must surface them, never drop them.
func Match(cands []SpanRef, decs []Anchor, p MatchParams) map[int64]int64 {
	type pair struct {
		cand    int
		dec     int
		overlap time.Duration
	}
	var pairs []pair
	for ci, c := range cands {
		for di, d := range decs {
			overlap := minTime(c.End, d.End).Sub(maxTime(c.Start, d.Start))
			if overlap <= 0 {
				continue
			}
			if geo.HaversineM(c.Dest, d.Dest) > p.MatchKm*1000 {
				continue
			}
			pairs = append(pairs, pair{ci, di, overlap})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		a, b := pairs[i], pairs[j]
		if a.overlap != b.overlap {
			return a.overlap > b.overlap
		}
		ca, cb := cands[a.cand], cands[b.cand]
		if !ca.Start.Equal(cb.Start) {
			return ca.Start.Before(cb.Start)
		}
		da, db := decs[a.dec], decs[b.dec]
		if !da.CreatedAt.Equal(db.CreatedAt) {
			return da.CreatedAt.Before(db.CreatedAt)
		}
		if ca.ID != cb.ID {
			return ca.ID < cb.ID
		}
		return da.ID < db.ID
	})

	matched := make(map[int64]int64)
	usedCand := make(map[int]bool)
	usedDec := make(map[int]bool)
	for _, p := range pairs {
		if usedCand[p.cand] || usedDec[p.dec] {
			continue
		}
		usedCand[p.cand] = true
		usedDec[p.dec] = true
		matched[cands[p.cand].ID] = decs[p.dec].ID
	}
	return matched
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
