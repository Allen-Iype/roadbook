package backup_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"roadbook/internal/backup"
	"roadbook/internal/domain"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

var ist = time.FixedZone("", 19800)

func anchor(day int) store.CandidateRow {
	return store.CandidateRow{
		SpanStart: time.Date(2026, 3, day, 8, 0, 0, 0, ist),
		SpanEnd:   time.Date(2026, 3, day+2, 20, 0, 0, 0, ist),
		Dest:      domain.LatLng{Lat: 12.3456, Lon: 45.6789},
	}
}

func str(s string) *string { return &s }

// TestBackupRestoreRoundTrip is the §3D contract: an archive written from one
// instance restores into another — rows by durable identity, timestamps
// preserved, thumbnail bytes carried — and restoring again (or into the
// source) skips everything, because merge means skip-the-overlap.
func TestBackupRestoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	src := storetest.Open(t)
	srcFiles := store.PhotoFiles{Dir: filepath.Join(t.TempDir(), "photos")}
	if err := srcFiles.Init(); err != nil {
		t.Fatal(err)
	}

	confirmed, err := src.InsertDecision(ctx, "confirmed", str("March hills"), anchor(1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.InsertDecision(ctx, "dismissed", nil, anchor(10)); err != nil {
		t.Fatal(err)
	}

	jpeg := []byte("not-really-jpeg-but-bytes-are-bytes")
	if err := srcFiles.WriteThumb("hash1", jpeg); err != nil {
		t.Fatal(err)
	}
	taken := time.Date(2026, 3, 2, 12, 30, 0, 0, ist)
	off := 19800
	lat, lon := 12.35, 45.68
	if _, _, err := src.InsertPhoto(ctx, store.PhotoRow{
		DecisionID: confirmed.ID, ContentHash: "hash1", OriginalName: "IMG_1.jpg",
		TakenAt: &taken, TakenOffsetSec: &off, TimeSource: "gps",
		Lat: &lat, Lon: &lon, PosSource: "exif", ThumbW: 512, ThumbH: 384,
	}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	man, warnings, err := backup.Write(ctx, src, srcFiles, &buf, time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings on a consistent instance: %v", warnings)
	}
	if man.Decisions != 2 || man.Photos != 1 || man.Thumbnails != 1 {
		t.Fatalf("manifest counts = %d/%d/%d, want 2/1/1", man.Decisions, man.Photos, man.Thumbnails)
	}
	archive := buf.Bytes()

	// Restore into an empty instance.
	dst := storetest.Open(t)
	dstFiles := store.PhotoFiles{Dir: filepath.Join(t.TempDir(), "photos")}
	if err := dstFiles.Init(); err != nil {
		t.Fatal(err)
	}
	rep, err := backup.Restore(ctx, dst, dstFiles, bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	if rep.DecisionsRestored != 2 || rep.PhotosRestored != 1 || rep.ThumbsWritten != 1 {
		t.Fatalf("restore = %+v, want 2 decisions, 1 photo, 1 thumbnail restored", rep)
	}

	decs, err := dst.ListDecisions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(decs) != 2 {
		t.Fatalf("restored decisions = %d, want 2", len(decs))
	}
	var got store.DecisionRow
	for _, d := range decs {
		if d.Action == "confirmed" {
			got = d
		}
	}
	if got.Name == nil || *got.Name != "March hills" {
		t.Errorf("restored name = %v, want March hills", got.Name)
	}
	if !got.AnchorStart.Equal(confirmed.AnchorStart) || !got.CreatedAt.Equal(confirmed.CreatedAt) {
		t.Errorf("timestamps not preserved: anchor %v vs %v, created %v vs %v",
			got.AnchorStart, confirmed.AnchorStart, got.CreatedAt, confirmed.CreatedAt)
	}

	photos, err := dst.ListAllPhotos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(photos) != 1 || photos[0].DecisionID != got.ID {
		t.Fatalf("restored photo rows = %+v, want 1 attached to the restored confirmed decision", photos)
	}
	if !photos[0].TakenAt.Equal(taken) {
		t.Errorf("taken_at not preserved: %v vs %v", photos[0].TakenAt, taken)
	}
	if b, err := dstFiles.ReadThumb("hash1"); err != nil || !bytes.Equal(b, jpeg) {
		t.Errorf("thumbnail bytes not carried: %v", err)
	}

	// Restoring the same archive again is a reported no-op.
	rep2, err := backup.Restore(ctx, dst, dstFiles, bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	if rep2.DecisionsRestored != 0 || rep2.DecisionsSkipped != 2 ||
		rep2.PhotosRestored != 0 || rep2.PhotosSkipped != 1 || rep2.ThumbsWritten != 0 {
		t.Errorf("second restore = %+v, want everything skipped", rep2)
	}

	// Restoring into the source is equally a no-op — merge by identity.
	rep3, err := backup.Restore(ctx, src, srcFiles, bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	if rep3.DecisionsRestored != 0 || rep3.PhotosRestored != 0 {
		t.Errorf("restore into source = %+v, want everything skipped", rep3)
	}
}

// TestBackupSkipsPhotoWithoutThumbnail: a row whose file is gone is excluded
// from the archive with a warning — a backup must not preserve a state it
// would be a bug to create (row without file = permanently broken image).
func TestBackupSkipsPhotoWithoutThumbnail(t *testing.T) {
	ctx := context.Background()
	s := storetest.Open(t)
	files := store.PhotoFiles{Dir: filepath.Join(t.TempDir(), "photos")}
	if err := files.Init(); err != nil {
		t.Fatal(err)
	}

	d, err := s.InsertDecision(ctx, "confirmed", str("trip"), anchor(1))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.InsertPhoto(ctx, store.PhotoRow{
		DecisionID: d.ID, ContentHash: "gone", OriginalName: "lost.jpg",
		TimeSource: "none", PosSource: "none", ThumbW: 1, ThumbH: 1,
	}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	man, warnings, err := backup.Write(ctx, s, files, &buf, time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if man.Photos != 0 || man.Thumbnails != 0 {
		t.Errorf("manifest = %+v, want the fileless photo excluded", man)
	}
	if len(warnings) != 1 {
		t.Errorf("warnings = %v, want exactly one naming lost.jpg", warnings)
	}
}
