package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/watches"
)

// WatchesStore implements watches.Store
type WatchesStore struct {
	db *sql.DB
}

// NewWatchesStore creates a new watches store
func NewWatchesStore(db *sql.DB) *WatchesStore {
	return &WatchesStore{db: db}
}

// Create creates a new watch - uses composite PK (user_id, repo_id)
func (s *WatchesStore) Create(ctx context.Context, w *watches.Watch) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO watches (user_id, repo_id, level, created_at)
		VALUES ($1, $2, $3, $4)
	`, w.UserID, w.RepoID, w.Level, w.CreatedAt)
	return err
}

// Update updates a watch
func (s *WatchesStore) Update(ctx context.Context, w *watches.Watch) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE watches SET level = $3
		WHERE user_id = $1 AND repo_id = $2
	`, w.UserID, w.RepoID, w.Level)
	return err
}

// Delete deletes a watch
func (s *WatchesStore) Delete(ctx context.Context, userID, repoID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM watches WHERE user_id = $1 AND repo_id = $2`, userID, repoID)
	return err
}

// Get retrieves a watch by user ID and repo ID
func (s *WatchesStore) Get(ctx context.Context, userID, repoID string) (*watches.Watch, error) {
	w := &watches.Watch{}
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, repo_id, level, created_at
		FROM watches WHERE user_id = $1 AND repo_id = $2
	`, userID, repoID).Scan(&w.UserID, &w.RepoID, &w.Level, &w.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

// ListByRepo lists watchers for a repository
func (s *WatchesStore) ListByRepo(ctx context.Context, repoID string, limit, offset int) ([]*watches.Watch, int, error) {
	// Get total count (excluding ignoring)
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM watches WHERE repo_id = $1 AND level != 'ignoring'
	`, repoID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get watches
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, repo_id, level, created_at
		FROM watches WHERE repo_id = $1 AND level != 'ignoring'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, repoID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*watches.Watch
	for rows.Next() {
		w := &watches.Watch{}
		if err := rows.Scan(&w.UserID, &w.RepoID, &w.Level, &w.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, w)
	}
	return list, total, rows.Err()
}

// ListByUser lists watches by a user
func (s *WatchesStore) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*watches.Watch, int, error) {
	// Get total count (excluding ignoring)
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM watches WHERE user_id = $1 AND level != 'ignoring'
	`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get watches
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, repo_id, level, created_at
		FROM watches WHERE user_id = $1 AND level != 'ignoring'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*watches.Watch
	for rows.Next() {
		w := &watches.Watch{}
		if err := rows.Scan(&w.UserID, &w.RepoID, &w.Level, &w.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, w)
	}
	return list, total, rows.Err()
}

// Count counts watchers for a repository (excluding ignoring)
func (s *WatchesStore) Count(ctx context.Context, repoID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM watches WHERE repo_id = $1 AND level != 'ignoring'
	`, repoID).Scan(&count)
	return count, err
}
