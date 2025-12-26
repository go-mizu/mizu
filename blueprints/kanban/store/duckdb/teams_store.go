package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/teams"
)

// TeamsStore handles team data access.
type TeamsStore struct {
	db *sql.DB
}

// NewTeamsStore creates a new teams store.
func NewTeamsStore(db *sql.DB) *TeamsStore {
	return &TeamsStore{db: db}
}

func (s *TeamsStore) Create(ctx context.Context, t *teams.Team) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO teams (id, workspace_id, key, name)
		VALUES ($1, $2, $3, $4)
	`, t.ID, t.WorkspaceID, t.Key, t.Name)
	return err
}

func (s *TeamsStore) GetByID(ctx context.Context, id string) (*teams.Team, error) {
	t := &teams.Team{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, key, name FROM teams WHERE id = $1
	`, id).Scan(&t.ID, &t.WorkspaceID, &t.Key, &t.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (s *TeamsStore) GetByKey(ctx context.Context, workspaceID, key string) (*teams.Team, error) {
	t := &teams.Team{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, key, name FROM teams WHERE workspace_id = $1 AND key = $2
	`, workspaceID, key).Scan(&t.ID, &t.WorkspaceID, &t.Key, &t.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (s *TeamsStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*teams.Team, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, key, name FROM teams WHERE workspace_id = $1 ORDER BY name
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*teams.Team
	for rows.Next() {
		t := &teams.Team{}
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Key, &t.Name); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (s *TeamsStore) Update(ctx context.Context, id string, in *teams.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE teams SET
			key = COALESCE($2, key),
			name = COALESCE($3, name)
		WHERE id = $1
	`, id, in.Key, in.Name)
	return err
}

func (s *TeamsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}

// Member operations

func (s *TeamsStore) AddMember(ctx context.Context, m *teams.Member) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO team_members (team_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (team_id, user_id) DO UPDATE SET role = $3
	`, m.TeamID, m.UserID, m.Role, m.JoinedAt)
	return err
}

func (s *TeamsStore) GetMember(ctx context.Context, teamID, userID string) (*teams.Member, error) {
	m := &teams.Member{}
	err := s.db.QueryRowContext(ctx, `
		SELECT team_id, user_id, role, joined_at
		FROM team_members
		WHERE team_id = $1 AND user_id = $2
	`, teamID, userID).Scan(&m.TeamID, &m.UserID, &m.Role, &m.JoinedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *TeamsStore) ListMembers(ctx context.Context, teamID string) ([]*teams.Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT team_id, user_id, role, joined_at
		FROM team_members
		WHERE team_id = $1
		ORDER BY joined_at
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*teams.Member
	for rows.Next() {
		m := &teams.Member{}
		if err := rows.Scan(&m.TeamID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *TeamsStore) UpdateMemberRole(ctx context.Context, teamID, userID, role string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE team_members SET role = $3 WHERE team_id = $1 AND user_id = $2
	`, teamID, userID, role)
	return err
}

func (s *TeamsStore) RemoveMember(ctx context.Context, teamID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM team_members WHERE team_id = $1 AND user_id = $2
	`, teamID, userID)
	return err
}

// Helper to get current time for JoinedAt
func TeamMemberNow() time.Time {
	return time.Now()
}
