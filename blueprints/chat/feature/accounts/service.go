package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/go-mizu/blueprints/chat/pkg/password"
	"github.com/go-mizu/blueprints/chat/pkg/ulid"
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
	// Check if username exists
	exists, err := s.store.ExistsUsername(ctx, in.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		// Get next discriminator
		disc, err := s.store.GetNextDiscriminator(ctx, in.Username)
		if err != nil {
			return nil, ErrUsernameTaken
		}
		// Create with new discriminator
		return s.createWithDiscriminator(ctx, in, disc)
	}

	return s.createWithDiscriminator(ctx, in, "0001")
}

func (s *Service) createWithDiscriminator(ctx context.Context, in *CreateIn, disc string) (*User, error) {
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
		ID:            ulid.New(),
		Username:      in.Username,
		Discriminator: disc,
		DisplayName:   in.DisplayName,
		Email:         in.Email,
		Status:        StatusOffline,
		CreatedAt:     now,
		UpdatedAt:     now,
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
		ID:        ulid.New(),
		UserID:    id,
		Token:     token,
		ExpiresAt: now.Add(30 * 24 * time.Hour), // 30 days
		CreatedAt: now,
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

// Search searches for users.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*User, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.store.Search(ctx, query, limit)
}

// UpdateStatus updates a user's status.
func (s *Service) UpdateStatus(ctx context.Context, userID string, status Status) error {
	return s.store.UpdateStatus(ctx, userID, string(status))
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
