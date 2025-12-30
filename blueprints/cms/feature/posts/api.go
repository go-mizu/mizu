// Package posts provides blog post management functionality.
package posts

import (
	"context"
	"time"
)

// Post represents a blog post.
type Post struct {
	ID              string     `json:"id"`
	AuthorID        string     `json:"author_id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Excerpt         string     `json:"excerpt,omitempty"`
	Content         string     `json:"content,omitempty"`
	ContentFormat   string     `json:"content_format"`
	FeaturedImageID string     `json:"featured_image_id,omitempty"`
	Status          string     `json:"status"`
	Visibility      string     `json:"visibility"`
	Password        string     `json:"-"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
	Meta            string     `json:"meta,omitempty"`
	ReadingTime     int        `json:"reading_time"`
	WordCount       int        `json:"word_count"`
	AllowComments   bool       `json:"allow_comments"`
	IsFeatured      bool       `json:"is_featured"`
	IsSticky        bool       `json:"is_sticky"`
	SortOrder       int        `json:"sort_order"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CreateIn contains input for creating a post.
type CreateIn struct {
	Title           string   `json:"title"`
	Slug            string   `json:"slug,omitempty"`
	Excerpt         string   `json:"excerpt,omitempty"`
	Content         string   `json:"content,omitempty"`
	ContentFormat   string   `json:"content_format,omitempty"`
	FeaturedImageID string   `json:"featured_image_id,omitempty"`
	Status          string   `json:"status,omitempty"`
	Visibility      string   `json:"visibility,omitempty"`
	CategoryIDs     []string `json:"category_ids,omitempty"`
	TagIDs          []string `json:"tag_ids,omitempty"`
	Meta            string   `json:"meta,omitempty"`
	AllowComments   *bool    `json:"allow_comments,omitempty"`
}

// UpdateIn contains input for updating a post.
type UpdateIn struct {
	Title           *string    `json:"title,omitempty"`
	Slug            *string    `json:"slug,omitempty"`
	Excerpt         *string    `json:"excerpt,omitempty"`
	Content         *string    `json:"content,omitempty"`
	ContentFormat   *string    `json:"content_format,omitempty"`
	FeaturedImageID *string    `json:"featured_image_id,omitempty"`
	Status          *string    `json:"status,omitempty"`
	Visibility      *string    `json:"visibility,omitempty"`
	Password        *string    `json:"password,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
	Meta            *string    `json:"meta,omitempty"`
	AllowComments   *bool      `json:"allow_comments,omitempty"`
	IsFeatured      *bool      `json:"is_featured,omitempty"`
	IsSticky        *bool      `json:"is_sticky,omitempty"`
	CategoryIDs     []string   `json:"category_ids,omitempty"`
	TagIDs          []string   `json:"tag_ids,omitempty"`
}

// ListIn contains input for listing posts.
type ListIn struct {
	AuthorID   string
	Status     string
	Visibility string
	CategoryID string
	TagID      string
	IsFeatured *bool
	Search     string
	Limit      int
	Offset     int
	OrderBy    string
	Order      string
}

// API defines the posts service contract.
type API interface {
	Create(ctx context.Context, authorID string, in *CreateIn) (*Post, error)
	GetByID(ctx context.Context, id string) (*Post, error)
	GetBySlug(ctx context.Context, slug string) (*Post, error)
	List(ctx context.Context, in *ListIn) ([]*Post, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Post, error)
	Delete(ctx context.Context, id string) error
	Publish(ctx context.Context, id string) (*Post, error)
	Unpublish(ctx context.Context, id string) (*Post, error)
	GetCategoryIDs(ctx context.Context, postID string) ([]string, error)
	GetTagIDs(ctx context.Context, postID string) ([]string, error)
	SetCategories(ctx context.Context, postID string, categoryIDs []string) error
	SetTags(ctx context.Context, postID string, tagIDs []string) error
}

// Store defines the data access contract for posts.
type Store interface {
	Create(ctx context.Context, p *Post) error
	GetByID(ctx context.Context, id string) (*Post, error)
	GetBySlug(ctx context.Context, slug string) (*Post, error)
	List(ctx context.Context, in *ListIn) ([]*Post, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	GetCategoryIDs(ctx context.Context, postID string) ([]string, error)
	GetTagIDs(ctx context.Context, postID string) ([]string, error)
	SetCategories(ctx context.Context, postID string, categoryIDs []string) error
	SetTags(ctx context.Context, postID string, tagIDs []string) error
}
