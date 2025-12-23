package duckdb

import (
	"context"
	"database/sql"
	"strings"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
)

// AccountsStore implements accounts.Store.
type AccountsStore struct {
	db *sql.DB
}

// NewAccountsStore creates a new accounts store.
func NewAccountsStore(db *sql.DB) *AccountsStore {
	return &AccountsStore{db: db}
}

// Create creates an account.
func (s *AccountsStore) Create(ctx context.Context, account *accounts.Account) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (
			id, username, email, password_hash, display_name, bio,
			avatar_url, banner_url, karma, post_karma, comment_karma,
			is_admin, is_suspended, suspend_reason, suspend_until,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, account.ID, account.Username, account.Email, account.PasswordHash,
		account.DisplayName, account.Bio, account.AvatarURL, account.BannerURL,
		account.Karma, account.PostKarma, account.CommentKarma,
		account.IsAdmin, account.IsSuspended, account.SuspendReason, account.SuspendUntil,
		account.CreatedAt, account.UpdatedAt)
	return err
}

// GetByID retrieves an account by ID.
func (s *AccountsStore) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, banner_url, karma, post_karma, comment_karma,
			is_admin, is_suspended, suspend_reason, suspend_until,
			created_at, updated_at
		FROM accounts WHERE id = $1
	`, id))
}

// GetByUsername retrieves an account by username.
func (s *AccountsStore) GetByUsername(ctx context.Context, username string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, banner_url, karma, post_karma, comment_karma,
			is_admin, is_suspended, suspend_reason, suspend_until,
			created_at, updated_at
		FROM accounts WHERE LOWER(username) = LOWER($1)
	`, username))
}

// GetByEmail retrieves an account by email.
func (s *AccountsStore) GetByEmail(ctx context.Context, email string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, banner_url, karma, post_karma, comment_karma,
			is_admin, is_suspended, suspend_reason, suspend_until,
			created_at, updated_at
		FROM accounts WHERE LOWER(email) = LOWER($1)
	`, email))
}

// Update updates an account.
func (s *AccountsStore) Update(ctx context.Context, account *accounts.Account) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE accounts SET
			username = $2, email = $3, password_hash = $4, display_name = $5,
			bio = $6, avatar_url = $7, banner_url = $8, karma = $9,
			post_karma = $10, comment_karma = $11, is_admin = $12,
			is_suspended = $13, suspend_reason = $14, suspend_until = $15,
			updated_at = $16
		WHERE id = $1
	`, account.ID, account.Username, account.Email, account.PasswordHash,
		account.DisplayName, account.Bio, account.AvatarURL, account.BannerURL,
		account.Karma, account.PostKarma, account.CommentKarma,
		account.IsAdmin, account.IsSuspended, account.SuspendReason, account.SuspendUntil,
		account.UpdatedAt)
	return err
}

// Delete deletes an account.
func (s *AccountsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1`, id)
	return err
}

// CreateSession creates a session.
func (s *AccountsStore) CreateSession(ctx context.Context, session *accounts.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, account_id, token, user_agent, ip, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, session.ID, session.AccountID, session.Token, session.UserAgent, session.IP,
		session.ExpiresAt, session.CreatedAt)
	return err
}

// GetSessionByToken retrieves a session by token.
func (s *AccountsStore) GetSessionByToken(ctx context.Context, token string) (*accounts.Session, error) {
	session := &accounts.Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, token, user_agent, ip, expires_at, created_at
		FROM sessions WHERE token = $1
	`, token).Scan(
		&session.ID, &session.AccountID, &session.Token,
		&session.UserAgent, &session.IP, &session.ExpiresAt, &session.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, accounts.ErrSessionExpired
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

// DeleteSession deletes a session.
func (s *AccountsStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

// DeleteSessionsByAccount deletes all sessions for an account.
func (s *AccountsStore) DeleteSessionsByAccount(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE account_id = $1`, accountID)
	return err
}

// CleanExpiredSessions removes expired sessions.
func (s *AccountsStore) CleanExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`)
	return err
}

// List lists accounts.
func (s *AccountsStore) List(ctx context.Context, opts accounts.ListOpts) ([]*accounts.Account, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, banner_url, karma, post_karma, comment_karma,
			is_admin, is_suspended, suspend_reason, suspend_until,
			created_at, updated_at
		FROM accounts
	`

	orderBy := "created_at DESC"
	if opts.OrderBy == "karma" {
		orderBy = "karma DESC"
	}
	query += " ORDER BY " + orderBy
	query += " LIMIT $1"

	rows, err := s.db.QueryContext(ctx, query, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*accounts.Account
	for rows.Next() {
		account, err := s.scanAccountFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, account)
	}
	return result, rows.Err()
}

// Search searches for accounts.
func (s *AccountsStore) Search(ctx context.Context, query string, limit int) ([]*accounts.Account, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, banner_url, karma, post_karma, comment_karma,
			is_admin, is_suspended, suspend_reason, suspend_until,
			created_at, updated_at
		FROM accounts
		WHERE LOWER(username) LIKE $1 OR LOWER(display_name) LIKE $1
		ORDER BY karma DESC
		LIMIT $2
	`, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*accounts.Account
	for rows.Next() {
		account, err := s.scanAccountFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, account)
	}
	return result, rows.Err()
}

func (s *AccountsStore) scanAccount(row *sql.Row) (*accounts.Account, error) {
	account := &accounts.Account{}
	var suspendReason sql.NullString
	var suspendUntil sql.NullTime

	err := row.Scan(
		&account.ID, &account.Username, &account.Email, &account.PasswordHash,
		&account.DisplayName, &account.Bio, &account.AvatarURL, &account.BannerURL,
		&account.Karma, &account.PostKarma, &account.CommentKarma,
		&account.IsAdmin, &account.IsSuspended, &suspendReason, &suspendUntil,
		&account.CreatedAt, &account.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, accounts.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if suspendReason.Valid {
		account.SuspendReason = suspendReason.String
	}
	if suspendUntil.Valid {
		account.SuspendUntil = &suspendUntil.Time
	}

	return account, nil
}

func (s *AccountsStore) scanAccountFromRows(rows *sql.Rows) (*accounts.Account, error) {
	account := &accounts.Account{}
	var suspendReason sql.NullString
	var suspendUntil sql.NullTime

	err := rows.Scan(
		&account.ID, &account.Username, &account.Email, &account.PasswordHash,
		&account.DisplayName, &account.Bio, &account.AvatarURL, &account.BannerURL,
		&account.Karma, &account.PostKarma, &account.CommentKarma,
		&account.IsAdmin, &account.IsSuspended, &suspendReason, &suspendUntil,
		&account.CreatedAt, &account.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if suspendReason.Valid {
		account.SuspendReason = suspendReason.String
	}
	if suspendUntil.Valid {
		account.SuspendUntil = &suspendUntil.Time
	}

	return account, nil
}
