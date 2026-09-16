-- +goose Up
-- Phase 13 CP2: sessions. Server-side sessions in Postgres over stateless
-- JWTs (BRIEF §1: revocation is a DELETE, sign-out is real, no signing-key
-- lifecycle). The cookie carries an opaque random token; this table holds
-- only its SHA-256 — a leaked table row cannot be replayed as a session.
CREATE TABLE sessions (
    token_hash text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions (expires_at);

-- +goose Down
DROP TABLE sessions;
