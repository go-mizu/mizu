package accounts

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/go-mizu/blueprints/drive/pkg/crypto"
	"github.com/go-mizu/blueprints/drive/pkg/password"
	"github.com/go-mizu/blueprints/drive/pkg/ulid"
)

const (
	DefaultQuota     = 15 * 1024 * 1024 * 1024 // 15 GB
	SessionDuration  = 30 * 24 * time.Hour     // 30 days
	MinPasswordLen   = 8
)

// Service implements the accounts API.
type Service struct {
	store Store
}

// NewService creates a new accounts service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Register creates a new account.
func (s *Service) Register(ctx context.Context, in *RegisterIn) (*Account, error) {
	// Validate input
	username := strings.TrimSpace(in.Username)
	if len(username) < 3 || len(username) > 50 {
		return nil, ErrWeakPassword
	}
	if !isValidUsername(username) {
		return nil, ErrWeakPassword
	}

	email := strings.TrimSpace(strings.ToLower(in.Email))
	if !strings.Contains(email, "@") {
		return nil, ErrWeakPassword
	}

	if len(in.Password) < MinPasswordLen {
		return nil, ErrWeakPassword
	}

	// Check uniqueness
	if existing, _ := s.store.GetByUsername(ctx, username); existing != nil {
		return nil, ErrUsernameTaken
	}
	if existing, _ := s.store.GetByEmail(ctx, email); existing != nil {
		return nil, ErrEmailTaken
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	account := &Account{
		ID:           ulid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(in.DisplayName),
		StorageQuota: DefaultQuota,
		StorageUsed:  0,
		Preferences:  "{}", // Valid empty JSON
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Create(ctx, account); err != nil {
		return nil, err
	}

	return account, nil
}

// Login authenticates a user and creates a session.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*Session, *Account, error) {
	account, err := s.store.GetByUsername(ctx, strings.TrimSpace(in.Username))
	if err != nil {
		// Also try email
		account, err = s.store.GetByEmail(ctx, strings.TrimSpace(strings.ToLower(in.Username)))
		if err != nil {
			return nil, nil, ErrInvalidCredentials
		}
	}

	if account.IsSuspended {
		return nil, nil, ErrAccountSuspended
	}

	// Verify password
	match, err := password.Verify(in.Password, account.PasswordHash)
	if err != nil || !match {
		return nil, nil, ErrInvalidCredentials
	}

	// Create session
	token, err := crypto.GenerateSessionToken()
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	session := &Session{
		ID:        ulid.New(),
		AccountID: account.ID,
		Token:     token,
		UserAgent: in.UserAgent,
		IPAddress: in.IPAddress,
		LastUsed:  now,
		ExpiresAt: now.Add(SessionDuration),
		CreatedAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return session, account, nil
}

// Logout invalidates a session.
func (s *Service) Logout(ctx context.Context, token string) error {
	session, err := s.store.GetSessionByToken(ctx, token)
	if err != nil {
		return nil // Already logged out
	}
	return s.store.DeleteSession(ctx, session.ID)
}

// GetByID retrieves an account by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Account, error) {
	return s.store.GetByID(ctx, id)
}

// GetByToken retrieves account and session by token.
func (s *Service) GetByToken(ctx context.Context, token string) (*Account, *Session, error) {
	session, err := s.store.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, nil, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		s.store.DeleteSession(ctx, session.ID)
		return nil, nil, ErrSessionExpired
	}

	account, err := s.store.GetByID(ctx, session.AccountID)
	if err != nil {
		return nil, nil, err
	}

	// Update last used
	s.store.UpdateSessionLastUsed(ctx, session.ID)

	return account, session, nil
}

// Update updates account profile.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Account, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// ChangePassword changes account password.
func (s *Service) ChangePassword(ctx context.Context, id string, in *ChangePasswordIn) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify current password
	match, err := password.Verify(in.CurrentPassword, account.PasswordHash)
	if err != nil || !match {
		return ErrInvalidPassword
	}

	if len(in.NewPassword) < MinPasswordLen {
		return ErrWeakPassword
	}

	hash, err := password.Hash(in.NewPassword)
	if err != nil {
		return err
	}

	return s.store.UpdatePassword(ctx, id, hash)
}

// GetStorageUsage returns storage usage breakdown.
func (s *Service) GetStorageUsage(ctx context.Context, id string) (*StorageUsage, error) {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	breakdown, err := s.store.GetStorageByCategory(ctx, id)
	if err != nil {
		return nil, err
	}

	available := account.StorageQuota - account.StorageUsed
	if available < 0 {
		available = 0
	}

	percent := 0.0
	if account.StorageQuota > 0 {
		percent = float64(account.StorageUsed) / float64(account.StorageQuota) * 100
	}

	return &StorageUsage{
		Quota:     account.StorageQuota,
		Used:      account.StorageUsed,
		Available: available,
		Percent:   percent,
		Breakdown: breakdown,
	}, nil
}

// UpdateStorageUsed adjusts storage usage.
func (s *Service) UpdateStorageUsed(ctx context.Context, id string, delta int64) error {
	return s.store.UpdateStorageUsed(ctx, id, delta)
}

// ListSessions lists all sessions for an account.
func (s *Service) ListSessions(ctx context.Context, accountID string) ([]*Session, error) {
	return s.store.ListSessionsByAccount(ctx, accountID)
}

// RevokeSession revokes a specific session.
func (s *Service) RevokeSession(ctx context.Context, accountID, sessionID string) error {
	sessions, err := s.store.ListSessionsByAccount(ctx, accountID)
	if err != nil {
		return err
	}

	for _, sess := range sessions {
		if sess.ID == sessionID {
			return s.store.DeleteSession(ctx, sessionID)
		}
	}

	return ErrSessionNotFound
}

func isValidUsername(username string) bool {
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}
