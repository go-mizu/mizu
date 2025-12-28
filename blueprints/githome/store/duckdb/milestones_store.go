package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/milestones"
)

// MilestonesStore handles milestone data access.
type MilestonesStore struct {
	db *sql.DB
}

// NewMilestonesStore creates a new milestones store.
func NewMilestonesStore(db *sql.DB) *MilestonesStore {
	return &MilestonesStore{db: db}
}

func (s *MilestonesStore) Create(ctx context.Context, m *milestones.Milestone) error {
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO milestones (node_id, repo_id, number, state, title, description, creator_id,
			open_issues, closed_issues, closed_at, due_on, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, "", m.RepoID, m.Number, m.State, m.Title, m.Description, m.CreatorID,
		m.OpenIssues, m.ClosedIssues, nullTime(m.ClosedAt), nullTime(m.DueOn),
		m.CreatedAt, m.UpdatedAt).Scan(&m.ID)
	if err != nil {
		return err
	}

	m.NodeID = generateNodeID("M", m.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE milestones SET node_id = $1 WHERE id = $2`, m.NodeID, m.ID)
	return err
}

func (s *MilestonesStore) GetByID(ctx context.Context, id int64) (*milestones.Milestone, error) {
	m := &milestones.Milestone{}
	var closedAt, dueOn sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, number, state, title, description, creator_id,
			open_issues, closed_issues, closed_at, due_on, created_at, updated_at
		FROM milestones WHERE id = $1
	`, id).Scan(&m.ID, &m.NodeID, &m.RepoID, &m.Number, &m.State, &m.Title, &m.Description,
		&m.CreatorID, &m.OpenIssues, &m.ClosedIssues, &closedAt, &dueOn, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if closedAt.Valid {
		m.ClosedAt = &closedAt.Time
	}
	if dueOn.Valid {
		m.DueOn = &dueOn.Time
	}
	return m, err
}

func (s *MilestonesStore) GetByNumber(ctx context.Context, repoID int64, number int) (*milestones.Milestone, error) {
	m := &milestones.Milestone{}
	var closedAt, dueOn sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, number, state, title, description, creator_id,
			open_issues, closed_issues, closed_at, due_on, created_at, updated_at
		FROM milestones WHERE repo_id = $1 AND number = $2
	`, repoID, number).Scan(&m.ID, &m.NodeID, &m.RepoID, &m.Number, &m.State, &m.Title, &m.Description,
		&m.CreatorID, &m.OpenIssues, &m.ClosedIssues, &closedAt, &dueOn, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if closedAt.Valid {
		m.ClosedAt = &closedAt.Time
	}
	if dueOn.Valid {
		m.DueOn = &dueOn.Time
	}
	return m, err
}

func (s *MilestonesStore) GetByTitle(ctx context.Context, repoID int64, title string) (*milestones.Milestone, error) {
	m := &milestones.Milestone{}
	var closedAt, dueOn sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, number, state, title, description, creator_id,
			open_issues, closed_issues, closed_at, due_on, created_at, updated_at
		FROM milestones WHERE repo_id = $1 AND title = $2
	`, repoID, title).Scan(&m.ID, &m.NodeID, &m.RepoID, &m.Number, &m.State, &m.Title, &m.Description,
		&m.CreatorID, &m.OpenIssues, &m.ClosedIssues, &closedAt, &dueOn, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if closedAt.Valid {
		m.ClosedAt = &closedAt.Time
	}
	if dueOn.Valid {
		m.DueOn = &dueOn.Time
	}
	return m, err
}

func (s *MilestonesStore) Update(ctx context.Context, id int64, in *milestones.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE milestones SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			state = COALESCE($4, state),
			due_on = COALESCE($5, due_on),
			updated_at = $6
		WHERE id = $1
	`, id, nullStringPtr(in.Title), nullStringPtr(in.Description),
		nullStringPtr(in.State), nullTime(in.DueOn), time.Now())
	return err
}

func (s *MilestonesStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM milestones WHERE id = $1`, id)
	return err
}

func (s *MilestonesStore) List(ctx context.Context, repoID int64, opts *milestones.ListOpts) ([]*milestones.Milestone, error) {
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
		SELECT id, node_id, repo_id, number, state, title, description, creator_id,
			open_issues, closed_issues, closed_at, due_on, created_at, updated_at
		FROM milestones WHERE repo_id = $1`

	args := []any{repoID}
	if state != "all" {
		query += ` AND state = $2`
		args = append(args, state)
	}
	query += ` ORDER BY number ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*milestones.Milestone
	for rows.Next() {
		m := &milestones.Milestone{}
		var closedAt, dueOn sql.NullTime
		if err := rows.Scan(&m.ID, &m.NodeID, &m.RepoID, &m.Number, &m.State, &m.Title, &m.Description,
			&m.CreatorID, &m.OpenIssues, &m.ClosedIssues, &closedAt, &dueOn, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		if closedAt.Valid {
			m.ClosedAt = &closedAt.Time
		}
		if dueOn.Valid {
			m.DueOn = &dueOn.Time
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *MilestonesStore) NextNumber(ctx context.Context, repoID int64) (int, error) {
	var maxNumber sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(number) FROM milestones WHERE repo_id = $1
	`, repoID).Scan(&maxNumber)
	if err != nil {
		return 0, err
	}
	if maxNumber.Valid {
		return int(maxNumber.Int64) + 1, nil
	}
	return 1, nil
}

func (s *MilestonesStore) IncrementOpenIssues(ctx context.Context, id int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE milestones SET open_issues = open_issues + $2, updated_at = $3 WHERE id = $1
	`, id, delta, time.Now())
	return err
}

func (s *MilestonesStore) IncrementClosedIssues(ctx context.Context, id int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE milestones SET closed_issues = closed_issues + $2, updated_at = $3 WHERE id = $1
	`, id, delta, time.Now())
	return err
}
