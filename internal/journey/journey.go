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
	"math"
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
	// MaxSpeedKmh and ClusterRadiusM drive teleport rejection: consecutive
	// selected points within ClusterRadiusM of a cluster's first point form
	// a cluster, and a cluster is rejected iff (its entry OR exit implied
	// speed exceeds MaxSpeedKmh) AND bridging its neighbours directly is
	// plausible (≤ MaxSpeedKmh). The bridge condition is what makes the rule
	// safe: a real flight has no impossible edge (581 km/h HYD→TVM in the
	// measured case) so it is never even evaluated, and a genuine relocation
	// cannot be falsely killed because removing it would not restore
	// plausibility. Cluster-level, not point-level, because the observed
	// failure is a RUN of identical mislocated fixes (a wrong entry in the
	// geolocation database recurring across days) whose interior looks
	// self-consistent — the detector's point-wise both-neighbours rule and
	// Dawarich's speed sandwich both pass runs. MaxSpeedKmh 0 disables the
	// pass; the default is the detector's MAX_KMH constant. Flagging, never
	// deleting: rows are untouched, RejectedSpeed counts.
	MaxSpeedKmh    float64 `json:"max_speed_kmh"`
	ClusterRadiusM float64 `json:"cluster_radius_m"`
	// DivergenceWarnPct: the ground reconstruction (observed + routed +
	// unknown chords — air excluded by construction) is compared against
	// Google's own ground figure (activity distances minus FLYING ones —
	// air must leave both sides or neither, else every flight adventure
	// fails validation systematically); a divergence beyond this many
	// percent, either direction, is flagged. A conversation starter, never
	// a gate: unroutable gaps under-count and OSM detours over-count, and
	// both are visible on the map explaining themselves (phase 3 BRIEF
	// §3E).
	DivergenceWarnPct float64 `json:"divergence_warn_pct"`
}

func DefaultParams() Params {
	return Params{
		GapThresholdMinutes: 20, ThinSpacingSeconds: 30, MinStopDwellSeconds: 300,
		MaxAccuracyM: 0, AirSpeedMinKmh: 250, MaxSpeedKmh: 900, ClusterRadiusM: 1000,
		DivergenceWarnPct: 15,
	}
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

	// RoutedPoints and RoutedKm are set only by routing application
	// (internal/route.Apply, from the batch-filled cache) — never by
	// Assemble, which stays routing-free so the golden fixtures pin the
	// whole pure pipeline. The routed geometry rides alongside Points: a
	// gap leg still carries exactly its two timestamped endpoints, because
	// routed vertices have no timestamps and are inference, not
	// measurement.
	RoutedPoints []domain.LatLng
	RoutedKm     float64
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
	// RejectedSpeed counts points in teleport clusters (see MaxSpeedKmh).
	// Unlike the two above, these are rejected after selection — the test
	// needs temporal context — so they are included in the InWindow counts.
	RejectedSpeed int

	TotalKm    float64 // ObservedKm + InferredKm: the chord sum over all kept points
	ObservedKm float64
	InferredKm float64
	// AirKm is the chord sum over air-classified gaps — a subset of
	// InferredKm. The endpoint chord is the great-circle distance, so no
	// separate figure exists for the arc. Excluded from road-distance
	// validation (phase 3 BRIEF §3E).
	AirKm float64
	// UnknownKm is the chord sum over gaps still unknown — from Assemble,
	// InferredKm − AirKm; route.Apply recomputes it as cached road answers
	// claim gaps. RoutedKm is the routed-road distance over road legs, set
	// only by route.Apply — Assemble never routes.
	UnknownKm float64
	RoutedKm  float64

	// GoogleDistanceKm sums Google's own distanceMeters over the activities
	// intersecting the window — the independent figure routed distance is
	// validated against in phase 3. GoogleGroundKm is the same sum minus
	// FLYING-mode activities: the comparable side of the road validation
	// (see Params.DivergenceWarnPct). Mode is a guess, so a flight Google
	// mislabelled as ground inflates GoogleGroundKm — and the divergence
	// flag is exactly the tripwire that surfaces it.
	GoogleDistanceKm float64
	GoogleGroundKm   float64
}

// GroundKm is the road-comparable reconstruction: observed + routed +
// still-unknown chords. Air is excluded by construction. A method, not a
// stored field, so it can never go stale across routing application.
func (j Journey) GroundKm() float64 { return j.ObservedKm + j.RoutedKm + j.UnknownKm }

// DivergencePct is the signed percentage by which the ground reconstruction
// differs from Google's ground figure; ok is false when Google recorded no
// ground distance in the window, in which case there is nothing to compare.
func (j Journey) DivergencePct() (pct float64, ok bool) {
	if j.GoogleGroundKm <= 0 {
		return 0, false
	}
	return (j.GroundKm() - j.GoogleGroundKm) / j.GoogleGroundKm * 100, true
}

// DivergenceFlagged reports whether the divergence exceeds
// Params.DivergenceWarnPct in either direction.
func (j Journey) DivergenceFlagged() bool {
	pct, ok := j.DivergencePct()
	return ok && j.Params.DivergenceWarnPct > 0 && math.Abs(pct) > j.Params.DivergenceWarnPct
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

	// Teleport rejection runs on the full sorted stream, before thinning,
	// so cluster edges see the finest temporal context available.
	pts, j.RejectedSpeed = rejectTeleports(pts, p)

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
			} else {
				j.UnknownKm += l.DistanceKm
			}
		}
	}
	j.TotalKm = j.ObservedKm + j.InferredKm

	j.Stops = findStops(obs.Activities, kept, winStart, winEnd, p)
	for _, a := range obs.Activities {
		if intersectsWindow(a, winStart, winEnd) {
			j.GoogleDistanceKm += a.DistanceM / 1000
			if a.Mode != "FLYING" {
				j.GoogleGroundKm += a.DistanceM / 1000
			}
		}
	}
	return j
}

// rejectTeleports drops clusters of mislocated fixes from the sorted stream
// (see Params.MaxSpeedKmh for the rule and its reasoning). It walks clusters
// keeping an anchor at the last accepted cluster, so a rejected cluster's
// neighbours are judged against each other, and consecutive bogus clusters
// at one wrong location fall together.
func rejectTeleports(pts []TimedPoint, p Params) ([]TimedPoint, int) {
	if p.MaxSpeedKmh <= 0 || len(pts) < 3 {
		return pts, 0
	}

	// Cluster: a maximal run of consecutive points within ClusterRadiusM of
	// the run's first point.
	type cluster struct{ start, end int } // half-open [start, end)
	var clusters []cluster
	cs := 0
	for i := 1; i <= len(pts); i++ {
		if i == len(pts) || geo.HaversineM(pts[cs].Loc, pts[i].Loc) > p.ClusterRadiusM {
			clusters = append(clusters, cluster{cs, i})
			cs = i
		}
	}

	// Implied speed between two points. Trace timestamps are minute-
	// truncated, so a same-minute displacement divides by zero without
	// being a teleport — the duration is floored at ThinSpacingSeconds,
	// the stream's own stated temporal resolution, so quantization noise
	// (130 m in "no time") reads as walking pace while a genuine 300 km
	// jump still reads as thousands of km/h.
	minH := p.ThinSpacingSeconds / 3600
	speed := func(a, b TimedPoint) float64 {
		km := geo.HaversineM(a.Loc, b.Loc) / 1000
		h := math.Max(b.Time.Sub(a.Time).Hours(), minH)
		if h <= 0 {
			if km == 0 {
				return 0
			}
			return math.Inf(1)
		}
		return km / h
	}

	rejected := make([]bool, len(clusters))
	prev := 0 // index of the last accepted cluster
	for i := 1; i < len(clusters)-1; i++ {
		entryFrom := pts[clusters[prev].end-1]
		first := pts[clusters[i].start]
		last := pts[clusters[i].end-1]
		next := pts[clusters[i+1].start]
		impossible := speed(entryFrom, first) > p.MaxSpeedKmh || speed(last, next) > p.MaxSpeedKmh
		bridgeable := speed(entryFrom, next) <= p.MaxSpeedKmh
		if impossible && bridgeable {
			rejected[i] = true
		} else {
			prev = i
		}
	}

	var kept []TimedPoint
	dropped := 0
	for i, c := range clusters {
		if rejected[i] {
			dropped += c.end - c.start
			continue
		}
		kept = append(kept, pts[c.start:c.end]...)
	}
	if dropped == 0 {
		return pts, 0
	}
	return kept, dropped
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
