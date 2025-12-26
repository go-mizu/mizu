package duckdb

import (
	"context"
	"database/sql"
)

// AssigneesStore handles issue assignee data access.
type AssigneesStore struct {
	db *sql.DB
}

// NewAssigneesStore creates a new assignees store.
func NewAssigneesStore(db *sql.DB) *AssigneesStore {
	return &AssigneesStore{db: db}
}

func (s *AssigneesStore) Add(ctx context.Context, issueID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_assignees (issue_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (issue_id, user_id) DO NOTHING
	`, issueID, userID)
	return err
}

func (s *AssigneesStore) Remove(ctx context.Context, issueID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM issue_assignees WHERE issue_id = $1 AND user_id = $2
	`, issueID, userID)
	return err
}

func (s *AssigneesStore) List(ctx context.Context, issueID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id FROM issue_assignees WHERE issue_id = $1
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *AssigneesStore) ListByUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT issue_id FROM issue_assignees WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
