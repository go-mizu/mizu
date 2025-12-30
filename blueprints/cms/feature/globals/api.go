// Package globals provides global document management.
package globals

import (
	"context"
)

// API defines the globals service interface.
type API interface {
	// Get retrieves a global by slug.
	Get(ctx context.Context, slug string) (map[string]any, error)

	// Update updates a global.
	Update(ctx context.Context, slug string, data map[string]any) (map[string]any, error)
}
