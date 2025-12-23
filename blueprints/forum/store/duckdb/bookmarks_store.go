package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/forum/feature/bookmarks"
)

// BookmarksStore implements bookmarks.Store.
type BookmarksStore struct {
	db *sql.DB
}

// NewBookmarksStore creates a new bookmarks store.
func NewBookmarksStore(db *sql.DB) *BookmarksStore {
	return &BookmarksStore{db: db}
}

// Create creates a bookmark.
func (s *BookmarksStore) Create(ctx context.Context, bookmark *bookmarks.Bookmark) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bookmarks (id, account_id, target_type, target_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, bookmark.ID, bookmark.AccountID, bookmark.TargetType, bookmark.TargetID, bookmark.CreatedAt)
	return err
}

// GetByTarget retrieves a bookmark by target.
func (s *BookmarksStore) GetByTarget(ctx context.Context, accountID, targetType, targetID string) (*bookmarks.Bookmark, error) {
	bookmark := &bookmarks.Bookmark{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, target_type, target_id, created_at
		FROM bookmarks WHERE account_id = $1 AND target_type = $2 AND target_id = $3
	`, accountID, targetType, targetID).Scan(
		&bookmark.ID, &bookmark.AccountID, &bookmark.TargetType, &bookmark.TargetID, &bookmark.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, bookmarks.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return bookmark, nil
}

// Delete deletes a bookmark.
func (s *BookmarksStore) Delete(ctx context.Context, accountID, targetType, targetID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM bookmarks WHERE account_id = $1 AND target_type = $2 AND target_id = $3
	`, accountID, targetType, targetID)
	return err
}

// List lists bookmarks.
func (s *BookmarksStore) List(ctx context.Context, accountID, targetType string, opts bookmarks.ListOpts) ([]*bookmarks.Bookmark, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, target_type, target_id, created_at
		FROM bookmarks WHERE account_id = $1 AND target_type = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, accountID, targetType, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*bookmarks.Bookmark
	for rows.Next() {
		bookmark := &bookmarks.Bookmark{}
		err := rows.Scan(&bookmark.ID, &bookmark.AccountID, &bookmark.TargetType, &bookmark.TargetID, &bookmark.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, bookmark)
	}
	return result, rows.Err()
}

// GetByTargets retrieves bookmarks for multiple targets.
func (s *BookmarksStore) GetByTargets(ctx context.Context, accountID, targetType string, targetIDs []string) ([]*bookmarks.Bookmark, error) {
	if len(targetIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, account_id, target_type, target_id, created_at
		FROM bookmarks WHERE account_id = $1 AND target_type = $2 AND target_id IN (`

	args := []any{accountID, targetType}
	for i, id := range targetIDs {
		if i > 0 {
			query += ", "
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*bookmarks.Bookmark
	for rows.Next() {
		bookmark := &bookmarks.Bookmark{}
		err := rows.Scan(&bookmark.ID, &bookmark.AccountID, &bookmark.TargetType, &bookmark.TargetID, &bookmark.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, bookmark)
	}
	return result, rows.Err()
}
