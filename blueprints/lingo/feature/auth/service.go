package auth

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailExists        = errors.New("email already exists")
	ErrUsernameExists     = errors.New("username already exists")
	ErrWeakPassword       = errors.New("password must be at least 6 characters")
	ErrInvalidEmail       = errors.New("invalid email format")
)

// Service handles authentication business logic
type Service struct {
	store store.Store
	users store.UserStore
}

// NewService creates a new auth service
func NewService(st store.Store) *Service {
	return &Service{
		store: st,
		users: st.Users(),
	}
}

// SignupInput represents signup request data
type SignupInput struct {
	Email       string
	Username    string
	Password    string
	DisplayName string
}

// LoginInput represents login request data
type LoginInput struct {
	Email    string
	Password string
}

// AuthResult represents authentication result
type AuthResult struct {
	User        *store.User
	AccessToken string
	ExpiresAt   time.Time
}

// Signup creates a new user account
func (s *Service) Signup(ctx context.Context, input SignupInput) (*AuthResult, error) {
	// Validate input
	if len(input.Password) < 6 {
		return nil, ErrWeakPassword
	}

	// Check if email exists
	if existing, _ := s.users.GetByEmail(ctx, input.Email); existing != nil {
		return nil, ErrEmailExists
	}

	// Check if username exists
	if existing, _ := s.users.GetByUsername(ctx, input.Username); existing != nil {
		return nil, ErrUsernameExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	displayName := input.DisplayName
	if displayName == "" {
		displayName = input.Username
	}

	// Create user with default values
	user := &store.User{
		ID:                uuid.New(),
		Email:             input.Email,
		Username:          input.Username,
		DisplayName:       displayName,
		EncryptedPassword: string(hashedPassword),
		XPTotal:           0,
		Gems:              500, // Starting gems
		Hearts:            5,   // Full hearts
		StreakDays:        0,
		StreakFreezeCount: 0,
		IsPremium:         false,
		DailyGoalMinutes:  10, // Default goal
		CreatedAt:         time.Now(),
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate access token (simplified - use JWT in production)
	token := s.generateToken()

	return &AuthResult{
		User:        user,
		AccessToken: token,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}, nil
}

// Login authenticates a user
func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	user, err := s.users.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.EncryptedPassword), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last active time
	now := time.Now()
	user.LastActiveAt = &now
	_ = s.users.Update(ctx, user)

	// Generate access token
	token := s.generateToken()

	return &AuthResult{
		User:        user,
		AccessToken: token,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}, nil
}

// RefreshToken generates a new access token
func (s *Service) RefreshToken(ctx context.Context, userID uuid.UUID) (*AuthResult, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	token := s.generateToken()

	return &AuthResult{
		User:        user,
		AccessToken: token,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}, nil
}

// ValidateToken validates an access token and returns the user
// In production, this would validate JWT and check expiration
func (s *Service) ValidateToken(ctx context.Context, token string) (*store.User, error) {
	// Simplified implementation - in production use JWT validation
	// For now, we'll use the user ID from request headers
	return nil, nil
}

func (s *Service) generateToken() string {
	// Simplified token generation - use JWT in production
	return uuid.New().String()
}
