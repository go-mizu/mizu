// Package jwt provides JSON Web Token utilities.
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

// Claims represents JWT claims.
type Claims struct {
	Sub string `json:"sub"` // Subject (user ID)
	Exp int64  `json:"exp"` // Expiration time
	Iat int64  `json:"iat"` // Issued at
}

// Header represents JWT header.
type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Sign creates a signed JWT token.
func Sign(userID string, secret []byte, duration time.Duration) (string, error) {
	now := time.Now()

	header := Header{
		Alg: "HS256",
		Typ: "JWT",
	}

	claims := Claims{
		Sub: userID,
		Iat: now.Unix(),
		Exp: now.Add(duration).Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	message := headerB64 + "." + claimsB64
	signature := sign(message, secret)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return message + "." + signatureB64, nil
}

// Verify verifies a JWT token and returns the claims.
func Verify(token string, secret []byte) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	message := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}

	expectedSig := sign(message, secret)
	if !hmac.Equal(signature, expectedSig) {
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

	if time.Now().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

func sign(message string, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	return h.Sum(nil)
}
