package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/kanban/feature/columns"
)

// ColumnsStore handles column data access.
type ColumnsStore struct {
	db *sql.DB
}

// NewColumnsStore creates a new columns store.
func NewColumnsStore(db *sql.DB) *ColumnsStore {
	return &ColumnsStore{db: db}
}

func (s *ColumnsStore) Create(ctx context.Context, c *columns.Column) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO columns (id, project_id, name, position, is_default, is_archived)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, c.ID, c.ProjectID, c.Name, c.Position, c.IsDefault, c.IsArchived)
	return err
}

func (s *ColumnsStore) GetByID(ctx context.Context, id string) (*columns.Column, error) {
	c := &columns.Column{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, position, is_default, is_archived
		FROM columns WHERE id = $1
	`, id).Scan(&c.ID, &c.ProjectID, &c.Name, &c.Position, &c.IsDefault, &c.IsArchived)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (s *ColumnsStore) ListByProject(ctx context.Context, projectID string) ([]*columns.Column, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, name, position, is_default, is_archived
		FROM columns WHERE project_id = $1 AND is_archived = FALSE
		ORDER BY position
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*columns.Column
	for rows.Next() {
		c := &columns.Column{}
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Position, &c.IsDefault, &c.IsArchived); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *ColumnsStore) Update(ctx context.Context, id string, in *columns.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE columns SET name = COALESCE($2, name) WHERE id = $1
	`, id, in.Name)
	return err
}

func (s *ColumnsStore) UpdatePosition(ctx context.Context, id string, position int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE columns SET position = $2 WHERE id = $1
	`, id, position)
	return err
}

func (s *ColumnsStore) SetDefault(ctx context.Context, projectID, columnID string) error {
	// First clear all defaults for this project
	_, err := s.db.ExecContext(ctx, `
		UPDATE columns SET is_default = FALSE WHERE project_id = $1
	`, projectID)
	if err != nil {
		return err
	}
	// Then set the new default
	_, err = s.db.ExecContext(ctx, `
		UPDATE columns SET is_default = TRUE WHERE id = $1
	`, columnID)
	return err
}

func (s *ColumnsStore) Archive(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE columns SET is_archived = TRUE WHERE id = $1
	`, id)
	return err
}

func (s *ColumnsStore) Unarchive(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE columns SET is_archived = FALSE WHERE id = $1
	`, id)
	return err
}

func (s *ColumnsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM columns WHERE id = $1`, id)
	return err
}

func (s *ColumnsStore) GetDefault(ctx context.Context, projectID string) (*columns.Column, error) {
	c := &columns.Column{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, position, is_default, is_archived
		FROM columns WHERE project_id = $1 AND is_default = TRUE
	`, projectID).Scan(&c.ID, &c.ProjectID, &c.Name, &c.Position, &c.IsDefault, &c.IsArchived)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}
