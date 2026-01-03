package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/workspace/feature/members"
)

// MembersStore implements members.Store.
type MembersStore struct {
	db *sql.DB
}

// NewMembersStore creates a new MembersStore.
func NewMembersStore(db *sql.DB) *MembersStore {
	return &MembersStore{db: db}
}

func (s *MembersStore) Create(ctx context.Context, m *members.Member) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO members (id, workspace_id, user_id, role, joined_at, invited_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`, m.ID, m.WorkspaceID, m.UserID, m.Role, m.JoinedAt, m.InvitedBy)
	return err
}

func (s *MembersStore) GetByID(ctx context.Context, id string) (*members.Member, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, user_id, role, joined_at, invited_by
		FROM members WHERE id = ?
	`, id)
	return s.scanMember(row)
}

func (s *MembersStore) GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID string) (*members.Member, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, user_id, role, joined_at, invited_by
		FROM members WHERE workspace_id = ? AND user_id = ?
	`, workspaceID, userID)
	return s.scanMember(row)
}

func (s *MembersStore) List(ctx context.Context, workspaceID string) ([]*members.Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, user_id, role, joined_at, invited_by
		FROM members WHERE workspace_id = ?
		ORDER BY joined_at
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*members.Member
	for rows.Next() {
		m, err := s.scanMemberFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (s *MembersStore) UpdateRole(ctx context.Context, id string, role members.Role) error {
	_, err := s.db.ExecContext(ctx, "UPDATE members SET role = ? WHERE id = ?", role, id)
	return err
}

func (s *MembersStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM members WHERE id = ?", id)
	return err
}

func (s *MembersStore) CreateInvite(ctx context.Context, inv *members.Invite) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO invites (id, workspace_id, email, role, token, expires_at, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, inv.ID, inv.WorkspaceID, inv.Email, inv.Role, inv.Token, inv.ExpiresAt, inv.CreatedBy, inv.CreatedAt)
	return err
}

func (s *MembersStore) GetInviteByToken(ctx context.Context, token string) (*members.Invite, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, email, role, token, expires_at, created_by, created_at
		FROM invites WHERE token = ?
	`, token)

	var inv members.Invite
	err := row.Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token, &inv.ExpiresAt, &inv.CreatedBy, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (s *MembersStore) DeleteInvite(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM invites WHERE id = ?", id)
	return err
}

func (s *MembersStore) ListPendingInvites(ctx context.Context, workspaceID string) ([]*members.Invite, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, email, role, token, expires_at, created_by, created_at
		FROM invites WHERE workspace_id = ? AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*members.Invite
	for rows.Next() {
		var inv members.Invite
		err := rows.Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token, &inv.ExpiresAt, &inv.CreatedBy, &inv.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, &inv)
	}
	return result, rows.Err()
}

func (s *MembersStore) scanMember(row *sql.Row) (*members.Member, error) {
	var m members.Member
	err := row.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt, &m.InvitedBy)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *MembersStore) scanMemberFromRows(rows *sql.Rows) (*members.Member, error) {
	var m members.Member
	err := rows.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt, &m.InvitedBy)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
