package accounts

import "errors"

var (
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("invalid password")
	ErrNotFound        = errors.New("user not found")
	ErrMissingEmail    = errors.New("email is required")
	ErrMissingName     = errors.New("name is required")
	ErrMissingPassword = errors.New("password is required")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrUnauthorized    = errors.New("unauthorized")
)
