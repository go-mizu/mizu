package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/collaborators"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// CollaboratorsStore handles collaborator data access.
type CollaboratorsStore struct {
	db *sql.DB
}

// NewCollaboratorsStore creates a new collaborators store.
func NewCollaboratorsStore(db *sql.DB) *CollaboratorsStore {
	return &CollaboratorsStore{db: db}
}

func (s *CollaboratorsStore) Add(ctx context.Context, repoID, userID int64, permission string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collaborators (repo_id, user_id, permission, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (repo_id, user_id) DO UPDATE SET permission = $3
	`, repoID, userID, permission, time.Now())
	return err
}

func (s *CollaboratorsStore) Remove(ctx context.Context, repoID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM collaborators WHERE repo_id = $1 AND user_id = $2
	`, repoID, userID)
	return err
}

func (s *CollaboratorsStore) Get(ctx context.Context, repoID, userID int64) (*collaborators.Collaborator, error) {
	user := &users.SimpleUser{}
	c := &collaborators.Collaborator{SimpleUser: user}
	var email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin, c.permission
		FROM collaborators c
		JOIN users u ON u.id = c.user_id
		WHERE c.repo_id = $1 AND c.user_id = $2
	`, repoID, userID).Scan(&user.ID, &user.NodeID, &user.Login, &user.Name, &email,
		&user.AvatarURL, &user.Type, &user.SiteAdmin, &c.RoleName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if email.Valid {
		user.Email = email.String
	}
	c.Permissions = collaborators.PermissionToPermissions(c.RoleName)
	return c, err
}

func (s *CollaboratorsStore) List(ctx context.Context, repoID int64, opts *collaborators.ListOpts) ([]*collaborators.Collaborator, error) {
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
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin, c.permission
		FROM collaborators c
		JOIN users u ON u.id = c.user_id
		WHERE c.repo_id = $1
		ORDER BY c.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*collaborators.Collaborator
	for rows.Next() {
		user := &users.SimpleUser{}
		c := &collaborators.Collaborator{SimpleUser: user}
		var email sql.NullString
		if err := rows.Scan(&user.ID, &user.NodeID, &user.Login, &user.Name, &email,
			&user.AvatarURL, &user.Type, &user.SiteAdmin, &c.RoleName); err != nil {
			return nil, err
		}
		if email.Valid {
			user.Email = email.String
		}
		c.Permissions = collaborators.PermissionToPermissions(c.RoleName)
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *CollaboratorsStore) UpdatePermission(ctx context.Context, repoID, userID int64, permission string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE collaborators SET permission = $3 WHERE repo_id = $1 AND user_id = $2
	`, repoID, userID, permission)
	return err
}

// Invitation methods

func (s *CollaboratorsStore) CreateInvitation(ctx context.Context, inv *collaborators.Invitation) error {
	now := time.Now()
	inv.CreatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO collaborator_invitations (node_id, repo_id, invitee_id, inviter_id, permissions, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "", inv.RepoID, inv.InviteeID, inv.InviterID, inv.Permissions, inv.CreatedAt).Scan(&inv.ID)
	if err != nil {
		return err
	}

	inv.NodeID = generateNodeID("RI", inv.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE collaborator_invitations SET node_id = $1 WHERE id = $2`, inv.NodeID, inv.ID)
	return err
}

func (s *CollaboratorsStore) GetInvitationByID(ctx context.Context, id int64) (*collaborators.Invitation, error) {
	inv := &collaborators.Invitation{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, invitee_id, inviter_id, permissions, expired, created_at
		FROM collaborator_invitations WHERE id = $1
	`, id).Scan(&inv.ID, &inv.NodeID, &inv.RepoID, &inv.InviteeID, &inv.InviterID,
		&inv.Permissions, &inv.Expired, &inv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return inv, err
}

func (s *CollaboratorsStore) UpdateInvitation(ctx context.Context, id int64, permission string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE collaborator_invitations SET permissions = $2 WHERE id = $1
	`, id, permission)
	return err
}

func (s *CollaboratorsStore) DeleteInvitation(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collaborator_invitations WHERE id = $1`, id)
	return err
}

func (s *CollaboratorsStore) ListInvitationsForRepo(ctx context.Context, repoID int64, opts *collaborators.ListOpts) ([]*collaborators.Invitation, error) {
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
		SELECT id, node_id, repo_id, invitee_id, inviter_id, permissions, expired, created_at
		FROM collaborator_invitations
		WHERE repo_id = $1 AND expired = FALSE
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanInvitations(rows)
}

func (s *CollaboratorsStore) ListInvitationsForUser(ctx context.Context, userID int64, opts *collaborators.ListOpts) ([]*collaborators.Invitation, error) {
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
		SELECT id, node_id, repo_id, invitee_id, inviter_id, permissions, expired, created_at
		FROM collaborator_invitations
		WHERE invitee_id = $1 AND expired = FALSE
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanInvitations(rows)
}

func (s *CollaboratorsStore) AcceptInvitation(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collaborator_invitations WHERE id = $1`, id)
	return err
}

func scanInvitations(rows *sql.Rows) ([]*collaborators.Invitation, error) {
	var list []*collaborators.Invitation
	for rows.Next() {
		inv := &collaborators.Invitation{}
		if err := rows.Scan(&inv.ID, &inv.NodeID, &inv.RepoID, &inv.InviteeID, &inv.InviterID,
			&inv.Permissions, &inv.Expired, &inv.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}
	return list, rows.Err()
}
