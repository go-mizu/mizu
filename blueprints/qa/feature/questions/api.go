package questions

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

var (
	ErrNotFound     = errors.New("question not found")
	ErrClosed       = errors.New("question is closed")
	ErrNotAuthor    = errors.New("not the question author")
	ErrInvalidTitle = errors.New("invalid title")
	ErrInvalidBody  = errors.New("invalid body")
)

const (
	TitleMinLen = 10
	TitleMaxLen = 300
	BodyMaxLen  = 40000
)

// SortBy defines sorting options.
type SortBy string

const (
	SortNewest     SortBy = "newest"
	SortActive     SortBy = "active"
	SortScore      SortBy = "score"
	SortUnanswered SortBy = "unanswered"
)

// Question represents a Q&A question.
type Question struct {
	ID               string    `json:"id"`
	AuthorID         string    `json:"author_id"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	BodyHTML         string    `json:"body_html"`
	Score            int64     `json:"score"`
	ViewCount        int64     `json:"view_count"`
	AnswerCount      int64     `json:"answer_count"`
	CommentCount     int64     `json:"comment_count"`
	FavoriteCount    int64     `json:"favorite_count"`
	AcceptedAnswerID string    `json:"accepted_answer_id"`
	BountyAmount     int64     `json:"bounty_amount"`
	IsClosed         bool      `json:"is_closed"`
	CloseReason      string    `json:"close_reason"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	Author *accounts.Account `json:"author,omitempty"`
	Tags   []*tags.Tag       `json:"tags,omitempty"`
}

// CreateIn contains input for creating a question.
type CreateIn struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	Tags  []string `json:"tags"`
}

// UpdateIn contains input for updating a question.
type UpdateIn struct {
	Title *string `json:"title,omitempty"`
	Body  *string `json:"body,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

// ListOpts contains options for listing questions.
type ListOpts struct {
	Limit  int
	Cursor string
	SortBy SortBy
	Query  string
}

// API defines the questions service interface.
type API interface {
	Create(ctx context.Context, authorID string, in CreateIn) (*Question, error)
	GetByID(ctx context.Context, id string) (*Question, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Question, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Question, error)
	Delete(ctx context.Context, id string) error
	IncrementViews(ctx context.Context, id string) error

	List(ctx context.Context, opts ListOpts) ([]*Question, error)
	ListByTag(ctx context.Context, tag string, opts ListOpts) ([]*Question, error)
	ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Question, error)
	Search(ctx context.Context, query string, limit int) ([]*Question, error)

	SetAcceptedAnswer(ctx context.Context, id string, answerID string) error
	Close(ctx context.Context, id string, reason string) error
	Reopen(ctx context.Context, id string) error
	UpdateStats(ctx context.Context, id string, answerDelta, commentDelta, favoriteDelta int64) error
	UpdateScore(ctx context.Context, id string, delta int64) error
}

// Store defines the data storage interface for questions.
type Store interface {
	Create(ctx context.Context, question *Question) error
	GetByID(ctx context.Context, id string) (*Question, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Question, error)
	Update(ctx context.Context, question *Question) error
	Delete(ctx context.Context, id string) error
	IncrementViews(ctx context.Context, id string) error

	List(ctx context.Context, opts ListOpts) ([]*Question, error)
	ListByTag(ctx context.Context, tag string, opts ListOpts) ([]*Question, error)
	ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Question, error)
	Search(ctx context.Context, query string, limit int) ([]*Question, error)

	SetAcceptedAnswer(ctx context.Context, id string, answerID string) error
	SetClosed(ctx context.Context, id string, closed bool, reason string) error
	UpdateStats(ctx context.Context, id string, answerDelta, commentDelta, favoriteDelta int64) error
	UpdateScore(ctx context.Context, id string, delta int64) error
	SetTags(ctx context.Context, questionID string, tags []string) error
	GetTags(ctx context.Context, questionID string) ([]*tags.Tag, error)
}
