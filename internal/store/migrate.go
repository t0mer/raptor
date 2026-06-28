package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

//go:embed migrations
var migrationsFS embed.FS

// migrate applies any embedded migrations for the active driver that are not yet
// recorded in schema_migrations, each within its own transaction, in filename
// order. Each driver has its own dialect-specific migration directory under
// migrations/<driver>/.
func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    VARCHAR(255) PRIMARY KEY,
		applied_at VARCHAR(64) NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := s.appliedVersions(ctx)
	if err != nil {
		return err
	}

	dir := "migrations/" + s.driver
	entries, err := fs.ReadDir(migrationsFS, dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if applied[name] {
			continue
		}
		body, err := migrationsFS.ReadFile(dir + "/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := s.applyMigration(ctx, name, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func (s *Store) appliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := s.query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func (s *Store) applyMigration(ctx context.Context, name, body string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after a successful commit

	// Run each statement individually: the MySQL driver does not allow multiple
	// statements per Exec by default, and splitting avoids enabling the
	// multi-statement connection mode (a safer default for app queries).
	for _, stmt := range splitStatements(body) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("statement %q: %w", truncate(stmt), err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		s.rebind(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`),
		name, time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return err
	}
	return tx.Commit()
}

// splitStatements strips "--" line comments and splits a migration body on ";"
// into individual statements (empty fragments dropped). Migration SQL never
// contains ";" or "--" inside string literals, so this naive split is safe.
func splitStatements(body string) []string {
	var clean strings.Builder
	for _, line := range strings.Split(body, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		clean.WriteString(line)
		clean.WriteByte('\n')
	}
	var out []string
	for _, part := range strings.Split(clean.String(), ";") {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return out
}

func truncate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}
