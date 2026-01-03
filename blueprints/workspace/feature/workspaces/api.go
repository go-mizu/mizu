// Package workspaces provides workspace management functionality.
package workspaces

import (
	"context"
	"time"
)

// Plan represents a workspace plan level.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanTeam       Plan = "team"
	PlanEnterprise Plan = "enterprise"
)

// Workspace represents a workspace (tenant).
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Icon      string    `json:"icon,omitempty"`
	Domain    string    `json:"domain,omitempty"`
	Plan      Plan      `json:"plan"`
	Settings  Settings  `json:"settings"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Settings holds workspace configuration.
type Settings struct {
	AllowPublicPages  bool     `json:"allow_public_pages"`
	AllowGuestInvites bool     `json:"allow_guest_invites"`
	DefaultPermission string   `json:"default_permission"`
	AllowedDomains    []string `json:"allowed_domains"`
	ExportEnabled     bool     `json:"export_enabled"`
}

// CreateIn contains input for creating a workspace.
type CreateIn struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Icon string `json:"icon,omitempty"`
}

// UpdateIn contains input for updating a workspace.
type UpdateIn struct {
	Name *string `json:"name,omitempty"`
	Icon *string `json:"icon,omitempty"`
}

// API defines the workspaces service contract.
type API interface {
	Create(ctx context.Context, ownerID string, in *CreateIn) (*Workspace, error)
	GetByID(ctx context.Context, id string) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	ListByUser(ctx context.Context, userID string) ([]*Workspace, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Workspace, error)
	UpdateSettings(ctx context.Context, id string, settings Settings) error
	Delete(ctx context.Context, id string) error
	Transfer(ctx context.Context, id, newOwnerID string) error
}

// Store defines the data access contract for workspaces.
type Store interface {
	Create(ctx context.Context, ws *Workspace) error
	GetByID(ctx context.Context, id string) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	ListByUser(ctx context.Context, userID string) ([]*Workspace, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdateSettings(ctx context.Context, id string, settings Settings) error
	UpdateOwner(ctx context.Context, id, ownerID string) error
	Delete(ctx context.Context, id string) error
}
