// Package password provides secure password hashing and verification.
package password

import (
	"errors"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the bcrypt cost factor.
	DefaultCost = 12

	// MinLength is the minimum password length.
	MinLength = 8

	// MaxLength is the maximum password length (bcrypt limit is 72 bytes).
	MaxLength = 128
)

// Validation errors.
var (
	ErrTooShort       = errors.New("password must be at least 8 characters")
	ErrTooLong        = errors.New("password must not exceed 128 characters")
	ErrCommonPassword = errors.New("password is too common")
	ErrNoLetter       = errors.New("password must contain at least one letter")
	ErrNoDigit        = errors.New("password must contain at least one number")
)

// Policy defines password requirements.
type Policy struct {
	MinLength      int
	MaxLength      int
	RequireLetter  bool
	RequireDigit   bool
	RequireSpecial bool
	CheckCommon    bool
}

// DefaultPolicy is the default password policy.
var DefaultPolicy = Policy{
	MinLength:      MinLength,
	MaxLength:      MaxLength,
	RequireLetter:  true,
	RequireDigit:   true,
	RequireSpecial: false, // Not required for better UX
	CheckCommon:    true,
}

// Validate checks if a password meets the security requirements.
func Validate(password string) error {
	return ValidateWithPolicy(password, DefaultPolicy)
}

// ValidateWithPolicy checks if a password meets the given policy.
func ValidateWithPolicy(password string, policy Policy) error {
	if len(password) < policy.MinLength {
		return ErrTooShort
	}
	if len(password) > policy.MaxLength {
		return ErrTooLong
	}

	var hasLetter, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if policy.RequireLetter && !hasLetter {
		return ErrNoLetter
	}
	if policy.RequireDigit && !hasDigit {
		return ErrNoDigit
	}
	if policy.RequireSpecial && !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	if policy.CheckCommon && isCommonPassword(password) {
		return ErrCommonPassword
	}

	return nil
}

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

// NeedsRehash checks if a hash needs to be regenerated.
func NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true
	}
	return cost < DefaultCost
}

// isCommonPassword checks if the password is in the list of common passwords.
func isCommonPassword(password string) bool {
	lower := strings.ToLower(password)
	for _, common := range commonPasswords {
		if lower == common {
			return true
		}
	}
	return false
}

// commonPasswords is a list of the most common passwords.
// Based on various security research and data breach analyses.
var commonPasswords = []string{
	"123456",
	"password",
	"12345678",
	"qwerty",
	"123456789",
	"12345",
	"1234",
	"111111",
	"1234567",
	"dragon",
	"123123",
	"baseball",
	"abc123",
	"football",
	"monkey",
	"letmein",
	"shadow",
	"master",
	"666666",
	"qwertyuiop",
	"123321",
	"mustang",
	"1234567890",
	"michael",
	"654321",
	"superman",
	"1qaz2wsx",
	"7777777",
	"121212",
	"000000",
	"qazwsx",
	"password1",
	"password123",
	"passw0rd",
	"admin",
	"admin123",
	"welcome",
	"welcome1",
	"p@ssw0rd",
	"trustno1",
	"iloveyou",
	"princess",
	"sunshine",
	"computer",
	"internet",
	"whatever",
	"cheese",
	"starwars",
	"pokemon",
	"matrix",
}
