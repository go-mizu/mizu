package storage

import (
	"context"
	"time"
)

// Event describes a change notification.
type Event struct {
	Bucket   string            `json:"bucket,omitempty"`
	Key      string            `json:"key,omitempty"`
	Type     string            `json:"type,omitempty"` // create, update, delete, move
	Time     time.Time         `json:"at,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Watcher streams change notifications.
type Watcher interface {
	Watch(ctx context.Context, bucket string, prefix string, opts Options) (<-chan Event, error)
}
