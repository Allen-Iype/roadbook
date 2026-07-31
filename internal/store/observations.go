package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"roadbook/internal/domain"
)

// ImportResult reports one import: how much the file contained and how much was
// genuinely new. Parsed − Inserted = rows already present from earlier imports.
type ImportResult struct {
	ImportID int64
	Parsed   int
	Inserted int
}

const batchSize = 2000

// ImportObservations writes observations idempotently: each row carries a
// content hash with a unique index, and inserts are ON CONFLICT DO NOTHING, so
// re-importing the same or an overlapping export accumulates instead of
// duplicating (BRIEF §4 choice 3). Observations are never updated or deleted
// (CLAUDE.md invariant 2).
func (s *Store) ImportObservations(ctx context.Context, label string, winStart, winEnd *time.Time, obs domain.Observations, skipped int) (ImportResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	defer tx.Rollback(ctx)

	var res ImportResult
	b := &pgx.Batch{}
	flush := func() error {
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
			res.Inserted += int(ct.RowsAffected())
		}
		if err := br.Close(); err != nil {
			return err
		}
		b = &pgx.Batch{}
		return nil
	}
	queue := func(sql string, args ...any) error {
		b.Queue(sql, args...)
		if b.Len() >= batchSize {
			return flush()
		}
		return nil
	}

	for _, v := range obs.Visits {
		if err := queue(`INSERT INTO visits (start_ts, start_offset_sec, end_ts, end_offset_sec, lat, lon, semantic_type, content_hash)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (content_hash) DO NOTHING`,
			v.Start, offsetOf(v.Start), v.End, offsetOf(v.End),
			latOf(v.Loc), lonOf(v.Loc), v.SemanticType, hashVisit(v)); err != nil {
			return res, err
		}
	}
	for _, a := range obs.Activities {
		if err := queue(`INSERT INTO activities (start_ts, start_offset_sec, end_ts, end_offset_sec, start_lat, start_lon, end_lat, end_lon, distance_m, mode, content_hash)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (content_hash) DO NOTHING`,
			a.Start, offsetOf(a.Start), a.End, offsetOf(a.End),
			latOf(a.From), lonOf(a.From), latOf(a.To), lonOf(a.To),
			a.DistanceM, a.Mode, hashActivity(a)); err != nil {
			return res, err
		}
	}
	for _, p := range obs.Points {
		if err := queue(`INSERT INTO path_points (ts, ts_offset_sec, lat, lon, content_hash)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (content_hash) DO NOTHING`,
			p.Time, offsetOf(p.Time), latOf(p.Loc), lonOf(p.Loc), hashPoint(p)); err != nil {
			return res, err
		}
	}
	if err := flush(); err != nil {
		return res, err
	}

	res.Parsed = len(obs.Visits) + len(obs.Activities) + len(obs.Points)
	err = tx.QueryRow(ctx, `INSERT INTO imports (source_label, window_start, window_end, visits, activities, points, skipped)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		label, winStart, winEnd, len(obs.Visits), len(obs.Activities), len(obs.Points), skipped).Scan(&res.ImportID)
	if err != nil {
		return res, err
	}
	return res, tx.Commit(ctx)
}

// LoadObservations reads everything back in (start, id) order — chronological,
// with insertion order breaking ties, which reproduces source-file order for
// rows from a single import.
func (s *Store) LoadObservations(ctx context.Context) (domain.Observations, error) {
	var obs domain.Observations

	rows, err := s.pool.Query(ctx, `SELECT start_ts, start_offset_sec, end_ts, end_offset_sec, lat, lon, semantic_type
		FROM visits ORDER BY start_ts, id`)
	if err != nil {
		return obs, err
	}
	for rows.Next() {
		var start, end time.Time
		var soff, eoff int
		var lat, lon *float64
		var st string
		if err := rows.Scan(&start, &soff, &end, &eoff, &lat, &lon, &st); err != nil {
			return obs, err
		}
		obs.Visits = append(obs.Visits, domain.Visit{
			Start: withOffset(start, soff), End: withOffset(end, eoff),
			Loc: locOf(lat, lon), SemanticType: st,
		})
	}
	if err := rows.Err(); err != nil {
		return obs, err
	}

	rows, err = s.pool.Query(ctx, `SELECT start_ts, start_offset_sec, end_ts, end_offset_sec, start_lat, start_lon, end_lat, end_lon, distance_m, mode
		FROM activities ORDER BY start_ts, id`)
	if err != nil {
		return obs, err
	}
	for rows.Next() {
		var start, end time.Time
		var soff, eoff int
		var slat, slon, elat, elon *float64
		var dist float64
		var mode string
		if err := rows.Scan(&start, &soff, &end, &eoff, &slat, &slon, &elat, &elon, &dist, &mode); err != nil {
			return obs, err
		}
		obs.Activities = append(obs.Activities, domain.Activity{
			Start: withOffset(start, soff), End: withOffset(end, eoff),
			From: locOf(slat, slon), To: locOf(elat, elon), DistanceM: dist, Mode: mode,
		})
	}
	if err := rows.Err(); err != nil {
		return obs, err
	}

	rows, err = s.pool.Query(ctx, `SELECT ts, ts_offset_sec, lat, lon FROM path_points ORDER BY ts, id`)
	if err != nil {
		return obs, err
	}
	for rows.Next() {
		var ts time.Time
		var off int
		var lat, lon *float64
		if err := rows.Scan(&ts, &off, &lat, &lon); err != nil {
			return obs, err
		}
		obs.Points = append(obs.Points, domain.PathPoint{Time: withOffset(ts, off), Loc: locOf(lat, lon)})
	}
	return obs, rows.Err()
}

func latOf(l *domain.LatLng) *float64 {
	if l == nil {
		return nil
	}
	return &l.Lat
}

func lonOf(l *domain.LatLng) *float64 {
	if l == nil {
		return nil
	}
	return &l.Lon
}

func locOf(lat, lon *float64) *domain.LatLng {
	if lat == nil || lon == nil {
		return nil
	}
	return &domain.LatLng{Lat: *lat, Lon: *lon}
}

// Content hashes: SHA-256 over a canonical text form — times as UTC
// RFC 3339 nano (so the same instant with a different offset notation dedups),
// coordinates in Go's shortest exact float form. Changing this canonical form
// changes every hash and orphans nothing but re-inserts everything; don't.
func hashVisit(v domain.Visit) string {
	return hashParts("v", canonTime(v.Start), canonTime(v.End), canonLoc(v.Loc), v.SemanticType)
}

func hashActivity(a domain.Activity) string {
	return hashParts("a", canonTime(a.Start), canonTime(a.End), canonLoc(a.From), canonLoc(a.To),
		strconv.FormatFloat(a.DistanceM, 'g', -1, 64), a.Mode)
}

func hashPoint(p domain.PathPoint) string {
	return hashParts("p", canonTime(p.Time), canonLoc(p.Loc))
}

func hashParts(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}

func canonTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func canonLoc(l *domain.LatLng) string {
	if l == nil {
		return "-"
	}
	return fmt.Sprintf("%s,%s",
		strconv.FormatFloat(l.Lat, 'g', -1, 64), strconv.FormatFloat(l.Lon, 'g', -1, 64))
}
