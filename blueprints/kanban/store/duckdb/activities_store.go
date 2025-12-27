package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/kanban/feature/activities"
)

// ActivitiesStore handles activity data access.
type ActivitiesStore struct {
	db *sql.DB
}

// NewActivitiesStore creates a new activities store.
func NewActivitiesStore(db *sql.DB) *ActivitiesStore {
	return &ActivitiesStore{db: db}
}

func (s *ActivitiesStore) Create(ctx context.Context, a *activities.Activity) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO activities (id, issue_id, actor_id, action, old_value, new_value, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, a.ID, a.IssueID, a.ActorID, a.Action, nullString(a.OldValue), nullString(a.NewValue), nullString(a.Metadata), a.CreatedAt)
	return err
}

func (s *ActivitiesStore) GetByID(ctx context.Context, id string) (*activities.Activity, error) {
	a := &activities.Activity{}
	var oldValue, newValue, metadata sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, issue_id, actor_id, action, old_value, new_value, metadata, created_at
		FROM activities WHERE id = $1
	`, id).Scan(&a.ID, &a.IssueID, &a.ActorID, &a.Action, &oldValue, &newValue, &metadata, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if oldValue.Valid {
		a.OldValue = oldValue.String
	}
	if newValue.Valid {
		a.NewValue = newValue.String
	}
	if metadata.Valid {
		a.Metadata = metadata.String
	}
	return a, err
}

func (s *ActivitiesStore) ListByIssue(ctx context.Context, issueID string) ([]*activities.Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, issue_id, actor_id, action, old_value, new_value, metadata, created_at
		FROM activities WHERE issue_id = $1
		ORDER BY created_at ASC
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*activities.Activity
	for rows.Next() {
		a := &activities.Activity{}
		var oldValue, newValue, metadata sql.NullString
		if err := rows.Scan(&a.ID, &a.IssueID, &a.ActorID, &a.Action, &oldValue, &newValue, &metadata, &a.CreatedAt); err != nil {
			return nil, err
		}
		if oldValue.Valid {
			a.OldValue = oldValue.String
		}
		if newValue.Valid {
			a.NewValue = newValue.String
		}
		if metadata.Valid {
			a.Metadata = metadata.String
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (s *ActivitiesStore) ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*activities.ActivityWithContext, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.issue_id, a.actor_id, a.action, a.old_value, a.new_value, a.metadata, a.created_at,
		       u.display_name, i.key
		FROM activities a
		JOIN issues i ON a.issue_id = i.id
		JOIN projects p ON i.project_id = p.id
		JOIN teams t ON p.team_id = t.id
		JOIN users u ON a.actor_id = u.id
		WHERE t.workspace_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*activities.ActivityWithContext
	for rows.Next() {
		a := &activities.Activity{}
		ac := &activities.ActivityWithContext{Activity: a}
		var oldValue, newValue, metadata sql.NullString
		if err := rows.Scan(&a.ID, &a.IssueID, &a.ActorID, &a.Action, &oldValue, &newValue, &metadata, &a.CreatedAt, &ac.ActorName, &ac.IssueKey); err != nil {
			return nil, err
		}
		if oldValue.Valid {
			a.OldValue = oldValue.String
		}
		if newValue.Valid {
			a.NewValue = newValue.String
		}
		if metadata.Valid {
			a.Metadata = metadata.String
		}
		list = append(list, ac)
	}
	return list, rows.Err()
}

func (s *ActivitiesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM activities WHERE id = $1`, id)
	return err
}

func (s *ActivitiesStore) CountByIssue(ctx context.Context, issueID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM activities WHERE issue_id = $1
	`, issueID).Scan(&count)
	return count, err
}
