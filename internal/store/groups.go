package store

import (
	"context"
	"fmt"
	"time"

	"github.com/t0mer/raptor/internal/models"
)

// CreateGroup inserts a new group.
func (s *Store) CreateGroup(ctx context.Context, g *models.Group) error {
	if g.CreatedAt.IsZero() {
		g.CreatedAt = time.Now().UTC()
	}
	_, err := s.exec(ctx,
		`INSERT INTO "groups" (id, name, color, created_at) VALUES (?,?,?,?)`,
		g.ID, g.Name, g.Color, nowRFC3339(g.CreatedAt))
	if err != nil {
		return fmt.Errorf("insert group: %w", err)
	}
	return nil
}

// GetGroup returns a group by id.
func (s *Store) GetGroup(ctx context.Context, id string) (*models.Group, error) {
	row := s.queryRow(ctx,
		`SELECT id, name, color, created_at FROM "groups" WHERE id = ?`, id)
	return scanGroup(row)
}

// ListGroups returns all groups, newest first.
func (s *Store) ListGroups(ctx context.Context) ([]*models.Group, error) {
	rows, err := s.query(ctx,
		`SELECT id, name, color, created_at FROM "groups" ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}
	defer rows.Close()

	var out []*models.Group
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// UpdateGroup persists a group's mutable fields.
func (s *Store) UpdateGroup(ctx context.Context, g *models.Group) error {
	res, err := s.exec(ctx,
		`UPDATE "groups" SET name = ?, color = ? WHERE id = ?`, g.Name, g.Color, g.ID)
	if err != nil {
		return fmt.Errorf("update group: %w", err)
	}
	return requireAffected(res)
}

// DeleteGroup removes a group and clears the group_id of any tokens in it (the
// tokens themselves are kept).
func (s *Store) DeleteGroup(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	if _, err := tx.ExecContext(ctx, s.rebind(`UPDATE tokens SET group_id = '' WHERE group_id = ?`), id); err != nil {
		return fmt.Errorf("clear token groups: %w", err)
	}
	res, err := tx.ExecContext(ctx, s.rebind(`DELETE FROM "groups" WHERE id = ?`), id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if err := requireAffected(res); err != nil {
		return err
	}
	return tx.Commit()
}

func scanGroup(sc scanner) (*models.Group, error) {
	var (
		g       models.Group
		created string
	)
	if err := sc.Scan(&g.ID, &g.Name, &g.Color, &created); err != nil {
		return nil, mapNoRows(err)
	}
	var err error
	if g.CreatedAt, err = parseTime(created); err != nil {
		return nil, err
	}
	return &g, nil
}
