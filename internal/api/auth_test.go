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
	return newCappedAuthServer(t, svc, 0)
}

// newCappedAuthServer is newAuthServer with a user cap (phase 15 BRIEF §9);
// 0 is the unlimited default every other test runs under.
func newCappedAuthServer(t *testing.T, svc *auth.Service, cap int) (*httptest.Server, *store.Store) {
	t.Helper()
	s := storetest.Open(t)
	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), Auth: svc, UserCap: cap}
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
// signInRaw drives the whole dance for one identity and hands back the
// callback response untouched, so a test can assert a refusal (phase 15:
// the user cap answers 302 to the sign-in page's full state with no
// cookie) as easily as a success.
func signInRaw(t *testing.T, c *http.Client, baseURL string, iss *authtest.Issuer, subject, email string) *http.Response {
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
	return resp
}

// signIn is signInRaw asserting success: a 302 to the app root carrying a
// session cookie, which it returns.
func signIn(t *testing.T, c *http.Client, baseURL string, iss *authtest.Issuer, subject, email string) *http.Cookie {
	t.Helper()
	resp := signInRaw(t, c, baseURL, iss, subject, email)
	if resp.StatusCode != 302 {
		t.Fatalf("callback: %d — want 302", resp.StatusCode)
	}
	if ck := sessionCookie(resp); ck != nil {
		return ck
	}
	t.Fatal("callback set no session cookie")
	return nil
}

func sessionCookie(resp *http.Response) *http.Cookie {
	for _, ck := range resp.Cookies() {
		if ck.Name == auth.SessionCookie && ck.Value != "" {
			return ck
		}
	}
	return nil
}

// The user cap (phase 15 BRIEF §9), end to end through the mux: at cap 1
// the first identity is admitted, an unknown second is refused with a 302
// to the sign-in page's full state and NO cookie, the first keeps
// returning, the second stays refused, and the first deleting itself
// frees the seat the second then takes. The cap is observable only through
// sign-in behaviour — no count endpoint exists.
func TestUserCapRefusesUnknownAdmitsReturning(t *testing.T) {
	ctx := context.Background()
	iss := authtest.New(t)
	svc, err := auth.New(ctx, auth.Config{
		IssuerURL: iss.URL, ClientID: "test-client", ClientSecret: "test-secret",
		RedirectURL: "http://127.0.0.1:3000/api/auth/callback",
	})
	if err != nil {
		t.Fatal(err)
	}
	ts, _ := newCappedAuthServer(t, svc, 1)
	c := noRedirect()

	sessA := signIn(t, c, ts.URL, iss, "subject-a", "a@example.com")

	refused := signInRaw(t, c, ts.URL, iss, "subject-b", "b@example.com")
	if refused.StatusCode != 302 || refused.Header.Get("Location") != "/signin?full=1" {
		t.Fatalf("B at cap: %d %q — want 302 /signin?full=1", refused.StatusCode, refused.Header.Get("Location"))
	}
	if sessionCookie(refused) != nil || refused.Header.Get("Set-Cookie") != "" {
		t.Fatalf("B at cap received a cookie: %q", refused.Header.Get("Set-Cookie"))
	}

	// A returns; B is still refused.
	if ck := signIn(t, c, ts.URL, iss, "subject-a", "a@example.com"); ck == nil {
		t.Fatal("A could not return")
	}
	if r := signInRaw(t, c, ts.URL, iss, "subject-b", "b@example.com"); r.Header.Get("Location") != "/signin?full=1" {
		t.Fatalf("B second attempt: %d %q", r.StatusCode, r.Header.Get("Location"))
	}

	// A deletes itself → the seat frees → B is admitted.
	if resp, body := do(t, c, http.MethodDelete, ts.URL+"/me", []*http.Cookie{sessA}); resp.StatusCode != 200 {
		t.Fatalf("A DELETE /me: %d %s", resp.StatusCode, body)
	}
	admitted := signInRaw(t, c, ts.URL, iss, "subject-b", "b@example.com")
	if admitted.StatusCode != 302 || admitted.Header.Get("Location") != "/" || sessionCookie(admitted) == nil {
		t.Fatalf("B after A deleted: %d %q cookie=%v", admitted.StatusCode, admitted.Header.Get("Location"), sessionCookie(admitted) != nil)
	}
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
