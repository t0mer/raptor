package store

import (
	"context"
	"fmt"

	"github.com/t0mer/raptor/internal/models"
)

// CreateFile records a stored file blob associated with a request.
func (s *Store) CreateFile(ctx context.Context, f *models.File) error {
	_, err := s.exec(ctx,
		`INSERT INTO files (id, request_id, filename, content_type, size, path)
		 VALUES (?,?,?,?,?,?)`,
		f.ID, f.RequestID, f.Filename, f.ContentType, f.Size, f.Path)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}
	return nil
}

// GetFile returns a single file record by id.
func (s *Store) GetFile(ctx context.Context, id string) (*models.File, error) {
	row := s.queryRow(ctx,
		`SELECT id, request_id, filename, content_type, size, path FROM files WHERE id = ?`, id)
	var f models.File
	if err := row.Scan(&f.ID, &f.RequestID, &f.Filename, &f.ContentType, &f.Size, &f.Path); err != nil {
		return nil, mapNoRows(err)
	}
	return &f, nil
}

// ListFilesByRequest returns the files attached to a request.
func (s *Store) ListFilesByRequest(ctx context.Context, requestID string) ([]models.File, error) {
	rows, err := s.query(ctx,
		`SELECT id, request_id, filename, content_type, size, path FROM files WHERE request_id = ?`, requestID)
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()

	var out []models.File
	for rows.Next() {
		var f models.File
		if err := rows.Scan(&f.ID, &f.RequestID, &f.Filename, &f.ContentType, &f.Size, &f.Path); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
