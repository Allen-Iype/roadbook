package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
)

// PhotoRow is user data beside decisions (phase 4 BRIEF §1.4): never
// regenerated, never deleted by any automated process. The row pairs with a
// thumbnail file named <content_hash>.jpg in the photos directory; the
// original bytes were discarded after extraction, so row + file are the
// photo, as far as this system is concerned.
type PhotoRow struct {
	ID           int64
	DecisionID   int64
	ContentHash  string
	OriginalName string
	// TakenAt nil = no usable capture time (time_source "none"): stored,
	// shown, unplaced. When present, TakenOffsetSec carries the civil
	// offset for display, restored on load like every other instant.
	TakenAt        *time.Time
	TakenOffsetSec *int
	TimeSource     string
	Lat, Lon       *float64 // nil pair when pos_source is "none"
	PosSource      string
	ThumbW, ThumbH int
	UploadedAt     time.Time
}

const photoCols = `id, decision_id, content_hash, original_name, taken_at,
	taken_offset_sec, time_source, lat, lon, pos_source, thumb_w, thumb_h, uploaded_at`

func scanPhoto(row interface{ Scan(...any) error }) (PhotoRow, error) {
	var p PhotoRow
	err := row.Scan(&p.ID, &p.DecisionID, &p.ContentHash, &p.OriginalName, &p.TakenAt,
		&p.TakenOffsetSec, &p.TimeSource, &p.Lat, &p.Lon, &p.PosSource,
		&p.ThumbW, &p.ThumbH, &p.UploadedAt)
	if err == nil && p.TakenAt != nil && p.TakenOffsetSec != nil {
		t := withOffset(*p.TakenAt, *p.TakenOffsetSec)
		p.TakenAt = &t
	}
	return p, err
}

// InsertPhoto stores one photo's metadata. Idempotent by content hash (the
// imports precedent): identical bytes return the existing row with
// inserted=false, and nothing changes.
func (s *Store) InsertPhoto(ctx context.Context, p PhotoRow) (PhotoRow, bool, error) {
	row, err := scanPhoto(s.pool.QueryRow(ctx, `
		INSERT INTO photos (decision_id, content_hash, original_name, taken_at,
			taken_offset_sec, time_source, lat, lon, pos_source, thumb_w, thumb_h)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (content_hash) DO NOTHING
		RETURNING `+photoCols,
		p.DecisionID, p.ContentHash, p.OriginalName, p.TakenAt, p.TakenOffsetSec,
		p.TimeSource, p.Lat, p.Lon, p.PosSource, p.ThumbW, p.ThumbH))
	if err == nil {
		return row, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return PhotoRow{}, false, err
	}
	// DO NOTHING returned no row: the hash already exists — fetch the original.
	existing, ferr := scanPhoto(s.pool.QueryRow(ctx,
		`SELECT `+photoCols+` FROM photos WHERE content_hash = $1`, p.ContentHash))
	if ferr != nil {
		return PhotoRow{}, false, fmt.Errorf("photo insert conflicted but original not found: %w", ferr)
	}
	return existing, false, nil
}

// ListPhotos returns a decision's photos in upload order.
func (s *Store) ListPhotos(ctx context.Context, decisionID int64) ([]PhotoRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+photoCols+` FROM photos WHERE decision_id = $1 ORDER BY id`, decisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PhotoRow{}
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPhoto returns one photo, or nil when it does not exist.
func (s *Store) GetPhoto(ctx context.Context, id int64) (*PhotoRow, error) {
	p, err := scanPhoto(s.pool.QueryRow(ctx,
		`SELECT `+photoCols+` FROM photos WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// DeletePhoto removes the row. The caller then removes the file — row first,
// so a mid-failure leaves unreachable garbage, never a broken image
// (BRIEF §3B). Returns false when the photo did not exist.
func (s *Store) DeletePhoto(ctx context.Context, id int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM photos WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// PhotoFiles is the thumbnail directory: the filesystem half of the photos
// stratum, reachable only from Go (BRIEF §1.3). Files are named by content
// hash, flat — the working set is tens to low hundreds of files, and
// fan-out subdirectories would be scale theatre (BRIEF §3B).
type PhotoFiles struct {
	Dir string
}

// Init creates the directory. Called once at startup by whatever binary
// serves or writes photos.
func (f PhotoFiles) Init() error {
	return os.MkdirAll(f.Dir, 0o755)
}

func (f PhotoFiles) path(contentHash string) string {
	return filepath.Join(f.Dir, contentHash+".jpg")
}

func (f PhotoFiles) WriteThumb(contentHash string, jpeg []byte) error {
	return os.WriteFile(f.path(contentHash), jpeg, 0o644)
}

func (f PhotoFiles) ReadThumb(contentHash string) ([]byte, error) {
	return os.ReadFile(f.path(contentHash))
}

// DeleteThumb removes the file; a file already gone is success, not an
// error — the row is authoritative and it is already deleted.
func (f PhotoFiles) DeleteThumb(contentHash string) error {
	err := os.Remove(f.path(contentHash))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
