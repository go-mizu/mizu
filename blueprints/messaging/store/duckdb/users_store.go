package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
)

// UsersStore implements accounts.Store.
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new UsersStore.
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

// Insert creates a new user.
func (s *UsersStore) Insert(ctx context.Context, u *accounts.User, passwordHash string) error {
	query := `
		INSERT INTO users (id, phone, email, username, display_name, bio, avatar_url, password_hash, status,
			privacy_last_seen, privacy_profile_photo, privacy_about, privacy_groups, privacy_read_receipts,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		u.ID, nullString(u.Phone), nullString(u.Email), u.Username, u.DisplayName, u.Bio, u.AvatarURL, passwordHash, u.Status,
		u.PrivacyLastSeen, u.PrivacyProfilePhoto, u.PrivacyAbout, u.PrivacyGroups, u.PrivacyReadReceipts,
		u.CreatedAt, u.UpdatedAt,
	)
	return err
}

// GetByID retrieves a user by ID.
func (s *UsersStore) GetByID(ctx context.Context, id string) (*accounts.User, error) {
	query := `
		SELECT id, phone, email, username, display_name, bio, avatar_url, status, last_seen_at, is_online,
			privacy_last_seen, privacy_profile_photo, privacy_about, privacy_groups, privacy_read_receipts,
			two_fa_enabled, created_at, updated_at
		FROM users WHERE id = ?
	`
	u, err := scanUser(s.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return u, err
}

func scanUser(row interface{ Scan(...any) error }) (*accounts.User, error) {
	u := &accounts.User{}
	var phone, email, bio, avatarURL, status sql.NullString
	var lastSeenAt sql.NullTime
	err := row.Scan(
		&u.ID, &phone, &email, &u.Username, &u.DisplayName, &bio, &avatarURL, &status, &lastSeenAt, &u.IsOnline,
		&u.PrivacyLastSeen, &u.PrivacyProfilePhoto, &u.PrivacyAbout, &u.PrivacyGroups, &u.PrivacyReadReceipts,
		&u.TwoFAEnabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Phone = phone.String
	u.Email = email.String
	u.Bio = bio.String
	u.AvatarURL = avatarURL.String
	u.Status = status.String
	if lastSeenAt.Valid {
		u.LastSeenAt = lastSeenAt.Time
	}
	return u, nil
}

// GetByIDs retrieves multiple users by IDs.
func (s *UsersStore) GetByIDs(ctx context.Context, ids []string) ([]*accounts.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT id, phone, email, username, display_name, bio, avatar_url, status, last_seen_at, is_online,
			privacy_last_seen, privacy_profile_photo, privacy_about, privacy_groups, privacy_read_receipts,
			two_fa_enabled, created_at, updated_at
		FROM users WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*accounts.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func scanUserRow(rows *sql.Rows) (*accounts.User, error) {
	u := &accounts.User{}
	var phone, email, bio, avatarURL, status sql.NullString
	var lastSeenAt sql.NullTime
	err := rows.Scan(
		&u.ID, &phone, &email, &u.Username, &u.DisplayName, &bio, &avatarURL, &status, &lastSeenAt, &u.IsOnline,
		&u.PrivacyLastSeen, &u.PrivacyProfilePhoto, &u.PrivacyAbout, &u.PrivacyGroups, &u.PrivacyReadReceipts,
		&u.TwoFAEnabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Phone = phone.String
	u.Email = email.String
	u.Bio = bio.String
	u.AvatarURL = avatarURL.String
	u.Status = status.String
	if lastSeenAt.Valid {
		u.LastSeenAt = lastSeenAt.Time
	}
	return u, nil
}

// GetByUsername retrieves a user by username.
func (s *UsersStore) GetByUsername(ctx context.Context, username string) (*accounts.User, error) {
	query := `
		SELECT id, phone, email, username, display_name, bio, avatar_url, status, last_seen_at, is_online,
			privacy_last_seen, privacy_profile_photo, privacy_about, privacy_groups, privacy_read_receipts,
			two_fa_enabled, created_at, updated_at
		FROM users WHERE username = ?
	`
	u, err := scanUser(s.db.QueryRowContext(ctx, query, username))
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return u, err
}

// GetByPhone retrieves a user by phone.
func (s *UsersStore) GetByPhone(ctx context.Context, phone string) (*accounts.User, error) {
	query := `
		SELECT id, phone, email, username, display_name, bio, avatar_url, status, last_seen_at, is_online,
			privacy_last_seen, privacy_profile_photo, privacy_about, privacy_groups, privacy_read_receipts,
			two_fa_enabled, created_at, updated_at
		FROM users WHERE phone = ?
	`
	u, err := scanUser(s.db.QueryRowContext(ctx, query, phone))
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return u, err
}

// GetByEmail retrieves a user by email.
func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*accounts.User, error) {
	query := `
		SELECT id, phone, email, username, display_name, bio, avatar_url, status, last_seen_at, is_online,
			privacy_last_seen, privacy_profile_photo, privacy_about, privacy_groups, privacy_read_receipts,
			two_fa_enabled, created_at, updated_at
		FROM users WHERE email = ?
	`
	u, err := scanUser(s.db.QueryRowContext(ctx, query, email))
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return u, err
}

// Update updates a user.
func (s *UsersStore) Update(ctx context.Context, id string, in *accounts.UpdateIn) error {
	var sets []string
	var args []any

	if in.DisplayName != nil {
		sets = append(sets, "display_name = ?")
		args = append(args, *in.DisplayName)
	}
	if in.Bio != nil {
		sets = append(sets, "bio = ?")
		args = append(args, *in.Bio)
	}
	if in.AvatarURL != nil {
		sets = append(sets, "avatar_url = ?")
		args = append(args, *in.AvatarURL)
	}
	if in.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *in.Status)
	}
	if in.PrivacyLastSeen != nil {
		sets = append(sets, "privacy_last_seen = ?")
		args = append(args, *in.PrivacyLastSeen)
	}
	if in.PrivacyProfilePhoto != nil {
		sets = append(sets, "privacy_profile_photo = ?")
		args = append(args, *in.PrivacyProfilePhoto)
	}
	if in.PrivacyAbout != nil {
		sets = append(sets, "privacy_about = ?")
		args = append(args, *in.PrivacyAbout)
	}
	if in.PrivacyGroups != nil {
		sets = append(sets, "privacy_groups = ?")
		args = append(args, *in.PrivacyGroups)
	}
	if in.PrivacyReadReceipts != nil {
		sets = append(sets, "privacy_read_receipts = ?")
		args = append(args, *in.PrivacyReadReceipts)
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a user.
func (s *UsersStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}

// ExistsUsername checks if a username exists.
func (s *UsersStore) ExistsUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	return exists, err
}

// ExistsPhone checks if a phone exists.
func (s *UsersStore) ExistsPhone(ctx context.Context, phone string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE phone = ?)", phone).Scan(&exists)
	return exists, err
}

// ExistsEmail checks if an email exists.
func (s *UsersStore) ExistsEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
	return exists, err
}

// GetPasswordHash retrieves password hash for authentication.
func (s *UsersStore) GetPasswordHash(ctx context.Context, login string) (id, hash string, err error) {
	query := `SELECT id, password_hash FROM users WHERE username = ? OR email = ? OR phone = ?`
	err = s.db.QueryRowContext(ctx, query, login, login, login).Scan(&id, &hash)
	if err == sql.ErrNoRows {
		return "", "", accounts.ErrNotFound
	}
	return id, hash, err
}

// Search searches for users.
func (s *UsersStore) Search(ctx context.Context, query string, limit int) ([]*accounts.User, error) {
	searchQuery := `
		SELECT id, phone, email, username, display_name, bio, avatar_url, status, last_seen_at, is_online,
			privacy_last_seen, privacy_profile_photo, privacy_about, privacy_groups, privacy_read_receipts,
			two_fa_enabled, created_at, updated_at
		FROM users
		WHERE username ILIKE ? OR display_name ILIKE ? OR phone ILIKE ?
		LIMIT ?
	`
	pattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, searchQuery, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*accounts.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateOnlineStatus updates online status.
func (s *UsersStore) UpdateOnlineStatus(ctx context.Context, userID string, online bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE users SET is_online = ?, updated_at = ? WHERE id = ?", online, time.Now(), userID)
	return err
}

// UpdateLastSeen updates last seen timestamp.
func (s *UsersStore) UpdateLastSeen(ctx context.Context, userID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, "UPDATE users SET last_seen_at = ?, updated_at = ? WHERE id = ?", now, now, userID)
	return err
}

// CreateSession creates a new session.
func (s *UsersStore) CreateSession(ctx context.Context, sess *accounts.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, device_name, device_type, push_token, ip_address, user_agent, last_active_at, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		sess.ID, sess.UserID, sess.Token, sess.DeviceName, sess.DeviceType, sess.PushToken,
		sess.IPAddress, sess.UserAgent, sess.LastActiveAt, sess.ExpiresAt, sess.CreatedAt,
	)
	return err
}

// GetSession retrieves a session by token.
func (s *UsersStore) GetSession(ctx context.Context, token string) (*accounts.Session, error) {
	query := `
		SELECT id, user_id, token, device_name, device_type, push_token, ip_address, user_agent, last_active_at, expires_at, created_at
		FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP
	`
	sess := &accounts.Session{}
	var deviceName, deviceType, pushToken, ipAddress, userAgent sql.NullString
	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&sess.ID, &sess.UserID, &sess.Token, &deviceName, &deviceType, &pushToken,
		&ipAddress, &userAgent, &sess.LastActiveAt, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, accounts.ErrInvalidSession
	}
	if err != nil {
		return nil, err
	}
	sess.DeviceName = deviceName.String
	sess.DeviceType = deviceType.String
	sess.PushToken = pushToken.String
	sess.IPAddress = ipAddress.String
	sess.UserAgent = userAgent.String
	return sess, nil
}

// DeleteSession deletes a session.
func (s *UsersStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	return err
}

// DeleteAllSessions deletes all sessions for a user.
func (s *UsersStore) DeleteAllSessions(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// DeleteExpiredSessions deletes all expired sessions.
func (s *UsersStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
