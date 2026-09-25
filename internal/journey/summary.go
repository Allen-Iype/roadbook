package journey

import (
	"math"
	"time"
)

// Summary is the journey summary block (phase 14 BRIEF §2a): what the trip
// was, stated from the assembled journey alone. Like ModeBreakdown it lives
// BESIDE Assemble, never inside it — the golden contract pins assembly's
// output byte-for-byte, and these are derived figures. Pure: same journey,
// same summary. Every field is printed by `roadbook journey` and served on
// the API, so no surface can show a figure the CLI cannot reproduce
// (invariant 13).
type Summary struct {
	// SpanHours is the window's length — the journey's measured "time
	// away". A truncated window (bug 4) is a truncated span; the truncation
	// words travel with it on every surface.
	SpanHours float64
	// CivilDays counts the civil dates the journey touches, each timestamp
	// read in its own recorded offset — the narrative's day count, by the
	// same rule the web's sliceDays applies: the range runs from the
	// earliest civil date among the window edges, leg endpoints, and stop
	// endpoints to the latest, inclusive. Stops can reach past the window
	// (a halt between two activities that straddle it), which is why the
	// rule reads more than the window edges.
	CivilDays int
	// Stops and DwellHours: how many halts the assembler reported and the
	// hours they add up to. A dwell is a measured pause — two activities
	// with a recorded gap between them — never a guess.
	Stops      int
	DwellHours float64
	// ObservedHours is the summed duration of MOVING observed legs;
	// ObservedPaceKmh is their summed distance over it — the average pace
	// across recorded stretches only. A moving observed leg is one that is
	// not a fix: more than one point and a non-zero distance, the web's
	// isFixLeg predicate mirrored — a stationary one-minute run is a fix,
	// and a fix has no pace (the demo's Westfjords loop showed "0 km/h"
	// before this rule). Pauses shorter than GapThresholdMinutes sit inside
	// an observed leg and are included, which the display's caveat states
	// by naming that parameter; gaps of every kind are excluded, because a
	// routed or unknown stretch has no measured duration of its own.
	// PaceOK is false when no moving observed leg exists: absent, never
	// zero.
	ObservedHours   float64
	ObservedPaceKmh float64
	PaceOK          bool
}

// Summarize derives the summary from an assembled (and optionally
// route-applied) journey. Routing changes nothing here: observed legs are
// the only ones with a measured duration, and their distance is measured.
func Summarize(j Journey) Summary {
	s := Summary{
		SpanHours: j.WindowEnd.Sub(j.WindowStart).Hours(),
		CivilDays: civilDays(j),
		Stops:     len(j.Stops),
	}
	for _, st := range j.Stops {
		s.DwellHours += st.End.Sub(st.Start).Hours()
	}
	var km float64
	for _, l := range j.Legs {
		if l.Kind != LegObserved || len(l.Points) < 2 || l.DistanceKm == 0 {
			continue // a fix, not a stretch
		}
		h := l.End().Sub(l.Start()).Hours()
		if h <= 0 {
			continue // a same-instant run: no duration to average over
		}
		s.ObservedHours += h
		km += l.DistanceKm
	}
	if s.ObservedHours > 0 {
		s.ObservedPaceKmh = km / s.ObservedHours
		s.PaceOK = true
	}
	return s
}

// civilDate is the calendar date of t in t's own recorded offset — the
// traveller's date, never the server's. Times parsed from the export or
// loaded from the store carry their offsets as fixed zones, so Format reads
// the right wall clock.
func civilDate(t time.Time) string { return t.Format("2006-01-02") }

func civilDays(j Journey) int {
	first, last := civilDate(j.WindowStart), civilDate(j.WindowEnd)
	touch := func(t time.Time) {
		d := civilDate(t)
		if d < first {
			first = d
		}
		if d > last {
			last = d
		}
	}
	for _, l := range j.Legs {
		touch(l.Start())
		touch(l.End())
	}
	for _, st := range j.Stops {
		touch(st.Start)
		touch(st.End)
	}
	// Lexical order is chronological for YYYY-MM-DD; the count is the
	// difference in whole days, measured on the dates alone (never on the
	// timestamps, whose offsets would shift it), plus one for inclusivity.
	a, _ := time.Parse("2006-01-02", first)
	b, _ := time.Parse("2006-01-02", last)
	return int(math.Round(b.Sub(a).Hours()/24)) + 1
}
