package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/issues"
)

// IssuesStore implements issues.Store
type IssuesStore struct {
	db *sql.DB
}

// NewIssuesStore creates a new issues store
func NewIssuesStore(db *sql.DB) *IssuesStore {
	return &IssuesStore{db: db}
}

// Create creates a new issue
func (s *IssuesStore) Create(ctx context.Context, i interface{}) error {
	// Type assertion to handle both issues.Issue and anonymous struct from seed
	switch issue := i.(type) {
	case *issues.Issue:
		var closedAt, closedByID interface{}
		if issue.ClosedAt != nil {
			closedAt = *issue.ClosedAt
		}
		if issue.ClosedByID != "" {
			closedByID = issue.ClosedByID
		}
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO issues (id, repo_id, number, title, body, author_id, assignee_id, state, state_reason, is_locked, lock_reason, milestone_id, comment_count, reactions_count, created_at, updated_at, closed_at, closed_by_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		`, issue.ID, issue.RepoID, issue.Number, issue.Title, issue.Body, issue.AuthorID, nullString(issue.AssigneeID), issue.State, issue.StateReason, issue.IsLocked, issue.LockReason, nullString(issue.MilestoneID), issue.CommentCount, issue.ReactionsCount, issue.CreatedAt, issue.UpdatedAt, closedAt, closedByID)
		return err
	default:
		// Handle the seed struct type
		type seedIssue struct {
			ID       string
			RepoID   string
			Number   int
			Title    string
			Body     string
			AuthorID string
			State    string
		}
		if si, ok := i.(*seedIssue); ok {
			now := time.Now()
			_, err := s.db.ExecContext(ctx, `
				INSERT INTO issues (id, repo_id, number, title, body, author_id, state, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`, si.ID, si.RepoID, si.Number, si.Title, si.Body, si.AuthorID, si.State, now, now)
			return err
		}
		return nil
	}
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetByID retrieves an issue by ID
func (s *IssuesStore) GetByID(ctx context.Context, id string) (*issues.Issue, error) {
	i := &issues.Issue{}
	var assigneeID, milestoneID, closedByID sql.NullString
	var closedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, number, title, body, author_id, assignee_id, state, state_reason, is_locked, lock_reason, milestone_id, comment_count, reactions_count, created_at, updated_at, closed_at, closed_by_id
		FROM issues WHERE id = $1
	`, id).Scan(&i.ID, &i.RepoID, &i.Number, &i.Title, &i.Body, &i.AuthorID, &assigneeID, &i.State, &i.StateReason, &i.IsLocked, &i.LockReason, &milestoneID, &i.CommentCount, &i.ReactionsCount, &i.CreatedAt, &i.UpdatedAt, &closedAt, &closedByID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if assigneeID.Valid {
		i.AssigneeID = assigneeID.String
	}
	if milestoneID.Valid {
		i.MilestoneID = milestoneID.String
	}
	if closedAt.Valid {
		i.ClosedAt = &closedAt.Time
	}
	if closedByID.Valid {
		i.ClosedByID = closedByID.String
	}
	return i, nil
}

// GetByNumber retrieves an issue by repo ID and number
func (s *IssuesStore) GetByNumber(ctx context.Context, repoID string, number int) (*issues.Issue, error) {
	i := &issues.Issue{}
	var assigneeID, milestoneID, closedByID sql.NullString
	var closedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, number, title, body, author_id, assignee_id, state, state_reason, is_locked, lock_reason, milestone_id, comment_count, reactions_count, created_at, updated_at, closed_at, closed_by_id
		FROM issues WHERE repo_id = $1 AND number = $2
	`, repoID, number).Scan(&i.ID, &i.RepoID, &i.Number, &i.Title, &i.Body, &i.AuthorID, &assigneeID, &i.State, &i.StateReason, &i.IsLocked, &i.LockReason, &milestoneID, &i.CommentCount, &i.ReactionsCount, &i.CreatedAt, &i.UpdatedAt, &closedAt, &closedByID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if assigneeID.Valid {
		i.AssigneeID = assigneeID.String
	}
	if milestoneID.Valid {
		i.MilestoneID = milestoneID.String
	}
	if closedAt.Valid {
		i.ClosedAt = &closedAt.Time
	}
	if closedByID.Valid {
		i.ClosedByID = closedByID.String
	}
	return i, nil
}

// Update updates an issue
func (s *IssuesStore) Update(ctx context.Context, i *issues.Issue) error {
	i.UpdatedAt = time.Now()
	var closedAt, closedByID interface{}
	if i.ClosedAt != nil {
		closedAt = *i.ClosedAt
	}
	if i.ClosedByID != "" {
		closedByID = i.ClosedByID
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET title = $2, body = $3, assignee_id = $4, state = $5, state_reason = $6, is_locked = $7, lock_reason = $8, milestone_id = $9, comment_count = $10, reactions_count = $11, updated_at = $12, closed_at = $13, closed_by_id = $14
		WHERE id = $1
	`, i.ID, i.Title, i.Body, nullString(i.AssigneeID), i.State, i.StateReason, i.IsLocked, i.LockReason, nullString(i.MilestoneID), i.CommentCount, i.ReactionsCount, i.UpdatedAt, closedAt, closedByID)
	return err
}

// Delete deletes an issue
func (s *IssuesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issues WHERE id = $1`, id)
	return err
}

// List lists issues for a repository
func (s *IssuesStore) List(ctx context.Context, repoID string, state string, limit, offset int) ([]*issues.Issue, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM issues WHERE repo_id = $1`
	args := []interface{}{repoID}
	if state != "" && state != "all" {
		countQuery += ` AND state = $2`
		args = append(args, state)
	}
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get issues
	query := `
		SELECT id, repo_id, number, title, body, author_id, assignee_id, state, state_reason, is_locked, lock_reason, milestone_id, comment_count, reactions_count, created_at, updated_at, closed_at, closed_by_id
		FROM issues WHERE repo_id = $1`
	if state != "" && state != "all" {
		query += ` AND state = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = append(args, limit, offset)
	} else {
		query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*issues.Issue
	for rows.Next() {
		i := &issues.Issue{}
		var assigneeID, milestoneID, closedByID sql.NullString
		var closedAt sql.NullTime
		if err := rows.Scan(&i.ID, &i.RepoID, &i.Number, &i.Title, &i.Body, &i.AuthorID, &assigneeID, &i.State, &i.StateReason, &i.IsLocked, &i.LockReason, &milestoneID, &i.CommentCount, &i.ReactionsCount, &i.CreatedAt, &i.UpdatedAt, &closedAt, &closedByID); err != nil {
			return nil, 0, err
		}
		if assigneeID.Valid {
			i.AssigneeID = assigneeID.String
		}
		if milestoneID.Valid {
			i.MilestoneID = milestoneID.String
		}
		if closedAt.Valid {
			i.ClosedAt = &closedAt.Time
		}
		if closedByID.Valid {
			i.ClosedByID = closedByID.String
		}
		list = append(list, i)
	}
	return list, total, rows.Err()
}

// GetNextNumber gets the next issue number for a repository
func (s *IssuesStore) GetNextNumber(ctx context.Context, repoID string) (int, error) {
	var maxNum sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(number) FROM issues WHERE repo_id = $1
	`, repoID).Scan(&maxNum)
	if err != nil {
		return 1, err
	}
	if !maxNum.Valid {
		return 1, nil
	}
	return int(maxNum.Int64) + 1, nil
}

// AddLabel adds a label to an issue
func (s *IssuesStore) AddLabel(ctx context.Context, issueLabel *issues.IssueLabel) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_labels (id, issue_id, label_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, issueLabel.ID, issueLabel.IssueID, issueLabel.LabelID, issueLabel.CreatedAt)
	return err
}

// RemoveLabel removes a label from an issue
func (s *IssuesStore) RemoveLabel(ctx context.Context, issueID, labelID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issue_labels WHERE issue_id = $1 AND label_id = $2`, issueID, labelID)
	return err
}

// ListLabels lists labels for an issue
func (s *IssuesStore) ListLabels(ctx context.Context, issueID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT label_id FROM issue_labels WHERE issue_id = $1
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []string
	for rows.Next() {
		var labelID string
		if err := rows.Scan(&labelID); err != nil {
			return nil, err
		}
		labels = append(labels, labelID)
	}
	return labels, rows.Err()
}

// AddAssignee adds an assignee to an issue
func (s *IssuesStore) AddAssignee(ctx context.Context, issueAssignee *issues.IssueAssignee) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_assignees (id, issue_id, user_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, issueAssignee.ID, issueAssignee.IssueID, issueAssignee.UserID, issueAssignee.CreatedAt)
	return err
}

// RemoveAssignee removes an assignee from an issue
func (s *IssuesStore) RemoveAssignee(ctx context.Context, issueID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issue_assignees WHERE issue_id = $1 AND user_id = $2`, issueID, userID)
	return err
}

// ListAssignees lists assignees for an issue
func (s *IssuesStore) ListAssignees(ctx context.Context, issueID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id FROM issue_assignees WHERE issue_id = $1
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignees []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		assignees = append(assignees, userID)
	}
	return assignees, rows.Err()
}
