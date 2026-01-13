package api

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Auth handles authentication requests.
type Auth struct {
	store store.UserStore
}

// NewAuth creates a new Auth handler.
func NewAuth(store store.UserStore) *Auth {
	return &Auth{store: store}
}

// LoginInput is the input for login.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates a user.
func (h *Auth) Login(c *mizu.Ctx) error {
	var input LoginInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Email == "" || input.Password == "" {
		return c.JSON(400, map[string]string{"error": "Email and password are required"})
	}

	user, err := h.store.GetByEmail(c.Request().Context(), input.Email)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return c.JSON(401, map[string]string{"error": "Invalid credentials"})
	}

	// Create session
	token, err := generateToken()
	if err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to create session"})
	}

	session := &store.Session{
		ID:        ulid.Make().String(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateSession(c.Request().Context(), session); err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to create session"})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"token":      session.Token,
			"expires_at": session.ExpiresAt,
			"user": map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
				"name":  user.Name,
				"role":  user.Role,
			},
		},
	})
}

// RegisterInput is the input for registration.
type RegisterInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Register creates a new user.
func (h *Auth) Register(c *mizu.Ctx) error {
	var input RegisterInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Email == "" || input.Password == "" {
		return c.JSON(400, map[string]string{"error": "Email and password are required"})
	}

	// Check if user already exists
	_, err := h.store.GetByEmail(c.Request().Context(), input.Email)
	if err == nil {
		return c.JSON(409, map[string]string{"error": "User already exists"})
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to hash password"})
	}

	user := &store.User{
		ID:           ulid.Make().String(),
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: string(hash),
		Role:         "user",
		CreatedAt:    time.Now(),
	}

	if err := h.store.Create(c.Request().Context(), user); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Create session
	token, err := generateToken()
	if err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to create session"})
	}

	session := &store.Session{
		ID:        ulid.Make().String(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateSession(c.Request().Context(), session); err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to create session"})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"token":      session.Token,
			"expires_at": session.ExpiresAt,
			"user": map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
				"name":  user.Name,
				"role":  user.Role,
			},
		},
	})
}

// Logout invalidates a session.
func (h *Auth) Logout(c *mizu.Ctx) error {
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if err := h.store.DeleteSession(c.Request().Context(), token); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  nil,
	})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	session, err := h.store.GetSession(c.Request().Context(), token)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	if session.ExpiresAt.Before(time.Now()) {
		return c.JSON(401, map[string]string{"error": "Session expired"})
	}

	user, err := h.store.GetByID(c.Request().Context(), session.UserID)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "User not found"})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
