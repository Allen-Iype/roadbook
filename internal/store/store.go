// Package store persists observations, detection runs, and decisions via pgx
// with hand-written SQL — no ORM (CLAUDE.md, "do not add"). It is a thin,
// boring layer by intent: every query is visible and explainable.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"

	"roadbook/migrations"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, dbURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: cannot reach database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// Migrate applies the embedded goose migrations. Go owns the schema; this is
// the only place it changes.
func Migrate(ctx context.Context, dbURL string) (int, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return 0, err
	}
	results, err := provider.Up(ctx)
	return len(results), err
}

// withOffset restores the civil-time view of a stored instant. timestamptz
// forgets the writer's UTC offset; detection needs it (see pyport.go), so it is
// stored alongside and re-applied on load.
func withOffset(t time.Time, offsetSec int) time.Time {
	return t.In(time.FixedZone("", offsetSec))
}

func offsetOf(t time.Time) int {
	_, off := t.Zone()
	return off
}
