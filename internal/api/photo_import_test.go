package api_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"roadbook/internal/api"
	"roadbook/internal/detect"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

const corpusDir = "../../testdata/photos/corpus"

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

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

	// CP4: the read-time span join. Each pinned candidate lists exactly the
	// records whose capture falls inside its span, placed against the drawn
	// geometry; nothing is stored to make this true.
	totalListed := 0
	for _, c := range cands {
		resp, err := http.Get(ts.URL + "/candidates/" + itoa(c.ID) + "/import-photos")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("import-photos status %d for candidate %d", resp.StatusCode, c.ID)
		}
		list := decodeJSON[api.ImportPhotoList](t, resp.Body)
		resp.Body.Close()
		if len(list.Photos) == 0 {
			t.Errorf("candidate %d: no import photos in span — the trip bursts should land here", c.ID)
		}
		if _, ok := list.Params["photo_far_warn_m"]; !ok {
			t.Errorf("params missing photo_far_warn_m (invariant 3 echo)")
		}
		placed := 0
		for _, p := range list.Photos {
			if p.TakenAt.Before(c.SpanStart) || p.TakenAt.After(c.SpanEnd) {
				t.Errorf("candidate %d lists photo %s taken %v outside span", c.ID, p.OriginalName, p.TakenAt)
			}
			if p.PlaceKind != nil {
				placed++
			}
		}
		if placed == 0 {
			t.Errorf("candidate %d: no record placed on the journey", c.ID)
		}
		totalListed += len(list.Photos)
	}
	if totalListed == 0 || totalListed > 55 {
		t.Errorf("span join listed %d records total, want within (0, 55]", totalListed)
	}

	// Thumbnails through the record endpoint: JPEG serves, HEIC 404s
	// honestly, unknown id 404s.
	var jpegID, heicID int64
	for _, r := range recs {
		if r.ThumbW > 0 && jpegID == 0 {
			jpegID = r.ID
		}
		if r.ThumbW == 0 && heicID == 0 {
			heicID = r.ID
		}
	}
	for _, tc := range []struct {
		id   int64
		want int
	}{{jpegID, http.StatusOK}, {heicID, http.StatusNotFound}, {999999, http.StatusNotFound}} {
		resp, err := http.Get(ts.URL + "/import-photos/" + itoa(tc.id) + "/thumbnail")
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Errorf("thumbnail for record %d: status %d, want %d", tc.id, resp.StatusCode, tc.want)
		}
	}

	// A photo-sourced journey has no activities: mode_breakdown is ABSENT
	// (phase 11 §6.2) — the display says "no mode record", never zeros.
	jresp, err := http.Get(ts.URL + "/candidates/" + itoa(cands[0].ID) + "/journey")
	if err != nil {
		t.Fatal(err)
	}
	if jresp.StatusCode != http.StatusOK {
		t.Fatalf("journey status %d", jresp.StatusCode)
	}
	jd := decodeJSON[api.Journey](t, jresp.Body)
	jresp.Body.Close()
	if jd.ModeBreakdown != nil {
		t.Errorf("photo-sourced journey mode_breakdown = %v, want absent", *jd.ModeBreakdown)
	}

	// A stale candidate id (no such candidate in the latest run) → 404.
	resp404, err := http.Get(ts.URL + "/candidates/999999/import-photos")
	if err != nil {
		t.Fatal(err)
	}
	resp404.Body.Close()
	if resp404.StatusCode != http.StatusNotFound {
		t.Errorf("stale candidate import-photos status = %d, want 404", resp404.StatusCode)
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

// TestBulkDecisions walks the atomic bulk-triage endpoint (phase 11 §6.1)
// over the corpus's two pinned candidates: mixed confirm+dismiss lands, a
// stale id rejects the whole batch (nothing applied), and the validation
// rules (confirm needs a name; one decision per candidate) answer 400.
func TestBulkDecisions(t *testing.T) {
	ts, s, _ := newPhotoImportServer(t)
	ctx := context.Background()

	body, ct := corpusMultipart(t)
	resp, err := http.Post(ts.URL+"/imports/photos", ct, body)
	if err != nil {
		t.Fatal(err)
	}
	res := decodeJSON[api.PhotoImportResult](t, resp.Body)
	resp.Body.Close()
	pollImport(t, ts, res.Import.Id, func(i api.Import) bool {
		return i.DetectStatus != nil && *i.DetectStatus == "completed"
	})
	_, cands, err := s.LatestRun(ctx)
	if err != nil || len(cands) != 2 {
		t.Fatalf("corpus candidates: %d, %v", len(cands), err)
	}

	post := func(payload string) (*http.Response, string) {
		resp, err := http.Post(ts.URL+"/candidates/decisions", "application/json",
			bytes.NewBufferString(payload))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, string(b)
	}
	id0, id1 := cands[0].ID, cands[1].ID

	// Validation answers 400 and stores nothing.
	for _, bad := range []string{
		`{"decisions":[]}`,
		fmt.Sprintf(`{"decisions":[{"id":%d,"action":"confirmed"}]}`, id0),
		fmt.Sprintf(`{"decisions":[{"id":%d,"action":"dismissed"},{"id":%d,"action":"dismissed"}]}`, id0, id0),
	} {
		if resp, b := post(bad); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("payload %s: status %d (%s), want 400", bad, resp.StatusCode, b)
		}
	}

	// A stale id anywhere rejects the WHOLE batch — atomicity's visible half.
	resp404, _ := post(fmt.Sprintf(`{"decisions":[{"id":%d,"action":"dismissed"},{"id":999999,"action":"dismissed"}]}`, id0))
	if resp404.StatusCode != http.StatusNotFound {
		t.Errorf("stale-id batch status %d, want 404", resp404.StatusCode)
	}
	if decs, _ := s.ListDecisions(ctx); len(decs) != 0 {
		t.Errorf("failed batch left %d decisions — atomicity broken", len(decs))
	}

	// The real sweep: one confirm with a name, one dismiss, one request.
	respOK, bodyOK := post(fmt.Sprintf(
		`{"decisions":[{"id":%d,"action":"confirmed","name":"Höfn run"},{"id":%d,"action":"dismissed"}]}`, id0, id1))
	if respOK.StatusCode != http.StatusOK {
		t.Fatalf("bulk decide status %d: %s", respOK.StatusCode, bodyOK)
	}

	// Both visible through the list, attached to the right candidates.
	listResp, err := http.Get(ts.URL + "/candidates")
	if err != nil {
		t.Fatal(err)
	}
	list := decodeJSON[api.CandidateList](t, listResp.Body)
	listResp.Body.Close()
	byID := map[int64]api.Candidate{}
	for _, c := range list.Candidates {
		byID[c.Id] = c
	}
	if d := byID[id0].Decision; d == nil || d.Action != "confirmed" || d.Name == nil || *d.Name != "Höfn run" {
		t.Errorf("candidate %d decision = %+v, want confirmed 'Höfn run'", id0, d)
	}
	if d := byID[id1].Decision; d == nil || d.Action != "dismissed" {
		t.Errorf("candidate %d decision = %+v, want dismissed", id1, d)
	}

	// Re-deciding in bulk updates in place: total decisions stays 2.
	respRe, bodyRe := post(fmt.Sprintf(`{"decisions":[{"id":%d,"action":"dismissed"}]}`, id0))
	if respRe.StatusCode != http.StatusOK {
		t.Fatalf("bulk re-decide status %d: %s", respRe.StatusCode, bodyRe)
	}
	if decs, _ := s.ListDecisions(ctx); len(decs) != 2 {
		t.Errorf("re-decide created rows: %d decisions, want 2", len(decs))
	}
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
