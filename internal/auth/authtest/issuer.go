// Package authtest is a miniature OIDC issuer for tests (phase 13 BRIEF
// §5: CP2 is proven against a test issuer in the harness; live Google is a
// one-time manual walk). It serves discovery, a JWKS, and a token endpoint
// that exchanges minted codes for RS256-signed ID tokens — just enough
// protocol for go-oidc to verify for real. No browser is involved: tests
// parse the authorize redirect themselves and mint the code directly,
// standing in for the human who consented at the provider.
package authtest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type Issuer struct {
	URL string

	key   *rsa.PrivateKey
	srv   *httptest.Server
	mu    sync.Mutex
	codes map[string]claims
}

type claims struct {
	Subject string
	Email   string
	Nonce   string
}

// New starts the issuer; it stops with the test.
func New(t *testing.T) *Issuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	iss := &Issuer{key: key, codes: map[string]claims{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", iss.discovery)
	mux.HandleFunc("GET /jwks", iss.jwks)
	mux.HandleFunc("POST /token", iss.token)
	iss.srv = httptest.NewServer(mux)
	iss.URL = iss.srv.URL
	t.Cleanup(iss.srv.Close)
	return iss
}

// MintCode registers a one-time code as if subject had just consented at
// the provider. nonce must be the one the authorize redirect carried.
func (i *Issuer) MintCode(subject, email, nonce string) string {
	i.mu.Lock()
	defer i.mu.Unlock()
	code := fmt.Sprintf("code-%d", len(i.codes)+1)
	i.codes[code] = claims{Subject: subject, Email: email, Nonce: nonce}
	return code
}

func (i *Issuer) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"issuer":                                i.URL,
		"authorization_endpoint":                i.URL + "/authorize",
		"token_endpoint":                        i.URL + "/token",
		"jwks_uri":                              i.URL + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (i *Issuer) jwks(w http.ResponseWriter, _ *http.Request) {
	pub := &i.key.PublicKey
	writeJSON(w, map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "test",
			"n": b64url(pub.N.Bytes()),
			"e": b64url(big.NewInt(int64(pub.E)).Bytes()),
		}},
	})
}

func (i *Issuer) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	i.mu.Lock()
	c, ok := i.codes[r.PostFormValue("code")]
	if ok {
		delete(i.codes, r.PostFormValue("code")) // codes are one-time
	}
	i.mu.Unlock()
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		fmt.Fprint(w, `{"error":"invalid_grant"}`)
		return
	}
	clientID, _, _ := r.BasicAuth()
	if clientID == "" {
		clientID = r.PostFormValue("client_id")
	}
	now := time.Now()
	idTok, err := i.sign(map[string]any{
		"iss": i.URL, "aud": clientID, "sub": c.Subject,
		"email": c.Email, "nonce": c.Nonce,
		"iat": now.Add(-10 * time.Second).Unix(), "exp": now.Add(time.Hour).Unix(),
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]any{
		"access_token": "test-access", "token_type": "Bearer",
		"expires_in": 3600, "id_token": idTok,
	})
}

// sign produces a compact RS256 JWS over the claims — hand-rolled with the
// stdlib; three base64url segments are not worth a dependency in testdata.
func (i *Issuer) sign(claims map[string]any) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "test"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := b64url(header) + "." + b64url(payload)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, i.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + b64url(sig), nil
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err) // test server: fail loudly
	}
}
