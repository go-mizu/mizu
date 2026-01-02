package accounts

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/pkg/password"
	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
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

	if existing, _ := s.store.GetByUsername(ctx, in.Username); existing != nil {
		return nil, ErrUsernameTaken
	}
	if existing, _ := s.store.GetByEmail(ctx, in.Email); existing != nil {
		return nil, ErrEmailTaken
	}

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
		Reputation:   1,
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

// GetByIDs retrieves accounts by IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) (map[string]*Account, error) {
	return s.store.GetByIDs(ctx, ids)
}

// GetByUsername retrieves an account by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*Account, error) {
	return s.store.GetByUsername(ctx, username)
}

// GetByEmail retrieves an account by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*Account, error) {
	return s.store.GetByEmail(ctx, email)
}

// Update updates an account profile.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Account, error) {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.DisplayName != nil {
		account.DisplayName = *in.DisplayName
	}
	if in.Bio != nil {
		if len(*in.Bio) > BioMaxLen {
			bio := (*in.Bio)[:BioMaxLen]
			account.Bio = bio
		} else {
			account.Bio = *in.Bio
		}
	}
	if in.AvatarURL != nil {
		account.AvatarURL = *in.AvatarURL
	}
	if in.Location != nil {
		account.Location = *in.Location
	}
	if in.WebsiteURL != nil {
		account.WebsiteURL = *in.WebsiteURL
	}

	account.UpdatedAt = time.Now()
	if err := s.store.Update(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

// UpdatePassword changes the account password.
func (s *Service) UpdatePassword(ctx context.Context, id string, currentPassword, newPassword string) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	valid, err := password.Verify(currentPassword, account.PasswordHash)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidPassword
	}
	if len(newPassword) < PasswordMinLen {
		return ErrInvalidPassword
	}
	newHash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}
	account.PasswordHash = newHash
	account.UpdatedAt = time.Now()
	return s.store.Update(ctx, account)
}

// Delete removes an account.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Login authenticates a user.
func (s *Service) Login(ctx context.Context, in LoginIn) (*Account, error) {
	account, err := s.store.GetByUsername(ctx, in.Username)
	if err != nil {
		return nil, err
	}
	if account.IsSuspended {
		return nil, ErrAccountSuspended
	}
	valid, err := password.Verify(in.Password, account.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidPassword
	}
	return account, nil
}

// CreateSession creates a new session.
func (s *Service) CreateSession(ctx context.Context, accountID, userAgent, ip string) (*Session, error) {
	session := &Session{
		ID:        ulid.New(),
		AccountID: accountID,
		Token:     ulid.New(),
		UserAgent: userAgent,
		IP:        ip,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(SessionTTL),
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

// DeleteSession deletes a session by token.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}

// DeleteAllSessions deletes all sessions for an account.
func (s *Service) DeleteAllSessions(ctx context.Context, accountID string) error {
	return s.store.DeleteSessionsByAccount(ctx, accountID)
}

// UpdateReputation updates user reputation.
func (s *Service) UpdateReputation(ctx context.Context, id string, delta int64) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	account.Reputation += delta
	if account.Reputation < 1 {
		account.Reputation = 1
	}
	account.UpdatedAt = time.Now()
	return s.store.Update(ctx, account)
}

// SetModerator toggles moderator flag.
func (s *Service) SetModerator(ctx context.Context, id string, isModerator bool) error {
	account, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	account.IsModerator = isModerator
	account.UpdatedAt = time.Now()
	return s.store.Update(ctx, account)
}

// SetAdmin toggles admin flag.
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
	return s.store.List(ctx, opts)
}

// Search searches accounts.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Account, error) {
	return s.store.Search(ctx, query, limit)
}
