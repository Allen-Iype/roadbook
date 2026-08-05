package api_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"roadbook/internal/api"
	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// TestDeletePhotoOrderingUnderFileFailure asserts the failure mode the
// row-first ordering was chosen for (docs/phase-4/DECISIONS.md): when the
// file removal fails after the row is gone, the surviving state must be an
// orphaned file — sweepable garbage — never a surviving row pointing at a
// missing file, which would be a permanently broken image on the page. The
// failure is injected by replacing the thumbnail with a non-empty directory
// of the same name, which os.Remove cannot delete.
func TestDeletePhotoOrderingUnderFileFailure(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	ist := time.FixedZone("", 19800)
	start := time.Date(2026, 3, 1, 8, 0, 0, 0, ist)
	end := time.Date(2026, 3, 4, 20, 0, 0, 0, ist)
	if _, err := s.SaveRun(ctx, detect.DefaultParams(), detect.Result{
		Bases: []detect.Base{},
		Candidates: []detect.Candidate{{
			Start: start, End: end, Days: 3.5,
			Dest: domain.LatLng{Lat: 12.3, Lon: 45.6}, DestKm: 300, TrackKm: 700,
			Modes: []detect.ModeCount{},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	_, rows, err := s.LatestRun(ctx)
	if err != nil || len(rows) != 1 {
		t.Fatalf("latest run: %v, %d rows", err, len(rows))
	}
	name := "Trip"
	dec, err := s.InsertDecision(ctx, "confirmed", &name, rows[0])
	if err != nil {
		t.Fatal(err)
	}

	photos := store.PhotoFiles{Dir: t.TempDir()}
	const hash = "deadbeef"
	row, _, err := s.InsertPhoto(ctx, store.PhotoRow{
		DecisionID: dec.ID, ContentHash: hash, OriginalName: "x.jpg",
		TimeSource: "none", PosSource: "none", ThumbW: 10, ThumbH: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The injected failure: a directory with a child where the thumbnail
	// file should be. os.Remove on a non-empty directory fails.
	thumbPath := filepath.Join(photos.Dir, hash+".jpg")
	if err := os.MkdirAll(filepath.Join(thumbPath, "child"), 0o755); err != nil {
		t.Fatal(err)
	}

	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), Photos: photos}
	_, err = srv.DeletePhoto(ctx, api.DeletePhotoRequestObject{Id: row.ID})
	if err == nil {
		t.Fatal("DeletePhoto reported success despite the file-delete failure — the failure must surface")
	}

	// Row first: the row must be gone even though the file step failed…
	p, gerr := s.GetPhoto(ctx, row.ID)
	if gerr != nil || p != nil {
		t.Errorf("row survived the failed delete: %+v, %v — a row without its file is a broken page", p, gerr)
	}
	// …and the orphan must still be there to sweep: detectable as a file
	// (or here, directory) in the photos dir whose hash no row references.
	if _, serr := os.Stat(thumbPath); serr != nil {
		t.Errorf("orphaned path missing: %v — the failure mode should leave sweepable garbage", serr)
	}
}
