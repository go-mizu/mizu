// Package shares provides sharing and permissions functionality.
package shares

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound    = errors.New("share not found")
	ErrNotAllowed  = errors.New("not allowed")
	ErrTokenExpired = errors.New("share token expired")
)

// Share types
const (
	TypeUser  = "user"
	TypeEmail = "email"
	TypeLink  = "link"
)

// Permissions
const (
	PermRead    = "read"
	PermComment = "comment"
	PermEdit    = "edit"
	PermAdmin   = "admin"
)

// Share represents a share configuration.
type Share struct {
	ID         string     `json:"id"`
	BaseID     string     `json:"base_id"`
	TableID    string     `json:"table_id,omitempty"`
	ViewID     string     `json:"view_id,omitempty"`
	Type       string     `json:"type"`
	Permission string     `json:"permission"`
	UserID     string     `json:"user_id,omitempty"`
	Email      string     `json:"email,omitempty"`
	Token      string     `json:"token,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateIn contains input for creating a share.
type CreateIn struct {
	BaseID     string     `json:"base_id"`
	TableID    string     `json:"table_id,omitempty"`
	ViewID     string     `json:"view_id,omitempty"`
	Type       string     `json:"type"`
	Permission string     `json:"permission"`
	UserID     string     `json:"user_id,omitempty"`
	Email      string     `json:"email,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// UpdateIn contains input for updating a share.
type UpdateIn struct {
	Permission *string    `json:"permission,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// API defines the shares service interface.
type API interface {
	Create(ctx context.Context, userID string, in CreateIn) (*Share, error)
	GetByID(ctx context.Context, id string) (*Share, error)
	GetByToken(ctx context.Context, token string) (*Share, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Share, error)
	Delete(ctx context.Context, id string) error
	ListByBase(ctx context.Context, baseID string) ([]*Share, error)

	// Permissions
	CanAccess(ctx context.Context, userID string, baseID string, permission string) (bool, error)
	CanAccessTable(ctx context.Context, userID string, tableID string, permission string) (bool, error)
	CanAccessView(ctx context.Context, userID string, viewID string, permission string) (bool, error)
}

// Store defines the shares data access interface.
type Store interface {
	Create(ctx context.Context, share *Share) error
	GetByID(ctx context.Context, id string) (*Share, error)
	GetByToken(ctx context.Context, token string) (*Share, error)
	Update(ctx context.Context, share *Share) error
	Delete(ctx context.Context, id string) error
	ListByBase(ctx context.Context, baseID string) ([]*Share, error)
	ListByUser(ctx context.Context, userID string) ([]*Share, error)
}
