package store

import "context"

// SelfUser is the owner of everything on a single-user instance — the
// authless self-host reference (PRODUCT.md). Every caller that has no
// session concept passes this; the CP2 auth layer is what will ever pass
// anything else. The row is created by migration 00011.
const SelfUser = "self"

// EnsureUser makes the user row exist. Idempotent; used by tests now and
// by the OIDC callback (CP2) when a subject signs in for the first time.
func (s *Store) EnsureUser(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, id)
	return err
}
