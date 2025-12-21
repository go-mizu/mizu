package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/microblog/pkg/password"
	"github.com/go-mizu/blueprints/microblog/pkg/ulid"
)

var (
	ErrNotFound         = errors.New("account not found")
	ErrUsernameTaken    = errors.New("username already taken")
	ErrEmailTaken       = errors.New("email already registered")
	ErrInvalidUsername  = errors.New("invalid username format")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrAccountSuspended = errors.New("account is suspended")
	ErrInvalidSession   = errors.New("invalid session")

	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,30}$`)
)

// Service handles account operations.
// Implements API interface.
type Service struct {
	store Store
}

// NewService creates a new accounts service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new account.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Account, error) {
	// Validate username
	if !usernameRegex.MatchString(in.Username) {
		return nil, ErrInvalidUsername
	}

	// Check username availability
	exists, err := s.store.ExistsUsername(ctx, in.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	// Check email availability
	if in.Email != "" {
		exists, err = s.store.ExistsEmail(ctx, in.Email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrEmailTaken
		}
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	// Create account
	now := time.Now()
	displayName := in.DisplayName
	if displayName == "" {
		displayName = in.Username
	}

	account := &Account{
		ID:          ulid.New(),
		Username:    strings.ToLower(in.Username),
		DisplayName: displayName,
		Email:       in.Email,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Insert(ctx, account, hash); err != nil {
		return nil, err
	}

	return account, nil
}

// GetByID retrieves an account by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Account, error) {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return account, nil
}

// GetByUsername retrieves an account by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*Account, error) {
	account, err := s.store.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrNotFound
	}
	return account, nil
}

// GetByEmail retrieves an account by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*Account, error) {
	account, err := s.store.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrNotFound
	}
	return account, nil
}

// Update updates an account.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Account, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Login authenticates a user and creates a session.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*Session, error) {
	id, hash, suspended, err := s.store.GetPasswordHash(ctx, in.Username)
	if err != nil {
		return nil, ErrNotFound
	}

	if suspended {
		return nil, ErrAccountSuspended
	}

	match, err := password.Verify(in.Password, hash)
	if err != nil || !match {
		return nil, ErrInvalidPassword
	}

	return s.CreateSession(ctx, id)
}

// CreateSession creates a new session for an account.
func (s *Service) CreateSession(ctx context.Context, accountID string) (*Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	session := &Session{
		ID:        ulid.New(),
		AccountID: accountID,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*Session, error) {
	session, err := s.store.GetSession(ctx, token)
	if err != nil {
		return nil, ErrInvalidSession
	}
	return session, nil
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}

// Verify marks an account as verified.
func (s *Service) Verify(ctx context.Context, id string, verified bool) error {
	return s.store.SetVerified(ctx, id, verified)
}

// Suspend suspends or unsuspends an account.
func (s *Service) Suspend(ctx context.Context, id string, suspended bool) error {
	return s.store.SetSuspended(ctx, id, suspended)
}

// SetAdmin sets admin status for an account.
func (s *Service) SetAdmin(ctx context.Context, id string, admin bool) error {
	return s.store.SetAdmin(ctx, id, admin)
}

// List returns a paginated list of accounts.
func (s *Service) List(ctx context.Context, limit, offset int) (*AccountList, error) {
	accounts, total, err := s.store.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return &AccountList{Accounts: accounts, Total: total}, nil
}

// Search searches for accounts by username or display name.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Account, error) {
	return s.store.Search(ctx, query, limit)
}
