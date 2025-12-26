package users

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/password"
	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrUserExists       = errors.New("user already exists")
	ErrInvalidEmail     = errors.New("invalid email")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrNotFound         = errors.New("user not found")
	ErrMissingUsername  = errors.New("username is required")
	ErrMissingEmailAddr = errors.New("email is required")
)

// Service implements the users API.
type Service struct {
	store Store
}

// NewService creates a new users service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Register(ctx context.Context, in *RegisterIn) (*User, *Session, error) {
	// Validate required fields
	if in.Username == "" {
		return nil, nil, ErrMissingUsername
	}
	if in.Email == "" {
		return nil, nil, ErrMissingEmailAddr
	}

	// Check if user already exists
	existing, err := s.store.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrUserExists
	}

	existing, err = s.store.GetByUsername(ctx, in.Username)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrUserExists
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, nil, err
	}

	user := &User{
		ID:           ulid.New(),
		Email:        in.Email,
		Username:     in.Username,
		DisplayName:  in.DisplayName,
		PasswordHash: hash,
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Create session
	now := time.Now()
	session := &Session{
		ID:        ulid.New(),
		UserID:    user.ID,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

func (s *Service) Login(ctx context.Context, in *LoginIn) (*User, *Session, error) {
	user, err := s.store.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrInvalidEmail
	}

	valid, err := password.Verify(in.Password, user.PasswordHash)
	if err != nil {
		return nil, nil, err
	}
	if !valid {
		return nil, nil, ErrInvalidPassword
	}

	now := time.Now()
	session := &Session{
		ID:        ulid.New(),
		UserID:    user.ID,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, err := s.store.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

func (s *Service) GetBySession(ctx context.Context, sessionID string) (*User, error) {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.ExpiresAt.Before(time.Now()) {
		return nil, ErrNotFound
	}

	return s.store.GetByID(ctx, session.UserID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*User, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}
