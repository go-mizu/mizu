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
		INSERT INTO issues (id, project_id, number, key, title, description, type, status, priority, parent_id, creator_id, sprint_id, due_date, estimate, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, i.ID, i.ProjectID, i.Number, i.Key, i.Title, i.Description, i.Type, i.Status, i.Priority, i.ParentID, i.CreatorID, i.SprintID, i.DueDate, i.Estimate, i.Position, i.CreatedAt, i.UpdatedAt)
	return err
}

func (s *IssuesStore) GetByID(ctx context.Context, id string) (*issues.Issue, error) {
	i := &issues.Issue{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, number, key, title, description, type, status, priority, parent_id, creator_id, sprint_id, due_date, estimate, position, created_at, updated_at
		FROM issues WHERE id = $1
	`, id).Scan(&i.ID, &i.ProjectID, &i.Number, &i.Key, &i.Title, &i.Description, &i.Type, &i.Status, &i.Priority, &i.ParentID, &i.CreatorID, &i.SprintID, &i.DueDate, &i.Estimate, &i.Position, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return i, err
}

func (s *IssuesStore) GetByKey(ctx context.Context, key string) (*issues.Issue, error) {
	i := &issues.Issue{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, number, key, title, description, type, status, priority, parent_id, creator_id, sprint_id, due_date, estimate, position, created_at, updated_at
		FROM issues WHERE key = $1
	`, key).Scan(&i.ID, &i.ProjectID, &i.Number, &i.Key, &i.Title, &i.Description, &i.Type, &i.Status, &i.Priority, &i.ParentID, &i.CreatorID, &i.SprintID, &i.DueDate, &i.Estimate, &i.Position, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return i, err
}

func (s *IssuesStore) ListByProject(ctx context.Context, projectID string, filter *issues.Filter) ([]*issues.Issue, error) {
	query := `
		SELECT id, project_id, number, key, title, description, type, status, priority, parent_id, creator_id, sprint_id, due_date, estimate, position, created_at, updated_at
		FROM issues WHERE project_id = $1
	`
	args := []any{projectID}
	argIdx := 2

	if filter != nil {
		if filter.Status != "" {
			query += " AND status = $" + itoa(argIdx)
			args = append(args, filter.Status)
			argIdx++
		}
		if filter.Priority != "" {
			query += " AND priority = $" + itoa(argIdx)
			args = append(args, filter.Priority)
			argIdx++
		}
		if filter.Type != "" {
			query += " AND type = $" + itoa(argIdx)
			args = append(args, filter.Type)
			argIdx++
		}
		if filter.AssigneeID != "" {
			query += " AND id IN (SELECT issue_id FROM issue_assignees WHERE user_id = $" + itoa(argIdx) + ")"
			args = append(args, filter.AssigneeID)
			argIdx++
		}
		if filter.SprintID != "" {
			query += " AND sprint_id = $" + itoa(argIdx)
			args = append(args, filter.SprintID)
			argIdx++
		}
		if filter.ParentID != "" {
			query += " AND parent_id = $" + itoa(argIdx)
			args = append(args, filter.ParentID)
			argIdx++
		}
	}

	query += " ORDER BY position, created_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += " LIMIT " + itoa(filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*issues.Issue
	for rows.Next() {
		i := &issues.Issue{}
		if err := rows.Scan(&i.ID, &i.ProjectID, &i.Number, &i.Key, &i.Title, &i.Description, &i.Type, &i.Status, &i.Priority, &i.ParentID, &i.CreatorID, &i.SprintID, &i.DueDate, &i.Estimate, &i.Position, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

func (s *IssuesStore) ListByStatus(ctx context.Context, projectID string) (map[string][]*issues.Issue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, number, key, title, description, type, status, priority, parent_id, creator_id, sprint_id, due_date, estimate, position, created_at, updated_at
		FROM issues WHERE project_id = $1
		ORDER BY status, position, created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]*issues.Issue)
	for rows.Next() {
		i := &issues.Issue{}
		if err := rows.Scan(&i.ID, &i.ProjectID, &i.Number, &i.Key, &i.Title, &i.Description, &i.Type, &i.Status, &i.Priority, &i.ParentID, &i.CreatorID, &i.SprintID, &i.DueDate, &i.Estimate, &i.Position, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		result[i.Status] = append(result[i.Status], i)
	}
	return result, rows.Err()
}

func (s *IssuesStore) Update(ctx context.Context, id string, in *issues.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			type = COALESCE($4, type),
			status = COALESCE($5, status),
			priority = COALESCE($6, priority),
			parent_id = COALESCE($7, parent_id),
			sprint_id = COALESCE($8, sprint_id),
			due_date = COALESCE($9, due_date),
			estimate = COALESCE($10, estimate),
			updated_at = $11
		WHERE id = $1
	`, id, in.Title, in.Description, in.Type, in.Status, in.Priority, in.ParentID, in.SprintID, in.DueDate, in.Estimate, time.Now())
	return err
}

func (s *IssuesStore) UpdatePosition(ctx context.Context, id string, status string, position int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET status = $2, position = $3, updated_at = $4
		WHERE id = $1
	`, id, status, position, time.Now())
	return err
}

func (s *IssuesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issues WHERE id = $1`, id)
	return err
}

// Assignee operations

func (s *IssuesStore) AddAssignee(ctx context.Context, issueID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_assignees (issue_id, user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, issueID, userID)
	return err
}

func (s *IssuesStore) RemoveAssignee(ctx context.Context, issueID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM issue_assignees WHERE issue_id = $1 AND user_id = $2
	`, issueID, userID)
	return err
}

func (s *IssuesStore) GetAssignees(ctx context.Context, issueID string) ([]string, error) {
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

// Label operations

func (s *IssuesStore) AddLabel(ctx context.Context, issueID, labelID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, issueID, labelID)
	return err
}

func (s *IssuesStore) RemoveLabel(ctx context.Context, issueID, labelID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM issue_labels WHERE issue_id = $1 AND label_id = $2
	`, issueID, labelID)
	return err
}

func (s *IssuesStore) GetLabels(ctx context.Context, issueID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT label_id FROM issue_labels WHERE issue_id = $1
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

// Search issues
func (s *IssuesStore) Search(ctx context.Context, projectID, query string, limit int) ([]*issues.Issue, error) {
	searchQuery := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, number, key, title, description, type, status, priority, parent_id, creator_id, sprint_id, due_date, estimate, position, created_at, updated_at
		FROM issues
		WHERE project_id = $1 AND (LOWER(title) LIKE $2 OR LOWER(key) LIKE $2 OR LOWER(description) LIKE $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, projectID, searchQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*issues.Issue
	for rows.Next() {
		i := &issues.Issue{}
		if err := rows.Scan(&i.ID, &i.ProjectID, &i.Number, &i.Key, &i.Title, &i.Description, &i.Type, &i.Status, &i.Priority, &i.ParentID, &i.CreatorID, &i.SprintID, &i.DueDate, &i.Estimate, &i.Position, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
