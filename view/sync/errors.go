package sync

import "errors"

// Error values returned by the sync client.
var (
	// ErrOffline is returned when an operation requires network but client is offline.
	ErrOffline = errors.New("viewsync: offline")

	// ErrConflict is returned when a mutation conflicts with server state.
	ErrConflict = errors.New("viewsync: conflict")

	// ErrNotFound is returned when an entity does not exist.
	ErrNotFound = errors.New("viewsync: not found")

	// ErrInvalidState is returned when client state is invalid.
	ErrInvalidState = errors.New("viewsync: invalid state")

	// ErrNotStarted is returned when client operations are called before Start.
	ErrNotStarted = errors.New("viewsync: client not started")

	// ErrAlreadyStarted is returned when Start is called on a running client.
	ErrAlreadyStarted = errors.New("viewsync: client already started")

	// ErrCursorTooOld is returned when the cursor has fallen behind the log.
	ErrCursorTooOld = errors.New("viewsync: cursor too old")
)
