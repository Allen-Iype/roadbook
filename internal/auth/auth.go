// Package auth is the OIDC + session layer (phase 13 BRIEF §1). Identity is
// delegated: the user proves who they are to a configured OIDC issuer, and
// this package verifies the issuer's signed answer — no password is ever
// stored or checked here (the charter's never-hand-rolled rule). What IS
// ours is the session: an opaque random token in an HttpOnly cookie, its
// hash in Postgres, revoked by deletion.
//
// The package never touches the store: it turns HTTP artifacts (redirects,
// codes, cookies) into verified identity claims and back. Persistence stays
// in internal/store, enforcement in internal/api — one job each.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	// SessionCookie carries the opaque session token.
	SessionCookie = "roadbook_session"
	// StateCookie carries "state.nonce" across the provider round trip —
	// HttpOnly and short-lived; the callback consumes it exactly once.
	StateCookie = "roadbook_auth_state"

	// SessionTTL is deliberately long for a personal tool: the cost of a
	// stale session is bounded by revocation being a DELETE.
	SessionTTL = 30 * 24 * time.Hour
	stateTTL   = 10 * time.Minute
)

// Config is everything multi-user mode needs, all operator-supplied env
// (invariant 9: nothing about any specific user or provider is hardcoded).
type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	// RedirectURL is the browser-facing callback — the Next.js origin, not
	// this process: the browser only ever talks to the web layer.
	RedirectURL string
	// SecureCookies marks cookies Secure; derived from the public origin's
	// scheme by the caller (localhost development stays plain HTTP).
	SecureCookies bool
}

// Service is a configured OIDC client. A nil *Service means mode off.
type Service struct {
	cfg      Config
	provider *oidc.Provider
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// New performs issuer discovery and fails fast: a serve that cannot reach
// its issuer must not come up half-authenticated.
func New(ctx context.Context, cfg Config) (*Service, error) {
	switch {
	case cfg.IssuerURL == "":
		return nil, fmt.Errorf("auth: issuer URL is required in oidc mode")
	case cfg.ClientID == "":
		return nil, fmt.Errorf("auth: client id is required in oidc mode")
	case cfg.ClientSecret == "":
		return nil, fmt.Errorf("auth: client secret is required in oidc mode")
	case cfg.RedirectURL == "":
		return nil, fmt.Errorf("auth: redirect URL is required in oidc mode")
	}
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("auth: issuer discovery: %w", err)
	}
	return &Service{
		cfg:      cfg,
		provider: provider,
		oauth: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
	}, nil
}

// Identity is what a verified callback yields — the two claims we keep.
type Identity struct {
	Subject string
	Email   string
}

// BeginSignIn mints a state+nonce pair and returns the provider URL plus
// the state cookie that must ride the redirect response.
func (s *Service) BeginSignIn() (authURL string, cookie *http.Cookie, err error) {
	state, err := randomToken()
	if err != nil {
		return "", nil, err
	}
	nonce, err := randomToken()
	if err != nil {
		return "", nil, err
	}
	return s.oauth.AuthCodeURL(state, oidc.Nonce(nonce)), &http.Cookie{
		Name:     StateCookie,
		Value:    state + "." + nonce,
		Path:     "/",
		MaxAge:   int(stateTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}, nil
}

// CompleteSignIn verifies the provider's answer: state against the cookie,
// then the code exchanged (server-to-server, with the client secret) and
// the ID token's signature, audience, and nonce checked.
func (s *Service) CompleteSignIn(ctx context.Context, code, state, stateCookie string) (Identity, error) {
	wantState, wantNonce, ok := strings.Cut(stateCookie, ".")
	if !ok || wantState == "" {
		return Identity{}, fmt.Errorf("sign-in state cookie missing or malformed — start again from the sign-in page")
	}
	if subtle.ConstantTimeCompare([]byte(state), []byte(wantState)) != 1 {
		return Identity{}, fmt.Errorf("sign-in state mismatch — start again from the sign-in page")
	}
	if code == "" {
		return Identity{}, fmt.Errorf("the provider sent no code")
	}
	tok, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return Identity{}, fmt.Errorf("code exchange failed: %w", err)
	}
	rawID, ok := tok.Extra("id_token").(string)
	if !ok {
		return Identity{}, fmt.Errorf("the provider's token response carried no id_token")
	}
	idTok, err := s.verifier.Verify(ctx, rawID)
	if err != nil {
		return Identity{}, fmt.Errorf("id_token verification failed: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idTok.Nonce), []byte(wantNonce)) != 1 {
		return Identity{}, fmt.Errorf("sign-in nonce mismatch — start again from the sign-in page")
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := idTok.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("reading id_token claims: %w", err)
	}
	return Identity{Subject: idTok.Subject, Email: claims.Email}, nil
}

// NewSessionToken mints the opaque session value: the raw token goes in
// the cookie, only the hash is ever stored (a leaked table row cannot be
// replayed).
func NewSessionToken() (raw, hash string, err error) {
	raw, err = randomToken()
	if err != nil {
		return "", "", err
	}
	return raw, HashToken(raw), nil
}

// HashToken is the storage form of a session token.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// SessionCookieFor wraps a raw token for the browser.
func (s *Service) SessionCookieFor(raw string) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    raw,
		Path:     "/",
		MaxAge:   int(SessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

// ClearSessionCookie is the sign-out cookie: same attributes, expired.
func (s *Service) ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

func randomToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
