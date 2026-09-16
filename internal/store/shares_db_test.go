package store_test

import (
	"context"
	"errors"
	"testing"

	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// Share links (phase 13 CP4): the token-hash lifecycle, and the
// cross-tenant cases the standing family requires for every new query —
// listing and revoking are scoped, creation refuses another user's
// decision, and resolution is global BY DESIGN (the token is the
// credential) but hands back the owner every downstream read is scoped by.

func TestShareLinkLifecycle(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	_, _, dec, _ := seedTenant(t, s, userA)

	link, err := s.CreateShareLink(ctx, userA, dec.ID, "hash-1")
	if err != nil {
		t.Fatal(err)
	}
	if link.UserID != userA || link.DecisionID != dec.ID {
		t.Fatalf("created link = %+v", link)
	}
	if _, err := s.CreateShareLink(ctx, userA, dec.ID, "hash-2"); err != nil {
		t.Fatal(err)
	}
	rows, err := s.ListShareLinks(ctx, userA, dec.ID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("ListShareLinks: %d rows, err %v — want 2", len(rows), err)
	}

	got, err := s.ResolveShareLink(ctx, "hash-1")
	if err != nil || got == nil || got.ID != link.ID {
		t.Fatalf("ResolveShareLink(hash-1) = %+v, err %v", got, err)
	}
	if got, err := s.ResolveShareLink(ctx, "never-minted"); err != nil || got != nil {
		t.Fatalf("ResolveShareLink(unknown) = %+v, err %v — want nil", got, err)
	}

	deleted, err := s.DeleteShareLink(ctx, userA, link.ID)
	if err != nil || !deleted {
		t.Fatalf("DeleteShareLink: deleted=%v err=%v", deleted, err)
	}
	if got, _ := s.ResolveShareLink(ctx, "hash-1"); got != nil {
		t.Fatalf("revoked hash still resolves: %+v", got)
	}
	if deleted, _ := s.DeleteShareLink(ctx, userA, link.ID); deleted {
		t.Fatal("deleting an already-revoked link reported true")
	}
	rows, _ = s.ListShareLinks(ctx, userA, dec.ID)
	if len(rows) != 1 || rows[0].ID == link.ID {
		t.Fatalf("after revoke: %+v", rows)
	}
}

func TestCrossTenantShareLinksAreDisjoint(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	_, _, decA, _ := seedTenant(t, s, userA)
	if err := s.EnsureUser(ctx, userB); err != nil {
		t.Fatal(err)
	}

	// B cannot mint a link to A's adventure, even knowing its decision id.
	if _, err := s.CreateShareLink(ctx, userB, decA.ID, "hash-b"); !errors.Is(err, store.ErrNotOwned) {
		t.Fatalf("CreateShareLink for A's decision as B: err %v — want ErrNotOwned", err)
	}

	linkA, err := s.CreateShareLink(ctx, userA, decA.ID, "hash-a")
	if err != nil {
		t.Fatal(err)
	}
	if rows, err := s.ListShareLinks(ctx, userB, decA.ID); err != nil || len(rows) != 0 {
		t.Errorf("ListShareLinks of A's decision as B: %d rows, err %v — want 0", len(rows), err)
	}
	if deleted, err := s.DeleteShareLink(ctx, userB, linkA.ID); err != nil || deleted {
		t.Errorf("DeleteShareLink of A's link as B: deleted=%v err=%v — want false", deleted, err)
	}
	// Resolution is global on purpose — and what it returns is A's
	// ownership, which is what scopes every read the token unlocks.
	got, err := s.ResolveShareLink(ctx, "hash-a")
	if err != nil || got == nil || got.UserID != userA {
		t.Fatalf("ResolveShareLink after B's attempts: %+v, err %v — want A's live link", got, err)
	}
}
