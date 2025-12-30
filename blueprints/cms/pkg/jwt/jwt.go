// Package jwt provides JWT token generation and validation.
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims represents the JWT claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID     string `json:"user"`
	Collection string `json:"collection"`
}

// Config holds JWT configuration.
type Config struct {
	Secret          string
	TokenExpiration time.Duration
}

// Manager handles JWT operations.
type Manager struct {
	config Config
}

// NewManager creates a new JWT manager.
func NewManager(config Config) *Manager {
	return &Manager{config: config}
}

// Generate creates a new JWT token for a user.
func (m *Manager) Generate(userID, collection string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.config.TokenExpiration)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID:     userID,
		Collection: collection,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.config.Secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// Validate validates a JWT token and returns the claims.
func (m *Manager) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Refresh creates a new token with extended expiration.
func (m *Manager) Refresh(tokenString string) (string, time.Time, error) {
	claims, err := m.Validate(tokenString)
	if err != nil && !errors.Is(err, ErrExpiredToken) {
		return "", time.Time{}, err
	}

	// Allow refresh of expired tokens within a grace period (e.g., 24 hours)
	if claims == nil {
		return "", time.Time{}, ErrInvalidToken
	}

	return m.Generate(claims.UserID, claims.Collection)
}
