// Package pages provides page management functionality.
package pages

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// ParentType indicates the type of parent for a page.
type ParentType string

const (
	ParentWorkspace ParentType = "workspace"
	ParentPage      ParentType = "page"
	ParentDatabase  ParentType = "database"
)

// Page represents a document page.
// When DatabaseID is set, this page acts as a database row.
type Page struct {
	ID          string      `json:"id"`
	WorkspaceID string      `json:"workspace_id"`
	ParentID    string      `json:"parent_id,omitempty"`
	ParentType  ParentType  `json:"parent_type"`
	DatabaseID  string      `json:"database_id,omitempty"`  // When set, this page is a database row
	RowPosition int64       `json:"row_position,omitempty"` // Ordering within a database
	Title       string      `json:"title"`
	Icon        string      `json:"icon,omitempty"`
	Cover       string      `json:"cover,omitempty"`
	CoverY      float64     `json:"cover_y"`
	Properties  Properties  `json:"properties,omitempty"`
	IsTemplate  bool        `json:"is_template"`
	IsArchived  bool        `json:"is_archived"`
	IsFavorite  bool        `json:"is_favorite,omitempty"` // Per-user computed
	CreatedBy   string      `json:"created_by"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedBy   string      `json:"updated_by"`
	UpdatedAt   time.Time   `json:"updated_at"`

	// Enriched fields
	Children   []*Page     `json:"children,omitempty"`
	Breadcrumb []*PageRef  `json:"breadcrumb,omitempty"`
	Author     *users.User `json:"author,omitempty"`
}

// PageRef is a lightweight page reference.
type PageRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon,omitempty"`
}

// Properties holds database item properties.
type Properties map[string]PropertyValue

// PropertyValue represents a property value.
type PropertyValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// CreateIn contains input for creating a page.
type CreateIn struct {
	WorkspaceID string     `json:"workspace_id"`
	ParentID    string     `json:"parent_id,omitempty"`
	ParentType  ParentType `json:"parent_type"`
	DatabaseID  string     `json:"database_id,omitempty"` // When set, creates a database row
	Title       string     `json:"title"`
	Icon        string     `json:"icon,omitempty"`
	Cover       string     `json:"cover,omitempty"`
	Properties  Properties `json:"properties,omitempty"`
	IsTemplate  bool       `json:"is_template"`
	CreatedBy   string     `json:"-"`
}

// UpdateIn contains input for updating a page.
type UpdateIn struct {
	Title      *string     `json:"title,omitempty"`
	Icon       *string     `json:"icon,omitempty"`
	Cover      *string     `json:"cover,omitempty"`
	CoverY     *float64    `json:"cover_y,omitempty"`
	Properties *Properties `json:"properties,omitempty"`
	UpdatedBy  string      `json:"-"`
}

// ListOpts contains options for listing pages.
type ListOpts struct {
	IncludeArchived bool
	Limit           int
	Cursor          string
}

// SearchOpts contains options for searching pages.
type SearchOpts struct {
	Types    []string
	Cursor   string
	Limit    int
}

// API defines the pages service contract.
type API interface {
	// CRUD
	Create(ctx context.Context, in *CreateIn) (*Page, error)
	GetByID(ctx context.Context, id string) (*Page, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Page, error)
	Delete(ctx context.Context, id string) error

	// Navigation
	ListByWorkspace(ctx context.Context, workspaceID string, opts ListOpts) ([]*Page, error)
	ListByParent(ctx context.Context, parentID string, parentType ParentType) ([]*Page, error)
	GetBreadcrumb(ctx context.Context, id string) ([]*PageRef, error)
	Move(ctx context.Context, id, newParentID string, newParentType ParentType) error
	Duplicate(ctx context.Context, id string, targetParentID string, userID string) (*Page, error)

	// Archive
	Archive(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	ListArchived(ctx context.Context, workspaceID string) ([]*Page, error)

	// Search
	Search(ctx context.Context, workspaceID, query string, opts SearchOpts) ([]*Page, error)
	GetRecent(ctx context.Context, userID, workspaceID string, limit int) ([]*Page, error)

	// Authorization
	CanAccess(ctx context.Context, pageID, userID string) (bool, error)
}

// Store defines the data access contract for pages.
type Store interface {
	Create(ctx context.Context, p *Page) error
	GetByID(ctx context.Context, id string) (*Page, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	ListByWorkspace(ctx context.Context, workspaceID string, opts ListOpts) ([]*Page, error)
	ListByParent(ctx context.Context, parentID string, parentType ParentType) ([]*Page, error)
	ListByParentIDs(ctx context.Context, parentIDs []string, parentType ParentType) (map[string][]*Page, error)
	Archive(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	ListArchived(ctx context.Context, workspaceID string) ([]*Page, error)
	Move(ctx context.Context, id, newParentID string, newParentType ParentType) error
	Search(ctx context.Context, workspaceID, query string, opts SearchOpts) ([]*Page, error)
	GetRecent(ctx context.Context, userID, workspaceID string, limit int) ([]*Page, error)

	// Database row operations (pages with database_id set)
	ListByDatabase(ctx context.Context, databaseID string, limit int, cursor string) ([]*Page, error)
	CountByDatabase(ctx context.Context, databaseID string) (int, error)
	DeleteByDatabase(ctx context.Context, databaseID string) error
	GetMaxRowPosition(ctx context.Context, databaseID string) (int64, error)
}
