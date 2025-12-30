// Package tags provides flat tag management.
package tags

import (
	"context"
	"time"
)

// Tag represents a flat tag.
type Tag struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Description     string    `json:"description,omitempty"`
	FeaturedImageID string    `json:"featured_image_id,omitempty"`
	Meta            string    `json:"meta,omitempty"`
	PostCount       int       `json:"post_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a tag.
type CreateIn struct {
	Name            string `json:"name"`
	Slug            string `json:"slug,omitempty"`
	Description     string `json:"description,omitempty"`
	FeaturedImageID string `json:"featured_image_id,omitempty"`
	Meta            string `json:"meta,omitempty"`
}

// UpdateIn contains input for updating a tag.
type UpdateIn struct {
	Name            *string `json:"name,omitempty"`
	Slug            *string `json:"slug,omitempty"`
	Description     *string `json:"description,omitempty"`
	FeaturedImageID *string `json:"featured_image_id,omitempty"`
	Meta            *string `json:"meta,omitempty"`
}

// ListIn contains input for listing tags.
type ListIn struct {
	Search  string
	Limit   int
	Offset  int
	OrderBy string
	Order   string
}

// API defines the tags service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Tag, error)
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetBySlug(ctx context.Context, slug string) (*Tag, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Tag, error)
	List(ctx context.Context, in *ListIn) ([]*Tag, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Tag, error)
	Delete(ctx context.Context, id string) error
	IncrementPostCount(ctx context.Context, id string) error
	DecrementPostCount(ctx context.Context, id string) error
}

// Store defines the data access contract for tags.
type Store interface {
	Create(ctx context.Context, t *Tag) error
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetBySlug(ctx context.Context, slug string) (*Tag, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Tag, error)
	List(ctx context.Context, in *ListIn) ([]*Tag, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	IncrementPostCount(ctx context.Context, id string) error
	DecrementPostCount(ctx context.Context, id string) error
}
