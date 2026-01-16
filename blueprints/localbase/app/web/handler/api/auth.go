package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// DefaultJWTSecret is the Supabase local development default JWT secret.
// In production, always use LOCALBASE_JWT_SECRET environment variable.
// WARNING: This is INSECURE and should only be used for local development.
const DefaultJWTSecret = "super-secret-jwt-token-with-at-least-32-characters-long"

// MinPasswordLength is the minimum required password length for security.
const MinPasswordLength = 8

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	store *postgres.Store
	// JWT secret - loaded from LOCALBASE_JWT_SECRET env var or uses default
	jwtSecret []byte
	// Issuer URL for JWT tokens
	issuer string
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(store *postgres.Store) *AuthHandler {
	// Load JWT secret from environment (same as API key middleware)
	secret := os.Getenv("LOCALBASE_JWT_SECRET")
	if secret == "" {
		secret = DefaultJWTSecret
	}

	// Load issuer from environment or use default
	issuer := os.Getenv("LOCALBASE_AUTH_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:54321/auth/v1"
	}

	return &AuthHandler{
		store:     store,
		jwtSecret: []byte(secret),
		issuer:    issuer,
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

// AuthError represents a Supabase-compatible auth error.
type AuthError struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code"`
	Msg       string `json:"msg"`
}

// authError returns a Supabase-compatible error response.
func authError(c *mizu.Ctx, httpCode int, errorCode, msg string) error {
	return c.JSON(httpCode, AuthError{
		Code:      httpCode,
		ErrorCode: errorCode,
		Msg:       msg,
	})
}

// Signup handles user registration.
func (h *AuthHandler) Signup(c *mizu.Ctx) error {
	var req SignupRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return authError(c, 400, "validation_failed", "Unable to validate request body")
	}

	if req.Email == "" && req.Phone == "" {
		return authError(c, 422, "validation_failed", "Email or phone is required")
	}

	if req.Password == "" {
		return authError(c, 422, "validation_failed", "Password is required")
	}

	// Validate password length for security
	if len(req.Password) < MinPasswordLength {
		return authError(c, 422, "weak_password", fmt.Sprintf("Password must be at least %d characters", MinPasswordLength))
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to process signup")
	}

	// Create user
	now := time.Now()
	userID := uuid.New().String()

	// Build user_metadata matching Supabase format
	userMetadata := map[string]any{
		"email":          req.Email,
		"email_verified": true,
		"phone_verified": false,
		"sub":            userID,
	}
	if req.Data != nil {
		for k, v := range req.Data {
			userMetadata[k] = v
		}
	}

	user := &store.User{
		ID:                userID,
		Aud:               "authenticated",
		Role:              "authenticated",
		Email:             req.Email,
		Phone:             req.Phone,
		EncryptedPassword: string(hash),
		EmailConfirmedAt:  &now, // Auto-confirm for development
		LastSignInAt:      &now,
		AppMetadata:       map[string]any{"provider": "email", "providers": []string{"email"}},
		UserMetadata:      userMetadata,
		IsAnonymous:       false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := h.store.Auth().CreateUser(c.Context(), user); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return authError(c, 422, "user_already_exists", "User already registered")
		}
		return authError(c, 500, "unexpected_failure", "Failed to create user")
	}

	// Create email identity for the user
	if req.Email != "" {
		identityID := uuid.New().String()
		identity := &store.Identity{
			IdentityID:   identityID,
			ID:           userID,
			UserID:       userID,
			IdentityData: map[string]any{"email": req.Email, "email_verified": true, "phone_verified": false, "sub": userID},
			Provider:     "email",
			LastSignInAt: &now,
			CreatedAt:    now,
			UpdatedAt:    now,
			Email:        req.Email,
		}
		_ = h.store.Auth().CreateIdentity(c.Context(), identity)
		user.Identities = []*store.Identity{identity}
	}

	// Generate tokens
	resp, err := h.generateAuthResponse(c, user)
	if err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to generate tokens")
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
		return authError(c, 400, "validation_failed", "Unable to validate request body")
	}

	if grantType == "" {
		grantType = req.GrantType
	}

	if grantType == "refresh_token" {
		return h.refreshToken(c, req.RefreshToken)
	}

	// Password grant
	if req.Email == "" && req.Phone == "" {
		return authError(c, 400, "validation_failed", "Email or phone is required")
	}

	var user *store.User
	var err error

	if req.Email != "" {
		user, err = h.store.Auth().GetUserByEmail(c.Context(), req.Email)
	} else {
		user, err = h.store.Auth().GetUserByPhone(c.Context(), req.Phone)
	}

	if err != nil {
		return authError(c, 400, "invalid_credentials", "Invalid login credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.EncryptedPassword), []byte(req.Password)); err != nil {
		return authError(c, 400, "invalid_credentials", "Invalid login credentials")
	}

	// Check if banned
	if user.BannedUntil != nil && user.BannedUntil.After(time.Now()) {
		return authError(c, 403, "user_banned", "User is banned")
	}

	// Update last sign in
	now := time.Now()
	user.LastSignInAt = &now
	user.Aud = "authenticated"
	_ = h.store.Auth().UpdateUser(c.Context(), user)

	// Load user identities
	if identities, err := h.store.Auth().GetUserIdentities(c.Context(), user.ID); err == nil {
		user.Identities = identities
	}

	// Generate tokens
	resp, err := h.generateAuthResponse(c, user)
	if err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to generate tokens")
	}

	return c.JSON(200, resp)
}

func (h *AuthHandler) refreshToken(c *mizu.Ctx, token string) error {
	if token == "" {
		return authError(c, 400, "validation_failed", "Refresh token is required")
	}

	rt, err := h.store.Auth().GetRefreshToken(c.Context(), token)
	if err != nil {
		return authError(c, 400, "invalid_grant", "Invalid refresh token")
	}

	if rt.Revoked {
		return authError(c, 400, "invalid_grant", "Refresh token has been revoked")
	}

	user, err := h.store.Auth().GetUserByID(c.Context(), rt.UserID)
	if err != nil {
		return authError(c, 400, "invalid_grant", "User not found")
	}

	// Set aud and load identities
	user.Aud = "authenticated"
	if identities, err := h.store.Auth().GetUserIdentities(c.Context(), user.ID); err == nil {
		user.Identities = identities
	}

	// Rotate refresh token
	newToken := generateToken(32)
	if err := h.store.Auth().RotateRefreshToken(c.Context(), token, newToken); err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to rotate token")
	}

	// Generate new tokens
	resp, err := h.generateAuthResponseWithRefresh(c, user, newToken)
	if err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to generate tokens")
	}

	return c.JSON(200, resp)
}

// Logout handles user logout.
func (h *AuthHandler) Logout(c *mizu.Ctx) error {
	// Get user from token
	user, err := h.getUserFromToken(c)
	if err != nil {
		return authError(c, 401, "not_authenticated", "Not authenticated")
	}

	// Delete all sessions
	if err := h.store.Auth().DeleteUserSessions(c.Context(), user.ID); err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to logout")
	}

	return c.NoContent()
}

// Recover initiates password recovery.
func (h *AuthHandler) Recover(c *mizu.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return authError(c, 400, "validation_failed", "Unable to validate request body")
	}

	if req.Email == "" {
		return authError(c, 400, "validation_failed", "Email is required")
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
		return authError(c, 401, "not_authenticated", "Not authenticated")
	}

	// Set aud and load identities
	user.Aud = "authenticated"
	if identities, err := h.store.Auth().GetUserIdentities(c.Context(), user.ID); err == nil {
		user.Identities = identities
	}

	return c.JSON(200, user)
}

// UpdateUser updates the current user.
func (h *AuthHandler) UpdateUser(c *mizu.Ctx) error {
	user, err := h.getUserFromToken(c)
	if err != nil {
		return authError(c, 401, "not_authenticated", "Not authenticated")
	}

	var req struct {
		Email    string         `json:"email"`
		Phone    string         `json:"phone"`
		Password string         `json:"password"`
		Data     map[string]any `json:"data"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return authError(c, 400, "validation_failed", "Unable to validate request body")
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
			return authError(c, 500, "unexpected_failure", "Failed to process password")
		}
		user.EncryptedPassword = string(hash)
	}
	if req.Data != nil {
		user.UserMetadata = req.Data
	}

	user.UpdatedAt = time.Now()

	if err := h.store.Auth().UpdateUser(c.Context(), user); err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to update user")
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
		return authError(c, 400, "validation_failed", "Unable to validate request body")
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
		return authError(c, 400, "validation_failed", "Unable to validate request body")
	}

	// In production, verify the token
	return c.JSON(200, map[string]string{"message": "verified"})
}

// EnrollMFA enrolls a new MFA factor.
func (h *AuthHandler) EnrollMFA(c *mizu.Ctx) error {
	user, err := h.getUserFromToken(c)
	if err != nil {
		return authError(c, 401, "not_authenticated", "Not authenticated")
	}

	var req struct {
		FactorType   string `json:"factor_type"`
		FriendlyName string `json:"friendly_name"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return authError(c, 400, "validation_failed", "Unable to validate request body")
	}

	if req.FactorType != "totp" {
		return authError(c, 400, "invalid_factor_type", "Only TOTP is supported")
	}

	// Generate TOTP secret
	secret := generateToken(20)

	factor := &store.MFAFactor{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		FriendlyName: req.FriendlyName,
		FactorType:   req.FactorType,
		Status:       "unverified",
		Secret:       secret,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.store.Auth().CreateMFAFactor(c.Context(), factor); err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to create MFA factor")
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
		return authError(c, 401, "not_authenticated", "Not authenticated")
	}

	factorID := c.Param("id")

	factor, err := h.store.Auth().GetMFAFactor(c.Context(), factorID)
	if err != nil {
		return authError(c, 404, "mfa_factor_not_found", "MFA factor not found")
	}

	if factor.UserID != user.ID {
		return authError(c, 403, "forbidden", "Access denied")
	}

	if err := h.store.Auth().DeleteMFAFactor(c.Context(), factorID); err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to delete MFA factor")
	}

	return c.NoContent()
}

// ChallengeMFA creates an MFA challenge.
func (h *AuthHandler) ChallengeMFA(c *mizu.Ctx) error {
	factorID := c.Param("id")

	_, err := h.store.Auth().GetMFAFactor(c.Context(), factorID)
	if err != nil {
		return authError(c, 404, "mfa_factor_not_found", "MFA factor not found")
	}

	// Create challenge
	challengeID := uuid.New().String()

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
		return authError(c, 400, "validation_failed", "Unable to validate request body")
	}

	factor, err := h.store.Auth().GetMFAFactor(c.Context(), factorID)
	if err != nil {
		return authError(c, 404, "mfa_factor_not_found", "MFA factor not found")
	}

	// Validate code format
	if len(req.Code) != 6 {
		return authError(c, 400, "mfa_verification_failed", "Invalid verification code format")
	}

	// Verify TOTP code using the factor's secret
	if !verifyTOTP(factor.Secret, req.Code) {
		return authError(c, 400, "mfa_verification_failed", "Invalid verification code")
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
		return authError(c, 500, "unexpected_failure", "Failed to list users")
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
		return authError(c, 400, "validation_failed", "Unable to validate request body")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to process password")
	}

	role := req.Role
	if role == "" {
		role = "authenticated"
	}

	now := time.Now()
	userID := uuid.New().String()

	// Build user_metadata matching Supabase format
	userMetadata := map[string]any{
		"email":          req.Email,
		"email_verified": true,
		"phone_verified": false,
		"sub":            userID,
	}
	if req.Data != nil {
		for k, v := range req.Data {
			userMetadata[k] = v
		}
	}

	user := &store.User{
		ID:                userID,
		Aud:               "authenticated",
		Role:              role,
		Email:             req.Email,
		Phone:             req.Phone,
		EncryptedPassword: string(hash),
		EmailConfirmedAt:  &now,
		LastSignInAt:      &now,
		AppMetadata:       map[string]any{"provider": "email", "providers": []string{"email"}},
		UserMetadata:      userMetadata,
		IsAnonymous:       false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := h.store.Auth().CreateUser(c.Context(), user); err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to create user")
	}

	// Create email identity for the user
	if req.Email != "" {
		identityID := uuid.New().String()
		identity := &store.Identity{
			IdentityID:   identityID,
			ID:           userID,
			UserID:       userID,
			IdentityData: map[string]any{"email": req.Email, "email_verified": true, "phone_verified": false, "sub": userID},
			Provider:     "email",
			LastSignInAt: &now,
			CreatedAt:    now,
			UpdatedAt:    now,
			Email:        req.Email,
		}
		_ = h.store.Auth().CreateIdentity(c.Context(), identity)
		user.Identities = []*store.Identity{identity}
	}

	return c.JSON(201, user)
}

// GetUserByID gets a user by ID (admin).
func (h *AuthHandler) GetUserByID(c *mizu.Ctx) error {
	id := c.Param("id")

	user, err := h.store.Auth().GetUserByID(c.Context(), id)
	if err != nil {
		return authError(c, 404, "user_not_found", "User not found")
	}

	// Set aud and load identities
	user.Aud = "authenticated"
	if identities, err := h.store.Auth().GetUserIdentities(c.Context(), user.ID); err == nil {
		user.Identities = identities
	}

	return c.JSON(200, user)
}

// UpdateUserByID updates a user by ID (admin).
func (h *AuthHandler) UpdateUserByID(c *mizu.Ctx) error {
	id := c.Param("id")

	user, err := h.store.Auth().GetUserByID(c.Context(), id)
	if err != nil {
		return authError(c, 404, "user_not_found", "User not found")
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
		return authError(c, 400, "validation_failed", "Unable to validate request body")
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
			return authError(c, 500, "unexpected_failure", "Failed to process password")
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
		return authError(c, 500, "unexpected_failure", "Failed to update user")
	}

	return c.JSON(200, user)
}

// DeleteUser deletes a user (admin).
func (h *AuthHandler) DeleteUser(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.Auth().DeleteUser(c.Context(), id); err != nil {
		return authError(c, 500, "unexpected_failure", "Failed to delete user")
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

func (h *AuthHandler) generateAuthResponseWithRefresh(c *mizu.Ctx, user *store.User, refreshToken string) (*AuthResponse, error) {
	expiresIn := 3600 // 1 hour
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresIn) * time.Second)

	// Generate session ID
	sessionID := uuid.New().String()

	// Create session in database
	session := &store.Session{
		ID:        sessionID,
		UserID:    user.ID,
		CreatedAt: now,
		UpdatedAt: now,
		AAL:       "aal1",
		NotAfter:  expiresAt,
	}
	_ = h.store.Auth().CreateSession(c.Context(), session)

	// Determine AMR (authentication methods reference)
	amr := []map[string]any{
		{"method": "password", "timestamp": now.Unix()},
	}

	// Create JWT with Supabase-compatible claims
	claims := jwt.MapClaims{
		"aud":           "authenticated",
		"exp":           expiresAt.Unix(),
		"iat":           now.Unix(),
		"iss":           h.issuer,
		"sub":           user.ID,
		"email":         user.Email,
		"phone":         user.Phone,
		"app_metadata":  user.AppMetadata,
		"user_metadata": user.UserMetadata,
		"role":          user.Role,
		"aal":           "aal1",
		"amr":           amr,
		"session_id":    sessionID,
		"is_anonymous":  user.IsAnonymous,
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

// verifyTOTP verifies a TOTP code against a secret.
// It checks the current time period and allows for one period of clock drift.
func verifyTOTP(secret, code string) bool {
	// Decode the secret (hex-encoded in our implementation)
	secretBytes, err := hex.DecodeString(secret)
	if err != nil {
		// Try base32 decoding as fallback (standard TOTP format)
		secretBytes, err = base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
		if err != nil {
			return false
		}
	}

	// Get current time period (30-second intervals)
	now := time.Now().Unix()
	period := int64(30)

	// Check current period and allow for clock drift (±1 period)
	for _, offset := range []int64{0, -1, 1} {
		counter := (now / period) + offset
		expectedCode := generateTOTPCode(secretBytes, counter)
		if expectedCode == code {
			return true
		}
	}

	return false
}

// generateTOTPCode generates a 6-digit TOTP code for the given counter.
func generateTOTPCode(secret []byte, counter int64) string {
	// Convert counter to big-endian bytes
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, uint64(counter))

	// Calculate HMAC-SHA1
	h := hmac.New(sha1.New, secret)
	h.Write(counterBytes)
	hash := h.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	truncatedHash := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Generate 6-digit code
	code := truncatedHash % uint32(math.Pow10(6))
	return fmt.Sprintf("%06d", code)
}
