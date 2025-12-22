// Package votes provides voting and karma functionality.
package votes

import (
	"context"
	"errors"
)

var (
	// ErrInvalidTarget is returned for invalid vote target.
	ErrInvalidTarget = errors.New("invalid vote target")

	// ErrInvalidValue is returned for invalid vote value.
	ErrInvalidValue = errors.New("invalid vote value")

	// ErrSelfVote is returned when trying to vote on own content.
	ErrSelfVote = errors.New("cannot vote on own content")
)

const (
	// TargetThread is for thread votes.
	TargetThread = "thread"

	// TargetPost is for post votes.
	TargetPost = "post"
)

// Vote represents a vote on content.
type Vote struct {
	AccountID  string `json:"account_id"`
	TargetType string `json:"target_type"` // thread, post
	TargetID   string `json:"target_id"`
	Value      int    `json:"value"` // -1, 0, 1
}

// API defines the votes service contract.
type API interface {
	// Voting
	Vote(ctx context.Context, accountID, targetType, targetID string, value int) error
	GetVote(ctx context.Context, accountID, targetType, targetID string) (int, error)
	HasVoted(ctx context.Context, accountID, targetType, targetID string) (bool, int, error)
}

// Store defines the data access contract for votes.
type Store interface {
	// Vote operations
	Upsert(ctx context.Context, accountID, targetType, targetID string, value int) (oldValue int, err error)
	Get(ctx context.Context, accountID, targetType, targetID string) (int, error)
	GetByTarget(ctx context.Context, targetType, targetID string) (upvotes, downvotes int, err error)

	// Owner check
	GetTargetOwner(ctx context.Context, targetType, targetID string) (string, error)
}
