package live

import "errors"

var (
	// ErrSessionClosed is returned when sending to a closed session.
	ErrSessionClosed = errors.New("live: session closed")

	// ErrQueueFull is returned when the send queue is full.
	// When this happens, the session is closed to protect server health.
	ErrQueueFull = errors.New("live: send queue full")

	// ErrAuthFailed is returned when authentication fails.
	ErrAuthFailed = errors.New("live: authentication failed")

	// ErrUpgradeFailed is returned when WebSocket upgrade fails.
	ErrUpgradeFailed = errors.New("live: websocket upgrade failed")

	// ErrInvalidMessage is returned when message decoding fails.
	ErrInvalidMessage = errors.New("live: invalid message")
)
