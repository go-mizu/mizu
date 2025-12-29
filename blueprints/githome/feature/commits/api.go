package commits

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound = errors.New("commit not found")
)

// Commit represents a Git commit
type Commit struct {
	URL         string            `json:"url"`
	SHA         string            `json:"sha"`
	NodeID      string            `json:"node_id"`
	HTMLURL     string            `json:"html_url"`
	CommentsURL string            `json:"comments_url"`
	Commit      *CommitData       `json:"commit"`
	Author      *users.SimpleUser `json:"author,omitempty"`
	Committer   *users.SimpleUser `json:"committer,omitempty"`
	Parents     []*CommitRef      `json:"parents"`
	Stats       *CommitStats      `json:"stats,omitempty"`
	Files       []*CommitFile     `json:"files,omitempty"`
}

// CommitData contains the commit details
type CommitData struct {
	URL          string        `json:"url"`
	Author       *CommitAuthor `json:"author"`
	Committer    *CommitAuthor `json:"committer"`
	Message      string        `json:"message"`
	Tree         *TreeRef      `json:"tree"`
	CommentCount int           `json:"comment_count"`
	Verification *Verification `json:"verification,omitempty"`
}

// CommitAuthor represents a commit author
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// TreeRef represents a reference to a tree
type TreeRef struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// CommitRef is a minimal commit reference
type CommitRef struct {
	SHA     string `json:"sha"`
	URL     string `json:"url"`
	HTMLURL string `json:"html_url,omitempty"`
}

// CommitStats contains commit statistics
type CommitStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Total     int `json:"total"`
}

// CommitFile represents a file in a commit
type CommitFile struct {
	SHA              string `json:"sha"`
	Filename         string `json:"filename"`
	Status           string `json:"status"` // added, removed, modified, renamed, copied, changed, unchanged
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Changes          int    `json:"changes"`
	BlobURL          string `json:"blob_url"`
	RawURL           string `json:"raw_url"`
	ContentsURL      string `json:"contents_url"`
	Patch            string `json:"patch,omitempty"`
	PreviousFilename string `json:"previous_filename,omitempty"`
}

// Verification represents signature verification
type Verification struct {
	Verified   bool    `json:"verified"`
	Reason     string  `json:"reason"`
	Signature  *string `json:"signature"`
	Payload    *string `json:"payload"`
	VerifiedAt *string `json:"verified_at"`
}

// Comparison represents a commit comparison
type Comparison struct {
	URL             string        `json:"url"`
	HTMLURL         string        `json:"html_url"`
	PermalinkURL    string        `json:"permalink_url"`
	DiffURL         string        `json:"diff_url"`
	PatchURL        string        `json:"patch_url"`
	BaseCommit      *Commit       `json:"base_commit"`
	MergeBaseCommit *Commit       `json:"merge_base_commit"`
	Status          string        `json:"status"` // diverged, ahead, behind, identical
	AheadBy         int           `json:"ahead_by"`
	BehindBy        int           `json:"behind_by"`
	TotalCommits    int           `json:"total_commits"`
	Commits         []*Commit     `json:"commits"`
	Files           []*CommitFile `json:"files"`
}

// Branch represents a branch containing a commit
type Branch struct {
	Name      string `json:"name"`
	Commit    *CommitRef `json:"commit"`
	Protected bool   `json:"protected"`
}

// CombinedStatus represents combined status for a ref
type CombinedStatus struct {
	State      string            `json:"state"` // pending, success, failure, error
	Statuses   []*Status         `json:"statuses"`
	SHA        string            `json:"sha"`
	TotalCount int               `json:"total_count"`
	Repository *Repository       `json:"repository"`
	CommitURL  string            `json:"commit_url"`
	URL        string            `json:"url"`
}

// Status represents a commit status
type Status struct {
	ID          int64     `json:"id"`
	NodeID      string    `json:"node_id"`
	URL         string    `json:"url"`
	State       string    `json:"state"` // pending, success, failure, error
	Description string    `json:"description,omitempty"`
	TargetURL   string    `json:"target_url,omitempty"`
	Context     string    `json:"context"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Creator     *users.SimpleUser `json:"creator"`
}

// Repository is a minimal repo reference
type Repository struct {
	ID       int64  `json:"id"`
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
	URL      string `json:"url"`
}

// CreateStatusIn represents input for creating a status
type CreateStatusIn struct {
	State       string `json:"state"` // pending, success, failure, error
	TargetURL   string `json:"target_url,omitempty"`
	Description string `json:"description,omitempty"`
	Context     string `json:"context,omitempty"`
}

// ListOpts contains options for listing commits
type ListOpts struct {
	Page      int       `json:"page,omitempty"`
	PerPage   int       `json:"per_page,omitempty"`
	SHA       string    `json:"sha,omitempty"`
	Path      string    `json:"path,omitempty"`
	Author    string    `json:"author,omitempty"`
	Committer string    `json:"committer,omitempty"`
	Since     time.Time `json:"since,omitempty"`
	Until     time.Time `json:"until,omitempty"`
}

// API defines the commits service interface
type API interface {
	// List returns commits for a repository
	List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Commit, error)

	// Get retrieves a commit by ref
	Get(ctx context.Context, owner, repo, ref string) (*Commit, error)

	// Compare compares two commits
	Compare(ctx context.Context, owner, repo, base, head string) (*Comparison, error)

	// ListBranchesForHead returns branches containing the commit
	ListBranchesForHead(ctx context.Context, owner, repo, sha string) ([]*Branch, error)

	// ListPullsForCommit returns PRs associated with a commit
	ListPullsForCommit(ctx context.Context, owner, repo, sha string, opts *ListOpts) ([]*PullRequest, error)

	// GetCombinedStatus returns combined status for a ref
	GetCombinedStatus(ctx context.Context, owner, repo, ref string) (*CombinedStatus, error)

	// ListStatuses returns statuses for a ref
	ListStatuses(ctx context.Context, owner, repo, ref string, opts *ListOpts) ([]*Status, error)

	// CreateStatus creates a status for a SHA
	CreateStatus(ctx context.Context, owner, repo, sha string, creatorID int64, in *CreateStatusIn) (*Status, error)
}

// PullRequest is a minimal PR reference
type PullRequest struct {
	ID      int64  `json:"id"`
	NodeID  string `json:"node_id"`
	URL     string `json:"url"`
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
	State   string `json:"state"`
	Title   string `json:"title"`
}

// Store defines the data access interface for commits
type Store interface {
	// Statuses
	CreateStatus(ctx context.Context, repoID int64, sha string, s *Status) error
	GetStatusByID(ctx context.Context, id int64) (*Status, error)
	ListStatuses(ctx context.Context, repoID int64, sha string, opts *ListOpts) ([]*Status, error)
	GetCombinedStatus(ctx context.Context, repoID int64, sha string) (*CombinedStatus, error)
}
