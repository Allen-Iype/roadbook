package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// The share-link store (phase 13 CP4). A share link is a capability: the
// token is the permission, the table holds only its hash (the sessions
// rule, 00012), and revocation is a DELETE. Links belong to a decision —
// the adventure's durable identity — so they survive re-detection the way
// photos do.

// ShareLinkRow is one live link. The raw token is never here: it exists
// only in the create response, once.
type ShareLinkRow struct {
	ID         int64
	UserID     string
	DecisionID int64
	CreatedAt  time.Time
}

const shareLinkCols = `id, user_id, decision_id, created_at`

// CreateShareLink records a minted link by token hash. The decision must
// be the caller's own: the FK alone would accept another user's decision
// id, so the ownership check is explicit — a link to someone else's
// adventure is unrepresentable, not merely unlikely.
func (s *Store) CreateShareLink(ctx context.Context, userID string, decisionID int64, tokenHash string) (ShareLinkRow, error) {
	var r ShareLinkRow
	err := s.pool.QueryRow(ctx, `INSERT INTO share_links (user_id, decision_id, token_hash)
		SELECT $1, id, $3 FROM decisions WHERE id = $2 AND user_id = $1
		RETURNING `+shareLinkCols, userID, decisionID, tokenHash).
		Scan(&r.ID, &r.UserID, &r.DecisionID, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShareLinkRow{}, ErrNotOwned
	}
	return r, err
}

// ErrNotOwned: the decision named is not the caller's (or does not exist —
// the two are the same from the caller's side, by design).
var ErrNotOwned = errors.New("decision is not the caller's")

// ListShareLinks lists a decision's live links, oldest first.
func (s *Store) ListShareLinks(ctx context.Context, userID string, decisionID int64) ([]ShareLinkRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+shareLinkCols+` FROM share_links
		WHERE user_id = $1 AND decision_id = $2 ORDER BY id`, userID, decisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ShareLinkRow
	for rows.Next() {
		var r ShareLinkRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.DecisionID, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteShareLink revokes one link; false when no link of the caller's has
// that id (another user's id is exactly as absent as an unknown one).
func (s *Store) DeleteShareLink(ctx context.Context, userID string, id int64) (bool, error) {
	ct, err := s.pool.Exec(ctx, `DELETE FROM share_links WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}

// ResolveShareLink looks a token hash up across ALL users — the one store
// read that is deliberately unscoped, because the token is the credential:
// whoever presents it gets exactly the one adventure it names, and the row's
// UserID is what every subsequent read is scoped by. nil when unknown or
// revoked — indistinguishable, as they should be.
func (s *Store) ResolveShareLink(ctx context.Context, tokenHash string) (*ShareLinkRow, error) {
	var r ShareLinkRow
	err := s.pool.QueryRow(ctx, `SELECT `+shareLinkCols+` FROM share_links WHERE token_hash = $1`, tokenHash).
		Scan(&r.ID, &r.UserID, &r.DecisionID, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}
