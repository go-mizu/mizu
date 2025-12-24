// Package servers provides server (guild/workspace) management.
package servers

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("server not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// Server represents a chat server (guild/workspace).
type Server struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	IconURL        string    `json:"icon_url,omitempty"`
	BannerURL      string    `json:"banner_url,omitempty"`
	OwnerID        string    `json:"owner_id"`
	IsPublic       bool      `json:"is_public"`
	IsVerified     bool      `json:"is_verified"`
	InviteCode     string    `json:"invite_code,omitempty"`
	DefaultChannel string    `json:"default_channel,omitempty"`
	MemberCount    int       `json:"member_count"`
	Features       []string  `json:"features,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a server.
type CreateIn struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IconURL     string `json:"icon_url,omitempty"`
	IsPublic    bool   `json:"is_public,omitempty"`
}

// UpdateIn contains input for updating a server.
type UpdateIn struct {
	Name           *string `json:"name,omitempty"`
	Description    *string `json:"description,omitempty"`
	IconURL        *string `json:"icon_url,omitempty"`
	BannerURL      *string `json:"banner_url,omitempty"`
	IsPublic       *bool   `json:"is_public,omitempty"`
	DefaultChannel *string `json:"default_channel,omitempty"`
}

// API defines the servers service contract.
type API interface {
	Create(ctx context.Context, ownerID string, in *CreateIn) (*Server, error)
	GetByID(ctx context.Context, id string) (*Server, error)
	GetByInviteCode(ctx context.Context, code string) (*Server, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Server, error)
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Server, error)
	ListPublic(ctx context.Context, limit, offset int) ([]*Server, error)
	Search(ctx context.Context, query string, limit int) ([]*Server, error)
	GenerateInviteCode(ctx context.Context, serverID string) (string, error)
	TransferOwnership(ctx context.Context, serverID, newOwnerID string) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, s *Server) error
	GetByID(ctx context.Context, id string) (*Server, error)
	GetByInviteCode(ctx context.Context, code string) (*Server, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Server, error)
	ListPublic(ctx context.Context, limit, offset int) ([]*Server, error)
	Search(ctx context.Context, query string, limit int) ([]*Server, error)
	UpdateMemberCount(ctx context.Context, serverID string, delta int) error
	SetDefaultChannel(ctx context.Context, serverID, channelID string) error
}
