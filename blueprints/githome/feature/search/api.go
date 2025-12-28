package search

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Result is a generic search result
type Result[T any] struct {
	TotalCount        int  `json:"total_count"`
	IncompleteResults bool `json:"incomplete_results"`
	Items             []T  `json:"items"`
}

// CodeResult represents a code search result
type CodeResult struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	SHA        string      `json:"sha"`
	URL        string      `json:"url"`
	GitURL     string      `json:"git_url"`
	HTMLURL    string      `json:"html_url"`
	Repository *Repository `json:"repository"`
	Score      float64     `json:"score"`
	// Text matches
	TextMatches []*TextMatch `json:"text_matches,omitempty"`
}

// TextMatch represents a text match in search results
type TextMatch struct {
	ObjectURL  string    `json:"object_url"`
	ObjectType string    `json:"object_type"`
	Property   string    `json:"property"`
	Fragment   string    `json:"fragment"`
	Matches    []*Match  `json:"matches"`
}

// Match represents a specific match
type Match struct {
	Text    string `json:"text"`
	Indices []int  `json:"indices"`
}

// CommitResult represents a commit search result
type CommitResult struct {
	URL         string            `json:"url"`
	SHA         string            `json:"sha"`
	HTMLURL     string            `json:"html_url"`
	CommentsURL string            `json:"comments_url"`
	Commit      *CommitData       `json:"commit"`
	Author      *users.SimpleUser `json:"author"`
	Committer   *users.SimpleUser `json:"committer"`
	Parents     []*CommitRef      `json:"parents"`
	Repository  *Repository       `json:"repository"`
	Score       float64           `json:"score"`
}

// CommitData contains commit details
type CommitData struct {
	Author       *CommitAuthor `json:"author"`
	Committer    *CommitAuthor `json:"committer"`
	Message      string        `json:"message"`
	CommentCount int           `json:"comment_count"`
	Tree         *CommitRef    `json:"tree"`
	URL          string        `json:"url"`
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

// Issue represents an issue in search results
type Issue struct {
	ID                int64             `json:"id"`
	NodeID            string            `json:"node_id"`
	URL               string            `json:"url"`
	RepositoryURL     string            `json:"repository_url"`
	HTMLURL           string            `json:"html_url"`
	Number            int               `json:"number"`
	State             string            `json:"state"`
	Title             string            `json:"title"`
	Body              string            `json:"body,omitempty"`
	User              *users.SimpleUser `json:"user"`
	Labels            []*Label          `json:"labels"`
	Assignee          *users.SimpleUser `json:"assignee,omitempty"`
	Assignees         []*users.SimpleUser `json:"assignees"`
	Comments          int               `json:"comments"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	ClosedAt          *time.Time        `json:"closed_at"`
	AuthorAssociation string            `json:"author_association"`
	Score             float64           `json:"score"`
	PullRequest       *PullRequestRef   `json:"pull_request,omitempty"`
}

// Label represents a label
type Label struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

// PullRequestRef indicates if an issue is a PR
type PullRequestRef struct {
	URL      string `json:"url"`
	HTMLURL  string `json:"html_url"`
	DiffURL  string `json:"diff_url"`
	PatchURL string `json:"patch_url"`
}

// Repository represents a repository in search results
type Repository struct {
	ID              int64             `json:"id"`
	NodeID          string            `json:"node_id"`
	Name            string            `json:"name"`
	FullName        string            `json:"full_name"`
	Owner           *users.SimpleUser `json:"owner"`
	Private         bool              `json:"private"`
	HTMLURL         string            `json:"html_url"`
	Description     string            `json:"description,omitempty"`
	Fork            bool              `json:"fork"`
	URL             string            `json:"url"`
	Language        string            `json:"language,omitempty"`
	ForksCount      int               `json:"forks_count"`
	StargazersCount int               `json:"stargazers_count"`
	WatchersCount   int               `json:"watchers_count"`
	DefaultBranch   string            `json:"default_branch"`
	OpenIssuesCount int               `json:"open_issues_count"`
	Topics          []string          `json:"topics"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	PushedAt        *time.Time        `json:"pushed_at"`
	Score           float64           `json:"score"`
}

// Topic represents a topic
type Topic struct {
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name,omitempty"`
	ShortDescription string    `json:"short_description,omitempty"`
	Description      string    `json:"description,omitempty"`
	CreatedBy        string    `json:"created_by,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Featured         bool      `json:"featured"`
	Curated          bool      `json:"curated"`
	Score            float64   `json:"score"`
}

// User represents a user in search results
type User struct {
	ID        int64   `json:"id"`
	NodeID    string  `json:"node_id"`
	Login     string  `json:"login"`
	AvatarURL string  `json:"avatar_url"`
	HTMLURL   string  `json:"html_url"`
	URL       string  `json:"url"`
	Type      string  `json:"type"`
	SiteAdmin bool    `json:"site_admin"`
	Score     float64 `json:"score"`
}

// SearchCodeOpts contains options for code search
type SearchCodeOpts struct {
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per_page,omitempty"`
	Sort    string `json:"sort,omitempty"` // indexed
	Order   string `json:"order,omitempty"` // asc, desc
}

// SearchOpts contains generic search options
type SearchOpts struct {
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per_page,omitempty"`
	Sort    string `json:"sort,omitempty"`
	Order   string `json:"order,omitempty"`
}

// SearchIssuesOpts contains options for issue/PR search
type SearchIssuesOpts struct {
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per_page,omitempty"`
	Sort    string `json:"sort,omitempty"` // comments, reactions, reactions-+1, reactions--1, reactions-smile, reactions-thinking_face, reactions-heart, reactions-tada, interactions, created, updated
	Order   string `json:"order,omitempty"`
}

// SearchReposOpts contains options for repo search
type SearchReposOpts struct {
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per_page,omitempty"`
	Sort    string `json:"sort,omitempty"` // stars, forks, help-wanted-issues, updated
	Order   string `json:"order,omitempty"`
}

// SearchUsersOpts contains options for user search
type SearchUsersOpts struct {
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per_page,omitempty"`
	Sort    string `json:"sort,omitempty"` // followers, repositories, joined
	Order   string `json:"order,omitempty"`
}

// API defines the search service interface
type API interface {
	// Code searches code
	Code(ctx context.Context, query string, opts *SearchCodeOpts) (*Result[CodeResult], error)

	// Commits searches commits
	Commits(ctx context.Context, query string, opts *SearchOpts) (*Result[CommitResult], error)

	// IssuesAndPullRequests searches issues and PRs
	IssuesAndPullRequests(ctx context.Context, query string, opts *SearchIssuesOpts) (*Result[Issue], error)

	// Labels searches labels
	Labels(ctx context.Context, repoID int64, query string, opts *SearchOpts) (*Result[Label], error)

	// Repositories searches repositories
	Repositories(ctx context.Context, query string, opts *SearchReposOpts) (*Result[Repository], error)

	// Topics searches topics
	Topics(ctx context.Context, query string, opts *SearchOpts) (*Result[Topic], error)

	// Users searches users
	Users(ctx context.Context, query string, opts *SearchUsersOpts) (*Result[User], error)
}

// Store defines the data access interface for search
type Store interface {
	SearchCode(ctx context.Context, query string, opts *SearchCodeOpts) (*Result[CodeResult], error)
	SearchCommits(ctx context.Context, query string, opts *SearchOpts) (*Result[CommitResult], error)
	SearchIssues(ctx context.Context, query string, opts *SearchIssuesOpts) (*Result[Issue], error)
	SearchLabels(ctx context.Context, repoID int64, query string, opts *SearchOpts) (*Result[Label], error)
	SearchRepositories(ctx context.Context, query string, opts *SearchReposOpts) (*Result[Repository], error)
	SearchTopics(ctx context.Context, query string, opts *SearchOpts) (*Result[Topic], error)
	SearchUsers(ctx context.Context, query string, opts *SearchUsersOpts) (*Result[User], error)
}
