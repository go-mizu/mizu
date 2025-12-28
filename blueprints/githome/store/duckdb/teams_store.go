package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/teams"
)

// TeamsStore implements teams.Store
type TeamsStore struct {
	db *sql.DB
}

// NewTeamsStore creates a new teams store
func NewTeamsStore(db *sql.DB) *TeamsStore {
	return &TeamsStore{db: db}
}

// Create creates a new team
func (s *TeamsStore) Create(ctx context.Context, t *teams.Team) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO teams (id, org_id, name, slug, description, permission, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, t.ID, t.OrgID, t.Name, t.Slug, t.Description, t.Permission, nullString(t.ParentID), t.CreatedAt, t.UpdatedAt)
	return err
}

// GetByID retrieves a team by ID
func (s *TeamsStore) GetByID(ctx context.Context, id string) (*teams.Team, error) {
	t := &teams.Team{}
	var parentID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, name, slug, description, permission, parent_id, created_at, updated_at
		FROM teams WHERE id = $1
	`, id).Scan(&t.ID, &t.OrgID, &t.Name, &t.Slug, &t.Description, &t.Permission, &parentID, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		t.ParentID = parentID.String
	}
	return t, nil
}

// GetBySlug retrieves a team by organization ID and slug
func (s *TeamsStore) GetBySlug(ctx context.Context, orgID, slug string) (*teams.Team, error) {
	t := &teams.Team{}
	var parentID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, name, slug, description, permission, parent_id, created_at, updated_at
		FROM teams WHERE org_id = $1 AND slug = $2
	`, orgID, slug).Scan(&t.ID, &t.OrgID, &t.Name, &t.Slug, &t.Description, &t.Permission, &parentID, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		t.ParentID = parentID.String
	}
	return t, nil
}

// Update updates a team
func (s *TeamsStore) Update(ctx context.Context, t *teams.Team) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE teams SET name = $2, slug = $3, description = $4, permission = $5, parent_id = $6, updated_at = $7
		WHERE id = $1
	`, t.ID, t.Name, t.Slug, t.Description, t.Permission, nullString(t.ParentID), t.UpdatedAt)
	return err
}

// Delete deletes a team
func (s *TeamsStore) Delete(ctx context.Context, id string) error {
	// Delete members and repos first
	s.db.ExecContext(ctx, `DELETE FROM team_members WHERE team_id = $1`, id)
	s.db.ExecContext(ctx, `DELETE FROM team_repos WHERE team_id = $1`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}

// List lists teams in an organization
func (s *TeamsStore) List(ctx context.Context, orgID string, limit, offset int) ([]*teams.Team, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, org_id, name, slug, description, permission, parent_id, created_at, updated_at
		FROM teams WHERE org_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanTeams(rows)
}

// ListChildren lists child teams of a parent team
func (s *TeamsStore) ListChildren(ctx context.Context, parentID string) ([]*teams.Team, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, org_id, name, slug, description, permission, parent_id, created_at, updated_at
		FROM teams WHERE parent_id = $1
		ORDER BY name ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanTeams(rows)
}

func (s *TeamsStore) scanTeams(rows *sql.Rows) ([]*teams.Team, error) {
	var list []*teams.Team
	for rows.Next() {
		t := &teams.Team{}
		var parentID sql.NullString
		if err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.Slug, &t.Description, &t.Permission, &parentID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			t.ParentID = parentID.String
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// AddMember adds a member to a team - uses composite PK (team_id, user_id)
func (s *TeamsStore) AddMember(ctx context.Context, m *teams.TeamMember) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO team_members (team_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4)
	`, m.TeamID, m.UserID, m.Role, m.CreatedAt)
	return err
}

// RemoveMember removes a member from a team
func (s *TeamsStore) RemoveMember(ctx context.Context, teamID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	return err
}

// UpdateMember updates a team member
func (s *TeamsStore) UpdateMember(ctx context.Context, m *teams.TeamMember) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE team_members SET role = $3
		WHERE team_id = $1 AND user_id = $2
	`, m.TeamID, m.UserID, m.Role)
	return err
}

// GetMember retrieves a team member - uses composite PK lookup
func (s *TeamsStore) GetMember(ctx context.Context, teamID, userID string) (*teams.TeamMember, error) {
	m := &teams.TeamMember{}
	err := s.db.QueryRowContext(ctx, `
		SELECT team_id, user_id, role, created_at
		FROM team_members WHERE team_id = $1 AND user_id = $2
	`, teamID, userID).Scan(&m.TeamID, &m.UserID, &m.Role, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// ListMembers lists members of a team
func (s *TeamsStore) ListMembers(ctx context.Context, teamID string, limit, offset int) ([]*teams.TeamMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT team_id, user_id, role, created_at
		FROM team_members WHERE team_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*teams.TeamMember
	for rows.Next() {
		m := &teams.TeamMember{}
		if err := rows.Scan(&m.TeamID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ListByUser lists teams a user belongs to in an organization
func (s *TeamsStore) ListByUser(ctx context.Context, orgID, userID string) ([]*teams.Team, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.org_id, t.name, t.slug, t.description, t.permission, t.parent_id, t.created_at, t.updated_at
		FROM teams t
		JOIN team_members m ON t.id = m.team_id
		WHERE t.org_id = $1 AND m.user_id = $2
		ORDER BY t.name ASC
	`, orgID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanTeams(rows)
}

// AddRepo adds a repository to a team - uses composite PK (team_id, repo_id)
func (s *TeamsStore) AddRepo(ctx context.Context, tr *teams.TeamRepo) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO team_repos (team_id, repo_id, permission, created_at)
		VALUES ($1, $2, $3, $4)
	`, tr.TeamID, tr.RepoID, tr.Permission, tr.CreatedAt)
	return err
}

// RemoveRepo removes a repository from a team
func (s *TeamsStore) RemoveRepo(ctx context.Context, teamID, repoID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM team_repos WHERE team_id = $1 AND repo_id = $2`, teamID, repoID)
	return err
}

// UpdateRepo updates a team's repository permission
func (s *TeamsStore) UpdateRepo(ctx context.Context, tr *teams.TeamRepo) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE team_repos SET permission = $3
		WHERE team_id = $1 AND repo_id = $2
	`, tr.TeamID, tr.RepoID, tr.Permission)
	return err
}

// GetRepo retrieves a team's repository access - uses composite PK lookup
func (s *TeamsStore) GetRepo(ctx context.Context, teamID, repoID string) (*teams.TeamRepo, error) {
	tr := &teams.TeamRepo{}
	err := s.db.QueryRowContext(ctx, `
		SELECT team_id, repo_id, permission, created_at
		FROM team_repos WHERE team_id = $1 AND repo_id = $2
	`, teamID, repoID).Scan(&tr.TeamID, &tr.RepoID, &tr.Permission, &tr.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tr, err
}

// ListRepos lists repositories a team has access to
func (s *TeamsStore) ListRepos(ctx context.Context, teamID string, limit, offset int) ([]*teams.TeamRepo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT team_id, repo_id, permission, created_at
		FROM team_repos WHERE team_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*teams.TeamRepo
	for rows.Next() {
		tr := &teams.TeamRepo{}
		if err := rows.Scan(&tr.TeamID, &tr.RepoID, &tr.Permission, &tr.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, tr)
	}
	return list, rows.Err()
}
