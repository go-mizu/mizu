package comments

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound     = errors.New("comment not found")
	ErrAccessDenied = errors.New("access denied")
)

// IssueComment represents a comment on an issue
type IssueComment struct {
	ID                int64             `json:"id"`
	NodeID            string            `json:"node_id"`
	URL               string            `json:"url"`
	HTMLURL           string            `json:"html_url"`
	Body              string            `json:"body"`
	User              *users.SimpleUser `json:"user"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	IssueURL          string            `json:"issue_url"`
	AuthorAssociation string            `json:"author_association"`
	Reactions         *Reactions        `json:"reactions,omitempty"`
	// Internal
	IssueID   int64 `json:"-"`
	RepoID    int64 `json:"-"`
	CreatorID int64 `json:"-"`
}

// CommitComment represents a comment on a commit
type CommitComment struct {
	ID        int64             `json:"id"`
	NodeID    string            `json:"node_id"`
	URL       string            `json:"url"`
	HTMLURL   string            `json:"html_url"`
	Body      string            `json:"body"`
	User      *users.SimpleUser `json:"user"`
	Path      string            `json:"path,omitempty"`
	Position  int               `json:"position,omitempty"`
	Line      int               `json:"line,omitempty"`
	CommitID  string            `json:"commit_id"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	// Internal
	RepoID    int64 `json:"-"`
	CreatorID int64 `json:"-"`
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

// CreateIssueCommentIn represents input for creating an issue comment
type CreateIssueCommentIn struct {
	Body string `json:"body"`
}

// CreateCommitCommentIn represents input for creating a commit comment
type CreateCommitCommentIn struct {
	Body     string `json:"body"`
	Path     string `json:"path,omitempty"`
	Position int    `json:"position,omitempty"`
	Line     int    `json:"line,omitempty"`
}

// UpdateCommentIn represents input for updating a comment
type UpdateCommentIn struct {
	Body string `json:"body"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page      int       `json:"page,omitempty"`
	PerPage   int       `json:"per_page,omitempty"`
	Sort      string    `json:"sort,omitempty"`      // created, updated
	Direction string    `json:"direction,omitempty"` // asc, desc
	Since     time.Time `json:"since,omitempty"`
}

// API defines the comments service interface
type API interface {
	// Issue comments
	ListForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*IssueComment, error)
	ListForIssue(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*IssueComment, error)
	ListForPR(ctx context.Context, owner, repo string, prID int64, opts *ListOpts) ([]*IssueComment, error)
	ListUniqueCommentersForIssue(ctx context.Context, owner, repo string, number int) ([]*users.SimpleUser, error)
	GetIssueComment(ctx context.Context, owner, repo string, commentID int64) (*IssueComment, error)
	CreateIssueComment(ctx context.Context, owner, repo string, number int, creatorID int64, body string) (*IssueComment, error)
	UpdateIssueComment(ctx context.Context, owner, repo string, commentID int64, body string) (*IssueComment, error)
	DeleteIssueComment(ctx context.Context, owner, repo string, commentID int64) error

	// Commit comments
	ListCommitCommentsForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*CommitComment, error)
	ListForCommit(ctx context.Context, owner, repo, sha string, opts *ListOpts) ([]*CommitComment, error)
	GetCommitComment(ctx context.Context, owner, repo string, commentID int64) (*CommitComment, error)
	CreateCommitComment(ctx context.Context, owner, repo, sha string, creatorID int64, in *CreateCommitCommentIn) (*CommitComment, error)
	UpdateCommitComment(ctx context.Context, owner, repo string, commentID int64, body string) (*CommitComment, error)
	DeleteCommitComment(ctx context.Context, owner, repo string, commentID int64) error
}

// Store defines the data access interface for comments
type Store interface {
	// Issue comments
	CreateIssueComment(ctx context.Context, c *IssueComment) error
	GetIssueCommentByID(ctx context.Context, id int64) (*IssueComment, error)
	UpdateIssueComment(ctx context.Context, id int64, body string) error
	DeleteIssueComment(ctx context.Context, id int64) error
	ListIssueCommentsForRepo(ctx context.Context, repoID int64, opts *ListOpts) ([]*IssueComment, error)
	ListIssueCommentsForIssue(ctx context.Context, issueID int64, opts *ListOpts) ([]*IssueComment, error)
	ListUniqueCommenters(ctx context.Context, issueID int64) ([]*users.SimpleUser, error)

	// Commit comments
	CreateCommitComment(ctx context.Context, c *CommitComment) error
	GetCommitCommentByID(ctx context.Context, id int64) (*CommitComment, error)
	UpdateCommitComment(ctx context.Context, id int64, body string) error
	DeleteCommitComment(ctx context.Context, id int64) error
	ListCommitCommentsForRepo(ctx context.Context, repoID int64, opts *ListOpts) ([]*CommitComment, error)
	ListCommitCommentsForCommit(ctx context.Context, repoID int64, sha string, opts *ListOpts) ([]*CommitComment, error)
}
