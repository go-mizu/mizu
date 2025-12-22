package duckdb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-mizu/blueprints/forum/feature/accounts"
)

// AccountsStore implements accounts.Store using DuckDB.
type AccountsStore struct {
	db *sql.DB
}

// NewAccountsStore creates a new accounts store.
func NewAccountsStore(db *sql.DB) *AccountsStore {
	return &AccountsStore{db: db}
}

// Insert inserts a new account.
func (s *AccountsStore) Insert(ctx context.Context, a *accounts.Account, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (
			id, username, display_name, email, password_hash,
			post_karma, comment_karma, total_karma, trust_level,
			verified, admin, suspended, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, a.ID, a.Username, a.DisplayName, a.Email, passwordHash,
		a.PostKarma, a.CommentKarma, a.TotalKarma, a.TrustLevel,
		a.Verified, a.Admin, a.Suspended, a.CreatedAt, a.UpdatedAt)
	return err
}

// GetByID retrieves an account by ID.
func (s *AccountsStore) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, signature,
		       post_karma, comment_karma, total_karma, trust_level,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE id = $1
	`, id)

	return s.scanAccount(row)
}

// GetByUsername retrieves an account by username.
func (s *AccountsStore) GetByUsername(ctx context.Context, username string) (*accounts.Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, signature,
		       post_karma, comment_karma, total_karma, trust_level,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE username = $1
	`, username)

	return s.scanAccount(row)
}

// GetByEmail retrieves an account by email.
func (s *AccountsStore) GetByEmail(ctx context.Context, email string) (*accounts.Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, signature,
		       post_karma, comment_karma, total_karma, trust_level,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE email = $1
	`, email)

	return s.scanAccount(row)
}

// Update updates an account.
func (s *AccountsStore) Update(ctx context.Context, id string, in *accounts.UpdateIn) error {
	query := `UPDATE accounts SET updated_at = CURRENT_TIMESTAMP`
	args := []any{}
	argNum := 1

	if in.DisplayName != nil {
		query += `, display_name = $` + string(rune('0'+argNum))
		args = append(args, *in.DisplayName)
		argNum++
	}
	if in.Bio != nil {
		query += `, bio = $` + string(rune('0'+argNum))
		args = append(args, *in.Bio)
		argNum++
	}
	if in.AvatarURL != nil {
		query += `, avatar_url = $` + string(rune('0'+argNum))
		args = append(args, *in.AvatarURL)
		argNum++
	}
	if in.HeaderURL != nil {
		query += `, header_url = $` + string(rune('0'+argNum))
		args = append(args, *in.HeaderURL)
		argNum++
	}
	if in.Signature != nil {
		query += `, signature = $` + string(rune('0'+argNum))
		args = append(args, *in.Signature)
		argNum++
	}

	query += ` WHERE id = $` + string(rune('0'+argNum))
	args = append(args, id)

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ExistsUsername checks if a username exists.
func (s *AccountsStore) ExistsUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE username = $1)`, username).Scan(&exists)
	return exists, err
}

// ExistsEmail checks if an email exists.
func (s *AccountsStore) ExistsEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

// GetPasswordHash retrieves the password hash for login.
func (s *AccountsStore) GetPasswordHash(ctx context.Context, usernameOrEmail string) (id, hash string, suspended bool, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT id, password_hash, suspended
		FROM accounts
		WHERE username = $1 OR email = $1
	`, usernameOrEmail).Scan(&id, &hash, &suspended)

	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, accounts.ErrNotFound
	}

	return id, hash, suspended, err
}

// List lists accounts with pagination.
func (s *AccountsStore) List(ctx context.Context, limit, offset int) ([]*accounts.Account, int, error) {
	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM accounts`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get accounts
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, signature,
		       post_karma, comment_karma, total_karma, trust_level,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accountsList []*accounts.Account
	for rows.Next() {
		account, err := s.scanAccountFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		accountsList = append(accountsList, account)
	}

	return accountsList, total, rows.Err()
}

// Search searches for accounts.
func (s *AccountsStore) Search(ctx context.Context, query string, limit int) ([]*accounts.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, signature,
		       post_karma, comment_karma, total_karma, trust_level,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts
		WHERE username LIKE $1 OR display_name LIKE $1
		ORDER BY total_karma DESC
		LIMIT $2
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accountsList []*accounts.Account
	for rows.Next() {
		account, err := s.scanAccountFromRows(rows)
		if err != nil {
			return nil, err
		}
		accountsList = append(accountsList, account)
	}

	return accountsList, rows.Err()
}

// CreateSession creates a new session.
func (s *AccountsStore) CreateSession(ctx context.Context, session *accounts.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token, account_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`, session.Token, session.AccountID, session.ExpiresAt, session.CreatedAt)
	return err
}

// GetSession retrieves a session by token.
func (s *AccountsStore) GetSession(ctx context.Context, token string) (*accounts.Session, error) {
	session := &accounts.Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT token, account_id, expires_at, created_at
		FROM sessions WHERE token = $1
	`, token).Scan(&session.Token, &session.AccountID, &session.ExpiresAt, &session.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, accounts.ErrNotFound
	}

	return session, err
}

// DeleteSession deletes a session.
func (s *AccountsStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

// AddKarma adds karma to an account.
func (s *AccountsStore) AddKarma(ctx context.Context, accountID string, postKarma, commentKarma int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE accounts
		SET post_karma = post_karma + $1,
		    comment_karma = comment_karma + $2,
		    total_karma = total_karma + $1 + $2
		WHERE id = $3
	`, postKarma, commentKarma, accountID)
	return err
}

// SetVerified sets the verified status.
func (s *AccountsStore) SetVerified(ctx context.Context, id string, verified bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE accounts SET verified = $1 WHERE id = $2`, verified, id)
	return err
}

// SetSuspended sets the suspended status.
func (s *AccountsStore) SetSuspended(ctx context.Context, id string, suspended bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE accounts SET suspended = $1 WHERE id = $2`, suspended, id)
	return err
}

// SetAdmin sets the admin status.
func (s *AccountsStore) SetAdmin(ctx context.Context, id string, admin bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE accounts SET admin = $1 WHERE id = $2`, admin, id)
	return err
}

// scanAccount scans a single account from a row.
func (s *AccountsStore) scanAccount(row *sql.Row) (*accounts.Account, error) {
	var a accounts.Account
	var bio, avatarURL, headerURL, signature sql.NullString

	err := row.Scan(
		&a.ID, &a.Username, &a.DisplayName, &a.Email,
		&bio, &avatarURL, &headerURL, &signature,
		&a.PostKarma, &a.CommentKarma, &a.TotalKarma, &a.TrustLevel,
		&a.Verified, &a.Admin, &a.Suspended, &a.CreatedAt, &a.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, accounts.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if bio.Valid {
		a.Bio = bio.String
	}
	if avatarURL.Valid {
		a.AvatarURL = avatarURL.String
	}
	if headerURL.Valid {
		a.HeaderURL = headerURL.String
	}
	if signature.Valid {
		a.Signature = signature.String
	}

	return &a, nil
}

// scanAccountFromRows scans an account from rows.
func (s *AccountsStore) scanAccountFromRows(rows *sql.Rows) (*accounts.Account, error) {
	var a accounts.Account
	var bio, avatarURL, headerURL, signature sql.NullString

	err := rows.Scan(
		&a.ID, &a.Username, &a.DisplayName, &a.Email,
		&bio, &avatarURL, &headerURL, &signature,
		&a.PostKarma, &a.CommentKarma, &a.TotalKarma, &a.TrustLevel,
		&a.Verified, &a.Admin, &a.Suspended, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if bio.Valid {
		a.Bio = bio.String
	}
	if avatarURL.Valid {
		a.AvatarURL = avatarURL.String
	}
	if headerURL.Valid {
		a.HeaderURL = headerURL.String
	}
	if signature.Valid {
		a.Signature = signature.String
	}

	return &a, nil
}
