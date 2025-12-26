package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/projects"
)

// ProjectsStore handles project data access.
type ProjectsStore struct {
	db *sql.DB
}

// NewProjectsStore creates a new projects store.
func NewProjectsStore(db *sql.DB) *ProjectsStore {
	return &ProjectsStore{db: db}
}

func (s *ProjectsStore) Create(ctx context.Context, p *projects.Project) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO projects (id, workspace_id, key, name, description, color, lead_id, status, issue_counter, start_date, target_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, p.ID, p.WorkspaceID, p.Key, p.Name, p.Description, p.Color, p.LeadID, p.Status, p.IssueCounter, p.StartDate, p.TargetDate, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *ProjectsStore) GetByID(ctx context.Context, id string) (*projects.Project, error) {
	p := &projects.Project{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, key, name, description, color, lead_id, status, issue_counter, start_date, target_date, created_at, updated_at
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.WorkspaceID, &p.Key, &p.Name, &p.Description, &p.Color, &p.LeadID, &p.Status, &p.IssueCounter, &p.StartDate, &p.TargetDate, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *ProjectsStore) GetByKey(ctx context.Context, workspaceID, key string) (*projects.Project, error) {
	p := &projects.Project{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, key, name, description, color, lead_id, status, issue_counter, start_date, target_date, created_at, updated_at
		FROM projects WHERE workspace_id = $1 AND key = $2
	`, workspaceID, key).Scan(&p.ID, &p.WorkspaceID, &p.Key, &p.Name, &p.Description, &p.Color, &p.LeadID, &p.Status, &p.IssueCounter, &p.StartDate, &p.TargetDate, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *ProjectsStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*projects.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, key, name, description, color, lead_id, status, issue_counter, start_date, target_date, created_at, updated_at
		FROM projects WHERE workspace_id = $1 AND status != 'archived'
		ORDER BY name
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*projects.Project
	for rows.Next() {
		p := &projects.Project{}
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Key, &p.Name, &p.Description, &p.Color, &p.LeadID, &p.Status, &p.IssueCounter, &p.StartDate, &p.TargetDate, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (s *ProjectsStore) Update(ctx context.Context, id string, in *projects.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE projects SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			color = COALESCE($4, color),
			lead_id = COALESCE($5, lead_id),
			status = COALESCE($6, status),
			start_date = COALESCE($7, start_date),
			target_date = COALESCE($8, target_date),
			updated_at = $9
		WHERE id = $1
	`, id, in.Name, in.Description, in.Color, in.LeadID, in.Status, in.StartDate, in.TargetDate, time.Now())
	return err
}

func (s *ProjectsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

func (s *ProjectsStore) IncrementIssueCounter(ctx context.Context, id string) (int, error) {
	var counter int
	err := s.db.QueryRowContext(ctx, `
		UPDATE projects SET issue_counter = issue_counter + 1
		WHERE id = $1
		RETURNING issue_counter
	`, id).Scan(&counter)
	return counter, err
}

// Stats returns project statistics
func (s *ProjectsStore) GetStats(ctx context.Context, id string) (*projects.Stats, error) {
	stats := &projects.Stats{}
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'done' THEN 1 END) as done,
			COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress,
			COUNT(CASE WHEN status = 'backlog' THEN 1 END) as backlog
		FROM issues WHERE project_id = $1
	`, id).Scan(&stats.TotalIssues, &stats.DoneIssues, &stats.InProgressIssues, &stats.BacklogIssues)
	return stats, err
}
