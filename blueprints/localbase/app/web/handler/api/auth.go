package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	store *postgres.Store
	// JWT secret - in production, this should come from config
	jwtSecret []byte
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(store *postgres.Store) *AuthHandler {
	return &AuthHandler{
		store:     store,
		jwtSecret: []byte("your-super-secret-jwt-key-min-32-chars"),
	}
}

// SignupRequest represents a signup request.
type SignupRequest struct {
	Email    string         `json:"email"`
	Phone    string         `json:"phone"`
	Password string         `json:"password"`
	Data     map[string]any `json:"data,omitempty"`
}

// TokenRequest represents a token request.
type TokenRequest struct {
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Password     string `json:"password"`
	RefreshToken string `json:"refresh_token"`
	GrantType    string `json:"grant_type"` // password, refresh_token
}

// AuthResponse represents an auth response.
type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"`
	ExpiresAt    int64       `json:"expires_at"`
	RefreshToken string      `json:"refresh_token"`
	User         *store.User `json:"user"`
}

// Signup handles user registration.
func (h *AuthHandler) Signup(c *mizu.Ctx) error {
	var req SignupRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" && req.Phone == "" {
		return c.JSON(400, map[string]string{"error": "email or phone required"})
	}

	if req.Password == "" {
		return c.JSON(400, map[string]string{"error": "password required"})
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to hash password"})
	}

	// Create user
	now := time.Now()
	user := &store.User{
		ID:                ulid.Make().String(),
		Email:             req.Email,
		Phone:             req.Phone,
		EncryptedPassword: string(hash),
		EmailConfirmedAt:  &now, // Auto-confirm for development
		AppMetadata:       map[string]any{"provider": "email"},
		UserMetadata:      req.Data,
		Role:              "authenticated",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := h.store.Auth().CreateUser(c.Context(), user); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return c.JSON(400, map[string]string{"error": "user already exists"})
		}
		return c.JSON(500, map[string]string{"error": "failed to create user"})
	}

	// Generate tokens
	resp, err := h.generateAuthResponse(c, user)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate tokens"})
	}

	return c.JSON(201, resp)
}

// Token handles token generation (login).
func (h *AuthHandler) Token(c *mizu.Ctx) error {
	grantType := c.Query("grant_type")
	if grantType == "" {
		grantType = "password"
	}

	var req TokenRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if grantType == "" {
		grantType = req.GrantType
	}

	if grantType == "refresh_token" {
		return h.refreshToken(c, req.RefreshToken)
	}

	// Password grant
	if req.Email == "" && req.Phone == "" {
		return c.JSON(400, map[string]string{"error": "email or phone required"})
	}

	var user *store.User
	var err error

	if req.Email != "" {
		user, err = h.store.Auth().GetUserByEmail(c.Context(), req.Email)
	} else {
		user, err = h.store.Auth().GetUserByPhone(c.Context(), req.Phone)
	}

	if err != nil {
		return c.JSON(401, map[string]string{"error": "invalid credentials"})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.EncryptedPassword), []byte(req.Password)); err != nil {
		return c.JSON(401, map[string]string{"error": "invalid credentials"})
	}

	// Check if banned
	if user.BannedUntil != nil && user.BannedUntil.After(time.Now()) {
		return c.JSON(403, map[string]string{"error": "user is banned"})
	}

	// Update last sign in
	now := time.Now()
	user.LastSignInAt = &now
	_ = h.store.Auth().UpdateUser(c.Context(), user)

	// Generate tokens
	resp, err := h.generateAuthResponse(c, user)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate tokens"})
	}

	return c.JSON(200, resp)
}

func (h *AuthHandler) refreshToken(c *mizu.Ctx, token string) error {
	if token == "" {
		return c.JSON(400, map[string]string{"error": "refresh_token required"})
	}

	rt, err := h.store.Auth().GetRefreshToken(c.Context(), token)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "invalid refresh token"})
	}

	if rt.Revoked {
		return c.JSON(401, map[string]string{"error": "refresh token revoked"})
	}

	user, err := h.store.Auth().GetUserByID(c.Context(), rt.UserID)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "user not found"})
	}

	// Rotate refresh token
	newToken := generateToken(32)
	if err := h.store.Auth().RotateRefreshToken(c.Context(), token, newToken); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to rotate token"})
	}

	// Generate new tokens
	resp, err := h.generateAuthResponseWithRefresh(c, user, newToken)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate tokens"})
	}

	return c.JSON(200, resp)
}

// Logout handles user logout.
func (h *AuthHandler) Logout(c *mizu.Ctx) error {
	// Get user from token
	user, err := h.getUserFromToken(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	// Delete all sessions
	if err := h.store.Auth().DeleteUserSessions(c.Context(), user.ID); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to logout"})
	}

	return c.NoContent()
}

// Recover initiates password recovery.
func (h *AuthHandler) Recover(c *mizu.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" {
		return c.JSON(400, map[string]string{"error": "email required"})
	}

	user, err := h.store.Auth().GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		// Don't reveal if user exists
		return c.JSON(200, map[string]string{"message": "recovery email sent"})
	}

	// Generate recovery token
	token := generateToken(32)
	user.RecoveryToken = token
	_ = h.store.Auth().UpdateUser(c.Context(), user)

	// In production, send email here
	return c.JSON(200, map[string]string{"message": "recovery email sent"})
}

// GetUser returns the current user.
func (h *AuthHandler) GetUser(c *mizu.Ctx) error {
	user, err := h.getUserFromToken(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	return c.JSON(200, user)
}

// UpdateUser updates the current user.
func (h *AuthHandler) UpdateUser(c *mizu.Ctx) error {
	user, err := h.getUserFromToken(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		Email    string         `json:"email"`
		Phone    string         `json:"phone"`
		Password string         `json:"password"`
		Data     map[string]any `json:"data"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(500, map[string]string{"error": "failed to hash password"})
		}
		user.EncryptedPassword = string(hash)
	}
	if req.Data != nil {
		user.UserMetadata = req.Data
	}

	user.UpdatedAt = time.Now()

	if err := h.store.Auth().UpdateUser(c.Context(), user); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update user"})
	}

	return c.JSON(200, user)
}

// SendOTP sends a one-time password.
func (h *AuthHandler) SendOTP(c *mizu.Ctx) error {
	var req struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	// In production, generate and send OTP
	return c.JSON(200, map[string]string{"message": "OTP sent"})
}

// Verify verifies OTP or magic link token.
func (h *AuthHandler) Verify(c *mizu.Ctx) error {
	var req struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
		Token string `json:"token"`
		Type  string `json:"type"` // signup, recovery, magiclink
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	// In production, verify the token
	return c.JSON(200, map[string]string{"message": "verified"})
}

// EnrollMFA enrolls a new MFA factor.
func (h *AuthHandler) EnrollMFA(c *mizu.Ctx) error {
	user, err := h.getUserFromToken(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		FactorType   string `json:"factor_type"`
		FriendlyName string `json:"friendly_name"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.FactorType != "totp" {
		return c.JSON(400, map[string]string{"error": "only totp is supported"})
	}

	// Generate TOTP secret
	secret := generateToken(20)

	factor := &store.MFAFactor{
		ID:           ulid.Make().String(),
		UserID:       user.ID,
		FriendlyName: req.FriendlyName,
		FactorType:   req.FactorType,
		Status:       "unverified",
		Secret:       secret,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.store.Auth().CreateMFAFactor(c.Context(), factor); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create factor"})
	}

	return c.JSON(201, map[string]any{
		"id":     factor.ID,
		"type":   factor.FactorType,
		"totp":   map[string]string{"secret": secret},
		"status": factor.Status,
	})
}

// UnenrollMFA removes an MFA factor.
func (h *AuthHandler) UnenrollMFA(c *mizu.Ctx) error {
	user, err := h.getUserFromToken(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	factorID := c.Param("id")

	factor, err := h.store.Auth().GetMFAFactor(c.Context(), factorID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "factor not found"})
	}

	if factor.UserID != user.ID {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}

	if err := h.store.Auth().DeleteMFAFactor(c.Context(), factorID); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete factor"})
	}

	return c.NoContent()
}

// ChallengeMFA creates an MFA challenge.
func (h *AuthHandler) ChallengeMFA(c *mizu.Ctx) error {
	factorID := c.Param("id")

	_, err := h.store.Auth().GetMFAFactor(c.Context(), factorID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "factor not found"})
	}

	// Create challenge
	challengeID := ulid.Make().String()

	return c.JSON(200, map[string]string{
		"id":         challengeID,
		"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	})
}

// VerifyMFA verifies an MFA challenge.
func (h *AuthHandler) VerifyMFA(c *mizu.Ctx) error {
	factorID := c.Param("id")

	var req struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"code"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	factor, err := h.store.Auth().GetMFAFactor(c.Context(), factorID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "factor not found"})
	}

	// In production, verify TOTP code
	// For now, accept any 6-digit code
	if len(req.Code) != 6 {
		return c.JSON(400, map[string]string{"error": "invalid code"})
	}

	// Mark as verified if unverified
	if factor.Status == "unverified" {
		factor.Status = "verified"
		factor.UpdatedAt = time.Now()
		_ = h.store.Auth().UpdateMFAFactor(c.Context(), factor)
	}

	return c.JSON(200, map[string]string{"message": "verified"})
}

// ListUsers lists all users (admin).
func (h *AuthHandler) ListUsers(c *mizu.Ctx) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 50)

	users, total, err := h.store.Auth().ListUsers(c.Context(), page, perPage)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list users"})
	}

	return c.JSON(200, map[string]any{
		"users": users,
		"total": total,
		"page":  page,
	})
}

// CreateUser creates a user (admin).
func (h *AuthHandler) CreateUser(c *mizu.Ctx) error {
	var req struct {
		Email    string         `json:"email"`
		Phone    string         `json:"phone"`
		Password string         `json:"password"`
		Data     map[string]any `json:"data"`
		Role     string         `json:"role"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to hash password"})
	}

	role := req.Role
	if role == "" {
		role = "authenticated"
	}

	now := time.Now()
	user := &store.User{
		ID:                ulid.Make().String(),
		Email:             req.Email,
		Phone:             req.Phone,
		EncryptedPassword: string(hash),
		EmailConfirmedAt:  &now,
		AppMetadata:       map[string]any{"provider": "email"},
		UserMetadata:      req.Data,
		Role:              role,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := h.store.Auth().CreateUser(c.Context(), user); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create user"})
	}

	return c.JSON(201, user)
}

// GetUserByID gets a user by ID (admin).
func (h *AuthHandler) GetUserByID(c *mizu.Ctx) error {
	id := c.Param("id")

	user, err := h.store.Auth().GetUserByID(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "user not found"})
	}

	return c.JSON(200, user)
}

// UpdateUserByID updates a user by ID (admin).
func (h *AuthHandler) UpdateUserByID(c *mizu.Ctx) error {
	id := c.Param("id")

	user, err := h.store.Auth().GetUserByID(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "user not found"})
	}

	var req struct {
		Email    string         `json:"email"`
		Phone    string         `json:"phone"`
		Password string         `json:"password"`
		Data     map[string]any `json:"data"`
		Role     string         `json:"role"`
		Banned   bool           `json:"banned"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(500, map[string]string{"error": "failed to hash password"})
		}
		user.EncryptedPassword = string(hash)
	}
	if req.Data != nil {
		user.UserMetadata = req.Data
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Banned {
		bannedUntil := time.Now().AddDate(100, 0, 0)
		user.BannedUntil = &bannedUntil
	} else {
		user.BannedUntil = nil
	}

	user.UpdatedAt = time.Now()

	if err := h.store.Auth().UpdateUser(c.Context(), user); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update user"})
	}

	return c.JSON(200, user)
}

// DeleteUser deletes a user (admin).
func (h *AuthHandler) DeleteUser(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.Auth().DeleteUser(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete user"})
	}

	return c.NoContent()
}

// Helper functions

func (h *AuthHandler) generateAuthResponse(c *mizu.Ctx, user *store.User) (*AuthResponse, error) {
	refreshToken := generateToken(32)

	// Create refresh token in database
	rt := &store.RefreshToken{
		Token:  refreshToken,
		UserID: user.ID,
	}
	if err := h.store.Auth().CreateRefreshToken(c.Context(), rt); err != nil {
		return nil, err
	}

	return h.generateAuthResponseWithRefresh(c, user, refreshToken)
}

func (h *AuthHandler) generateAuthResponseWithRefresh(_ *mizu.Ctx, user *store.User, refreshToken string) (*AuthResponse, error) {
	expiresIn := 3600 // 1 hour
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	// Create JWT
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"phone": user.Phone,
		"role":  user.Role,
		"iat":   time.Now().Unix(),
		"exp":   expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(h.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		TokenType:    "bearer",
		ExpiresIn:    expiresIn,
		ExpiresAt:    expiresAt.Unix(),
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (h *AuthHandler) getUserFromToken(c *mizu.Ctx) (*store.User, error) {
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("no authorization header")
	}

	parts := strings.Split(auth, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, fmt.Errorf("invalid authorization header")
	}

	tokenString := parts[1]

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return h.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return h.store.Auth().GetUserByID(c.Context(), userID)
}

func generateToken(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func queryInt(c *mizu.Ctx, key string, defaultVal int) int {
	v := c.Query(key)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return i
}
