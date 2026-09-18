package store_test

import (
	"context"
	"testing"

	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// Self-serve deletion (phase 13 BRIEF §5): the deleted user's rows count
// zero across EVERY user-data table, the other user's are untouched, and
// the file report keeps a hash the other user still holds.
func TestDeleteUserDataIsCompleteAndScoped(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	_, _, decA, _ := seedTenant(t, s, userA)
	_, _, decB, _ := seedTenant(t, s, userB)
	if _, err := s.CreateShareLink(ctx, userA, decA.ID, "share-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateShareLink(ctx, userB, decB.ID, "share-b"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, "sess-a", userA, at(2027, 1, 1, 0)); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, "sess-b", userB, at(2027, 1, 1, 0)); err != nil {
		t.Fatal(err)
	}
	// B also holds A's photo bytes: the thumbnail must survive A's deletion.
	if _, _, err := s.InsertPhoto(ctx, userB, store.PhotoRow{
		DecisionID: decB.ID, ContentHash: "hash-" + userA, OriginalName: "same-bytes.jpg",
		TimeSource: "none", PosSource: "none", ThumbW: 512, ThumbH: 384,
	}); err != nil {
		t.Fatal(err)
	}

	// Every table holds at least one A row before — the test would be
	// vacuous otherwise.
	for _, table := range store.UserDataTables {
		n, err := s.CountUserRows(ctx, table, userA)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			t.Fatalf("seed left %s empty for A — the deletion proof would be vacuous", table)
		}
	}

	rep, err := s.DeleteUserData(ctx, userA, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range store.UserDataTables {
		if rep.Rows[table] == 0 {
			t.Errorf("report says 0 %s rows deleted", table)
		}
		n, err := s.CountUserRows(ctx, table, userA)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s still holds %d rows of A's after deletion", table, n)
		}
		nb, err := s.CountUserRows(ctx, table, userB)
		if err != nil {
			t.Fatal(err)
		}
		if nb == 0 {
			t.Errorf("%s lost B's rows to A's deletion", table)
		}
	}
	if rep.Rows["users"] != 1 {
		t.Errorf("user row not dropped: %+v", rep.Rows)
	}
	if got, _ := s.ResolveShareLink(ctx, "share-a"); got != nil {
		t.Error("A's share link still resolves")
	}
	if got, _ := s.ResolveShareLink(ctx, "share-b"); got == nil {
		t.Error("B's share link died with A")
	}
	if _, ok, _ := s.SessionUser(ctx, "sess-a"); ok {
		t.Error("A's session survived deletion")
	}
	// File report: A's record hash is orphaned; A's photo hash is B's too.
	if len(rep.OrphanThumbs) != 1 || rep.OrphanThumbs[0] != "rec-"+userA {
		t.Errorf("orphan thumbs = %v — want exactly [rec-%s]", rep.OrphanThumbs, userA)
	}

	// The authless shape: 'self' keeps its user row.
	if _, _, _, _ = seedTenant(t, s, store.SelfUser); true {
		rep, err := s.DeleteUserData(ctx, store.SelfUser, false)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Rows["users"] != 0 {
			t.Errorf("mode-off deletion dropped the self row")
		}
		if err := s.CreateSession(ctx, "sess-self", store.SelfUser, at(2027, 1, 1, 0)); err != nil {
			t.Errorf("self row gone after mode-off deletion: %v", err)
		}
	}
}
