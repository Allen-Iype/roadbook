package store

import (
	"context"
)

// Self-serve deletion (phase 13 CP5, BRIEF §1 "Data lifecycle"). One
// transaction removes every row a user owns, in foreign-key order, and
// reports what went. Files are NOT touched here: the store hands back the
// content hashes whose last owning row has just gone, and the API layer
// removes those files after the commit — failing toward an orphaned file
// on disk, never toward a row pointing at nothing (the photos rule).
//
// The user row itself goes only when asked (dropUser): a multi-user
// instance forgets the account so the next sign-in starts fresh; the
// authless instance keeps 'self', which every table's DEFAULT names.

// DeletionReport is what a deletion removed, by table, plus the files now
// unreferenced by anyone.
type DeletionReport struct {
	Rows map[string]int64
	// Thumbnail content hashes no remaining photo or photo record (of any
	// user) names — safe to delete from the shared thumbnail directory.
	OrphanThumbs []string
	// Upload content hashes no remaining import (of any user) names.
	OrphanUploads []string
}

// DeleteUserData removes everything owned by userID. Deliberately spelled
// out table by table rather than through cascades: the list IS the
// statement of what "everything of yours" means, and a new user-data
// table missing from it is a visible omission at review (the cross-tenant
// family's own rule, applied to deletion).
func (s *Store) DeleteUserData(ctx context.Context, userID string, dropUser bool) (DeletionReport, error) {
	rep := DeletionReport{Rows: map[string]int64{}}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return rep, err
	}
	defer tx.Rollback(ctx)

	// Hashes first, while the rows still exist.
	var thumbHashes, uploadHashes []string
	if err := tx.QueryRow(ctx, `SELECT coalesce(array_agg(DISTINCT h), '{}') FROM (
			SELECT content_hash h FROM photos WHERE user_id = $1
			UNION SELECT content_hash FROM photo_records WHERE user_id = $1) x`, userID).Scan(&thumbHashes); err != nil {
		return rep, err
	}
	if err := tx.QueryRow(ctx, `SELECT coalesce(array_agg(DISTINCT content_hash), '{}') FROM imports
			WHERE user_id = $1 AND content_hash IS NOT NULL`, userID).Scan(&uploadHashes); err != nil {
		return rep, err
	}

	// Foreign-key order: photos before decisions (RESTRICT), photo records
	// before imports (RESTRICT), candidates before runs, sessions and share
	// links before the user row.
	for _, table := range UserDataTables {
		ct, err := tx.Exec(ctx, `DELETE FROM `+table+` WHERE user_id = $1`, userID)
		if err != nil {
			return rep, err
		}
		rep.Rows[table] = ct.RowsAffected()
	}
	if dropUser {
		ct, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		if err != nil {
			return rep, err
		}
		rep.Rows["users"] = ct.RowsAffected()
	}

	// What is now unreferenced: identical bytes are one file on disk
	// regardless of owner, so a hash another user still holds stays.
	if err := tx.QueryRow(ctx, `SELECT coalesce(array_agg(h), '{}') FROM unnest($1::text[]) h
			WHERE NOT EXISTS (SELECT 1 FROM photos WHERE content_hash = h)
			  AND NOT EXISTS (SELECT 1 FROM photo_records WHERE content_hash = h)`, thumbHashes).Scan(&rep.OrphanThumbs); err != nil {
		return rep, err
	}
	if err := tx.QueryRow(ctx, `SELECT coalesce(array_agg(h), '{}') FROM unnest($1::text[]) h
			WHERE NOT EXISTS (SELECT 1 FROM imports WHERE content_hash = h)`, uploadHashes).Scan(&rep.OrphanUploads); err != nil {
		return rep, err
	}
	return rep, tx.Commit(ctx)
}

// UserDataTables is the list DeleteUserData clears — exported so the
// deletion test can count every one of them to zero without a second copy
// of the list drifting.
var UserDataTables = []string{
	"photos", "share_links", "decisions", "photo_records",
	"raw_positions", "path_points", "activities", "visits",
	"candidates", "detection_runs", "imports", "sessions",
}

// CountUserRows counts a user's rows in one user-data table — a test and
// audit helper (the table name comes from UserDataTables, never from input).
func (s *Store) CountUserRows(ctx context.Context, table, userID string) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}
