package sync

import "errors"

var (
	// ErrUnknownMutation is returned when a mutation name has no registered handler.
	ErrUnknownMutation = errors.New("sync: unknown mutation")

	// ErrNotFound is returned when an entity is not found.
	ErrNotFound = errors.New("sync: entity not found")

	// ErrInvalidScope is returned when a scope is invalid or missing.
	ErrInvalidScope = errors.New("sync: invalid scope")

	// ErrInvalidMutation is returned when a mutation is malformed.
	ErrInvalidMutation = errors.New("sync: invalid mutation")

	// ErrConflict is returned when there's a conflict during mutation.
	ErrConflict = errors.New("sync: conflict")
)
