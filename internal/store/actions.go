package store

import (
	"context"
	"fmt"
	"time"

	"github.com/t0mer/raptor/internal/models"
)

const actionColumns = `uuid, token_id, type, position, name, disabled, parameters,
	queue, delay, condition, created_at, updated_at`

// CreateAction inserts a new action. If Position is zero it is appended to the
// end of the token's chain.
func (s *Store) CreateAction(ctx context.Context, a *models.Action) error {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	if a.Position == 0 {
		a.Position = s.nextActionPosition(ctx, a.TokenID)
	}
	params, err := marshalJSON(a.Parameters)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO actions (`+actionColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.UUID, a.TokenID, a.Type, a.Position, a.Name, boolToInt(a.Disabled), params,
		boolToInt(a.Queue), a.Delay, a.Condition, nowRFC3339(a.CreatedAt), nowRFC3339(a.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert action: %w", err)
	}
	return nil
}

// GetAction returns a single action.
func (s *Store) GetAction(ctx context.Context, uuid string) (*models.Action, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+actionColumns+` FROM actions WHERE uuid = ?`, uuid)
	return scanAction(row)
}

// ListActions returns a token's actions in execution order.
func (s *Store) ListActions(ctx context.Context, tokenID string) ([]*models.Action, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+actionColumns+`
		FROM actions WHERE token_id = ? ORDER BY position ASC, created_at ASC`, tokenID)
	if err != nil {
		return nil, fmt.Errorf("query actions: %w", err)
	}
	defer rows.Close()
	var out []*models.Action
	for rows.Next() {
		a, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateAction persists an action's mutable fields.
func (s *Store) UpdateAction(ctx context.Context, a *models.Action) error {
	a.UpdatedAt = time.Now().UTC()
	params, err := marshalJSON(a.Parameters)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `UPDATE actions SET
		type=?, position=?, name=?, disabled=?, parameters=?, queue=?, delay=?, condition=?, updated_at=?
		WHERE uuid=?`,
		a.Type, a.Position, a.Name, boolToInt(a.Disabled), params,
		boolToInt(a.Queue), a.Delay, a.Condition, nowRFC3339(a.UpdatedAt), a.UUID,
	)
	if err != nil {
		return fmt.Errorf("update action: %w", err)
	}
	return requireAffected(res)
}

// DeleteAction removes an action.
func (s *Store) DeleteAction(ctx context.Context, uuid string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM actions WHERE uuid = ?`, uuid)
	if err != nil {
		return fmt.Errorf("delete action: %w", err)
	}
	return requireAffected(res)
}

func (s *Store) nextActionPosition(ctx context.Context, tokenID string) int {
	var max *int
	_ = s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM actions WHERE token_id = ?`, tokenID).Scan(&max)
	if max == nil {
		return 1
	}
	return *max + 1
}

// CreateActionRun records the outcome of one executed action.
func (s *Store) CreateActionRun(ctx context.Context, r *models.ActionRun) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO action_runs (id, request_id, action_id, action_type, action_name, position, output, error, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		r.ID, r.RequestID, r.ActionID, r.ActionType, r.ActionName, r.Position, r.Output, r.Error, nowRFC3339(r.CreatedAt))
	if err != nil {
		return fmt.Errorf("insert action_run: %w", err)
	}
	return nil
}

// ListActionRuns returns the action run log for a request, in order.
func (s *Store) ListActionRuns(ctx context.Context, requestID string) ([]*models.ActionRun, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, request_id, action_id, action_type, action_name, position, output, error, created_at
		 FROM action_runs WHERE request_id = ? ORDER BY position ASC, created_at ASC`, requestID)
	if err != nil {
		return nil, fmt.Errorf("query action_runs: %w", err)
	}
	defer rows.Close()
	var out []*models.ActionRun
	for rows.Next() {
		var (
			r       models.ActionRun
			created string
		)
		if err := rows.Scan(&r.ID, &r.RequestID, &r.ActionID, &r.ActionType, &r.ActionName,
			&r.Position, &r.Output, &r.Error, &created); err != nil {
			return nil, err
		}
		r.CreatedAt, _ = parseTime(created)
		out = append(out, &r)
	}
	return out, rows.Err()
}

func scanAction(sc scanner) (*models.Action, error) {
	var (
		a                models.Action
		disabled, queue  int
		params           string
		created, updated string
	)
	if err := sc.Scan(&a.UUID, &a.TokenID, &a.Type, &a.Position, &a.Name, &disabled, &params,
		&queue, &a.Delay, &a.Condition, &created, &updated); err != nil {
		return nil, mapNoRows(err)
	}
	a.Disabled = disabled != 0
	a.Queue = queue != 0
	if err := unmarshalJSON(params, &a.Parameters); err != nil {
		return nil, err
	}
	a.CreatedAt, _ = parseTime(created)
	a.UpdatedAt, _ = parseTime(updated)
	return &a, nil
}
