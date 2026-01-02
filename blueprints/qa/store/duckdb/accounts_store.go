package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
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
			avatar_url, location, website_url, reputation,
			is_moderator, is_admin, is_suspended, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, account.ID, account.Username, account.Email, account.PasswordHash,
		account.DisplayName, account.Bio, account.AvatarURL, account.Location,
		account.WebsiteURL, account.Reputation, account.IsModerator, account.IsAdmin,
		account.IsSuspended, account.CreatedAt, account.UpdatedAt)
	return err
}

// GetByID retrieves an account by ID.
func (s *AccountsStore) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, location, website_url, reputation,
			is_moderator, is_admin, is_suspended, created_at, updated_at
		FROM accounts WHERE id = $1
	`, id))
}

// GetByIDs retrieves multiple accounts.
func (s *AccountsStore) GetByIDs(ctx context.Context, ids []string) (map[string]*accounts.Account, error) {
	if len(ids) == 0 {
		return make(map[string]*accounts.Account), nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, location, website_url, reputation,
			is_moderator, is_admin, is_suspended, created_at, updated_at
		FROM accounts WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*accounts.Account)
	for rows.Next() {
		account, err := s.scanAccountFromRows(rows)
		if err != nil {
			return nil, err
		}
		result[account.ID] = account
	}
	return result, rows.Err()
}

// GetByUsername retrieves an account by username.
func (s *AccountsStore) GetByUsername(ctx context.Context, username string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, location, website_url, reputation,
			is_moderator, is_admin, is_suspended, created_at, updated_at
		FROM accounts WHERE LOWER(username) = LOWER($1)
	`, username))
}

// GetByEmail retrieves an account by email.
func (s *AccountsStore) GetByEmail(ctx context.Context, email string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, location, website_url, reputation,
			is_moderator, is_admin, is_suspended, created_at, updated_at
		FROM accounts WHERE LOWER(email) = LOWER($1)
	`, email))
}

// Update updates an account.
func (s *AccountsStore) Update(ctx context.Context, account *accounts.Account) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE accounts SET
			username = $2, email = $3, password_hash = $4, display_name = $5,
			bio = $6, avatar_url = $7, location = $8, website_url = $9,
			reputation = $10, is_moderator = $11, is_admin = $12,
			is_suspended = $13, updated_at = $14
		WHERE id = $1
	`, account.ID, account.Username, account.Email, account.PasswordHash,
		account.DisplayName, account.Bio, account.AvatarURL, account.Location,
		account.WebsiteURL, account.Reputation, account.IsModerator,
		account.IsAdmin, account.IsSuspended, account.UpdatedAt)
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
		&session.ID, &session.AccountID, &session.Token, &session.UserAgent,
		&session.IP, &session.ExpiresAt, &session.CreatedAt,
	)
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
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	orderBy := "created_at DESC"
	if opts.OrderBy == "reputation" {
		orderBy = "reputation DESC"
	}

	query := `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, location, website_url, reputation,
			is_moderator, is_admin, is_suspended, created_at, updated_at
		FROM accounts
		ORDER BY ` + orderBy + `
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
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

// Search searches accounts.
func (s *AccountsStore) Search(ctx context.Context, query string, limit int) ([]*accounts.Account, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, email, password_hash, display_name, bio,
			avatar_url, location, website_url, reputation,
			is_moderator, is_admin, is_suspended, created_at, updated_at
		FROM accounts
		WHERE LOWER(username) LIKE LOWER($1) OR LOWER(display_name) LIKE LOWER($1)
		ORDER BY reputation DESC
		LIMIT $2
	`, "%"+query+"%", limit)
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
	if err := row.Scan(
		&account.ID,
		&account.Username,
		&account.Email,
		&account.PasswordHash,
		&account.DisplayName,
		&account.Bio,
		&account.AvatarURL,
		&account.Location,
		&account.WebsiteURL,
		&account.Reputation,
		&account.IsModerator,
		&account.IsAdmin,
		&account.IsSuspended,
		&account.CreatedAt,
		&account.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, accounts.ErrNotFound
		}
		return nil, err
	}
	return account, nil
}

func (s *AccountsStore) scanAccountFromRows(rows *sql.Rows) (*accounts.Account, error) {
	account := &accounts.Account{}
	if err := rows.Scan(
		&account.ID,
		&account.Username,
		&account.Email,
		&account.PasswordHash,
		&account.DisplayName,
		&account.Bio,
		&account.AvatarURL,
		&account.Location,
		&account.WebsiteURL,
		&account.Reputation,
		&account.IsModerator,
		&account.IsAdmin,
		&account.IsSuspended,
		&account.CreatedAt,
		&account.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return account, nil
}
