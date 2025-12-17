package sync

import "errors"

var (
	// ErrNotFound is returned when an entity is not found.
	ErrNotFound = errors.New("sync: not found")

	// ErrUnknownMutation is returned when a mutation name has no handler.
	ErrUnknownMutation = errors.New("sync: unknown mutation")

	// ErrInvalidMutation is returned when a mutation is malformed.
	ErrInvalidMutation = errors.New("sync: invalid mutation")

	// ErrInvalidScope is returned when a scope is invalid or missing.
	ErrInvalidScope = errors.New("sync: invalid scope")

	// ErrCursorTooOld is returned when a cursor has been trimmed from the log.
	ErrCursorTooOld = errors.New("sync: cursor too old")

	// ErrConflict is returned when there is a conflict during mutation.
	ErrConflict = errors.New("sync: conflict")
)

// Error codes for Result.Code field.
const (
	CodeOK           = ""
	CodeNotFound     = "not_found"
	CodeUnknown      = "unknown_mutation"
	CodeInvalid      = "invalid_mutation"
	CodeCursorTooOld = "cursor_too_old"
	CodeConflict     = "conflict"
	CodeInternal     = "internal_error"
)
