// Package posts provides post management for posts, pages, and custom post types.
package posts

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("post not found")
	ErrInvalidSlug  = errors.New("invalid slug")
	ErrSlugTaken    = errors.New("slug already taken")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidStatus = errors.New("invalid status")
)

// Post statuses
const (
	StatusPublish    = "publish"
	StatusDraft      = "draft"
	StatusPending    = "pending"
	StatusPrivate    = "private"
	StatusFuture     = "future"
	StatusTrash      = "trash"
	StatusAutoDraft  = "auto-draft"
	StatusInherit    = "inherit"
)

// Post types
const (
	TypePost       = "post"
	TypePage       = "page"
	TypeAttachment = "attachment"
	TypeRevision   = "revision"
	TypeNavMenuItem = "nav_menu_item"
)

// Post represents a WordPress-compatible post.
type Post struct {
	ID              string                 `json:"id"`
	Date            time.Time              `json:"date"`
	DateGmt         time.Time              `json:"date_gmt"`
	GUID            RenderedField          `json:"guid"`
	Modified        time.Time              `json:"modified"`
	ModifiedGmt     time.Time              `json:"modified_gmt"`
	Slug            string                 `json:"slug"`
	Status          string                 `json:"status"`
	Type            string                 `json:"type"`
	Link            string                 `json:"link,omitempty"`
	Title           RenderedField          `json:"title"`
	Content         RenderedProtected      `json:"content"`
	Excerpt         RenderedProtected      `json:"excerpt"`
	Author          string                 `json:"author"`
	FeaturedMedia   string                 `json:"featured_media,omitempty"`
	CommentStatus   string                 `json:"comment_status"`
	PingStatus      string                 `json:"ping_status"`
	Sticky          bool                   `json:"sticky,omitempty"`
	Template        string                 `json:"template,omitempty"`
	Format          string                 `json:"format,omitempty"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
	Categories      []string               `json:"categories,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	// Page-specific
	Parent          string                 `json:"parent,omitempty"`
	MenuOrder       int                    `json:"menu_order,omitempty"`
	// Password protection
	Password        string                 `json:"-"`
	// Links
	Links           map[string]interface{} `json:"_links,omitempty"`
	// Embedded data
	Embedded        map[string]interface{} `json:"_embedded,omitempty"`
}

// RenderedField represents a field with raw and rendered versions.
type RenderedField struct {
	Raw      string `json:"raw,omitempty"`
	Rendered string `json:"rendered"`
}

// RenderedProtected is like RenderedField but can be password-protected.
type RenderedProtected struct {
	Raw       string `json:"raw,omitempty"`
	Rendered  string `json:"rendered"`
	Protected bool   `json:"protected"`
}

// CreateIn contains input for creating a post.
type CreateIn struct {
	Title         string                 `json:"title"`
	Content       string                 `json:"content"`
	Excerpt       string                 `json:"excerpt,omitempty"`
	Status        string                 `json:"status,omitempty"`
	Slug          string                 `json:"slug,omitempty"`
	Author        string                 `json:"author,omitempty"`
	FeaturedMedia string                 `json:"featured_media,omitempty"`
	CommentStatus string                 `json:"comment_status,omitempty"`
	PingStatus    string                 `json:"ping_status,omitempty"`
	Format        string                 `json:"format,omitempty"`
	Sticky        bool                   `json:"sticky,omitempty"`
	Template      string                 `json:"template,omitempty"`
	Categories    []string               `json:"categories,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Date          *time.Time             `json:"date,omitempty"`
	Password      string                 `json:"password,omitempty"`
	// Page-specific
	Parent        string                 `json:"parent,omitempty"`
	MenuOrder     int                    `json:"menu_order,omitempty"`
	// Post type (defaults to "post")
	Type          string                 `json:"-"`
}

// UpdateIn contains input for updating a post.
type UpdateIn struct {
	Title         *string                `json:"title,omitempty"`
	Content       *string                `json:"content,omitempty"`
	Excerpt       *string                `json:"excerpt,omitempty"`
	Status        *string                `json:"status,omitempty"`
	Slug          *string                `json:"slug,omitempty"`
	Author        *string                `json:"author,omitempty"`
	FeaturedMedia *string                `json:"featured_media,omitempty"`
	CommentStatus *string                `json:"comment_status,omitempty"`
	PingStatus    *string                `json:"ping_status,omitempty"`
	Format        *string                `json:"format,omitempty"`
	Sticky        *bool                  `json:"sticky,omitempty"`
	Template      *string                `json:"template,omitempty"`
	Categories    []string               `json:"categories,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Date          *time.Time             `json:"date,omitempty"`
	Password      *string                `json:"password,omitempty"`
	// Page-specific
	Parent        *string                `json:"parent,omitempty"`
	MenuOrder     *int                   `json:"menu_order,omitempty"`
}

// ListOpts contains options for listing posts.
type ListOpts struct {
	Page              int        `json:"page"`
	PerPage           int        `json:"per_page"`
	Search            string     `json:"search"`
	After             *time.Time `json:"after"`
	Before            *time.Time `json:"before"`
	ModifiedAfter     *time.Time `json:"modified_after"`
	ModifiedBefore    *time.Time `json:"modified_before"`
	Author            []string   `json:"author"`
	AuthorExclude     []string   `json:"author_exclude"`
	Include           []string   `json:"include"`
	Exclude           []string   `json:"exclude"`
	Offset            int        `json:"offset"`
	Order             string     `json:"order"`
	OrderBy           string     `json:"orderby"`
	Slug              []string   `json:"slug"`
	Status            []string   `json:"status"`
	Categories        []string   `json:"categories"`
	CategoriesExclude []string   `json:"categories_exclude"`
	Tags              []string   `json:"tags"`
	TagsExclude       []string   `json:"tags_exclude"`
	Sticky            *bool      `json:"sticky"`
	Parent            []string   `json:"parent"`
	ParentExclude     []string   `json:"parent_exclude"`
	// Post type for filtering
	Type              string     `json:"-"`
}

// Revision represents a post revision.
type Revision struct {
	ID         string        `json:"id"`
	Author     string        `json:"author"`
	Date       time.Time     `json:"date"`
	DateGmt    time.Time     `json:"date_gmt"`
	Parent     string        `json:"parent"`
	Title      RenderedField `json:"title"`
	Content    RenderedField `json:"content"`
	Excerpt    RenderedField `json:"excerpt"`
}

// API defines the posts service interface.
type API interface {
	// Post management
	Create(ctx context.Context, in CreateIn) (*Post, error)
	GetByID(ctx context.Context, id string) (*Post, error)
	GetBySlug(ctx context.Context, slug, postType string) (*Post, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Post, error)
	Delete(ctx context.Context, id string, force bool) error

	// Trash management
	Trash(ctx context.Context, id string) (*Post, error)
	Restore(ctx context.Context, id string) (*Post, error)

	// Lists
	List(ctx context.Context, opts ListOpts) ([]*Post, int, error)
	Count(ctx context.Context, postType, status string) (int, error)

	// Revisions
	CreateRevision(ctx context.Context, postID string) (*Revision, error)
	GetRevisions(ctx context.Context, postID string) ([]*Revision, error)
	GetRevision(ctx context.Context, postID, revisionID string) (*Revision, error)
	DeleteRevision(ctx context.Context, postID, revisionID string) error

	// Autosaves
	CreateAutosave(ctx context.Context, postID string, in UpdateIn) (*Revision, error)
	GetAutosaves(ctx context.Context, postID string) ([]*Revision, error)

	// Meta
	GetMeta(ctx context.Context, postID, key string) (string, error)
	SetMeta(ctx context.Context, postID, key, value string) error
	DeleteMeta(ctx context.Context, postID, key string) error
	GetAllMeta(ctx context.Context, postID string) (map[string]string, error)

	// Terms
	SetTerms(ctx context.Context, postID string, taxonomy string, termIDs []string) error
	GetTerms(ctx context.Context, postID string, taxonomy string) ([]string, error)

	// Sticky
	GetStickyPosts(ctx context.Context) ([]string, error)
	SetSticky(ctx context.Context, postID string, sticky bool) error
}
