package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/stars"
)

// StarsStore implements stars.Store
type StarsStore struct {
	db *sql.DB
}

// NewStarsStore creates a new stars store
func NewStarsStore(db *sql.DB) *StarsStore {
	return &StarsStore{db: db}
}

// Create creates a new star - uses composite PK (user_id, repo_id)
func (s *StarsStore) Create(ctx context.Context, star *stars.Star) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stars (user_id, repo_id, created_at)
		VALUES ($1, $2, $3)
	`, star.UserID, star.RepoID, star.CreatedAt)
	return err
}

// Delete deletes a star
func (s *StarsStore) Delete(ctx context.Context, userID, repoID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM stars WHERE user_id = $1 AND repo_id = $2`, userID, repoID)
	return err
}

// Get retrieves a star by user ID and repo ID
func (s *StarsStore) Get(ctx context.Context, userID, repoID string) (*stars.Star, error) {
	star := &stars.Star{}
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, repo_id, created_at
		FROM stars WHERE user_id = $1 AND repo_id = $2
	`, userID, repoID).Scan(&star.UserID, &star.RepoID, &star.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return star, err
}

// ListByRepo lists stars for a repository
func (s *StarsStore) ListByRepo(ctx context.Context, repoID string, limit, offset int) ([]*stars.Star, int, error) {
	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM stars WHERE repo_id = $1
	`, repoID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get stars
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, repo_id, created_at
		FROM stars WHERE repo_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, repoID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*stars.Star
	for rows.Next() {
		star := &stars.Star{}
		if err := rows.Scan(&star.UserID, &star.RepoID, &star.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, star)
	}
	return list, total, rows.Err()
}

// ListByUser lists stars by a user
func (s *StarsStore) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*stars.Star, int, error) {
	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM stars WHERE user_id = $1
	`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get stars
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, repo_id, created_at
		FROM stars WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*stars.Star
	for rows.Next() {
		star := &stars.Star{}
		if err := rows.Scan(&star.UserID, &star.RepoID, &star.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, star)
	}
	return list, total, rows.Err()
}

// Count counts stars for a repository
func (s *StarsStore) Count(ctx context.Context, repoID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM stars WHERE repo_id = $1
	`, repoID).Scan(&count)
	return count, err
}
