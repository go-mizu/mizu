package postgres

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
)

// UsersStore implements users.Store.
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new users store.
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

// Create creates a new user.
func (s *UsersStore) Create(ctx context.Context, user *users.User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password, avatar, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.Email, user.Name, user.Password, user.Avatar, user.CreatedAt, user.UpdatedAt)
	return err
}

// GetByID retrieves a user by ID.
func (s *UsersStore) GetByID(ctx context.Context, id string) (*users.User, error) {
	user := &users.User{}
	var avatar sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password, avatar, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.Password, &avatar, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, users.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if avatar.Valid {
		user.Avatar = avatar.String
	}
	return user, nil
}

// GetByEmail retrieves a user by email.
func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	user := &users.User{}
	var avatar sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password, avatar, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.Password, &avatar, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, users.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if avatar.Valid {
		user.Avatar = avatar.String
	}
	return user, nil
}

// Update updates a user.
func (s *UsersStore) Update(ctx context.Context, user *users.User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET name = $1, avatar = $2, updated_at = $3
		WHERE id = $4
	`, user.Name, user.Avatar, user.UpdatedAt, user.ID)
	return err
}
