package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/users"
)

// UsersStore handles user data access.
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new users store.
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

func (s *UsersStore) Create(ctx context.Context, u *users.User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, username, display_name, password_hash, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, u.ID, u.Email, u.Username, u.DisplayName, u.PasswordHash, u.AvatarURL, u.CreatedAt, u.UpdatedAt)
	return err
}

func (s *UsersStore) GetByID(ctx context.Context, id string) (*users.User, error) {
	u := &users.User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	u := &users.User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (s *UsersStore) GetByUsername(ctx context.Context, username string) (*users.User, error) {
	u := &users.User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE username = $1
	`, username).Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (s *UsersStore) Update(ctx context.Context, id string, in *users.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			display_name = COALESCE($2, display_name),
			avatar_url = COALESCE($3, avatar_url),
			updated_at = $4
		WHERE id = $1
	`, id, in.DisplayName, in.AvatarURL, time.Now())
	return err
}

// Session operations

func (s *UsersStore) CreateSession(ctx context.Context, sess *users.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`, sess.ID, sess.UserID, sess.ExpiresAt, sess.CreatedAt)
	return err
}

func (s *UsersStore) GetSession(ctx context.Context, id string) (*users.Session, error) {
	sess := &users.Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, created_at
		FROM sessions WHERE id = $1
	`, id).Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sess, err
}

func (s *UsersStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

func (s *UsersStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < $1`, time.Now())
	return err
}
