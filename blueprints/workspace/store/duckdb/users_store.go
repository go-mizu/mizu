package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// UsersStore implements users.Store.
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new UsersStore.
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

func (s *UsersStore) Create(ctx context.Context, u *users.User) error {
	settingsJSON, _ := json.Marshal(u.Settings)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, avatar_url, password_hash, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, u.ID, u.Email, u.Name, u.AvatarURL, u.PasswordHash, string(settingsJSON), u.CreatedAt, u.UpdatedAt)
	return err
}

func (s *UsersStore) GetByID(ctx context.Context, id string) (*users.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, avatar_url, password_hash, CAST(settings AS VARCHAR), created_at, updated_at
		FROM users WHERE id = ?
	`, id)
	return s.scanUser(row)
}

func (s *UsersStore) GetByIDs(ctx context.Context, ids []string) ([]*users.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, email, name, avatar_url, password_hash, CAST(settings AS VARCHAR), created_at, updated_at
		FROM users WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*users.User
	for rows.Next() {
		u, err := s.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, avatar_url, password_hash, CAST(settings AS VARCHAR), created_at, updated_at
		FROM users WHERE email = ?
	`, email)
	return s.scanUser(row)
}

func (s *UsersStore) Update(ctx context.Context, id string, in *users.UpdateIn) error {
	sets := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{}

	if in.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.AvatarURL != nil {
		sets = append(sets, "avatar_url = ?")
		args = append(args, *in.AvatarURL)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *UsersStore) UpdateSettings(ctx context.Context, id string, settings users.Settings) error {
	settingsJSON, _ := json.Marshal(settings)
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET settings = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, string(settingsJSON), id)
	return err
}

func (s *UsersStore) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, passwordHash, id)
	return err
}

func (s *UsersStore) CreateSession(ctx context.Context, sess *users.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, sess.ID, sess.UserID, sess.ExpiresAt, sess.CreatedAt)
	return err
}

func (s *UsersStore) GetSession(ctx context.Context, id string) (*users.Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?
	`, id)

	var sess users.Session
	err := row.Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *UsersStore) GetUserBySession(ctx context.Context, sessionID string) (*users.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.avatar_url, u.password_hash, CAST(u.settings AS VARCHAR), u.created_at, u.updated_at
		FROM users u
		JOIN sessions s ON u.id = s.user_id
		WHERE s.id = ?
	`, sessionID)
	return s.scanUser(row)
}

func (s *UsersStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (s *UsersStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

func (s *UsersStore) scanUser(row *sql.Row) (*users.User, error) {
	var u users.User
	var settingsJSON string
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.PasswordHash, &settingsJSON, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(settingsJSON), &u.Settings)
	return &u, nil
}

func (s *UsersStore) scanUserFromRows(rows *sql.Rows) (*users.User, error) {
	var u users.User
	var settingsJSON string
	err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.PasswordHash, &settingsJSON, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(settingsJSON), &u.Settings)
	return &u, nil
}
