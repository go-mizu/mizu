package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/roles"
)

// RolesStore implements roles.Store.
type RolesStore struct {
	db *sql.DB
}

// NewRolesStore creates a new RolesStore.
func NewRolesStore(db *sql.DB) *RolesStore {
	return &RolesStore{db: db}
}

// Insert creates a new role.
func (s *RolesStore) Insert(ctx context.Context, r *roles.Role) error {
	query := `
		INSERT INTO roles (id, server_id, name, color, position, permissions, is_default, is_hoisted, is_mentionable, icon_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		r.ID, r.ServerID, r.Name, r.Color, r.Position, r.Permissions,
		r.IsDefault, r.IsHoisted, r.IsMentionable, r.IconURL, r.CreatedAt,
	)
	return err
}

// GetByID retrieves a role by ID.
func (s *RolesStore) GetByID(ctx context.Context, id string) (*roles.Role, error) {
	query := `
		SELECT id, server_id, name, color, position, permissions, is_default, is_hoisted, is_mentionable, icon_url, created_at
		FROM roles WHERE id = ?
	`
	r := &roles.Role{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.ServerID, &r.Name, &r.Color, &r.Position, &r.Permissions,
		&r.IsDefault, &r.IsHoisted, &r.IsMentionable, &r.IconURL, &r.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, roles.ErrNotFound
	}
	return r, err
}

// Update updates a role.
func (s *RolesStore) Update(ctx context.Context, id string, in *roles.UpdateIn) error {
	var sets []string
	var args []any

	if in.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Color != nil {
		sets = append(sets, "color = ?")
		args = append(args, *in.Color)
	}
	if in.Position != nil {
		sets = append(sets, "position = ?")
		args = append(args, *in.Position)
	}
	if in.Permissions != nil {
		sets = append(sets, "permissions = ?")
		args = append(args, *in.Permissions)
	}
	if in.IsHoisted != nil {
		sets = append(sets, "is_hoisted = ?")
		args = append(args, *in.IsHoisted)
	}
	if in.IsMentionable != nil {
		sets = append(sets, "is_mentionable = ?")
		args = append(args, *in.IsMentionable)
	}
	if in.IconURL != nil {
		sets = append(sets, "icon_url = ?")
		args = append(args, *in.IconURL)
	}

	if len(sets) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE roles SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a role.
func (s *RolesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM roles WHERE id = ?", id)
	return err
}

// ListByServer lists roles in a server.
func (s *RolesStore) ListByServer(ctx context.Context, serverID string) ([]*roles.Role, error) {
	query := `
		SELECT id, server_id, name, color, position, permissions, is_default, is_hoisted, is_mentionable, icon_url, created_at
		FROM roles
		WHERE server_id = ?
		ORDER BY position DESC
	`
	rows, err := s.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rs []*roles.Role
	for rows.Next() {
		r := &roles.Role{}
		if err := rows.Scan(
			&r.ID, &r.ServerID, &r.Name, &r.Color, &r.Position, &r.Permissions,
			&r.IsDefault, &r.IsHoisted, &r.IsMentionable, &r.IconURL, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	return rs, rows.Err()
}

// GetDefaultRole gets the @everyone role for a server.
func (s *RolesStore) GetDefaultRole(ctx context.Context, serverID string) (*roles.Role, error) {
	query := `
		SELECT id, server_id, name, color, position, permissions, is_default, is_hoisted, is_mentionable, icon_url, created_at
		FROM roles WHERE server_id = ? AND is_default = TRUE
	`
	r := &roles.Role{}
	err := s.db.QueryRowContext(ctx, query, serverID).Scan(
		&r.ID, &r.ServerID, &r.Name, &r.Color, &r.Position, &r.Permissions,
		&r.IsDefault, &r.IsHoisted, &r.IsMentionable, &r.IconURL, &r.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, roles.ErrNotFound
	}
	return r, err
}

// GetByIDs retrieves multiple roles by IDs.
func (s *RolesStore) GetByIDs(ctx context.Context, ids []string) ([]*roles.Role, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, server_id, name, color, position, permissions, is_default, is_hoisted, is_mentionable, icon_url, created_at
		FROM roles WHERE id IN (%s)
		ORDER BY position DESC
	`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rs []*roles.Role
	for rows.Next() {
		r := &roles.Role{}
		if err := rows.Scan(
			&r.ID, &r.ServerID, &r.Name, &r.Color, &r.Position, &r.Permissions,
			&r.IsDefault, &r.IsHoisted, &r.IsMentionable, &r.IconURL, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	return rs, rows.Err()
}

// UpdatePositions updates role positions.
func (s *RolesStore) UpdatePositions(ctx context.Context, serverID string, positions map[string]int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for id, pos := range positions {
		if _, err := tx.ExecContext(ctx,
			"UPDATE roles SET position = ? WHERE id = ? AND server_id = ?",
			pos, id, serverID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// InsertChannelPermission creates a channel permission override.
func (s *RolesStore) InsertChannelPermission(ctx context.Context, cp *roles.ChannelPermission) error {
	query := `
		INSERT INTO channel_permissions (channel_id, target_id, target_type, allow, deny)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (channel_id, target_id) DO UPDATE SET
			allow = EXCLUDED.allow,
			deny = EXCLUDED.deny
	`
	_, err := s.db.ExecContext(ctx, query,
		cp.ChannelID, cp.TargetID, cp.TargetType, cp.Allow, cp.Deny,
	)
	return err
}

// GetChannelPermissions gets permission overrides for a channel.
func (s *RolesStore) GetChannelPermissions(ctx context.Context, channelID string) ([]*roles.ChannelPermission, error) {
	query := `
		SELECT channel_id, target_id, target_type, allow, deny
		FROM channel_permissions WHERE channel_id = ?
	`
	rows, err := s.db.QueryContext(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cps []*roles.ChannelPermission
	for rows.Next() {
		cp := &roles.ChannelPermission{}
		if err := rows.Scan(&cp.ChannelID, &cp.TargetID, &cp.TargetType, &cp.Allow, &cp.Deny); err != nil {
			return nil, err
		}
		cps = append(cps, cp)
	}
	return cps, rows.Err()
}

// DeleteChannelPermission deletes a permission override.
func (s *RolesStore) DeleteChannelPermission(ctx context.Context, channelID, targetID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM channel_permissions WHERE channel_id = ? AND target_id = ?",
		channelID, targetID,
	)
	return err
}

// CreateDefaultRole creates the @everyone role for a server.
func (s *RolesStore) CreateDefaultRole(ctx context.Context, serverID string) (*roles.Role, error) {
	r := &roles.Role{
		ID:          serverID, // @everyone role ID equals server ID
		ServerID:    serverID,
		Name:        "@everyone",
		Position:    0,
		Permissions: roles.PermissionViewChannel | roles.PermissionSendMessages | roles.PermissionAddReactions,
		IsDefault:   true,
		CreatedAt:   time.Now(),
	}
	if err := s.Insert(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}
