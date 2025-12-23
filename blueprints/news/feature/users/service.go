package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/pkg/password"
	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
)

// Service implements the users.API interface.
type Service struct {
	store Store
}

// NewService creates a new users service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new user.
func (s *Service) Create(ctx context.Context, in CreateIn) (*User, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Check if username is taken
	if existing, _ := s.store.GetByUsername(ctx, in.Username); existing != nil {
		return nil, ErrUsernameTaken
	}

	// Check if email is taken
	if existing, _ := s.store.GetByEmail(ctx, in.Email); existing != nil {
		return nil, ErrEmailTaken
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           ulid.New(),
		Username:     in.Username,
		Email:        strings.ToLower(in.Email),
		PasswordHash: hash,
		Karma:        1,
		CreatedAt:    time.Now(),
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple users by their IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) (map[string]*User, error) {
	return s.store.GetByIDs(ctx, ids)
}

// GetByUsername retrieves a user by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*User, error) {
	return s.store.GetByUsername(ctx, username)
}

// Update updates a user.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.About != nil {
		user.About = *in.About
	}

	if err := s.store.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login validates credentials and returns the user.
func (s *Service) Login(ctx context.Context, in LoginIn) (*User, error) {
	user, err := s.store.GetByUsername(ctx, in.Username)
	if err != nil {
		return nil, ErrNotFound
	}

	valid, err := password.Verify(in.Password, user.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidPassword
	}

	return user, nil
}

// CreateSession creates a new session for a user.
func (s *Service) CreateSession(ctx context.Context, userID string) (*Session, error) {
	token := generateToken()

	session := &Session{
		ID:        ulid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(SessionTTL),
		CreatedAt: time.Now(),
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

// UpdateKarma updates a user's karma.
func (s *Service) UpdateKarma(ctx context.Context, id string, delta int64) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	user.Karma += delta
	return s.store.Update(ctx, user)
}

// SetAdmin sets a user's admin status.
func (s *Service) SetAdmin(ctx context.Context, id string, isAdmin bool) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	user.IsAdmin = isAdmin
	return s.store.Update(ctx, user)
}

// List lists users.
func (s *Service) List(ctx context.Context, limit, offset int) ([]*User, error) {
	if limit <= 0 {
		limit = 30
	}
	return s.store.List(ctx, limit, offset)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
