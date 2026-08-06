// Command gen regenerates testdata/demo/demo.json — the committed demo
// dataset (phase 5 BRIEF §3C). A fictional persona over real public
// geography: the coordinates are Icelandic towns and sights on public
// roads, the person and every timestamp are invented, so the file carries
// no one's history and is safe to commit (authored fiction, not
// anonymised real data). Iceland deliberately: the whole demo sits inside
// one small Geofabrik extract (europe/iceland), so the optional routing
// walkthrough is cheap for anyone who tries it.
//
// Run from the repository root:
//
//	go run ./testdata/demo/gen
//
// The output is deterministic — no clock, no randomness — so regenerating
// must reproduce the committed file byte-identically; the README's numbers
// trace to these bytes (invariant 13).
//
// The persona's three months (2026-04-01 .. 2026-06-30) are scripted to
// exercise every honesty state the product renders:
//
//   - daily commute noise around Reykjavík, all inside NEAR — detection
//     must ignore it;
//   - a Borgarnes day trip (~44 km): an away-span with no far destination —
//     beyond NEAR, dwelt, and still correctly not a candidate;
//   - a dense south-coast drive to Höfn (Apr 24–26): timelinePath points
//     along the road — mostly observed legs;
//   - a sparse Westfjords trip to Ísafjörður (May 22–24): a handful of
//     observations across 450 road-km — routed/unknown gaps dominate;
//   - an Akureyri weekend by air (Jun 12–14): FLYING activities with no
//     path points — the gap classifies as air by speed and renders as an
//     arc, excluded from ground validation.
//
// The output is the current phone-export shape only — the one supported
// format (BRIEF §5). No rawSignals section: a real export carries one only
// for its final ~30 days, and none of the demo's honesty states needs it.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

const outPath = "testdata/demo/demo.json"

type latLng struct{ lat, lon float64 }

// Real public places, approximate town/sight coordinates on or beside
// public roads. The persona is fiction; the map is not.
var (
	home     = latLng{64.1420, -21.9270} // Reykjavík, Vesturbær
	work     = latLng{64.1180, -21.8560} // Kópavogur
	grocery  = latLng{64.1520, -21.8950} // Laugardalur
	borgarn  = latLng{64.5390, -21.9210} // Borgarnes
	selfoss  = latLng{63.9330, -20.9970}
	seljal   = latLng{63.6156, -19.9886} // Seljalandsfoss
	skogaf   = latLng{63.5321, -19.5114} // Skógafoss
	vik      = latLng{63.4187, -19.0060} // Vík í Mýrdal
	klaustur = latLng{63.7867, -18.0620} // Kirkjubæjarklaustur
	skafta   = latLng{64.0166, -16.9662} // Skaftafell
	jokuls   = latLng{64.0479, -16.1794} // Jökulsárlón
	hofn     = latLng{64.2539, -15.2101} // Höfn í Hornafirði
	holmavik = latLng{65.7060, -21.6700}
	budard   = latLng{65.1060, -21.7700} // Búðardalur
	isafj    = latLng{66.0749, -23.1240} // Ísafjörður
	kef      = latLng{63.9850, -22.6056} // Keflavík airport
	aey      = latLng{65.6600, -18.0727} // Akureyri airport
	akureyri = latLng{65.6885, -18.1262}
	godafoss = latLng{65.6828, -17.5503}
)

// seg builds one semanticSegments entry as a map; JSON object keys marshal
// sorted, which keeps the output deterministic.
type seg map[string]any

func coord(p latLng) string { return fmt.Sprintf("%.7f°, %.7f°", p.lat, p.lon) }

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// at builds an instant on a civil date; Iceland is UTC year-round, so the
// demo has no offset arithmetic anywhere.
func at(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, time.UTC)
}

func visit(start, end time.Time, p latLng, semanticType string) seg {
	return seg{
		"startTime": ts(start),
		"endTime":   ts(end),
		"visit": seg{
			"topCandidate": seg{
				"placeLocation": seg{"latLng": coord(p)},
				"semanticType":  semanticType,
			},
		},
	}
}

func activity(start, end time.Time, from, to latLng, meters float64, mode string) seg {
	return seg{
		"startTime": ts(start),
		"endTime":   ts(end),
		"activity": seg{
			"start":          seg{"latLng": coord(from)},
			"end":            seg{"latLng": coord(to)},
			"distanceMeters": meters,
			"topCandidate":   seg{"type": mode},
		},
	}
}

// path emits a timelinePath segment: points linearly interpolated between
// consecutive waypoints, evenly spaced in time. Linear interpolation cuts
// corners the road does not — which is exactly how sparse Timeline data
// looks, and the honest rendering of it is the product's point.
func path(start, end time.Time, waypoints []latLng, n int) seg {
	pts := make([]seg, 0, n)
	total := end.Sub(start)
	for i := range n {
		f := float64(i) / float64(n-1)
		// position along the waypoint chain
		fs := f * float64(len(waypoints)-1)
		k := min(int(fs), len(waypoints)-2)
		r := fs - float64(k)
		p := latLng{
			lat: waypoints[k].lat + r*(waypoints[k+1].lat-waypoints[k].lat),
			lon: waypoints[k].lon + r*(waypoints[k+1].lon-waypoints[k].lon),
		}
		pts = append(pts, seg{
			"point": coord(p),
			"time":  ts(start.Add(time.Duration(f * float64(total)))),
		})
	}
	return seg{
		"startTime":    ts(start),
		"endTime":      ts(end),
		"timelinePath": pts,
	}
}

func main() {
	if _, err := os.Stat("testdata/demo"); err != nil {
		fmt.Fprintln(os.Stderr, "gen: run from the repository root:", err)
		os.Exit(1)
	}

	var segs []seg
	add := func(s ...seg) { segs = append(segs, s...) }

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	// Days the trip scripts own wholesale; the daily loop skips them.
	trip := map[string]bool{}
	own := func(days ...time.Time) {
		for _, d := range days {
			trip[d.Format("2006-01-02")] = true
		}
	}
	own(at(2026, 4, 18, 0, 0), at(2026, 4, 24, 0, 0), at(2026, 4, 25, 0, 0), at(2026, 4, 26, 0, 0))
	own(at(2026, 5, 22, 0, 0), at(2026, 5, 23, 0, 0), at(2026, 5, 24, 0, 0))
	own(at(2026, 6, 12, 0, 0), at(2026, 6, 13, 0, 0), at(2026, 6, 14, 0, 0))

	// ── the ordinary days: commute and errand noise, all inside NEAR ──
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if trip[d.Format("2006-01-02")] {
			continue
		}
		y, m, day := d.Date()
		switch d.Weekday() {
		case time.Saturday:
			// home all day: one long INFERRED_HOME visit
			add(visit(at(y, m, day, 0, 0), at(y, m, day, 23, 59), home, "INFERRED_HOME"))
		case time.Sunday:
			add(visit(at(y, m, day, 0, 0), at(y, m, day, 10, 55), home, "INFERRED_HOME"))
			add(activity(at(y, m, day, 10, 55), at(y, m, day, 11, 5), home, grocery, 3200, "IN_PASSENGER_VEHICLE"))
			add(visit(at(y, m, day, 11, 5), at(y, m, day, 12, 30), grocery, "UNKNOWN"))
			add(activity(at(y, m, day, 12, 30), at(y, m, day, 12, 40), grocery, home, 3200, "IN_PASSENGER_VEHICLE"))
			add(visit(at(y, m, day, 12, 40), at(y, m, day, 23, 59), home, "INFERRED_HOME"))
		default:
			add(visit(at(y, m, day, 0, 0), at(y, m, day, 8, 10), home, "INFERRED_HOME"))
			add(activity(at(y, m, day, 8, 10), at(y, m, day, 8, 35), home, work, 6800, "IN_PASSENGER_VEHICLE"))
			add(path(at(y, m, day, 8, 15), at(y, m, day, 8, 30), []latLng{home, work}, 3))
			add(visit(at(y, m, day, 8, 35), at(y, m, day, 17, 5), work, "INFERRED_WORK"))
			add(activity(at(y, m, day, 17, 5), at(y, m, day, 17, 30), work, home, 6800, "IN_PASSENGER_VEHICLE"))
			add(visit(at(y, m, day, 17, 30), at(y, m, day, 23, 59), home, "INFERRED_HOME"))
		}
	}

	// ── Apr 18: Borgarnes day trip — away (44 km > NEAR) but no far
	// destination (< FAR), so correctly not a candidate ──
	add(visit(at(2026, 4, 18, 0, 0), at(2026, 4, 18, 9, 30), home, "INFERRED_HOME"))
	add(activity(at(2026, 4, 18, 9, 30), at(2026, 4, 18, 10, 30), home, borgarn, 74000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 18, 9, 35), at(2026, 4, 18, 10, 25), []latLng{home, borgarn}, 6))
	add(visit(at(2026, 4, 18, 10, 30), at(2026, 4, 18, 14, 0), borgarn, "UNKNOWN"))
	add(activity(at(2026, 4, 18, 14, 0), at(2026, 4, 18, 15, 0), borgarn, home, 74000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 18, 14, 5), at(2026, 4, 18, 14, 55), []latLng{borgarn, home}, 6))
	add(visit(at(2026, 4, 18, 15, 0), at(2026, 4, 18, 23, 59), home, "INFERRED_HOME"))

	// ── Apr 24–26: the dense adventure — south coast to Höfn, path points
	// along Route 1, stops dwelt on the way. Mostly observed legs. ──
	add(visit(at(2026, 4, 24, 0, 0), at(2026, 4, 24, 9, 0), home, "INFERRED_HOME"))
	add(activity(at(2026, 4, 24, 9, 0), at(2026, 4, 24, 12, 10), home, seljal, 128000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 24, 9, 5), at(2026, 4, 24, 12, 5), []latLng{home, selfoss, seljal}, 12))
	add(visit(at(2026, 4, 24, 12, 10), at(2026, 4, 24, 13, 0), seljal, "UNKNOWN"))
	add(activity(at(2026, 4, 24, 13, 0), at(2026, 4, 24, 13, 30), seljal, skogaf, 29000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 24, 13, 2), at(2026, 4, 24, 13, 28), []latLng{seljal, skogaf}, 4))
	add(visit(at(2026, 4, 24, 13, 30), at(2026, 4, 24, 14, 10), skogaf, "UNKNOWN"))
	add(activity(at(2026, 4, 24, 14, 10), at(2026, 4, 24, 14, 50), skogaf, vik, 34000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 24, 14, 12), at(2026, 4, 24, 14, 48), []latLng{skogaf, vik}, 5))
	add(visit(at(2026, 4, 24, 14, 50), at(2026, 4, 25, 9, 0), vik, "UNKNOWN")) // guesthouse

	add(activity(at(2026, 4, 25, 9, 0), at(2026, 4, 25, 11, 40), vik, skafta, 141000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 25, 9, 5), at(2026, 4, 25, 11, 35), []latLng{vik, klaustur, skafta}, 12))
	add(visit(at(2026, 4, 25, 11, 40), at(2026, 4, 25, 14, 0), skafta, "UNKNOWN"))
	add(activity(at(2026, 4, 25, 14, 0), at(2026, 4, 25, 14, 50), skafta, jokuls, 57000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 25, 14, 2), at(2026, 4, 25, 14, 48), []latLng{skafta, jokuls}, 6))
	add(visit(at(2026, 4, 25, 14, 50), at(2026, 4, 25, 16, 20), jokuls, "UNKNOWN"))
	add(activity(at(2026, 4, 25, 16, 20), at(2026, 4, 25, 17, 30), jokuls, hofn, 80000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 25, 16, 22), at(2026, 4, 25, 17, 28), []latLng{jokuls, hofn}, 7))
	add(visit(at(2026, 4, 25, 17, 30), at(2026, 4, 26, 10, 0), hofn, "UNKNOWN")) // hotel — the farthest dwelt place

	add(activity(at(2026, 4, 26, 10, 0), at(2026, 4, 26, 16, 30), hofn, selfoss, 396000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 26, 10, 5), at(2026, 4, 26, 16, 25), []latLng{hofn, jokuls, skafta, klaustur, vik, skogaf, seljal, selfoss}, 24))
	add(visit(at(2026, 4, 26, 16, 30), at(2026, 4, 26, 17, 10), selfoss, "UNKNOWN")) // fuel and dinner
	add(activity(at(2026, 4, 26, 17, 10), at(2026, 4, 26, 18, 10), selfoss, home, 57000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 4, 26, 17, 12), at(2026, 4, 26, 18, 8), []latLng{selfoss, home}, 5))
	add(visit(at(2026, 4, 26, 18, 10), at(2026, 4, 26, 23, 59), home, "INFERRED_HOME"))

	// ── May 22–24: the sparse adventure — Ísafjörður with a handful of
	// observations across 450 road-km. Unknown/routed gaps dominate. ──
	add(visit(at(2026, 5, 22, 0, 0), at(2026, 5, 22, 8, 10), home, "INFERRED_HOME"))
	add(activity(at(2026, 5, 22, 8, 10), at(2026, 5, 22, 8, 35), home, work, 6800, "IN_PASSENGER_VEHICLE"))
	add(visit(at(2026, 5, 22, 8, 35), at(2026, 5, 22, 13, 30), work, "INFERRED_WORK")) // half day
	add(activity(at(2026, 5, 22, 13, 30), at(2026, 5, 22, 13, 50), work, home, 6800, "IN_PASSENGER_VEHICLE"))
	add(visit(at(2026, 5, 22, 13, 50), at(2026, 5, 22, 14, 10), home, "INFERRED_HOME"))
	add(activity(at(2026, 5, 22, 14, 10), at(2026, 5, 22, 20, 0), home, isafj, 455000, "IN_PASSENGER_VEHICLE"))
	// two lonely fixes on a six-hour drive — the sparse case
	add(path(at(2026, 5, 22, 15, 10), at(2026, 5, 22, 17, 30), []latLng{borgarn, holmavik}, 2))
	add(visit(at(2026, 5, 22, 20, 0), at(2026, 5, 23, 10, 30), isafj, "UNKNOWN")) // guesthouse

	add(visit(at(2026, 5, 23, 11, 0), at(2026, 5, 23, 13, 0), latLng{66.0705, -23.1180}, "UNKNOWN")) // harbour café
	add(visit(at(2026, 5, 23, 13, 30), at(2026, 5, 24, 10, 0), isafj, "UNKNOWN"))

	add(activity(at(2026, 5, 24, 10, 0), at(2026, 5, 24, 16, 0), isafj, home, 455000, "IN_PASSENGER_VEHICLE"))
	// one lonely stationary pair on the return — a minute apart so the two
	// rows hash distinctly (identical rows would dedupe on import and make
	// file-mode and DB-mode counts disagree)
	add(path(at(2026, 5, 24, 13, 0), at(2026, 5, 24, 13, 1), []latLng{budard, budard}, 2))
	add(visit(at(2026, 5, 24, 16, 0), at(2026, 5, 24, 23, 59), home, "INFERRED_HOME"))

	// ── Jun 12–14: the flight adventure — Akureyri by air. FLYING
	// activities with no path points: the silence between airports
	// classifies as air by implied speed and renders as an arc. The
	// flights go through Keflavík deliberately: the city airport sits
	// inside NEAR, so a flight from it would fall outside the away-span —
	// and therefore outside the journey window — and no arc would ever
	// render. Keflavík is beyond NEAR, which keeps the flight inside the
	// observed span. ──
	add(visit(at(2026, 6, 12, 0, 0), at(2026, 6, 12, 6, 45), home, "INFERRED_HOME"))
	add(activity(at(2026, 6, 12, 6, 45), at(2026, 6, 12, 7, 35), home, kef, 47000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 6, 12, 6, 50), at(2026, 6, 12, 7, 30), []latLng{home, kef}, 4))
	add(visit(at(2026, 6, 12, 7, 35), at(2026, 6, 12, 8, 40), kef, "UNKNOWN")) // the airport gate
	// journey assembly draws from trace points, not visits or activity
	// endpoints — a real phone logs a fix at the gate and after landing,
	// and those two fixes are what bracket the flight-speed silence that
	// classifies the gap as air
	add(path(at(2026, 6, 12, 8, 34), at(2026, 6, 12, 8, 35), []latLng{kef, kef}, 2))
	add(activity(at(2026, 6, 12, 8, 40), at(2026, 6, 12, 9, 30), kef, aey, 283000, "FLYING"))
	add(path(at(2026, 6, 12, 9, 32), at(2026, 6, 12, 9, 33), []latLng{aey, aey}, 2))
	add(visit(at(2026, 6, 12, 9, 30), at(2026, 6, 12, 9, 45), aey, "UNKNOWN"))
	add(activity(at(2026, 6, 12, 9, 45), at(2026, 6, 12, 10, 0), aey, akureyri, 4000, "IN_PASSENGER_VEHICLE"))
	add(visit(at(2026, 6, 12, 10, 0), at(2026, 6, 13, 10, 30), akureyri, "UNKNOWN")) // hotel

	add(visit(at(2026, 6, 13, 11, 0), at(2026, 6, 13, 12, 30), latLng{65.6800, -18.0910}, "UNKNOWN")) // botanical garden
	add(activity(at(2026, 6, 13, 14, 0), at(2026, 6, 13, 14, 40), akureyri, godafoss, 35000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 6, 13, 14, 5), at(2026, 6, 13, 14, 35), []latLng{akureyri, godafoss}, 4))
	add(visit(at(2026, 6, 13, 14, 40), at(2026, 6, 13, 15, 30), godafoss, "UNKNOWN"))
	add(activity(at(2026, 6, 13, 15, 30), at(2026, 6, 13, 16, 10), godafoss, akureyri, 35000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 6, 13, 15, 32), at(2026, 6, 13, 16, 8), []latLng{godafoss, akureyri}, 4))
	add(visit(at(2026, 6, 13, 16, 10), at(2026, 6, 14, 15, 0), akureyri, "UNKNOWN"))

	add(activity(at(2026, 6, 14, 14, 30), at(2026, 6, 14, 14, 45), akureyri, aey, 4000, "IN_PASSENGER_VEHICLE"))
	add(visit(at(2026, 6, 14, 14, 45), at(2026, 6, 14, 15, 30), aey, "UNKNOWN"))
	add(path(at(2026, 6, 14, 15, 24), at(2026, 6, 14, 15, 25), []latLng{aey, aey}, 2))
	add(activity(at(2026, 6, 14, 15, 30), at(2026, 6, 14, 16, 20), aey, kef, 283000, "FLYING"))
	add(path(at(2026, 6, 14, 16, 22), at(2026, 6, 14, 16, 23), []latLng{kef, kef}, 2))
	add(activity(at(2026, 6, 14, 16, 30), at(2026, 6, 14, 17, 20), kef, home, 47000, "IN_PASSENGER_VEHICLE"))
	add(path(at(2026, 6, 14, 16, 35), at(2026, 6, 14, 17, 15), []latLng{kef, home}, 4))
	add(visit(at(2026, 6, 14, 17, 20), at(2026, 6, 14, 23, 59), home, "INFERRED_HOME"))

	// A real export is chronological, and the detector's dwell lookup
	// bisects on that order — the trip scripts above append out of sequence,
	// so sort by startTime (RFC3339 in one offset sorts lexicographically).
	sort.SliceStable(segs, func(i, j int) bool {
		return segs[i]["startTime"].(string) < segs[j]["startTime"].(string)
	})

	out, err := json.MarshalIndent(map[string]any{"semanticSegments": segs}, "", " ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	out = append(out, '\n')
	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s: %d segments, %d bytes\n", outPath, len(segs), len(out))
}
