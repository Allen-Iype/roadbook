package detect

import (
	"sort"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
)

// BaseParams are the home-base derivation thresholds. They were literals in
// this file until phase 11 lifted them into Params (closing an invariant-3
// gap found while generalising home evidence); the defaults reproduce the
// prior behaviour byte-identically, which the regression suite proves.
type BaseParams struct {
	// GridPerDeg: evidence buckets on a 1/GridPerDeg-degree grid — 20 ⇒
	// ~5 km cells. Expressed as cells-per-degree (a multiplication) rather
	// than degrees-per-cell so the arithmetic is bit-identical to the
	// prototype's `round(lat*20)/20`.
	GridPerDeg float64 `json:"GRID_PER_DEG"`
	// MergeM: groups whose medians sit within this merge into one cluster.
	MergeM float64 `json:"MERGE_M"`
	// MinVisits: clusters below this evidence count are not homes.
	MinVisits int `json:"MIN_VISITS"`
	// EraPadDays: each base is active first-to-last evidence date ± this.
	EraPadDays int `json:"ERA_PAD_DAYS"`
}

// DefaultBaseParams are the measured defaults from the exploration phase —
// the exact values that lived here as literals.
func DefaultBaseParams() BaseParams {
	return BaseParams{GridPerDeg: 20, MergeM: 10000, MinVisits: 8, EraPadDays: 45}
}

// baseEvidence is one piece of home evidence: a place asserted (or, for
// synthetic stays, observed) with a civil date.
type baseEvidence struct {
	date time.Time // civil date in the evidence's own UTC offset
	loc  domain.LatLng
}

// deriveBases clusters INFERRED_HOME visits into home bases. Home is derived
// from the data because the export's own frequentPlaces knows only the
// current home and is wrong for every historical period. This is the primary
// home evidence; synthetic stays (synth.go) are consulted only when it yields
// nothing.
func deriveBases(visits []domain.Visit, bp BaseParams) []Base {
	var ev []baseEvidence
	for _, v := range visits {
		if v.SemanticType != "INFERRED_HOME" || v.Loc == nil {
			continue
		}
		ev = append(ev, baseEvidence{date: civilDate(v.Start), loc: *v.Loc})
	}
	return clusterBases(ev, bp, 0)
}

// clusterBases is the shared derivation core: bucket evidence on the grid,
// merge clusters whose medians sit within MergeM, keep clusters with at
// least MinVisits pieces of evidence (and, when minDays > 0, spanning at
// least that many distinct civil days), and mark each base active from its
// first to its last evidence date ± EraPadDays. A deliberate port of the
// prototype; Python semantics (stable sorts, insertion order) replicated.
func clusterBases(ev []baseEvidence, bp BaseParams, minDays int) []Base {
	type cell struct{ lat, lon float64 }
	groups := map[cell][]baseEvidence{}
	var order []cell // first-seen order; Python dicts iterate in insertion order
	for _, e := range ev {
		c := cell{pyRound(e.loc.Lat*bp.GridPerDeg, 0) / bp.GridPerDeg,
			pyRound(e.loc.Lon*bp.GridPerDeg, 0) / bp.GridPerDeg}
		if _, seen := groups[c]; !seen {
			order = append(order, c)
		}
		groups[c] = append(groups[c], e)
	}
	// Largest group first; stable, so equal sizes keep first-seen order —
	// matching Python's sorted() over dict insertion order.
	sort.SliceStable(order, func(i, j int) bool { return len(groups[order[i]]) > len(groups[order[j]]) })

	type cluster struct {
		center domain.LatLng // median of the founding group; not updated on merge (as in the prototype)
		items  []baseEvidence
	}
	var clusters []*cluster
	for _, c := range order {
		items := groups[c]
		center := domain.LatLng{Lat: medianOf(items, func(e baseEvidence) float64 { return e.loc.Lat }),
			Lon: medianOf(items, func(e baseEvidence) float64 { return e.loc.Lon })}
		merged := false
		for _, cl := range clusters {
			if geo.HaversineM(cl.center, center) < bp.MergeM {
				cl.items = append(cl.items, items...)
				merged = true
				break
			}
		}
		if !merged {
			clusters = append(clusters, &cluster{center: center, items: items})
		}
	}

	var bases []Base
	for _, cl := range clusters {
		if len(cl.items) < bp.MinVisits {
			continue
		}
		if minDays > 0 {
			days := map[time.Time]bool{}
			for _, it := range cl.items {
				days[it.date] = true
			}
			if len(days) < minDays {
				continue
			}
		}
		dates := make([]time.Time, len(cl.items))
		for i, it := range cl.items {
			dates[i] = it.date
		}
		sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
		bases = append(bases, Base{
			Center: domain.LatLng{
				Lat: pyRound(medianOf(cl.items, func(e baseEvidence) float64 { return e.loc.Lat }), 6),
				Lon: pyRound(medianOf(cl.items, func(e baseEvidence) float64 { return e.loc.Lon }), 6),
			},
			From: dates[0].AddDate(0, 0, -bp.EraPadDays),
			To:   dates[len(dates)-1].AddDate(0, 0, bp.EraPadDays),
			N:    len(cl.items),
		})
	}
	return bases
}

func medianOf[T any](items []T, f func(T) float64) float64 {
	vals := make([]float64, len(items))
	for i, it := range items {
		vals[i] = f(it)
	}
	sort.Float64s(vals)
	n := len(vals)
	if n%2 == 1 {
		return vals[n/2]
	}
	return (vals[n/2-1] + vals[n/2]) / 2
}
