// Package comments provides comment functionality for issues.
package comments

import (
	"context"
	"time"
)

// Comment represents a comment on an issue.
type Comment struct {
	ID        string     `json:"id"`
	IssueID   string     `json:"issue_id"`
	AuthorID  string     `json:"author_id"`
	Content   string     `json:"content"`
	EditedAt  *time.Time `json:"edited_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateIn contains input for creating a comment.
type CreateIn struct {
	Content string `json:"content"`
}

// API defines the comments service contract.
type API interface {
	Create(ctx context.Context, issueID, authorID string, in *CreateIn) (*Comment, error)
	GetByID(ctx context.Context, id string) (*Comment, error)
	ListByIssue(ctx context.Context, issueID string) ([]*Comment, error)
	Update(ctx context.Context, id, content string) (*Comment, error)
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for comments.
type Store interface {
	Create(ctx context.Context, c *Comment) error
	GetByID(ctx context.Context, id string) (*Comment, error)
	ListByIssue(ctx context.Context, issueID string) ([]*Comment, error)
	Update(ctx context.Context, id, content string) error
	Delete(ctx context.Context, id string) error
	CountByIssue(ctx context.Context, issueID string) (int, error)
}
