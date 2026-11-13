package journey

import (
	"sort"
	"time"
)

// Nights (phase 19): where each night of a journey was spent.
//
// A journey that touches N civil days has N-1 nights between them. For
// each, this pass opens a fixed window on the traveller's own clock —
// NightStartHour on one civil date to NightEndHour on the next — and looks
// at what the assembled journey holds inside it:
//
//   - at a stop: one stop covers at least MinNightHours of the window. If
//     several do, the one covering the most holds the night.
//   - on the move: no stop does, and moving observed legs cover at least
//     MinNightHours — an overnight bus, a night drive.
//   - not known: neither. The night is still counted; the record is too
//     thin to say where it was spent.
//
// The pass is pure and runs after Assemble without changing it, like
// Summarize and NameStops, so the journey goldens cannot move. It names
// nothing: a night at a stop points at the stop, and the stop already has
// its town (names.go).
//
// Why hours of overlap and not "the stop at 03:00": a stop that ends ten
// minutes before the sampled instant, because the phone logged a short
// walk, would lose the night to it. Why not "the stop at midnight": a
// half-hour halt across 00:00 would be a night. Both were weighed in the
// brief (docs/phase-19/BRIEF.md §1, §4A).

// NightKind is the class a night falls in.
type NightKind string

const (
	NightStop    NightKind = "stop"
	NightMoving  NightKind = "moving"
	NightUnknown NightKind = "unknown"
)

// NightParams are the pass's thresholds (invariant 3), served with every
// journey that carries nights.
type NightParams struct {
	// StartHour and EndHour bound the window on the local clock: it opens
	// at StartHour on one civil date and closes at EndHour on the next.
	StartHour int
	EndHour   int
	// MinHours is how much of the window one stop (or, failing that,
	// recorded travel) must cover to hold the night.
	MinHours float64
}

// DefaultNightParams: 22:00 to 06:00, three hours. Measured in the brief
// (§2.2): on the Android datasets the count of placed nights is nearly
// flat from two to four hours.
func DefaultNightParams() NightParams {
	return NightParams{StartHour: 22, EndHour: 6, MinHours: 3}
}

// Night is one night of a journey.
type Night struct {
	// Date is the civil date the night began on ("2026-05-22" for the
	// night of the 22nd into the 23rd).
	Date string
	Kind NightKind
	// StopIndex indexes Journey.Stops when Kind is NightStop; -1 otherwise.
	StopIndex int
	// Hours is the overlap that decided the class: the holding stop's for
	// NightStop, the moving observed legs' for NightMoving, zero for
	// NightUnknown.
	Hours float64
}

// Nights returns one Night for every pair of consecutive civil dates the
// journey touches, in order. The dates are the ones Summarize counts, so
// len(Nights(j, p)) is always Summarize(j).CivilDays - 1.
func Nights(j Journey, p NightParams) []Night {
	first, last := civilRange(j)
	day, _ := time.Parse("2006-01-02", first)
	end, _ := time.Parse("2006-01-02", last)
	clock := newClock(j)

	var out []Night
	for ; day.Before(end); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		loc := clock.at(date, p.StartHour)
		open := time.Date(day.Year(), day.Month(), day.Day(), p.StartHour, 0, 0, 0, loc)
		shut := time.Date(day.Year(), day.Month(), day.Day()+1, p.EndHour, 0, 0, 0, loc)

		n := Night{Date: date, Kind: NightUnknown, StopIndex: -1}
		best, bestIdx := 0.0, -1
		for i, s := range j.Stops {
			// Strictly more: of two equal overlaps the earlier stop holds.
			if h := overlapHours(s.Start, s.End, open, shut); h > best {
				best, bestIdx = h, i
			}
		}
		if bestIdx >= 0 && best >= p.MinHours {
			n.Kind, n.StopIndex, n.Hours = NightStop, bestIdx, best
			out = append(out, n)
			continue
		}
		var moving float64
		for _, l := range j.Legs {
			// The summary's "moving observed leg": not a fix, and with a
			// distance — a stationary run is not travel.
			if l.Kind != LegObserved || len(l.Points) < 2 || l.DistanceKm == 0 {
				continue
			}
			moving += overlapHours(l.Start(), l.End(), open, shut)
		}
		if moving >= p.MinHours {
			n.Kind, n.Hours = NightMoving, moving
		}
		out = append(out, n)
	}
	return out
}

func overlapHours(a0, a1, b0, b1 time.Time) float64 {
	if a0.Before(b0) {
		a0 = b0
	}
	if a1.After(b1) {
		a1 = b1
	}
	if !a1.After(a0) {
		return 0
	}
	return a1.Sub(a0).Hours()
}

// nightClock answers "which offset was the traveller on when this night
// began". Every timestamp in a journey keeps the offset it was recorded
// in; a window needs one. The rule: the offset of the last recorded
// timestamp whose own wall clock is at or before the window's opening,
// and the journey's first timestamp's offset when none is. Inside one
// time zone that is simply the zone; across a flight, each night is read
// on the clock the traveller was on when it began.
type nightClock struct {
	walls []string // wall clocks, "2006-01-02T15:04:05", in time order
	locs  []*time.Location
}

func newClock(j Journey) nightClock {
	ts := []time.Time{j.WindowStart, j.WindowEnd}
	for _, l := range j.Legs {
		ts = append(ts, l.Start(), l.End())
	}
	for _, s := range j.Stops {
		ts = append(ts, s.Start, s.End)
	}
	sort.SliceStable(ts, func(a, b int) bool { return ts[a].Before(ts[b]) })
	c := nightClock{}
	for _, t := range ts {
		c.walls = append(c.walls, t.Format("2006-01-02T15:04:05"))
		c.locs = append(c.locs, t.Location())
	}
	return c
}

func (c nightClock) at(date string, hour int) *time.Location {
	opening := date + "T" + time.Date(0, 1, 1, hour, 0, 0, 0, time.UTC).Format("15:04:05")
	loc := c.locs[0]
	for i, w := range c.walls {
		if w <= opening {
			loc = c.locs[i]
		}
	}
	return loc
}
