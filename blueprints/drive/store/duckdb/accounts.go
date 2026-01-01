package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/pkg/mime"
)

// AccountsStore handles account persistence.
type AccountsStore struct {
	db *sql.DB
}

// Create inserts a new account.
func (s *AccountsStore) Create(ctx context.Context, a *accounts.Account) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, username, email, password_hash, display_name, avatar_url,
			storage_quota, storage_used, is_admin, is_suspended, preferences, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Username, a.Email, a.PasswordHash, a.DisplayName, a.AvatarURL,
		a.StorageQuota, a.StorageUsed, a.IsAdmin, a.IsSuspended, a.Preferences, a.CreatedAt, a.UpdatedAt)
	return err
}

// GetByID retrieves an account by ID.
func (s *AccountsStore) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
	a := &accounts.Account{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, avatar_url,
			storage_quota, storage_used, is_admin, is_suspended, preferences, created_at, updated_at
		FROM accounts WHERE id = ?`, id).Scan(
		&a.ID, &a.Username, &a.Email, &a.PasswordHash, &a.DisplayName, &a.AvatarURL,
		&a.StorageQuota, &a.StorageUsed, &a.IsAdmin, &a.IsSuspended, &a.Preferences, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return a, err
}

// GetByUsername retrieves an account by username.
func (s *AccountsStore) GetByUsername(ctx context.Context, username string) (*accounts.Account, error) {
	a := &accounts.Account{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, avatar_url,
			storage_quota, storage_used, is_admin, is_suspended, preferences, created_at, updated_at
		FROM accounts WHERE username = ?`, username).Scan(
		&a.ID, &a.Username, &a.Email, &a.PasswordHash, &a.DisplayName, &a.AvatarURL,
		&a.StorageQuota, &a.StorageUsed, &a.IsAdmin, &a.IsSuspended, &a.Preferences, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return a, err
}

// GetByEmail retrieves an account by email.
func (s *AccountsStore) GetByEmail(ctx context.Context, email string) (*accounts.Account, error) {
	a := &accounts.Account{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, avatar_url,
			storage_quota, storage_used, is_admin, is_suspended, preferences, created_at, updated_at
		FROM accounts WHERE email = ?`, email).Scan(
		&a.ID, &a.Username, &a.Email, &a.PasswordHash, &a.DisplayName, &a.AvatarURL,
		&a.StorageQuota, &a.StorageUsed, &a.IsAdmin, &a.IsSuspended, &a.Preferences, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return a, err
}

// Update updates account fields.
func (s *AccountsStore) Update(ctx context.Context, id string, in *accounts.UpdateIn) error {
	if in.DisplayName != nil {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE accounts SET display_name = ?, updated_at = ? WHERE id = ?`,
			*in.DisplayName, time.Now(), id); err != nil {
			return err
		}
	}
	if in.AvatarURL != nil {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE accounts SET avatar_url = ?, updated_at = ? WHERE id = ?`,
			*in.AvatarURL, time.Now(), id); err != nil {
			return err
		}
	}
	return nil
}

// UpdatePassword updates the password hash.
func (s *AccountsStore) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE accounts SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, time.Now(), id)
	return err
}

// UpdateStorageUsed adjusts storage usage.
func (s *AccountsStore) UpdateStorageUsed(ctx context.Context, id string, delta int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE accounts SET storage_used = storage_used + ?, updated_at = ? WHERE id = ?`,
		delta, time.Now(), id)
	return err
}

// CreateSession creates a new session.
func (s *AccountsStore) CreateSession(ctx context.Context, sess *accounts.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, account_id, token, user_agent, ip_address, last_used, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.AccountID, sess.Token, sess.UserAgent, sess.IPAddress, sess.LastUsed, sess.ExpiresAt, sess.CreatedAt)
	return err
}

// GetSessionByToken retrieves a session by token.
func (s *AccountsStore) GetSessionByToken(ctx context.Context, token string) (*accounts.Session, error) {
	sess := &accounts.Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, token, user_agent, ip_address, last_used, expires_at, created_at
		FROM sessions WHERE token = ?`, token).Scan(
		&sess.ID, &sess.AccountID, &sess.Token, &sess.UserAgent, &sess.IPAddress, &sess.LastUsed, &sess.ExpiresAt, &sess.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, accounts.ErrSessionNotFound
	}
	return sess, err
}

// UpdateSessionLastUsed updates the last used timestamp.
func (s *AccountsStore) UpdateSessionLastUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET last_used = ? WHERE id = ?`, time.Now(), id)
	return err
}

// DeleteSession deletes a session.
func (s *AccountsStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteSessionsByAccount deletes all sessions for an account.
func (s *AccountsStore) DeleteSessionsByAccount(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE account_id = ?`, accountID)
	return err
}

// ListSessionsByAccount lists all sessions for an account.
func (s *AccountsStore) ListSessionsByAccount(ctx context.Context, accountID string) ([]*accounts.Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, token, user_agent, ip_address, last_used, expires_at, created_at
		FROM sessions WHERE account_id = ? ORDER BY last_used DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*accounts.Session
	for rows.Next() {
		sess := &accounts.Session{}
		if err := rows.Scan(&sess.ID, &sess.AccountID, &sess.Token, &sess.UserAgent, &sess.IPAddress,
			&sess.LastUsed, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// GetStorageByCategory returns storage usage by file category.
func (s *AccountsStore) GetStorageByCategory(ctx context.Context, accountID string) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT mime_type, SUM(size) as total
		FROM files WHERE owner_id = ? AND trashed = FALSE
		GROUP BY mime_type`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var mimeType string
		var total int64
		if err := rows.Scan(&mimeType, &total); err != nil {
			return nil, err
		}
		category := mime.Category(mimeType)
		result[category] += total
	}
	return result, rows.Err()
}
