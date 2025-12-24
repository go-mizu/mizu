package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/accounts"
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
		INSERT INTO users (id, username, discriminator, display_name, email, password_hash, avatar_url, bio, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		u.ID, u.Username, u.Discriminator, u.DisplayName, u.Email, passwordHash,
		u.AvatarURL, u.Bio, u.Status, u.CreatedAt, u.UpdatedAt,
	)
	return err
}

// GetByID retrieves a user by ID.
func (s *UsersStore) GetByID(ctx context.Context, id string) (*accounts.User, error) {
	query := `
		SELECT id, username, discriminator, display_name, email, avatar_url, banner_url, bio, status, custom_status, is_bot, is_verified, is_admin, created_at, updated_at
		FROM users WHERE id = ?
	`
	u, err := scanUser(s.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return u, err
}

// scanUser scans a user row, handling nullable columns.
func scanUser(row interface{ Scan(...any) error }) (*accounts.User, error) {
	u := &accounts.User{}
	var displayName, email, avatarURL, bannerURL, bio, status, customStatus sql.NullString
	err := row.Scan(
		&u.ID, &u.Username, &u.Discriminator, &displayName, &email,
		&avatarURL, &bannerURL, &bio, &status, &customStatus,
		&u.IsBot, &u.IsVerified, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.DisplayName = displayName.String
	u.Email = email.String
	u.AvatarURL = avatarURL.String
	u.BannerURL = bannerURL.String
	u.Bio = bio.String
	u.Status = accounts.Status(status.String)
	u.CustomStatus = customStatus.String
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
		SELECT id, username, discriminator, display_name, email, avatar_url, banner_url, bio, status, custom_status, is_bot, is_verified, is_admin, created_at, updated_at
		FROM users WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUsers(rows)
}

// scanUsers scans multiple user rows.
func scanUsers(rows *sql.Rows) ([]*accounts.User, error) {
	var users []*accounts.User
	for rows.Next() {
		u := &accounts.User{}
		var displayName, email, avatarURL, bannerURL, bio, status, customStatus sql.NullString
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Discriminator, &displayName, &email,
			&avatarURL, &bannerURL, &bio, &status, &customStatus,
			&u.IsBot, &u.IsVerified, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		u.DisplayName = displayName.String
		u.Email = email.String
		u.AvatarURL = avatarURL.String
		u.BannerURL = bannerURL.String
		u.Bio = bio.String
		u.Status = accounts.Status(status.String)
		u.CustomStatus = customStatus.String
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetByUsername retrieves a user by username.
func (s *UsersStore) GetByUsername(ctx context.Context, username string) (*accounts.User, error) {
	query := `
		SELECT id, username, discriminator, display_name, email, avatar_url, banner_url, bio, status, custom_status, is_bot, is_verified, is_admin, created_at, updated_at
		FROM users WHERE username = ?
	`
	u, err := scanUser(s.db.QueryRowContext(ctx, query, username))
	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	return u, err
}

// GetByEmail retrieves a user by email.
func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*accounts.User, error) {
	query := `
		SELECT id, username, discriminator, display_name, email, avatar_url, banner_url, bio, status, custom_status, is_bot, is_verified, is_admin, created_at, updated_at
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
	if in.AvatarURL != nil {
		sets = append(sets, "avatar_url = ?")
		args = append(args, *in.AvatarURL)
	}
	if in.BannerURL != nil {
		sets = append(sets, "banner_url = ?")
		args = append(args, *in.BannerURL)
	}
	if in.Bio != nil {
		sets = append(sets, "bio = ?")
		args = append(args, *in.Bio)
	}
	if in.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *in.Status)
	}
	if in.CustomStatus != nil {
		sets = append(sets, "custom_status = ?")
		args = append(args, *in.CustomStatus)
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

// ExistsUsername checks if a username exists.
func (s *UsersStore) ExistsUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	return exists, err
}

// ExistsEmail checks if an email exists.
func (s *UsersStore) ExistsEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
	return exists, err
}

// GetPasswordHash retrieves password hash for authentication.
func (s *UsersStore) GetPasswordHash(ctx context.Context, usernameOrEmail string) (id, hash string, err error) {
	query := `SELECT id, password_hash FROM users WHERE username = ? OR email = ?`
	err = s.db.QueryRowContext(ctx, query, usernameOrEmail, usernameOrEmail).Scan(&id, &hash)
	if err == sql.ErrNoRows {
		return "", "", accounts.ErrNotFound
	}
	return id, hash, err
}

// Search searches for users by username or display name.
func (s *UsersStore) Search(ctx context.Context, query string, limit int) ([]*accounts.User, error) {
	searchQuery := `
		SELECT id, username, discriminator, display_name, email, avatar_url, banner_url, bio, status, custom_status, is_bot, is_verified, is_admin, created_at, updated_at
		FROM users
		WHERE username ILIKE ? OR display_name ILIKE ?
		LIMIT ?
	`
	pattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, searchQuery, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUsers(rows)
}

// GetNextDiscriminator gets the next available discriminator for a username.
func (s *UsersStore) GetNextDiscriminator(ctx context.Context, username string) (string, error) {
	var maxDisc sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT MAX(discriminator) FROM users WHERE username = ?",
		username,
	).Scan(&maxDisc)
	if err != nil {
		return "0001", err
	}
	if !maxDisc.Valid {
		return "0001", nil
	}
	var num int
	fmt.Sscanf(maxDisc.String, "%04d", &num)
	if num >= 9999 {
		return "", fmt.Errorf("username exhausted")
	}
	return fmt.Sprintf("%04d", num+1), nil
}

// CreateSession creates a new session.
func (s *UsersStore) CreateSession(ctx context.Context, sess *accounts.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, user_agent, ip_address, device_type, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		sess.ID, sess.UserID, sess.Token, sess.UserAgent, sess.IPAddress, sess.DeviceType, sess.ExpiresAt, sess.CreatedAt,
	)
	return err
}

// GetSession retrieves a session by token.
func (s *UsersStore) GetSession(ctx context.Context, token string) (*accounts.Session, error) {
	query := `
		SELECT id, user_id, token, user_agent, ip_address, device_type, expires_at, created_at
		FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP
	`
	sess := &accounts.Session{}
	var userAgent, ipAddress, deviceType sql.NullString
	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&sess.ID, &sess.UserID, &sess.Token, &userAgent, &ipAddress, &deviceType, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, accounts.ErrInvalidSession
	}
	if err != nil {
		return nil, err
	}
	sess.UserAgent = userAgent.String
	sess.IPAddress = ipAddress.String
	sess.DeviceType = deviceType.String
	return sess, nil
}

// DeleteSession deletes a session.
func (s *UsersStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	return err
}

// DeleteExpiredSessions deletes all expired sessions.
func (s *UsersStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

// UpdateStatus updates a user's status.
func (s *UsersStore) UpdateStatus(ctx context.Context, userID string, status string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE users SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), userID)
	return err
}
