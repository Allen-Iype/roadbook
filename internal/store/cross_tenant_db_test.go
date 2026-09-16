package store_test

import (
	"context"
	"testing"

	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// The cross-tenant family (phase 13 BRIEF §5): for every store read and
// write path, prove one user's data is invisible and untouchable from
// another user's scope. This is the test class that pays for row-scoped
// tenancy — the harness's own doc comment (2026-08-05) predicted it. A new
// store query without a case here is a review failure by standing practice.

const (
	userA = "tenant-a"
	userB = "tenant-b"
)

// seedTenant gives a user one import with observations, one detection run
// with one candidate, one decision on it, one attached photo, and one
// photo record. Everything downstream reads through these six tables.
func seedTenant(t *testing.T, s *store.Store, user string) (impID int64, cand store.CandidateRow, dec store.DecisionRow, photo store.PhotoRow) {
	t.Helper()
	ctx := context.Background()
	if err := s.EnsureUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	impID, err := s.BeginImport(ctx, user, "seed", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	obs := domain.Observations{
		Visits: []domain.Visit{{
			Start: at(2026, 4, 1, 8), End: at(2026, 4, 1, 20),
			Loc: &domain.LatLng{Lat: 12.3456, Lon: 45.6789}, SemanticType: "INFERRED_HOME",
		}},
		Activities: []domain.Activity{{
			Start: at(2026, 4, 2, 8), End: at(2026, 4, 2, 12),
			From: &domain.LatLng{Lat: 12.3456, Lon: 45.6789}, To: &domain.LatLng{Lat: 12.9, Lon: 45.9},
			DistanceM: 70000, Mode: "IN_BUS",
		}},
		Points:       []domain.PathPoint{{Time: at(2026, 4, 2, 9), Loc: &domain.LatLng{Lat: 12.5, Lon: 45.7}}},
		RawPositions: []domain.RawPosition{{Time: at(2026, 4, 2, 10), Loc: &domain.LatLng{Lat: 12.6, Lon: 45.8}, AccuracyM: 16, Source: "GPS"}},
	}
	if _, err := s.ImportObservations(ctx, user, impID, "phone-timeline", obs, 0); err != nil {
		t.Fatal(err)
	}

	if _, err := s.SaveRun(ctx, user, detect.DefaultParams(), detect.Result{
		Bases:      []detect.Base{},
		Candidates: []detect.Candidate{candidate(at(2026, 4, 2, 8), at(2026, 4, 4, 20), domain.LatLng{Lat: 12.9, Lon: 45.9})},
	}); err != nil {
		t.Fatal(err)
	}
	_, rows, err := s.LatestRun(ctx, user)
	if err != nil || len(rows) != 1 {
		t.Fatalf("latest run for %s: %d candidates, err %v", user, len(rows), err)
	}
	cand = rows[0]

	name := "Seed journey"
	dec, err = s.InsertDecision(ctx, user, "confirmed", &name, cand)
	if err != nil {
		t.Fatal(err)
	}

	photo, _, err = s.InsertPhoto(ctx, user, store.PhotoRow{
		DecisionID: dec.ID, ContentHash: "hash-" + user, OriginalName: "seed.jpg",
		TimeSource: "none", PosSource: "none", ThumbW: 512, ThumbH: 384,
	})
	if err != nil {
		t.Fatal(err)
	}

	recImp, err := s.BeginImport(ctx, user, "photos", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fix := domain.RawPosition{Time: at(2026, 4, 3, 11), Loc: &domain.LatLng{Lat: 12.7, Lon: 45.85}, AccuracyM: 25, Source: "PHOTO"}
	if _, err := s.ImportPhotos(ctx, user, recImp, []store.PhotoIngest{{
		Fix: fix,
		Record: store.PhotoRecord{
			ContentHash: "rec-" + user, OriginalName: "rec.jpg",
			TimeSource: "gps", PosSource: "exif", ThumbW: 512, ThumbH: 384,
		},
	}}); err != nil {
		t.Fatal(err)
	}
	return impID, cand, dec, photo
}

func TestCrossTenantReadsAreDisjoint(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	impA, candA, _, _ := seedTenant(t, s, userA)
	if err := s.EnsureUser(ctx, userB); err != nil {
		t.Fatal(err)
	}

	if rows, err := s.ListImports(ctx, userB); err != nil || len(rows) != 0 {
		t.Errorf("ListImports as B: %d rows, err %v — want 0", len(rows), err)
	}
	if row, err := s.GetImport(ctx, userB, impA); err != nil || row != nil {
		t.Errorf("GetImport of A's import as B: %+v, err %v — want nil", row, err)
	}
	obs, err := s.LoadObservations(ctx, userB)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(obs.Visits) + len(obs.Activities) + len(obs.Points) + len(obs.RawPositions); n != 0 {
		t.Errorf("LoadObservations as B: %d observations — want 0", n)
	}
	ji, err := s.LoadJourneyInputs(ctx, userB, candA.SpanStart, candA.SpanEnd)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(ji.Activities) + len(ji.Points) + len(ji.RawPositions); n != 0 {
		t.Errorf("LoadJourneyInputs as B: %d rows — want 0", n)
	}
	if run, cands, err := s.LatestRun(ctx, userB); err != nil || run != nil || len(cands) != 0 {
		t.Errorf("LatestRun as B: run %+v, %d candidates, err %v — want none", run, len(cands), err)
	}
	if c, err := s.LatestCandidate(ctx, userB, candA.ID); err != nil || c != nil {
		t.Errorf("LatestCandidate of A's id as B: %+v, err %v — want nil", c, err)
	}
	if decs, err := s.ListDecisions(ctx, userB); err != nil || len(decs) != 0 {
		t.Errorf("ListDecisions as B: %d rows, err %v — want 0", len(decs), err)
	}
	if recs, err := s.ListPhotoRecordsInSpan(ctx, userB, candA.SpanStart.AddDate(0, -1, 0), candA.SpanEnd.AddDate(0, 1, 0)); err != nil || len(recs) != 0 {
		t.Errorf("ListPhotoRecordsInSpan as B: %d rows, err %v — want 0", len(recs), err)
	}
}

func TestCrossTenantPhotoPathsAreDisjoint(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	_, _, decA, photoA := seedTenant(t, s, userA)
	if err := s.EnsureUser(ctx, userB); err != nil {
		t.Fatal(err)
	}

	if rows, err := s.ListPhotos(ctx, userB, decA.ID); err != nil || len(rows) != 0 {
		t.Errorf("ListPhotos of A's decision as B: %d rows, err %v — want 0", len(rows), err)
	}
	if p, err := s.GetPhoto(ctx, userB, photoA.ID); err != nil || p != nil {
		t.Errorf("GetPhoto of A's photo as B: %+v, err %v — want nil", p, err)
	}
	if deleted, err := s.DeletePhoto(ctx, userB, photoA.ID); err != nil || deleted {
		t.Errorf("DeletePhoto of A's photo as B: deleted=%v, err %v — want false", deleted, err)
	}
	if p, err := s.GetPhoto(ctx, userA, photoA.ID); err != nil || p == nil {
		t.Errorf("A's photo must survive B's delete attempt: %+v, err %v", p, err)
	}
	if all, err := s.ListAllPhotos(ctx, userB); err != nil || len(all) != 0 {
		t.Errorf("ListAllPhotos as B: %d rows, err %v — want 0", len(all), err)
	}
}

func TestCrossTenantWritesCannotTouchOtherRows(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	_, candA, decA, _ := seedTenant(t, s, userA)
	if err := s.EnsureUser(ctx, userB); err != nil {
		t.Fatal(err)
	}

	// Re-deciding A's decision from B's scope must touch nothing.
	name := "hijack"
	if _, err := s.UpdateDecision(ctx, userB, decA.ID, "dismissed", nil, candA); err == nil {
		t.Error("UpdateDecision of A's decision as B succeeded — want no-rows error")
	}
	id := decA.ID
	if err := s.DecideBulk(ctx, userB, []store.BulkDecision{{Anchor: candA, Action: "confirmed", Name: &name, UpdateID: &id}}); err != nil {
		// The bulk update silently affects zero rows inside its tx; the
		// authoritative check is A's row below.
		t.Fatalf("DecideBulk as B errored unexpectedly: %v", err)
	}
	decs, err := s.ListDecisions(ctx, userA)
	if err != nil || len(decs) != 1 {
		t.Fatalf("A's decisions after B's attempts: %d, err %v", len(decs), err)
	}
	if decs[0].Action != "confirmed" || decs[0].Name == nil || *decs[0].Name != "Seed journey" {
		t.Errorf("A's decision was altered from B's scope: %+v", decs[0])
	}
	if err := s.SetImportDetectStatus(ctx, userB, 1, "failed"); err != nil {
		t.Fatalf("SetImportDetectStatus as B errored: %v", err)
	}
}

func TestContentHashDedupeIsPerUser(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	for _, u := range []string{userA, userB} {
		if err := s.EnsureUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	obs := domain.Observations{
		Points: []domain.PathPoint{{Time: at(2026, 5, 1, 9), Loc: &domain.LatLng{Lat: 12.5, Lon: 45.7}}},
	}
	for _, u := range []string{userA, userB} {
		impID, err := s.BeginImport(ctx, u, "same-bytes", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.ImportObservations(ctx, u, impID, "phone-timeline", obs, 0)
		if err != nil {
			t.Fatal(err)
		}
		// Identical bytes, two owners: each user's first import inserts.
		if res.Inserted != 1 {
			t.Errorf("user %s inserted %d — want 1 (dedupe must scope per user)", u, res.Inserted)
		}
	}
	// The second import of the same bytes by the SAME user still dedupes.
	impID, err := s.BeginImport(ctx, userA, "same-bytes-again", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.ImportObservations(ctx, userA, impID, "phone-timeline", obs, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inserted != 0 {
		t.Errorf("same user re-import inserted %d — want 0", res.Inserted)
	}
}
