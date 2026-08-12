package store_test

import (
	"context"
	"testing"

	"roadbook/internal/countries"
	"roadbook/internal/store/storetest"
)

// The -if-empty startup path (phase 8 §3E) rests on two store facts: a
// fresh schema counts zero, and a populated table counts what was loaded —
// so `countries -if-empty` fires exactly once per instance lifetime and
// never overwrites an operator's own load. The wholesale-replace semantics
// of ReplaceCountries are exercised on the way (load twice, count once).
func TestCountCountries(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	n, err := s.CountCountries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("fresh schema: want 0 countries, got %d", n)
	}

	list, err := countries.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceCountries(ctx, list); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceCountries(ctx, list); err != nil { // replace, not append
		t.Fatal(err)
	}

	n, err = s.CountCountries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(list)) {
		t.Fatalf("after double load: want %d countries, got %d", len(list), n)
	}
}
