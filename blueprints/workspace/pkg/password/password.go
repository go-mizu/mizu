// Package password provides password hashing and verification.
package password

import (
	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is the default bcrypt cost.
const DefaultCost = 10

// Hash hashes a password using bcrypt.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify checks if a password matches a hash.
func Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
