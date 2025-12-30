// Package pages provides static page management functionality.
package pages

import (
	"context"
	"time"
)

// Page represents a static page.
type Page struct {
	ID              string    `json:"id"`
	AuthorID        string    `json:"author_id"`
	ParentID        string    `json:"parent_id,omitempty"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	Content         string    `json:"content,omitempty"`
	ContentFormat   string    `json:"content_format"`
	FeaturedImageID string    `json:"featured_image_id,omitempty"`
	Template        string    `json:"template,omitempty"`
	Status          string    `json:"status"`
	Visibility      string    `json:"visibility"`
	Meta            string    `json:"meta,omitempty"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a page.
type CreateIn struct {
	ParentID        string `json:"parent_id,omitempty"`
	Title           string `json:"title"`
	Slug            string `json:"slug,omitempty"`
	Content         string `json:"content,omitempty"`
	ContentFormat   string `json:"content_format,omitempty"`
	FeaturedImageID string `json:"featured_image_id,omitempty"`
	Template        string `json:"template,omitempty"`
	Status          string `json:"status,omitempty"`
	Visibility      string `json:"visibility,omitempty"`
	Meta            string `json:"meta,omitempty"`
	SortOrder       int    `json:"sort_order,omitempty"`
}

// UpdateIn contains input for updating a page.
type UpdateIn struct {
	ParentID        *string `json:"parent_id,omitempty"`
	Title           *string `json:"title,omitempty"`
	Slug            *string `json:"slug,omitempty"`
	Content         *string `json:"content,omitempty"`
	ContentFormat   *string `json:"content_format,omitempty"`
	FeaturedImageID *string `json:"featured_image_id,omitempty"`
	Template        *string `json:"template,omitempty"`
	Status          *string `json:"status,omitempty"`
	Visibility      *string `json:"visibility,omitempty"`
	Meta            *string `json:"meta,omitempty"`
	SortOrder       *int    `json:"sort_order,omitempty"`
}

// ListIn contains input for listing pages.
type ListIn struct {
	ParentID   string
	AuthorID   string
	Status     string
	Visibility string
	Search     string
	Limit      int
	Offset     int
}

// API defines the pages service contract.
type API interface {
	Create(ctx context.Context, authorID string, in *CreateIn) (*Page, error)
	GetByID(ctx context.Context, id string) (*Page, error)
	GetBySlug(ctx context.Context, slug string) (*Page, error)
	GetByPath(ctx context.Context, path string) (*Page, error)
	List(ctx context.Context, in *ListIn) ([]*Page, int, error)
	GetTree(ctx context.Context) ([]*Page, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Page, error)
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for pages.
type Store interface {
	Create(ctx context.Context, p *Page) error
	GetByID(ctx context.Context, id string) (*Page, error)
	GetBySlug(ctx context.Context, slug string) (*Page, error)
	GetByParentAndSlug(ctx context.Context, parentID, slug string) (*Page, error)
	List(ctx context.Context, in *ListIn) ([]*Page, int, error)
	GetTree(ctx context.Context) ([]*Page, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
}
