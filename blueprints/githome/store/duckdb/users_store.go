package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

// UsersStore implements users.Store
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new users store
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

// Create creates a new user
func (s *UsersStore) Create(ctx context.Context, u *users.User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, username, email, password_hash, full_name, avatar_url, bio, location, website, company, is_admin, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, u.ID, u.Username, u.Email, u.PasswordHash, u.FullName, u.AvatarURL, u.Bio, u.Location, u.Website, u.Company, u.IsAdmin, u.IsActive, u.CreatedAt, u.UpdatedAt)
	return err
}

// GetByID retrieves a user by ID
func (s *UsersStore) GetByID(ctx context.Context, id string) (*users.User, error) {
	u := &users.User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, full_name, avatar_url, bio, location, website, company, is_admin, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FullName, &u.AvatarURL, &u.Bio, &u.Location, &u.Website, &u.Company, &u.IsAdmin, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetByUsername retrieves a user by username
func (s *UsersStore) GetByUsername(ctx context.Context, username string) (*users.User, error) {
	u := &users.User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, full_name, avatar_url, bio, location, website, company, is_admin, is_active, created_at, updated_at
		FROM users WHERE username = $1
	`, username).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FullName, &u.AvatarURL, &u.Bio, &u.Location, &u.Website, &u.Company, &u.IsAdmin, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetByEmail retrieves a user by email
func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	u := &users.User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, full_name, avatar_url, bio, location, website, company, is_admin, is_active, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FullName, &u.AvatarURL, &u.Bio, &u.Location, &u.Website, &u.Company, &u.IsAdmin, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// Update updates a user
func (s *UsersStore) Update(ctx context.Context, u *users.User) error {
	u.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET username = $2, email = $3, password_hash = $4, full_name = $5, avatar_url = $6, bio = $7, location = $8, website = $9, company = $10, is_admin = $11, is_active = $12, updated_at = $13
		WHERE id = $1
	`, u.ID, u.Username, u.Email, u.PasswordHash, u.FullName, u.AvatarURL, u.Bio, u.Location, u.Website, u.Company, u.IsAdmin, u.IsActive, u.UpdatedAt)
	return err
}

// Delete deletes a user
func (s *UsersStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// List lists all users with pagination
func (s *UsersStore) List(ctx context.Context, limit, offset int) ([]*users.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, email, password_hash, full_name, avatar_url, bio, location, website, company, is_admin, is_active, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*users.User
	for rows.Next() {
		u := &users.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FullName, &u.AvatarURL, &u.Bio, &u.Location, &u.Website, &u.Company, &u.IsAdmin, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

// CreateSession creates a new session
func (s *UsersStore) CreateSession(ctx context.Context, sess *users.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, user_agent, ip_address, created_at, last_active_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, sess.ID, sess.UserID, sess.ExpiresAt, sess.UserAgent, sess.IPAddress, sess.CreatedAt, sess.LastActiveAt)
	return err
}

// GetSession retrieves a session by ID
func (s *UsersStore) GetSession(ctx context.Context, id string) (*users.Session, error) {
	sess := &users.Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, user_agent, ip_address, created_at, last_active_at
		FROM sessions WHERE id = $1
	`, id).Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt, &sess.UserAgent, &sess.IPAddress, &sess.CreatedAt, &sess.LastActiveAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sess, err
}

// DeleteSession deletes a session
func (s *UsersStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

// DeleteUserSessions deletes all sessions for a user
func (s *UsersStore) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

// DeleteExpiredSessions deletes all expired sessions
func (s *UsersStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`)
	return err
}

// UpdateSessionActivity updates the last active time of a session
func (s *UsersStore) UpdateSessionActivity(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET last_active_at = CURRENT_TIMESTAMP WHERE id = $1`, id)
	return err
}
