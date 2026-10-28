package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// The session store (phase 13 CP2). Sessions are opaque tokens: the cookie
// carries the random value, the table carries only its hash, and a lookup
// is one indexed read. Revocation is a DELETE — the observable sign-out
// the brief promised.

// ErrUserCapReached: a first sign-in arrived while the instance already
// holds its cap of accounts (phase 15 BRIEF §9). The caller turns it into
// the sign-in page's "full" state; nothing was written.
var ErrUserCapReached = errors.New("user cap reached")

// UpsertOIDCUser resolves a verified OIDC subject to a user id, creating
// the user on first sign-in and refreshing the display email on every one.
// The id is generated here, never the provider subject: subjects are
// provider-scoped and belong in their own column.
//
// cap is the instance's account limit (phase 15 BRIEF §9): 0 means
// unlimited; otherwise a NEW subject is admitted only while the count of
// OIDC accounts is below it. A returning subject is never refused, whatever
// the count — lowering the cap below the population locks the door, it
// does not evict. The 'self' row (migration 00011, NULL subject) is the
// auth-off owner and never counts. First sign-ins are serialised by a
// named transaction-scoped advisory lock so two simultaneous newcomers
// cannot both read "one seat left" and both take it.
func (s *Store) UpsertOIDCUser(ctx context.Context, subject, email string, cap int) (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	newID := "u-" + hex.EncodeToString(buf[:])
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('roadbook.users.admission'))`); err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO users (id, oidc_subject, email)
		SELECT $1, $2, $3
		WHERE $4 = 0
		   OR EXISTS (SELECT 1 FROM users WHERE oidc_subject = $2)
		   OR (SELECT count(*) FROM users WHERE oidc_subject IS NOT NULL) < $4
		ON CONFLICT (oidc_subject) DO UPDATE SET email = EXCLUDED.email
		RETURNING id`, newID, subject, email, cap).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserCapReached
	}
	if err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

// GetUserEmail returns the display email for a user, "" when none is known.
func (s *Store) GetUserEmail(ctx context.Context, userID string) (string, error) {
	var email *string
	err := s.pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) || email == nil {
		return "", nil
	}
	return *email, err
}

// CreateSession records a minted session by token hash.
func (s *Store) CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1,$2,$3)`, tokenHash, userID, expiresAt)
	return err
}

// SessionUser resolves a token hash to its live user; ok is false for
// unknown and expired tokens alike — the caller cannot tell which, and
// should not be able to.
func (s *Store) SessionUser(ctx context.Context, tokenHash string) (string, bool, error) {
	var userID string
	err := s.pool.QueryRow(ctx, `SELECT user_id FROM sessions
		WHERE token_hash = $1 AND expires_at > now()`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return userID, true, nil
}

// DeleteSession revokes one session. Deleting an absent token succeeds:
// the goal is absence.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

// DeleteExpiredSessions sweeps rows past their expiry — called at serve
// startup, the same housekeeping moment as the running-imports sweep.
func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	ct, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
