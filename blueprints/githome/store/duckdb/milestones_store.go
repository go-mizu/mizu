package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/milestones"
)

// MilestonesStore implements milestones.Store
type MilestonesStore struct {
	db *sql.DB
}

// NewMilestonesStore creates a new milestones store
func NewMilestonesStore(db *sql.DB) *MilestonesStore {
	return &MilestonesStore{db: db}
}

// Create creates a new milestone
func (s *MilestonesStore) Create(ctx context.Context, m *milestones.Milestone) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO milestones (id, repo_id, number, title, description, state, due_date, created_at, updated_at, closed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, m.ID, m.RepoID, m.Number, m.Title, m.Description, m.State, m.DueDate, m.CreatedAt, m.UpdatedAt, m.ClosedAt)
	return err
}

// GetByID retrieves a milestone by ID
func (s *MilestonesStore) GetByID(ctx context.Context, id string) (*milestones.Milestone, error) {
	m := &milestones.Milestone{}
	var dueDate, closedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, number, title, description, state, due_date, created_at, updated_at, closed_at
		FROM milestones WHERE id = $1
	`, id).Scan(&m.ID, &m.RepoID, &m.Number, &m.Title, &m.Description, &m.State, &dueDate, &m.CreatedAt, &m.UpdatedAt, &closedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if dueDate.Valid {
		m.DueDate = &dueDate.Time
	}
	if closedAt.Valid {
		m.ClosedAt = &closedAt.Time
	}
	return m, nil
}

// GetByNumber retrieves a milestone by repository ID and number
func (s *MilestonesStore) GetByNumber(ctx context.Context, repoID string, number int) (*milestones.Milestone, error) {
	m := &milestones.Milestone{}
	var dueDate, closedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, number, title, description, state, due_date, created_at, updated_at, closed_at
		FROM milestones WHERE repo_id = $1 AND number = $2
	`, repoID, number).Scan(&m.ID, &m.RepoID, &m.Number, &m.Title, &m.Description, &m.State, &dueDate, &m.CreatedAt, &m.UpdatedAt, &closedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if dueDate.Valid {
		m.DueDate = &dueDate.Time
	}
	if closedAt.Valid {
		m.ClosedAt = &closedAt.Time
	}
	return m, nil
}

// Update updates a milestone
func (s *MilestonesStore) Update(ctx context.Context, m *milestones.Milestone) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE milestones SET title = $2, description = $3, state = $4, due_date = $5, updated_at = $6, closed_at = $7
		WHERE id = $1
	`, m.ID, m.Title, m.Description, m.State, m.DueDate, m.UpdatedAt, m.ClosedAt)
	return err
}

// Delete deletes a milestone
func (s *MilestonesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM milestones WHERE id = $1`, id)
	return err
}

// List lists milestones for a repository
func (s *MilestonesStore) List(ctx context.Context, repoID string, state string) ([]*milestones.Milestone, error) {
	query := `
		SELECT id, repo_id, number, title, description, state, due_date, created_at, updated_at, closed_at
		FROM milestones WHERE repo_id = $1`
	args := []interface{}{repoID}

	if state != "" && state != "all" {
		query += ` AND state = $2`
		args = append(args, state)
	}
	query += ` ORDER BY number ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*milestones.Milestone
	for rows.Next() {
		m := &milestones.Milestone{}
		var dueDate, closedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.RepoID, &m.Number, &m.Title, &m.Description, &m.State, &dueDate, &m.CreatedAt, &m.UpdatedAt, &closedAt); err != nil {
			return nil, err
		}
		if dueDate.Valid {
			m.DueDate = &dueDate.Time
		}
		if closedAt.Valid {
			m.ClosedAt = &closedAt.Time
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// GetNextNumber gets the next milestone number for a repository
func (s *MilestonesStore) GetNextNumber(ctx context.Context, repoID string) (int, error) {
	var maxNum sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(number) FROM milestones WHERE repo_id = $1
	`, repoID).Scan(&maxNum)
	if err != nil {
		return 1, err
	}
	if !maxNum.Valid {
		return 1, nil
	}
	return int(maxNum.Int64) + 1, nil
}
