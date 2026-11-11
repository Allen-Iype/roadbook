package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"roadbook/internal/domain"
)

// DecisionRow is user data: never regenerated, never deleted by any automated
// process. The anchor fields are the candidate as the user saw it when
// deciding; association with current candidates is recomputed by
// detect.Match, never stored.
type DecisionRow struct {
	ID          int64
	Action      string // 'confirmed' | 'dismissed'
	Name        *string
	Notes       *string // the owner's own words (migration 15); nil = none
	AnchorStart time.Time
	AnchorEnd   time.Time
	AnchorDest  domain.LatLng
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const decisionCols = `id, action, name, notes, anchor_span_start, anchor_span_end,
	anchor_dest_lat, anchor_dest_lon, created_at, updated_at`

// scanDecision reads one row in decisionCols order.
func scanDecision(row interface{ Scan(...any) error }) (DecisionRow, error) {
	var d DecisionRow
	err := row.Scan(&d.ID, &d.Action, &d.Name, &d.Notes, &d.AnchorStart, &d.AnchorEnd,
		&d.AnchorDest.Lat, &d.AnchorDest.Lon, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (s *Store) ListDecisions(ctx context.Context, userID string) ([]DecisionRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+decisionCols+` FROM decisions WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DecisionRow
	for rows.Next() {
		d, err := scanDecision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// InsertDecision records a new decision with its anchor snapshot.
func (s *Store) InsertDecision(ctx context.Context, userID, action string, name *string, anchor CandidateRow) (DecisionRow, error) {
	return scanDecision(s.pool.QueryRow(ctx, `INSERT INTO decisions (user_id, action, name, anchor_span_start, anchor_span_end, anchor_dest_lat, anchor_dest_lon)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING `+decisionCols,
		userID, action, name, anchor.SpanStart, anchor.SpanEnd, anchor.Dest.Lat, anchor.Dest.Lon))
}

// BulkDecision is one item of an atomic bulk triage write (phase 11 §6.1).
// The caller (the API layer) resolves candidate identity and the matched
// decision id; the store's job is exactly the single-decision SQL, once per
// item, inside one transaction.
type BulkDecision struct {
	Anchor   CandidateRow
	Action   string
	Name     *string
	UpdateID *int64 // matched decision to re-decide in place; nil = fresh insert
}

// DecideBulk applies every item or none: a mid-list failure rolls the whole
// batch back, so "some of your sweep landed" is a state that cannot exist.
func (s *Store) DecideBulk(ctx context.Context, userID string, items []BulkDecision) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, it := range items {
		if it.UpdateID != nil {
			_, err = tx.Exec(ctx, `UPDATE decisions SET action=$2, name=$3, anchor_span_start=$4, anchor_span_end=$5,
				anchor_dest_lat=$6, anchor_dest_lon=$7, updated_at=now() WHERE id=$1 AND user_id=$8`,
				*it.UpdateID, it.Action, it.Name, it.Anchor.SpanStart, it.Anchor.SpanEnd,
				it.Anchor.Dest.Lat, it.Anchor.Dest.Lon, userID)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO decisions (user_id, action, name, anchor_span_start, anchor_span_end, anchor_dest_lat, anchor_dest_lon)
				VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				userID, it.Action, it.Name, it.Anchor.SpanStart, it.Anchor.SpanEnd,
				it.Anchor.Dest.Lat, it.Anchor.Dest.Lon)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// UpdateDecision re-decides in place, refreshing the anchor to the candidate
// the user is looking at now (BRIEF §3.1). The user overwriting their own
// decision is the one legitimate mutation of user data.
// Notes are not in the SET list: a re-decide never blanks what the owner
// wrote (phase 17, #3).
func (s *Store) UpdateDecision(ctx context.Context, userID string, id int64, action string, name *string, anchor CandidateRow) (DecisionRow, error) {
	return scanDecision(s.pool.QueryRow(ctx, `UPDATE decisions SET action=$2, name=$3, anchor_span_start=$4, anchor_span_end=$5,
		anchor_dest_lat=$6, anchor_dest_lon=$7, updated_at=now() WHERE id=$1 AND user_id=$8 RETURNING `+decisionCols,
		id, action, name, anchor.SpanStart, anchor.SpanEnd, anchor.Dest.Lat, anchor.Dest.Lon, userID))
}

// EditDecisionText changes a kept trip's name and notes (phase 17, #3) and
// nothing else: not the action, not the anchor. Only a confirmed row of
// this user's is touched; ok is false when there is none — unknown id,
// another user's row, or a dismissed decision, which has no name to edit.
// A nil notes clears them.
func (s *Store) EditDecisionText(ctx context.Context, userID string, id int64, name string, notes *string) (DecisionRow, bool, error) {
	d, err := scanDecision(s.pool.QueryRow(ctx, `UPDATE decisions SET name=$3, notes=$4, updated_at=now()
		WHERE id=$1 AND user_id=$2 AND action='confirmed' RETURNING `+decisionCols,
		id, userID, name, notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return DecisionRow{}, false, nil
	}
	return d, err == nil, err
}
