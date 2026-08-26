package journey

import (
	"sort"
	"time"

	"roadbook/internal/domain"
)

// ModeKm is one mode's summed distance, in the source's own labels.
type ModeKm struct {
	Mode string
	Km   float64
}

// ModeBreakdown sums activity distance by mode over the window — the
// source-asserted per-mode figures (phase 11 BRIEF §6.2), deliberately NOT
// part of Assemble: the golden contract pins assembly's output byte-for-byte,
// and these figures are a different kind of claim. Assembly measures
// geometry; this echoes Google's own labels and distances, which are guesses
// with recorded failures (a 1,023 km "motorcycling" relocation; probability
// 0.00 modes), and the pipeline itself trusts speed over mode for flight
// classification. Display must label the figures source-asserted and never
// sum them against measured geometry.
//
// An activity counts iff it overlaps the window, and counts in full — no
// pro-rating: slicing a guessed distance by time pretends a precision the
// source does not have. Zero-distance activities contribute nothing and are
// dropped. Pure: same inputs, same output; order is km-descending, ties by
// mode name for determinism.
func ModeBreakdown(acts []domain.Activity, winStart, winEnd time.Time) []ModeKm {
	byMode := map[string]float64{}
	for _, a := range acts {
		if a.DistanceM <= 0 {
			continue
		}
		if a.End.Before(winStart) || a.Start.After(winEnd) {
			continue
		}
		mode := a.Mode
		if mode == "" {
			mode = "UNKNOWN"
		}
		byMode[mode] += a.DistanceM / 1000
	}
	out := make([]ModeKm, 0, len(byMode))
	for m, km := range byMode {
		out = append(out, ModeKm{Mode: m, Km: km})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Km != out[j].Km {
			return out[i].Km > out[j].Km
		}
		return out[i].Mode < out[j].Mode
	})
	return out
}
