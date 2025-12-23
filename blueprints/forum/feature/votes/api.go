package votes

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("vote not found")
	ErrInvalidValue = errors.New("invalid vote value")
)

// Target types
const (
	TargetThread  = "thread"
	TargetComment = "comment"
)

// Vote represents a user's vote on content.
type Vote struct {
	ID         string    `json:"id"`
	AccountID  string    `json:"account_id"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Value      int       `json:"value"` // -1 or 1
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// API defines the votes service interface.
type API interface {
	// Voting
	Vote(ctx context.Context, accountID, targetType, targetID string, value int) error
	Unvote(ctx context.Context, accountID, targetType, targetID string) error
	GetVote(ctx context.Context, accountID, targetType, targetID string) (*Vote, error)

	// Batch operations
	GetVotes(ctx context.Context, accountID, targetType string, targetIDs []string) (map[string]int, error)

	// Stats
	GetVoteCounts(ctx context.Context, targetType, targetID string) (up, down int64, err error)
}

// Store defines the data storage interface for votes.
type Store interface {
	Create(ctx context.Context, vote *Vote) error
	GetByTarget(ctx context.Context, accountID, targetType, targetID string) (*Vote, error)
	Update(ctx context.Context, vote *Vote) error
	Delete(ctx context.Context, accountID, targetType, targetID string) error

	// Batch operations
	GetByTargets(ctx context.Context, accountID, targetType string, targetIDs []string) ([]*Vote, error)

	// Stats
	CountByTarget(ctx context.Context, targetType, targetID string) (up, down int64, err error)
}

// ThreadVoter interface for updating thread votes.
type ThreadVoter interface {
	UpdateVotes(ctx context.Context, id string, upDelta, downDelta int64) error
}

// CommentVoter interface for updating comment votes.
type CommentVoter interface {
	UpdateVotes(ctx context.Context, id string, upDelta, downDelta int64) error
}
