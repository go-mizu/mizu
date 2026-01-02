package votes

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidVote = errors.New("invalid vote")
)

// TargetType defines vote target.
type TargetType string

const (
	TargetQuestion TargetType = "question"
	TargetAnswer   TargetType = "answer"
	TargetComment  TargetType = "comment"
)

// Vote represents a vote.
type Vote struct {
	ID         string     `json:"id"`
	VoterID    string     `json:"voter_id"`
	TargetType TargetType `json:"target_type"`
	TargetID   string     `json:"target_id"`
	Value      int        `json:"value"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// API defines the votes service interface.
type API interface {
	Cast(ctx context.Context, voterID string, targetType TargetType, targetID string, value int) (*Vote, error)
	Get(ctx context.Context, voterID string, targetType TargetType, targetID string) (*Vote, error)
}

// Store defines the data storage interface for votes.
type Store interface {
	Upsert(ctx context.Context, vote *Vote) (*Vote, error)
	Get(ctx context.Context, voterID string, targetType TargetType, targetID string) (*Vote, error)
}
