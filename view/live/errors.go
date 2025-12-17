package live

import "errors"

// Standard errors.
var (
	// ErrSessionNotFound is returned when a session ID doesn't exist.
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired is returned when a session has timed out.
	ErrSessionExpired = errors.New("session expired")

	// ErrInvalidCSRF is returned when the CSRF token is invalid.
	ErrInvalidCSRF = errors.New("invalid CSRF token")

	// ErrNotConnected is returned when trying to send on a closed connection.
	ErrNotConnected = errors.New("not connected")

	// ErrPageNotFound is returned when a page handler doesn't exist for a URL.
	ErrPageNotFound = errors.New("page not found")

	// ErrInvalidMessage is returned when a message cannot be decoded.
	ErrInvalidMessage = errors.New("invalid message")
)

// RecoverableError is an error that can be shown to the user.
type RecoverableError struct {
	Err     error
	Message string // Shown to user
}

func (e *RecoverableError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *RecoverableError) Unwrap() error {
	return e.Err
}

// NewRecoverableError creates a recoverable error with a user-friendly message.
func NewRecoverableError(err error, message string) *RecoverableError {
	return &RecoverableError{
		Err:     err,
		Message: message,
	}
}
