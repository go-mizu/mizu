package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/servers"
)

// ServersStore implements servers.Store.
type ServersStore struct {
	db *sql.DB
}

// NewServersStore creates a new ServersStore.
func NewServersStore(db *sql.DB) *ServersStore {
	return &ServersStore{db: db}
}

// Insert creates a new server.
func (s *ServersStore) Insert(ctx context.Context, srv *servers.Server) error {
	query := `
		INSERT INTO servers (id, name, description, icon_url, banner_url, owner_id, is_public, invite_code, member_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		srv.ID, srv.Name, srv.Description, srv.IconURL, srv.BannerURL,
		srv.OwnerID, srv.IsPublic, srv.InviteCode, srv.MemberCount,
		srv.CreatedAt, srv.UpdatedAt,
	)
	return err
}

// GetByID retrieves a server by ID.
func (s *ServersStore) GetByID(ctx context.Context, id string) (*servers.Server, error) {
	query := `
		SELECT id, name, description, icon_url, banner_url, owner_id, is_public, is_verified, invite_code, default_channel, member_count, created_at, updated_at
		FROM servers WHERE id = ?
	`
	srv, err := scanServer(s.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, servers.ErrNotFound
	}
	return srv, err
}

// scanServer scans a server row, handling nullable columns.
func scanServer(row interface{ Scan(...any) error }) (*servers.Server, error) {
	srv := &servers.Server{}
	var description, iconURL, bannerURL, inviteCode, defaultChannel sql.NullString
	err := row.Scan(
		&srv.ID, &srv.Name, &description, &iconURL, &bannerURL,
		&srv.OwnerID, &srv.IsPublic, &srv.IsVerified, &inviteCode, &defaultChannel,
		&srv.MemberCount, &srv.CreatedAt, &srv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	srv.Description = description.String
	srv.IconURL = iconURL.String
	srv.BannerURL = bannerURL.String
	srv.InviteCode = inviteCode.String
	srv.DefaultChannel = defaultChannel.String
	return srv, nil
}

// scanServers scans multiple server rows.
func scanServers(rows *sql.Rows) ([]*servers.Server, error) {
	var srvs []*servers.Server
	for rows.Next() {
		srv := &servers.Server{}
		var description, iconURL, bannerURL, inviteCode, defaultChannel sql.NullString
		if err := rows.Scan(
			&srv.ID, &srv.Name, &description, &iconURL, &bannerURL,
			&srv.OwnerID, &srv.IsPublic, &srv.IsVerified, &inviteCode, &defaultChannel,
			&srv.MemberCount, &srv.CreatedAt, &srv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		srv.Description = description.String
		srv.IconURL = iconURL.String
		srv.BannerURL = bannerURL.String
		srv.InviteCode = inviteCode.String
		srv.DefaultChannel = defaultChannel.String
		srvs = append(srvs, srv)
	}
	return srvs, rows.Err()
}

// GetByInviteCode retrieves a server by invite code.
func (s *ServersStore) GetByInviteCode(ctx context.Context, code string) (*servers.Server, error) {
	query := `
		SELECT id, name, description, icon_url, banner_url, owner_id, is_public, is_verified, invite_code, default_channel, member_count, created_at, updated_at
		FROM servers WHERE invite_code = ?
	`
	srv, err := scanServer(s.db.QueryRowContext(ctx, query, code))
	if err == sql.ErrNoRows {
		return nil, servers.ErrNotFound
	}
	return srv, err
}

// Update updates a server.
func (s *ServersStore) Update(ctx context.Context, id string, in *servers.UpdateIn) error {
	var sets []string
	var args []any

	if in.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *in.Description)
	}
	if in.IconURL != nil {
		sets = append(sets, "icon_url = ?")
		args = append(args, *in.IconURL)
	}
	if in.BannerURL != nil {
		sets = append(sets, "banner_url = ?")
		args = append(args, *in.BannerURL)
	}
	if in.IsPublic != nil {
		sets = append(sets, "is_public = ?")
		args = append(args, *in.IsPublic)
	}
	if in.DefaultChannel != nil {
		sets = append(sets, "default_channel = ?")
		args = append(args, *in.DefaultChannel)
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE servers SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a server.
func (s *ServersStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM servers WHERE id = ?", id)
	return err
}

// ListByUser lists servers a user is a member of.
func (s *ServersStore) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*servers.Server, error) {
	query := `
		SELECT s.id, s.name, s.description, s.icon_url, s.banner_url, s.owner_id, s.is_public, s.is_verified, s.invite_code, s.default_channel, s.member_count, s.created_at, s.updated_at
		FROM servers s
		JOIN members m ON s.id = m.server_id
		WHERE m.user_id = ?
		ORDER BY s.name
		LIMIT ? OFFSET ?
	`
	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanServers(rows)
}

// ListPublic lists public servers.
func (s *ServersStore) ListPublic(ctx context.Context, limit, offset int) ([]*servers.Server, error) {
	query := `
		SELECT id, name, description, icon_url, banner_url, owner_id, is_public, is_verified, invite_code, default_channel, member_count, created_at, updated_at
		FROM servers
		WHERE is_public = TRUE
		ORDER BY member_count DESC
		LIMIT ? OFFSET ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanServers(rows)
}

// UpdateMemberCount updates the member count.
func (s *ServersStore) UpdateMemberCount(ctx context.Context, serverID string, delta int) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE servers SET member_count = member_count + ?, updated_at = ? WHERE id = ?",
		delta, time.Now(), serverID,
	)
	return err
}

// SetDefaultChannel sets the default channel.
func (s *ServersStore) SetDefaultChannel(ctx context.Context, serverID, channelID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE servers SET default_channel = ?, updated_at = ? WHERE id = ?",
		channelID, time.Now(), serverID,
	)
	return err
}

// Search searches for public servers.
func (s *ServersStore) Search(ctx context.Context, query string, limit int) ([]*servers.Server, error) {
	searchQuery := `
		SELECT id, name, description, icon_url, banner_url, owner_id, is_public, is_verified, invite_code, default_channel, member_count, created_at, updated_at
		FROM servers
		WHERE is_public = TRUE AND (name ILIKE ? OR description ILIKE ?)
		ORDER BY member_count DESC
		LIMIT ?
	`
	pattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, searchQuery, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanServers(rows)
}
