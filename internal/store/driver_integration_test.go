package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/t0mer/raptor/internal/models"
)

// TestNetworkedDrivers exercises the full migrate + a representative round-trip
// against a real PostgreSQL and/or MySQL server. It is skipped unless the
// corresponding DSN env var is set, so it stays a no-op in normal CI:
//
//	RAPTOR_TEST_PG_DSN=postgres://raptor:raptor@localhost:5432/raptor?sslmode=disable
//	RAPTOR_TEST_MYSQL_DSN="raptor:raptor@tcp(localhost:3306)/raptor"
func TestNetworkedDrivers(t *testing.T) {
	for _, tc := range []struct{ driver, env string }{
		{DriverPostgres, "RAPTOR_TEST_PG_DSN"},
		{DriverMySQL, "RAPTOR_TEST_MYSQL_DSN"},
	} {
		dsn := os.Getenv(tc.env)
		if dsn == "" {
			t.Logf("skipping %s: %s not set", tc.driver, tc.env)
			continue
		}
		t.Run(tc.driver, func(t *testing.T) {
			st, err := OpenWith(Options{Driver: tc.driver, DSN: dsn})
			if err != nil {
				t.Fatalf("open %s: %v", tc.driver, err)
			}
			defer st.Close()
			roundTrip(t, st)
		})
	}
}

// roundTrip touches every dialect-sensitive path: reserved-word identifiers
// ("groups", "condition"), the partial/plain alias index, JSON columns, the
// LOWER(email) lookup, and the rebind placeholder rewriting.
func roundTrip(t *testing.T, st *Store) {
	t.Helper()
	ctx := context.Background()

	tok := &models.Token{UUID: uuid.NewString(), Alias: uuid.NewString(), CORS: true, Actions: true}
	if err := st.CreateToken(ctx, tok); err != nil {
		t.Fatalf("create token: %v", err)
	}
	got, err := st.GetToken(ctx, tok.UUID)
	if err != nil || !got.CORS || !got.Actions {
		t.Fatalf("get token: %v cors=%v actions=%v", err, got.CORS, got.Actions)
	}

	req := &models.Request{
		UUID: uuid.NewString(), TokenID: tok.UUID, Method: "POST",
		Headers: map[string][]string{"X-Test": {"1"}}, Content: "hello ☃",
	}
	if err := st.CreateRequest(ctx, req, 0); err != nil {
		t.Fatalf("create request: %v", err)
	}
	reqs, err := st.ListRequests(ctx, tok.UUID, 10, 0)
	if err != nil || len(reqs) != 1 {
		t.Fatalf("list requests: %v n=%d", err, len(reqs))
	}

	// "groups" table (reserved word).
	grp := &models.Group{ID: uuid.NewString(), Name: "g"}
	if err := st.CreateGroup(ctx, grp); err != nil {
		t.Fatalf("create group: %v", err)
	}

	// "condition" column (reserved word).
	act := &models.Action{UUID: uuid.NewString(), TokenID: tok.UUID, Type: "stop", Condition: "x"}
	if err := st.CreateAction(ctx, act); err != nil {
		t.Fatalf("create action: %v", err)
	}
	acts, err := st.ListActions(ctx, tok.UUID)
	if err != nil || len(acts) != 1 || acts[0].Condition != "x" {
		t.Fatalf("list actions: %v", err)
	}

	// LOWER(email) case-insensitive lookup.
	email := uuid.NewString() + "@Example.COM"
	usr := &models.User{ID: uuid.NewString(), Email: email}
	if err := st.CreateUser(ctx, usr); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := st.GetUserByEmail(ctx, email); err != nil {
		t.Fatalf("get user by exact email: %v", err)
	}

	// Cleanup so reruns stay clean.
	_ = st.DeleteToken(ctx, tok.UUID)
	_ = st.DeleteUser(ctx, usr.ID)
	_, _ = st.exec(ctx, `DELETE FROM "groups" WHERE id = ?`, grp.ID)
	_ = time.Now
}
