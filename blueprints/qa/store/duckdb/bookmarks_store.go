package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/qa/feature/bookmarks"
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
		INSERT INTO bookmarks (id, account_id, question_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_id, question_id) DO NOTHING
	`, bookmark.ID, bookmark.AccountID, bookmark.QuestionID, bookmark.CreatedAt)
	return err
}

// Delete removes a bookmark.
func (s *BookmarksStore) Delete(ctx context.Context, accountID, questionID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM bookmarks WHERE account_id = $1 AND question_id = $2
	`, accountID, questionID)
	return err
}

// ListByAccount lists bookmarks.
func (s *BookmarksStore) ListByAccount(ctx context.Context, accountID string, limit int) ([]*bookmarks.Bookmark, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, question_id, created_at
		FROM bookmarks
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*bookmarks.Bookmark
	for rows.Next() {
		bm := &bookmarks.Bookmark{}
		if err := rows.Scan(&bm.ID, &bm.AccountID, &bm.QuestionID, &bm.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, bm)
	}
	return result, rows.Err()
}
