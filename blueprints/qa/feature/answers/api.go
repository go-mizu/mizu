package answers

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
)

var (
	ErrNotFound  = errors.New("answer not found")
	ErrNotAuthor = errors.New("not the answer author")
	ErrInvalid   = errors.New("invalid answer")
)

const (
	BodyMaxLen = 30000
)

// Answer represents an answer.
type Answer struct {
	ID         string    `json:"id"`
	QuestionID string    `json:"question_id"`
	AuthorID   string    `json:"author_id"`
	Body       string    `json:"body"`
	BodyHTML   string    `json:"body_html"`
	Score      int64     `json:"score"`
	IsAccepted bool      `json:"is_accepted"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Author *accounts.Account `json:"author,omitempty"`
}

// CreateIn contains input for creating an answer.
type CreateIn struct {
	QuestionID string `json:"question_id"`
	Body       string `json:"body"`
}

// UpdateIn contains input for updating an answer.
type UpdateIn struct {
	Body *string `json:"body,omitempty"`
}

// ListOpts contains options for listing answers.
type ListOpts struct {
	Limit int
}

// API defines the answers service interface.
type API interface {
	Create(ctx context.Context, authorID string, in CreateIn) (*Answer, error)
	GetByID(ctx context.Context, id string) (*Answer, error)
	ListByQuestion(ctx context.Context, questionID string, opts ListOpts) ([]*Answer, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Answer, error)
	Delete(ctx context.Context, id string) error
	SetAccepted(ctx context.Context, id string, accepted bool) error
	UpdateScore(ctx context.Context, id string, delta int64) error
}

// Store defines the data storage interface for answers.
type Store interface {
	Create(ctx context.Context, answer *Answer) error
	GetByID(ctx context.Context, id string) (*Answer, error)
	ListByQuestion(ctx context.Context, questionID string, opts ListOpts) ([]*Answer, error)
	Update(ctx context.Context, answer *Answer) error
	Delete(ctx context.Context, id string) error
	SetAccepted(ctx context.Context, id string, accepted bool) error
	UpdateScore(ctx context.Context, id string, delta int64) error
}
