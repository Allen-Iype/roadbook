package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"roadbook/internal/api"
	"roadbook/internal/auth"
	"roadbook/internal/auth/authtest"
	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// The CP2 verification (BRIEF §5): the full OIDC flow against the test
// issuer, through the generated mux and a real scratch database — sign-in,
// session, 401 enforcement, tenant isolation at the HTTP level, and
// observable server-side revocation.

func at(y int, mo time.Month, d, h int) time.Time {
	return time.Date(y, mo, d, h, 0, 0, 0, time.FixedZone("", 19800))
}

func newAuthServer(t *testing.T, svc *auth.Service) (*httptest.Server, *store.Store) {
	t.Helper()
	s := storetest.Open(t)
	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), Auth: svc}
	ts := httptest.NewServer(api.HandlerFromMux(
		api.NewStrictHandler(srv, []api.StrictMiddlewareFunc{srv.AuthMiddleware}), http.NewServeMux()))
	t.Cleanup(ts.Close)
	return ts, s
}

// noRedirect returns a client that hands 302s back instead of following
// them — the redirects under test point at the provider and the app shell.
func noRedirect() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func getJSON(t *testing.T, c *http.Client, url string, cookies []*http.Cookie, out any) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
	if out != nil && resp.StatusCode == 200 {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode
}

func TestAuthOffIsTheReference(t *testing.T) {
	ts, _ := newAuthServer(t, nil)
	c := noRedirect()

	var sess api.SessionInfo
	if code := getJSON(t, c, ts.URL+"/auth/session", nil, &sess); code != 200 {
		t.Fatalf("auth/session: %d", code)
	}
	if sess.Mode != api.AuthModeOff || sess.SignedIn {
		t.Fatalf("off-mode session = %+v — want mode off, signed_in false", sess)
	}
	if code := getJSON(t, c, ts.URL+"/auth/signin", nil, nil); code != 409 {
		t.Fatalf("signin in off mode: %d — want 409", code)
	}
	// Data reads need no session: the authless single-user reference.
	if code := getJSON(t, c, ts.URL+"/candidates", nil, nil); code != 200 {
		t.Fatalf("candidates in off mode: %d — want 200", code)
	}
	resp, err := c.Post(ts.URL+"/auth/signout", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatalf("signout in off mode: %d — want 204", resp.StatusCode)
	}
}

// signIn drives the whole dance for one subject and returns the session
// cookie: signin redirect → state cookie + nonce parsed → code minted at
// the issuer (the human consent stand-in) → callback → session cookie.
func signIn(t *testing.T, c *http.Client, baseURL string, iss *authtest.Issuer, subject, email string) *http.Cookie {
	t.Helper()
	resp, err := c.Get(baseURL + "/auth/signin")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Fatalf("signin: %d — want 302", resp.StatusCode)
	}
	var stateCookie *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == auth.StateCookie {
			stateCookie = ck
		}
	}
	if stateCookie == nil {
		t.Fatal("signin set no state cookie")
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state, nonce := loc.Query().Get("state"), loc.Query().Get("nonce")
	if state == "" || nonce == "" {
		t.Fatalf("authorize URL missing state/nonce: %s", loc)
	}

	code := iss.MintCode(subject, email, nonce)
	req, err := http.NewRequest(http.MethodGet,
		baseURL+"/auth/callback?code="+url.QueryEscape(code)+"&state="+url.QueryEscape(state), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(stateCookie)
	resp, err = c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Fatalf("callback: %d — want 302", resp.StatusCode)
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == auth.SessionCookie && ck.Value != "" {
			return ck
		}
	}
	t.Fatal("callback set no session cookie")
	return nil
}

func TestOIDCFlowSessionsAndTenancy(t *testing.T) {
	iss := authtest.New(t)
	svc, err := auth.New(context.Background(), auth.Config{
		IssuerURL: iss.URL, ClientID: "test-client", ClientSecret: "test-secret",
		RedirectURL: "http://127.0.0.1:3000/api/auth/callback",
	})
	if err != nil {
		t.Fatal(err)
	}
	ts, s := newAuthServer(t, svc)
	c := noRedirect()
	ctx := context.Background()

	// Signed out: data 401, session endpoint honest about it.
	if code := getJSON(t, c, ts.URL+"/candidates", nil, nil); code != 401 {
		t.Fatalf("candidates signed out: %d — want 401", code)
	}
	var sess api.SessionInfo
	if code := getJSON(t, c, ts.URL+"/auth/session", nil, &sess); code != 200 || sess.Mode != api.AuthModeOIDC || sess.SignedIn {
		t.Fatalf("signed-out session: %d %+v", code, sess)
	}

	// A tampered state must not mint a session.
	badReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/auth/callback?code=x&state=forged", nil)
	badReq.AddCookie(&http.Cookie{Name: auth.StateCookie, Value: "real.nonce"})
	resp, err := c.Do(badReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("forged-state callback: %d — want 400", resp.StatusCode)
	}

	sessA := signIn(t, c, ts.URL, iss, "subject-a", "a@example.com")
	sessB := signIn(t, c, ts.URL, iss, "subject-b", "b@example.com")

	if code := getJSON(t, c, ts.URL+"/auth/session", []*http.Cookie{sessA}, &sess); code != 200 || !sess.SignedIn || sess.Email == nil || *sess.Email != "a@example.com" {
		t.Fatalf("A's session: %d %+v", code, sess)
	}

	// Seed data as A through the store (the CLI/import path) and prove the
	// HTTP boundary keeps it A's: the cross-tenant family, one level up.
	userA, ok, err := s.SessionUser(ctx, auth.HashToken(sessA.Value))
	if err != nil || !ok {
		t.Fatalf("resolving A's session: ok=%v err=%v", ok, err)
	}
	if _, err := s.SaveRun(ctx, userA, detect.DefaultParams(), detect.Result{
		Bases: []detect.Base{},
		Candidates: []detect.Candidate{{
			Start: at(2026, 4, 2, 8), End: at(2026, 4, 4, 20),
			Days: 2.5, Dest: domain.LatLng{Lat: 12.9, Lon: 45.9},
			DestKm: 300, TrackKm: 700, Stops: 2, ObsCount: 40,
			Modes: []detect.ModeCount{{Mode: "IN_BUS", N: 3}},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	var listA, listB struct {
		Candidates []json.RawMessage `json:"candidates"`
	}
	if code := getJSON(t, c, ts.URL+"/candidates", []*http.Cookie{sessA}, &listA); code != 200 || len(listA.Candidates) != 1 {
		t.Fatalf("A sees %d candidates (code %d) — want 1", len(listA.Candidates), code)
	}
	if code := getJSON(t, c, ts.URL+"/candidates", []*http.Cookie{sessB}, &listB); code != 200 || len(listB.Candidates) != 0 {
		t.Fatalf("B sees %d candidates (code %d) — want 0", len(listB.Candidates), code)
	}

	// Sign-out revokes server-side: the same cookie value is dead after.
	soReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/auth/signout", nil)
	soReq.AddCookie(sessA)
	resp, err = c.Do(soReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatalf("signout: %d — want 204", resp.StatusCode)
	}
	if code := getJSON(t, c, ts.URL+"/candidates", []*http.Cookie{sessA}, nil); code != 401 {
		t.Fatalf("candidates with revoked session: %d — want 401", code)
	}
	// B's session is untouched by A's sign-out.
	if code := getJSON(t, c, ts.URL+"/candidates", []*http.Cookie{sessB}, nil); code != 200 {
		t.Fatalf("B after A's signout: %d — want 200", code)
	}
}
