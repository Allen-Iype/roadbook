package api_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"roadbook/internal/api"
	"roadbook/internal/detect"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

const corpusDir = "../../testdata/photos/corpus"

func newPhotoImportServer(t *testing.T) (*httptest.Server, *store.Store, string) {
	t.Helper()
	s := storetest.Open(t)
	photosDir := t.TempDir()
	photos := store.PhotoFiles{Dir: photosDir}
	if err := photos.Init(); err != nil {
		t.Fatal(err)
	}
	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), Photos: photos}
	ts := httptest.NewServer(api.HandlerFromMux(api.NewStrictHandler(srv, nil), http.NewServeMux()))
	t.Cleanup(ts.Close)
	return ts, s, photosDir
}

func corpusMultipart(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(corpusDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		fw, err := w.CreateFormFile("file", e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, w.FormDataContentType()
}

// TestPhotoImportEndToEnd is CP2's core walk: the committed corpus uploads
// as a photo batch, per-file verdicts come back, the import row is already
// completed in the 202, auto-detect lands the corpus's pinned 2 candidates,
// records and thumbnails exist exactly where §4D says — and the identical
// batch again is fully idempotent.
func TestPhotoImportEndToEnd(t *testing.T) {
	ts, s, photosDir := newPhotoImportServer(t)
	ctx := context.Background()

	body, ct := corpusMultipart(t)
	resp, err := http.Post(ts.URL+"/imports/photos", ct, body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, b)
	}
	res := decodeJSON[api.PhotoImportResult](t, resp.Body)
	resp.Body.Close()

	if res.Import.Status != "completed" {
		t.Errorf("202 import status = %q — the photo path imports before responding", res.Import.Status)
	}
	if res.Import.RawPositions != 55 {
		t.Errorf("raw_positions = %d, want 55 (the corpus's fixes)", res.Import.RawPositions)
	}
	if res.Import.DetectedFormat == nil || *res.Import.DetectedFormat != "photos" {
		t.Errorf("detected_format = %v, want photos", res.Import.DetectedFormat)
	}

	counts := map[api.PhotoImportFileStatus]int{}
	for _, f := range res.Files {
		counts[f.Status]++
	}
	want := map[api.PhotoImportFileStatus]int{
		"fix": 55, "no_position": 2, "sidecar_paired": 1,
	}
	for st, n := range want {
		if counts[st] != n {
			t.Errorf("verdict %s: %d files, want %d (all: %v)", st, counts[st], n, counts)
		}
	}

	// Auto-detect follows in the background; the corpus pins 2 candidates.
	pollImport(t, ts, res.Import.Id, func(i api.Import) bool {
		return i.DetectStatus != nil && *i.DetectStatus == "completed"
	})
	run, cands, err := s.LatestRun(ctx)
	if err != nil || run == nil {
		t.Fatalf("latest run: %v, %v", run, err)
	}
	if len(cands) != 2 {
		t.Errorf("auto-detect found %d candidates, the corpus pins 2", len(cands))
	}

	// Records: one per fix; the HEIC pair carries no thumbnail dims.
	recs, err := s.ListPhotoRecords(ctx, res.Import.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 55 {
		t.Fatalf("got %d photo records, want 55", len(recs))
	}
	heicNoThumb, jpegThumbs := 0, 0
	for _, r := range recs {
		if r.ThumbW == 0 {
			heicNoThumb++
		} else {
			jpegThumbs++
		}
	}
	if heicNoThumb != 2 || jpegThumbs != 53 {
		t.Errorf("thumbnail split = %d without / %d with, want 2/53", heicNoThumb, jpegThumbs)
	}

	// Thumbnail files on disk: exactly the decodable fixes, named by hash.
	files, err := os.ReadDir(photosDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 53 {
		t.Errorf("photos dir holds %d files, want 53 thumbnails", len(files))
	}

	// The identical batch again: 0 new fixes, 0 new records, no new files.
	body2, ct2 := corpusMultipart(t)
	resp2, err := http.Post(ts.URL+"/imports/photos", ct2, body2)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("duplicate status %d", resp2.StatusCode)
	}
	res2 := decodeJSON[api.PhotoImportResult](t, resp2.Body)
	resp2.Body.Close()
	if res2.Import.Inserted == nil || *res2.Import.Inserted != 0 {
		t.Errorf("duplicate batch inserted = %v, want 0", res2.Import.Inserted)
	}
	dupRecs, err := s.ListPhotoRecords(ctx, res2.Import.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(dupRecs) != 0 {
		t.Errorf("duplicate batch created %d records, want 0", len(dupRecs))
	}
	pollImport(t, ts, res2.Import.Id, func(i api.Import) bool {
		return i.DetectStatus != nil && *i.DetectStatus == "completed"
	})
}

func TestPhotoImportRejections(t *testing.T) {
	ts, _, _ := newPhotoImportServer(t)

	// No file parts at all → 400.
	var empty bytes.Buffer
	w := multipart.NewWriter(&empty)
	w.Close()
	resp, err := http.Post(ts.URL+"/imports/photos", w.FormDataContentType(), &empty)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty batch status = %d, want 400", resp.StatusCode)
	}

	// A wrong file gets a per-file verdict with the sniffer's message —
	// the batch itself still lands (one bad file must not sink the rest).
	body, ct := multipartBytes(t, "not_a_photo.pdf", []byte("%PDF-1.7 nonsense"))
	resp2, err := http.Post(ts.URL+"/imports/photos", ct, body)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("status %d: %s", resp2.StatusCode, b)
	}
	res := decodeJSON[api.PhotoImportResult](t, resp2.Body)
	resp2.Body.Close()
	if len(res.Files) != 1 || res.Files[0].Status != "unsupported" || res.Files[0].Message == nil {
		t.Errorf("PDF verdict = %+v, want unsupported with a message", res.Files)
	}
	if res.Import.RawPositions != 0 {
		t.Errorf("PDF-only batch stored %d fixes", res.Import.RawPositions)
	}
}
