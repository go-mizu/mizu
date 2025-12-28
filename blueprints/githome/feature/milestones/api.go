package milestones

import (
	"context"
	"errors"
	"time"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound        = errors.New("milestone not found")
	ErrMilestoneExists = errors.New("milestone already exists")
)

// Milestone represents a GitHub milestone
type Milestone struct {
	ID           int64             `json:"id"`
	NodeID       string            `json:"node_id"`
	URL          string            `json:"url"`
	HTMLURL      string            `json:"html_url"`
	LabelsURL    string            `json:"labels_url"`
	Number       int               `json:"number"`
	State        string            `json:"state"` // open, closed
	Title        string            `json:"title"`
	Description  string            `json:"description,omitempty"`
	Creator      *users.SimpleUser `json:"creator"`
	OpenIssues   int               `json:"open_issues"`
	ClosedIssues int               `json:"closed_issues"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	ClosedAt     *time.Time        `json:"closed_at"`
	DueOn        *time.Time        `json:"due_on"`
	// Internal
	RepoID    int64 `json:"-"`
	CreatorID int64 `json:"-"`
}

// CreateIn represents the input for creating a milestone
type CreateIn struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	State       string     `json:"state,omitempty"` // open, closed
	DueOn       *time.Time `json:"due_on,omitempty"`
}

// UpdateIn represents the input for updating a milestone
type UpdateIn struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	State       *string    `json:"state,omitempty"`
	DueOn       *time.Time `json:"due_on,omitempty"`
}

// ListOpts contains options for listing milestones
type ListOpts struct {
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per_page,omitempty"`
	State     string `json:"state,omitempty"`     // open, closed, all
	Sort      string `json:"sort,omitempty"`      // due_on, completeness
	Direction string `json:"direction,omitempty"` // asc, desc
}

// API defines the milestones service interface
type API interface {
	// List returns milestones for a repository
	List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Milestone, error)

	// Get retrieves a milestone by number
	Get(ctx context.Context, owner, repo string, number int) (*Milestone, error)

	// GetByID retrieves a milestone by ID
	GetByID(ctx context.Context, id int64) (*Milestone, error)

	// Create creates a new milestone
	Create(ctx context.Context, owner, repo string, creatorID int64, in *CreateIn) (*Milestone, error)

	// Update updates a milestone
	Update(ctx context.Context, owner, repo string, number int, in *UpdateIn) (*Milestone, error)

	// Delete removes a milestone
	Delete(ctx context.Context, owner, repo string, number int) error

	// IncrementOpenIssues adjusts the open issues count
	IncrementOpenIssues(ctx context.Context, id int64, delta int) error

	// IncrementClosedIssues adjusts the closed issues count
	IncrementClosedIssues(ctx context.Context, id int64, delta int) error
}

// Store defines the data access interface for milestones
type Store interface {
	Create(ctx context.Context, m *Milestone) error
	GetByID(ctx context.Context, id int64) (*Milestone, error)
	GetByNumber(ctx context.Context, repoID int64, number int) (*Milestone, error)
	GetByTitle(ctx context.Context, repoID int64, title string) (*Milestone, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, repoID int64, opts *ListOpts) ([]*Milestone, error)
	NextNumber(ctx context.Context, repoID int64) (int, error)

	// Counter operations
	IncrementOpenIssues(ctx context.Context, id int64, delta int) error
	IncrementClosedIssues(ctx context.Context, id int64, delta int) error
}
