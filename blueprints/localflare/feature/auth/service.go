package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"
)

// Service implements the Auth API.
type Service struct {
	store Store
}

// NewService creates a new Auth service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Login authenticates a user.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*LoginResult, error) {
	if in.Email == "" {
		return nil, ErrEmailRequired
	}
	if in.Password == "" {
		return nil, ErrPasswordRequired
	}

	user, err := s.store.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        ulid.Make().String(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
		User: &UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Role:  user.Role,
		},
	}, nil
}

// Register creates a new user.
func (s *Service) Register(ctx context.Context, in *RegisterIn) (*LoginResult, error) {
	if in.Email == "" {
		return nil, ErrEmailRequired
	}
	if in.Password == "" {
		return nil, ErrPasswordRequired
	}

	// Check if user exists
	if existing, _ := s.store.GetByEmail(ctx, in.Email); existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           ulid.Make().String(),
		Email:        in.Email,
		Name:         in.Name,
		PasswordHash: string(hash),
		Role:         "user",
		CreatedAt:    time.Now(),
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        ulid.Make().String(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
		User: &UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Role:  user.Role,
		},
	}, nil
}

// Logout invalidates a session.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}

// GetCurrentUser returns the current user.
func (s *Service) GetCurrentUser(ctx context.Context, token string) (*UserInfo, error) {
	session, err := s.store.GetSession(ctx, token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, ErrSessionExpired
	}

	user, err := s.store.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
