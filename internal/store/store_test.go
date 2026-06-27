package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newToken() *models.Token {
	return &models.Token{
		UUID:               uuid.NewString(),
		DefaultStatus:      200,
		DefaultContentType: "text/plain",
		Premium:            true,
	}
}

func TestMigrateIdempotent(t *testing.T) {
	s := newTestStore(t)
	// Re-running migrate must be a no-op.
	if err := s.migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestTokenRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	tok := newToken()
	tok.Alias = "my-alias"
	tok.Description = "demo"
	tok.CORS = true
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	got, err := s.GetToken(ctx, tok.UUID)
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if got.Alias != "my-alias" || !got.CORS || got.DefaultStatus != 200 {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	byAlias, err := s.GetTokenByAlias(ctx, "my-alias")
	if err != nil || byAlias.UUID != tok.UUID {
		t.Fatalf("GetTokenByAlias: %v / %v", err, byAlias)
	}

	tok.Description = "updated"
	if err := s.UpdateToken(ctx, tok); err != nil {
		t.Fatalf("UpdateToken: %v", err)
	}
	got, _ = s.GetToken(ctx, tok.UUID)
	if got.Description != "updated" {
		t.Errorf("Description = %q, want updated", got.Description)
	}
}

func TestGetTokenNotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.GetToken(context.Background(), "nope"); err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRequestRoundTripAndCascade(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	tok := newToken()
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatal(err)
	}

	req := &models.Request{
		UUID:    uuid.NewString(),
		TokenID: tok.UUID,
		Method:  "POST",
		IP:      "1.2.3.4",
		Content: "hello",
		Query:   map[string][]string{"a": {"1"}},
		Headers: map[string][]string{"Content-Type": {"text/plain"}},
		URL:     "http://localhost/" + tok.UUID,
		Size:    5,
	}
	if err := s.CreateRequest(ctx, req, 0); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}

	got, err := s.GetRequest(ctx, req.UUID)
	if err != nil {
		t.Fatalf("GetRequest: %v", err)
	}
	if got.Method != "POST" || got.Content != "hello" || got.Query["a"][0] != "1" {
		t.Errorf("request round-trip mismatch: %+v", got)
	}
	if got.Headers["Content-Type"][0] != "text/plain" {
		t.Errorf("headers not preserved: %+v", got.Headers)
	}

	// token latest_request_at updated
	tk, _ := s.GetToken(ctx, tok.UUID)
	if tk.LatestRequestAt == nil {
		t.Error("LatestRequestAt not set after request")
	}

	// attach a file, then delete the token and confirm cascade
	f := &models.File{ID: uuid.NewString(), RequestID: req.UUID, Filename: "a.txt", Size: 1}
	if err := s.CreateFile(ctx, f); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteToken(ctx, tok.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetRequest(ctx, req.UUID); err != ErrNotFound {
		t.Errorf("request not cascade-deleted: %v", err)
	}
	if _, err := s.GetFile(ctx, f.ID); err != ErrNotFound {
		t.Errorf("file not cascade-deleted: %v", err)
	}
}

func TestRequestLimitPrune(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	tok := newToken()
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		req := &models.Request{
			UUID:    uuid.NewString(),
			TokenID: tok.UUID,
			Method:  "GET",
			Sorting: int64(i + 1), // ascending so the last is newest
		}
		if err := s.CreateRequest(ctx, req, 3); err != nil {
			t.Fatalf("CreateRequest %d: %v", i, err)
		}
	}

	n, err := s.CountRequests(ctx, tok.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("CountRequests = %d, want 3 (pruned to request_limit)", n)
	}

	latest, err := s.LatestRequest(ctx, tok.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Sorting != 5 {
		t.Errorf("latest Sorting = %d, want 5", latest.Sorting)
	}
}

func TestListRequestsPaging(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	tok := newToken()
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		req := &models.Request{UUID: uuid.NewString(), TokenID: tok.UUID, Sorting: int64(i + 1)}
		if err := s.CreateRequest(ctx, req, 0); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.ListRequests(ctx, tok.UUID, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 3 {
		t.Fatalf("len(page) = %d, want 3", len(page))
	}
	if page[0].Sorting != 10 {
		t.Errorf("first Sorting = %d, want 10 (newest first)", page[0].Sorting)
	}
}
