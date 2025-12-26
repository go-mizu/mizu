package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/go-mizu/blueprints/messaging/pkg/password"
	"github.com/go-mizu/blueprints/messaging/pkg/ulid"
)

// Service implements the accounts API.
type Service struct {
	store Store
}

// NewService creates a new accounts service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new user account.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*User, error) {
	// Check username
	if in.Username != "" {
		exists, err := s.store.ExistsUsername(ctx, in.Username)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUsernameTaken
		}
	}

	// Check phone
	if in.Phone != "" {
		exists, err := s.store.ExistsPhone(ctx, in.Phone)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrPhoneTaken
		}
	}

	// Check email
	if in.Email != "" {
		exists, err := s.store.ExistsEmail(ctx, in.Email)
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

	now := time.Now()
	u := &User{
		ID:                  ulid.New(),
		Phone:               in.Phone,
		Email:               in.Email,
		Username:            in.Username,
		DisplayName:         in.DisplayName,
		Status:              "Hey there! I am using Messaging",
		IsOnline:            false,
		PrivacyLastSeen:     "everyone",
		PrivacyProfilePhoto: "everyone",
		PrivacyAbout:        "everyone",
		PrivacyGroups:       "everyone",
		PrivacyReadReceipts: true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if u.DisplayName == "" {
		u.DisplayName = u.Username
	}

	if err := s.store.Insert(ctx, u, hash); err != nil {
		return nil, err
	}

	return u, nil
}

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple users by IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) ([]*User, error) {
	return s.store.GetByIDs(ctx, ids)
}

// GetByUsername retrieves a user by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*User, error) {
	return s.store.GetByUsername(ctx, username)
}

// Update updates a user.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*User, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a user account.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ChangePassword changes a user's password.
func (s *Service) ChangePassword(ctx context.Context, userID string, in *ChangePasswordIn) error {
	// Get current password hash
	currentHash, err := s.store.GetPasswordHashByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if !password.Verify(in.CurrentPassword, currentHash) {
		return ErrInvalidCredentials
	}

	// Validate new password
	if err := password.Validate(in.NewPassword); err != nil {
		return err
	}

	// Hash new password
	newHash, err := password.Hash(in.NewPassword)
	if err != nil {
		return err
	}

	// Update password
	return s.store.UpdatePassword(ctx, userID, newHash)
}

// Login authenticates a user.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*Session, error) {
	id, hash, err := s.store.GetPasswordHash(ctx, in.Login)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !password.Verify(in.Password, hash) {
		return nil, ErrInvalidCredentials
	}

	// Create session
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sess := &Session{
		ID:           ulid.New(),
		UserID:       id,
		Token:        token,
		LastActiveAt: now,
		ExpiresAt:    now.Add(30 * 24 * time.Hour), // 30 days
		CreatedAt:    now,
	}

	if err := s.store.CreateSession(ctx, sess); err != nil {
		return nil, err
	}

	return sess, nil
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*Session, error) {
	return s.store.GetSession(ctx, token)
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}

// DeleteAllSessions deletes all sessions for a user.
func (s *Service) DeleteAllSessions(ctx context.Context, userID string) error {
	return s.store.DeleteAllSessions(ctx, userID)
}

// Search searches for users.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*User, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.store.Search(ctx, query, limit)
}

// UpdateOnlineStatus updates a user's online status.
func (s *Service) UpdateOnlineStatus(ctx context.Context, userID string, online bool) error {
	if !online {
		if err := s.store.UpdateLastSeen(ctx, userID); err != nil {
			return err
		}
	}
	return s.store.UpdateOnlineStatus(ctx, userID, online)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
