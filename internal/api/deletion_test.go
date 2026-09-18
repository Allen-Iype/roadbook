package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"roadbook/internal/api"
	"roadbook/internal/auth"
	"roadbook/internal/auth/authtest"
	"roadbook/internal/store"
)

// DELETE /me through the mux in both modes: rows gone, files swept only
// when unreferenced, the session cleared and the account forgotten in
// oidc mode, and the single user's instance empty again in mode off.
func TestDeleteMyDataAuthOff(t *testing.T) {
	ts, s, photosDir := newShareServer(t, nil)
	c := http.DefaultClient
	ctx := context.Background()
	confirmedID, _, _, _, _ := seedShareable(t, s, photosDir, store.SelfUser)
	if resp, _ := do(t, c, http.MethodPost, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", nil); resp.StatusCode != 201 {
		t.Fatalf("share: %d", resp.StatusCode)
	}

	resp, body := do(t, c, http.MethodDelete, ts.URL+"/me", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("delete: %d %s", resp.StatusCode, body)
	}
	var rep api.DeletionReport
	if err := json.Unmarshal(body, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Rows["decisions"] != 1 || rep.Rows["share_links"] != 1 || rep.Rows["photos"] != 1 || rep.Rows["photo_records"] != 1 {
		t.Errorf("report rows = %v", rep.Rows)
	}
	if rep.ThumbnailsRemoved != 2 {
		t.Errorf("thumbnails removed = %d — want 2 (photo + record)", rep.ThumbnailsRemoved)
	}
	for _, name := range []string{"thumb-self.jpg", "rec-self.jpg"} {
		if _, err := os.Stat(filepath.Join(photosDir, name)); !os.IsNotExist(err) {
			t.Errorf("%s still on disk after deletion", name)
		}
	}
	if resp.Header.Get("Set-Cookie") != "" {
		t.Error("mode off set a cookie on deletion — there is no session to clear")
	}
	// Empty again: the instance behaves like a fresh one.
	var list struct{ Candidates []json.RawMessage }
	if code := getJSON(t, c, ts.URL+"/candidates", nil, &list); code != 200 || len(list.Candidates) != 0 {
		t.Errorf("candidates after deletion: %d, %d rows", code, len(list.Candidates))
	}
	var imports struct{ Imports []json.RawMessage }
	if code := getJSON(t, c, ts.URL+"/imports", nil, &imports); code != 200 || len(imports.Imports) != 0 {
		t.Errorf("imports after deletion: %d, %d rows", code, len(imports.Imports))
	}
	_ = ctx
}

func TestDeleteMyDataOIDC(t *testing.T) {
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
	userA, _, _ := s.SessionUser(ctx, auth.HashToken(sessA.Value))
	userB, _, _ := s.SessionUser(ctx, auth.HashToken(sessB.Value))
	candA, _, _, _, _ := seedShareable(t, s, photosDir, userA)
	candB, _, _, _, _ := seedShareable(t, s, photosDir, userB)

	if resp, _ := do(t, c, http.MethodDelete, ts.URL+"/me", nil); resp.StatusCode != 401 {
		t.Fatalf("signed-out delete: %d — want 401", resp.StatusCode)
	}
	resp, body := do(t, c, http.MethodDelete, ts.URL+"/me", []*http.Cookie{sessA})
	if resp.StatusCode != 200 {
		t.Fatalf("A delete: %d %s", resp.StatusCode, body)
	}
	cleared := false
	for _, ck := range resp.Cookies() {
		if ck.Name == auth.SessionCookie && ck.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("deletion did not clear the session cookie")
	}
	// A is signed out for real, and the account is gone.
	if code := getJSON(t, c, ts.URL+"/candidates", []*http.Cookie{sessA}, nil); code != 401 {
		t.Errorf("A's old session after deletion: %d — want 401", code)
	}
	if email, err := s.GetUserEmail(ctx, userA); err != nil || email != "" {
		t.Errorf("A's user row survived: email %q err %v", email, err)
	}
	for _, table := range store.UserDataTables {
		if n, _ := s.CountUserRows(ctx, table, userA); n != 0 {
			t.Errorf("%s still holds %d of A's rows", table, n)
		}
	}
	// B untouched, including B's files.
	if resp, _ := do(t, c, http.MethodGet, ts.URL+"/candidates/"+itoa(candB)+"/journey", []*http.Cookie{sessB}); resp.StatusCode != 200 {
		t.Errorf("B's journey after A's deletion: %d", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(photosDir, "thumb-"+userB+".jpg")); err != nil {
		t.Errorf("B's thumbnail gone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(photosDir, "thumb-"+userA+".jpg")); !os.IsNotExist(err) {
		t.Errorf("A's thumbnail still on disk")
	}
	_ = candA
	// Signing in again as the same subject starts from nothing.
	sessA2 := signIn(t, c, ts.URL, iss, "subject-a", "a@example.com")
	var list struct{ Candidates []json.RawMessage }
	if code := getJSON(t, c, ts.URL+"/candidates", []*http.Cookie{sessA2}, &list); code != 200 || len(list.Candidates) != 0 {
		t.Errorf("re-signed-in A: %d, %d candidates — want a fresh account", code, len(list.Candidates))
	}
}
