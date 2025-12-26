package duckdb

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/issues"
)

// IssuesStore handles issue data access.
type IssuesStore struct {
	db *sql.DB
}

// NewIssuesStore creates a new issues store.
func NewIssuesStore(db *sql.DB) *IssuesStore {
	return &IssuesStore{db: db}
}

func (s *IssuesStore) Create(ctx context.Context, i *issues.Issue) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issues (id, project_id, number, key, title, column_id, position, creator_id, cycle_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, i.ID, i.ProjectID, i.Number, i.Key, i.Title, i.ColumnID, i.Position, i.CreatorID, nullString(i.CycleID), i.CreatedAt, i.UpdatedAt)
	return err
}

func (s *IssuesStore) GetByID(ctx context.Context, id string) (*issues.Issue, error) {
	i := &issues.Issue{}
	var cycleID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, number, key, title, column_id, position, creator_id, cycle_id, created_at, updated_at
		FROM issues WHERE id = $1
	`, id).Scan(&i.ID, &i.ProjectID, &i.Number, &i.Key, &i.Title, &i.ColumnID, &i.Position, &i.CreatorID, &cycleID, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if cycleID.Valid {
		i.CycleID = cycleID.String
	}
	return i, err
}

func (s *IssuesStore) GetByKey(ctx context.Context, key string) (*issues.Issue, error) {
	i := &issues.Issue{}
	var cycleID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, number, key, title, column_id, position, creator_id, cycle_id, created_at, updated_at
		FROM issues WHERE key = $1
	`, key).Scan(&i.ID, &i.ProjectID, &i.Number, &i.Key, &i.Title, &i.ColumnID, &i.Position, &i.CreatorID, &cycleID, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if cycleID.Valid {
		i.CycleID = cycleID.String
	}
	return i, err
}

func (s *IssuesStore) ListByProject(ctx context.Context, projectID string) ([]*issues.Issue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, number, key, title, column_id, position, creator_id, cycle_id, created_at, updated_at
		FROM issues WHERE project_id = $1
		ORDER BY position, created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *IssuesStore) ListByColumn(ctx context.Context, columnID string) ([]*issues.Issue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, number, key, title, column_id, position, creator_id, cycle_id, created_at, updated_at
		FROM issues WHERE column_id = $1
		ORDER BY position, created_at DESC
	`, columnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *IssuesStore) ListByCycle(ctx context.Context, cycleID string) ([]*issues.Issue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, number, key, title, column_id, position, creator_id, cycle_id, created_at, updated_at
		FROM issues WHERE cycle_id = $1
		ORDER BY position, created_at DESC
	`, cycleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *IssuesStore) Update(ctx context.Context, id string, in *issues.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET
			title = COALESCE($2, title),
			updated_at = $3
		WHERE id = $1
	`, id, in.Title, time.Now())
	return err
}

func (s *IssuesStore) Move(ctx context.Context, id, columnID string, position int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET column_id = $2, position = $3, updated_at = $4
		WHERE id = $1
	`, id, columnID, position, time.Now())
	return err
}

func (s *IssuesStore) AttachCycle(ctx context.Context, id, cycleID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET cycle_id = $2, updated_at = $3
		WHERE id = $1
	`, id, cycleID, time.Now())
	return err
}

func (s *IssuesStore) DetachCycle(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET cycle_id = NULL, updated_at = $2
		WHERE id = $1
	`, id, time.Now())
	return err
}

func (s *IssuesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issues WHERE id = $1`, id)
	return err
}

func (s *IssuesStore) Search(ctx context.Context, projectID, query string, limit int) ([]*issues.Issue, error) {
	searchQuery := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, number, key, title, column_id, position, creator_id, cycle_id, created_at, updated_at
		FROM issues
		WHERE project_id = $1 AND (LOWER(title) LIKE $2 OR LOWER(key) LIKE $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, projectID, searchQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

// Helper functions

func scanIssues(rows *sql.Rows) ([]*issues.Issue, error) {
	var list []*issues.Issue
	for rows.Next() {
		i := &issues.Issue{}
		var cycleID sql.NullString
		if err := rows.Scan(&i.ID, &i.ProjectID, &i.Number, &i.Key, &i.Title, &i.ColumnID, &i.Position, &i.CreatorID, &cycleID, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		if cycleID.Valid {
			i.CycleID = cycleID.String
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
