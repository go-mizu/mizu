package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/cycles"
)

// CyclesStore handles cycle data access.
type CyclesStore struct {
	db *sql.DB
}

// NewCyclesStore creates a new cycles store.
func NewCyclesStore(db *sql.DB) *CyclesStore {
	return &CyclesStore{db: db}
}

func (s *CyclesStore) Create(ctx context.Context, c *cycles.Cycle) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cycles (id, team_id, number, name, status, start_date, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.TeamID, c.Number, c.Name, c.Status, c.StartDate, c.EndDate, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *CyclesStore) GetByID(ctx context.Context, id string) (*cycles.Cycle, error) {
	c := &cycles.Cycle{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, number, name, status, start_date, end_date, created_at, updated_at
		FROM cycles WHERE id = $1
	`, id).Scan(&c.ID, &c.TeamID, &c.Number, &c.Name, &c.Status, &c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (s *CyclesStore) GetByNumber(ctx context.Context, teamID string, number int) (*cycles.Cycle, error) {
	c := &cycles.Cycle{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, number, name, status, start_date, end_date, created_at, updated_at
		FROM cycles WHERE team_id = $1 AND number = $2
	`, teamID, number).Scan(&c.ID, &c.TeamID, &c.Number, &c.Name, &c.Status, &c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (s *CyclesStore) ListByTeam(ctx context.Context, teamID string) ([]*cycles.Cycle, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, number, name, status, start_date, end_date, created_at, updated_at
		FROM cycles WHERE team_id = $1
		ORDER BY number DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*cycles.Cycle
	for rows.Next() {
		c := &cycles.Cycle{}
		if err := rows.Scan(&c.ID, &c.TeamID, &c.Number, &c.Name, &c.Status, &c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *CyclesStore) GetActive(ctx context.Context, teamID string) (*cycles.Cycle, error) {
	c := &cycles.Cycle{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, number, name, status, start_date, end_date, created_at, updated_at
		FROM cycles WHERE team_id = $1 AND status = 'active'
		LIMIT 1
	`, teamID).Scan(&c.ID, &c.TeamID, &c.Number, &c.Name, &c.Status, &c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (s *CyclesStore) Update(ctx context.Context, id string, in *cycles.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE cycles SET
			name = COALESCE($2, name),
			start_date = COALESCE($3, start_date),
			end_date = COALESCE($4, end_date),
			updated_at = $5
		WHERE id = $1
	`, id, in.Name, in.StartDate, in.EndDate, time.Now())
	return err
}

func (s *CyclesStore) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE cycles SET status = $2, updated_at = $3 WHERE id = $1
	`, id, status, time.Now())
	return err
}

func (s *CyclesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cycles WHERE id = $1`, id)
	return err
}

func (s *CyclesStore) GetNextNumber(ctx context.Context, teamID string) (int, error) {
	var maxNum sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(number) FROM cycles WHERE team_id = $1
	`, teamID).Scan(&maxNum)
	if err != nil {
		return 0, err
	}
	if !maxNum.Valid {
		return 1, nil
	}
	return int(maxNum.Int64) + 1, nil
}
