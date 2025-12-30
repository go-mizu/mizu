// Package preferences provides user preference management.
package preferences

import (
	"context"
)

// API defines the preferences service interface.
type API interface {
	// Get retrieves a preference value.
	Get(ctx context.Context, userID, key string) (any, error)

	// Set sets a preference value.
	Set(ctx context.Context, userID, key string, value any) error

	// Delete removes a preference.
	Delete(ctx context.Context, userID, key string) error
}
