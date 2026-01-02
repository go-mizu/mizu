package tags

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("tag not found")
)

// Tag represents a tag.
type Tag struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Excerpt       string    `json:"excerpt"`
	Wiki          string    `json:"wiki"`
	QuestionCount int64     `json:"question_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// ListOpts contains options for listing tags.
type ListOpts struct {
	Limit  int
	Query  string
	SortBy string
}

// API defines the tags service interface.
type API interface {
	Create(ctx context.Context, name string, excerpt string) (*Tag, error)
	UpsertBatch(ctx context.Context, names []string) error
	GetByName(ctx context.Context, name string) (*Tag, error)
	GetByNames(ctx context.Context, names []string) (map[string]*Tag, error)
	List(ctx context.Context, opts ListOpts) ([]*Tag, error)
	IncrementQuestionCount(ctx context.Context, name string, delta int64) error
	IncrementQuestionCountBatch(ctx context.Context, names []string, delta int64) error
}

// Store defines the data storage interface for tags.
type Store interface {
	Create(ctx context.Context, tag *Tag) error
	GetByName(ctx context.Context, name string) (*Tag, error)
	GetByNames(ctx context.Context, names []string) (map[string]*Tag, error)
	List(ctx context.Context, opts ListOpts) ([]*Tag, error)
	IncrementQuestionCount(ctx context.Context, name string, delta int64) error
	IncrementQuestionCountBatch(ctx context.Context, names []string, delta int64) error
}
