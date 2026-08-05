// Package storetest provides a real-database harness for store tests: a
// fresh scratch database per test, migrated with the embedded migrations,
// dropped on cleanup. Mocks are deliberately absent — the store's risk is
// the SQL against real Postgres, and mocking pgx would test the mock
// (docs/phase-4/DECISIONS.md).
//
// The admin connection comes from ROADBOOK_TEST_DB, resolved by
// scripts/test.sh so the tests run by default; a missing value skips them
// visibly. The harness only ever creates and drops databases named
// roadbook_test_*; whatever else lives on the server is not touched.
//
// Context worth keeping (from the maintainer, 2026-08-05): if Roadbook is
// ever hosted for other people, every store query gains a user filter, and
// a missed one is a cross-user data leak. This harness is what makes that
// change safe to attempt.
package storetest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"roadbook/internal/store"
)

var counter atomic.Int64

// Open returns a Store bound to a fresh, fully-migrated scratch database.
// Cleanup (close + drop) is registered on t automatically.
func Open(t *testing.T) *store.Store {
	t.Helper()
	admin := os.Getenv("ROADBOOK_TEST_DB")
	if admin == "" {
		t.Skip("ROADBOOK_TEST_DB not set — DB-backed store tests skipped; `make test` resolves a database automatically (scripts/test.sh)")
	}
	ctx := context.Background()

	name := fmt.Sprintf("roadbook_test_%d_%d_%d", os.Getpid(), time.Now().UnixNano(), counter.Add(1))
	adminConn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatalf("storetest: connecting to admin database %s: %v", admin, err)
	}
	if _, err := adminConn.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		adminConn.Close(ctx)
		t.Fatalf("storetest: creating scratch database: %v", err)
	}
	adminConn.Close(ctx)

	scratchURL, err := withDatabase(admin, name)
	if err != nil {
		t.Fatalf("storetest: %v", err)
	}
	if _, err := store.Migrate(ctx, scratchURL); err != nil {
		dropDatabase(t, admin, name)
		t.Fatalf("storetest: migrating scratch database: %v", err)
	}
	s, err := store.Open(ctx, scratchURL)
	if err != nil {
		dropDatabase(t, admin, name)
		t.Fatalf("storetest: opening scratch database: %v", err)
	}
	t.Cleanup(func() {
		s.Close()
		dropDatabase(t, admin, name)
	})
	return s
}

func dropDatabase(t *testing.T, admin, name string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Errorf("storetest: reconnecting to drop %s: %v", name, err)
		return
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
		t.Errorf("storetest: dropping %s: %v", name, err)
	}
}

// withDatabase swaps the database name in a postgres URL, preserving
// credentials, host, port, and options.
func withDatabase(dbURL, name string) (string, error) {
	u, err := url.Parse(dbURL)
	if err != nil {
		return "", fmt.Errorf("parsing ROADBOOK_TEST_DB: %w", err)
	}
	u.Path = "/" + name
	return u.String(), nil
}
