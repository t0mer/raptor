// Package store provides persistence for Raptor. The default driver is the
// pure-Go modernc.org/sqlite (keeping CGO disabled); a Postgres path is added
// in a later phase behind the same Store API.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a database handle and the resolved driver.
type Store struct {
	db     *sql.DB
	driver string
}

// Open opens (and migrates) a SQLite database at the given file path. Pass
// ":memory:" for an ephemeral in-memory database (tests).
func Open(path string) (*Store, error) {
	dsn := sqliteDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite is single-writer; cap connections to avoid "database is locked".
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &Store{db: db, driver: "sqlite"}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func sqliteDSN(path string) string {
	if path != ":memory:" {
		path = filepath.Clean(path)
	}
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(WAL)")
	return "file:" + path + "?" + q.Encode()
}

// DB exposes the underlying handle for advanced callers (e.g. health checks).
func (s *Store) DB() *sql.DB { return s.db }

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// nowRFC3339 formats a timestamp for storage in TEXT columns.
func nowRFC3339(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// parseTime parses a stored RFC3339 timestamp.
func parseTime(s string) (time.Time, error) { return time.Parse(time.RFC3339Nano, s) }

// mapNoRows converts sql.ErrNoRows into the package-level ErrNotFound.
func mapNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
