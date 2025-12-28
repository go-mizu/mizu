package duckdb

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/teams"
	"github.com/go-mizu/blueprints/githome/feature/users"
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
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO teams (node_id, org_id, name, slug, description, privacy, permission, parent_id,
			members_count, repos_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, "", t.OrgID, t.Name, t.Slug, t.Description, t.Privacy, t.Permission, nullInt64Ptr(t.ParentID),
		t.MembersCount, t.ReposCount, t.CreatedAt, t.UpdatedAt).Scan(&t.ID)
	if err != nil {
		return err
	}

	t.NodeID = generateNodeID("T", t.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE teams SET node_id = $1 WHERE id = $2`, t.NodeID, t.ID)
	return err
}

func (s *TeamsStore) GetByID(ctx context.Context, id int64) (*teams.Team, error) {
	t := &teams.Team{}
	var parentID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, org_id, name, slug, description, privacy, permission, parent_id,
			members_count, repos_count, created_at, updated_at
		FROM teams WHERE id = $1
	`, id).Scan(&t.ID, &t.NodeID, &t.OrgID, &t.Name, &t.Slug, &t.Description, &t.Privacy, &t.Permission,
		&parentID, &t.MembersCount, &t.ReposCount, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if parentID.Valid {
		t.ParentID = &parentID.Int64
	}
	return t, err
}

func (s *TeamsStore) GetBySlug(ctx context.Context, orgID int64, slug string) (*teams.Team, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM teams WHERE org_id = $1 AND slug = $2`, orgID, slug).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *TeamsStore) Update(ctx context.Context, id int64, in *teams.UpdateIn) error {
	// Compute new slug if name is being updated
	var slugPtr *string
	if in.Name != nil {
		slug := slugifyTeamName(*in.Name)
		slugPtr = &slug
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE teams SET
			name = COALESCE($2, name),
			slug = COALESCE($7, slug),
			description = COALESCE($3, description),
			privacy = COALESCE($4, privacy),
			permission = COALESCE($5, permission),
			parent_id = COALESCE($6, parent_id),
			updated_at = $8
		WHERE id = $1
	`, id, nullStringPtr(in.Name), nullStringPtr(in.Description), nullStringPtr(in.Privacy),
		nullStringPtr(in.Permission), nullInt64Ptr(in.ParentTeamID), nullStringPtr(slugPtr), time.Now())
	return err
}

func (s *TeamsStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}

func (s *TeamsStore) List(ctx context.Context, orgID int64, opts *teams.ListOpts) ([]*teams.Team, error) {
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
		SELECT id, node_id, org_id, name, slug, description, privacy, permission, parent_id,
			members_count, repos_count, created_at, updated_at
		FROM teams WHERE org_id = $1
		ORDER BY name ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTeams(rows)
}

func (s *TeamsStore) ListChildren(ctx context.Context, parentID int64, opts *teams.ListOpts) ([]*teams.Team, error) {
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
		SELECT id, node_id, org_id, name, slug, description, privacy, permission, parent_id,
			members_count, repos_count, created_at, updated_at
		FROM teams WHERE parent_id = $1
		ORDER BY name ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTeams(rows)
}

// Membership

func (s *TeamsStore) AddMember(ctx context.Context, teamID, userID int64, role string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO team_members (team_id, user_id, role, state, created_at)
		VALUES ($1, $2, $3, 'active', $4)
		ON CONFLICT (team_id, user_id) DO UPDATE SET role = $3, state = 'active'
	`, teamID, userID, role, time.Now())
	return err
}

func (s *TeamsStore) UpdateMemberRole(ctx context.Context, teamID, userID int64, role string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE team_members SET role = $3 WHERE team_id = $1 AND user_id = $2
	`, teamID, userID, role)
	return err
}

func (s *TeamsStore) RemoveMember(ctx context.Context, teamID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM team_members WHERE team_id = $1 AND user_id = $2
	`, teamID, userID)
	return err
}

func (s *TeamsStore) GetMember(ctx context.Context, teamID, userID int64) (*teams.Membership, error) {
	m := &teams.Membership{}
	err := s.db.QueryRowContext(ctx, `
		SELECT role, state FROM team_members WHERE team_id = $1 AND user_id = $2
	`, teamID, userID).Scan(&m.Role, &m.State)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *TeamsStore) ListMembers(ctx context.Context, teamID int64, opts *teams.ListOpts) ([]*users.SimpleUser, error) {
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
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM users u
		JOIN team_members m ON m.user_id = u.id
		WHERE m.team_id = $1 AND m.state = 'active'
		ORDER BY u.login ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func (s *TeamsStore) IsMember(ctx context.Context, teamID, userID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id = $2 AND state = 'active'
	`, teamID, userID).Scan(&count)
	return count > 0, err
}

// Repositories

func (s *TeamsStore) AddRepo(ctx context.Context, teamID, repoID int64, permission string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO team_repos (team_id, repo_id, permission)
		VALUES ($1, $2, $3)
		ON CONFLICT (team_id, repo_id) DO UPDATE SET permission = $3
	`, teamID, repoID, permission)
	return err
}

func (s *TeamsStore) UpdateRepoPermission(ctx context.Context, teamID, repoID int64, permission string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE team_repos SET permission = $3 WHERE team_id = $1 AND repo_id = $2
	`, teamID, repoID, permission)
	return err
}

func (s *TeamsStore) RemoveRepo(ctx context.Context, teamID, repoID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM team_repos WHERE team_id = $1 AND repo_id = $2
	`, teamID, repoID)
	return err
}

func (s *TeamsStore) GetRepoPermission(ctx context.Context, teamID, repoID int64) (*teams.RepoPermission, error) {
	p := &teams.RepoPermission{}
	err := s.db.QueryRowContext(ctx, `
		SELECT permission FROM team_repos WHERE team_id = $1 AND repo_id = $2
	`, teamID, repoID).Scan(&p.Permission)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *TeamsStore) ListRepos(ctx context.Context, teamID int64, opts *teams.ListOpts) ([]*teams.Repository, error) {
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
		SELECT r.id, r.node_id, r.name, r.full_name, r.private, r.description, tr.permission
		FROM repositories r
		JOIN team_repos tr ON tr.repo_id = r.id
		WHERE tr.team_id = $1
		ORDER BY r.full_name ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*teams.Repository
	for rows.Next() {
		repo := &teams.Repository{}
		var permission string
		if err := rows.Scan(&repo.ID, &repo.NodeID, &repo.Name, &repo.FullName,
			&repo.Private, &repo.Description, &permission); err != nil {
			return nil, err
		}
		repo.Permissions = permissionToRepoPermissions(permission)
		list = append(list, repo)
	}
	return list, rows.Err()
}

func permissionToRepoPermissions(permission string) *teams.RepoPermissions {
	p := &teams.RepoPermissions{}
	switch permission {
	case "admin":
		p.Admin = true
		p.Maintain = true
		p.Push = true
		p.Triage = true
		p.Pull = true
	case "maintain":
		p.Maintain = true
		p.Push = true
		p.Triage = true
		p.Pull = true
	case "push", "write":
		p.Push = true
		p.Triage = true
		p.Pull = true
	case "triage":
		p.Triage = true
		p.Pull = true
	case "pull", "read":
		p.Pull = true
	}
	return p
}

// Counter operations

func (s *TeamsStore) IncrementMembers(ctx context.Context, teamID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE teams SET members_count = members_count + $2, updated_at = $3 WHERE id = $1
	`, teamID, delta, time.Now())
	return err
}

func (s *TeamsStore) IncrementRepos(ctx context.Context, teamID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE teams SET repos_count = repos_count + $2, updated_at = $3 WHERE id = $1
	`, teamID, delta, time.Now())
	return err
}

func scanTeams(rows *sql.Rows) ([]*teams.Team, error) {
	var list []*teams.Team
	for rows.Next() {
		t := &teams.Team{}
		var parentID sql.NullInt64
		if err := rows.Scan(&t.ID, &t.NodeID, &t.OrgID, &t.Name, &t.Slug, &t.Description,
			&t.Privacy, &t.Permission, &parentID, &t.MembersCount, &t.ReposCount,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			t.ParentID = &parentID.Int64
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// slugifyTeamName converts a team name to a URL-friendly slug
func slugifyTeamName(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}
