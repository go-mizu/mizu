// Package history provides version history and activity tracking.
package history

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// ActionType represents the type of action.
type ActionType string

const (
	ActionCreate       ActionType = "create"
	ActionUpdate       ActionType = "update"
	ActionDelete       ActionType = "delete"
	ActionRestore      ActionType = "restore"
	ActionMove         ActionType = "move"
	ActionShare        ActionType = "share"
	ActionComment      ActionType = "comment"
	ActionResolve      ActionType = "resolve"
	ActionAddMember    ActionType = "add_member"
	ActionRemoveMember ActionType = "remove_member"
)

// Revision represents a saved version of a page.
type Revision struct {
	ID         string           `json:"id"`
	PageID     string           `json:"page_id"`
	Version    int              `json:"version"`
	Title      string           `json:"title"`
	Blocks     []*blocks.Block  `json:"blocks,omitempty"`
	Properties pages.Properties `json:"properties,omitempty"`
	AuthorID   string           `json:"author_id"`
	CreatedAt  time.Time        `json:"created_at"`

	// Enriched
	Author *users.User `json:"author,omitempty"`
}

// Activity represents an action in the workspace.
type Activity struct {
	ID          string      `json:"id"`
	WorkspaceID string      `json:"workspace_id"`
	PageID      string      `json:"page_id,omitempty"`
	BlockID     string      `json:"block_id,omitempty"`
	ActorID     string      `json:"actor_id"`
	Action      ActionType  `json:"action"`
	Details     interface{} `json:"details,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`

	// Enriched
	Actor *users.User    `json:"actor,omitempty"`
	Page  *pages.PageRef `json:"page,omitempty"`
}

// ActivityOpts contains options for listing activities.
type ActivityOpts struct {
	Limit  int
	Cursor string
}

// Diff represents differences between two revisions.
type Diff struct {
	Added   []*blocks.Block `json:"added,omitempty"`
	Removed []*blocks.Block `json:"removed,omitempty"`
	Changed []Change        `json:"changed,omitempty"`
}

// Change represents a single change.
type Change struct {
	BlockID string        `json:"block_id"`
	Before  *blocks.Block `json:"before"`
	After   *blocks.Block `json:"after"`
}

// API defines the history service contract.
type API interface {
	// Revisions
	CreateRevision(ctx context.Context, pageID, authorID string) (*Revision, error)
	GetRevision(ctx context.Context, id string) (*Revision, error)
	ListRevisions(ctx context.Context, pageID string, limit int) ([]*Revision, error)
	RestoreRevision(ctx context.Context, pageID, revisionID, userID string) error
	CompareRevisions(ctx context.Context, revID1, revID2 string) (*Diff, error)

	// Activity
	RecordActivity(ctx context.Context, workspaceID, pageID, blockID, actorID string, action ActionType, details interface{}) error
	ListByWorkspace(ctx context.Context, workspaceID string, opts ActivityOpts) ([]*Activity, error)
	ListByPage(ctx context.Context, pageID string, opts ActivityOpts) ([]*Activity, error)
	ListByUser(ctx context.Context, userID string, opts ActivityOpts) ([]*Activity, error)
}

// Store defines the data access contract for history.
type Store interface {
	// Revisions
	CreateRevision(ctx context.Context, r *Revision) error
	GetRevision(ctx context.Context, id string) (*Revision, error)
	ListRevisions(ctx context.Context, pageID string, limit int) ([]*Revision, error)
	GetLatestVersion(ctx context.Context, pageID string) (int, error)

	// Activity
	CreateActivity(ctx context.Context, a *Activity) error
	ListByWorkspace(ctx context.Context, workspaceID string, opts ActivityOpts) ([]*Activity, error)
	ListByPage(ctx context.Context, pageID string, opts ActivityOpts) ([]*Activity, error)
	ListByUser(ctx context.Context, userID string, opts ActivityOpts) ([]*Activity, error)
}
