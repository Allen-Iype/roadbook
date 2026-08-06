package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// The store half of backup/restore (phase 5 BRIEF §3D). Backup reads the two
// irreplaceable strata — decisions and photos — and restore merges them back
// by their durable identities: the decision anchor (unique since migration
// 00008) and the photo content hash. Restore never updates an existing row:
// the overlap of two archives is skipped, not reconciled.

// SchemaVersion is the highest applied migration — recorded in a backup
// manifest so a restore can say what schema wrote the archive.
func (s *Store) SchemaVersion(ctx context.Context) (int64, error) {
	var v int64
	err := s.pool.QueryRow(ctx, `SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&v)
	return v, err
}

// ListAllPhotos returns every photo row, id order, for backup.
func (s *Store) ListAllPhotos(ctx context.Context) ([]PhotoRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+photoCols+` FROM photos ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PhotoRow
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RestoreDecision inserts an archived decision with its original timestamps,
// skipping on anchor conflict. Either way it returns the id the anchor now
// maps to, which is what archived photos attach through.
func (s *Store) RestoreDecision(ctx context.Context, d DecisionRow) (int64, bool, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO decisions (action, name, anchor_span_start, anchor_span_end,
			anchor_dest_lat, anchor_dest_lon, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (user_id, anchor_span_start, anchor_span_end, anchor_dest_lat, anchor_dest_lon) DO NOTHING
		RETURNING id`,
		d.Action, d.Name, d.AnchorStart, d.AnchorEnd, d.AnchorDest.Lat, d.AnchorDest.Lon,
		d.CreatedAt, d.UpdatedAt).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}
	// Conflict: the identity already exists here — find the row it names.
	err = s.pool.QueryRow(ctx, `SELECT id FROM decisions
		WHERE user_id = 'self' AND anchor_span_start = $1 AND anchor_span_end = $2
		  AND anchor_dest_lat = $3 AND anchor_dest_lon = $4`,
		d.AnchorStart, d.AnchorEnd, d.AnchorDest.Lat, d.AnchorDest.Lon).Scan(&id)
	return id, false, err
}

// RestorePhoto inserts an archived photo row with its original upload time,
// skipping on content-hash conflict (the same identity that makes re-upload
// idempotent).
func (s *Store) RestorePhoto(ctx context.Context, p PhotoRow) (bool, error) {
	ct, err := s.pool.Exec(ctx, `INSERT INTO photos (decision_id, content_hash, original_name, taken_at,
			taken_offset_sec, time_source, lat, lon, pos_source, thumb_w, thumb_h, uploaded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (content_hash) DO NOTHING`,
		p.DecisionID, p.ContentHash, p.OriginalName, p.TakenAt, p.TakenOffsetSec,
		p.TimeSource, p.Lat, p.Lon, p.PosSource, p.ThumbW, p.ThumbH, p.UploadedAt)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}
