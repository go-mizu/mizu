package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
)

// OrgsStore implements orgs.Store
type OrgsStore struct {
	db *sql.DB
}

// NewOrgsStore creates a new orgs store
func NewOrgsStore(db *sql.DB) *OrgsStore {
	return &OrgsStore{db: db}
}

// Create creates a new organization
func (s *OrgsStore) Create(ctx context.Context, o *orgs.Organization) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO organizations (id, name, slug, display_name, description, avatar_url, location, website, email, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, o.ID, o.Name, o.Slug, o.DisplayName, o.Description, o.AvatarURL, o.Location, o.Website, o.Email, o.IsVerified, o.CreatedAt, o.UpdatedAt)
	return err
}

// GetByID retrieves an organization by ID
func (s *OrgsStore) GetByID(ctx context.Context, id string) (*orgs.Organization, error) {
	o := &orgs.Organization{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, display_name, description, avatar_url, location, website, email, is_verified, created_at, updated_at
		FROM organizations WHERE id = $1
	`, id).Scan(&o.ID, &o.Name, &o.Slug, &o.DisplayName, &o.Description, &o.AvatarURL, &o.Location, &o.Website, &o.Email, &o.IsVerified, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return o, err
}

// GetBySlug retrieves an organization by slug
func (s *OrgsStore) GetBySlug(ctx context.Context, slug string) (*orgs.Organization, error) {
	o := &orgs.Organization{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, display_name, description, avatar_url, location, website, email, is_verified, created_at, updated_at
		FROM organizations WHERE slug = $1
	`, slug).Scan(&o.ID, &o.Name, &o.Slug, &o.DisplayName, &o.Description, &o.AvatarURL, &o.Location, &o.Website, &o.Email, &o.IsVerified, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return o, err
}

// Update updates an organization
func (s *OrgsStore) Update(ctx context.Context, o *orgs.Organization) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE organizations SET display_name = $2, description = $3, avatar_url = $4, location = $5, website = $6, email = $7, is_verified = $8, updated_at = $9
		WHERE id = $1
	`, o.ID, o.DisplayName, o.Description, o.AvatarURL, o.Location, o.Website, o.Email, o.IsVerified, o.UpdatedAt)
	return err
}

// Delete deletes an organization
func (s *OrgsStore) Delete(ctx context.Context, id string) error {
	// Delete members first
	s.db.ExecContext(ctx, `DELETE FROM org_members WHERE org_id = $1`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	return err
}

// List lists organizations
func (s *OrgsStore) List(ctx context.Context, limit, offset int) ([]*orgs.Organization, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, slug, display_name, description, avatar_url, location, website, email, is_verified, created_at, updated_at
		FROM organizations ORDER BY name ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*orgs.Organization
	for rows.Next() {
		o := &orgs.Organization{}
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.DisplayName, &o.Description, &o.AvatarURL, &o.Location, &o.Website, &o.Email, &o.IsVerified, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// AddMember adds a member to an organization
func (s *OrgsStore) AddMember(ctx context.Context, m *orgs.Member) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO org_members (id, org_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, m.ID, m.OrgID, m.UserID, m.Role, m.CreatedAt)
	return err
}

// RemoveMember removes a member from an organization
func (s *OrgsStore) RemoveMember(ctx context.Context, orgID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM org_members WHERE org_id = $1 AND user_id = $2`, orgID, userID)
	return err
}

// UpdateMember updates a member
func (s *OrgsStore) UpdateMember(ctx context.Context, m *orgs.Member) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE org_members SET role = $3
		WHERE org_id = $1 AND user_id = $2
	`, m.OrgID, m.UserID, m.Role)
	return err
}

// GetMember retrieves a member
func (s *OrgsStore) GetMember(ctx context.Context, orgID, userID string) (*orgs.Member, error) {
	m := &orgs.Member{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, user_id, role, created_at
		FROM org_members WHERE org_id = $1 AND user_id = $2
	`, orgID, userID).Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// ListMembers lists members of an organization
func (s *OrgsStore) ListMembers(ctx context.Context, orgID string, limit, offset int) ([]*orgs.Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, org_id, user_id, role, created_at
		FROM org_members WHERE org_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*orgs.Member
	for rows.Next() {
		m := &orgs.Member{}
		if err := rows.Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ListByUser lists organizations a user belongs to
func (s *OrgsStore) ListByUser(ctx context.Context, userID string) ([]*orgs.Organization, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT o.id, o.name, o.slug, o.display_name, o.description, o.avatar_url, o.location, o.website, o.email, o.is_verified, o.created_at, o.updated_at
		FROM organizations o
		JOIN org_members m ON o.id = m.org_id
		WHERE m.user_id = $1
		ORDER BY o.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*orgs.Organization
	for rows.Next() {
		o := &orgs.Organization{}
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.DisplayName, &o.Description, &o.AvatarURL, &o.Location, &o.Website, &o.Email, &o.IsVerified, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// CountOwners counts owners of an organization
func (s *OrgsStore) CountOwners(ctx context.Context, orgID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND role = 'owner'
	`, orgID).Scan(&count)
	return count, err
}
