package store_test

import (
	"context"
	"testing"

	"roadbook/internal/store/storetest"
)

// A kept trip's name and notes (phase 17, #3): EditDecisionText changes
// those two and nothing else, only on a confirmed row, and a later
// re-decide never blanks the notes.
func TestEditDecisionText(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	_, cand, dec, _ := seedTenant(t, s, userA)

	notes := "Line one.\n\n  Indented line two."
	got, ok, err := s.EditDecisionText(ctx, userA, dec.ID, "Renamed", &notes)
	if err != nil || !ok {
		t.Fatalf("edit: ok %v, err %v", ok, err)
	}
	if got.Name == nil || *got.Name != "Renamed" || got.Notes == nil || *got.Notes != notes {
		t.Fatalf("edit returned %+v", got)
	}
	if got.Action != "confirmed" || !got.AnchorStart.Equal(dec.AnchorStart) || got.AnchorDest != dec.AnchorDest {
		t.Errorf("edit touched the action or anchor: %+v", got)
	}
	if !got.UpdatedAt.After(dec.UpdatedAt) && !got.UpdatedAt.Equal(dec.UpdatedAt) {
		t.Errorf("updated_at went backwards")
	}

	// A re-decide (here: re-keeping under a new name) leaves the notes.
	again := "Kept again"
	re, err := s.UpdateDecision(ctx, userA, dec.ID, "confirmed", &again, cand)
	if err != nil {
		t.Fatal(err)
	}
	if re.Notes == nil || *re.Notes != notes {
		t.Errorf("re-decide blanked or changed the notes: %v", re.Notes)
	}

	// nil notes clears them.
	cleared, ok, err := s.EditDecisionText(ctx, userA, dec.ID, "Renamed", nil)
	if err != nil || !ok || cleared.Notes != nil {
		t.Errorf("clear: ok %v, err %v, notes %v", ok, err, cleared.Notes)
	}

	// A dismissed decision has no name to edit.
	if _, err := s.UpdateDecision(ctx, userA, dec.ID, "dismissed", nil, cand); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.EditDecisionText(ctx, userA, dec.ID, "Sneaky", &notes); err != nil || ok {
		t.Errorf("edit of a dismissed decision: ok %v, err %v — want not found", ok, err)
	}

	// An unknown id is not found, not an error.
	if _, ok, err := s.EditDecisionText(ctx, userA, dec.ID+999, "x", nil); err != nil || ok {
		t.Errorf("unknown id: ok %v, err %v", ok, err)
	}
}
