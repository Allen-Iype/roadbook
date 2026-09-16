package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"roadbook/internal/auth"
	"roadbook/internal/store"
)

// The auth half of the API layer (phase 13 CP2). Two jobs: the strict
// middleware that resolves the session cookie into the request context and
// enforces 401 in multi-user mode, and the four /auth handlers. Identity
// verification lives in internal/auth; persistence in internal/store; this
// file only wires HTTP to both.

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxCookies
)

// authExempt are the operations that must answer without a session: the
// health probe, and the auth surface itself (you cannot sign in from
// behind the sign-in wall).
var authExempt = map[string]bool{
	"GetHealth":      true,
	"GetAuthSession": true,
	"StartSignIn":    true,
	"CompleteSignIn": true,
	"SignOut":        true,
}

// AuthMiddleware is the one place a request becomes a user. Mode off
// (s.Auth == nil): pass through untouched — currentUser falls back to
// store.SelfUser and the instance behaves exactly as before this phase.
// Mode oidc: resolve the session cookie; data operations without a live
// session answer 401 before their handler runs.
func (s *Server) AuthMiddleware(f StrictHandlerFunc, operationID string) StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		ctx = context.WithValue(ctx, ctxCookies, r.Cookies())
		if s.Auth == nil {
			return f(ctx, w, r, request)
		}
		if c, err := r.Cookie(auth.SessionCookie); err == nil && c.Value != "" {
			userID, ok, err := s.Store.SessionUser(ctx, auth.HashToken(c.Value))
			if err != nil {
				return nil, err
			}
			if ok {
				ctx = context.WithValue(ctx, ctxUser, userID)
			}
		}
		if ctx.Value(ctxUser) == nil && !authExempt[operationID] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			if err := json.NewEncoder(w).Encode(Error{Error: "no session — sign in first"}); err != nil {
				log.Printf("writing 401: %v", err)
			}
			return nil, nil
		}
		return f(ctx, w, r, request)
	}
}

// currentUser names the owner of the request's data. Mode off: the
// single-user owner, always (the authless self-host reference). Mode
// oidc: the session user the middleware resolved — data handlers only
// run once one exists.
func (s *Server) currentUser(ctx context.Context) string {
	if u, ok := ctx.Value(ctxUser).(string); ok {
		return u
	}
	return store.SelfUser
}

// cookieValue reads a request cookie the middleware stashed in ctx —
// strict handlers never see the raw *http.Request.
func cookieValue(ctx context.Context, name string) string {
	cookies, _ := ctx.Value(ctxCookies).([]*http.Cookie)
	for _, c := range cookies {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func (s *Server) GetAuthSession(ctx context.Context, _ GetAuthSessionRequestObject) (GetAuthSessionResponseObject, error) {
	if s.Auth == nil {
		return GetAuthSession200JSONResponse{Mode: AuthModeOff, SignedIn: false}, nil
	}
	out := GetAuthSession200JSONResponse{Mode: AuthModeOIDC}
	if u, ok := ctx.Value(ctxUser).(string); ok {
		out.SignedIn = true
		email, err := s.Store.GetUserEmail(ctx, u)
		if err != nil {
			return nil, err
		}
		if email != "" {
			out.Email = &email
		}
	}
	return out, nil
}

func (s *Server) StartSignIn(ctx context.Context, _ StartSignInRequestObject) (StartSignInResponseObject, error) {
	if s.Auth == nil {
		return StartSignIn409JSONResponse{Error: "this instance runs without sign-in"}, nil
	}
	url, cookie, err := s.Auth.BeginSignIn()
	if err != nil {
		return nil, err
	}
	sc := cookie.String()
	return StartSignIn302Response{Headers: StartSignIn302ResponseHeaders{
		Location: &url, SetCookie: &sc,
	}}, nil
}

func (s *Server) CompleteSignIn(ctx context.Context, req CompleteSignInRequestObject) (CompleteSignInResponseObject, error) {
	if s.Auth == nil {
		return CompleteSignIn409JSONResponse{Error: "this instance runs without sign-in"}, nil
	}
	if req.Params.Error != nil && *req.Params.Error != "" {
		return CompleteSignIn400JSONResponse{Error: "the provider reported: " + *req.Params.Error}, nil
	}
	code, state := "", ""
	if req.Params.Code != nil {
		code = *req.Params.Code
	}
	if req.Params.State != nil {
		state = *req.Params.State
	}
	ident, err := s.Auth.CompleteSignIn(ctx, code, state, cookieValue(ctx, auth.StateCookie))
	if err != nil {
		return CompleteSignIn400JSONResponse{Error: err.Error()}, nil
	}
	userID, err := s.Store.UpsertOIDCUser(ctx, ident.Subject, ident.Email)
	if err != nil {
		return nil, err
	}
	raw, hash, err := auth.NewSessionToken()
	if err != nil {
		return nil, err
	}
	if err := s.Store.CreateSession(ctx, hash, userID, time.Now().Add(auth.SessionTTL)); err != nil {
		return nil, err
	}
	loc := "/"
	sc := s.Auth.SessionCookieFor(raw).String()
	return CompleteSignIn302Response{Headers: CompleteSignIn302ResponseHeaders{
		Location: &loc, SetCookie: &sc,
	}}, nil
}

func (s *Server) SignOut(ctx context.Context, _ SignOutRequestObject) (SignOutResponseObject, error) {
	if s.Auth == nil {
		return SignOut204Response{}, nil
	}
	if raw := cookieValue(ctx, auth.SessionCookie); raw != "" {
		if err := s.Store.DeleteSession(ctx, auth.HashToken(raw)); err != nil {
			return nil, err
		}
	}
	sc := s.Auth.ClearSessionCookie().String()
	return SignOut204Response{Headers: SignOut204ResponseHeaders{SetCookie: &sc}}, nil
}
