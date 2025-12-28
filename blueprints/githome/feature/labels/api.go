package labels

import (
	"context"
	"errors"
)

var (
	ErrNotFound    = errors.New("label not found")
	ErrLabelExists = errors.New("label already exists")
)

// Label represents a GitHub label
type Label struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color"` // 6-char hex without #
	Default     bool   `json:"default"`
	// Internal
	RepoID int64 `json:"-"`
}

// CreateIn represents the input for creating a label
type CreateIn struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color"` // 6-char hex without #
}

// UpdateIn represents the input for updating a label
type UpdateIn struct {
	NewName     *string `json:"new_name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
}

// API defines the labels service interface
type API interface {
	// List returns labels for a repository
	List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Label, error)

	// Get retrieves a label by name
	Get(ctx context.Context, owner, repo, name string) (*Label, error)

	// GetByID retrieves a label by ID
	GetByID(ctx context.Context, id int64) (*Label, error)

	// Create creates a new label
	Create(ctx context.Context, owner, repo string, in *CreateIn) (*Label, error)

	// Update updates a label
	Update(ctx context.Context, owner, repo, name string, in *UpdateIn) (*Label, error)

	// Delete removes a label
	Delete(ctx context.Context, owner, repo, name string) error

	// ListForIssue returns labels for an issue
	ListForIssue(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Label, error)

	// AddToIssue adds labels to an issue
	AddToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*Label, error)

	// SetForIssue replaces all labels on an issue
	SetForIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*Label, error)

	// RemoveFromIssue removes a label from an issue
	RemoveFromIssue(ctx context.Context, owner, repo string, number int, name string) error

	// RemoveAllFromIssue removes all labels from an issue
	RemoveAllFromIssue(ctx context.Context, owner, repo string, number int) error

	// ListForMilestone returns labels for issues in a milestone
	ListForMilestone(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Label, error)
}

// Store defines the data access interface for labels
type Store interface {
	Create(ctx context.Context, label *Label) error
	GetByID(ctx context.Context, id int64) (*Label, error)
	GetByName(ctx context.Context, repoID int64, name string) (*Label, error)
	GetByNames(ctx context.Context, repoID int64, names []string) ([]*Label, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, repoID int64, opts *ListOpts) ([]*Label, error)

	// Issue-label relationships
	ListForIssue(ctx context.Context, issueID int64, opts *ListOpts) ([]*Label, error)
	AddToIssue(ctx context.Context, issueID, labelID int64) error
	RemoveFromIssue(ctx context.Context, issueID, labelID int64) error
	SetForIssue(ctx context.Context, issueID int64, labelIDs []int64) error
	RemoveAllFromIssue(ctx context.Context, issueID int64) error

	// Milestone labels (aggregate)
	ListForMilestone(ctx context.Context, milestoneID int64, opts *ListOpts) ([]*Label, error)
}
