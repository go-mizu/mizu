package users

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/pkg/password"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("invalid password")
	ErrEmailExists     = errors.New("email already registered")
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// Service implements the users API.
type Service struct {
	store Store
}

// NewService creates a new users service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, in *RegisterIn) (*User, *Session, error) {
	if in.Email == "" {
		return nil, nil, ErrInvalidEmail
	}
	if len(in.Password) < 6 {
		return nil, nil, ErrInvalidPassword
	}

	// Check if email already exists
	existing, _ := s.store.GetByEmail(ctx, in.Email)
	if existing != nil {
		return nil, nil, ErrEmailExists
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	user := &User{
		ID:           ulid.New(),
		Email:        in.Email,
		Name:         in.Name,
		PasswordHash: hash,
		Settings: Settings{
			Theme:         "system",
			Timezone:      "UTC",
			DateFormat:    "YYYY-MM-DD",
			StartOfWeek:   1,
			EmailDigest:   true,
			DesktopNotify: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if user.Name == "" {
		user.Name = in.Email
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Create session
	session := &Session{
		ID:        ulid.New(),
		UserID:    user.ID,
		ExpiresAt: now.Add(30 * 24 * time.Hour), // 30 days
		CreatedAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

// Login authenticates a user and creates a session.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*User, *Session, error) {
	user, err := s.store.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, nil, ErrUserNotFound
	}

	if !password.Verify(in.Password, user.PasswordHash) {
		return nil, nil, ErrInvalidPassword
	}

	now := time.Now()
	session := &Session{
		ID:        ulid.New(),
		UserID:    user.ID,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		CreatedAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

// Logout invalidates a session.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple users by their IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) (map[string]*User, error) {
	if len(ids) == 0 {
		return make(map[string]*User), nil
	}

	users, err := s.store.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*User, len(users))
	for _, u := range users {
		result[u.ID] = u
	}
	return result, nil
}

// GetByEmail retrieves a user by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetByEmail(ctx, email)
}

// GetBySession retrieves a user by session ID.
func (s *Service) GetBySession(ctx context.Context, sessionID string) (*User, error) {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		s.store.DeleteSession(ctx, sessionID)
		return nil, ErrSessionExpired
	}

	return s.store.GetByID(ctx, session.UserID)
}

// Update updates a user's profile.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*User, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// UpdateSettings updates a user's settings.
func (s *Service) UpdateSettings(ctx context.Context, id string, settings Settings) error {
	return s.store.UpdateSettings(ctx, id, settings)
}

// UpdatePassword changes a user's password.
func (s *Service) UpdatePassword(ctx context.Context, id string, oldPass, newPass string) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}

	if !password.Verify(oldPass, user.PasswordHash) {
		return ErrInvalidPassword
	}

	if len(newPass) < 6 {
		return ErrInvalidPassword
	}

	hash, err := password.Hash(newPass)
	if err != nil {
		return err
	}

	return s.store.UpdatePassword(ctx, id, hash)
}
