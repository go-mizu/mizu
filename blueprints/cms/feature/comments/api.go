// Package comments provides comment management functionality.
package comments

import (
	"context"
	"time"
)

// Comment represents a comment on a post.
type Comment struct {
	ID          string    `json:"id"`
	PostID      string    `json:"post_id"`
	ParentID    string    `json:"parent_id,omitempty"`
	AuthorID    string    `json:"author_id,omitempty"`
	AuthorName  string    `json:"author_name,omitempty"`
	AuthorEmail string    `json:"author_email,omitempty"`
	AuthorURL   string    `json:"author_url,omitempty"`
	Content     string    `json:"content"`
	Status      string    `json:"status"`
	IPAddress   string    `json:"ip_address,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
	LikesCount  int       `json:"likes_count"`
	Meta        string    `json:"meta,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a comment.
type CreateIn struct {
	PostID      string `json:"post_id"`
	ParentID    string `json:"parent_id,omitempty"`
	AuthorID    string `json:"author_id,omitempty"`
	AuthorName  string `json:"author_name,omitempty"`
	AuthorEmail string `json:"author_email,omitempty"`
	AuthorURL   string `json:"author_url,omitempty"`
	Content     string `json:"content"`
	IPAddress   string `json:"-"`
	UserAgent   string `json:"-"`
}

// UpdateIn contains input for updating a comment.
type UpdateIn struct {
	Content *string `json:"content,omitempty"`
	Status  *string `json:"status,omitempty"`
	Meta    *string `json:"meta,omitempty"`
}

// ListIn contains input for listing comments.
type ListIn struct {
	PostID   string
	ParentID string
	AuthorID string
	Status   string
	Limit    int
	Offset   int
}

// API defines the comments service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Comment, error)
	GetByID(ctx context.Context, id string) (*Comment, error)
	ListByPost(ctx context.Context, postID string, in *ListIn) ([]*Comment, int, error)
	List(ctx context.Context, in *ListIn) ([]*Comment, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Comment, error)
	Delete(ctx context.Context, id string) error
	Approve(ctx context.Context, id string) (*Comment, error)
	MarkAsSpam(ctx context.Context, id string) (*Comment, error)
	CountByPost(ctx context.Context, postID string) (int, error)
}

// Store defines the data access contract for comments.
type Store interface {
	Create(ctx context.Context, c *Comment) error
	GetByID(ctx context.Context, id string) (*Comment, error)
	ListByPost(ctx context.Context, postID string, in *ListIn) ([]*Comment, int, error)
	List(ctx context.Context, in *ListIn) ([]*Comment, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	CountByPost(ctx context.Context, postID string) (int, error)
}
