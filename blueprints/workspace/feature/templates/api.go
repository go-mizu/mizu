// Package templates provides template management.
package templates

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// Template represents a page template.
type Template struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Category    string    `json:"category,omitempty"`
	Preview     string    `json:"preview,omitempty"` // preview image URL
	PageID      string    `json:"page_id"`
	IsSystem    bool      `json:"is_system"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// API defines the templates service contract.
type API interface {
	// System templates
	ListSystemTemplates(ctx context.Context, category string) ([]*Template, error)
	GetSystemTemplate(ctx context.Context, id string) (*Template, error)

	// Workspace templates
	CreateTemplate(ctx context.Context, pageID string, name, category, workspaceID, userID string) (*Template, error)
	ListWorkspaceTemplates(ctx context.Context, workspaceID, category string) ([]*Template, error)
	DeleteTemplate(ctx context.Context, id string) error

	// Use template
	UseTemplate(ctx context.Context, templateID, targetParentID, userID string) (*pages.Page, error)
}

// Store defines the data access contract for templates.
type Store interface {
	Create(ctx context.Context, t *Template) error
	GetByID(ctx context.Context, id string) (*Template, error)
	Delete(ctx context.Context, id string) error
	ListSystem(ctx context.Context, category string) ([]*Template, error)
	ListByWorkspace(ctx context.Context, workspaceID, category string) ([]*Template, error)
}
