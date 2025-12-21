package accounts

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/microblog/pkg/password"
	"github.com/go-mizu/blueprints/microblog/pkg/ulid"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

var (
	ErrNotFound          = errors.New("account not found")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrEmailTaken        = errors.New("email already registered")
	ErrInvalidUsername   = errors.New("invalid username format")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrAccountSuspended  = errors.New("account is suspended")
	ErrInvalidSession    = errors.New("invalid session")

	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,30}$`)
)

// Service handles account operations.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new accounts service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

// Create creates a new account.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Account, error) {
	// Validate username
	if !usernameRegex.MatchString(in.Username) {
		return nil, ErrInvalidUsername
	}

	// Check username availability
	var exists bool
	err := s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE LOWER(username) = LOWER($1))", in.Username).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("accounts: check username: %w", err)
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	// Check email availability
	if in.Email != "" {
		err = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE LOWER(email) = LOWER($1))", in.Email).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("accounts: check email: %w", err)
		}
		if exists {
			return nil, ErrEmailTaken
		}
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("accounts: hash password: %w", err)
	}

	// Create account
	id := ulid.New()
	now := time.Now()

	displayName := in.DisplayName
	if displayName == "" {
		displayName = in.Username
	}

	_, err = s.store.Exec(ctx, `
		INSERT INTO accounts (id, username, display_name, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, strings.ToLower(in.Username), displayName, in.Email, hash, now, now)
	if err != nil {
		return nil, fmt.Errorf("accounts: insert: %w", err)
	}

	return &Account{
		ID:          id,
		Username:    strings.ToLower(in.Username),
		DisplayName: displayName,
		Email:       in.Email,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetByID retrieves an account by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Account, error) {
	return s.scanAccount(s.store.QueryRow(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE id = $1
	`, id))
}

// GetByUsername retrieves an account by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*Account, error) {
	return s.scanAccount(s.store.QueryRow(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE LOWER(username) = LOWER($1)
	`, username))
}

// GetByEmail retrieves an account by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*Account, error) {
	return s.scanAccount(s.store.QueryRow(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE LOWER(email) = LOWER($1)
	`, email))
}

func (s *Service) scanAccount(row *sql.Row) (*Account, error) {
	var a Account
	var bio, avatarURL, headerURL, fieldsJSON sql.NullString
	var email sql.NullString

	err := row.Scan(
		&a.ID, &a.Username, &a.DisplayName, &email,
		&bio, &avatarURL, &headerURL, &fieldsJSON,
		&a.Verified, &a.Admin, &a.Suspended, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("accounts: scan: %w", err)
	}

	a.Email = email.String
	a.Bio = bio.String
	a.AvatarURL = avatarURL.String
	a.HeaderURL = headerURL.String

	if fieldsJSON.Valid && fieldsJSON.String != "" {
		_ = json.Unmarshal([]byte(fieldsJSON.String), &a.Fields)
	}

	return &a, nil
}

// Update updates an account.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Account, error) {
	// Build update query dynamically
	var sets []string
	var args []any
	argIdx := 1

	if in.DisplayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *in.DisplayName)
		argIdx++
	}
	if in.Bio != nil {
		sets = append(sets, fmt.Sprintf("bio = $%d", argIdx))
		args = append(args, *in.Bio)
		argIdx++
	}
	if in.AvatarURL != nil {
		sets = append(sets, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *in.AvatarURL)
		argIdx++
	}
	if in.HeaderURL != nil {
		sets = append(sets, fmt.Sprintf("header_url = $%d", argIdx))
		args = append(args, *in.HeaderURL)
		argIdx++
	}
	if in.Fields != nil {
		fieldsJSON, _ := json.Marshal(in.Fields)
		sets = append(sets, fmt.Sprintf("fields = $%d", argIdx))
		args = append(args, string(fieldsJSON))
		argIdx++
	}

	if len(sets) == 0 {
		return s.GetByID(ctx, id)
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE accounts SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)

	_, err := s.store.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("accounts: update: %w", err)
	}

	return s.GetByID(ctx, id)
}

// Login authenticates a user and creates a session.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*Session, error) {
	// Get account
	var id, hash string
	var suspended bool
	err := s.store.QueryRow(ctx, `
		SELECT id, password_hash, suspended
		FROM accounts WHERE LOWER(username) = LOWER($1) OR LOWER(email) = LOWER($1)
	`, in.Username).Scan(&id, &hash, &suspended)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("accounts: login query: %w", err)
	}

	if suspended {
		return nil, ErrAccountSuspended
	}

	// Verify password
	match, err := password.Verify(in.Password, hash)
	if err != nil || !match {
		return nil, ErrInvalidPassword
	}

	// Create session
	return s.CreateSession(ctx, id)
}

// CreateSession creates a new session for an account.
func (s *Service) CreateSession(ctx context.Context, accountID string) (*Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("accounts: generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	session := &Session{
		ID:        ulid.New(),
		AccountID: accountID,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
		CreatedAt: time.Now(),
	}

	_, err := s.store.Exec(ctx, `
		INSERT INTO sessions (id, account_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, session.ID, session.AccountID, session.Token, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("accounts: create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*Session, error) {
	var session Session
	err := s.store.QueryRow(ctx, `
		SELECT id, account_id, token, expires_at, created_at
		FROM sessions WHERE token = $1 AND expires_at > CURRENT_TIMESTAMP
	`, token).Scan(&session.ID, &session.AccountID, &session.Token, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidSession
		}
		return nil, fmt.Errorf("accounts: get session: %w", err)
	}
	return &session, nil
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	_, err := s.store.Exec(ctx, "DELETE FROM sessions WHERE token = $1", token)
	return err
}

// Verify marks an account as verified.
func (s *Service) Verify(ctx context.Context, id string, verified bool) error {
	_, err := s.store.Exec(ctx, "UPDATE accounts SET verified = $1, updated_at = $2 WHERE id = $3",
		verified, time.Now(), id)
	return err
}

// Suspend suspends or unsuspends an account.
func (s *Service) Suspend(ctx context.Context, id string, suspended bool) error {
	_, err := s.store.Exec(ctx, "UPDATE accounts SET suspended = $1, updated_at = $2 WHERE id = $3",
		suspended, time.Now(), id)
	return err
}

// SetAdmin sets admin status for an account.
func (s *Service) SetAdmin(ctx context.Context, id string, admin bool) error {
	_, err := s.store.Exec(ctx, "UPDATE accounts SET admin = $1, updated_at = $2 WHERE id = $3",
		admin, time.Now(), id)
	return err
}

// List returns a paginated list of accounts.
func (s *Service) List(ctx context.Context, limit, offset int) (*AccountList, error) {
	// Get total count
	var total int
	err := s.store.QueryRow(ctx, "SELECT count(*) FROM accounts WHERE suspended = FALSE").Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("accounts: count: %w", err)
	}

	// Get accounts
	rows, err := s.store.Query(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE suspended = FALSE
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("accounts: list: %w", err)
	}
	defer rows.Close()

	var accounts []*Account
	for rows.Next() {
		var a Account
		var bio, avatarURL, headerURL, fieldsJSON, email sql.NullString

		if err := rows.Scan(
			&a.ID, &a.Username, &a.DisplayName, &email,
			&bio, &avatarURL, &headerURL, &fieldsJSON,
			&a.Verified, &a.Admin, &a.Suspended, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("accounts: scan row: %w", err)
		}

		a.Email = email.String
		a.Bio = bio.String
		a.AvatarURL = avatarURL.String
		a.HeaderURL = headerURL.String

		if fieldsJSON.Valid && fieldsJSON.String != "" {
			_ = json.Unmarshal([]byte(fieldsJSON.String), &a.Fields)
		}

		accounts = append(accounts, &a)
	}

	return &AccountList{Accounts: accounts, Total: total}, nil
}

// Search searches for accounts by username or display name.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Account, error) {
	rows, err := s.store.Query(ctx, `
		SELECT id, username, display_name, bio, avatar_url, verified, created_at
		FROM accounts
		WHERE suspended = FALSE AND (
			LOWER(username) LIKE LOWER($1) || '%'
			OR LOWER(display_name) LIKE '%' || LOWER($1) || '%'
		)
		ORDER BY
			CASE WHEN LOWER(username) = LOWER($1) THEN 0 ELSE 1 END,
			CASE WHEN LOWER(username) LIKE LOWER($1) || '%' THEN 0 ELSE 1 END,
			LENGTH(username)
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("accounts: search: %w", err)
	}
	defer rows.Close()

	var accounts []*Account
	for rows.Next() {
		var a Account
		var bio, avatarURL sql.NullString

		if err := rows.Scan(&a.ID, &a.Username, &a.DisplayName, &bio, &avatarURL, &a.Verified, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("accounts: scan search: %w", err)
		}

		a.Bio = bio.String
		a.AvatarURL = avatarURL.String
		accounts = append(accounts, &a)
	}

	return accounts, nil
}
