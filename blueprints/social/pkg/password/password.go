// Package password provides password hashing utilities.
package password

import (
	"golang.org/x/crypto/bcrypt"
)

const cost = 12

// Hash hashes a password using bcrypt.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

// Verify checks if a password matches a hash.
func Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
