package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// User represents a user record.
type User struct {
	ID            string
	Email         string
	Name          string
	PasswordHash  string
	AvatarURL     sql.NullString
	StorageQuota  int64
	StorageUsed   int64
	IsAdmin       bool
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Session represents a session record.
type Session struct {
	ID           string
	UserID       string
	TokenHash    string
	IPAddress    sql.NullString
	UserAgent    sql.NullString
	LastActiveAt sql.NullTime
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// CreateUser inserts a new user.
func (s *Store) CreateUser(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, avatar_url, storage_quota, storage_used, is_admin, email_verified, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, u.ID, u.Email, u.Name, u.PasswordHash, u.AvatarURL, u.StorageQuota, u.StorageUsed, u.IsAdmin, u.EmailVerified, u.CreatedAt, u.UpdatedAt)
	return err
}

// GetUserByID retrieves a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, avatar_url, storage_quota, storage_used, is_admin, email_verified, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AvatarURL, &u.StorageQuota, &u.StorageUsed, &u.IsAdmin, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetUserByEmail retrieves a user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, avatar_url, storage_quota, storage_used, is_admin, email_verified, created_at, updated_at
		FROM users WHERE email = ?
	`, email).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AvatarURL, &u.StorageQuota, &u.StorageUsed, &u.IsAdmin, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET email = ?, name = ?, password_hash = ?, avatar_url = ?, storage_quota = ?, storage_used = ?, is_admin = ?, email_verified = ?, updated_at = ?
		WHERE id = ?
	`, u.Email, u.Name, u.PasswordHash, u.AvatarURL, u.StorageQuota, u.StorageUsed, u.IsAdmin, u.EmailVerified, u.UpdatedAt, u.ID)
	return err
}

// DeleteUser deletes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

// ListUsers lists all users.
func (s *Store) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, name, password_hash, avatar_url, storage_quota, storage_used, is_admin, email_verified, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AvatarURL, &u.StorageQuota, &u.StorageUsed, &u.IsAdmin, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// CreateSession inserts a new session.
func (s *Store) CreateSession(ctx context.Context, sess *Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, ip_address, user_agent, last_active_at, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, sess.ID, sess.UserID, sess.TokenHash, sess.IPAddress, sess.UserAgent, sess.LastActiveAt, sess.ExpiresAt, sess.CreatedAt)
	return err
}

// GetSessionByID retrieves a session by ID.
func (s *Store) GetSessionByID(ctx context.Context, id string) (*Session, error) {
	sess := &Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, ip_address, user_agent, last_active_at, expires_at, created_at
		FROM sessions WHERE id = ?
	`, id).Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.IPAddress, &sess.UserAgent, &sess.LastActiveAt, &sess.ExpiresAt, &sess.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sess, err
}

// GetSessionByToken retrieves a session by token hash.
func (s *Store) GetSessionByToken(ctx context.Context, tokenHash string) (*Session, error) {
	sess := &Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, ip_address, user_agent, last_active_at, expires_at, created_at
		FROM sessions WHERE token_hash = ? AND expires_at > CURRENT_TIMESTAMP
	`, tokenHash).Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.IPAddress, &sess.UserAgent, &sess.LastActiveAt, &sess.ExpiresAt, &sess.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sess, err
}

// UpdateSessionActivity updates the last active time of a session.
func (s *Store) UpdateSessionActivity(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET last_active_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// DeleteSession deletes a session by ID.
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteUserSessions deletes all sessions for a user.
func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

// ListUserSessions lists all sessions for a user.
func (s *Store) ListUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, token_hash, ip_address, user_agent, last_active_at, expires_at, created_at
		FROM sessions WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		sess := &Session{}
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.IPAddress, &sess.UserAgent, &sess.LastActiveAt, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// CleanupExpiredSessions removes expired sessions.
func (s *Store) CleanupExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	return err
}

// UpdateUserStorageUsed updates the storage_used for a user.
func (s *Store) UpdateUserStorageUsed(ctx context.Context, userID string, delta int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET storage_used = storage_used + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, delta, userID)
	return err
}
