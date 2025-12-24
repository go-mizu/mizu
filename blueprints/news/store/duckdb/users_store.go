package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/news/feature/users"
)

// UsersStore implements users.Store.
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new users store.
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

// GetByID retrieves a user by ID.
func (s *UsersStore) GetByID(ctx context.Context, id string) (*users.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, about, karma, is_admin, created_at
		FROM users WHERE id = $1
	`, id))
}

// GetByIDs retrieves multiple users by their IDs.
func (s *UsersStore) GetByIDs(ctx context.Context, ids []string) (map[string]*users.User, error) {
	if len(ids) == 0 {
		return make(map[string]*users.User), nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := `
		SELECT id, username, email, password_hash, about, karma, is_admin, created_at
		FROM users WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*users.User)
	for rows.Next() {
		user, err := s.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		result[user.ID] = user
	}
	return result, rows.Err()
}

// GetByUsername retrieves a user by username.
func (s *UsersStore) GetByUsername(ctx context.Context, username string) (*users.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, about, karma, is_admin, created_at
		FROM users WHERE LOWER(username) = LOWER($1)
	`, username))
}

// GetByEmail retrieves a user by email.
func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, about, karma, is_admin, created_at
		FROM users WHERE LOWER(email) = LOWER($1)
	`, email))
}

// GetSessionByToken retrieves a session by token.
func (s *UsersStore) GetSessionByToken(ctx context.Context, token string) (*users.Session, error) {
	session := &users.Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions WHERE token = $1
	`, token).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, users.ErrSessionExpired
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

// List lists users.
func (s *UsersStore) List(ctx context.Context, limit, offset int) ([]*users.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, email, password_hash, about, karma, is_admin, created_at
		FROM users
		ORDER BY karma DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*users.User
	for rows.Next() {
		user, err := s.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, user)
	}
	return result, rows.Err()
}

func (s *UsersStore) scanUser(row *sql.Row) (*users.User, error) {
	user := &users.User{}
	var about sql.NullString

	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&about, &user.Karma, &user.IsAdmin, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, users.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if about.Valid {
		user.About = about.String
	}

	return user, nil
}

func (s *UsersStore) scanUserFromRows(rows *sql.Rows) (*users.User, error) {
	user := &users.User{}
	var about sql.NullString

	err := rows.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&about, &user.Karma, &user.IsAdmin, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	if about.Valid {
		user.About = about.String
	}

	return user, nil
}

// Create creates a new user.
func (s *UsersStore) Create(ctx context.Context, user *users.User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, username, email, password_hash, about, karma, is_admin, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.ID, user.Username, user.Email, user.PasswordHash, user.About, user.Karma, user.IsAdmin, user.CreatedAt)
	return err
}

// CreateSession creates a new session.
func (s *UsersStore) CreateSession(ctx context.Context, session *users.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt)
	return err
}

// DeleteSession deletes a session by token.
func (s *UsersStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}
