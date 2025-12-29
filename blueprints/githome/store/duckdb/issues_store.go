package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/users"
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
	now := time.Now()
	// Only set timestamps if not already set (preserves original timestamps during seeding)
	if i.CreatedAt.IsZero() {
		i.CreatedAt = now
	}
	if i.UpdatedAt.IsZero() {
		i.UpdatedAt = now
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO issues (node_id, repo_id, number, state, state_reason, title, body,
			creator_id, locked, active_lock_reason, comments, closed_at, closed_by_id,
			milestone_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id
	`, "", i.RepoID, i.Number, i.State, i.StateReason, i.Title, i.Body,
		i.CreatorID, i.Locked, i.ActiveLockReason, i.Comments,
		nullTime(i.ClosedAt), nullInt64(0), nullInt64(0), i.CreatedAt, i.UpdatedAt,
	).Scan(&i.ID)
	if err != nil {
		return err
	}

	i.NodeID = generateNodeID("I", i.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE issues SET node_id = $1 WHERE id = $2`, i.NodeID, i.ID)
	return err
}

func (s *IssuesStore) GetByID(ctx context.Context, id int64) (*issues.Issue, error) {
	i := &issues.Issue{}
	var closedAt sql.NullTime
	var closedByID, milestoneID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, number, state, state_reason, title, body,
			creator_id, locked, active_lock_reason, comments, closed_at, closed_by_id,
			milestone_id, created_at, updated_at
		FROM issues WHERE id = $1
	`, id).Scan(&i.ID, &i.NodeID, &i.RepoID, &i.Number, &i.State, &i.StateReason, &i.Title,
		&i.Body, &i.CreatorID, &i.Locked, &i.ActiveLockReason, &i.Comments, &closedAt,
		&closedByID, &milestoneID, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if closedAt.Valid {
		i.ClosedAt = &closedAt.Time
	}
	return i, err
}

func (s *IssuesStore) GetByNumber(ctx context.Context, repoID int64, number int) (*issues.Issue, error) {
	i := &issues.Issue{}
	var closedAt sql.NullTime
	var closedByID, milestoneID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, number, state, state_reason, title, body,
			creator_id, locked, active_lock_reason, comments, closed_at, closed_by_id,
			milestone_id, created_at, updated_at
		FROM issues WHERE repo_id = $1 AND number = $2
	`, repoID, number).Scan(&i.ID, &i.NodeID, &i.RepoID, &i.Number, &i.State, &i.StateReason,
		&i.Title, &i.Body, &i.CreatorID, &i.Locked, &i.ActiveLockReason, &i.Comments,
		&closedAt, &closedByID, &milestoneID, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if closedAt.Valid {
		i.ClosedAt = &closedAt.Time
	}
	return i, err
}

func (s *IssuesStore) Update(ctx context.Context, id int64, in *issues.UpdateIn) error {
	now := time.Now()
	// Set closed_at when state changes to "closed", clear it when state changes to "open"
	var closedAt sql.NullTime
	if in.State != nil {
		if *in.State == "closed" {
			closedAt = sql.NullTime{Time: now, Valid: true}
		}
		// If reopening, closed_at will be set to NULL (Valid: false)
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET
			title = COALESCE($2, title),
			body = COALESCE($3, body),
			state = COALESCE($4, state),
			state_reason = COALESCE($5, state_reason),
			closed_at = CASE WHEN $4 IS NOT NULL THEN $6 ELSE closed_at END,
			updated_at = $7
		WHERE id = $1
	`, id, nullStringPtr(in.Title), nullStringPtr(in.Body), nullStringPtr(in.State),
		nullStringPtr(in.StateReason), closedAt, now)
	return err
}

func (s *IssuesStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issues WHERE id = $1`, id)
	return err
}

func (s *IssuesStore) List(ctx context.Context, repoID int64, opts *issues.ListOpts) ([]*issues.Issue, error) {
	page, perPage := 1, 30
	state := "open"
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
		if opts.State != "" {
			state = opts.State
		}
	}

	query := `
		SELECT id, node_id, repo_id, number, state, state_reason, title, body,
			creator_id, locked, active_lock_reason, comments, closed_at,
			created_at, updated_at
		FROM issues WHERE repo_id = $1`

	args := []any{repoID}
	if state != "all" {
		query += ` AND state = $2`
		args = append(args, state)
	}
	query += ` ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *IssuesStore) ListForOrg(ctx context.Context, orgID int64, opts *issues.ListOpts) ([]*issues.Issue, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT i.id, i.node_id, i.repo_id, i.number, i.state, i.state_reason, i.title, i.body,
			i.creator_id, i.locked, i.active_lock_reason, i.comments, i.closed_at,
			i.created_at, i.updated_at
		FROM issues i
		JOIN repositories r ON r.id = i.repo_id
		WHERE r.owner_id = $1 AND r.owner_type = 'Organization'
		ORDER BY i.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *IssuesStore) ListForUser(ctx context.Context, userID int64, opts *issues.ListOpts) ([]*issues.Issue, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT i.id, i.node_id, i.repo_id, i.number, i.state, i.state_reason, i.title, i.body,
			i.creator_id, i.locked, i.active_lock_reason, i.comments, i.closed_at,
			i.created_at, i.updated_at
		FROM issues i
		LEFT JOIN issue_assignees ia ON ia.issue_id = i.id
		WHERE i.creator_id = $1 OR ia.user_id = $1
		ORDER BY i.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *IssuesStore) NextNumber(ctx context.Context, repoID int64) (int, error) {
	var maxNumber sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(number) FROM issues WHERE repo_id = $1
	`, repoID).Scan(&maxNumber)
	if err != nil {
		return 0, err
	}
	if maxNumber.Valid {
		return int(maxNumber.Int64) + 1, nil
	}
	return 1, nil
}

func (s *IssuesStore) SetLocked(ctx context.Context, id int64, locked bool, reason string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET locked = $2, active_lock_reason = $3, updated_at = $4 WHERE id = $1
	`, id, locked, reason, time.Now())
	return err
}

func (s *IssuesStore) AddAssignee(ctx context.Context, issueID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_assignees (issue_id, user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, issueID, userID)
	return err
}

func (s *IssuesStore) RemoveAssignee(ctx context.Context, issueID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM issue_assignees WHERE issue_id = $1 AND user_id = $2
	`, issueID, userID)
	return err
}

func (s *IssuesStore) ListAssignees(ctx context.Context, issueID int64) ([]*users.SimpleUser, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM users u
		JOIN issue_assignees ia ON ia.user_id = u.id
		WHERE ia.issue_id = $1
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func (s *IssuesStore) AddLabel(ctx context.Context, issueID, labelID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, issueID, labelID)
	return err
}

func (s *IssuesStore) RemoveLabel(ctx context.Context, issueID, labelID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM issue_labels WHERE issue_id = $1 AND label_id = $2
	`, issueID, labelID)
	return err
}

func (s *IssuesStore) ListLabels(ctx context.Context, issueID int64) ([]*issues.Label, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.id, l.node_id, l.name, l.description, l.color, l.is_default
		FROM labels l
		JOIN issue_labels il ON il.label_id = l.id
		WHERE il.issue_id = $1
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []*issues.Label
	for rows.Next() {
		l := &issues.Label{}
		if err := rows.Scan(&l.ID, &l.NodeID, &l.Name, &l.Description, &l.Color, &l.Default); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

func (s *IssuesStore) SetLabels(ctx context.Context, issueID int64, labelIDs []int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issue_labels WHERE issue_id = $1`, issueID)
	if err != nil {
		return err
	}

	for _, labelID := range labelIDs {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2)
		`, issueID, labelID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *IssuesStore) SetMilestone(ctx context.Context, issueID int64, milestoneID *int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET milestone_id = $2, updated_at = $3 WHERE id = $1
	`, issueID, nullInt64Ptr(milestoneID), time.Now())
	return err
}

func (s *IssuesStore) CreateEvent(ctx context.Context, event *issues.IssueEvent) error {
	event.CreatedAt = time.Now()
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO issue_events (node_id, issue_id, actor_id, event, commit_id, commit_url,
			label_id, assignee_id, assigner_id, milestone_id, rename_from, rename_to, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, "", event.ID, event.Actor, event.Event, event.CommitID, event.CommitURL,
		nullInt64(0), nullInt64(0), nullInt64(0), nullInt64(0),
		nullString(""), nullString(""), event.CreatedAt,
	).Scan(&event.ID)
	if err != nil {
		return err
	}

	event.NodeID = generateNodeID("IE", event.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE issue_events SET node_id = $1 WHERE id = $2`, event.NodeID, event.ID)
	return err
}

func (s *IssuesStore) ListEvents(ctx context.Context, issueID int64, opts *issues.ListOpts) ([]*issues.IssueEvent, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT id, node_id, issue_id, actor_id, event, commit_id, commit_url, created_at
		FROM issue_events WHERE issue_id = $1
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*issues.IssueEvent
	for rows.Next() {
		e := &issues.IssueEvent{}
		var actorID int64
		if err := rows.Scan(&e.ID, &e.NodeID, &issueID, &actorID, &e.Event, &e.CommitID, &e.CommitURL, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *IssuesStore) IncrementComments(ctx context.Context, issueID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET comments = comments + $2, updated_at = $3 WHERE id = $1
	`, issueID, delta, time.Now())
	return err
}

// CountByState returns the count of issues for a given state (open, closed, or all).
func (s *IssuesStore) CountByState(ctx context.Context, repoID int64, state string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM issues WHERE repo_id = $1`
	args := []any{repoID}

	if state != "all" && state != "" {
		query += ` AND state = $2`
		args = append(args, state)
	}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// Helper function

func scanIssues(rows *sql.Rows) ([]*issues.Issue, error) {
	var list []*issues.Issue
	for rows.Next() {
		i := &issues.Issue{}
		var closedAt sql.NullTime
		if err := rows.Scan(&i.ID, &i.NodeID, &i.RepoID, &i.Number, &i.State, &i.StateReason,
			&i.Title, &i.Body, &i.CreatorID, &i.Locked, &i.ActiveLockReason, &i.Comments,
			&closedAt, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		if closedAt.Valid {
			i.ClosedAt = &closedAt.Time
		}
		list = append(list, i)
	}
	return list, rows.Err()
}
