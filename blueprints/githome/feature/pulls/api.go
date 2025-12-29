package pulls

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound       = errors.New("pull request not found")
	ErrAccessDenied   = errors.New("access denied")
	ErrNotMergeable   = errors.New("pull request is not mergeable")
	ErrAlreadyMerged  = errors.New("pull request is already merged")
)

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID                  int64             `json:"id"`
	NodeID              string            `json:"node_id"`
	URL                 string            `json:"url"`
	HTMLURL             string            `json:"html_url"`
	DiffURL             string            `json:"diff_url"`
	PatchURL            string            `json:"patch_url"`
	IssueURL            string            `json:"issue_url"`
	CommitsURL          string            `json:"commits_url"`
	ReviewCommentsURL   string            `json:"review_comments_url"`
	ReviewCommentURL    string            `json:"review_comment_url"`
	CommentsURL         string            `json:"comments_url"`
	StatusesURL         string            `json:"statuses_url"`
	Number              int               `json:"number"`
	State               string            `json:"state"` // open, closed
	Locked              bool              `json:"locked"`
	Title               string            `json:"title"`
	User                *users.SimpleUser `json:"user"`
	Body                string            `json:"body,omitempty"`
	Labels              []*Label          `json:"labels"`
	Milestone           *Milestone        `json:"milestone,omitempty"`
	ActiveLockReason    string            `json:"active_lock_reason,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	ClosedAt            *time.Time        `json:"closed_at"`
	MergedAt            *time.Time        `json:"merged_at"`
	MergeCommitSHA      string            `json:"merge_commit_sha,omitempty"`
	Assignee            *users.SimpleUser `json:"assignee,omitempty"`
	Assignees           []*users.SimpleUser `json:"assignees"`
	RequestedReviewers  []*users.SimpleUser `json:"requested_reviewers"`
	RequestedTeams      []*TeamSimple     `json:"requested_teams"`
	Head                *PRBranch         `json:"head"`
	Base                *PRBranch         `json:"base"`
	Draft               bool              `json:"draft"`
	Merged              bool              `json:"merged"`
	Mergeable           *bool             `json:"mergeable"`
	Rebaseable          *bool             `json:"rebaseable"`
	MergeableState      string            `json:"mergeable_state"`
	MergedBy            *users.SimpleUser `json:"merged_by,omitempty"`
	Comments            int               `json:"comments"`
	ReviewComments      int               `json:"review_comments"`
	MaintainerCanModify bool              `json:"maintainer_can_modify"`
	Commits             int               `json:"commits"`
	Additions           int               `json:"additions"`
	Deletions           int               `json:"deletions"`
	ChangedFiles        int               `json:"changed_files"`
	AuthorAssociation   string            `json:"author_association"`
	// Internal
	RepoID    int64 `json:"-"`
	CreatorID int64 `json:"-"`
}

// PRBranch represents head/base branch info
type PRBranch struct {
	Label string            `json:"label"`
	Ref   string            `json:"ref"`
	SHA   string            `json:"sha"`
	User  *users.SimpleUser `json:"user"`
	Repo  *RepoRef          `json:"repo"`
}

// RepoRef is a minimal repository reference
type RepoRef struct {
	ID       int64  `json:"id"`
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

// Label is a minimal label reference
type Label struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

// Milestone is a minimal milestone reference
type Milestone struct {
	ID     int64  `json:"id"`
	NodeID string `json:"node_id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

// TeamSimple is a minimal team reference
type TeamSimple struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
}

// Review represents a PR review
type Review struct {
	ID                int64             `json:"id"`
	NodeID            string            `json:"node_id"`
	User              *users.SimpleUser `json:"user"`
	Body              string            `json:"body,omitempty"`
	State             string            `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING, DISMISSED
	HTMLURL           string            `json:"html_url"`
	PullRequestURL    string            `json:"pull_request_url"`
	CommitID          string            `json:"commit_id"`
	SubmittedAt       time.Time         `json:"submitted_at"`
	AuthorAssociation string            `json:"author_association"`
	// Internal
	PRID   int64 `json:"-"`
	UserID int64 `json:"-"`
}

// ReviewComment represents a PR review comment
type ReviewComment struct {
	ID                  int64             `json:"id"`
	NodeID              string            `json:"node_id"`
	URL                 string            `json:"url"`
	PullRequestReviewID int64             `json:"pull_request_review_id"`
	DiffHunk            string            `json:"diff_hunk"`
	Path                string            `json:"path"`
	Position            int               `json:"position,omitempty"`
	OriginalPosition    int               `json:"original_position,omitempty"`
	CommitID            string            `json:"commit_id"`
	OriginalCommitID    string            `json:"original_commit_id"`
	InReplyToID         int64             `json:"in_reply_to_id,omitempty"`
	User                *users.SimpleUser `json:"user"`
	Body                string            `json:"body"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	HTMLURL             string            `json:"html_url"`
	PullRequestURL      string            `json:"pull_request_url"`
	AuthorAssociation   string            `json:"author_association"`
	Line                int               `json:"line,omitempty"`
	OriginalLine        int               `json:"original_line,omitempty"`
	StartLine           int               `json:"start_line,omitempty"`
	OriginalStartLine   int               `json:"original_start_line,omitempty"`
	Side                string            `json:"side,omitempty"` // LEFT, RIGHT
	StartSide           string            `json:"start_side,omitempty"`
	// Internal
	PRID     int64  `json:"-"`
	ReviewID *int64 `json:"-"`
	UserID   int64  `json:"-"`
}

// PRFile represents a file in a PR
type PRFile struct {
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

// MergeResult represents the result of a merge operation
type MergeResult struct {
	SHA     string `json:"sha"`
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
}

// CreateIn represents input for creating a PR
type CreateIn struct {
	Title               string `json:"title"`
	Body                string `json:"body,omitempty"`
	Head                string `json:"head"` // branch name or owner:branch
	Base                string `json:"base"`
	Draft               bool   `json:"draft,omitempty"`
	MaintainerCanModify bool   `json:"maintainer_can_modify,omitempty"`
}

// UpdateIn represents input for updating a PR
type UpdateIn struct {
	Title               *string `json:"title,omitempty"`
	Body                *string `json:"body,omitempty"`
	State               *string `json:"state,omitempty"` // open, closed
	Base                *string `json:"base,omitempty"`
	MaintainerCanModify *bool   `json:"maintainer_can_modify,omitempty"`
}

// MergeIn represents input for merging a PR
type MergeIn struct {
	CommitTitle   string `json:"commit_title,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	SHA           string `json:"sha,omitempty"`
	MergeMethod   string `json:"merge_method,omitempty"` // merge, squash, rebase
}

// CreateReviewIn represents input for creating a review
type CreateReviewIn struct {
	CommitID string           `json:"commit_id,omitempty"`
	Body     string           `json:"body,omitempty"`
	Event    string           `json:"event,omitempty"` // APPROVE, REQUEST_CHANGES, COMMENT
	Comments []*ReviewComment `json:"comments,omitempty"`
}

// SubmitReviewIn represents input for submitting a review
type SubmitReviewIn struct {
	Body  string `json:"body,omitempty"`
	Event string `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
}

// ListOpts contains options for listing PRs
type ListOpts struct {
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per_page,omitempty"`
	State     string `json:"state,omitempty"`     // open, closed, all
	Head      string `json:"head,omitempty"`
	Base      string `json:"base,omitempty"`
	Sort      string `json:"sort,omitempty"`      // created, updated, popularity, long-running
	Direction string `json:"direction,omitempty"` // asc, desc
}

// API defines the pulls service interface
type API interface {
	// List returns PRs for a repository
	List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*PullRequest, error)

	// Get retrieves a PR by number
	Get(ctx context.Context, owner, repo string, number int) (*PullRequest, error)

	// Create creates a new PR
	Create(ctx context.Context, owner, repo string, creatorID int64, in *CreateIn) (*PullRequest, error)

	// Update updates a PR
	Update(ctx context.Context, owner, repo string, number int, in *UpdateIn) (*PullRequest, error)

	// ListCommits returns commits in a PR
	ListCommits(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Commit, error)

	// GetCommitBySHA retrieves a PR commit by SHA (from pr_commits table)
	GetCommitBySHA(ctx context.Context, owner, repo, sha string) (*Commit, error)

	// ListFiles returns files in a PR
	ListFiles(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*PRFile, error)

	// ListFilesByCommitSHA returns files for the PR that contains the given commit
	ListFilesByCommitSHA(ctx context.Context, owner, repo, sha string) ([]*PRFile, error)

	// IsMerged checks if a PR is merged
	IsMerged(ctx context.Context, owner, repo string, number int) (bool, error)

	// Merge merges a PR
	Merge(ctx context.Context, owner, repo string, number int, in *MergeIn) (*MergeResult, error)

	// UpdateBranch updates a PR branch
	UpdateBranch(ctx context.Context, owner, repo string, number int) error

	// Reviews
	ListReviews(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Review, error)
	GetReview(ctx context.Context, owner, repo string, number int, reviewID int64) (*Review, error)
	CreateReview(ctx context.Context, owner, repo string, number int, userID int64, in *CreateReviewIn) (*Review, error)
	UpdateReview(ctx context.Context, owner, repo string, number int, reviewID int64, body string) (*Review, error)
	SubmitReview(ctx context.Context, owner, repo string, number int, reviewID int64, in *SubmitReviewIn) (*Review, error)
	DismissReview(ctx context.Context, owner, repo string, number int, reviewID int64, message string) (*Review, error)

	// Review comments
	ListReviewComments(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*ReviewComment, error)
	CreateReviewComment(ctx context.Context, owner, repo string, number int, userID int64, in *CreateReviewCommentIn) (*ReviewComment, error)

	// Reviewers
	RequestReviewers(ctx context.Context, owner, repo string, number int, reviewers, teamReviewers []string) (*PullRequest, error)
	RemoveReviewers(ctx context.Context, owner, repo string, number int, reviewers, teamReviewers []string) (*PullRequest, error)
}

// CreateReviewCommentIn represents input for creating a review comment
type CreateReviewCommentIn struct {
	Body      string `json:"body"`
	CommitID  string `json:"commit_id"`
	Path      string `json:"path"`
	Position  int    `json:"position,omitempty"`
	Side      string `json:"side,omitempty"`
	Line      int    `json:"line,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	StartSide string `json:"start_side,omitempty"`
	InReplyTo int64  `json:"in_reply_to,omitempty"`
}

// Commit represents a commit in a PR
type Commit struct {
	SHA       string            `json:"sha"`
	NodeID    string            `json:"node_id"`
	Commit    *CommitData       `json:"commit"`
	URL       string            `json:"url"`
	HTMLURL   string            `json:"html_url"`
	Author    *users.SimpleUser `json:"author"`
	Committer *users.SimpleUser `json:"committer"`
	Parents   []*CommitRef      `json:"parents"`
}

// CommitData contains commit details
type CommitData struct {
	Author    *CommitAuthor `json:"author"`
	Committer *CommitAuthor `json:"committer"`
	Message   string        `json:"message"`
	Tree      *CommitRef    `json:"tree"`
	URL       string        `json:"url"`
}

// CommitAuthor represents a commit author
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// CommitRef represents a commit reference
type CommitRef struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// Store defines the data access interface for pulls
type Store interface {
	Create(ctx context.Context, pr *PullRequest) error
	GetByID(ctx context.Context, id int64) (*PullRequest, error)
	GetByNumber(ctx context.Context, repoID int64, number int) (*PullRequest, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, repoID int64, opts *ListOpts) ([]*PullRequest, error)
	NextNumber(ctx context.Context, repoID int64) (int, error)

	// Merge
	SetMerged(ctx context.Context, id int64, mergedAt time.Time, mergeCommitSHA string, mergedByID int64) error

	// Reviews
	CreateReview(ctx context.Context, review *Review) error
	GetReviewByID(ctx context.Context, id int64) (*Review, error)
	UpdateReview(ctx context.Context, id int64, body string) error
	SetReviewState(ctx context.Context, id int64, state string) error
	ListReviews(ctx context.Context, prID int64, opts *ListOpts) ([]*Review, error)

	// Review comments
	CreateReviewComment(ctx context.Context, comment *ReviewComment) error
	GetReviewCommentByID(ctx context.Context, id int64) (*ReviewComment, error)
	UpdateReviewComment(ctx context.Context, id int64, body string) error
	DeleteReviewComment(ctx context.Context, id int64) error
	ListReviewComments(ctx context.Context, prID int64, opts *ListOpts) ([]*ReviewComment, error)

	// Requested reviewers
	AddRequestedReviewer(ctx context.Context, prID, userID int64) error
	RemoveRequestedReviewer(ctx context.Context, prID, userID int64) error
	ListRequestedReviewers(ctx context.Context, prID int64) ([]*users.SimpleUser, error)

	// Requested teams
	AddRequestedTeam(ctx context.Context, prID, teamID int64) error
	RemoveRequestedTeam(ctx context.Context, prID, teamID int64) error
	ListRequestedTeams(ctx context.Context, prID int64) ([]*TeamSimple, error)

	// Commits
	CreateCommit(ctx context.Context, prID int64, commit *Commit) error
	ListCommits(ctx context.Context, prID int64, opts *ListOpts) ([]*Commit, error)
	GetCommitBySHA(ctx context.Context, repoID int64, sha string) (*Commit, error)

	// Files
	CreateFile(ctx context.Context, prID int64, file *PRFile) error
	ListFiles(ctx context.Context, prID int64, opts *ListOpts) ([]*PRFile, error)
	ListFilesByCommitSHA(ctx context.Context, repoID int64, sha string) ([]*PRFile, error)
}
