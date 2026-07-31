package detect

// Helpers that replicate Python behaviour the reference detector depends on.
// They exist so the Go port and the prototype agree bit-for-bit; do not
// "simplify" them to the Go-idiomatic equivalents, which behave differently.

import (
	"strconv"
	"time"
)

// pyRound replicates Python's round(x, n): correctly-rounded decimal rounding
// with ties-to-even (banker's rounding). Go's math.Round(x*10ⁿ)/10ⁿ is
// half-away-from-zero *after* a lossy multiplication and disagrees in the last
// digit often enough to break regression parity. strconv formats the exact
// binary value with the same correct rounding CPython uses.
func pyRound(x float64, n int) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	return v
}

// civilDate replicates Python's datetime.date(): the calendar date in the
// timestamp's own UTC offset, returned as midnight UTC so dates compare and
// sort as instants. Taking the date in UTC instead would shift late-evening
// timestamps (e.g. +05:30) to the previous day and move home-base era edges.
func civilDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// bisectLeft / bisectRight replicate Python's bisect module exactly, including
// midpoint arithmetic, so both implementations probe the same indices even if
// the input ordering assumption were ever violated.
func bisectLeft(ts []time.Time, target time.Time) int {
	lo, hi := 0, len(ts)
	for lo < hi {
		mid := (lo + hi) / 2
		if ts[mid].Before(target) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func bisectRight(ts []time.Time, target time.Time) int {
	lo, hi := 0, len(ts)
	for lo < hi {
		mid := (lo + hi) / 2
		if target.Before(ts[mid]) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo
}
