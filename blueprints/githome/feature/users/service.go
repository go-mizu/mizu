package users

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/password"
	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the users API
type Service struct {
	store Store
}

// NewService creates a new users service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Register creates a new user account
func (s *Service) Register(ctx context.Context, in *RegisterIn) (*User, *Session, error) {
	// Validate input
	if in.Username == "" {
		return nil, nil, ErrMissingUsername
	}
	if in.Email == "" {
		return nil, nil, ErrMissingEmail
	}
	if in.Password == "" {
		return nil, nil, ErrMissingPassword
	}
	if len(in.Password) < 8 {
		return nil, nil, ErrPasswordTooShort
	}

	// Normalize
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	// Check if username exists
	existing, _ := s.store.GetByUsername(ctx, in.Username)
	if existing != nil {
		return nil, nil, ErrUserExists
	}

	// Check if email exists
	existing, _ = s.store.GetByEmail(ctx, in.Email)
	if existing != nil {
		return nil, nil, ErrUserExists
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	user := &User{
		ID:           ulid.New(),
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: hash,
		FullName:     in.FullName,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Create session
	session := &Session{
		ID:           ulid.New(),
		UserID:       user.ID,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:    now,
		LastActiveAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

// Login authenticates a user
func (s *Service) Login(ctx context.Context, in *LoginIn) (*User, *Session, error) {
	if in.Login == "" || in.Password == "" {
		return nil, nil, ErrInvalidInput
	}

	// Try to find user by username or email
	in.Login = strings.ToLower(strings.TrimSpace(in.Login))
	var user *User
	var err error

	if strings.Contains(in.Login, "@") {
		user, err = s.store.GetByEmail(ctx, in.Login)
	} else {
		user, err = s.store.GetByUsername(ctx, in.Login)
	}

	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrNotFound
	}

	// Verify password
	valid, err := password.Verify(in.Password, user.PasswordHash)
	if err != nil {
		return nil, nil, err
	}
	if !valid {
		return nil, nil, ErrInvalidPassword
	}

	// Create session
	now := time.Now()
	session := &Session{
		ID:           ulid.New(),
		UserID:       user.ID,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:    now,
		LastActiveAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

// Logout invalidates a session
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

// ValidateSession validates a session and returns the user
func (s *Service) ValidateSession(ctx context.Context, sessionID string) (*User, error) {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrNotFound
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		s.store.DeleteSession(ctx, sessionID)
		return nil, ErrSessionExpired
	}

	// Get user
	user, err := s.store.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	// Update session activity
	s.store.UpdateSessionActivity(ctx, sessionID)

	return user, nil
}

// GetByID retrieves a user by ID
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

// GetByUsername retrieves a user by username
func (s *Service) GetByUsername(ctx context.Context, username string) (*User, error) {
	user, err := s.store.GetByUsername(ctx, strings.ToLower(username))
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, err := s.store.GetByEmail(ctx, strings.ToLower(email))
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

// Update updates a user's profile
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	if in.FullName != nil {
		user.FullName = *in.FullName
	}
	if in.Bio != nil {
		user.Bio = *in.Bio
	}
	if in.Location != nil {
		user.Location = *in.Location
	}
	if in.Website != nil {
		user.Website = *in.Website
	}
	if in.Company != nil {
		user.Company = *in.Company
	}
	if in.AvatarURL != nil {
		user.AvatarURL = *in.AvatarURL
	}

	if err := s.store.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Delete deletes a user
func (s *Service) Delete(ctx context.Context, id string) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}

	// Delete all sessions
	if err := s.store.DeleteUserSessions(ctx, id); err != nil {
		return err
	}

	return s.store.Delete(ctx, id)
}

// List lists all users
func (s *Service) List(ctx context.Context, limit, offset int) ([]*User, error) {
	return s.store.List(ctx, limit, offset)
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, id string, in *ChangePasswordIn) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}

	// Verify old password
	valid, err := password.Verify(in.OldPassword, user.PasswordHash)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidPassword
	}

	// Validate new password
	if len(in.NewPassword) < 8 {
		return ErrPasswordTooShort
	}

	// Hash new password
	hash, err := password.Hash(in.NewPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = hash
	if err := s.store.Update(ctx, user); err != nil {
		return err
	}

	// Invalidate all sessions except current (TODO: keep current session)
	return s.store.DeleteUserSessions(ctx, id)
}
