package threads

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
)

// Errors
var (
	ErrNotFound     = errors.New("thread not found")
	ErrBoardLocked  = errors.New("board is archived")
	ErrThreadLocked = errors.New("thread is locked")
	ErrNotAuthor    = errors.New("not the thread author")
)

// Validation constants
const (
	TitleMinLen   = 1
	TitleMaxLen   = 300
	ContentMaxLen = 40000
	URLMaxLen     = 2000
)

// ThreadType defines the type of thread.
type ThreadType string

const (
	ThreadTypeText  ThreadType = "text"
	ThreadTypeLink  ThreadType = "link"
	ThreadTypeImage ThreadType = "image"
	ThreadTypePoll  ThreadType = "poll"
)

// SortBy defines sorting options.
type SortBy string

const (
	SortHot           SortBy = "hot"
	SortNew           SortBy = "new"
	SortTop           SortBy = "top"
	SortRising        SortBy = "rising"
	SortControversial SortBy = "controversial"
)

// TimeRange defines time filtering options.
type TimeRange string

const (
	TimeHour  TimeRange = "hour"
	TimeDay   TimeRange = "day"
	TimeWeek  TimeRange = "week"
	TimeMonth TimeRange = "month"
	TimeYear  TimeRange = "year"
	TimeAll   TimeRange = "all"
)

// Thread represents a discussion thread.
type Thread struct {
	ID            string     `json:"id"`
	BoardID       string     `json:"board_id"`
	AuthorID      string     `json:"author_id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	ContentHTML   string     `json:"content_html"`
	URL           string     `json:"url,omitempty"`
	Domain        string     `json:"domain,omitempty"`
	ThumbnailURL  string     `json:"thumbnail_url,omitempty"`
	Type          ThreadType `json:"type"`
	Score         int64      `json:"score"`
	UpvoteCount   int64      `json:"upvote_count"`
	DownvoteCount int64      `json:"downvote_count"`
	CommentCount  int64      `json:"comment_count"`
	ViewCount     int64      `json:"view_count"`
	HotScore      float64    `json:"hot_score"`
	IsPinned      bool       `json:"is_pinned"`
	IsLocked      bool       `json:"is_locked"`
	IsRemoved     bool       `json:"is_removed"`
	IsNSFW        bool       `json:"is_nsfw"`
	IsSpoiler     bool       `json:"is_spoiler"`
	IsOC          bool       `json:"is_oc"`
	RemoveReason  string     `json:"remove_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	EditedAt      *time.Time `json:"edited_at,omitempty"`

	// Relationships
	Author *accounts.Account `json:"author,omitempty"`
	Board  *boards.Board     `json:"board,omitempty"`

	// Viewer state
	Vote         int  `json:"vote,omitempty"`
	IsBookmarked bool `json:"is_bookmarked,omitempty"`
	IsOwner      bool `json:"is_owner,omitempty"`
	CanEdit      bool `json:"can_edit,omitempty"`
	CanDelete    bool `json:"can_delete,omitempty"`
}

// CreateIn contains input for creating a thread.
type CreateIn struct {
	BoardID   string     `json:"board_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	URL       string     `json:"url"`
	Type      ThreadType `json:"type"`
	IsNSFW    bool       `json:"is_nsfw"`
	IsSpoiler bool       `json:"is_spoiler"`

	// Seeding fields (optional, for importing from external sources)
	InitialUpvotes   int64      `json:"initial_upvotes,omitempty"`
	InitialDownvotes int64      `json:"initial_downvotes,omitempty"`
	InitialComments  int64      `json:"initial_comments,omitempty"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
}

// UpdateIn contains input for updating a thread.
type UpdateIn struct {
	Content   *string `json:"content,omitempty"`
	IsNSFW    *bool   `json:"is_nsfw,omitempty"`
	IsSpoiler *bool   `json:"is_spoiler,omitempty"`
}

// ListOpts contains options for listing threads.
type ListOpts struct {
	Limit     int
	Cursor    string
	SortBy    SortBy
	TimeRange TimeRange
}

// API defines the threads service interface.
type API interface {
	// Thread management
	Create(ctx context.Context, authorID string, in CreateIn) (*Thread, error)
	GetByID(ctx context.Context, id string) (*Thread, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Thread, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Thread, error)
	Delete(ctx context.Context, id string) error
	IncrementViews(ctx context.Context, id string) error

	// Listing
	List(ctx context.Context, opts ListOpts) ([]*Thread, error)
	ListByBoard(ctx context.Context, boardID string, opts ListOpts) ([]*Thread, error)
	ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Thread, error)

	// Moderation
	Remove(ctx context.Context, id string, reason string) error
	Approve(ctx context.Context, id string) error
	Lock(ctx context.Context, id string) error
	Unlock(ctx context.Context, id string) error
	Pin(ctx context.Context, id string) error
	Unpin(ctx context.Context, id string) error
	SetNSFW(ctx context.Context, id string, nsfw bool) error
	SetSpoiler(ctx context.Context, id string, spoiler bool) error

	// Voting
	UpdateVotes(ctx context.Context, id string, upDelta, downDelta int64) error

	// Comments
	IncrementCommentCount(ctx context.Context, id string, delta int64) error

	// Viewer state
	EnrichThread(ctx context.Context, thread *Thread, viewerID string) error
	EnrichThreads(ctx context.Context, threads []*Thread, viewerID string) error

	// Recalculation
	RecalculateHotScores(ctx context.Context) error
}

// Store defines the data storage interface for threads.
type Store interface {
	Create(ctx context.Context, thread *Thread) error
	GetByID(ctx context.Context, id string) (*Thread, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Thread, error)
	Update(ctx context.Context, thread *Thread) error
	Delete(ctx context.Context, id string) error

	// Lists
	List(ctx context.Context, opts ListOpts) ([]*Thread, error)
	ListByBoard(ctx context.Context, boardID string, opts ListOpts) ([]*Thread, error)
	ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Thread, error)

	// Batch operations
	UpdateHotScores(ctx context.Context) error
}

// HotScore calculates the Reddit-style hot ranking score.
func HotScore(ups, downs int64, created time.Time) float64 {
	score := float64(ups - downs)
	order := math.Log10(math.Max(math.Abs(score), 1))

	sign := 0.0
	if score > 0 {
		sign = 1
	} else if score < 0 {
		sign = -1
	}

	// Reddit epoch: Dec 8, 2005
	seconds := created.Unix() - 1134028003

	return sign*order + float64(seconds)/45000
}

// ControversialScore calculates the controversial score.
func ControversialScore(ups, downs int64) float64 {
	if ups <= 0 || downs <= 0 {
		return 0
	}

	magnitude := float64(ups + downs)
	balance := float64(min(ups, downs)) / float64(max(ups, downs))

	return magnitude * balance
}
