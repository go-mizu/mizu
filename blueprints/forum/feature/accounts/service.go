package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/forum/pkg/password"
	"github.com/go-mizu/blueprints/forum/pkg/ulid"
)

// Service handles account operations.
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

	// Check if username exists
	exists, err := s.store.ExistsUsername(ctx, strings.ToLower(in.Username))
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	// Check if email exists
	if in.Email != "" {
		exists, err := s.store.ExistsEmail(ctx, strings.ToLower(in.Email))
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
	account := &Account{
		ID:           ulid.New(),
		Username:     strings.ToLower(in.Username),
		DisplayName:  in.DisplayName,
		Email:        strings.ToLower(in.Email),
		PostKarma:    0,
		CommentKarma: 0,
		TotalKarma:   0,
		TrustLevel:   0,
		Verified:     false,
		Admin:        false,
		Suspended:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if account.DisplayName == "" {
		account.DisplayName = account.Username
	}

	if err := s.store.Insert(ctx, account, hash); err != nil {
		return nil, err
	}

	return account, nil
}

// GetByID retrieves an account by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Account, error) {
	return s.store.GetByID(ctx, id)
}

// GetByUsername retrieves an account by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*Account, error) {
	return s.store.GetByUsername(ctx, strings.ToLower(username))
}

// Update updates an account.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Account, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// List lists accounts with pagination.
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

// Login authenticates a user and creates a session.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*Session, error) {
	// Get account and password hash
	id, hash, suspended, err := s.store.GetPasswordHash(ctx, in.UsernameOrEmail)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if suspended {
		return nil, ErrSuspended
	}

	// Verify password
	valid, err := password.Verify(hash, in.Password)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}

	// Generate session token
	token := generateToken()

	// Create session (expires in 30 days)
	session := &Session{
		Token:     token,
		AccountID: id,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	// Load account
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	session.Account = account

	return session, nil
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*Session, error) {
	session, err := s.store.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		s.store.DeleteSession(ctx, token)
		return nil, ErrNotFound
	}

	// Load account
	account, err := s.store.GetByID(ctx, session.AccountID)
	if err != nil {
		return nil, err
	}

	if account.Suspended {
		return nil, ErrSuspended
	}

	session.Account = account
	return session, nil
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}

// AddKarma adds karma to an account.
func (s *Service) AddKarma(ctx context.Context, accountID string, postKarma, commentKarma int) error {
	if err := s.store.AddKarma(ctx, accountID, postKarma, commentKarma); err != nil {
		return err
	}

	// Update trust level based on new karma
	return s.UpdateTrustLevel(ctx, accountID)
}

// UpdateTrustLevel calculates and updates trust level based on karma.
func (s *Service) UpdateTrustLevel(ctx context.Context, accountID string) error {
	account, err := s.store.GetByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Calculate trust level based on karma and activity
	// Level 0: New user (0 karma)
	// Level 1: Basic (10+ karma)
	// Level 2: Member (100+ karma)
	// Level 3: Regular (1000+ karma)
	// Level 4: Leader (granted by admins)

	newLevel := 0
	if account.TotalKarma >= 10 {
		newLevel = 1
	}
	if account.TotalKarma >= 100 {
		newLevel = 2
	}
	if account.TotalKarma >= 1000 {
		newLevel = 3
	}

	// Don't downgrade level 4 (leader)
	if account.TrustLevel == 4 {
		return nil
	}

	// Update if changed (simplified - would need a separate method in store)
	_ = newLevel // TODO: Implement trust level update in store

	return nil
}

// SetVerified sets the verified status.
func (s *Service) SetVerified(ctx context.Context, id string, verified bool) error {
	return s.store.SetVerified(ctx, id, verified)
}

// SetSuspended sets the suspended status.
func (s *Service) SetSuspended(ctx context.Context, id string, suspended bool) error {
	return s.store.SetSuspended(ctx, id, suspended)
}

// SetAdmin sets the admin status.
func (s *Service) SetAdmin(ctx context.Context, id string, admin bool) error {
	return s.store.SetAdmin(ctx, id, admin)
}

// generateToken generates a random session token.
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
