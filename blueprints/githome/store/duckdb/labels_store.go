package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/githome/feature/labels"
)

// LabelsStore implements labels.Store
type LabelsStore struct {
	db *sql.DB
}

// NewLabelsStore creates a new labels store
func NewLabelsStore(db *sql.DB) *LabelsStore {
	return &LabelsStore{db: db}
}

// Create creates a new label
func (s *LabelsStore) Create(ctx context.Context, l *labels.Label) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO labels (id, repo_id, name, color, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, l.ID, l.RepoID, l.Name, l.Color, l.Description, l.CreatedAt)
	return err
}

// GetByID retrieves a label by ID
func (s *LabelsStore) GetByID(ctx context.Context, id string) (*labels.Label, error) {
	l := &labels.Label{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, name, color, description, created_at
		FROM labels WHERE id = $1
	`, id).Scan(&l.ID, &l.RepoID, &l.Name, &l.Color, &l.Description, &l.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return l, err
}

// GetByName retrieves a label by repository ID and name
func (s *LabelsStore) GetByName(ctx context.Context, repoID, name string) (*labels.Label, error) {
	l := &labels.Label{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, name, color, description, created_at
		FROM labels WHERE repo_id = $1 AND name = $2
	`, repoID, name).Scan(&l.ID, &l.RepoID, &l.Name, &l.Color, &l.Description, &l.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return l, err
}

// Update updates a label
func (s *LabelsStore) Update(ctx context.Context, l *labels.Label) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE labels SET name = $2, color = $3, description = $4
		WHERE id = $1
	`, l.ID, l.Name, l.Color, l.Description)
	return err
}

// Delete deletes a label
func (s *LabelsStore) Delete(ctx context.Context, id string) error {
	// Also delete from issue_labels
	s.db.ExecContext(ctx, `DELETE FROM issue_labels WHERE label_id = $1`, id)
	s.db.ExecContext(ctx, `DELETE FROM pr_labels WHERE label_id = $1`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM labels WHERE id = $1`, id)
	return err
}

// List lists all labels for a repository
func (s *LabelsStore) List(ctx context.Context, repoID string) ([]*labels.Label, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, name, color, description, created_at
		FROM labels WHERE repo_id = $1 ORDER BY name ASC
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*labels.Label
	for rows.Next() {
		l := &labels.Label{}
		if err := rows.Scan(&l.ID, &l.RepoID, &l.Name, &l.Color, &l.Description, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// ListByIDs lists labels by their IDs
func (s *LabelsStore) ListByIDs(ctx context.Context, ids []string) ([]*labels.Label, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := `
		SELECT id, repo_id, name, color, description, created_at
		FROM labels WHERE id IN (` + strings.Join(placeholders, ",") + `) ORDER BY name ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*labels.Label
	for rows.Next() {
		l := &labels.Label{}
		if err := rows.Scan(&l.ID, &l.RepoID, &l.Name, &l.Color, &l.Description, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}
