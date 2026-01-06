// Package jwt provides JWT token utilities.
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

// UserID extracts the user ID from a token without full verification.
func UserID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", ErrInvalidToken
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return "", ErrInvalidToken
	}

	return claims.UserID, nil
}

func sign(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// GenerateSecret generates a random secret for JWT signing.
func GenerateSecret() string {
	return fmt.Sprintf("spreadsheet-secret-%d", time.Now().UnixNano())
}
