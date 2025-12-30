package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// Session represents a user session.
type Session struct {
	SessionID    string
	UserID       string
	Token        string
	IPAddress    string
	UserAgent    string
	Payload      string
	LastActivity time.Time
	ExpiresAt    time.Time
}

// SessionsStore handles session persistence.
type SessionsStore struct {
	db *sql.DB
}

// NewSessionsStore creates a new sessions store.
func NewSessionsStore(db *sql.DB) *SessionsStore {
	return &SessionsStore{db: db}
}

// Create creates a new session.
func (s *SessionsStore) Create(ctx context.Context, sess *Session) error {
	query := `
		INSERT INTO wp_sessions (session_id, user_id, token, ip_address, user_agent, payload, last_activity, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.db.ExecContext(ctx, query,
		sess.SessionID, sess.UserID, sess.Token, sess.IPAddress,
		sess.UserAgent, sess.Payload, sess.LastActivity, sess.ExpiresAt,
	)
	return err
}

// GetByToken retrieves a session by token.
func (s *SessionsStore) GetByToken(ctx context.Context, token string) (*Session, error) {
	query := `
		SELECT session_id, user_id, token, ip_address, user_agent, payload, last_activity, expires_at
		FROM wp_sessions WHERE token = $1
	`
	sess := &Session{}
	var ipAddress, userAgent, payload sql.NullString
	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&sess.SessionID, &sess.UserID, &sess.Token, &ipAddress,
		&userAgent, &payload, &sess.LastActivity, &sess.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess.IPAddress = ipAddress.String
	sess.UserAgent = userAgent.String
	sess.Payload = payload.String
	return sess, nil
}

// GetByID retrieves a session by ID.
func (s *SessionsStore) GetByID(ctx context.Context, id string) (*Session, error) {
	query := `
		SELECT session_id, user_id, token, ip_address, user_agent, payload, last_activity, expires_at
		FROM wp_sessions WHERE session_id = $1
	`
	sess := &Session{}
	var ipAddress, userAgent, payload sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&sess.SessionID, &sess.UserID, &sess.Token, &ipAddress,
		&userAgent, &payload, &sess.LastActivity, &sess.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess.IPAddress = ipAddress.String
	sess.UserAgent = userAgent.String
	sess.Payload = payload.String
	return sess, nil
}

// UpdateLastActivity updates the last activity time for a session.
func (s *SessionsStore) UpdateLastActivity(ctx context.Context, sessionID string) error {
	query := `UPDATE wp_sessions SET last_activity = $2 WHERE session_id = $1`
	_, err := s.db.ExecContext(ctx, query, sessionID, time.Now())
	return err
}

// Delete deletes a session by token.
func (s *SessionsStore) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM wp_sessions WHERE token = $1`
	_, err := s.db.ExecContext(ctx, query, token)
	return err
}

// DeleteByID deletes a session by ID.
func (s *SessionsStore) DeleteByID(ctx context.Context, id string) error {
	query := `DELETE FROM wp_sessions WHERE session_id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByUser deletes all sessions for a user.
func (s *SessionsStore) DeleteByUser(ctx context.Context, userID string) error {
	query := `DELETE FROM wp_sessions WHERE user_id = $1`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// DeleteExpired deletes all expired sessions.
func (s *SessionsStore) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM wp_sessions WHERE expires_at < $1`
	_, err := s.db.ExecContext(ctx, query, time.Now())
	return err
}

// ListByUser lists all sessions for a user.
func (s *SessionsStore) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	query := `
		SELECT session_id, user_id, token, ip_address, user_agent, payload, last_activity, expires_at
		FROM wp_sessions WHERE user_id = $1 ORDER BY last_activity DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		sess := &Session{}
		var ipAddress, userAgent, payload sql.NullString
		if err := rows.Scan(
			&sess.SessionID, &sess.UserID, &sess.Token, &ipAddress,
			&userAgent, &payload, &sess.LastActivity, &sess.ExpiresAt,
		); err != nil {
			return nil, err
		}
		sess.IPAddress = ipAddress.String
		sess.UserAgent = userAgent.String
		sess.Payload = payload.String
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}
