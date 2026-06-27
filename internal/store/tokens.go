package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/t0mer/raptor/internal/models"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

const tokenColumns = `uuid, alias, default_status, default_content, default_content_type,
	timeout, cors, expiry, actions, request_limit, description, listen, redirect,
	password, group_id, premium, created_at, updated_at, latest_request_at`

// CreateToken inserts a new token. CreatedAt/UpdatedAt are set if zero.
func (s *Store) CreateToken(ctx context.Context, t *models.Token) error {
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `INSERT INTO tokens (`+tokenColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.UUID, t.Alias, t.DefaultStatus, t.DefaultContent, t.DefaultContentType,
		t.Timeout, boolToInt(t.CORS), t.Expiry, boolToInt(t.Actions), t.RequestLimit,
		t.Description, t.Listen, t.Redirect, t.Password, t.GroupID, boolToInt(t.Premium),
		nowRFC3339(t.CreatedAt), nowRFC3339(t.UpdatedAt), nullTime(t.LatestRequestAt),
	)
	if err != nil {
		return fmt.Errorf("insert token: %w", err)
	}
	return nil
}

// GetToken returns the token with the given UUID.
func (s *Store) GetToken(ctx context.Context, uuid string) (*models.Token, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+tokenColumns+` FROM tokens WHERE uuid = ?`, uuid)
	return scanToken(row)
}

// GetTokenByAlias returns the token with the given non-empty alias.
func (s *Store) GetTokenByAlias(ctx context.Context, alias string) (*models.Token, error) {
	if alias == "" {
		return nil, ErrNotFound
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+tokenColumns+` FROM tokens WHERE alias = ?`, alias)
	return scanToken(row)
}

// ListTokens returns all tokens, most recently created first.
func (s *Store) ListTokens(ctx context.Context) ([]*models.Token, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+tokenColumns+` FROM tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query tokens: %w", err)
	}
	defer rows.Close()

	var out []*models.Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateToken persists mutable token fields and bumps UpdatedAt.
func (s *Store) UpdateToken(ctx context.Context, t *models.Token) error {
	t.UpdatedAt = time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `UPDATE tokens SET
		alias=?, default_status=?, default_content=?, default_content_type=?,
		timeout=?, cors=?, expiry=?, actions=?, request_limit=?, description=?,
		listen=?, redirect=?, password=?, group_id=?, premium=?, updated_at=?
		WHERE uuid=?`,
		t.Alias, t.DefaultStatus, t.DefaultContent, t.DefaultContentType,
		t.Timeout, boolToInt(t.CORS), t.Expiry, boolToInt(t.Actions), t.RequestLimit,
		t.Description, t.Listen, t.Redirect, t.Password, t.GroupID, boolToInt(t.Premium),
		nowRFC3339(t.UpdatedAt), t.UUID,
	)
	if err != nil {
		return fmt.Errorf("update token: %w", err)
	}
	return requireAffected(res)
}

// DeleteToken removes a token and (via cascade) its requests and files.
func (s *Store) DeleteToken(ctx context.Context, uuid string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tokens WHERE uuid = ?`, uuid)
	if err != nil {
		return fmt.Errorf("delete token: %w", err)
	}
	return requireAffected(res)
}

// touchToken updates latest_request_at to the given time.
func (s *Store) touchToken(ctx context.Context, uuid string, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tokens SET latest_request_at = ? WHERE uuid = ?`, nowRFC3339(at), uuid)
	return err
}

// scanner is satisfied by *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanToken(sc scanner) (*models.Token, error) {
	var (
		t        models.Token
		cors     int
		actions  int
		premium  int
		created  string
		updated  string
		latestNS sql.NullString
	)
	err := sc.Scan(
		&t.UUID, &t.Alias, &t.DefaultStatus, &t.DefaultContent, &t.DefaultContentType,
		&t.Timeout, &cors, &t.Expiry, &actions, &t.RequestLimit, &t.Description,
		&t.Listen, &t.Redirect, &t.Password, &t.GroupID, &premium,
		&created, &updated, &latestNS,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan token: %w", err)
	}
	t.CORS = cors != 0
	t.Actions = actions != 0
	t.Premium = premium != 0
	if t.CreatedAt, err = parseTime(created); err != nil {
		return nil, err
	}
	if t.UpdatedAt, err = parseTime(updated); err != nil {
		return nil, err
	}
	if latestNS.Valid && latestNS.String != "" {
		lt, err := parseTime(latestNS.String)
		if err != nil {
			return nil, err
		}
		t.LatestRequestAt = &lt
	}
	return &t, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return nowRFC3339(*t)
}

func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
