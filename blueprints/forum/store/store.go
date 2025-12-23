package store

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/bookmarks"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/notifications"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/feature/votes"
)

// Store provides access to all feature stores.
type Store interface {
	Accounts() accounts.Store
	Boards() boards.Store
	Threads() threads.Store
	Comments() comments.Store
	Votes() votes.Store
	Bookmarks() bookmarks.Store
	Notifications() notifications.Store
	SeedMappings() SeedMappingsStore
	Close() error
}

// SeedMapping represents a mapping from external to local IDs.
type SeedMapping struct {
	Source     string
	EntityType string
	ExternalID string
	LocalID    string
	CreatedAt  time.Time
}

// SeedMappingsStore handles seed mapping persistence for idempotent seeding.
type SeedMappingsStore interface {
	Create(ctx context.Context, mapping *SeedMapping) error
	GetByExternalID(ctx context.Context, source, entityType, externalID string) (*SeedMapping, error)
	GetByLocalID(ctx context.Context, localID string) (*SeedMapping, error)
	Exists(ctx context.Context, source, entityType, externalID string) (bool, error)
	Delete(ctx context.Context, source, entityType, externalID string) error
	DeleteBySource(ctx context.Context, source string) error
	List(ctx context.Context, source, entityType string) ([]*SeedMapping, error)
}
