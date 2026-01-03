// Package favorites provides favorite page management.
package favorites

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// Favorite represents a user's favorited page.
type Favorite struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	PageID      string    `json:"page_id"`
	WorkspaceID string    `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`

	// Enriched
	Page *pages.Page `json:"page,omitempty"`
}

// API defines the favorites service contract.
type API interface {
	Add(ctx context.Context, userID, pageID, workspaceID string) (*Favorite, error)
	Remove(ctx context.Context, userID, pageID string) error
	List(ctx context.Context, userID, workspaceID string) ([]*Favorite, error)
	IsFavorite(ctx context.Context, userID, pageID string) (bool, error)
}

// Store defines the data access contract for favorites.
type Store interface {
	Create(ctx context.Context, f *Favorite) error
	Delete(ctx context.Context, userID, pageID string) error
	List(ctx context.Context, userID, workspaceID string) ([]*Favorite, error)
	Exists(ctx context.Context, userID, pageID string) (bool, error)
}
