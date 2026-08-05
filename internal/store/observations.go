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

// ImportRow is one imports-table row, the bookkeeping the UI surfaces
// (phase 5 BRIEF §3B). Error and DetectedFormat are nil until known.
type ImportRow struct {
	ID             int64
	SourceLabel    string
	WindowStart    *time.Time
	WindowEnd      *time.Time
	ImportedAt     time.Time
	Visits         int
	Activities     int
	Points         int
	RawPositions   int
	Skipped        int
	Status         string
	Error          *string
	DetectedFormat *string
}

const batchSize = 2000

// BeginImport records the attempt before anything is parsed, status 'running',
// so a failure is visible in the product rather than only in a closed terminal
// (phase 5 BRIEF §3B). Counters are zero until ImportObservations finalises the
// row; FailImport finalises the other path.
func (s *Store) BeginImport(ctx context.Context, label string, winStart, winEnd *time.Time) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO imports (source_label, window_start, window_end, visits, activities, points, raw_positions, skipped, status)
		VALUES ($1,$2,$3,0,0,0,0,0,'running') RETURNING id`,
		label, winStart, winEnd).Scan(&id)
	return id, err
}

// FailImport finalises a failed attempt. detectedFormat is the sniffer's
// stable slug when the input was recognised ("" stores NULL — the failure
// happened before recognition); errMsg is the user-facing message, prose that
// may be reworded and is therefore never the queryable evidence.
func (s *Store) FailImport(ctx context.Context, importID int64, detectedFormat, errMsg string) error {
	_, err := s.pool.Exec(ctx, `UPDATE imports SET status = 'failed', error = $2, detected_format = nullif($3, '')
		WHERE id = $1`, importID, errMsg, detectedFormat)
	return err
}

// ImportObservations writes observations idempotently: each row carries a
// content hash with a unique index, and inserts are ON CONFLICT DO NOTHING, so
// re-importing the same or an overlapping export accumulates instead of
// duplicating (BRIEF §4 choice 3). Observations are never updated or deleted
// (CLAUDE.md invariant 2). The imports row created by BeginImport is finalised
// to 'completed' in the same transaction, so counters and observations land
// together or not at all.
func (s *Store) ImportObservations(ctx context.Context, importID int64, detectedFormat string, obs domain.Observations, skipped int) (ImportResult, error) {
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
	for _, rp := range obs.RawPositions {
		if err := queue(`INSERT INTO raw_positions (ts, ts_offset_sec, lat, lon, accuracy_m, source, content_hash)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (content_hash) DO NOTHING`,
			rp.Time, offsetOf(rp.Time), latOf(rp.Loc), lonOf(rp.Loc), rp.AccuracyM, rp.Source, hashRawPosition(rp)); err != nil {
			return res, err
		}
	}
	if err := flush(); err != nil {
		return res, err
	}

	res.Parsed = len(obs.Visits) + len(obs.Activities) + len(obs.Points) + len(obs.RawPositions)
	res.ImportID = importID
	ct, err := tx.Exec(ctx, `UPDATE imports SET visits = $2, activities = $3, points = $4, raw_positions = $5, skipped = $6,
		status = 'completed', detected_format = nullif($7, '')
		WHERE id = $1 AND status = 'running'`,
		importID, len(obs.Visits), len(obs.Activities), len(obs.Points), len(obs.RawPositions), skipped, detectedFormat)
	if err != nil {
		return res, err
	}
	if ct.RowsAffected() != 1 {
		return res, fmt.Errorf("import %d is not running — BeginImport must precede ImportObservations", importID)
	}
	return res, tx.Commit(ctx)
}

// ListImports returns every import attempt, newest first — the bookkeeping
// view (phase 5 BRIEF §3B).
func (s *Store) ListImports(ctx context.Context) ([]ImportRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, source_label, window_start, window_end, imported_at,
		visits, activities, points, raw_positions, skipped, status, error, detected_format
		FROM imports ORDER BY imported_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ImportRow
	for rows.Next() {
		var r ImportRow
		if err := rows.Scan(&r.ID, &r.SourceLabel, &r.WindowStart, &r.WindowEnd, &r.ImportedAt,
			&r.Visits, &r.Activities, &r.Points, &r.RawPositions, &r.Skipped, &r.Status, &r.Error, &r.DetectedFormat); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
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
	if err := rows.Err(); err != nil {
		return obs, err
	}

	rows, err = s.pool.Query(ctx, `SELECT ts, ts_offset_sec, lat, lon, accuracy_m, source FROM raw_positions ORDER BY ts, id`)
	if err != nil {
		return obs, err
	}
	for rows.Next() {
		var ts time.Time
		var off int
		var lat, lon *float64
		var acc float64
		var src string
		if err := rows.Scan(&ts, &off, &lat, &lon, &acc, &src); err != nil {
			return obs, err
		}
		obs.RawPositions = append(obs.RawPositions, domain.RawPosition{
			Time: withOffset(ts, off), Loc: locOf(lat, lon), AccuracyM: acc, Source: src,
		})
	}
	return obs, rows.Err()
}

// LoadJourneyInputs reads only what journey assembly needs — activities
// overlapping the window plus path points and raw positions inside it — so a
// journey request does not load the whole observation corpus. Visits stay
// empty; assembly does not use them.
func (s *Store) LoadJourneyInputs(ctx context.Context, winStart, winEnd time.Time) (domain.Observations, error) {
	var obs domain.Observations

	rows, err := s.pool.Query(ctx, `SELECT start_ts, start_offset_sec, end_ts, end_offset_sec, start_lat, start_lon, end_lat, end_lon, distance_m, mode
		FROM activities WHERE end_ts >= $1 AND start_ts <= $2 ORDER BY start_ts, id`, winStart, winEnd)
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

	rows, err = s.pool.Query(ctx, `SELECT ts, ts_offset_sec, lat, lon FROM path_points
		WHERE ts BETWEEN $1 AND $2 ORDER BY ts, id`, winStart, winEnd)
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
	if err := rows.Err(); err != nil {
		return obs, err
	}

	rows, err = s.pool.Query(ctx, `SELECT ts, ts_offset_sec, lat, lon, accuracy_m, source FROM raw_positions
		WHERE ts BETWEEN $1 AND $2 ORDER BY ts, id`, winStart, winEnd)
	if err != nil {
		return obs, err
	}
	for rows.Next() {
		var ts time.Time
		var off int
		var lat, lon *float64
		var acc float64
		var src string
		if err := rows.Scan(&ts, &off, &lat, &lon, &acc, &src); err != nil {
			return obs, err
		}
		obs.RawPositions = append(obs.RawPositions, domain.RawPosition{
			Time: withOffset(ts, off), Loc: locOf(lat, lon), AccuracyM: acc, Source: src,
		})
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

func hashRawPosition(rp domain.RawPosition) string {
	return hashParts("r", canonTime(rp.Time), canonLoc(rp.Loc),
		strconv.FormatFloat(rp.AccuracyM, 'g', -1, 64), rp.Source)
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
