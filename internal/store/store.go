// Package store provides persistence for Raptor. It supports three drivers
// behind one Store API: the pure-Go modernc.org/sqlite (default, keeps CGO
// disabled), PostgreSQL (via the pure-Go jackc/pgx stdlib driver) and MySQL
// (via go-sql-driver/mysql). All three are CGO-free.
//
// Portability notes:
//   - Booleans are stored as 0/1 integers, timestamps as RFC3339 TEXT strings,
//     and all primary keys are UUID/text — so the schema maps cleanly onto every
//     dialect. The only per-dialect concern at the query layer is the bind
//     placeholder: sqlite and MySQL use "?", PostgreSQL uses "$N". Queries are
//     written with "?" and rewritten for PostgreSQL by rebind().
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Supported driver identifiers (match config --db-driver values).
const (
	DriverSQLite   = "sqlite"
	DriverPostgres = "postgres"
	DriverMySQL    = "mysql"
)

// secretCipher encrypts/decrypts secret values at rest. Implemented by
// *crypto.Cipher; kept as an interface so store does not import crypto directly.
type secretCipher interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}

// Store wraps a database handle and the resolved driver.
type Store struct {
	db     *sql.DB
	driver string
	cipher secretCipher
}

// SetCipher enables transparent at-rest encryption of secret columns. When
// unset, secrets are stored as-is (used by tests that don't exercise secrets).
func (s *Store) SetCipher(c secretCipher) { s.cipher = c }

// encryptSecret encrypts a secret for storage (no-op without a cipher).
func (s *Store) encryptSecret(v string) (string, error) {
	if s.cipher == nil || v == "" {
		return v, nil
	}
	return s.cipher.Encrypt(v)
}

// decryptSecret reverses encryptSecret (no-op without a cipher).
func (s *Store) decryptSecret(v string) (string, error) {
	if s.cipher == nil || v == "" {
		return v, nil
	}
	return s.cipher.Decrypt(v)
}

// Options describe a database connection. Driver selects the backend; the other
// fields are interpreted per driver (sqlite uses Path; postgres/mysql use the
// host/credentials or a ready-made DSN override).
type Options struct {
	Driver string // sqlite | postgres | mysql

	// SQLite.
	Path string // file path or ":memory:"

	// Networked drivers (postgres/mysql).
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string // postgres sslmode (ignored by mysql)

	// DSN, when non-empty, fully overrides the structured fields above.
	DSN string
}

// Open opens (and migrates) a SQLite database at the given file path. Pass
// ":memory:" for an ephemeral in-memory database (tests). It is a thin shim over
// OpenWith for the common sqlite case used throughout the codebase and tests.
func Open(path string) (*Store, error) {
	return OpenWith(Options{Driver: DriverSQLite, Path: path})
}

// OpenWith opens (and migrates) a database for the configured driver.
func OpenWith(opts Options) (*Store, error) {
	if opts.Driver == "" {
		opts.Driver = DriverSQLite
	}

	sqlDriver, dsn, err := resolveDSN(opts)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", opts.Driver, err)
	}

	if opts.Driver == DriverSQLite {
		// SQLite is single-writer; cap connections to avoid "database is locked".
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s: %w", opts.Driver, err)
	}

	s := &Store{db: db, driver: opts.Driver}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// resolveDSN maps Options onto the database/sql driver name and a DSN string.
func resolveDSN(opts Options) (driverName, dsn string, err error) {
	switch opts.Driver {
	case DriverSQLite:
		return "sqlite", sqliteDSN(opts.Path), nil
	case DriverPostgres:
		if opts.DSN != "" {
			return "pgx", opts.DSN, nil
		}
		return "pgx", postgresDSN(opts), nil
	case DriverMySQL:
		dsn, err := mysqlDSN(opts)
		if err != nil {
			return "", "", err
		}
		return "mysql", dsn, nil
	default:
		return "", "", fmt.Errorf("unsupported db driver %q", opts.Driver)
	}
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

func postgresDSN(o Options) string {
	port := o.Port
	if port == 0 {
		port = 5432
	}
	sslmode := o.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(o.User, o.Password),
		Host:   fmt.Sprintf("%s:%d", o.Host, port),
		Path:   "/" + o.Name,
	}
	q := url.Values{}
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String()
}

// mysqlDSN builds (or normalises) the go-sql-driver DSN. Whether the connection
// comes from structured Options or a full DSN override, it is forced to enable
// ANSI_QUOTES — the shared queries quote reserved words ("groups", "condition")
// with "" and rely on it being an identifier quote (as it natively is in
// sqlite/postgres). ParseTime/UTC/utf8mb4 are also ensured.
func mysqlDSN(o Options) (string, error) {
	var c *mysql.Config
	if o.DSN != "" {
		parsed, err := mysql.ParseDSN(o.DSN)
		if err != nil {
			return "", fmt.Errorf("parse mysql dsn: %w", err)
		}
		c = parsed
	} else {
		port := o.Port
		if port == 0 {
			port = 3306
		}
		c = mysql.NewConfig()
		c.User = o.User
		c.Passwd = o.Password
		c.Net = "tcp"
		c.Addr = fmt.Sprintf("%s:%d", o.Host, port)
		c.DBName = o.Name
	}
	c.ParseTime = true
	c.Loc = time.UTC
	if c.Params == nil {
		c.Params = map[string]string{}
	}
	if _, ok := c.Params["charset"]; !ok {
		c.Params["charset"] = "utf8mb4"
	}
	c.Params["sql_mode"] = withANSIQuotes(c.Params["sql_mode"])
	return c.FormatDSN(), nil
}

// withANSIQuotes returns a quoted sql_mode value that includes ANSI_QUOTES,
// preserving any modes already present. An empty input yields a sensible strict
// default set.
func withANSIQuotes(current string) string {
	current = strings.Trim(strings.TrimSpace(current), "'")
	if current == "" {
		return "'ANSI_QUOTES,STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION'"
	}
	modes := make([]string, 0, 4)
	hasANSI := false
	for _, m := range strings.Split(current, ",") {
		if m = strings.TrimSpace(m); m == "" {
			continue
		}
		modes = append(modes, m)
		if strings.EqualFold(m, "ANSI_QUOTES") {
			hasANSI = true
		}
	}
	if !hasANSI {
		modes = append(modes, "ANSI_QUOTES")
	}
	return "'" + strings.Join(modes, ",") + "'"
}

// rebind rewrites "?" placeholders to "$1, $2, …" for PostgreSQL; for sqlite and
// MySQL (which use "?") it returns the query unchanged. Placeholders inside
// single-quoted string literals are left alone.
func (s *Store) rebind(query string) string {
	if s.driver != DriverPostgres || !strings.Contains(query, "?") {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n, inQuote := 0, false
	for i := 0; i < len(query); i++ {
		c := query[i]
		switch {
		case c == '\'':
			inQuote = !inQuote
			b.WriteByte(c)
		case c == '?' && !inQuote:
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// exec/query/queryRow wrap the database handle so every statement passes through
// rebind() — the single point where dialect placeholder differences are handled.
func (s *Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, s.rebind(query), args...)
}

func (s *Store) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, s.rebind(query), args...)
}

func (s *Store) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, s.rebind(query), args...)
}

// DB exposes the underlying handle for advanced callers (e.g. health checks).
func (s *Store) DB() *sql.DB { return s.db }

// Driver returns the resolved driver identifier (sqlite|postgres|mysql).
func (s *Store) Driver() string { return s.driver }

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
