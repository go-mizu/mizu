package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/table/feature/users"
)

// UsersStore provides SQLite-based user storage.
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new users store.
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

// Create creates a new user.
func (s *UsersStore) Create(ctx context.Context, user *users.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, avatar_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Email, user.Name, user.PasswordHash, user.AvatarURL, user.CreatedAt, user.UpdatedAt)
	return err
}

// GetByID retrieves a user by ID.
func (s *UsersStore) GetByID(ctx context.Context, id string) (*users.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE id = ?
	`, id)
	return s.scanUser(row)
}

// GetByEmail retrieves a user by email (case-insensitive).
func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE LOWER(email) = LOWER(?)
	`, email)
	return s.scanUser(row)
}

// Update updates a user.
func (s *UsersStore) Update(ctx context.Context, user *users.User) error {
	user.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			email = ?, name = ?, password_hash = ?, avatar_url = ?, updated_at = ?
		WHERE id = ?
	`, user.Email, user.Name, user.PasswordHash, user.AvatarURL, user.UpdatedAt, user.ID)
	return err
}

// Delete deletes a user.
func (s *UsersStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *UsersStore) scanUser(row *sql.Row) (*users.User, error) {
	user := &users.User{}
	var avatarURL sql.NullString

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&avatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, users.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}

	return user, nil
}
