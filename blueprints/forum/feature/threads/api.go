// Package threads provides thread/topic management functionality.
package threads

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/blueprints/forum/feature/forums"
)

var (
	// ErrNotFound is returned when a thread is not found.
	ErrNotFound = errors.New("thread not found")

	// ErrUnauthorized is returned when user lacks permission.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrLocked is returned when thread is locked.
	ErrLocked = errors.New("thread locked")
)

// Thread represents a discussion thread.
type Thread struct {
	ID                 string    `json:"id"`
	ForumID            string    `json:"forum_id"`
	AccountID          string    `json:"account_id"`
	Type               string    `json:"type"` // discussion, question, poll, announcement
	Title              string    `json:"title"`
	Content            string    `json:"content"`
	Slug               string    `json:"slug"`
	Sticky             bool      `json:"sticky"`
	Locked             bool      `json:"locked"`
	NSFW               bool      `json:"nsfw"`
	Spoiler            bool      `json:"spoiler"`
	State              string    `json:"state"` // open, locked, archived, removed, pending
	ViewCount          int       `json:"view_count"`
	Score              int       `json:"score"`
	Upvotes            int       `json:"upvotes"`
	Downvotes          int       `json:"downvotes"`
	PostCount          int       `json:"post_count"`
	BestPostID         string    `json:"best_post_id,omitempty"`
	HotScore           float64   `json:"hot_score"`
	BestScore          float64   `json:"best_score"`
	ControversialScore float64   `json:"controversial_score"`
	LastPostAt         time.Time `json:"last_post_at"`
	CreatedAt          time.Time `json:"created_at"`
	EditedAt           *time.Time `json:"edited_at,omitempty"`

	// Relationships
	Forum   *forums.Forum     `json:"forum,omitempty"`
	Account *accounts.Account `json:"account,omitempty"`
	Tags    []string          `json:"tags,omitempty"`

	// Current user state
	UserVote     int  `json:"user_vote,omitempty"` // -1, 0, 1
	IsSaved      bool `json:"is_saved"`
	IsSubscribed bool `json:"is_subscribed"`
	IsOwner      bool `json:"is_owner"`
}

// CreateIn contains input for creating a thread.
type CreateIn struct {
	ForumID string   `json:"forum_id"`
	Type    string   `json:"type,omitempty"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	NSFW    bool     `json:"nsfw,omitempty"`
	Spoiler bool     `json:"spoiler,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

// UpdateIn contains input for updating a thread.
type UpdateIn struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
	NSFW    *bool   `json:"nsfw,omitempty"`
	Spoiler *bool   `json:"spoiler,omitempty"`
}

// ThreadList is a paginated list of threads.
type ThreadList struct {
	Threads []*Thread `json:"threads"`
	MaxID   string    `json:"max_id,omitempty"`
	MinID   string    `json:"min_id,omitempty"`
}

// SortOption defines thread sorting options.
type SortOption string

const (
	SortHot           SortOption = "hot"
	SortNew           SortOption = "new"
	SortTop           SortOption = "top"
	SortBest          SortOption = "best"
	SortRising        SortOption = "rising"
	SortControversial SortOption = "controversial"
)

// API defines the threads service contract.
type API interface {
	// Thread operations
	Create(ctx context.Context, accountID string, in *CreateIn) (*Thread, error)
	GetByID(ctx context.Context, id, viewerID string) (*Thread, error)
	Update(ctx context.Context, id, accountID string, in *UpdateIn) (*Thread, error)
	Delete(ctx context.Context, id, accountID string) error

	// Listing
	ListByForum(ctx context.Context, forumID, viewerID string, sort SortOption, limit int, after string) (*ThreadList, error)
	ListByAccount(ctx context.Context, accountID, viewerID string, limit int, after string) (*ThreadList, error)

	// Moderation
	Lock(ctx context.Context, threadID, accountID string) error
	Unlock(ctx context.Context, threadID, accountID string) error
	Sticky(ctx context.Context, threadID, accountID string) error
	Unsticky(ctx context.Context, threadID, accountID string) error
	SetBestPost(ctx context.Context, threadID, postID, accountID string) error

	// View tracking
	IncrementViews(ctx context.Context, threadID string) error
}

// Store defines the data access contract for threads.
type Store interface {
	// Thread operations
	Insert(ctx context.Context, t *Thread) error
	GetByID(ctx context.Context, id string) (*Thread, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	GetOwner(ctx context.Context, id string) (accountID string, forumID string, err error)

	// Listing
	ListByForum(ctx context.Context, forumID string, sort SortOption, limit int, after string) ([]*Thread, error)
	ListByAccount(ctx context.Context, accountID string, limit int, after string) ([]*Thread, error)

	// Moderation
	SetLocked(ctx context.Context, id string, locked bool) error
	SetSticky(ctx context.Context, id string, sticky bool) error
	SetBestPost(ctx context.Context, threadID, postID string) error
	SetState(ctx context.Context, id, state string) error

	// Counters
	IncrementPosts(ctx context.Context, threadID string) error
	DecrementPosts(ctx context.Context, threadID string) error
	IncrementViews(ctx context.Context, threadID string) error
	UpdateLastPostAt(ctx context.Context, threadID string) error

	// Scores
	UpdateScores(ctx context.Context, threadID string, score, upvotes, downvotes int) error
}
