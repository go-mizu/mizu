package milestones

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound      = errors.New("milestone not found")
	ErrExists        = errors.New("milestone already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrMissingTitle  = errors.New("milestone title is required")
	ErrAlreadyClosed = errors.New("milestone is already closed")
	ErrAlreadyOpen   = errors.New("milestone is already open")
)

// Milestone represents a repository milestone
type Milestone struct {
	ID           string     `json:"id"`
	RepoID       string     `json:"repo_id"`
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"` // open, closed
	DueDate      *time.Time `json:"due_date,omitempty"`
	OpenIssues   int        `json:"open_issues"`
	ClosedIssues int        `json:"closed_issues"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

// CreateIn is the input for creating a milestone
type CreateIn struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// UpdateIn is the input for updating a milestone
type UpdateIn struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	State       *string    `json:"state,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// ListOpts are options for listing milestones
type ListOpts struct {
	State     string // open, closed, all
	Sort      string // due_on, completeness
	Direction string // asc, desc
}

// API is the milestones service interface
type API interface {
	Create(ctx context.Context, repoID string, in *CreateIn) (*Milestone, error)
	GetByID(ctx context.Context, id string) (*Milestone, error)
	GetByNumber(ctx context.Context, repoID string, number int) (*Milestone, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Milestone, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string, opts *ListOpts) ([]*Milestone, error)
	Close(ctx context.Context, id string) error
	Reopen(ctx context.Context, id string) error
	IncrementOpenIssues(ctx context.Context, id string) error
	DecrementOpenIssues(ctx context.Context, id string) error
	IncrementClosedIssues(ctx context.Context, id string) error
	DecrementClosedIssues(ctx context.Context, id string) error
}

// Store is the milestones data store interface
type Store interface {
	Create(ctx context.Context, m *Milestone) error
	GetByID(ctx context.Context, id string) (*Milestone, error)
	GetByNumber(ctx context.Context, repoID string, number int) (*Milestone, error)
	Update(ctx context.Context, m *Milestone) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string, state string) ([]*Milestone, error)
	GetNextNumber(ctx context.Context, repoID string) (int, error)
}
