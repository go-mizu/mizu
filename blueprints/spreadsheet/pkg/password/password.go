// Package password provides password hashing utilities.
package password

import (
	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is the default bcrypt cost.
const DefaultCost = 10

// Hash hashes a password using bcrypt.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	return string(bytes), err
}

// Verify compares a password with its hash.
func Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
