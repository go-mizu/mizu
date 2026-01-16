package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

// SettingsStoreImpl implements store.SettingsStore.
type SettingsStoreImpl struct {
	db *sql.DB
}

// CreateToken creates a new API token.
func (s *SettingsStoreImpl) CreateToken(ctx context.Context, token *store.APIToken) error {
	permissions, _ := json.Marshal(token.Permissions)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO api_tokens (id, name, token_hash, token_preview, permissions, not_before, expires_at, last_used_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.Name, token.TokenHash, token.TokenPreview, string(permissions),
		token.NotBefore, token.ExpiresAt, token.LastUsedAt, token.CreatedAt)
	return err
}

// GetToken retrieves a token by ID.
func (s *SettingsStoreImpl) GetToken(ctx context.Context, id string) (*store.APIToken, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, token_hash, token_preview, permissions, not_before, expires_at, last_used_at, created_at
		FROM api_tokens WHERE id = ?`, id)
	return s.scanToken(row)
}

// GetTokenByHash retrieves a token by hash.
func (s *SettingsStoreImpl) GetTokenByHash(ctx context.Context, tokenHash string) (*store.APIToken, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, token_hash, token_preview, permissions, not_before, expires_at, last_used_at, created_at
		FROM api_tokens WHERE token_hash = ?`, tokenHash)
	return s.scanToken(row)
}

// ListTokens lists all API tokens.
func (s *SettingsStoreImpl) ListTokens(ctx context.Context) ([]*store.APIToken, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, token_hash, token_preview, permissions, not_before, expires_at, last_used_at, created_at
		FROM api_tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*store.APIToken
	for rows.Next() {
		token, err := s.scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

// UpdateTokenLastUsed updates the last used timestamp.
func (s *SettingsStoreImpl) UpdateTokenLastUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, time.Now(), id)
	return err
}

// DeleteToken deletes an API token.
func (s *SettingsStoreImpl) DeleteToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM api_tokens WHERE id = ?`, id)
	return err
}

// CreateMember creates a new team member.
func (s *SettingsStoreImpl) CreateMember(ctx context.Context, member *store.TeamMember) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO team_members (id, user_id, email, name, role, status, invited_by, invited_at, accepted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		member.ID, member.UserID, member.Email, member.Name, member.Role, member.Status,
		member.InvitedBy, member.InvitedAt, member.AcceptedAt)
	return err
}

// GetMember retrieves a team member by ID.
func (s *SettingsStoreImpl) GetMember(ctx context.Context, id string) (*store.TeamMember, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, email, name, role, status, invited_by, invited_at, accepted_at
		FROM team_members WHERE id = ?`, id)
	return s.scanMember(row)
}

// GetMemberByEmail retrieves a team member by email.
func (s *SettingsStoreImpl) GetMemberByEmail(ctx context.Context, email string) (*store.TeamMember, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, email, name, role, status, invited_by, invited_at, accepted_at
		FROM team_members WHERE email = ?`, email)
	return s.scanMember(row)
}

// ListMembers lists all team members.
func (s *SettingsStoreImpl) ListMembers(ctx context.Context) ([]*store.TeamMember, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, email, name, role, status, invited_by, invited_at, accepted_at
		FROM team_members ORDER BY invited_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*store.TeamMember
	for rows.Next() {
		member, err := s.scanMember(rows)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

// UpdateMember updates a team member.
func (s *SettingsStoreImpl) UpdateMember(ctx context.Context, member *store.TeamMember) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE team_members SET name = ?, role = ?, status = ?, accepted_at = ? WHERE id = ?`,
		member.Name, member.Role, member.Status, member.AcceptedAt, member.ID)
	return err
}

// DeleteMember deletes a team member.
func (s *SettingsStoreImpl) DeleteMember(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM team_members WHERE id = ?`, id)
	return err
}

// WriteAuditLog writes an audit log entry.
func (s *SettingsStoreImpl) WriteAuditLog(ctx context.Context, log *store.AuditLog) error {
	metadata, _ := json.Marshal(log.Metadata)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_logs (id, actor_id, actor_email, action, resource_type, resource_id, metadata, ip_address, user_agent, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.ActorID, log.ActorEmail, log.Action, log.ResourceType, log.ResourceID,
		string(metadata), log.IPAddress, log.UserAgent, log.Timestamp)
	return err
}

// QueryAuditLogs queries audit logs with filtering.
func (s *SettingsStoreImpl) QueryAuditLogs(ctx context.Context, actorID string, resourceType string, limit, offset int) ([]*store.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, actor_id, actor_email, action, resource_type, resource_id, metadata, ip_address, user_agent, timestamp
		FROM audit_logs WHERE 1=1`
	args := []any{}

	if actorID != "" {
		query += ` AND actor_id = ?`
		args = append(args, actorID)
	}
	if resourceType != "" {
		query += ` AND resource_type = ?`
		args = append(args, resourceType)
	}

	query += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*store.AuditLog
	for rows.Next() {
		log, err := s.scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (s *SettingsStoreImpl) scanToken(row scanner) (*store.APIToken, error) {
	var token store.APIToken
	var permissions string
	var notBefore, expiresAt, lastUsedAt sql.NullTime
	if err := row.Scan(&token.ID, &token.Name, &token.TokenHash, &token.TokenPreview, &permissions,
		&notBefore, &expiresAt, &lastUsedAt, &token.CreatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(permissions), &token.Permissions)
	if notBefore.Valid {
		token.NotBefore = &notBefore.Time
	}
	if expiresAt.Valid {
		token.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		token.LastUsedAt = &lastUsedAt.Time
	}
	return &token, nil
}

func (s *SettingsStoreImpl) scanMember(row scanner) (*store.TeamMember, error) {
	var member store.TeamMember
	var userID, name, invitedBy sql.NullString
	var acceptedAt sql.NullTime
	if err := row.Scan(&member.ID, &userID, &member.Email, &name, &member.Role, &member.Status,
		&invitedBy, &member.InvitedAt, &acceptedAt); err != nil {
		return nil, err
	}
	member.UserID = userID.String
	member.Name = name.String
	member.InvitedBy = invitedBy.String
	if acceptedAt.Valid {
		member.AcceptedAt = &acceptedAt.Time
	}
	return &member, nil
}

func (s *SettingsStoreImpl) scanAuditLog(row scanner) (*store.AuditLog, error) {
	var log store.AuditLog
	var actorID, actorEmail, resourceID, ipAddress, userAgent, metadata sql.NullString
	if err := row.Scan(&log.ID, &actorID, &actorEmail, &log.Action, &log.ResourceType,
		&resourceID, &metadata, &ipAddress, &userAgent, &log.Timestamp); err != nil {
		return nil, err
	}
	log.ActorID = actorID.String
	log.ActorEmail = actorEmail.String
	log.ResourceID = resourceID.String
	log.IPAddress = ipAddress.String
	log.UserAgent = userAgent.String
	if metadata.Valid && metadata.String != "" {
		json.Unmarshal([]byte(metadata.String), &log.Metadata)
	}
	return &log, nil
}
