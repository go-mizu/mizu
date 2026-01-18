package auth

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for authentication
type Handler struct {
	svc *Service
}

// NewHandler creates a new auth handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers auth routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Post("/auth/signup", h.Signup)
	r.Post("/auth/login", h.Login)
	r.Post("/auth/logout", h.Logout)
	r.Post("/auth/refresh", h.Refresh)
}

// SignupRequest represents a signup request body
type SignupRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
}

// LoginRequest represents a login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User        interface{} `json:"user"`
	AccessToken string      `json:"access_token"`
	ExpiresAt   string      `json:"expires_at"`
}

// Signup handles POST /auth/signup
func (h *Handler) Signup(c *mizu.Ctx) error {
	var req SignupRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "email, username, and password are required",
		})
	}

	result, err := h.svc.Signup(c.Context(), SignupInput{
		Email:       req.Email,
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		switch err {
		case ErrEmailExists:
			return c.JSON(http.StatusConflict, map[string]string{"error": "email already exists"})
		case ErrUsernameExists:
			return c.JSON(http.StatusConflict, map[string]string{"error": "username already exists"})
		case ErrWeakPassword:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		}
	}

	return c.JSON(http.StatusCreated, AuthResponse{
		User:        result.User,
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Login handles POST /auth/login
func (h *Handler) Login(c *mizu.Ctx) error {
	var req LoginRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "email and password are required",
		})
	}

	result, err := h.svc.Login(c.Context(), LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}

	return c.JSON(http.StatusOK, AuthResponse{
		User:        result.User,
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Logout handles POST /auth/logout
func (h *Handler) Logout(c *mizu.Ctx) error {
	// In a full implementation, invalidate the token
	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

// Refresh handles POST /auth/refresh
func (h *Handler) Refresh(c *mizu.Ctx) error {
	// Get user ID from context (would be set by auth middleware)
	// For now, generate a new token
	result, err := h.svc.RefreshToken(c.Context(), getUserID(c))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	return c.JSON(http.StatusOK, AuthResponse{
		User:        result.User,
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// getUserID extracts the user ID from the request context
func getUserID(c *mizu.Ctx) uuid.UUID {
	if userIDStr := c.Header().Get("X-User-ID"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}
