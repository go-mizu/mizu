// Package forums provides forum/community management functionality.
package forums

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a forum is not found.
	ErrNotFound = errors.New("forum not found")

	// ErrUnauthorized is returned when user lacks permission.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrSlugTaken is returned when slug is already used.
	ErrSlugTaken = errors.New("slug already taken")
)

// Forum represents a discussion category.
type Forum struct {
	ID          string         `json:"id"`
	ParentID    string         `json:"parent_id,omitempty"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description"`
	Icon        string         `json:"icon,omitempty"`
	Banner      string         `json:"banner,omitempty"`
	Type        string         `json:"type"` // public, restricted, private, archived
	NSFW        bool           `json:"nsfw"`
	Archived    bool           `json:"archived"`
	ThreadCount int            `json:"thread_count"`
	PostCount   int            `json:"post_count"`
	MemberCount int            `json:"member_count"`
	Position    int            `json:"position"`
	Settings    *ForumSettings `json:"settings,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// Relationships
	Parent   *Forum   `json:"parent,omitempty"`
	Children []*Forum `json:"children,omitempty"`
	Rules    []*Rule  `json:"rules,omitempty"`

	// Current user state
	IsMember     bool   `json:"is_member"`
	IsModerator  bool   `json:"is_moderator"`
	Role         string `json:"role,omitempty"`
}

// ForumSettings contains forum configuration.
type ForumSettings struct {
	AllowPolls        bool `json:"allow_polls"`
	AllowImages       bool `json:"allow_images"`
	AllowVideos       bool `json:"allow_videos"`
	RequireApproval   bool `json:"require_approval"`
	MinKarmaToPost    int  `json:"min_karma_to_post"`
	MinAgeToPost      int  `json:"min_age_to_post"` // Days
	RateLimitPosts    int  `json:"rate_limit_posts"`    // Per day
	RateLimitComments int  `json:"rate_limit_comments"` // Per day
}

// Rule represents a forum rule.
type Rule struct {
	ID          string    `json:"id"`
	ForumID     string    `json:"forum_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateIn contains input for creating a forum.
type CreateIn struct {
	ParentID    string         `json:"parent_id,omitempty"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug,omitempty"`
	Description string         `json:"description"`
	Icon        string         `json:"icon,omitempty"`
	Type        string         `json:"type,omitempty"`
	NSFW        bool           `json:"nsfw,omitempty"`
	Settings    *ForumSettings `json:"settings,omitempty"`
}

// UpdateIn contains input for updating a forum.
type UpdateIn struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Icon        *string         `json:"icon,omitempty"`
	Banner      *string         `json:"banner,omitempty"`
	Type        *string         `json:"type,omitempty"`
	NSFW        *bool           `json:"nsfw,omitempty"`
	Archived    *bool           `json:"archived,omitempty"`
	Settings    *ForumSettings  `json:"settings,omitempty"`
}

// ForumList is a list of forums.
type ForumList struct {
	Forums []*Forum `json:"forums"`
	Total  int      `json:"total"`
}

// API defines the forums service contract.
type API interface {
	// Forum operations
	Create(ctx context.Context, accountID string, in *CreateIn) (*Forum, error)
	GetByID(ctx context.Context, id, viewerID string) (*Forum, error)
	GetBySlug(ctx context.Context, slug, viewerID string) (*Forum, error)
	Update(ctx context.Context, id, accountID string, in *UpdateIn) (*Forum, error)
	Delete(ctx context.Context, id, accountID string) error
	List(ctx context.Context, parentID, viewerID string) ([]*Forum, error)
	ListAll(ctx context.Context, viewerID string) ([]*Forum, error)

	// Membership
	Join(ctx context.Context, forumID, accountID string) error
	Leave(ctx context.Context, forumID, accountID string) error
	IsMember(ctx context.Context, forumID, accountID string) (bool, error)

	// Moderation
	AddModerator(ctx context.Context, forumID, accountID, moderatorID string) error
	RemoveModerator(ctx context.Context, forumID, accountID, moderatorID string) error
	IsModerator(ctx context.Context, forumID, accountID string) (bool, string, error)
}

// Store defines the data access contract for forums.
type Store interface {
	// Forum operations
	Insert(ctx context.Context, f *Forum) error
	GetByID(ctx context.Context, id string) (*Forum, error)
	GetBySlug(ctx context.Context, slug string) (*Forum, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, parentID string) ([]*Forum, error)
	ListAll(ctx context.Context) ([]*Forum, error)
	ExistsSlug(ctx context.Context, slug string) (bool, error)

	// Membership
	Join(ctx context.Context, forumID, accountID string) error
	Leave(ctx context.Context, forumID, accountID string) error
	IsMember(ctx context.Context, forumID, accountID string) (bool, error)
	GetRole(ctx context.Context, forumID, accountID string) (string, error)

	// Moderation
	AddModerator(ctx context.Context, forumID, accountID, role string) error
	RemoveModerator(ctx context.Context, forumID, accountID string) error

	// Counters
	IncrementThreads(ctx context.Context, forumID string) error
	DecrementThreads(ctx context.Context, forumID string) error
	IncrementPosts(ctx context.Context, forumID string) error
	DecrementPosts(ctx context.Context, forumID string) error
}
