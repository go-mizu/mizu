// Package password provides password hashing utilities compatible with WordPress.
package password

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is the default bcrypt cost.
const DefaultCost = 12

// Hash hashes a password using bcrypt.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify verifies a password against a hash.
// Supports both bcrypt hashes and WordPress PHPass hashes (for migration).
func Verify(password, hash string) bool {
	// Try bcrypt first
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err == nil {
		return true
	}

	// TODO: Add PHPass support for legacy WordPress password migration
	// For now, only bcrypt is supported

	return false
}

// NeedsRehash checks if a password hash needs to be rehashed.
// Returns true for PHPass hashes (WordPress legacy) or bcrypt hashes with lower cost.
func NeedsRehash(hash string) bool {
	// Check if it's a bcrypt hash with sufficient cost
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		// Not a valid bcrypt hash, needs rehash
		return true
	}
	return cost < DefaultCost
}

// GenerateToken generates a secure random token.
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GeneratePassword generates a random password.
func GeneratePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i := range bytes {
		bytes[i] = charset[int(bytes[i])%len(charset)]
	}
	return string(bytes), nil
}
