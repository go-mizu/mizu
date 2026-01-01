// Package crypto provides cryptographic utilities.
package crypto

import (
	"crypto/rand"
	"encoding/base64"
)

const (
	// SessionTokenBytes is the size of session tokens (32 bytes = 256 bits).
	SessionTokenBytes = 32
	// ShareTokenBytes is the size of share link tokens (16 bytes = 128 bits).
	ShareTokenBytes = 16
)

// GenerateSessionToken generates a secure session token.
func GenerateSessionToken() (string, error) {
	b := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateShareToken generates a URL-safe share link token.
func GenerateShareToken() (string, error) {
	b := make([]byte, ShareTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateRandomBytes generates n random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
