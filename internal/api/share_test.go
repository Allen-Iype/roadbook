package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"roadbook/internal/api"
	"roadbook/internal/auth"
	"roadbook/internal/auth/authtest"
	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// Share links (phase 13 CP4) through the generated mux and AuthMiddleware:
// mint, open signed-out, photos of both provenances reachable only through
// the token, revoke → 404, unshared and dismissed adventures unreachable,
// and — in oidc mode — the view working with no session at all while the
// owner-side operations stay the owner's.

func newShareServer(t *testing.T, svc *auth.Service) (*httptest.Server, *store.Store, string) {
	t.Helper()
	s := storetest.Open(t)
	photosDir := t.TempDir()
	photos := store.PhotoFiles{Dir: photosDir}
	if err := photos.Init(); err != nil {
		t.Fatal(err)
	}
	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), Photos: photos, Auth: svc}
	ts := httptest.NewServer(api.HandlerFromMux(
		api.NewStrictHandler(srv, []api.StrictMiddlewareFunc{srv.AuthMiddleware}), http.NewServeMux()))
	t.Cleanup(ts.Close)
	return ts, s, photosDir
}

// seedShareable gives a user one run with two candidates — the first
// confirmed with an attached photo (thumbnail on disk) and one photo record
// inside its span, the second undecided — and returns both candidate ids,
// the decision, and the photo/record ids.
func seedShareable(t *testing.T, s *store.Store, photosDir, user string) (confirmedID, undecidedID int64, dec store.DecisionRow, photoID, recordID int64) {
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
		Activities: []domain.Activity{{
			Start: at(2026, 4, 2, 8), End: at(2026, 4, 2, 12),
			From: &domain.LatLng{Lat: 12.3456, Lon: 45.6789}, To: &domain.LatLng{Lat: 12.9, Lon: 45.9},
			DistanceM: 70000, Mode: "IN_BUS",
		}},
		Points: []domain.PathPoint{
			{Time: at(2026, 4, 2, 9), Loc: &domain.LatLng{Lat: 12.5, Lon: 45.7}},
			{Time: at(2026, 4, 2, 10), Loc: &domain.LatLng{Lat: 12.6, Lon: 45.8}},
			{Time: at(2026, 4, 3, 10), Loc: &domain.LatLng{Lat: 12.9, Lon: 45.9}},
		},
	}
	if _, err := s.ImportObservations(ctx, user, impID, "phone-timeline", obs, 0); err != nil {
		t.Fatal(err)
	}
	mk := func(start, end domain.LatLng, from, to int) detect.Candidate {
		return detect.Candidate{Start: at(2026, 4, from, 8), End: at(2026, 4, to, 20), Days: 2.5,
			Dest: end, DestKm: 300, TrackKm: 700, Stops: 2, ObsCount: 40}
	}
	if _, err := s.SaveRun(ctx, user, detect.DefaultParams(), detect.Result{
		Bases: []detect.Base{},
		Candidates: []detect.Candidate{
			mk(domain.LatLng{}, domain.LatLng{Lat: 12.9, Lon: 45.9}, 2, 4),
			mk(domain.LatLng{}, domain.LatLng{Lat: 13.9, Lon: 46.9}, 10, 12),
		},
	}); err != nil {
		t.Fatal(err)
	}
	_, cands, err := s.LatestRun(ctx, user)
	if err != nil || len(cands) != 2 {
		t.Fatalf("latest run: %d candidates, err %v", len(cands), err)
	}
	name := "Seed journey"
	dec, err = s.InsertDecision(ctx, user, "confirmed", &name, cands[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(photosDir, "thumb-"+user+".jpg"), []byte("jpeg-bytes-"+user), 0o600); err != nil {
		t.Fatal(err)
	}
	taken := at(2026, 4, 2, 9)
	off := 19800
	lat, lon := 12.5, 45.7
	photo, _, err := s.InsertPhoto(ctx, user, store.PhotoRow{
		DecisionID: dec.ID, ContentHash: "thumb-" + user, OriginalName: "seed.jpg",
		TakenAt: &taken, TakenOffsetSec: &off, TimeSource: "gps", Lat: &lat, Lon: &lon, PosSource: "exif",
		ThumbW: 512, ThumbH: 384,
	})
	if err != nil {
		t.Fatal(err)
	}
	recImp, err := s.BeginImport(ctx, user, "photos", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(photosDir, "rec-"+user+".jpg"), []byte("rec-bytes-"+user), 0o600); err != nil {
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
	recs, err := s.ListPhotoRecords(ctx, user, recImp)
	if err != nil || len(recs) != 1 {
		t.Fatalf("records: %d, err %v", len(recs), err)
	}
	return cands[0].ID, cands[1].ID, dec, photo.ID, recs[0].ID
}

func do(t *testing.T, c *http.Client, method, url string, cookies []*http.Cookie) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

func TestShareLinksAuthOff(t *testing.T) {
	ts, s, photosDir := newShareServer(t, nil)
	c := http.DefaultClient
	ctx := context.Background()
	confirmedID, undecidedID, dec, photoID, recordID := seedShareable(t, s, photosDir, store.SelfUser)

	// Unshared by construction: nothing resolves before a link exists.
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/0123456789abcdef0123456789abcdef", nil); resp.StatusCode != 404 {
		t.Fatalf("unknown token: %d — want 404", resp.StatusCode)
	}
	// Only confirmed adventures are shareable.
	if resp, _ := do(t, c, http.MethodPost, ts.URL+"/candidates/"+itoa(undecidedID)+"/shares", nil); resp.StatusCode != 409 {
		t.Fatalf("share undecided: %d — want 409", resp.StatusCode)
	}
	if resp, _ := do(t, c, http.MethodPost, ts.URL+"/candidates/999999/shares", nil); resp.StatusCode != 404 {
		t.Fatalf("share stale id: %d — want 404", resp.StatusCode)
	}

	resp, body := do(t, c, http.MethodPost, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", nil)
	if resp.StatusCode != 201 {
		t.Fatalf("create: %d %s", resp.StatusCode, body)
	}
	var created api.ShareLinkCreated
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}
	if len(created.Token) != 32 {
		t.Fatalf("token %q — want 32 hex chars (128 bits)", created.Token)
	}
	// Hash-only storage: the raw token appears in no row.
	if got, _ := s.ResolveShareLink(ctx, created.Token); got != nil {
		t.Fatal("raw token resolves as its own hash — storage is not hash-only")
	}

	var list api.ShareLinkList
	resp, body = do(t, c, http.MethodGet, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", nil)
	if err := json.Unmarshal(body, &list); resp.StatusCode != 200 || err != nil || len(list.Shares) != 1 || list.Shares[0].Id != created.Id {
		t.Fatalf("list: %d %s", resp.StatusCode, body)
	}

	// The view: everything the owner's plate draws, with the honesty
	// channel intact — every leg names its kind.
	resp, body = do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("shared view: %d %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("X-Robots-Tag"); !strings.Contains(got, "noindex") {
		t.Errorf("X-Robots-Tag = %q — want noindex", got)
	}
	var shared api.SharedAdventure
	if err := json.Unmarshal(body, &shared); err != nil {
		t.Fatal(err)
	}
	if shared.Name != "Seed journey" || !shared.SpanStart.Equal(at(2026, 4, 2, 8)) {
		t.Errorf("shared cover = %q %s", shared.Name, shared.SpanStart)
	}
	if len(shared.Journey.Legs) == 0 {
		t.Fatal("shared journey has no legs")
	}
	for i, l := range shared.Journey.Legs {
		if l.Kind != api.LegKindObserved && l.Kind != api.LegKindGap {
			t.Errorf("leg %d kind %q — the honesty channel must survive to the stranger", i, l.Kind)
		}
		if l.Kind == api.LegKindGap && l.GapKind == nil {
			t.Errorf("gap leg %d carries no gap_kind", i)
		}
	}
	if len(shared.Photos) != 1 || shared.Photos[0].Id != photoID {
		t.Errorf("shared photos = %+v — want the one attached photo", shared.Photos)
	}
	if len(shared.ImportPhotos) != 1 || shared.ImportPhotos[0].Id != recordID {
		t.Errorf("shared import photos = %+v — want the one span-joined record", shared.ImportPhotos)
	}
	// Placement rode along: the attached photo sits on the drawn route.
	if shared.Photos[0].PlaceKind == nil {
		t.Errorf("attached photo unplaced on the shared view")
	}

	// Thumbnails through the token, and only the adventure's own.
	resp, body = do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token+"/photos/"+itoa(photoID)+"/thumbnail", nil)
	if resp.StatusCode != 200 || string(body) != "jpeg-bytes-self" {
		t.Errorf("shared photo thumb: %d %q", resp.StatusCode, body)
	}
	resp, body = do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token+"/import-photos/"+itoa(recordID)+"/thumbnail", nil)
	if resp.StatusCode != 200 || string(body) != "rec-bytes-self" {
		t.Errorf("shared record thumb: %d %q", resp.StatusCode, body)
	}
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token+"/photos/"+itoa(photoID+1000)+"/thumbnail", nil); resp.StatusCode != 404 {
		t.Errorf("thumb of a photo not on the adventure: %d — want 404", resp.StatusCode)
	}
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token+"/import-photos/"+itoa(recordID+1000)+"/thumbnail", nil); resp.StatusCode != 404 {
		t.Errorf("thumb of a record not on the adventure: %d — want 404", resp.StatusCode)
	}

	// A dismissed-after-sharing adventure closes its links without
	// anyone revoking them.
	_, cands, _ := s.LatestRun(ctx, store.SelfUser)
	if _, err := s.UpdateDecision(ctx, store.SelfUser, dec.ID, "dismissed", nil, cands[0]); err != nil {
		t.Fatal(err)
	}
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, nil); resp.StatusCode != 404 {
		t.Errorf("shared view of a since-dismissed adventure: %d — want 404", resp.StatusCode)
	}
	name := "Seed journey"
	if _, err := s.UpdateDecision(ctx, store.SelfUser, dec.ID, "confirmed", &name, cands[0]); err != nil {
		t.Fatal(err)
	}
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, nil); resp.StatusCode != 200 {
		t.Errorf("shared view after re-confirming: %d — want 200", resp.StatusCode)
	}

	// Revoke: 204, then the token and both thumbnails are dead, and a
	// second revoke finds nothing.
	if resp, _ := do(t, c, http.MethodDelete, ts.URL+"/shares/"+itoa(created.Id), nil); resp.StatusCode != 204 {
		t.Fatalf("revoke: %d — want 204", resp.StatusCode)
	}
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, nil); resp.StatusCode != 404 {
		t.Errorf("revoked token view: %d — want 404", resp.StatusCode)
	}
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token+"/photos/"+itoa(photoID)+"/thumbnail", nil); resp.StatusCode != 404 {
		t.Errorf("revoked token thumb: %d — want 404", resp.StatusCode)
	}
	if resp, _ := do(t, c, http.MethodDelete, ts.URL+"/shares/"+itoa(created.Id), nil); resp.StatusCode != 404 {
		t.Errorf("second revoke: %d — want 404", resp.StatusCode)
	}
	resp, body = do(t, c, http.MethodGet, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", nil)
	if err := json.Unmarshal(body, &list); err != nil || len(list.Shares) != 0 {
		t.Errorf("list after revoke: %s", body)
	}
}

func TestShareLinksOIDC(t *testing.T) {
	iss := authtest.New(t)
	svc, err := auth.New(context.Background(), auth.Config{
		IssuerURL: iss.URL, ClientID: "test-client", ClientSecret: "test-secret",
		RedirectURL: "http://127.0.0.1:3000/api/auth/callback",
	})
	if err != nil {
		t.Fatal(err)
	}
	ts, s, photosDir := newShareServer(t, svc)
	c := noRedirect()
	ctx := context.Background()

	sessA := signIn(t, c, ts.URL, iss, "subject-a", "a@example.com")
	sessB := signIn(t, c, ts.URL, iss, "subject-b", "b@example.com")
	userA, _, err := s.SessionUser(ctx, auth.HashToken(sessA.Value))
	if err != nil {
		t.Fatal(err)
	}
	confirmedID, _, _, photoID, _ := seedShareable(t, s, photosDir, userA)

	// Owner-side operations need a session.
	if resp, _ := do(t, c, http.MethodPost, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", nil); resp.StatusCode != 401 {
		t.Fatalf("create signed out: %d — want 401", resp.StatusCode)
	}
	// B cannot share A's adventure: A's candidate id is not in B's run.
	if resp, _ := do(t, c, http.MethodPost, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", []*http.Cookie{sessB}); resp.StatusCode != 404 {
		t.Fatalf("B sharing A's candidate: %d — want 404", resp.StatusCode)
	}
	resp, body := do(t, c, http.MethodPost, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", []*http.Cookie{sessA})
	if resp.StatusCode != 201 {
		t.Fatalf("A create: %d %s", resp.StatusCode, body)
	}
	var created api.ShareLinkCreated
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}

	// The view opens with NO session — the token is the credential.
	resp, body = do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("signed-out shared view: %d %s", resp.StatusCode, body)
	}
	var shared api.SharedAdventure
	if err := json.Unmarshal(body, &shared); err != nil || shared.Name != "Seed journey" || len(shared.Photos) != 1 {
		t.Fatalf("signed-out shared view body: %s", body)
	}
	if resp, body := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token+"/photos/"+itoa(photoID)+"/thumbnail", nil); resp.StatusCode != 200 || string(body) != "jpeg-bytes-"+userA {
		t.Errorf("signed-out thumb: %d %q", resp.StatusCode, body)
	}
	// And with B's session — a stranger who happens to be signed in sees
	// the same view, not their own data through it.
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, []*http.Cookie{sessB}); resp.StatusCode != 200 {
		t.Errorf("B opening A's link: %d — want 200", resp.StatusCode)
	}

	// B cannot list or revoke A's link.
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", []*http.Cookie{sessB}); resp.StatusCode != 404 {
		t.Errorf("B listing A's shares: %d — want 404", resp.StatusCode)
	}
	if resp, _ := do(t, c, http.MethodDelete, ts.URL+"/shares/"+itoa(created.Id), []*http.Cookie{sessB}); resp.StatusCode != 404 {
		t.Errorf("B revoking A's link: %d — want 404", resp.StatusCode)
	}
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, nil); resp.StatusCode != 200 {
		t.Errorf("link after B's revoke attempt: %d — want 200 (untouched)", resp.StatusCode)
	}
	// A revokes; the link dies for everyone.
	if resp, _ := do(t, c, http.MethodDelete, ts.URL+"/shares/"+itoa(created.Id), []*http.Cookie{sessA}); resp.StatusCode != 204 {
		t.Errorf("A revoking: %d — want 204", resp.StatusCode)
	}
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, nil); resp.StatusCode != 404 {
		t.Errorf("after A's revoke: %d — want 404", resp.StatusCode)
	}
}
