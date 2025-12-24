// Package lists provides user list management functionality.
package lists

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/social/feature/accounts"
)

// Errors
var (
	ErrNotFound      = errors.New("list not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrAlreadyMember = errors.New("already a member")
	ErrNotMember     = errors.New("not a member")
)

// List represents a curated list of accounts.
type List struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Computed
	MemberCount int `json:"member_count,omitempty"`
}

// ListMember represents a member of a list.
type ListMember struct {
	ListID    string    `json:"list_id"`
	AccountID string    `json:"account_id"`
	CreatedAt time.Time `json:"created_at"`

	// Enriched
	Account *accounts.Account `json:"account,omitempty"`
}

// CreateIn contains input for creating a list.
type CreateIn struct {
	Title string `json:"title"`
}

// UpdateIn contains input for updating a list.
type UpdateIn struct {
	Title *string `json:"title,omitempty"`
}

// API defines the lists service contract.
type API interface {
	Create(ctx context.Context, accountID string, in *CreateIn) (*List, error)
	GetByID(ctx context.Context, id string) (*List, error)
	GetByAccount(ctx context.Context, accountID string) ([]*List, error)
	Update(ctx context.Context, accountID, listID string, in *UpdateIn) (*List, error)
	Delete(ctx context.Context, accountID, listID string) error

	GetMembers(ctx context.Context, listID string, limit, offset int) ([]*ListMember, error)
	AddMember(ctx context.Context, accountID, listID, memberID string) error
	RemoveMember(ctx context.Context, accountID, listID, memberID string) error
	GetListsContaining(ctx context.Context, accountID, targetID string) ([]*List, error)
}

// Store defines the data access contract for lists.
type Store interface {
	Insert(ctx context.Context, l *List) error
	GetByID(ctx context.Context, id string) (*List, error)
	GetByAccount(ctx context.Context, accountID string) ([]*List, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error

	InsertMember(ctx context.Context, m *ListMember) error
	DeleteMember(ctx context.Context, listID, accountID string) error
	GetMembers(ctx context.Context, listID string, limit, offset int) ([]*ListMember, error)
	ExistsMember(ctx context.Context, listID, accountID string) (bool, error)
	GetMemberCount(ctx context.Context, listID string) (int, error)
	GetListsContaining(ctx context.Context, targetID string) ([]*List, error)
}
