package accounts

import "errors"

var (
	ErrNotFound          = errors.New("account not found")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrEmailTaken        = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionExpired    = errors.New("session expired")
	ErrSessionNotFound   = errors.New("session not found")
	ErrInvalidPassword   = errors.New("invalid current password")
	ErrWeakPassword      = errors.New("password too weak")
	ErrAccountSuspended  = errors.New("account suspended")
)
