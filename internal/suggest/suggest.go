// Package suggest is the name-suggestion seam (BRIEF §1.7): pluggable with an
// offline fallback, the same shape invariant 7 demands of routing. A
// suggestion is prefill for the confirm step's name input — never applied
// automatically, and the product works identically when it suggests nothing.
package suggest

import (
	"context"

	"roadbook/internal/domain"
)

// Suggestion is a proposed adventure name and where it came from. An empty
// Name is a real answer: "nothing to suggest", stated rather than hidden.
type Suggestion struct {
	Name   string
	Source string
}

// Suggester proposes a name for an adventure from its destination
// coordinate.
type Suggester interface {
	Suggest(ctx context.Context, dest domain.LatLng) (Suggestion, error)
}

// Null is the offline implementation: it suggests nothing and says so. The
// default — a self-hosted product must not make surprise network calls.
type Null struct{}

func (Null) Suggest(context.Context, domain.LatLng) (Suggestion, error) {
	return Suggestion{Source: "none"}, nil
}
