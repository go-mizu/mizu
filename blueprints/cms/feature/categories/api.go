// Package categories provides hierarchical category management.
package categories

import (
	"context"
	"time"
)

// Category represents a hierarchical category.
type Category struct {
	ID              string    `json:"id"`
	ParentID        string    `json:"parent_id,omitempty"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Description     string    `json:"description,omitempty"`
	FeaturedImageID string    `json:"featured_image_id,omitempty"`
	Meta            string    `json:"meta,omitempty"`
	SortOrder       int       `json:"sort_order"`
	PostCount       int       `json:"post_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a category.
type CreateIn struct {
	ParentID        string `json:"parent_id,omitempty"`
	Name            string `json:"name"`
	Slug            string `json:"slug,omitempty"`
	Description     string `json:"description,omitempty"`
	FeaturedImageID string `json:"featured_image_id,omitempty"`
	Meta            string `json:"meta,omitempty"`
	SortOrder       int    `json:"sort_order,omitempty"`
}

// UpdateIn contains input for updating a category.
type UpdateIn struct {
	ParentID        *string `json:"parent_id,omitempty"`
	Name            *string `json:"name,omitempty"`
	Slug            *string `json:"slug,omitempty"`
	Description     *string `json:"description,omitempty"`
	FeaturedImageID *string `json:"featured_image_id,omitempty"`
	Meta            *string `json:"meta,omitempty"`
	SortOrder       *int    `json:"sort_order,omitempty"`
}

// ListIn contains input for listing categories.
type ListIn struct {
	ParentID string
	Search   string
	Limit    int
	Offset   int
}

// API defines the categories service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetBySlug(ctx context.Context, slug string) (*Category, error)
	List(ctx context.Context, in *ListIn) ([]*Category, int, error)
	GetTree(ctx context.Context) ([]*Category, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Category, error)
	Delete(ctx context.Context, id string) error
	IncrementPostCount(ctx context.Context, id string) error
	DecrementPostCount(ctx context.Context, id string) error
}

// Store defines the data access contract for categories.
type Store interface {
	Create(ctx context.Context, c *Category) error
	GetByID(ctx context.Context, id string) (*Category, error)
	GetBySlug(ctx context.Context, slug string) (*Category, error)
	List(ctx context.Context, in *ListIn) ([]*Category, int, error)
	GetTree(ctx context.Context) ([]*Category, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	IncrementPostCount(ctx context.Context, id string) error
	DecrementPostCount(ctx context.Context, id string) error
}
