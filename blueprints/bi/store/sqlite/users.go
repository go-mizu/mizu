package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// UserStore implements store.UserStore.
type UserStore struct {
	db *sql.DB
}

func (s *UserStore) Create(ctx context.Context, user *store.User) error {
	if user.ID == "" {
		user.ID = generateID()
	}
	user.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, user.ID, user.Email, user.Name, user.PasswordHash, user.Role, user.CreatedAt)
	return err
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*store.User, error) {
	var user store.User
	var lastLogin sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, role, created_at, last_login
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	return &user, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	var user store.User
	var lastLogin sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, role, created_at, last_login
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	return &user, nil
}

func (s *UserStore) List(ctx context.Context) ([]*store.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, name, password_hash, role, created_at, last_login
		FROM users ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.User
	for rows.Next() {
		var user store.User
		var lastLogin sql.NullTime
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin); err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			user.LastLogin = lastLogin.Time
		}
		result = append(result, &user)
	}
	return result, rows.Err()
}

func (s *UserStore) Update(ctx context.Context, user *store.User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET email=?, name=?, password_hash=?, role=?
		WHERE id=?
	`, user.Email, user.Name, user.PasswordHash, user.Role, user.ID)
	return err
}

func (s *UserStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id=?`, id)
	return err
}

func (s *UserStore) CreateSession(ctx context.Context, session *store.Session) error {
	if session.ID == "" {
		session.ID = generateID()
	}
	session.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt)
	return err
}

func (s *UserStore) GetSession(ctx context.Context, token string) (*store.Session, error) {
	var session store.Session
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions WHERE token = ? AND expires_at > datetime('now')
	`, token).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *UserStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token=?`, token)
	return err
}

func (s *UserStore) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET last_login=datetime('now') WHERE id=?`, userID)
	return err
}
