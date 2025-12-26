package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/kanban/feature/sprints"
)

// SprintsStore handles sprint data access.
type SprintsStore struct {
	db *sql.DB
}

// NewSprintsStore creates a new sprints store.
func NewSprintsStore(db *sql.DB) *SprintsStore {
	return &SprintsStore{db: db}
}

func (s *SprintsStore) Create(ctx context.Context, sp *sprints.Sprint) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sprints (id, project_id, name, goal, status, start_date, end_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, sp.ID, sp.ProjectID, sp.Name, sp.Goal, sp.Status, sp.StartDate, sp.EndDate, sp.CreatedAt)
	return err
}

func (s *SprintsStore) GetByID(ctx context.Context, id string) (*sprints.Sprint, error) {
	sp := &sprints.Sprint{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, goal, status, start_date, end_date, created_at
		FROM sprints WHERE id = $1
	`, id).Scan(&sp.ID, &sp.ProjectID, &sp.Name, &sp.Goal, &sp.Status, &sp.StartDate, &sp.EndDate, &sp.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sp, err
}

func (s *SprintsStore) ListByProject(ctx context.Context, projectID string) ([]*sprints.Sprint, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, name, goal, status, start_date, end_date, created_at
		FROM sprints WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*sprints.Sprint
	for rows.Next() {
		sp := &sprints.Sprint{}
		if err := rows.Scan(&sp.ID, &sp.ProjectID, &sp.Name, &sp.Goal, &sp.Status, &sp.StartDate, &sp.EndDate, &sp.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, sp)
	}
	return list, rows.Err()
}

func (s *SprintsStore) GetActive(ctx context.Context, projectID string) (*sprints.Sprint, error) {
	sp := &sprints.Sprint{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, goal, status, start_date, end_date, created_at
		FROM sprints WHERE project_id = $1 AND status = 'active'
		LIMIT 1
	`, projectID).Scan(&sp.ID, &sp.ProjectID, &sp.Name, &sp.Goal, &sp.Status, &sp.StartDate, &sp.EndDate, &sp.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sp, err
}

func (s *SprintsStore) Update(ctx context.Context, id string, in *sprints.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sprints SET
			name = COALESCE($2, name),
			goal = COALESCE($3, goal),
			start_date = COALESCE($4, start_date),
			end_date = COALESCE($5, end_date)
		WHERE id = $1
	`, id, in.Name, in.Goal, in.StartDate, in.EndDate)
	return err
}

func (s *SprintsStore) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sprints SET status = $2 WHERE id = $1
	`, id, status)
	return err
}

func (s *SprintsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sprints WHERE id = $1`, id)
	return err
}

func (s *SprintsStore) GetIssueCount(ctx context.Context, sprintID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM issues WHERE sprint_id = $1
	`, sprintID).Scan(&count)
	return count, err
}

func (s *SprintsStore) GetDoneIssueCount(ctx context.Context, sprintID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM issues WHERE sprint_id = $1 AND status = 'done'
	`, sprintID).Scan(&count)
	return count, err
}
