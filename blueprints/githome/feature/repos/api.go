package repos

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound     = errors.New("repository not found")
	ErrRepoExists   = errors.New("repository already exists")
	ErrAccessDenied = errors.New("access denied")
)

// Repository represents a GitHub repository
type Repository struct {
	ID                  int64             `json:"id"`
	NodeID              string            `json:"node_id"`
	Name                string            `json:"name"`
	FullName            string            `json:"full_name"`
	Owner               *users.SimpleUser `json:"owner"`
	Private             bool              `json:"private"`
	HTMLURL             string            `json:"html_url"`
	Description         string            `json:"description,omitempty"`
	Fork                bool              `json:"fork"`
	URL                 string            `json:"url"`
	ForksURL            string            `json:"forks_url"`
	KeysURL             string            `json:"keys_url"`
	CollaboratorsURL    string            `json:"collaborators_url"`
	TeamsURL            string            `json:"teams_url"`
	HooksURL            string            `json:"hooks_url"`
	IssueEventsURL      string            `json:"issue_events_url"`
	EventsURL           string            `json:"events_url"`
	AssigneesURL        string            `json:"assignees_url"`
	BranchesURL         string            `json:"branches_url"`
	TagsURL             string            `json:"tags_url"`
	BlobsURL            string            `json:"blobs_url"`
	GitTagsURL          string            `json:"git_tags_url"`
	GitRefsURL          string            `json:"git_refs_url"`
	TreesURL            string            `json:"trees_url"`
	StatusesURL         string            `json:"statuses_url"`
	LanguagesURL        string            `json:"languages_url"`
	StargazersURL       string            `json:"stargazers_url"`
	ContributorsURL     string            `json:"contributors_url"`
	SubscribersURL      string            `json:"subscribers_url"`
	SubscriptionURL     string            `json:"subscription_url"`
	CommitsURL          string            `json:"commits_url"`
	GitCommitsURL       string            `json:"git_commits_url"`
	CommentsURL         string            `json:"comments_url"`
	IssueCommentURL     string            `json:"issue_comment_url"`
	ContentsURL         string            `json:"contents_url"`
	CompareURL          string            `json:"compare_url"`
	MergesURL           string            `json:"merges_url"`
	ArchiveURL          string            `json:"archive_url"`
	DownloadsURL        string            `json:"downloads_url"`
	IssuesURL           string            `json:"issues_url"`
	PullsURL            string            `json:"pulls_url"`
	MilestonesURL       string            `json:"milestones_url"`
	NotificationsURL    string            `json:"notifications_url"`
	LabelsURL           string            `json:"labels_url"`
	ReleasesURL         string            `json:"releases_url"`
	DeploymentsURL      string            `json:"deployments_url"`
	GitURL              string            `json:"git_url"`
	SSHURL              string            `json:"ssh_url"`
	CloneURL            string            `json:"clone_url"`
	SVNURL              string            `json:"svn_url"`
	Homepage            string            `json:"homepage,omitempty"`
	Language            string            `json:"language,omitempty"`
	ForksCount          int               `json:"forks_count"`
	Forks               int               `json:"forks"`
	StargazersCount     int               `json:"stargazers_count"`
	WatchersCount       int               `json:"watchers_count"`
	Watchers            int               `json:"watchers"`
	Size                int               `json:"size"`
	DefaultBranch       string            `json:"default_branch"`
	OpenIssuesCount     int               `json:"open_issues_count"`
	OpenIssues          int               `json:"open_issues"`
	IsTemplate          bool              `json:"is_template"`
	Topics              []string          `json:"topics"`
	HasIssues           bool              `json:"has_issues"`
	HasProjects         bool              `json:"has_projects"`
	HasWiki             bool              `json:"has_wiki"`
	HasPages            bool              `json:"has_pages"`
	HasDownloads        bool              `json:"has_downloads"`
	HasDiscussions      bool              `json:"has_discussions"`
	Archived            bool              `json:"archived"`
	Disabled            bool              `json:"disabled"`
	Visibility          string            `json:"visibility"` // public, private, internal
	PushedAt            *time.Time        `json:"pushed_at"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	Permissions         *Permissions      `json:"permissions,omitempty"`
	License             *License          `json:"license,omitempty"`
	AllowRebaseMerge    bool              `json:"allow_rebase_merge"`
	AllowSquashMerge    bool              `json:"allow_squash_merge"`
	AllowMergeCommit    bool              `json:"allow_merge_commit"`
	AllowAutoMerge      bool              `json:"allow_auto_merge"`
	DeleteBranchOnMerge bool              `json:"delete_branch_on_merge"`
	AllowForking        bool              `json:"allow_forking"`
	WebCommitSignoffRequired bool         `json:"web_commit_signoff_required"`
	// Internal fields
	OwnerID   int64  `json:"-"`
	OwnerType string `json:"-"` // User, Organization
}

// Permissions represents user permissions on a repo
type Permissions struct {
	Admin    bool `json:"admin"`
	Maintain bool `json:"maintain"`
	Push     bool `json:"push"`
	Triage   bool `json:"triage"`
	Pull     bool `json:"pull"`
}

// License represents a repository license
type License struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	SPDXID string `json:"spdx_id"`
	URL    string `json:"url,omitempty"`
	NodeID string `json:"node_id"`
}

// CreateIn represents the input for creating a repository
type CreateIn struct {
	Name                string `json:"name"`
	Description         string `json:"description,omitempty"`
	Homepage            string `json:"homepage,omitempty"`
	Private             bool   `json:"private"`
	Visibility          string `json:"visibility,omitempty"` // public, private, internal
	HasIssues           *bool  `json:"has_issues,omitempty"`
	HasProjects         *bool  `json:"has_projects,omitempty"`
	HasWiki             *bool  `json:"has_wiki,omitempty"`
	HasDiscussions      *bool  `json:"has_discussions,omitempty"`
	IsTemplate          bool   `json:"is_template"`
	AutoInit            bool   `json:"auto_init"`
	GitignoreTemplate   string `json:"gitignore_template,omitempty"`
	LicenseTemplate     string `json:"license_template,omitempty"`
	AllowSquashMerge    *bool  `json:"allow_squash_merge,omitempty"`
	AllowMergeCommit    *bool  `json:"allow_merge_commit,omitempty"`
	AllowRebaseMerge    *bool  `json:"allow_rebase_merge,omitempty"`
	AllowAutoMerge      *bool  `json:"allow_auto_merge,omitempty"`
	DeleteBranchOnMerge *bool  `json:"delete_branch_on_merge,omitempty"`
}

// UpdateIn represents the input for updating a repository
type UpdateIn struct {
	Name                  *string `json:"name,omitempty"`
	Description           *string `json:"description,omitempty"`
	Homepage              *string `json:"homepage,omitempty"`
	Private               *bool   `json:"private,omitempty"`
	Visibility            *string `json:"visibility,omitempty"`
	HasIssues             *bool   `json:"has_issues,omitempty"`
	HasProjects           *bool   `json:"has_projects,omitempty"`
	HasWiki               *bool   `json:"has_wiki,omitempty"`
	HasDiscussions        *bool   `json:"has_discussions,omitempty"`
	IsTemplate            *bool   `json:"is_template,omitempty"`
	DefaultBranch         *string `json:"default_branch,omitempty"`
	AllowSquashMerge      *bool   `json:"allow_squash_merge,omitempty"`
	AllowMergeCommit      *bool   `json:"allow_merge_commit,omitempty"`
	AllowRebaseMerge      *bool   `json:"allow_rebase_merge,omitempty"`
	AllowAutoMerge        *bool   `json:"allow_auto_merge,omitempty"`
	DeleteBranchOnMerge   *bool   `json:"delete_branch_on_merge,omitempty"`
	AllowForking          *bool   `json:"allow_forking,omitempty"`
	Archived              *bool   `json:"archived,omitempty"`
	WebCommitSignoffRequired *bool `json:"web_commit_signoff_required,omitempty"`
}

// TransferIn represents the input for transferring a repository
type TransferIn struct {
	NewOwner  string  `json:"new_owner"`
	NewName   *string `json:"new_name,omitempty"`
	TeamIDs   []int64 `json:"team_ids,omitempty"`
}

// ListOpts contains options for listing repositories
type ListOpts struct {
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per_page,omitempty"`
	Type      string `json:"type,omitempty"`       // all, public, private, forks, sources, member
	Sort      string `json:"sort,omitempty"`       // created, updated, pushed, full_name
	Direction string `json:"direction,omitempty"` // asc, desc
}

// Content represents file/directory content
type Content struct {
	Type        string `json:"type"` // file, dir, symlink, submodule
	Encoding    string `json:"encoding,omitempty"`
	Size        int    `json:"size"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content,omitempty"`
	SHA         string `json:"sha"`
	URL         string `json:"url"`
	GitURL      string `json:"git_url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url,omitempty"`
	Target      string `json:"target,omitempty"` // for symlinks
	SubmoduleGitURL string `json:"submodule_git_url,omitempty"`
}

// TreeEntry represents a file or directory entry in a tree listing
type TreeEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int64  `json:"size"`
	Type        string `json:"type"` // "file", "dir", "symlink", "submodule"
	Mode        string `json:"mode"` // "100644", "100755", "040000", etc.
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url,omitempty"`
	// Last commit info (optional, populated when requested)
	LastCommitSHA     string    `json:"last_commit_sha,omitempty"`
	LastCommitMessage string    `json:"last_commit_message,omitempty"`
	LastCommitAuthor  string    `json:"last_commit_author,omitempty"`
	LastCommitDate    time.Time `json:"last_commit_date,omitempty"`
}

// BlameLine represents a line with blame information
type BlameLine struct {
	LineNumber int       `json:"line_number"`
	Content    string    `json:"content"`
	CommitSHA  string    `json:"commit_sha"`
	Author     string    `json:"author"`
	AuthorMail string    `json:"author_mail"`
	Date       time.Time `json:"date"`
}

// BlameResult contains blame information for a file
type BlameResult struct {
	Path  string       `json:"path"`
	Lines []*BlameLine `json:"lines"`
}

// FileCommit represents the result of a file operation
type FileCommit struct {
	Content *Content `json:"content"`
	Commit  *Commit  `json:"commit"`
}

// Commit represents a commit in a file operation
type Commit struct {
	SHA       string        `json:"sha"`
	NodeID    string        `json:"node_id"`
	URL       string        `json:"url"`
	HTMLURL   string        `json:"html_url"`
	Author    *CommitAuthor `json:"author"`
	Committer *CommitAuthor `json:"committer"`
	Message   string        `json:"message"`
	Tree      *TreeRef      `json:"tree"`
	Parents   []*TreeRef    `json:"parents"`
}

// CommitAuthor represents a commit author
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// TreeRef represents a reference to a tree/commit
type TreeRef struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// Contributor represents a repository contributor
type Contributor struct {
	*users.SimpleUser
	Contributions int `json:"contributions"`
}

// API defines the repositories service interface
type API interface {
	// Create creates a new repository for a user
	Create(ctx context.Context, ownerID int64, in *CreateIn) (*Repository, error)

	// CreateForOrg creates a new repository for an organization
	CreateForOrg(ctx context.Context, org string, in *CreateIn) (*Repository, error)

	// Get retrieves a repository by owner and name
	Get(ctx context.Context, owner, repo string) (*Repository, error)

	// GetByID retrieves a repository by ID
	GetByID(ctx context.Context, id int64) (*Repository, error)

	// Update updates a repository
	Update(ctx context.Context, owner, repo string, in *UpdateIn) (*Repository, error)

	// Delete removes a repository
	Delete(ctx context.Context, owner, repo string) error

	// Transfer transfers a repository to a new owner
	Transfer(ctx context.Context, owner, repo string, in *TransferIn) (*Repository, error)

	// ListForUser returns repositories for a user
	ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*Repository, error)

	// ListForOrg returns repositories for an organization
	ListForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Repository, error)

	// ListForAuthenticatedUser returns repositories for the authenticated user
	ListForAuthenticatedUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Repository, error)

	// ListForks returns forks of a repository
	ListForks(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Repository, error)

	// CreateFork creates a fork of a repository
	CreateFork(ctx context.Context, owner, repo string, org, name string) (*Repository, error)

	// ListLanguages returns language statistics
	ListLanguages(ctx context.Context, owner, repo string) (map[string]int, error)

	// ListTopics returns repository topics
	ListTopics(ctx context.Context, owner, repo string) ([]string, error)

	// ReplaceTopics replaces all topics
	ReplaceTopics(ctx context.Context, owner, repo string, topics []string) ([]string, error)

	// ListContributors returns repository contributors
	ListContributors(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Contributor, error)

	// CountContributors returns the total number of contributors for a repository
	CountContributors(ctx context.Context, owner, repo string) (int, error)

	// GetReadme returns the README content
	GetReadme(ctx context.Context, owner, repo, ref string) (*Content, error)

	// GetContents returns file or directory contents
	GetContents(ctx context.Context, owner, repo, path, ref string) (*Content, error)

	// ListTreeEntries returns directory entries
	ListTreeEntries(ctx context.Context, owner, repo, path, ref string) ([]*TreeEntry, error)

	// ListTreeEntriesWithCommits returns directory entries with last commit info
	ListTreeEntriesWithCommits(ctx context.Context, owner, repo, path, ref string) ([]*TreeEntry, error)

	// GetBlame returns blame information for a file
	GetBlame(ctx context.Context, owner, repo, ref, path string) (*BlameResult, error)

	// GetCommitCount returns the total number of commits from a ref
	GetCommitCount(ctx context.Context, owner, repo, ref string) (int, error)

	// GetLatestCommit returns the latest commit for a ref
	GetLatestCommit(ctx context.Context, owner, repo, ref string) (*Commit, error)

	// CreateOrUpdateFile creates or updates a file
	CreateOrUpdateFile(ctx context.Context, owner, repo, path string, message, content, sha, branch string, author *CommitAuthor) (*FileCommit, error)

	// DeleteFile deletes a file
	DeleteFile(ctx context.Context, owner, repo, path, message, sha, branch string, author *CommitAuthor) (*FileCommit, error)

	// IncrementOpenIssues adjusts the open issues count
	IncrementOpenIssues(ctx context.Context, id int64, delta int) error

	// IncrementStargazers adjusts the stargazers count
	IncrementStargazers(ctx context.Context, id int64, delta int) error

	// IncrementWatchers adjusts the watchers count
	IncrementWatchers(ctx context.Context, id int64, delta int) error

	// IncrementForks adjusts the forks count
	IncrementForks(ctx context.Context, id int64, delta int) error
}

// Store defines the data access interface for repositories
type Store interface {
	Create(ctx context.Context, r *Repository) error
	GetByID(ctx context.Context, id int64) (*Repository, error)
	GetByOwnerAndName(ctx context.Context, ownerID int64, name string) (*Repository, error)
	GetByFullName(ctx context.Context, owner, name string) (*Repository, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, opts *ListOpts) ([]*Repository, error)
	ListByOwner(ctx context.Context, ownerID int64, opts *ListOpts) ([]*Repository, error)
	ListForks(ctx context.Context, repoID int64, opts *ListOpts) ([]*Repository, error)

	// Topics
	GetTopics(ctx context.Context, repoID int64) ([]string, error)
	SetTopics(ctx context.Context, repoID int64, topics []string) error

	// Languages
	GetLanguages(ctx context.Context, repoID int64) (map[string]int, error)
	SetLanguages(ctx context.Context, repoID int64, languages map[string]int) error

	// Counter operations
	IncrementOpenIssues(ctx context.Context, repoID int64, delta int) error
	IncrementStargazers(ctx context.Context, repoID int64, delta int) error
	IncrementWatchers(ctx context.Context, repoID int64, delta int) error
	IncrementForks(ctx context.Context, repoID int64, delta int) error

	// Seeded contributors (from GitHub import)
	ListSeededContributors(ctx context.Context, repoID int64, opts *ListOpts) ([]*Contributor, error)
	CountSeededContributors(ctx context.Context, repoID int64) (int, error)
	HasSeededContributors(ctx context.Context, repoID int64) (bool, error)
}
