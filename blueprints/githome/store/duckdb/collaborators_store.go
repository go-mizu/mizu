package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/collaborators"
)

// CollaboratorsStore implements collaborators.Store
type CollaboratorsStore struct {
	db *sql.DB
}

// NewCollaboratorsStore creates a new collaborators store
func NewCollaboratorsStore(db *sql.DB) *CollaboratorsStore {
	return &CollaboratorsStore{db: db}
}

// Create creates a new collaborator - uses composite PK (repo_id, user_id)
func (s *CollaboratorsStore) Create(ctx context.Context, c *collaborators.Collaborator) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collaborators (repo_id, user_id, permission, created_at)
		VALUES ($1, $2, $3, $4)
	`, c.RepoID, c.UserID, c.Permission, c.CreatedAt)
	return err
}

// Delete deletes a collaborator
func (s *CollaboratorsStore) Delete(ctx context.Context, repoID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collaborators WHERE repo_id = $1 AND user_id = $2`, repoID, userID)
	return err
}

// Update updates a collaborator
func (s *CollaboratorsStore) Update(ctx context.Context, c *collaborators.Collaborator) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE collaborators SET permission = $3
		WHERE repo_id = $1 AND user_id = $2
	`, c.RepoID, c.UserID, c.Permission)
	return err
}

// Get retrieves a collaborator
func (s *CollaboratorsStore) Get(ctx context.Context, repoID, userID string) (*collaborators.Collaborator, error) {
	c := &collaborators.Collaborator{}
	err := s.db.QueryRowContext(ctx, `
		SELECT repo_id, user_id, permission, created_at
		FROM collaborators WHERE repo_id = $1 AND user_id = $2
	`, repoID, userID).Scan(&c.RepoID, &c.UserID, &c.Permission, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// List lists collaborators for a repository
func (s *CollaboratorsStore) List(ctx context.Context, repoID string, limit, offset int) ([]*collaborators.Collaborator, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repo_id, user_id, permission, created_at
		FROM collaborators WHERE repo_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, repoID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*collaborators.Collaborator
	for rows.Next() {
		c := &collaborators.Collaborator{}
		if err := rows.Scan(&c.RepoID, &c.UserID, &c.Permission, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// ListByUser lists repository IDs a user is a collaborator on
func (s *CollaboratorsStore) ListByUser(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repo_id
		FROM collaborators WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var repoID string
		if err := rows.Scan(&repoID); err != nil {
			return nil, err
		}
		list = append(list, repoID)
	}
	return list, rows.Err()
}
