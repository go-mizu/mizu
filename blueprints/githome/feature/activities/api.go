package activities

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("activity not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Event types
const (
	EventPush          = "push"
	EventCreate        = "create"
	EventDelete        = "delete"
	EventFork          = "fork"
	EventStar          = "star"
	EventWatch         = "watch"
	EventIssueOpen     = "issue_open"
	EventIssueClose    = "issue_close"
	EventIssueReopen   = "issue_reopen"
	EventIssueComment  = "issue_comment"
	EventPROpen        = "pr_open"
	EventPRClose       = "pr_close"
	EventPRMerge       = "pr_merge"
	EventPRReopen      = "pr_reopen"
	EventPRComment     = "pr_comment"
	EventPRReview      = "pr_review"
	EventRelease       = "release"
	EventMember        = "member"
	EventPublic        = "public"
	EventCommitComment = "commit_comment"
)

// Ref types
const (
	RefTypeBranch = "branch"
	RefTypeTag    = "tag"
)

// Activity represents a user activity event
type Activity struct {
	ID         string    `json:"id"`
	ActorID    string    `json:"actor_id"`
	EventType  string    `json:"event_type"`
	RepoID     string    `json:"repo_id,omitempty"`
	TargetType string    `json:"target_type,omitempty"`
	TargetID   string    `json:"target_id,omitempty"`
	Ref        string    `json:"ref,omitempty"`
	RefType    string    `json:"ref_type,omitempty"`
	Payload    string    `json:"payload"`
	IsPublic   bool      `json:"is_public"`
	CreatedAt  time.Time `json:"created_at"`
}

// RecordIn is the input for recording an activity
type RecordIn struct {
	ActorID    string `json:"actor_id"`
	EventType  string `json:"event_type"`
	RepoID     string `json:"repo_id,omitempty"`
	TargetType string `json:"target_type,omitempty"`
	TargetID   string `json:"target_id,omitempty"`
	Ref        string `json:"ref,omitempty"`
	RefType    string `json:"ref_type,omitempty"`
	Payload    string `json:"payload,omitempty"`
	IsPublic   bool   `json:"is_public"`
}

// ListOpts are options for listing activities
type ListOpts struct {
	Limit  int
	Offset int
}

// API is the activities service interface
type API interface {
	Record(ctx context.Context, in *RecordIn) (*Activity, error)
	GetByID(ctx context.Context, id string) (*Activity, error)
	ListByUser(ctx context.Context, userID string, opts *ListOpts) ([]*Activity, error)
	ListByRepo(ctx context.Context, repoID string, opts *ListOpts) ([]*Activity, error)
	ListPublic(ctx context.Context, opts *ListOpts) ([]*Activity, error)
	ListFeed(ctx context.Context, userID string, opts *ListOpts) ([]*Activity, error)
	Delete(ctx context.Context, id string) error
}

// Store is the activities data store interface
type Store interface {
	Create(ctx context.Context, a *Activity) error
	GetByID(ctx context.Context, id string) (*Activity, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Activity, error)
	ListByRepo(ctx context.Context, repoID string, limit, offset int) ([]*Activity, error)
	ListPublic(ctx context.Context, limit, offset int) ([]*Activity, error)
	Delete(ctx context.Context, id string) error
}
