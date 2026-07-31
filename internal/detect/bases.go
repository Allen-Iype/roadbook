package detect

import (
	"sort"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
)

// deriveBases clusters INFERRED_HOME visits into home bases: bucket on a ~5 km
// grid, merge clusters whose medians sit within 10 km, keep clusters with ≥8
// visits, and mark each base active from its first to its last home visit ±45
// days. Home is derived from the data because the export's own frequentPlaces
// knows only the current home and is wrong for every historical period.
func deriveBases(visits []domain.Visit) []Base {
	type item struct {
		date time.Time // civil date in the visit's own UTC offset
		loc  domain.LatLng
	}
	type cell struct{ lat, lon float64 }
	groups := map[cell][]item{}
	var order []cell // first-seen order; Python dicts iterate in insertion order
	for _, v := range visits {
		if v.SemanticType != "INFERRED_HOME" || v.Loc == nil {
			continue
		}
		c := cell{pyRound(v.Loc.Lat*20, 0) / 20, pyRound(v.Loc.Lon*20, 0) / 20}
		if _, seen := groups[c]; !seen {
			order = append(order, c)
		}
		groups[c] = append(groups[c], item{civilDate(v.Start), *v.Loc})
	}
	// Largest group first; stable, so equal sizes keep first-seen order —
	// matching Python's sorted() over dict insertion order.
	sort.SliceStable(order, func(i, j int) bool { return len(groups[order[i]]) > len(groups[order[j]]) })

	type cluster struct {
		center domain.LatLng // median of the founding group; not updated on merge (as in the prototype)
		items  []item
	}
	var clusters []*cluster
	for _, c := range order {
		items := groups[c]
		center := domain.LatLng{Lat: medianOf(items, func(i item) float64 { return i.loc.Lat }),
			Lon: medianOf(items, func(i item) float64 { return i.loc.Lon })}
		merged := false
		for _, cl := range clusters {
			if geo.HaversineM(cl.center, center) < 10000 {
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
		if len(cl.items) < 8 {
			continue
		}
		dates := make([]time.Time, len(cl.items))
		for i, it := range cl.items {
			dates[i] = it.date
		}
		sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
		bases = append(bases, Base{
			Center: domain.LatLng{
				Lat: pyRound(medianOf(cl.items, func(i item) float64 { return i.loc.Lat }), 6),
				Lon: pyRound(medianOf(cl.items, func(i item) float64 { return i.loc.Lon }), 6),
			},
			From: dates[0].AddDate(0, 0, -45),
			To:   dates[len(dates)-1].AddDate(0, 0, 45),
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
