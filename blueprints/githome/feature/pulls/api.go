package pulls

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound        = errors.New("pull request not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrMissingTitle    = errors.New("pull request title is required")
	ErrAccessDenied    = errors.New("access denied")
	ErrLocked          = errors.New("pull request is locked")
	ErrAlreadyMerged   = errors.New("pull request is already merged")
	ErrNotMergeable    = errors.New("pull request is not mergeable")
	ErrAlreadyClosed   = errors.New("pull request is already closed")
	ErrAlreadyOpen     = errors.New("pull request is already open")
	ErrReviewNotFound  = errors.New("review not found")
	ErrCommentNotFound = errors.New("comment not found")
)

// States
const (
	StateOpen   = "open"
	StateClosed = "closed"
	StateMerged = "merged"
)

// Merge methods
const (
	MergeMethodMerge  = "merge"
	MergeMethodSquash = "squash"
	MergeMethodRebase = "rebase"
)

// Review states
const (
	ReviewPending          = "pending"
	ReviewApproved         = "approved"
	ReviewChangesRequested = "changes_requested"
	ReviewCommented        = "commented"
	ReviewDismissed        = "dismissed"
)

// PullRequest represents a pull request
type PullRequest struct {
	ID             string     `json:"id"`
	RepoID         string     `json:"repo_id"`
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	AuthorID       string     `json:"author_id"`
	HeadRepoID     string     `json:"head_repo_id,omitempty"`
	HeadBranch     string     `json:"head_branch"`
	HeadSHA        string     `json:"head_sha"`
	BaseBranch     string     `json:"base_branch"`
	BaseSHA        string     `json:"base_sha"`
	State          string     `json:"state"`
	IsDraft        bool       `json:"is_draft"`
	IsLocked       bool       `json:"is_locked"`
	LockReason     string     `json:"lock_reason,omitempty"`
	Mergeable      bool       `json:"mergeable"`
	MergeableState string     `json:"mergeable_state"`
	MergeCommitSHA string     `json:"merge_commit_sha,omitempty"`
	MergedAt       *time.Time `json:"merged_at,omitempty"`
	MergedByID     string     `json:"merged_by_id,omitempty"`
	Additions      int        `json:"additions"`
	Deletions      int        `json:"deletions"`
	ChangedFiles   int        `json:"changed_files"`
	CommentCount   int        `json:"comment_count"`
	ReviewComments int        `json:"review_comments"`
	Commits        int        `json:"commits"`
	MilestoneID    string     `json:"milestone_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`

	// Populated from joins
	Labels    []string `json:"labels,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
}

// Review represents a pull request review
type Review struct {
	ID          string     `json:"id"`
	PRID        string     `json:"pr_id"`
	UserID      string     `json:"user_id"`
	Body        string     `json:"body"`
	State       string     `json:"state"`
	CommitSHA   string     `json:"commit_sha"`
	CreatedAt   time.Time  `json:"created_at"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
}

// ReviewComment represents a comment on a pull request review
type ReviewComment struct {
	ID               string    `json:"id"`
	ReviewID         string    `json:"review_id"`
	PRID             string    `json:"pr_id"`
	UserID           string    `json:"user_id"`
	Path             string    `json:"path"`
	Position         int       `json:"position,omitempty"`
	OriginalPosition int       `json:"original_position,omitempty"`
	DiffHunk         string    `json:"diff_hunk,omitempty"`
	Line             int       `json:"line,omitempty"`
	OriginalLine     int       `json:"original_line,omitempty"`
	Side             string    `json:"side"`
	Body             string    `json:"body"`
	InReplyToID      string    `json:"in_reply_to_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// PRLabel represents the association between a PR and a label
type PRLabel struct {
	ID        string    `json:"id"`
	PRID      string    `json:"pr_id"`
	LabelID   string    `json:"label_id"`
	CreatedAt time.Time `json:"created_at"`
}

// PRAssignee represents the association between a PR and an assignee
type PRAssignee struct {
	ID        string    `json:"id"`
	PRID      string    `json:"pr_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// PRReviewer represents a requested reviewer
type PRReviewer struct {
	ID        string    `json:"id"`
	PRID      string    `json:"pr_id"`
	UserID    string    `json:"user_id"`
	State     string    `json:"state"` // pending, reviewed
	CreatedAt time.Time `json:"created_at"`
}

// CreateIn is the input for creating a pull request
type CreateIn struct {
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	HeadBranch string   `json:"head_branch"`
	BaseBranch string   `json:"base_branch"`
	HeadRepoID string   `json:"head_repo_id,omitempty"`
	IsDraft    bool     `json:"is_draft"`
	Assignees  []string `json:"assignees,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	Reviewers  []string `json:"reviewers,omitempty"`
}

// UpdateIn is the input for updating a pull request
type UpdateIn struct {
	Title       *string   `json:"title,omitempty"`
	Body        *string   `json:"body,omitempty"`
	BaseBranch  *string   `json:"base_branch,omitempty"`
	MilestoneID *string   `json:"milestone_id,omitempty"`
	Assignees   *[]string `json:"assignees,omitempty"`
	Labels      *[]string `json:"labels,omitempty"`
}

// CreateReviewIn is the input for creating a review
type CreateReviewIn struct {
	Body      string                    `json:"body"`
	CommitSHA string                    `json:"commit_sha"`
	Event     string                    `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
	Comments  []*CreateReviewCommentIn `json:"comments,omitempty"`
}

// CreateReviewCommentIn is the input for creating a review comment
type CreateReviewCommentIn struct {
	Path     string `json:"path"`
	Position int    `json:"position,omitempty"`
	Line     int    `json:"line,omitempty"`
	Side     string `json:"side,omitempty"`
	Body     string `json:"body"`
}

// ListOpts are options for listing pull requests
type ListOpts struct {
	State     string // open, closed, all
	Sort      string // created, updated, popularity
	Direction string // asc, desc
	Head      string
	Base      string
	Limit     int
	Offset    int
}

// API is the pulls service interface
type API interface {
	// CRUD
	Create(ctx context.Context, repoID, authorID string, in *CreateIn) (*PullRequest, error)
	GetByID(ctx context.Context, id string) (*PullRequest, error)
	GetByNumber(ctx context.Context, repoID string, number int) (*PullRequest, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*PullRequest, error)
	List(ctx context.Context, repoID string, opts *ListOpts) ([]*PullRequest, int, error)

	// State management
	Close(ctx context.Context, id string) error
	Reopen(ctx context.Context, id string) error
	Merge(ctx context.Context, id, userID string, method string, commitMessage string) error
	MarkReady(ctx context.Context, id string) error
	Lock(ctx context.Context, id, reason string) error
	Unlock(ctx context.Context, id string) error

	// Labels
	AddLabels(ctx context.Context, id string, labelIDs []string) error
	RemoveLabel(ctx context.Context, id, labelID string) error
	SetLabels(ctx context.Context, id string, labelIDs []string) error

	// Assignees
	AddAssignees(ctx context.Context, id string, userIDs []string) error
	RemoveAssignees(ctx context.Context, id string, userIDs []string) error

	// Reviewers
	RequestReview(ctx context.Context, id string, userIDs []string) error
	RemoveReviewRequest(ctx context.Context, id string, userIDs []string) error

	// Reviews
	CreateReview(ctx context.Context, prID, userID string, in *CreateReviewIn) (*Review, error)
	GetReview(ctx context.Context, id string) (*Review, error)
	SubmitReview(ctx context.Context, reviewID, event string) (*Review, error)
	DismissReview(ctx context.Context, reviewID, message string) error
	ListReviews(ctx context.Context, prID string) ([]*Review, error)

	// Review Comments
	CreateReviewComment(ctx context.Context, prID, reviewID, userID string, in *CreateReviewCommentIn) (*ReviewComment, error)
	UpdateReviewComment(ctx context.Context, commentID, body string) (*ReviewComment, error)
	DeleteReviewComment(ctx context.Context, commentID string) error
	ListReviewComments(ctx context.Context, prID string) ([]*ReviewComment, error)
}

// Store is the pulls data store interface
type Store interface {
	// PR CRUD
	Create(ctx context.Context, pr *PullRequest) error
	GetByID(ctx context.Context, id string) (*PullRequest, error)
	GetByNumber(ctx context.Context, repoID string, number int) (*PullRequest, error)
	Update(ctx context.Context, pr *PullRequest) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string, state string, limit, offset int) ([]*PullRequest, int, error)
	GetNextNumber(ctx context.Context, repoID string) (int, error)

	// Labels
	AddLabel(ctx context.Context, pl *PRLabel) error
	RemoveLabel(ctx context.Context, prID, labelID string) error
	ListLabels(ctx context.Context, prID string) ([]string, error)

	// Assignees
	AddAssignee(ctx context.Context, pa *PRAssignee) error
	RemoveAssignee(ctx context.Context, prID, userID string) error
	ListAssignees(ctx context.Context, prID string) ([]string, error)

	// Reviewers
	AddReviewer(ctx context.Context, pr *PRReviewer) error
	RemoveReviewer(ctx context.Context, prID, userID string) error
	ListReviewers(ctx context.Context, prID string) ([]*PRReviewer, error)

	// Reviews
	CreateReview(ctx context.Context, r *Review) error
	GetReview(ctx context.Context, id string) (*Review, error)
	UpdateReview(ctx context.Context, r *Review) error
	ListReviews(ctx context.Context, prID string) ([]*Review, error)

	// Review Comments
	CreateReviewComment(ctx context.Context, rc *ReviewComment) error
	GetReviewComment(ctx context.Context, id string) (*ReviewComment, error)
	UpdateReviewComment(ctx context.Context, rc *ReviewComment) error
	DeleteReviewComment(ctx context.Context, id string) error
	ListReviewComments(ctx context.Context, prID string) ([]*ReviewComment, error)
}
