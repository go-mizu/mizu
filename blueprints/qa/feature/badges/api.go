package badges

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("badge not found")
)

// Badge represents a badge.
type Badge struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Description string `json:"description"`
}

// Award represents a badge award.
type Award struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	BadgeID   string    `json:"badge_id"`
	CreatedAt time.Time `json:"created_at"`
}

// API defines the badges service interface.
type API interface {
	Create(ctx context.Context, in Badge) (*Badge, error)
	List(ctx context.Context, limit int) ([]*Badge, error)
	Award(ctx context.Context, accountID, badgeID string) (*Award, error)
	ListAwards(ctx context.Context, accountID string) ([]*Award, error)
}

// Store defines the data storage interface for badges.
type Store interface {
	Create(ctx context.Context, badge *Badge) error
	List(ctx context.Context, limit int) ([]*Badge, error)
	CreateAward(ctx context.Context, award *Award) error
	ListAwards(ctx context.Context, accountID string) ([]*Award, error)
}
