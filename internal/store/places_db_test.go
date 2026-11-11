package store_test

import (
	"context"
	"testing"
	"time"

	"roadbook/internal/places"
	"roadbook/internal/store/storetest"
)

// The -if-empty startup path rests on the same two store facts as countries
// and states: a fresh schema counts zero, and a loaded table counts exactly
// what was loaded — twice over, because ReplacePlaces replaces. This loads
// the WHOLE embedded list, unlike the states test's Iceland subset: the
// single-statement load is the thing under test, and its time is printed
// because every fresh instance pays it once at first start.
func TestReplaceAndCountPlaces(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	n, err := s.CountPlaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("fresh schema: want 0 places, got %d", n)
	}

	list, err := places.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := s.ReplacePlaces(ctx, list, places.SourceOSM); err != nil {
		t.Fatal(err)
	}
	t.Logf("loaded %d places in %s", len(list), time.Since(start).Round(time.Millisecond))
	if err := s.ReplacePlaces(ctx, list, places.SourceOSM); err != nil { // replace, not append
		t.Fatal(err)
	}

	n, err = s.CountPlaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(list)) {
		t.Fatalf("after double load: want %d places, got %d", len(list), n)
	}

	// A smaller list replaces the larger one outright.
	if err := s.ReplacePlaces(ctx, list[:3], "test"); err != nil {
		t.Fatal(err)
	}
	if n, _ = s.CountPlaces(ctx); n != 3 {
		t.Fatalf("after replacing with 3: got %d", n)
	}
}
