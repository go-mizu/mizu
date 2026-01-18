package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	store store.Store
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(st store.Store) *AuthHandler {
	return &AuthHandler{store: st}
}

// SignupRequest represents a signup request
type SignupRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User        *store.User `json:"user"`
	AccessToken string      `json:"access_token"`
	ExpiresAt   time.Time   `json:"expires_at"`
}

// Signup handles user registration
func (h *AuthHandler) Signup(c *mizu.Ctx) error {
	var req SignupRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email, username, and password are required"})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
	}

	// Create user
	user := &store.User{
		ID:                uuid.New(),
		Email:             req.Email,
		Username:          req.Username,
		DisplayName:       req.Username,
		EncryptedPassword: string(hashedPassword),
		XPTotal:           0,
		Gems:              500,
		Hearts:            5,
		StreakDays:        0,
		DailyGoalMinutes:  10,
		CreatedAt:         time.Now(),
	}

	if err := h.store.Users().Create(c.Context(), user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email or username already exists"})
	}

	// Generate token (simplified - in production use JWT)
	token := uuid.New().String()

	return c.JSON(http.StatusCreated, AuthResponse{
		User:        user,
		AccessToken: token,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *mizu.Ctx) error {
	var req LoginRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	user, err := h.store.Users().GetByEmail(c.Context(), req.Email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.EncryptedPassword), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}

	// Generate token
	token := uuid.New().String()

	return c.JSON(http.StatusOK, AuthResponse{
		User:        user,
		AccessToken: token,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *mizu.Ctx) error {
	// In a real implementation, invalidate the token
	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c *mizu.Ctx) error {
	// In a real implementation, validate and refresh the token
	token := uuid.New().String()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token": token,
		"expires_at":   time.Now().Add(24 * time.Hour),
	})
}
