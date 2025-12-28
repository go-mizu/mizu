package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// OrgsStore handles organization data access.
type OrgsStore struct {
	db *sql.DB
}

// NewOrgsStore creates a new organizations store.
func NewOrgsStore(db *sql.DB) *OrgsStore {
	return &OrgsStore{db: db}
}

func (s *OrgsStore) Create(ctx context.Context, org *orgs.Organization) error {
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO organizations (node_id, login, name, description, company, blog, location, email,
			twitter_username, avatar_url, is_verified, has_organization_projects, has_repository_projects,
			public_repos, default_repository_permission, members_can_create_repositories,
			members_can_create_public_repositories, members_can_create_private_repositories,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id
	`, "", org.Login, org.Name, org.Description, org.Company, org.Blog, org.Location, org.Email,
		org.TwitterUsername, org.AvatarURL, org.IsVerified, org.HasOrganizationProjects,
		org.HasRepositoryProjects, org.PublicRepos, org.DefaultRepositoryPermission,
		org.MembersCanCreateRepositories, org.MembersCanCreatePublicRepositories,
		org.MembersCanCreatePrivateRepositories, org.CreatedAt, org.UpdatedAt).Scan(&org.ID)
	if err != nil {
		return err
	}

	org.NodeID = generateNodeID("O", org.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE organizations SET node_id = $1 WHERE id = $2`, org.NodeID, org.ID)
	return err
}

func (s *OrgsStore) GetByID(ctx context.Context, id int64) (*orgs.Organization, error) {
	org := &orgs.Organization{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, login, name, description, company, blog, location, email,
			twitter_username, avatar_url, is_verified, has_organization_projects, has_repository_projects,
			public_repos, public_gists, followers, following, total_private_repos, owned_private_repos,
			default_repository_permission, members_can_create_repositories,
			members_can_create_public_repositories, members_can_create_private_repositories,
			created_at, updated_at
		FROM organizations WHERE id = $1
	`, id).Scan(&org.ID, &org.NodeID, &org.Login, &org.Name, &org.Description, &org.Company,
		&org.Blog, &org.Location, &org.Email, &org.TwitterUsername, &org.AvatarURL, &org.IsVerified,
		&org.HasOrganizationProjects, &org.HasRepositoryProjects, &org.PublicRepos, &org.PublicGists,
		&org.Followers, &org.Following, &org.TotalPrivateRepos, &org.OwnedPrivateRepos,
		&org.DefaultRepositoryPermission, &org.MembersCanCreateRepositories,
		&org.MembersCanCreatePublicRepositories, &org.MembersCanCreatePrivateRepositories,
		&org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return org, err
}

func (s *OrgsStore) GetByLogin(ctx context.Context, login string) (*orgs.Organization, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM organizations WHERE login = $1`, login).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *OrgsStore) Update(ctx context.Context, id int64, in *orgs.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE organizations SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			company = COALESCE($4, company),
			blog = COALESCE($5, blog),
			location = COALESCE($6, location),
			email = COALESCE($7, email),
			twitter_username = COALESCE($8, twitter_username),
			default_repository_permission = COALESCE($9, default_repository_permission),
			members_can_create_repositories = COALESCE($10, members_can_create_repositories),
			updated_at = $11
		WHERE id = $1
	`, id, nullStringPtr(in.Name), nullStringPtr(in.Description), nullStringPtr(in.Company),
		nullStringPtr(in.Blog), nullStringPtr(in.Location), nullStringPtr(in.Email),
		nullStringPtr(in.TwitterUsername), nullStringPtr(in.DefaultRepositoryPermission),
		nullBoolPtr(in.MembersCanCreateRepositories), time.Now())
	return err
}

func (s *OrgsStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	return err
}

func (s *OrgsStore) List(ctx context.Context, opts *orgs.ListOpts) ([]*orgs.OrgSimple, error) {
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
		SELECT id, node_id, login, avatar_url, description
		FROM organizations
		ORDER BY login ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOrgSimples(rows)
}

func (s *OrgsStore) ListForUser(ctx context.Context, userID int64, opts *orgs.ListOpts) ([]*orgs.OrgSimple, error) {
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
		SELECT o.id, o.node_id, o.login, o.avatar_url, o.description
		FROM organizations o
		JOIN org_members m ON m.org_id = o.id
		WHERE m.user_id = $1 AND m.state = 'active'
		ORDER BY o.login ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOrgSimples(rows)
}

// Membership operations

func (s *OrgsStore) AddMember(ctx context.Context, orgID, userID int64, role string, isPublic bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO org_members (org_id, user_id, role, is_public, state, created_at)
		VALUES ($1, $2, $3, $4, 'active', $5)
		ON CONFLICT (org_id, user_id) DO UPDATE SET role = $3, is_public = $4, state = 'active'
	`, orgID, userID, role, isPublic, time.Now())
	return err
}

func (s *OrgsStore) UpdateMemberRole(ctx context.Context, orgID, userID int64, role string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE org_members SET role = $3 WHERE org_id = $1 AND user_id = $2
	`, orgID, userID, role)
	return err
}

func (s *OrgsStore) RemoveMember(ctx context.Context, orgID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM org_members WHERE org_id = $1 AND user_id = $2
	`, orgID, userID)
	return err
}

func (s *OrgsStore) GetMember(ctx context.Context, orgID, userID int64) (*orgs.Member, error) {
	user := &users.SimpleUser{}
	m := &orgs.Member{SimpleUser: user}
	var email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin,
			m.role
		FROM org_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.org_id = $1 AND m.user_id = $2
	`, orgID, userID).Scan(&user.ID, &user.NodeID, &user.Login, &user.Name, &email,
		&user.AvatarURL, &user.Type, &user.SiteAdmin, &m.Role)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if email.Valid {
		user.Email = email.String
	}
	return m, err
}

func (s *OrgsStore) ListMembers(ctx context.Context, orgID int64, opts *orgs.ListMembersOpts) ([]*users.SimpleUser, error) {
	page, perPage := 1, 30
	role := ""
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
		role = opts.Role
	}

	query := `
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM users u
		JOIN org_members m ON m.user_id = u.id
		WHERE m.org_id = $1 AND m.state = 'active'`
	args := []any{orgID}
	if role != "" && role != "all" {
		query += ` AND m.role = $2`
		args = append(args, role)
	}
	query += ` ORDER BY u.login ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func (s *OrgsStore) ListPublicMembers(ctx context.Context, orgID int64, opts *orgs.ListOpts) ([]*users.SimpleUser, error) {
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
		JOIN org_members m ON m.user_id = u.id
		WHERE m.org_id = $1 AND m.state = 'active' AND m.is_public = TRUE
		ORDER BY u.login ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func (s *OrgsStore) IsMember(ctx context.Context, orgID, userID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND user_id = $2 AND state = 'active'
	`, orgID, userID).Scan(&count)
	return count > 0, err
}

func (s *OrgsStore) IsPublicMember(ctx context.Context, orgID, userID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND user_id = $2 AND state = 'active' AND is_public = TRUE
	`, orgID, userID).Scan(&count)
	return count > 0, err
}

func (s *OrgsStore) SetMemberPublicity(ctx context.Context, orgID, userID int64, isPublic bool) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE org_members SET is_public = $3 WHERE org_id = $1 AND user_id = $2
	`, orgID, userID, isPublic)
	return err
}

func (s *OrgsStore) CountOwners(ctx context.Context, orgID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND role = 'admin' AND state = 'active'
	`, orgID).Scan(&count)
	return count, err
}

func scanOrgSimples(rows *sql.Rows) ([]*orgs.OrgSimple, error) {
	var list []*orgs.OrgSimple
	for rows.Next() {
		o := &orgs.OrgSimple{}
		if err := rows.Scan(&o.ID, &o.NodeID, &o.Login, &o.AvatarURL, &o.Description); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}
