// Package journey assembles one adventure's observations into an ordered list
// of legs, each observed or a gap — never one undifferentiated line (CLAUDE.md
// invariants 5 and 8). It is pure: no I/O, clock, or randomness (invariant 1);
// identical inputs always yield identical output. The committed golden fixture
// (testdata/journey-27jul2026.anon.json and its .expected.json) pins this
// package the way the reference detector pins detection; the exact pipeline
// semantics — thinning, tie-break, distances, the rest-halt rule — are recorded
// in testdata/journey-27jul2026.CONTRACT.md.
package journey

import (
	"sort"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
)

// Params holds every threshold by name (invariant 3). The defaults are the
// values recovered from the July 2026 measurement (CONTRACT.md).
type Params struct {
	// GapThresholdMinutes: a silence longer than this between consecutive kept
	// points closes the observed leg and emits a gap leg.
	GapThresholdMinutes float64 `json:"gap_threshold_minutes"`
	// ThinSpacingSeconds: minimum spacing of the merged stream; a point closer
	// than this to the last kept point is dropped (keep-earliest).
	ThinSpacingSeconds float64 `json:"thin_spacing_seconds"`
	// MinStopDwellSeconds: a pause between consecutive activities shorter than
	// this is not reported as a stop.
	MinStopDwellSeconds float64 `json:"min_stop_dwell_seconds"`
	// MaxAccuracyM: raw positions reporting worse accuracy are excluded from
	// assembly; 0 disables the filter. Default off, deliberately: on the golden
	// journey, 40 of the 121 in-window fixes are worse than 100 m and they are
	// the highway densification, not noise — see DECISIONS.md 2026-08-02.
	// The rows themselves are untouched either way: flagging, never deleting.
	MaxAccuracyM float64 `json:"max_accuracy_m"`
	// AirSpeedMinKmh: a gap whose implied speed (endpoint chord over gap
	// duration) meets this classifies as air — it renders as a great-circle
	// arc, is never routed, and is excluded from road-distance validation
	// (phase 3 BRIEF §3D). 0 disables classification. The default sits above
	// any ground transport sustained over a ≥20-minute silence and below any
	// flight long enough to matter, even diluted by airport dwell at the
	// gap's edges; a short flight inside a long silence can dilute under it —
	// a stated blind spot the divergence check exists to catch. Speed, not
	// Google's activity mode: modes are guesses that fail at extremes.
	AirSpeedMinKmh float64 `json:"air_speed_min_kmh"`
}

func DefaultParams() Params {
	return Params{GapThresholdMinutes: 20, ThinSpacingSeconds: 30, MinStopDwellSeconds: 300, MaxAccuracyM: 0, AirSpeedMinKmh: 250}
}

type LegKind string

const (
	LegObserved LegKind = "observed"
	LegGap      LegKind = "gap"
)

// GapKind classifies a gap leg. Phase 2 only ever produces GapUnknown; phase 3
// classifies gaps as road or air and consumes the field. It exists from the
// first design because retrofitting it after the renderer exists means
// reworking both (docs/phase-2/BRIEF.md §1.3).
type GapKind string

const (
	GapNone    GapKind = "" // observed legs
	GapUnknown GapKind = "unknown"
	GapRoad    GapKind = "road"
	GapAir     GapKind = "air"
)

// PointSource says where a merged point came from.
type PointSource string

const (
	SourceTrace PointSource = "trace" // a semanticSegments timelinePath point
	SourceRaw   PointSource = "raw"   // a rawSignals position fix
)

type TimedPoint struct {
	Time   time.Time
	Loc    domain.LatLng
	Source PointSource
}

// Leg is one stretch of a journey. An observed leg carries its full point run;
// a gap leg carries exactly its two endpoints (CONTRACT.md §5) — the honesty
// distinction survives because the kind rides with the geometry.
type Leg struct {
	Kind       LegKind
	GapKind    GapKind
	Points     []TimedPoint
	DistanceKm float64
}

func (l Leg) Start() time.Time { return l.Points[0].Time }
func (l Leg) End() time.Time   { return l.Points[len(l.Points)-1].Time }

// Stop is a halt between two consecutive activities — the device dwelt, so the
// pause is a stop, not missing coverage. DisplacementKm is the straight-line
// distance from the first to the last merged point inside the halt, never the
// path sum: position scatter while stationary is noise, not movement
// (CONTRACT.md §8).
type Stop struct {
	Start          time.Time
	End            time.Time
	Loc            domain.LatLng // mean of the merged points inside the halt
	Points         int
	DisplacementKm float64
}

type Journey struct {
	WindowStart time.Time
	WindowEnd   time.Time
	Params      Params

	Legs  []Leg
	Stops []Stop

	TracePointsInWindow int
	RawPointsInWindow   int
	TracePointsKept     int
	RawPointsKept       int

	// Anomaly counters: how many in-window points the filters excluded from
	// assembly. The underlying rows are never modified (invariant 2).
	RejectedNullIsland int // points at exactly (0, 0) — a known export defect
	RejectedAccuracy   int // raw positions worse than MaxAccuracyM

	TotalKm    float64 // ObservedKm + InferredKm: the chord sum over all kept points
	ObservedKm float64
	InferredKm float64
	// AirKm is the chord sum over air-classified gaps — a subset of
	// InferredKm. The endpoint chord is the great-circle distance, so no
	// separate figure exists for the arc. Excluded from road-distance
	// validation (phase 3 BRIEF §3E).
	AirKm float64

	// GoogleDistanceKm sums Google's own distanceMeters over the activities
	// intersecting the window — the independent figure routed distance is
	// validated against in phase 3.
	GoogleDistanceKm float64
}

// MergedPoints is the size of the thinned point stream the legs are built on.
func (j Journey) MergedPoints() int { return j.TracePointsKept + j.RawPointsKept }

// Assemble builds the journey for one window over immutable observations
// (invariant 2: obs is never mutated). Both window ends are inclusive.
func Assemble(obs domain.Observations, winStart, winEnd time.Time, p Params) Journey {
	j := Journey{WindowStart: winStart, WindowEnd: winEnd, Params: p}
	inWindow := func(t time.Time) bool { return !t.Before(winStart) && !t.After(winEnd) }

	// Selection: every located trace point and raw position in the window,
	// minus anomalies — exact (0,0) is a known export defect, and raw fixes
	// worse than MaxAccuracyM (when the filter is on) are excluded.
	var pts []TimedPoint
	for _, pp := range obs.Points {
		if pp.Loc == nil || !inWindow(pp.Time) {
			continue
		}
		if pp.Loc.Lat == 0 && pp.Loc.Lon == 0 {
			j.RejectedNullIsland++
			continue
		}
		pts = append(pts, TimedPoint{Time: pp.Time, Loc: *pp.Loc, Source: SourceTrace})
		j.TracePointsInWindow++
	}
	for _, rp := range obs.RawPositions {
		if rp.Loc == nil || !inWindow(rp.Time) {
			continue
		}
		if rp.Loc.Lat == 0 && rp.Loc.Lon == 0 {
			j.RejectedNullIsland++
			continue
		}
		if p.MaxAccuracyM > 0 && rp.AccuracyM > p.MaxAccuracyM {
			j.RejectedAccuracy++
			continue
		}
		pts = append(pts, TimedPoint{Time: rp.Time, Loc: *rp.Loc, Source: SourceRaw})
		j.RawPointsInWindow++
	}

	// One time-sorted stream. At equal timestamps trace sorts before raw — a
	// convention the golden fixture cannot adjudicate (CONTRACT.md §4), fixed
	// here so output is deterministic.
	sort.SliceStable(pts, func(a, b int) bool {
		if !pts[a].Time.Equal(pts[b].Time) {
			return pts[a].Time.Before(pts[b].Time)
		}
		return pts[a].Source == SourceTrace && pts[b].Source == SourceRaw
	})

	// Thinning: keep-earliest at ThinSpacingSeconds, no source preference.
	var kept []TimedPoint
	for _, pt := range pts {
		if len(kept) == 0 || pt.Time.Sub(kept[len(kept)-1].Time).Seconds() >= p.ThinSpacingSeconds {
			kept = append(kept, pt)
		}
	}
	for _, k := range kept {
		if k.Source == SourceTrace {
			j.TracePointsKept++
		} else {
			j.RawPointsKept++
		}
	}

	// Leg split: a silence over the threshold emits a gap leg holding exactly
	// its two endpoints; every kept point belongs to exactly one observed leg,
	// so observed and gap distances partition the total chord sum.
	if len(kept) > 0 {
		run := []TimedPoint{kept[0]}
		for _, pt := range kept[1:] {
			prev := run[len(run)-1]
			if pt.Time.Sub(prev.Time).Minutes() > p.GapThresholdMinutes {
				j.Legs = append(j.Legs, observedLeg(run))
				j.Legs = append(j.Legs, gapLeg(prev, pt, p))
				run = []TimedPoint{pt}
			} else {
				run = append(run, pt)
			}
		}
		j.Legs = append(j.Legs, observedLeg(run))
	}
	for _, l := range j.Legs {
		if l.Kind == LegObserved {
			j.ObservedKm += l.DistanceKm
		} else {
			j.InferredKm += l.DistanceKm
			if l.GapKind == GapAir {
				j.AirKm += l.DistanceKm
			}
		}
	}
	j.TotalKm = j.ObservedKm + j.InferredKm

	j.Stops = findStops(obs.Activities, kept, winStart, winEnd, p)
	for _, a := range obs.Activities {
		if intersectsWindow(a, winStart, winEnd) {
			j.GoogleDistanceKm += a.DistanceM / 1000
		}
	}
	return j
}

func observedLeg(run []TimedPoint) Leg {
	pts := make([]TimedPoint, len(run))
	copy(pts, run)
	var km float64
	for i := 1; i < len(pts); i++ {
		km += geo.HaversineM(pts[i-1].Loc, pts[i].Loc) / 1000
	}
	return Leg{Kind: LegObserved, GapKind: GapNone, Points: pts, DistanceKm: km}
}

// gapLeg emits a gap holding exactly its two endpoints, classified at
// assembly time (pure arithmetic, so the batch router and the renderer can
// never disagree about which gaps are flights): implied speed at or above
// AirSpeedMinKmh is air, everything else stays unknown until the routing
// cache supplies a road. Gap duration is always above GapThresholdMinutes,
// so the division is safe.
func gapLeg(from, to TimedPoint, p Params) Leg {
	km := geo.HaversineM(from.Loc, to.Loc) / 1000
	kind := GapUnknown
	if p.AirSpeedMinKmh > 0 && km/to.Time.Sub(from.Time).Hours() >= p.AirSpeedMinKmh {
		kind = GapAir
	}
	return Leg{
		Kind:       LegGap,
		GapKind:    kind,
		Points:     []TimedPoint{from, to},
		DistanceKm: km,
	}
}

// findStops reports pauses between consecutive activities inside the window.
// Activities are copied before sorting: inputs are immutable (invariant 2).
func findStops(activities []domain.Activity, kept []TimedPoint, winStart, winEnd time.Time, p Params) []Stop {
	var acts []domain.Activity
	for _, a := range activities {
		if intersectsWindow(a, winStart, winEnd) {
			acts = append(acts, a)
		}
	}
	sort.SliceStable(acts, func(i, k int) bool { return acts[i].Start.Before(acts[k].Start) })

	var out []Stop
	for i := 1; i < len(acts); i++ {
		haltStart, haltEnd := acts[i-1].End, acts[i].Start
		if haltEnd.Sub(haltStart).Seconds() < p.MinStopDwellSeconds {
			continue
		}
		s := Stop{Start: haltStart, End: haltEnd}
		var members []TimedPoint
		for _, k := range kept {
			if !k.Time.Before(haltStart) && !k.Time.After(haltEnd) {
				members = append(members, k)
			}
		}
		s.Points = len(members)
		if len(members) > 0 {
			var lat, lon float64
			for _, m := range members {
				lat += m.Loc.Lat
				lon += m.Loc.Lon
			}
			n := float64(len(members))
			s.Loc = domain.LatLng{Lat: lat / n, Lon: lon / n}
			s.DisplacementKm = geo.HaversineM(members[0].Loc, members[len(members)-1].Loc) / 1000
		}
		out = append(out, s)
	}
	return out
}

func intersectsWindow(a domain.Activity, winStart, winEnd time.Time) bool {
	return !a.End.Before(winStart) && !a.Start.After(winEnd)
}
