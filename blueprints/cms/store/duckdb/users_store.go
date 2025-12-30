package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/users"
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
		INSERT INTO users (id, email, password_hash, name, slug, bio, avatar_url, role, status, meta, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, u.ID, u.Email, u.PasswordHash, u.Name, u.Slug, u.Bio, u.AvatarURL, u.Role, u.Status, u.Meta, u.CreatedAt, u.UpdatedAt)
	return err
}

func (s *UsersStore) GetByID(ctx context.Context, id string) (*users.User, error) {
	u := &users.User{}
	var bio, avatarURL, meta sql.NullString
	var lastLoginAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, name, slug, bio, avatar_url, role, status, meta, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Slug, &bio, &avatarURL, &u.Role, &u.Status, &meta, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Bio = bio.String
	u.AvatarURL = avatarURL.String
	u.Meta = meta.String
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	return u, nil
}

func (s *UsersStore) GetByIDs(ctx context.Context, ids []string) ([]*users.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, email, password_hash, name, slug, bio, avatar_url, role, status, meta, last_login_at, created_at, updated_at
		FROM users WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*users.User
	for rows.Next() {
		u := &users.User{}
		var bio, avatarURL, meta sql.NullString
		var lastLoginAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Slug, &bio, &avatarURL, &u.Role, &u.Status, &meta, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.Bio = bio.String
		u.AvatarURL = avatarURL.String
		u.Meta = meta.String
		if lastLoginAt.Valid {
			u.LastLoginAt = &lastLoginAt.Time
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	u := &users.User{}
	var bio, avatarURL, meta sql.NullString
	var lastLoginAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, name, slug, bio, avatar_url, role, status, meta, last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Slug, &bio, &avatarURL, &u.Role, &u.Status, &meta, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Bio = bio.String
	u.AvatarURL = avatarURL.String
	u.Meta = meta.String
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	return u, nil
}

func (s *UsersStore) GetBySlug(ctx context.Context, slug string) (*users.User, error) {
	u := &users.User{}
	var bio, avatarURL, meta sql.NullString
	var lastLoginAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, name, slug, bio, avatar_url, role, status, meta, last_login_at, created_at, updated_at
		FROM users WHERE slug = $1
	`, slug).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Slug, &bio, &avatarURL, &u.Role, &u.Status, &meta, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Bio = bio.String
	u.AvatarURL = avatarURL.String
	u.Meta = meta.String
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	return u, nil
}

func (s *UsersStore) List(ctx context.Context, in *users.ListIn) ([]*users.User, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	if in.Role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argNum))
		args = append(args, in.Role)
		argNum++
	}
	if in.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, in.Status)
		argNum++
	}
	if in.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR email ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+in.Search+"%")
		argNum++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get items
	args = append(args, in.Limit, in.Offset)
	query := fmt.Sprintf(`
		SELECT id, email, password_hash, name, slug, bio, avatar_url, role, status, meta, last_login_at, created_at, updated_at
		FROM users %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argNum, argNum+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*users.User
	for rows.Next() {
		u := &users.User{}
		var bio, avatarURL, meta sql.NullString
		var lastLoginAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Slug, &bio, &avatarURL, &u.Role, &u.Status, &meta, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		u.Bio = bio.String
		u.AvatarURL = avatarURL.String
		u.Meta = meta.String
		if lastLoginAt.Valid {
			u.LastLoginAt = &lastLoginAt.Time
		}
		list = append(list, u)
	}
	return list, total, rows.Err()
}

func (s *UsersStore) Update(ctx context.Context, id string, in *users.UpdateIn) error {
	var sets []string
	var args []any
	argNum := 1

	if in.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *in.Name)
		argNum++
	}
	if in.Bio != nil {
		sets = append(sets, fmt.Sprintf("bio = $%d", argNum))
		args = append(args, *in.Bio)
		argNum++
	}
	if in.AvatarURL != nil {
		sets = append(sets, fmt.Sprintf("avatar_url = $%d", argNum))
		args = append(args, *in.AvatarURL)
		argNum++
	}
	if in.Role != nil {
		sets = append(sets, fmt.Sprintf("role = $%d", argNum))
		args = append(args, *in.Role)
		argNum++
	}
	if in.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *in.Status)
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *UsersStore) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1
	`, id, passwordHash, time.Now())
	return err
}

func (s *UsersStore) UpdateLastLogin(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET last_login_at = $2 WHERE id = $1
	`, id, time.Now())
	return err
}

func (s *UsersStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// Session operations

func (s *UsersStore) CreateSession(ctx context.Context, sess *users.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, refresh_token, user_agent, ip_address, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, sess.ID, sess.UserID, sess.RefreshToken, sess.UserAgent, sess.IPAddress, sess.ExpiresAt, sess.CreatedAt)
	return err
}

func (s *UsersStore) GetSession(ctx context.Context, id string) (*users.Session, error) {
	sess := &users.Session{}
	var refreshToken, userAgent, ipAddress sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, refresh_token, user_agent, ip_address, expires_at, created_at
		FROM sessions WHERE id = $1
	`, id).Scan(&sess.ID, &sess.UserID, &refreshToken, &userAgent, &ipAddress, &sess.ExpiresAt, &sess.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess.RefreshToken = refreshToken.String
	sess.UserAgent = userAgent.String
	sess.IPAddress = ipAddress.String
	return sess, nil
}

func (s *UsersStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

func (s *UsersStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < $1`, time.Now())
	return err
}

func (s *UsersStore) GetUserBySession(ctx context.Context, sessionID string) (*users.User, error) {
	u := &users.User{}
	var bio, avatarURL, meta sql.NullString
	var lastLoginAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.password_hash, u.name, u.slug, u.bio, u.avatar_url, u.role, u.status, u.meta, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		INNER JOIN sessions s ON s.user_id = u.id
		WHERE s.id = $1 AND s.expires_at > $2
	`, sessionID, time.Now()).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Slug, &bio, &avatarURL, &u.Role, &u.Status, &meta, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Bio = bio.String
	u.AvatarURL = avatarURL.String
	u.Meta = meta.String
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	return u, nil
}
