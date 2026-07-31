package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"roadbook/internal/detect"
	"roadbook/internal/domain"
)

// Run is one stored detection run. Params is the exact JSON recorded at run
// time (invariant 3) — kept raw here; consumers unmarshal as needed.
type Run struct {
	ID              int64
	RanAt           time.Time
	Params          json.RawMessage
	OutliersDropped int
}

// CandidateRow is a stored candidate. Span times carry their original UTC
// offsets, restored on load.
type CandidateRow struct {
	ID             int64
	RunID          int64
	Seq            int
	SpanStart      time.Time
	SpanEnd        time.Time
	Days           float64
	Dest           domain.LatLng
	DestKm         int
	TrackKm        int
	Stops          int
	Repeat         int
	ObsCount       int
	StartTruncated bool
	EndTruncated   bool
	Modes          []detect.ModeCount
}

// SaveRun records a detection run and its candidates. Candidates are derived
// and disposable; previous runs' candidates are retained so runs can be
// compared, and the UI reads only the latest run.
func (s *Store) SaveRun(ctx context.Context, p detect.Params, res detect.Result) (int64, error) {
	paramsJSON, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}
	basesJSON, err := json.Marshal(res.Bases)
	if err != nil {
		return 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var runID int64
	if err := tx.QueryRow(ctx, `INSERT INTO detection_runs (params, outliers_dropped, bases)
		VALUES ($1,$2,$3) RETURNING id`, paramsJSON, res.OutliersDropped, basesJSON).Scan(&runID); err != nil {
		return 0, err
	}

	b := &pgx.Batch{}
	for i, c := range res.Candidates {
		modesJSON, err := json.Marshal(c.Modes)
		if err != nil {
			return 0, err
		}
		b.Queue(`INSERT INTO candidates (run_id, seq, span_start, span_start_offset_sec, span_end, span_end_offset_sec,
			days, dest_lat, dest_lon, dest_km, track_km, stop_count, repeat_count, obs_count,
			start_truncated, end_truncated, modes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
			runID, i, c.Start, offsetOf(c.Start), c.End, offsetOf(c.End),
			c.Days, c.Dest.Lat, c.Dest.Lon, c.DestKm, c.TrackKm, c.Stops, c.Repeat, c.ObsCount,
			c.StartTruncated, c.EndTruncated, modesJSON)
	}
	br := tx.SendBatch(ctx, b)
	for range b.Len() {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return 0, err
		}
	}
	if err := br.Close(); err != nil {
		return 0, err
	}
	return runID, tx.Commit(ctx)
}

// LatestRun returns the most recent run and its candidates in rank order, or
// (nil, nil, nil) when no run exists yet.
func (s *Store) LatestRun(ctx context.Context) (*Run, []CandidateRow, error) {
	var r Run
	err := s.pool.QueryRow(ctx, `SELECT id, ran_at, params, outliers_dropped
		FROM detection_runs ORDER BY id DESC LIMIT 1`).Scan(&r.ID, &r.RanAt, &r.Params, &r.OutliersDropped)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	cands, err := s.candidatesForRun(ctx, r.ID)
	return &r, cands, err
}

// LatestCandidate returns one candidate by id, but only if it belongs to the
// latest run — after re-detection, older ids are stale and must 404, never
// silently decide the wrong span.
func (s *Store) LatestCandidate(ctx context.Context, id int64) (*CandidateRow, error) {
	rows, err := s.pool.Query(ctx, candidateSelect+`WHERE id = $1 AND run_id = (SELECT max(id) FROM detection_runs)`, id)
	if err != nil {
		return nil, err
	}
	cands, err := scanCandidates(rows)
	if err != nil || len(cands) == 0 {
		return nil, err
	}
	return &cands[0], nil
}

const candidateSelect = `SELECT id, run_id, seq, span_start, span_start_offset_sec, span_end, span_end_offset_sec,
	days, dest_lat, dest_lon, dest_km, track_km, stop_count, repeat_count, obs_count,
	start_truncated, end_truncated, modes FROM candidates `

func (s *Store) candidatesForRun(ctx context.Context, runID int64) ([]CandidateRow, error) {
	rows, err := s.pool.Query(ctx, candidateSelect+`WHERE run_id = $1 ORDER BY seq`, runID)
	if err != nil {
		return nil, err
	}
	return scanCandidates(rows)
}

func scanCandidates(rows pgx.Rows) ([]CandidateRow, error) {
	defer rows.Close()
	var out []CandidateRow
	for rows.Next() {
		var c CandidateRow
		var soff, eoff int
		var modes []byte
		if err := rows.Scan(&c.ID, &c.RunID, &c.Seq, &c.SpanStart, &soff, &c.SpanEnd, &eoff,
			&c.Days, &c.Dest.Lat, &c.Dest.Lon, &c.DestKm, &c.TrackKm, &c.Stops, &c.Repeat, &c.ObsCount,
			&c.StartTruncated, &c.EndTruncated, &modes); err != nil {
			return nil, err
		}
		c.SpanStart = withOffset(c.SpanStart, soff)
		c.SpanEnd = withOffset(c.SpanEnd, eoff)
		if err := json.Unmarshal(modes, &c.Modes); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
