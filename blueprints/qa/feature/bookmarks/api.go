package bookmarks

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("bookmark not found")
)

// Bookmark represents a favorite.
type Bookmark struct {
	ID         string    `json:"id"`
	AccountID  string    `json:"account_id"`
	QuestionID string    `json:"question_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// API defines the bookmarks service interface.
type API interface {
	Add(ctx context.Context, accountID, questionID string) (*Bookmark, error)
	Remove(ctx context.Context, accountID, questionID string) error
	ListByAccount(ctx context.Context, accountID string, limit int) ([]*Bookmark, error)
}

// Store defines the data storage interface for bookmarks.
type Store interface {
	Create(ctx context.Context, bookmark *Bookmark) error
	Delete(ctx context.Context, accountID, questionID string) error
	ListByAccount(ctx context.Context, accountID string, limit int) ([]*Bookmark, error)
}
