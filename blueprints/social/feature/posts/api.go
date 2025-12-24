// Package posts provides post management functionality.
package posts

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/social/feature/accounts"
)

// Visibility levels
const (
	VisibilityPublic    = "public"
	VisibilityFollowers = "followers"
	VisibilityMentioned = "mentioned"
	VisibilityPrivate   = "private"
)

// Errors
var (
	ErrNotFound     = errors.New("post not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrTooLong      = errors.New("post content too long")
	ErrEmpty        = errors.New("post content is empty")
)

// Post represents a post/status.
type Post struct {
	ID             string     `json:"id"`
	AccountID      string     `json:"account_id"`
	Content        string     `json:"content"`
	ContentWarning string     `json:"content_warning,omitempty"`
	Visibility     string     `json:"visibility"`
	ReplyToID      string     `json:"reply_to_id,omitempty"`
	ThreadID       string     `json:"thread_id,omitempty"`
	QuoteOfID      string     `json:"quote_of_id,omitempty"`
	Language       string     `json:"language,omitempty"`
	Sensitive      bool       `json:"sensitive"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	LikesCount     int        `json:"likes_count"`
	RepostsCount   int        `json:"reposts_count"`
	RepliesCount   int        `json:"replies_count"`
	QuotesCount    int        `json:"quotes_count"`

	// Enriched fields
	Account  *accounts.Account `json:"account,omitempty"`
	ReplyTo  *Post             `json:"reply_to,omitempty"`
	QuoteOf  *Post             `json:"quote_of,omitempty"`
	Media    []*Media          `json:"media,omitempty"`
	Mentions []string          `json:"mentions,omitempty"`
	Hashtags []string          `json:"hashtags,omitempty"`

	// Viewer state
	Liked      bool `json:"liked,omitempty"`
	Reposted   bool `json:"reposted,omitempty"`
	Bookmarked bool `json:"bookmarked,omitempty"`
}

// Media represents a media attachment.
type Media struct {
	ID         string `json:"id"`
	PostID     string `json:"post_id"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	PreviewURL string `json:"preview_url,omitempty"`
	AltText    string `json:"alt_text,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Position   int    `json:"position"`
}

// CreateIn contains input for creating a post.
type CreateIn struct {
	Content        string   `json:"content"`
	ContentWarning string   `json:"content_warning,omitempty"`
	Visibility     string   `json:"visibility,omitempty"`
	ReplyToID      string   `json:"reply_to_id,omitempty"`
	QuoteOfID      string   `json:"quote_of_id,omitempty"`
	Language       string   `json:"language,omitempty"`
	Sensitive      bool     `json:"sensitive"`
	MediaIDs       []string `json:"media_ids,omitempty"`
}

// UpdateIn contains input for updating a post.
type UpdateIn struct {
	Content        *string `json:"content,omitempty"`
	ContentWarning *string `json:"content_warning,omitempty"`
	Sensitive      *bool   `json:"sensitive,omitempty"`
}

// ListOpts specifies options for listing posts.
type ListOpts struct {
	Limit          int
	Offset         int
	AccountID      string
	ExcludeReplies bool
	OnlyMedia      bool
}

// Context represents a post's thread context.
type Context struct {
	Ancestors   []*Post `json:"ancestors"`
	Descendants []*Post `json:"descendants"`
}

// API defines the posts service contract.
type API interface {
	Create(ctx context.Context, accountID string, in *CreateIn) (*Post, error)
	GetByID(ctx context.Context, id string) (*Post, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Post, error)
	Update(ctx context.Context, accountID, postID string, in *UpdateIn) (*Post, error)
	Delete(ctx context.Context, accountID, postID string) error
	GetContext(ctx context.Context, id string) (*Context, error)
	List(ctx context.Context, opts ListOpts) ([]*Post, error)
	GetReplies(ctx context.Context, postID string, limit, offset int) ([]*Post, error)

	// Enrichment
	PopulateAccount(ctx context.Context, p *Post) error
	PopulateAccounts(ctx context.Context, posts []*Post) error
	PopulateViewerState(ctx context.Context, p *Post, viewerID string) error
	PopulateViewerStates(ctx context.Context, posts []*Post, viewerID string) error
}

// Store defines the data access contract for posts.
type Store interface {
	Insert(ctx context.Context, p *Post) error
	GetByID(ctx context.Context, id string) (*Post, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Post, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, opts ListOpts) ([]*Post, error)
	GetReplies(ctx context.Context, postID string, limit, offset int) ([]*Post, error)
	GetAncestors(ctx context.Context, postID string) ([]*Post, error)
	GetDescendants(ctx context.Context, postID string, limit int) ([]*Post, error)
	IncrementRepliesCount(ctx context.Context, id string) error
	DecrementRepliesCount(ctx context.Context, id string) error
	IncrementQuotesCount(ctx context.Context, id string) error

	// Media
	InsertMedia(ctx context.Context, m *Media) error
	GetMediaByPostID(ctx context.Context, postID string) ([]*Media, error)
	DeleteMediaByPostID(ctx context.Context, postID string) error

	// Hashtags
	UpsertHashtag(ctx context.Context, name string) (string, error)
	LinkPostHashtag(ctx context.Context, postID, hashtagID string) error
	GetHashtagsByPostID(ctx context.Context, postID string) ([]string, error)

	// Mentions
	InsertMention(ctx context.Context, postID, accountID string) error
	GetMentionsByPostID(ctx context.Context, postID string) ([]string, error)

	// Edit history
	InsertEditHistory(ctx context.Context, postID, content, contentWarning string, sensitive bool) error
}
