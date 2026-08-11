package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"roadbook/internal/api"
	"roadbook/internal/detect"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

const demoExport = "../../testdata/demo/demo.json"

// newUploadServer builds the full HTTP stack — generated mux, strict
// handler, real scratch database, temp uploads dir — because the risk under
// test includes the multipart transport itself, not only the handler logic.
func newUploadServer(t *testing.T) (*httptest.Server, *store.Store, store.UploadFiles) {
	t.Helper()
	s := storetest.Open(t)
	uploads := store.UploadFiles{Dir: t.TempDir()}
	if err := uploads.Init(); err != nil {
		t.Fatal(err)
	}
	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), Uploads: uploads}
	ts := httptest.NewServer(api.HandlerFromMux(api.NewStrictHandler(srv, nil), http.NewServeMux()))
	t.Cleanup(ts.Close)
	return ts, s, uploads
}

func multipartUpload(t *testing.T, path string) (body *bytes.Buffer, contentType string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return multipartBytes(t, filepath.Base(path), data)
}

func multipartBytes(t *testing.T, filename string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, w.FormDataContentType()
}

func postUpload(t *testing.T, ts *httptest.Server, body io.Reader, contentType string) *http.Response {
	t.Helper()
	resp, err := http.Post(ts.URL+"/imports", contentType, body)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeJSON[T any](t *testing.T, r io.Reader) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// pollImport polls GET /imports/{id} until done reports true or the
// deadline passes — the front door's own mechanism (BRIEF §1.2).
func pollImport(t *testing.T, ts *httptest.Server, id int64, done func(api.Import) bool) api.Import {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := http.Get(fmt.Sprintf("%s/imports/%d", ts.URL, id))
		if err != nil {
			t.Fatal(err)
		}
		imp := decodeJSON[api.Import](t, resp.Body)
		resp.Body.Close()
		if done(imp) {
			return imp
		}
		if time.Now().After(deadline) {
			t.Fatalf("import %d never reached the awaited state: %+v", id, imp)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func uploadsDirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// TestUploadImportEndToEnd is the checkpoint's core walk: upload the demo
// export, poll to completion, and reach candidates with zero CLI commands
// (BRIEF §7 checkpoint 1). Then upload the identical bytes again: an
// inserted=0 row and no second retained file.
func TestUploadImportEndToEnd(t *testing.T) {
	ts, s, uploads := newUploadServer(t)

	body, ct := multipartUpload(t, demoExport)
	resp := postUpload(t, ts, body, ct)
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, b)
	}
	accepted := decodeJSON[api.Import](t, resp.Body)
	resp.Body.Close()
	if accepted.Status != "running" {
		t.Errorf("202 body status = %q, want running (the response is the row at accept time)", accepted.Status)
	}
	if accepted.SourceLabel != "demo.json" {
		t.Errorf("source label = %q, want the uploaded filename", accepted.SourceLabel)
	}
	if accepted.ContentHash == nil {
		t.Fatal("202 body has no content_hash")
	}

	imp := pollImport(t, ts, accepted.Id, func(i api.Import) bool {
		return i.Status == "completed" && i.DetectStatus != nil && *i.DetectStatus == "completed"
	})
	if imp.Inserted == nil || *imp.Inserted == 0 {
		t.Fatalf("first import inserted = %v, want > 0", imp.Inserted)
	}
	if imp.Visits == 0 || imp.Activities == 0 || imp.Points == 0 {
		t.Errorf("counters missing: %+v", imp)
	}

	// The retained file is byte-identical to what was uploaded (§3C).
	kept, err := os.ReadFile(uploads.Path(*accepted.ContentHash))
	if err != nil {
		t.Fatalf("retained file: %v", err)
	}
	orig, _ := os.ReadFile(demoExport)
	if !bytes.Equal(kept, orig) {
		t.Error("retained file differs from the uploaded bytes")
	}

	// Auto-detect landed a real run: the demo pins 3 candidates
	// (internal/detect/demo_test.go).
	run, cands, err := s.LatestRun(context.Background())
	if err != nil || run == nil {
		t.Fatalf("latest run: %v, %v", run, err)
	}
	if len(cands) != 3 {
		t.Errorf("auto-detect found %d candidates, demo pins 3", len(cands))
	}

	// Duplicate upload: same bytes → inserted 0, one retained file.
	body2, ct2 := multipartUpload(t, demoExport)
	resp2 := postUpload(t, ts, body2, ct2)
	if resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("duplicate upload status %d", resp2.StatusCode)
	}
	accepted2 := decodeJSON[api.Import](t, resp2.Body)
	resp2.Body.Close()
	dup := pollImport(t, ts, accepted2.Id, func(i api.Import) bool {
		return i.Status == "completed" && i.DetectStatus != nil && *i.DetectStatus != "running"
	})
	if dup.Inserted == nil || *dup.Inserted != 0 {
		t.Errorf("duplicate import inserted = %v, want 0 — the row must be able to say 'nothing new'", dup.Inserted)
	}
	if n := len(uploadsDirEntries(t, uploads.Dir)); n != 1 {
		t.Errorf("uploads dir holds %d files after duplicate upload, want 1", n)
	}
}

// TestUploadImportRejection: the synchronous sniff (BRIEF §1.3). A wrong
// file is answered in the response with its stable label, nothing is
// retained, and no imports row exists.
func TestUploadImportRejection(t *testing.T) {
	ts, s, uploads := newUploadServer(t)

	body, ct := multipartBytes(t, "holiday.pdf", []byte("%PDF-1.4 not a location export"))
	resp := postUpload(t, ts, body, ct)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", resp.StatusCode)
	}
	rej := decodeJSON[api.ImportRejection](t, resp.Body)
	resp.Body.Close()
	if rej.DetectedFormat == nil || *rej.DetectedFormat != "pdf" {
		t.Errorf("detected_format = %v, want pdf — the front door maps this to a walkthrough", rej.DetectedFormat)
	}
	if rej.Error == "" {
		t.Error("rejection carries no message")
	}
	if n := len(uploadsDirEntries(t, uploads.Dir)); n != 0 {
		t.Errorf("uploads dir holds %d files after a rejection, want 0 — rejected bytes are not retained", n)
	}
	rows, err := s.ListImports(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("%d import rows after a rejection, want 0 — no row until sniff passes", len(rows))
	}
}

// TestUploadImportAbortMidStream: an upload that dies mid-transfer leaves
// nothing behind — no row, no temp file (BRIEF §1.2's all-or-nothing rule,
// checkpoint 1's deliberate-abort visible). The truncation is simulated by
// sending a multipart body cut off inside the file part, which surfaces to
// the handler exactly as a client abort does: a read error mid-part.
func TestUploadImportAbortMidStream(t *testing.T) {
	ts, s, uploads := newUploadServer(t)

	full, ct := multipartBytes(t, "Timeline.json", bytes.Repeat([]byte(`{"semanticSegments":[]}`), 1000))
	cut := full.Bytes()[:full.Len()/2]
	resp := postUpload(t, ts, bytes.NewReader(cut), ct)
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Logf("note: truncated body answered %d (any non-2xx is acceptable)", resp.StatusCode)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Fatalf("truncated upload got %d — must not be accepted", resp.StatusCode)
	}

	if n := uploadsDirEntries(t, uploads.Dir); len(n) != 0 {
		t.Errorf("uploads dir holds %v after an aborted upload, want empty — no temp litter", n)
	}
	rows, err := s.ListImports(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("%d import rows after an aborted upload, want 0", len(rows))
	}
}

// TestUploadImportParseFailureVisible: a file that passes the sniff but
// breaks mid-parse lands as a failed row with the taxonomy's label — the
// asynchronous rejection moment (BRIEF §1.3).
func TestUploadImportParseFailureVisible(t *testing.T) {
	ts, _, _ := newUploadServer(t)

	// Sniffable head, truncated tail: the parser's "truncated" verdict.
	truncated := `{"semanticSegments":[{"startTime":"2026-01-01T10:00:00.000+00:00",`
	body, ct := multipartBytes(t, "Timeline.json", []byte(truncated))
	resp := postUpload(t, ts, body, ct)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d, want 202 — the sniff cannot see a truncated tail", resp.StatusCode)
	}
	accepted := decodeJSON[api.Import](t, resp.Body)
	resp.Body.Close()

	imp := pollImport(t, ts, accepted.Id, func(i api.Import) bool { return i.Status == "failed" })
	// The stable label is the assertion — it is the queryable evidence and
	// the front door's redirection key; the message is rewordable prose.
	if imp.DetectedFormat == nil || *imp.DetectedFormat != "truncated" {
		t.Errorf("detected_format = %v, want truncated", imp.DetectedFormat)
	}
	if imp.Error == nil || *imp.Error == "" {
		t.Error("failed row carries no message")
	}
	if imp.DetectStatus != nil {
		t.Errorf("detect_status = %v on a failed import, want absent — detection never started", *imp.DetectStatus)
	}
}

// TestUploadImportMissingFilePart: a request with no file part is a 400
// with guidance, not a 500.
func TestUploadImportMissingFilePart(t *testing.T) {
	ts, _, _ := newUploadServer(t)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("label", "no file here"); err != nil {
		t.Fatal(err)
	}
	w.Close()
	resp := postUpload(t, ts, &buf, w.FormDataContentType())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", resp.StatusCode)
	}
}
