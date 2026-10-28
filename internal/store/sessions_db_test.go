package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// The user cap (phase 15 BRIEF §9): UpsertOIDCUser admits a new subject only
// while the count of OIDC accounts is below the cap, never refuses a
// returning one, and counts GLOBALLY by design — like share-token
// resolution, this is the one users-table read that has no caller scope,
// because the question it answers ("is there a seat?") is about the
// instance, not about any one user. Everything downstream of the id it
// returns is scoped as before.

func TestUserCapZeroIsUnlimited(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	for i, sub := range []string{"sub-1", "sub-2", "sub-3"} {
		if _, err := s.UpsertOIDCUser(ctx, sub, "x@example.com", 0); err != nil {
			t.Fatalf("cap 0, sign-in %d: %v", i, err)
		}
	}
}

func TestUserCapAdmitsReturnsRefuses(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	idA, err := s.UpsertOIDCUser(ctx, "sub-a", "a@example.com", 1)
	if err != nil {
		t.Fatalf("A first sign-in at cap 1: %v", err)
	}
	// Returning: same id, email refreshed, never refused.
	again, err := s.UpsertOIDCUser(ctx, "sub-a", "a2@example.com", 1)
	if err != nil || again != idA {
		t.Fatalf("A returning: id %q err %v — want %q, nil", again, err, idA)
	}
	if email, _ := s.GetUserEmail(ctx, idA); email != "a2@example.com" {
		t.Fatalf("email after return = %q", email)
	}
	// New at the cap: refused, nothing written.
	if _, err := s.UpsertOIDCUser(ctx, "sub-b", "b@example.com", 1); !errors.Is(err, store.ErrUserCapReached) {
		t.Fatalf("B at cap 1: err %v — want ErrUserCapReached", err)
	}
	// Cap raised: B is in.
	idB, err := s.UpsertOIDCUser(ctx, "sub-b", "b@example.com", 2)
	if err != nil {
		t.Fatalf("B at cap 2: %v", err)
	}
	// Cap lowered below the population: both still return; a third is refused.
	for _, c := range []struct{ sub, want string }{{"sub-a", idA}, {"sub-b", idB}} {
		got, err := s.UpsertOIDCUser(ctx, c.sub, "", 1)
		if err != nil || got != c.want {
			t.Fatalf("%s returning at cap 1 below population: %q %v", c.sub, got, err)
		}
	}
	if _, err := s.UpsertOIDCUser(ctx, "sub-c", "", 1); !errors.Is(err, store.ErrUserCapReached) {
		t.Fatalf("C at cap 1 with 2 accounts: err %v", err)
	}
	// Deletion frees a seat (cap 2, two accounts, A deletes itself → C admitted).
	if _, err := s.DeleteUserData(ctx, idA, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpsertOIDCUser(ctx, "sub-c", "", 2); err != nil {
		t.Fatalf("C after A deleted, cap 2: %v", err)
	}
}

func TestUserCapSerialisesFirstSignIns(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = s.UpsertOIDCUser(ctx, "race-"+string(rune('a'+i)), "", 1)
		}(i)
	}
	wg.Wait()
	admitted := 0
	for i, err := range errs {
		switch {
		case err == nil:
			admitted++
		case errors.Is(err, store.ErrUserCapReached):
		default:
			t.Fatalf("goroutine %d: unexpected error %v", i, err)
		}
	}
	if admitted != 1 {
		t.Fatalf("admitted %d of %d concurrent first sign-ins at cap 1 — want exactly 1", admitted, n)
	}
}
