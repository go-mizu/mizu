package users

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/pkg/jwt"
	"github.com/go-mizu/blueprints/spreadsheet/pkg/password"
	"github.com/go-mizu/blueprints/spreadsheet/pkg/ulid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")
)

// Service implements the users API.
type Service struct {
	store  Store
	secret string
}

// NewService creates a new users service.
func NewService(store Store) *Service {
	return &Service{
		store:  store,
		secret: jwt.GenerateSecret(),
	}
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, in *RegisterIn) (*User, string, error) {
	// Check if email exists
	existing, _ := s.store.GetByEmail(ctx, in.Email)
	if existing != nil {
		return nil, "", ErrEmailExists
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	user := &User{
		ID:        ulid.New(),
		Email:     in.Email,
		Name:      in.Name,
		Password:  hash,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, "", err
	}

	// Generate token
	token, err := jwt.Token(s.secret, user.ID, 24*time.Hour)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login authenticates a user.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*User, string, error) {
	user, err := s.store.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if !password.Verify(in.Password, user.Password) {
		return nil, "", ErrInvalidCredentials
	}

	token, err := jwt.Token(s.secret, user.ID, 24*time.Hour)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.store.GetByID(ctx, id)
}

// GetByEmail retrieves a user by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetByEmail(ctx, email)
}

// Update updates a user's profile.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != "" {
		user.Name = in.Name
	}
	if in.Avatar != "" {
		user.Avatar = in.Avatar
	}
	user.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// VerifyToken verifies a JWT token.
func (s *Service) VerifyToken(ctx context.Context, token string) (string, error) {
	claims, err := jwt.Verify(token, s.secret)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// SetSecret sets the JWT secret (for testing).
func (s *Service) SetSecret(secret string) {
	s.secret = secret
}

// GetSecret returns the JWT secret.
func (s *Service) GetSecret() string {
	return s.secret
}
