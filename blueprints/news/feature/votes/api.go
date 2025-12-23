package votes

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound      = errors.New("vote not found")
	ErrAlreadyVoted  = errors.New("already voted")
	ErrInvalidTarget = errors.New("invalid vote target")
)

// Target types
const (
	TargetStory   = "story"
	TargetComment = "comment"
)

// Vote represents a user's vote on a story or comment.
type Vote struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TargetType string    `json:"target_type"` // "story" or "comment"
	TargetID   string    `json:"target_id"`
	Value      int       `json:"value"` // 1 for upvote
	CreatedAt  time.Time `json:"created_at"`
}

// VoteIn contains input for voting.
type VoteIn struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Value      int    `json:"value"`
}

// Validate validates the vote input.
func (in *VoteIn) Validate() error {
	if in.TargetType != TargetStory && in.TargetType != TargetComment {
		return ErrInvalidTarget
	}
	if in.TargetID == "" {
		return errors.New("target_id is required")
	}
	// Only upvotes for now (like HN)
	if in.Value != 1 {
		in.Value = 1
	}
	return nil
}

// API defines the votes service interface.
type API interface {
	// Vote on a target (story or comment)
	Vote(ctx context.Context, userID string, in VoteIn) error

	// Remove a vote
	Unvote(ctx context.Context, userID, targetType, targetID string) error

	// Get user's vote on a target
	GetVote(ctx context.Context, userID, targetType, targetID string) (*Vote, error)

	// Get user's votes on multiple targets
	GetVotes(ctx context.Context, userID, targetType string, targetIDs []string) (map[string]*Vote, error)
}

// Store defines the data storage interface for votes.
type Store interface {
	Create(ctx context.Context, vote *Vote) error
	GetByUserAndTarget(ctx context.Context, userID, targetType, targetID string) (*Vote, error)
	GetByUserAndTargets(ctx context.Context, userID, targetType string, targetIDs []string) (map[string]*Vote, error)
	Delete(ctx context.Context, userID, targetType, targetID string) error
	CountByTarget(ctx context.Context, targetType, targetID string) (int64, error)
}
