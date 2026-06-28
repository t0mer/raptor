package store

import "testing"

func TestRebindPostgres(t *testing.T) {
	pg := &Store{driver: DriverPostgres}
	cases := []struct{ in, want string }{
		{`SELECT 1`, `SELECT 1`},
		{`WHERE a = ?`, `WHERE a = $1`},
		{`WHERE a = ? AND b = ?`, `WHERE a = $1 AND b = $2`},
		{`VALUES (?,?,?)`, `VALUES ($1,$2,$3)`},
		// A "?" inside a string literal must not be treated as a placeholder.
		{`WHERE x = '?' AND y = ?`, `WHERE x = '?' AND y = $1`},
	}
	for _, c := range cases {
		if got := pg.rebind(c.in); got != c.want {
			t.Errorf("rebind(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRebindNonPostgresPassthrough(t *testing.T) {
	for _, drv := range []string{DriverSQLite, DriverMySQL} {
		s := &Store{driver: drv}
		const q = `WHERE a = ? AND b = ?`
		if got := s.rebind(q); got != q {
			t.Errorf("%s rebind changed query: %q", drv, got)
		}
	}
}

func TestResolveDSN(t *testing.T) {
	// Postgres: structured fields build a URL DSN; password is escaped.
	driver, dsn, err := resolveDSN(Options{
		Driver: DriverPostgres, Host: "db", Name: "raptor",
		User: "u", Password: "p@ss/word", SSLMode: "require",
	})
	if err != nil {
		t.Fatal(err)
	}
	if driver != "pgx" {
		t.Errorf("postgres sql driver = %q, want pgx", driver)
	}
	want := "postgres://u:p%40ss%2Fword@db:5432/raptor?sslmode=require"
	if dsn != want {
		t.Errorf("postgres dsn = %q, want %q", dsn, want)
	}

	// MySQL: structured fields build a go-sql-driver DSN with the tcp addr.
	driver, dsn, err = resolveDSN(Options{
		Driver: DriverMySQL, Host: "db", Name: "raptor", User: "u", Password: "p",
	})
	if err != nil {
		t.Fatal(err)
	}
	if driver != "mysql" {
		t.Errorf("mysql sql driver = %q, want mysql", driver)
	}
	for _, sub := range []string{"u:p@tcp(db:3306)/raptor", "parseTime=true", "ANSI_QUOTES"} {
		if !contains(dsn, sub) {
			t.Errorf("mysql dsn %q missing %q", dsn, sub)
		}
	}

	// A full DSN override is used verbatim.
	_, dsn, err = resolveDSN(Options{Driver: DriverPostgres, DSN: "postgres://x/y"})
	if err != nil {
		t.Fatal(err)
	}
	if dsn != "postgres://x/y" {
		t.Errorf("dsn override = %q", dsn)
	}

	if _, _, err := resolveDSN(Options{Driver: "oracle"}); err == nil {
		t.Error("expected error for unsupported driver")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
