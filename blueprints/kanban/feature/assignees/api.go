// Package assignees provides issue assignee management functionality.
package assignees

import (
	"context"
)

// API defines the assignees service contract.
type API interface {
	Add(ctx context.Context, issueID, userID string) error
	Remove(ctx context.Context, issueID, userID string) error
	List(ctx context.Context, issueID string) ([]string, error)
	ListByUser(ctx context.Context, userID string) ([]string, error)
}

// Store defines the data access contract for assignees.
type Store interface {
	Add(ctx context.Context, issueID, userID string) error
	Remove(ctx context.Context, issueID, userID string) error
	List(ctx context.Context, issueID string) ([]string, error)
	ListByUser(ctx context.Context, userID string) ([]string, error)
}
