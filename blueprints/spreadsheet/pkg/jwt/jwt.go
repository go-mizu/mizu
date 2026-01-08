// Package jwt provides JWT token utilities.
package jwt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims represents JWT claims.
type Claims struct {
	UserID     string `json:"user"`
	Collection string `json:"collection,omitempty"`
	ExpiresAt  int64  `json:"exp"`
	NotBefore  int64  `json:"nbf,omitempty"`
	IssuedAt   int64  `json:"iat,omitempty"`
}

// Token generates a JWT token.
func Token(secret string, userID string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		ExpiresAt: now.Add(duration).Unix(),
		NotBefore: now.Unix(),
		IssuedAt:  now.Unix(),
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	message := headerB64 + "." + claimsB64
	signature := sign(message, secret)

	return message + "." + signature, nil
}

// Verify verifies a JWT token and returns the claims.
func Verify(token, secret string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	message := parts[0] + "." + parts[1]
	expectedSig := sign(message, secret)

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, ErrInvalidToken
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	now := time.Now().Unix()
	if now > claims.ExpiresAt {
		return nil, ErrExpiredToken
	}

	// Validate NotBefore claim if present
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}

// UserID extracts the user ID from a verified token.
// SECURITY: This function now requires the secret and verifies the token.
// The previous implementation extracted claims without verification which was unsafe.
func UserID(token, secret string) (string, error) {
	claims, err := Verify(token, secret)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func sign(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// GenerateSecret generates a cryptographically secure random secret for JWT signing.
func GenerateSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("jwt: failed to generate random secret: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
