package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/members"
)

// MembersStore implements members.Store.
type MembersStore struct {
	db *sql.DB
}

// NewMembersStore creates a new MembersStore.
func NewMembersStore(db *sql.DB) *MembersStore {
	return &MembersStore{db: db}
}

// Insert creates a new member.
func (s *MembersStore) Insert(ctx context.Context, m *members.Member) error {
	query := `
		INSERT INTO members (server_id, user_id, nickname, avatar_url, is_muted, is_deafened, joined_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		m.ServerID, m.UserID, m.Nickname, m.AvatarURL, m.IsMuted, m.IsDeafened, m.JoinedAt,
	)
	return err
}

// Get retrieves a member.
func (s *MembersStore) Get(ctx context.Context, serverID, userID string) (*members.Member, error) {
	query := `
		SELECT server_id, user_id, nickname, avatar_url, is_muted, is_deafened, joined_at
		FROM members WHERE server_id = ? AND user_id = ?
	`
	m := &members.Member{}
	err := s.db.QueryRowContext(ctx, query, serverID, userID).Scan(
		&m.ServerID, &m.UserID, &m.Nickname, &m.AvatarURL, &m.IsMuted, &m.IsDeafened, &m.JoinedAt,
	)
	if err == sql.ErrNoRows {
		return nil, members.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Load roles
	m.RoleIDs, _ = s.getRoleIDs(ctx, serverID, userID)
	return m, nil
}

// Update updates a member.
func (s *MembersStore) Update(ctx context.Context, serverID, userID string, in *members.UpdateIn) error {
	if in.Nickname != nil {
		if _, err := s.db.ExecContext(ctx,
			"UPDATE members SET nickname = ? WHERE server_id = ? AND user_id = ?",
			*in.Nickname, serverID, userID,
		); err != nil {
			return err
		}
	}
	if in.AvatarURL != nil {
		if _, err := s.db.ExecContext(ctx,
			"UPDATE members SET avatar_url = ? WHERE server_id = ? AND user_id = ?",
			*in.AvatarURL, serverID, userID,
		); err != nil {
			return err
		}
	}
	if in.IsMuted != nil {
		if _, err := s.db.ExecContext(ctx,
			"UPDATE members SET is_muted = ? WHERE server_id = ? AND user_id = ?",
			*in.IsMuted, serverID, userID,
		); err != nil {
			return err
		}
	}
	if in.IsDeafened != nil {
		if _, err := s.db.ExecContext(ctx,
			"UPDATE members SET is_deafened = ? WHERE server_id = ? AND user_id = ?",
			*in.IsDeafened, serverID, userID,
		); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a member.
func (s *MembersStore) Delete(ctx context.Context, serverID, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM members WHERE server_id = ? AND user_id = ?", serverID, userID)
	return err
}

// List lists members in a server.
func (s *MembersStore) List(ctx context.Context, serverID string, limit, offset int) ([]*members.Member, error) {
	query := `
		SELECT server_id, user_id, nickname, avatar_url, is_muted, is_deafened, joined_at
		FROM members
		WHERE server_id = ?
		ORDER BY joined_at ASC
		LIMIT ? OFFSET ?
	`
	rows, err := s.db.QueryContext(ctx, query, serverID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mems []*members.Member
	for rows.Next() {
		m := &members.Member{}
		if err := rows.Scan(
			&m.ServerID, &m.UserID, &m.Nickname, &m.AvatarURL, &m.IsMuted, &m.IsDeafened, &m.JoinedAt,
		); err != nil {
			return nil, err
		}
		m.RoleIDs, _ = s.getRoleIDs(ctx, serverID, m.UserID)
		mems = append(mems, m)
	}
	return mems, rows.Err()
}

// Count counts members in a server.
func (s *MembersStore) Count(ctx context.Context, serverID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM members WHERE server_id = ?", serverID).Scan(&count)
	return count, err
}

// IsMember checks if a user is a member of a server.
func (s *MembersStore) IsMember(ctx context.Context, serverID, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM members WHERE server_id = ? AND user_id = ?)",
		serverID, userID,
	).Scan(&exists)
	return exists, err
}

// AddRole adds a role to a member.
func (s *MembersStore) AddRole(ctx context.Context, serverID, userID, roleID string) error {
	query := `INSERT INTO member_roles (server_id, user_id, role_id, created_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, serverID, userID, roleID, time.Now())
	return err
}

// RemoveRole removes a role from a member.
func (s *MembersStore) RemoveRole(ctx context.Context, serverID, userID, roleID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM member_roles WHERE server_id = ? AND user_id = ? AND role_id = ?",
		serverID, userID, roleID,
	)
	return err
}

// GetRoleIDs gets role IDs for a member.
func (s *MembersStore) getRoleIDs(ctx context.Context, serverID, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT role_id FROM member_roles WHERE server_id = ? AND user_id = ?",
		serverID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListByRole lists members with a specific role.
func (s *MembersStore) ListByRole(ctx context.Context, serverID, roleID string) ([]*members.Member, error) {
	query := `
		SELECT m.server_id, m.user_id, m.nickname, m.avatar_url, m.is_muted, m.is_deafened, m.joined_at
		FROM members m
		JOIN member_roles mr ON m.server_id = mr.server_id AND m.user_id = mr.user_id
		WHERE m.server_id = ? AND mr.role_id = ?
	`
	rows, err := s.db.QueryContext(ctx, query, serverID, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mems []*members.Member
	for rows.Next() {
		m := &members.Member{}
		if err := rows.Scan(
			&m.ServerID, &m.UserID, &m.Nickname, &m.AvatarURL, &m.IsMuted, &m.IsDeafened, &m.JoinedAt,
		); err != nil {
			return nil, err
		}
		mems = append(mems, m)
	}
	return mems, rows.Err()
}

// Search searches members by nickname or username.
func (s *MembersStore) Search(ctx context.Context, serverID, query string, limit int) ([]*members.Member, error) {
	searchQuery := `
		SELECT m.server_id, m.user_id, m.nickname, m.avatar_url, m.is_muted, m.is_deafened, m.joined_at
		FROM members m
		JOIN users u ON m.user_id = u.id
		WHERE m.server_id = ? AND (m.nickname ILIKE ? OR u.username ILIKE ? OR u.display_name ILIKE ?)
		LIMIT ?
	`
	pattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, searchQuery, serverID, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mems []*members.Member
	for rows.Next() {
		m := &members.Member{}
		if err := rows.Scan(
			&m.ServerID, &m.UserID, &m.Nickname, &m.AvatarURL, &m.IsMuted, &m.IsDeafened, &m.JoinedAt,
		); err != nil {
			return nil, err
		}
		mems = append(mems, m)
	}
	return mems, rows.Err()
}

// Ban bans a user from a server.
func (s *MembersStore) Ban(ctx context.Context, serverID, userID, bannedBy, reason string) error {
	// Remove member first
	s.Delete(ctx, serverID, userID)

	query := `INSERT INTO bans (server_id, user_id, reason, banned_by, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, serverID, userID, reason, bannedBy, time.Now())
	return err
}

// Unban removes a ban.
func (s *MembersStore) Unban(ctx context.Context, serverID, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM bans WHERE server_id = ? AND user_id = ?", serverID, userID)
	return err
}

// IsBanned checks if a user is banned.
func (s *MembersStore) IsBanned(ctx context.Context, serverID, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM bans WHERE server_id = ? AND user_id = ?)",
		serverID, userID,
	).Scan(&exists)
	return exists, err
}

// ListBans lists bans in a server.
func (s *MembersStore) ListBans(ctx context.Context, serverID string, limit, offset int) ([]*members.Ban, error) {
	query := `
		SELECT server_id, user_id, reason, banned_by, created_at
		FROM bans
		WHERE server_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := s.db.QueryContext(ctx, query, serverID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []*members.Ban
	for rows.Next() {
		b := &members.Ban{}
		if err := rows.Scan(&b.ServerID, &b.UserID, &b.Reason, &b.BannedBy, &b.CreatedAt); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, rows.Err()
}
