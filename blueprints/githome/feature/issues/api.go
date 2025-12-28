package issues

import (
	"context"
	"errors"
	"time"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound     = errors.New("issue not found")
	ErrAccessDenied = errors.New("access denied")
	ErrLocked       = errors.New("issue is locked")
)

// Issue represents a GitHub issue
type Issue struct {
	ID                int64             `json:"id"`
	NodeID            string            `json:"node_id"`
	URL               string            `json:"url"`
	RepositoryURL     string            `json:"repository_url"`
	LabelsURL         string            `json:"labels_url"`
	CommentsURL       string            `json:"comments_url"`
	EventsURL         string            `json:"events_url"`
	HTMLURL           string            `json:"html_url"`
	Number            int               `json:"number"`
	State             string            `json:"state"` // open, closed
	StateReason       string            `json:"state_reason,omitempty"` // completed, reopened, not_planned
	Title             string            `json:"title"`
	Body              string            `json:"body,omitempty"`
	User              *users.SimpleUser `json:"user"`
	Labels            []*Label          `json:"labels"`
	Assignee          *users.SimpleUser `json:"assignee,omitempty"`
	Assignees         []*users.SimpleUser `json:"assignees"`
	Milestone         *Milestone        `json:"milestone,omitempty"`
	Locked            bool              `json:"locked"`
	ActiveLockReason  string            `json:"active_lock_reason,omitempty"`
	Comments          int               `json:"comments"`
	PullRequest       *IssuePR          `json:"pull_request,omitempty"`
	ClosedAt          *time.Time        `json:"closed_at"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	ClosedBy          *users.SimpleUser `json:"closed_by,omitempty"`
	AuthorAssociation string            `json:"author_association"`
	Reactions         *Reactions        `json:"reactions,omitempty"`
	// Internal fields
	RepoID    int64  `json:"-"`
	CreatorID int64  `json:"-"`
}

// Label represents a GitHub label (lightweight for issues)
type Label struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

// Milestone represents a GitHub milestone (lightweight for issues)
type Milestone struct {
	ID           int64             `json:"id"`
	NodeID       string            `json:"node_id"`
	URL          string            `json:"url"`
	HTMLURL      string            `json:"html_url"`
	LabelsURL    string            `json:"labels_url"`
	Number       int               `json:"number"`
	State        string            `json:"state"`
	Title        string            `json:"title"`
	Description  string            `json:"description,omitempty"`
	Creator      *users.SimpleUser `json:"creator"`
	OpenIssues   int               `json:"open_issues"`
	ClosedIssues int               `json:"closed_issues"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	ClosedAt     *time.Time        `json:"closed_at"`
	DueOn        *time.Time        `json:"due_on"`
}

// IssuePR contains PR info if issue is associated with a PR
type IssuePR struct {
	URL      string     `json:"url"`
	HTMLURL  string     `json:"html_url"`
	DiffURL  string     `json:"diff_url"`
	PatchURL string     `json:"patch_url"`
	MergedAt *time.Time `json:"merged_at,omitempty"`
}

// Reactions represents reaction counts
type Reactions struct {
	URL        string `json:"url"`
	TotalCount int    `json:"total_count"`
	PlusOne    int    `json:"+1"`
	MinusOne   int    `json:"-1"`
	Laugh      int    `json:"laugh"`
	Confused   int    `json:"confused"`
	Heart      int    `json:"heart"`
	Hooray     int    `json:"hooray"`
	Rocket     int    `json:"rocket"`
	Eyes       int    `json:"eyes"`
}

// IssueEvent represents an issue event
type IssueEvent struct {
	ID             int64             `json:"id"`
	NodeID         string            `json:"node_id"`
	URL            string            `json:"url"`
	Actor          *users.SimpleUser `json:"actor"`
	Event          string            `json:"event"`
	CommitID       string            `json:"commit_id,omitempty"`
	CommitURL      string            `json:"commit_url,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	Label          *Label            `json:"label,omitempty"`
	Assignee       *users.SimpleUser `json:"assignee,omitempty"`
	Assigner       *users.SimpleUser `json:"assigner,omitempty"`
	Milestone      *Milestone        `json:"milestone,omitempty"`
	Rename         *Rename           `json:"rename,omitempty"`
}

// Rename represents a rename event
type Rename struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// CreateIn represents the input for creating an issue
type CreateIn struct {
	Title     string   `json:"title"`
	Body      string   `json:"body,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Milestone *int     `json:"milestone,omitempty"`
	Labels    []string `json:"labels,omitempty"`
}

// UpdateIn represents the input for updating an issue
type UpdateIn struct {
	Title       *string  `json:"title,omitempty"`
	Body        *string  `json:"body,omitempty"`
	State       *string  `json:"state,omitempty"`
	StateReason *string  `json:"state_reason,omitempty"`
	Assignee    *string  `json:"assignee,omitempty"`
	Assignees   []string `json:"assignees,omitempty"`
	Milestone   *int     `json:"milestone,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

// ListOpts contains options for listing issues
type ListOpts struct {
	Page      int       `json:"page,omitempty"`
	PerPage   int       `json:"per_page,omitempty"`
	State     string    `json:"state,omitempty"`     // open, closed, all
	Sort      string    `json:"sort,omitempty"`      // created, updated, comments
	Direction string    `json:"direction,omitempty"` // asc, desc
	Since     time.Time `json:"since,omitempty"`
	Labels    string    `json:"labels,omitempty"`
	Milestone string    `json:"milestone,omitempty"` // number, none, *
	Assignee  string    `json:"assignee,omitempty"`
	Creator   string    `json:"creator,omitempty"`
	Mentioned string    `json:"mentioned,omitempty"`
}

// API defines the issues service interface
type API interface {
	// Create creates a new issue
	Create(ctx context.Context, owner, repo string, creatorID int64, in *CreateIn) (*Issue, error)

	// Get retrieves an issue by number
	Get(ctx context.Context, owner, repo string, number int) (*Issue, error)

	// GetByID retrieves an issue by ID
	GetByID(ctx context.Context, id int64) (*Issue, error)

	// Update updates an issue
	Update(ctx context.Context, owner, repo string, number int, in *UpdateIn) (*Issue, error)

	// Lock locks an issue
	Lock(ctx context.Context, owner, repo string, number int, reason string) error

	// Unlock unlocks an issue
	Unlock(ctx context.Context, owner, repo string, number int) error

	// ListForRepo returns issues for a repository
	ListForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Issue, error)

	// ListForOrg returns issues for an organization
	ListForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Issue, error)

	// ListForUser returns issues assigned to/created by the authenticated user
	ListForUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Issue, error)

	// ListAssignees returns users who can be assigned to issues
	ListAssignees(ctx context.Context, owner, repo string) ([]*users.SimpleUser, error)

	// CheckAssignee checks if a user can be assigned
	CheckAssignee(ctx context.Context, owner, repo, assignee string) (bool, error)

	// AddAssignees adds assignees to an issue
	AddAssignees(ctx context.Context, owner, repo string, number int, assignees []string) (*Issue, error)

	// RemoveAssignees removes assignees from an issue
	RemoveAssignees(ctx context.Context, owner, repo string, number int, assignees []string) (*Issue, error)

	// ListEvents returns events for an issue
	ListEvents(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*IssueEvent, error)

	// CreateEvent creates an event (internal use)
	CreateEvent(ctx context.Context, issueID, actorID int64, eventType string, data map[string]interface{}) (*IssueEvent, error)
}

// Store defines the data access interface for issues
type Store interface {
	Create(ctx context.Context, issue *Issue) error
	GetByID(ctx context.Context, id int64) (*Issue, error)
	GetByNumber(ctx context.Context, repoID int64, number int) (*Issue, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, repoID int64, opts *ListOpts) ([]*Issue, error)
	ListForOrg(ctx context.Context, orgID int64, opts *ListOpts) ([]*Issue, error)
	ListForUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Issue, error)
	NextNumber(ctx context.Context, repoID int64) (int, error)

	// Lock/unlock
	SetLocked(ctx context.Context, id int64, locked bool, reason string) error

	// Assignees
	AddAssignee(ctx context.Context, issueID, userID int64) error
	RemoveAssignee(ctx context.Context, issueID, userID int64) error
	ListAssignees(ctx context.Context, issueID int64) ([]*users.SimpleUser, error)

	// Labels
	AddLabel(ctx context.Context, issueID, labelID int64) error
	RemoveLabel(ctx context.Context, issueID, labelID int64) error
	ListLabels(ctx context.Context, issueID int64) ([]*Label, error)
	SetLabels(ctx context.Context, issueID int64, labelIDs []int64) error

	// Milestone
	SetMilestone(ctx context.Context, issueID int64, milestoneID *int64) error

	// Events
	CreateEvent(ctx context.Context, event *IssueEvent) error
	ListEvents(ctx context.Context, issueID int64, opts *ListOpts) ([]*IssueEvent, error)

	// Comments count
	IncrementComments(ctx context.Context, issueID int64, delta int) error
}
