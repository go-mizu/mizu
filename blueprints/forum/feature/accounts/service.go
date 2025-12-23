package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/pkg/password"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/ulid"
)

// Service implements the accounts API.
type Service struct {
	store Store
}

// NewService creates a new accounts service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new account.
func (s *Service) Create(ctx context.Context, in CreateIn) (*Account, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Check if username is taken
	existing, err := s.store.GetByUsername(ctx, in.Username)
	if err != nil && err != ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	// Check if email is taken
	existing, err = s.store.GetByEmail(ctx, in.Email)
	if err != nil && err != ErrNotFound {
		return nil, err
	}
	if existing != nil {
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
		Username:     in.Username,
		Email:        strings.ToLower(in.Email),
		PasswordHash: hash,
		DisplayName:  in.Username,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Create(ctx, account); err != nil {
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
	return s.store.GetByUsername(ctx, username)
}

// GetByEmail retrieves an account by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*Account, error) {
	return s.store.GetByEmail(ctx, strings.ToLower(email))
}

// Update updates an account.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Account, error) {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.DisplayName != nil {
		account.DisplayName = *in.DisplayName
	}
	if in.Bio != nil {
		bio := *in.Bio
		if len(bio) > BioMaxLen {
			bio = bio[:BioMaxLen]
		}
		account.Bio = bio
	}
	if in.AvatarURL != nil {
		account.AvatarURL = *in.AvatarURL
	}
	if in.BannerURL != nil {
		account.BannerURL = *in.BannerURL
	}

	account.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, account); err != nil {
		return nil, err
	}

	return account, nil
}

// UpdatePassword updates an account's password.
func (s *Service) UpdatePassword(ctx context.Context, id string, currentPassword, newPassword string) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify current password
	valid, err := password.Verify(currentPassword, account.PasswordHash)
	if err != nil || !valid {
		return ErrInvalidPassword
	}

	// Validate new password
	if len(newPassword) < PasswordMinLen {
		return ErrInvalidPassword
	}

	// Hash new password
	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}

	account.PasswordHash = hash
	account.UpdatedAt = time.Now()

	return s.store.Update(ctx, account)
}

// Delete deletes an account.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Login authenticates a user.
func (s *Service) Login(ctx context.Context, in LoginIn) (*Account, error) {
	account, err := s.store.GetByUsername(ctx, in.Username)
	if err != nil {
		// Also try email
		account, err = s.store.GetByEmail(ctx, strings.ToLower(in.Username))
		if err != nil {
			return nil, ErrInvalidPassword
		}
	}

	if account.IsSuspended {
		if account.SuspendUntil != nil && time.Now().Before(*account.SuspendUntil) {
			return nil, ErrAccountSuspended
		}
		// Suspension expired, unsuspend
		account.IsSuspended = false
		account.SuspendReason = ""
		account.SuspendUntil = nil
		_ = s.store.Update(ctx, account)
	}

	valid, err := password.Verify(in.Password, account.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidPassword
	}

	return account, nil
}

// CreateSession creates a new session for an account.
func (s *Service) CreateSession(ctx context.Context, accountID, userAgent, ip string) (*Session, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}

	now := time.Now()
	session := &Session{
		ID:        ulid.New(),
		AccountID: accountID,
		Token:     base64.URLEncoding.EncodeToString(token),
		UserAgent: userAgent,
		IP:        ip,
		ExpiresAt: now.Add(SessionTTL),
		CreatedAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*Session, error) {
	session, err := s.store.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if session.IsExpired() {
		_ = s.store.DeleteSession(ctx, token)
		return nil, ErrSessionExpired
	}

	return session, nil
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}

// DeleteAllSessions deletes all sessions for an account.
func (s *Service) DeleteAllSessions(ctx context.Context, accountID string) error {
	return s.store.DeleteSessionsByAccount(ctx, accountID)
}

// UpdateKarma updates an account's karma.
func (s *Service) UpdateKarma(ctx context.Context, id string, postDelta, commentDelta int64) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	account.PostKarma += postDelta
	account.CommentKarma += commentDelta
	account.Karma = account.PostKarma + account.CommentKarma
	account.UpdatedAt = time.Now()

	return s.store.Update(ctx, account)
}

// Suspend suspends an account.
func (s *Service) Suspend(ctx context.Context, id string, reason string, until *time.Time) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	account.IsSuspended = true
	account.SuspendReason = reason
	account.SuspendUntil = until
	account.UpdatedAt = time.Now()

	// Delete all sessions
	_ = s.store.DeleteSessionsByAccount(ctx, id)

	return s.store.Update(ctx, account)
}

// Unsuspend unsuspends an account.
func (s *Service) Unsuspend(ctx context.Context, id string) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	account.IsSuspended = false
	account.SuspendReason = ""
	account.SuspendUntil = nil
	account.UpdatedAt = time.Now()

	return s.store.Update(ctx, account)
}

// SetAdmin sets the admin status of an account.
func (s *Service) SetAdmin(ctx context.Context, id string, isAdmin bool) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	account.IsAdmin = isAdmin
	account.UpdatedAt = time.Now()

	return s.store.Update(ctx, account)
}

// List lists accounts.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Account, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	return s.store.List(ctx, opts)
}

// Search searches for accounts.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Account, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.store.Search(ctx, query, limit)
}
