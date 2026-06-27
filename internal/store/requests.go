package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/t0mer/raptor/internal/models"
)

const requestColumns = `uuid, token_id, type, method, ip, country, country_code, region, city,
	hostname, user_agent, content, query, headers, url, size, sorting,
	custom_action_output, custom_action_errors, exec_time, created_at`

// CreateRequest stores a captured request and updates the token's
// latest_request_at. It also enforces the token's request_limit by pruning the
// oldest requests beyond the cap (0 = unlimited).
func (s *Store) CreateRequest(ctx context.Context, r *models.Request, requestLimit int) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	if r.Sorting == 0 {
		r.Sorting = r.CreatedAt.UnixMilli()
	}
	if r.Type == "" {
		r.Type = models.RequestTypeWeb
	}

	query, err := marshalJSON(r.Query)
	if err != nil {
		return fmt.Errorf("marshal query: %w", err)
	}
	headers, err := marshalJSON(r.Headers)
	if err != nil {
		return fmt.Errorf("marshal headers: %w", err)
	}
	caOut, err := marshalJSON(r.CustomActionOutput)
	if err != nil {
		return fmt.Errorf("marshal custom_action_output: %w", err)
	}
	caErr, err := marshalJSON(r.CustomActionErrors)
	if err != nil {
		return fmt.Errorf("marshal custom_action_errors: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO requests (`+requestColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.UUID, r.TokenID, r.Type, r.Method, r.IP, r.Country, r.CountryCode, r.Region, r.City,
		r.Hostname, r.UserAgent, r.Content, query, headers, r.URL, r.Size, r.Sorting,
		caOut, caErr, r.ExecTime, nowRFC3339(r.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert request: %w", err)
	}

	if err := s.touchToken(ctx, r.TokenID, r.CreatedAt); err != nil {
		return fmt.Errorf("touch token: %w", err)
	}
	if requestLimit > 0 {
		if err := s.pruneRequests(ctx, r.TokenID, requestLimit); err != nil {
			return fmt.Errorf("prune requests: %w", err)
		}
	}
	return nil
}

// GetRequest returns a single request (without attached files).
func (s *Store) GetRequest(ctx context.Context, uuid string) (*models.Request, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+requestColumns+` FROM requests WHERE uuid = ?`, uuid)
	return scanRequest(row)
}

// ListRequests returns a token's requests, newest first, with limit/offset
// paging. A limit <= 0 defaults to 50; it is capped at 100.
func (s *Store) ListRequests(ctx context.Context, tokenID string, limit, offset int) ([]*models.Request, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+requestColumns+`
		FROM requests WHERE token_id = ? ORDER BY sorting DESC LIMIT ? OFFSET ?`,
		tokenID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query requests: %w", err)
	}
	defer rows.Close()

	var out []*models.Request
	for rows.Next() {
		r, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LatestRequest returns the most recent request for a token.
func (s *Store) LatestRequest(ctx context.Context, tokenID string) (*models.Request, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+requestColumns+`
		FROM requests WHERE token_id = ? ORDER BY sorting DESC LIMIT 1`, tokenID)
	return scanRequest(row)
}

// CountRequests returns the number of requests stored for a token.
func (s *Store) CountRequests(ctx context.Context, tokenID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM requests WHERE token_id = ?`, tokenID).Scan(&n)
	return n, err
}

// DeleteRequest removes a single request.
func (s *Store) DeleteRequest(ctx context.Context, uuid string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM requests WHERE uuid = ?`, uuid)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	return requireAffected(res)
}

// DeleteAllRequests removes every request for a token.
func (s *Store) DeleteAllRequests(ctx context.Context, tokenID string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM requests WHERE token_id = ?`, tokenID)
	if err != nil {
		return 0, fmt.Errorf("delete requests: %w", err)
	}
	return res.RowsAffected()
}

// pruneRequests keeps only the newest `keep` requests for a token.
func (s *Store) pruneRequests(ctx context.Context, tokenID string, keep int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM requests WHERE token_id = ? AND uuid NOT IN (
		SELECT uuid FROM requests WHERE token_id = ? ORDER BY sorting DESC LIMIT ?
	)`, tokenID, tokenID, keep)
	return err
}

func scanRequest(sc scanner) (*models.Request, error) {
	var (
		r              models.Request
		query, headers string
		caOut, caErr   string
		created        string
	)
	err := sc.Scan(
		&r.UUID, &r.TokenID, &r.Type, &r.Method, &r.IP, &r.Country, &r.CountryCode, &r.Region, &r.City,
		&r.Hostname, &r.UserAgent, &r.Content, &query, &headers, &r.URL, &r.Size, &r.Sorting,
		&caOut, &caErr, &r.ExecTime, &created,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan request: %w", err)
	}
	if err := unmarshalJSON(query, &r.Query); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(headers, &r.Headers); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(caOut, &r.CustomActionOutput); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(caErr, &r.CustomActionErrors); err != nil {
		return nil, err
	}
	if r.CreatedAt, err = parseTime(created); err != nil {
		return nil, err
	}
	return &r, nil
}

func marshalJSON(v any) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalJSON(s string, dst any) error {
	if s == "" || s == "{}" || s == "null" {
		return nil
	}
	return json.Unmarshal([]byte(s), dst)
}
