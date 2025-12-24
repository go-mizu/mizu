package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/social/feature/accounts"
)

// AccountsStore implements accounts.Store.
type AccountsStore struct {
	db *sql.DB
}

// NewAccountsStore creates a new accounts store.
func NewAccountsStore(db *sql.DB) *AccountsStore {
	return &AccountsStore{db: db}
}

// Insert inserts a new account.
func (s *AccountsStore) Insert(ctx context.Context, a *accounts.Account, passwordHash string) error {
	fieldsJSON, _ := json.Marshal(a.Fields)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, username, display_name, email, password_hash, bio, avatar_url, header_url, location, website, fields, verified, admin, suspended, private, discoverable, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, a.ID, a.Username, a.DisplayName, a.Email, passwordHash, a.Bio, a.AvatarURL, a.HeaderURL, a.Location, a.Website, string(fieldsJSON), a.Verified, a.Admin, a.Suspended, a.Private, a.Discoverable, a.CreatedAt, a.UpdatedAt)
	return err
}

// GetByID retrieves an account by ID.
func (s *AccountsStore) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, location, website, fields, verified, admin, suspended, private, discoverable, created_at, updated_at
		FROM accounts WHERE id = $1
	`, id)
	return s.scanAccount(row)
}

// GetByIDs retrieves multiple accounts by IDs.
func (s *AccountsStore) GetByIDs(ctx context.Context, ids []string) ([]*accounts.Account, error) {
	if len(ids) == 0 {
		return []*accounts.Account{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, username, display_name, email, bio, avatar_url, header_url, location, website, fields, verified, admin, suspended, private, discoverable, created_at, updated_at
		FROM accounts WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accs []*accounts.Account
	for rows.Next() {
		a, err := s.scanAccountRow(rows)
		if err != nil {
			return nil, err
		}
		accs = append(accs, a)
	}
	return accs, rows.Err()
}

// GetByUsername retrieves an account by username.
func (s *AccountsStore) GetByUsername(ctx context.Context, username string) (*accounts.Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, location, website, fields, verified, admin, suspended, private, discoverable, created_at, updated_at
		FROM accounts WHERE LOWER(username) = LOWER($1)
	`, username)
	return s.scanAccount(row)
}

// GetByEmail retrieves an account by email.
func (s *AccountsStore) GetByEmail(ctx context.Context, email string) (*accounts.Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, location, website, fields, verified, admin, suspended, private, discoverable, created_at, updated_at
		FROM accounts WHERE LOWER(email) = LOWER($1)
	`, email)
	return s.scanAccount(row)
}

// Update updates an account.
func (s *AccountsStore) Update(ctx context.Context, id string, in *accounts.UpdateIn) error {
	sets := []string{"updated_at = $1"}
	args := []interface{}{time.Now()}
	argNum := 2

	if in.DisplayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", argNum))
		args = append(args, *in.DisplayName)
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
	if in.HeaderURL != nil {
		sets = append(sets, fmt.Sprintf("header_url = $%d", argNum))
		args = append(args, *in.HeaderURL)
		argNum++
	}
	if in.Location != nil {
		sets = append(sets, fmt.Sprintf("location = $%d", argNum))
		args = append(args, *in.Location)
		argNum++
	}
	if in.Website != nil {
		sets = append(sets, fmt.Sprintf("website = $%d", argNum))
		args = append(args, *in.Website)
		argNum++
	}
	if in.Fields != nil {
		fieldsJSON, _ := json.Marshal(*in.Fields)
		sets = append(sets, fmt.Sprintf("fields = $%d", argNum))
		args = append(args, string(fieldsJSON))
		argNum++
	}
	if in.Private != nil {
		sets = append(sets, fmt.Sprintf("private = $%d", argNum))
		args = append(args, *in.Private)
		argNum++
	}
	if in.Discoverable != nil {
		sets = append(sets, fmt.Sprintf("discoverable = $%d", argNum))
		args = append(args, *in.Discoverable)
		argNum++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE accounts SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ExistsUsername checks if a username exists.
func (s *AccountsStore) ExistsUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE LOWER(username) = LOWER($1))", username).Scan(&exists)
	return exists, err
}

// ExistsEmail checks if an email exists.
func (s *AccountsStore) ExistsEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE LOWER(email) = LOWER($1))", email).Scan(&exists)
	return exists, err
}

// GetPasswordHash retrieves password hash for login.
func (s *AccountsStore) GetPasswordHash(ctx context.Context, usernameOrEmail string) (id, hash string, suspended bool, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT id, password_hash, suspended FROM accounts
		WHERE LOWER(username) = LOWER($1) OR LOWER(email) = LOWER($1)
	`, usernameOrEmail).Scan(&id, &hash, &suspended)
	return
}

// List lists accounts with pagination.
func (s *AccountsStore) List(ctx context.Context, limit, offset int) ([]*accounts.Account, int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM accounts").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, location, website, fields, verified, admin, suspended, private, discoverable, created_at, updated_at
		FROM accounts ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accs []*accounts.Account
	for rows.Next() {
		a, err := s.scanAccountRow(rows)
		if err != nil {
			return nil, 0, err
		}
		accs = append(accs, a)
	}
	return accs, total, rows.Err()
}

// Search searches accounts by username or display name.
func (s *AccountsStore) Search(ctx context.Context, query string, limit int) ([]*accounts.Account, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, location, website, fields, verified, admin, suspended, private, discoverable, created_at, updated_at
		FROM accounts
		WHERE discoverable = TRUE AND suspended = FALSE AND (LOWER(username) LIKE $1 OR LOWER(display_name) LIKE $1)
		ORDER BY
			CASE WHEN LOWER(username) = LOWER($2) THEN 0 ELSE 1 END,
			created_at DESC
		LIMIT $3
	`, pattern, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accs []*accounts.Account
	for rows.Next() {
		a, err := s.scanAccountRow(rows)
		if err != nil {
			return nil, err
		}
		accs = append(accs, a)
	}
	return accs, rows.Err()
}

// SetVerified sets the verified status.
func (s *AccountsStore) SetVerified(ctx context.Context, id string, verified bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE accounts SET verified = $1, updated_at = $2 WHERE id = $3", verified, time.Now(), id)
	return err
}

// SetSuspended sets the suspended status.
func (s *AccountsStore) SetSuspended(ctx context.Context, id string, suspended bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE accounts SET suspended = $1, updated_at = $2 WHERE id = $3", suspended, time.Now(), id)
	return err
}

// SetAdmin sets the admin status.
func (s *AccountsStore) SetAdmin(ctx context.Context, id string, admin bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE accounts SET admin = $1, updated_at = $2 WHERE id = $3", admin, time.Now(), id)
	return err
}

// GetFollowersCount returns the follower count.
func (s *AccountsStore) GetFollowersCount(ctx context.Context, id string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM follows WHERE following_id = $1 AND pending = FALSE", id).Scan(&count)
	return count, err
}

// GetFollowingCount returns the following count.
func (s *AccountsStore) GetFollowingCount(ctx context.Context, id string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM follows WHERE follower_id = $1 AND pending = FALSE", id).Scan(&count)
	return count, err
}

// GetPostsCount returns the posts count.
func (s *AccountsStore) GetPostsCount(ctx context.Context, id string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE account_id = $1", id).Scan(&count)
	return count, err
}

// CreateSession creates a new session.
func (s *AccountsStore) CreateSession(ctx context.Context, sess *accounts.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, account_id, token, user_agent, ip_address, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, sess.ID, sess.AccountID, sess.Token, sess.UserAgent, sess.IPAddress, sess.ExpiresAt, sess.CreatedAt)
	return err
}

// GetSession retrieves a session by token.
func (s *AccountsStore) GetSession(ctx context.Context, token string) (*accounts.Session, error) {
	var sess accounts.Session
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, token, user_agent, ip_address, expires_at, created_at
		FROM sessions WHERE token = $1
	`, token).Scan(&sess.ID, &sess.AccountID, &sess.Token, &sess.UserAgent, &sess.IPAddress, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// DeleteSession deletes a session.
func (s *AccountsStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = $1", token)
	return err
}

// DeleteExpiredSessions deletes expired sessions.
func (s *AccountsStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < $1", time.Now())
	return err
}

func (s *AccountsStore) scanAccount(row *sql.Row) (*accounts.Account, error) {
	var a accounts.Account
	var displayName, email, bio, avatarURL, headerURL, location, website sql.NullString
	var fieldsJSON string

	err := row.Scan(&a.ID, &a.Username, &displayName, &email, &bio, &avatarURL, &headerURL, &location, &website, &fieldsJSON, &a.Verified, &a.Admin, &a.Suspended, &a.Private, &a.Discoverable, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}

	a.DisplayName = displayName.String
	a.Email = email.String
	a.Bio = bio.String
	a.AvatarURL = avatarURL.String
	a.HeaderURL = headerURL.String
	a.Location = location.String
	a.Website = website.String

	if fieldsJSON != "" {
		_ = json.Unmarshal([]byte(fieldsJSON), &a.Fields)
	}

	return &a, nil
}

func (s *AccountsStore) scanAccountRow(rows *sql.Rows) (*accounts.Account, error) {
	var a accounts.Account
	var displayName, email, bio, avatarURL, headerURL, location, website sql.NullString
	var fieldsJSON string

	err := rows.Scan(&a.ID, &a.Username, &displayName, &email, &bio, &avatarURL, &headerURL, &location, &website, &fieldsJSON, &a.Verified, &a.Admin, &a.Suspended, &a.Private, &a.Discoverable, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}

	a.DisplayName = displayName.String
	a.Email = email.String
	a.Bio = bio.String
	a.AvatarURL = avatarURL.String
	a.HeaderURL = headerURL.String
	a.Location = location.String
	a.Website = website.String

	if fieldsJSON != "" {
		_ = json.Unmarshal([]byte(fieldsJSON), &a.Fields)
	}

	return &a, nil
}
