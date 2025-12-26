package duckdb

import (
	"context"
	"database/sql"

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
		INSERT INTO projects (id, team_id, key, name, issue_counter)
		VALUES ($1, $2, $3, $4, $5)
	`, p.ID, p.TeamID, p.Key, p.Name, p.IssueCounter)
	return err
}

func (s *ProjectsStore) GetByID(ctx context.Context, id string) (*projects.Project, error) {
	p := &projects.Project{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, key, name, issue_counter
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.TeamID, &p.Key, &p.Name, &p.IssueCounter)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *ProjectsStore) GetByKey(ctx context.Context, teamID, key string) (*projects.Project, error) {
	p := &projects.Project{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, key, name, issue_counter
		FROM projects WHERE team_id = $1 AND key = $2
	`, teamID, key).Scan(&p.ID, &p.TeamID, &p.Key, &p.Name, &p.IssueCounter)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *ProjectsStore) ListByTeam(ctx context.Context, teamID string) ([]*projects.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, key, name, issue_counter
		FROM projects WHERE team_id = $1
		ORDER BY name
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*projects.Project
	for rows.Next() {
		p := &projects.Project{}
		if err := rows.Scan(&p.ID, &p.TeamID, &p.Key, &p.Name, &p.IssueCounter); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (s *ProjectsStore) Update(ctx context.Context, id string, in *projects.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE projects SET
			key = COALESCE($2, key),
			name = COALESCE($3, name)
		WHERE id = $1
	`, id, in.Key, in.Name)
	return err
}

func (s *ProjectsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

func (s *ProjectsStore) IncrementIssueCounter(ctx context.Context, id string) (int, error) {
	// Note: Using UPDATE + SELECT instead of UPDATE RETURNING due to
	// DuckDB limitation with RETURNING when FK references exist
	_, err := s.db.ExecContext(ctx, `
		UPDATE projects SET issue_counter = issue_counter + 1 WHERE id = $1
	`, id)
	if err != nil {
		return 0, err
	}

	var counter int
	err = s.db.QueryRowContext(ctx, `
		SELECT issue_counter FROM projects WHERE id = $1
	`, id).Scan(&counter)
	return counter, err
}
