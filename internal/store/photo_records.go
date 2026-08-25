package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"roadbook/internal/domain"
)

// PhotoRecord is the durable identity of one ingested photograph (phase 11
// BRIEF §4D): what keeps a fix on a map traceable back to its photo evidence
// after the original bytes are discarded. Position and instant live on the
// paired fix; this carries what the fix cannot.
type PhotoRecord struct {
	ID             int64
	ImportID       int64
	ContentHash    string // sha256 of the original bytes; thumbnail filename
	OriginalName   string
	TakenAt        time.Time
	TakenOffsetSec int
	TimeSource     string
	Lat, Lon       float64
	PosSource      string
	ThumbW, ThumbH int // 0,0 = no thumbnail (HEIC pixels are not decodable here)
}

// PhotoIngest pairs one photo's fix with its record for import.
type PhotoIngest struct {
	Fix    domain.RawPosition
	Record PhotoRecord // ImportID, TakenAt/offset, Lat/Lon, fix hash filled here
}

// ImportPhotos writes a photo batch in one transaction: each fix into
// raw_positions (content-hash idempotent, the observation stratum — nothing
// downstream learns about photos) and each record into photo_records, joined
// by the fix's content hash. The imports row is finalised the same way
// ImportObservations finalises it; Inserted counts new fixes, the number that
// means "new observations" everywhere else.
func (s *Store) ImportPhotos(ctx context.Context, importID int64, items []PhotoIngest) (ImportResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	defer tx.Rollback(ctx)

	res := ImportResult{ImportID: importID, Parsed: len(items)}

	flush := func(b *pgx.Batch, count bool) error {
		if b.Len() == 0 {
			return nil
		}
		br := tx.SendBatch(ctx, b)
		for range b.Len() {
			ct, err := br.Exec()
			if err != nil {
				br.Close()
				return err
			}
			if count {
				res.Inserted += int(ct.RowsAffected())
			}
		}
		return br.Close()
	}

	fixes := &pgx.Batch{}
	records := &pgx.Batch{}
	for _, it := range items {
		rp := it.Fix
		fixHash := hashRawPosition(rp)
		fixes.Queue(`INSERT INTO raw_positions (ts, ts_offset_sec, lat, lon, accuracy_m, source, content_hash)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (content_hash) DO NOTHING`,
			rp.Time, offsetOf(rp.Time), latOf(rp.Loc), lonOf(rp.Loc), rp.AccuracyM, rp.Source, fixHash)
		r := it.Record
		records.Queue(`INSERT INTO photo_records (import_id, content_hash, original_name, taken_at, taken_offset_sec,
			time_source, lat, lon, pos_source, thumb_w, thumb_h, fix_content_hash)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT (content_hash) DO NOTHING`,
			importID, r.ContentHash, r.OriginalName, rp.Time, offsetOf(rp.Time),
			r.TimeSource, latOf(rp.Loc), lonOf(rp.Loc), r.PosSource, r.ThumbW, r.ThumbH, fixHash)
		if fixes.Len() >= batchSize {
			if err := flush(fixes, true); err != nil {
				return res, err
			}
			if err := flush(records, false); err != nil {
				return res, err
			}
			fixes, records = &pgx.Batch{}, &pgx.Batch{}
		}
	}
	if err := flush(fixes, true); err != nil {
		return res, err
	}
	if err := flush(records, false); err != nil {
		return res, err
	}

	ct, err := tx.Exec(ctx, `UPDATE imports SET raw_positions = $2, status = 'completed',
		detected_format = 'photos', inserted = $3
		WHERE id = $1 AND status = 'running'`,
		importID, len(items), res.Inserted)
	if err != nil {
		return res, err
	}
	if ct.RowsAffected() != 1 {
		return res, fmt.Errorf("import %d is not running — BeginImport must precede ImportPhotos", importID)
	}
	return res, tx.Commit(ctx)
}

// ListPhotoRecords returns one import's records in upload order.
func (s *Store) ListPhotoRecords(ctx context.Context, importID int64) ([]PhotoRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, import_id, content_hash, original_name, taken_at, taken_offset_sec,
		time_source, lat, lon, pos_source, thumb_w, thumb_h
		FROM photo_records WHERE import_id = $1 ORDER BY id`, importID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PhotoRecord
	for rows.Next() {
		var r PhotoRecord
		if err := rows.Scan(&r.ID, &r.ImportID, &r.ContentHash, &r.OriginalName, &r.TakenAt, &r.TakenOffsetSec,
			&r.TimeSource, &r.Lat, &r.Lon, &r.PosSource, &r.ThumbW, &r.ThumbH); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
