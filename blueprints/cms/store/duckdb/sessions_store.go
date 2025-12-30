package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// SessionsStore handles session operations.
type SessionsStore struct {
	db *sql.DB
}

// NewSessionsStore creates a new SessionsStore.
func NewSessionsStore(db *sql.DB) *SessionsStore {
	return &SessionsStore{db: db}
}

// Session represents a user session.
type Session struct {
	ID           string
	UserID       string
	Collection   string
	Token        string
	RefreshToken string
	UserAgent    string
	IP           string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// Create creates a new session.
func (s *SessionsStore) Create(ctx context.Context, session *Session) error {
	session.ID = ulid.New()
	session.CreatedAt = time.Now()

	query := `INSERT INTO _sessions (id, user_id, collection, token, refresh_token, user_agent, ip, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.Collection, session.Token, session.RefreshToken,
		session.UserAgent, session.IP, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

// GetByToken retrieves a session by token.
func (s *SessionsStore) GetByToken(ctx context.Context, token string) (*Session, error) {
	query := `SELECT id, user_id, collection, token, refresh_token, user_agent, ip, expires_at, created_at
		FROM _sessions WHERE token = ? AND expires_at > ?`

	var session Session
	var refreshToken, userAgent, ip sql.NullString

	err := s.db.QueryRowContext(ctx, query, token, time.Now()).Scan(
		&session.ID, &session.UserID, &session.Collection, &session.Token, &refreshToken,
		&userAgent, &ip, &session.ExpiresAt, &session.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	if refreshToken.Valid {
		session.RefreshToken = refreshToken.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	if ip.Valid {
		session.IP = ip.String
	}

	return &session, nil
}

// GetByRefreshToken retrieves a session by refresh token.
func (s *SessionsStore) GetByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	query := `SELECT id, user_id, collection, token, refresh_token, user_agent, ip, expires_at, created_at
		FROM _sessions WHERE refresh_token = ?`

	var session Session
	var rt, userAgent, ip sql.NullString

	err := s.db.QueryRowContext(ctx, query, refreshToken).Scan(
		&session.ID, &session.UserID, &session.Collection, &session.Token, &rt,
		&userAgent, &ip, &session.ExpiresAt, &session.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get session by refresh token: %w", err)
	}

	if rt.Valid {
		session.RefreshToken = rt.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	if ip.Valid {
		session.IP = ip.String
	}

	return &session, nil
}

// Delete deletes a session by token.
func (s *SessionsStore) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM _sessions WHERE token = ?`
	_, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteByUser deletes all sessions for a user.
func (s *SessionsStore) DeleteByUser(ctx context.Context, userID string) error {
	query := `DELETE FROM _sessions WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	return nil
}

// Update updates a session.
func (s *SessionsStore) Update(ctx context.Context, session *Session) error {
	query := `UPDATE _sessions SET token = ?, refresh_token = ?, expires_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, session.Token, session.RefreshToken, session.ExpiresAt, session.ID)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	return nil
}

// CleanupExpired removes expired sessions.
func (s *SessionsStore) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM _sessions WHERE expires_at < ?`
	result, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("cleanup sessions: %w", err)
	}
	return result.RowsAffected()
}
