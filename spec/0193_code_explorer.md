# Code Explorer Specification

## Overview

The Code Explorer is the core feature of GitHome that allows users to browse repository files, view file contents, and navigate the repository structure. This specification details how to implement a GitHub-compatible code explorer with 100% look and feel parity.

## Architecture

### Components

```
pkg/git/                    # Git repository interaction
├── repository.go          # Repository operations
├── tree.go                # Tree/directory operations
├── blob.go                # File content operations
├── commit.go              # Commit operations
├── ref.go                 # Branch/tag references
├── language.go            # Language detection
└── git_test.go            # Unit tests

pkg/seed/github/           # GitHub data seeding
├── client.go              # GitHub API client
├── issues.go              # Fetch issues
├── pulls.go               # Fetch pull requests
├── comments.go            # Fetch comments
└── seed.go                # Main seeding logic

feature/repos/             # Extended repository model
└── api.go                 # Add Language, LanguageColor fields

app/web/handler/           # HTTP handlers
└── code.go                # Code explorer handlers

assets/views/default/pages/
├── repo_home.html         # Updated with file tree
├── repo_tree.html         # Directory listing page
├── repo_blob.html         # File viewer page
└── repo_blame.html        # Blame view page
```

## Data Models

### Git Object Models (pkg/git)

```go
// TreeEntry represents a file or directory in a git tree
type TreeEntry struct {
    Name        string    // File or directory name
    Path        string    // Full path from repository root
    Type        string    // "blob" (file) or "tree" (directory)
    Mode        string    // Git file mode (100644, 040000, etc.)
    SHA         string    // Object SHA
    Size        int64     // Size in bytes (only for blobs)
    URL         string    // URL to view this entry
}

// Tree represents a directory listing
type Tree struct {
    SHA         string       // Tree object SHA
    Path        string       // Current path
    Entries     []TreeEntry  // Directory entries
    TotalCount  int          // Total number of entries
}

// Blob represents file content
type Blob struct {
    SHA         string    // Blob SHA
    Path        string    // File path
    Name        string    // File name
    Size        int64     // Size in bytes
    Content     string    // File content (text files)
    IsBinary    bool      // Whether file is binary
    Encoding    string    // Content encoding (base64 for binary)
    Language    string    // Detected programming language
    Lines       int       // Number of lines (text files)
    SLOC        int       // Source lines of code
}

// Commit represents a git commit
type Commit struct {
    SHA         string    // Full commit SHA
    ShortSHA    string    // Short SHA (first 7 chars)
    Message     string    // Commit message
    Title       string    // First line of message
    Body        string    // Rest of message
    Author      Author    // Author info
    Committer   Author    // Committer info
    Parents     []string  // Parent commit SHAs
    CreatedAt   time.Time // Commit timestamp
}

// Author represents a commit author
type Author struct {
    Name  string
    Email string
    Date  time.Time
}

// Reference represents a branch or tag
type Reference struct {
    Name      string    // Reference name (e.g., "main", "v1.0.0")
    Type      string    // "branch" or "tag"
    SHA       string    // Commit SHA it points to
    IsDefault bool      // Is this the default branch?
}

// BlameHunk represents a section of a file with attribution
type BlameHunk struct {
    StartLine   int       // Starting line number (1-indexed)
    EndLine     int       // Ending line number
    Commit      *Commit   // Commit that introduced these lines
    Lines       []string  // Line contents
}

// FileLastCommit holds the last commit info for a file
type FileLastCommit struct {
    Path        string
    Commit      *Commit
    RelativeAge string    // "2 days ago", "last month"
}
```

### Extended Repository Model

```go
// Repository extended fields (add to feature/repos/api.go)
type Repository struct {
    // ... existing fields ...

    // Language detection
    Language      string `json:"language"`       // Primary language
    LanguageColor string `json:"language_color"` // Color for language dot

    // Additional computed fields
    DefaultBranchCommit string `json:"default_branch_commit,omitempty"` // Latest commit SHA
}

// LanguageStats represents language breakdown
type LanguageStats struct {
    Name       string  `json:"name"`
    Color      string  `json:"color"`
    Percentage float64 `json:"percentage"`
    Bytes      int64   `json:"bytes"`
}
```

## UI Components

### 1. Repository Header (Already Implemented)

- Repository path: `owner/repo-name`
- Visibility badge: Public/Private
- Star button with count
- Fork button with count

### 2. Branch Selector Dropdown

```html
<details class="dropdown details-reset details-overlay">
  <summary class="btn btn-sm">
    <svg class="octicon octicon-git-branch">...</svg>
    <span>main</span>
    <span class="dropdown-caret"></span>
  </summary>
  <div class="select-menu-modal">
    <div class="select-menu-header">
      <span class="select-menu-title">Switch branches/tags</span>
    </div>
    <div class="select-menu-tabs">
      <button class="select-menu-tab" data-tab="branches">Branches</button>
      <button class="select-menu-tab" data-tab="tags">Tags</button>
    </div>
    <div class="select-menu-filter">
      <input type="text" placeholder="Find or create a branch...">
    </div>
    <div class="select-menu-list">
      <!-- Branch items -->
      <a class="select-menu-item selected" href="...">
        <svg class="octicon octicon-check">...</svg>
        main
        <span class="label">default</span>
      </a>
    </div>
  </div>
</details>
```

### 3. Breadcrumb Navigation

```html
<nav class="file-navigation">
  <div class="breadcrumb">
    <a href="/owner/repo">repo-name</a>
    <span class="breadcrumb-separator">/</span>
    <a href="/owner/repo/tree/main/src">src</a>
    <span class="breadcrumb-separator">/</span>
    <span class="breadcrumb-current">components</span>
  </div>
</nav>
```

### 4. Action Buttons Row

```html
<div class="file-actions d-flex gap-2">
  <button class="btn btn-sm">
    <svg>...</svg> Go to file
  </button>
  <details class="dropdown">
    <summary class="btn btn-sm">
      Add file <span class="dropdown-caret"></span>
    </summary>
    <ul class="dropdown-menu">
      <li><a href="#">Create new file</a></li>
      <li><a href="#">Upload files</a></li>
    </ul>
  </details>
  <details class="dropdown">
    <summary class="btn btn-sm btn-primary">
      <svg>...</svg> Code <span class="dropdown-caret"></span>
    </summary>
    <div class="dropdown-menu">
      <div class="dropdown-header">Clone</div>
      <div class="clone-options">
        <button class="btn-clipboard" data-clipboard="https://...">HTTPS</button>
        <button class="btn-clipboard" data-clipboard="git@...">SSH</button>
      </div>
    </div>
  </details>
</div>
```

### 5. File Tree Table

```html
<div class="Box">
  <!-- Latest commit banner -->
  <div class="Box-header d-flex items-center">
    <a class="avatar avatar-2" href="/user">
      <img src="..." alt="@username">
    </a>
    <div class="flex-1">
      <a href="/owner/repo/commit/abc123" class="text-bold">
        Commit message title
      </a>
    </div>
    <div class="text-muted text-sm">
      <a href="/owner/repo/commit/abc123">abc123f</a>
      <span>2 days ago</span>
    </div>
    <a href="/owner/repo/commits/main" class="btn btn-sm">
      <svg>...</svg> 142 commits
    </a>
  </div>

  <!-- File listing -->
  <div class="Box-body p-0">
    <table class="files-list">
      <tbody>
        <!-- Directory row -->
        <tr class="file-row">
          <td class="icon">
            <svg class="octicon octicon-file-directory-fill color-fg-muted">...</svg>
          </td>
          <td class="name">
            <a href="/owner/repo/tree/main/src">src</a>
          </td>
          <td class="message">
            <a href="/owner/repo/commit/abc123">Add source files</a>
          </td>
          <td class="age">
            <span>2 days ago</span>
          </td>
        </tr>

        <!-- File row -->
        <tr class="file-row">
          <td class="icon">
            <svg class="octicon octicon-file">...</svg>
          </td>
          <td class="name">
            <a href="/owner/repo/blob/main/README.md">README.md</a>
          </td>
          <td class="message">
            <a href="/owner/repo/commit/def456">Update documentation</a>
          </td>
          <td class="age">
            <span>yesterday</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</div>
```

### 6. File Viewer (Blob View)

```html
<div class="Box">
  <!-- File header -->
  <div class="Box-header d-flex items-center justify-between">
    <div class="d-flex items-center gap-2">
      <span class="text-bold">app.go</span>
      <span class="text-muted">|</span>
      <span class="text-muted">189 lines (157 sloc)</span>
      <span class="text-muted">|</span>
      <span class="text-muted">4.93 KB</span>
    </div>
    <div class="d-flex gap-2">
      <button class="btn btn-sm" title="Copy raw content">
        <svg>...</svg>
      </button>
      <a class="btn btn-sm" href="..." title="Raw">Raw</a>
      <a class="btn btn-sm" href="..." title="Blame">Blame</a>
      <button class="btn btn-sm" title="Download">
        <svg>...</svg>
      </button>
      <button class="btn btn-sm" title="Edit">
        <svg>...</svg>
      </button>
      <button class="btn btn-sm" title="Delete">
        <svg>...</svg>
      </button>
    </div>
  </div>

  <!-- Code content -->
  <div class="Box-body p-0">
    <table class="highlight">
      <tbody>
        <tr class="code-line" id="L1">
          <td class="line-number" data-line="1">
            <a href="#L1">1</a>
          </td>
          <td class="code-content">
            <span class="pl-k">package</span> mizu
          </td>
        </tr>
        <!-- More lines... -->
      </tbody>
    </table>
  </div>
</div>
```

### 7. Symbols Panel (Code Navigation)

```html
<div class="symbols-pane">
  <div class="symbols-header">
    <span>Symbols</span>
    <input type="text" placeholder="Filter symbols...">
  </div>
  <div class="symbols-list">
    <div class="symbol-group">
      <span class="symbol-group-title">Functions</span>
      <a class="symbol-item" href="#L25">
        <svg class="symbol-icon">...</svg>
        <span>New</span>
      </a>
      <a class="symbol-item" href="#L50">
        <svg class="symbol-icon">...</svg>
        <span>Listen</span>
      </a>
    </div>
    <div class="symbol-group">
      <span class="symbol-group-title">Types</span>
      <a class="symbol-item" href="#L10">
        <svg class="symbol-icon">...</svg>
        <span>App</span>
      </a>
    </div>
  </div>
</div>
```

## URL Routing

### Route Patterns

```
# Tree (directory) view
GET /{owner}/{repo}/tree/{ref}/{path...}
GET /{owner}/{repo}/tree/{ref}           # Root of tree

# Blob (file) view
GET /{owner}/{repo}/blob/{ref}/{path...}

# Raw file content
GET /{owner}/{repo}/raw/{ref}/{path...}

# Blame view
GET /{owner}/{repo}/blame/{ref}/{path...}

# Commits
GET /{owner}/{repo}/commits/{ref}        # Commit history
GET /{owner}/{repo}/commits/{ref}/{path} # Commits for file
GET /{owner}/{repo}/commit/{sha}         # Single commit

# Branches and tags
GET /{owner}/{repo}/branches             # List branches
GET /{owner}/{repo}/tags                 # List tags
```

### Route Examples

```
/go-mizu/mizu/tree/main               # Root directory
/go-mizu/mizu/tree/main/cmd           # Subdirectory
/go-mizu/mizu/blob/main/app.go        # View file
/go-mizu/mizu/raw/main/app.go         # Raw file
/go-mizu/mizu/blame/main/app.go       # Blame view
/go-mizu/mizu/tree/v1.0.0             # Tag tree
/go-mizu/mizu/blob/abc123/app.go      # Specific commit
```

## pkg/git Implementation

### Repository Interface

```go
package git

import (
    "context"
    "io"
)

// Repository provides git operations for a repository
type Repository interface {
    // References
    ListBranches(ctx context.Context) ([]Reference, error)
    ListTags(ctx context.Context) ([]Reference, error)
    GetDefaultBranch(ctx context.Context) (string, error)
    ResolveRef(ctx context.Context, ref string) (string, error) // Returns commit SHA

    // Tree operations
    GetTree(ctx context.Context, ref, path string) (*Tree, error)
    GetTreeRecursive(ctx context.Context, ref string) (*Tree, error)

    // Blob operations
    GetBlob(ctx context.Context, ref, path string) (*Blob, error)
    GetBlobRaw(ctx context.Context, ref, path string) (io.ReadCloser, error)

    // Commit operations
    GetCommit(ctx context.Context, sha string) (*Commit, error)
    GetLatestCommit(ctx context.Context, ref string) (*Commit, error)
    GetCommitHistory(ctx context.Context, ref string, limit int) ([]Commit, error)
    GetFileLastCommit(ctx context.Context, ref, path string) (*FileLastCommit, error)
    GetTreeLastCommits(ctx context.Context, ref, path string) (map[string]*FileLastCommit, error)

    // Blame
    GetBlame(ctx context.Context, ref, path string) ([]BlameHunk, error)

    // Stats
    GetLanguageStats(ctx context.Context, ref string) ([]LanguageStats, error)
    GetCommitCount(ctx context.Context, ref string) (int, error)
    GetContributorCount(ctx context.Context) (int, error)
}

// Open opens a git repository at the given path
func Open(path string) (Repository, error)

// Clone clones a repository from a URL
func Clone(ctx context.Context, url, destPath string) (Repository, error)
```

### Language Detection

```go
// DetectLanguage returns the primary language of a file based on extension
func DetectLanguage(filename string) string

// LanguageColor returns the color associated with a language
func LanguageColor(language string) string

// Common language mappings
var languageColors = map[string]string{
    "Go":         "#00ADD8",
    "JavaScript": "#f1e05a",
    "TypeScript": "#3178c6",
    "Python":     "#3572A5",
    "Rust":       "#dea584",
    "Java":       "#b07219",
    "C":          "#555555",
    "C++":        "#f34b7d",
    "Ruby":       "#701516",
    "PHP":        "#4F5D95",
    "HTML":       "#e34c26",
    "CSS":        "#563d7c",
    "Shell":      "#89e051",
    "Markdown":   "#083fa1",
}
```

## CSS Styling

### File Icons

```css
.octicon-file-directory-fill {
  color: #54aeff;  /* Directory blue */
}

.octicon-file {
  color: #656d76;  /* File gray */
}

.octicon-file-code {
  color: #656d76;
}

.octicon-file-media {
  color: #bf8700;  /* Media yellow */
}

.octicon-file-zip {
  color: #7d4e00;  /* Archive brown */
}
```

### File Table

```css
.files-list {
  width: 100%;
  border-collapse: collapse;
}

.file-row {
  border-top: 1px solid var(--color-border-muted);
}

.file-row:hover {
  background-color: var(--color-canvas-subtle);
}

.file-row td {
  padding: 8px 16px;
  vertical-align: middle;
}

.file-row .icon {
  width: 32px;
  text-align: center;
}

.file-row .name {
  font-weight: 600;
}

.file-row .name a {
  color: var(--color-fg-default);
}

.file-row .message {
  color: var(--color-fg-muted);
  max-width: 500px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-row .age {
  color: var(--color-fg-muted);
  white-space: nowrap;
  text-align: right;
}
```

### Code Viewer

```css
.highlight {
  width: 100%;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 20px;
}

.code-line:hover {
  background-color: var(--color-canvas-subtle);
}

.line-number {
  width: 1%;
  min-width: 50px;
  padding: 0 16px;
  text-align: right;
  color: var(--color-fg-subtle);
  user-select: none;
}

.line-number a {
  color: inherit;
}

.line-number a:hover {
  color: var(--color-fg-default);
}

.code-content {
  padding: 0 16px;
  white-space: pre;
  overflow-x: auto;
}

/* Syntax highlighting classes */
.pl-k { color: #cf222e; }  /* Keyword */
.pl-c { color: #6e7781; }  /* Comment */
.pl-s { color: #0a3069; }  /* String */
.pl-en { color: #8250df; } /* Entity name (function) */
.pl-c1 { color: #0550ae; } /* Constant */
.pl-v { color: #953800; }  /* Variable */
.pl-pds { color: #0a3069; } /* String delimiter */
```

### Branch Selector

```css
.branch-selector {
  position: relative;
}

.branch-selector-menu {
  position: absolute;
  top: 100%;
  left: 0;
  width: 300px;
  background: var(--color-canvas-overlay);
  border: 1px solid var(--color-border-default);
  border-radius: 6px;
  box-shadow: var(--color-shadow-large);
  z-index: 100;
}

.branch-selector-tabs {
  display: flex;
  border-bottom: 1px solid var(--color-border-default);
}

.branch-selector-tab {
  flex: 1;
  padding: 8px;
  text-align: center;
  background: transparent;
  border: none;
  cursor: pointer;
}

.branch-selector-tab.selected {
  font-weight: 600;
  border-bottom: 2px solid var(--color-accent-fg);
}

.branch-item {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  cursor: pointer;
}

.branch-item:hover {
  background: var(--color-canvas-subtle);
}

.branch-item.selected .octicon-check {
  visibility: visible;
}

.branch-item .octicon-check {
  visibility: hidden;
  margin-right: 8px;
}
```

## API Endpoints

### Tree API

```
GET /api/v1/repos/{owner}/{repo}/git/trees/{ref}
GET /api/v1/repos/{owner}/{repo}/git/trees/{ref}?path={path}

Response:
{
  "sha": "abc123...",
  "path": "src",
  "entries": [
    {
      "name": "main.go",
      "path": "src/main.go",
      "type": "blob",
      "mode": "100644",
      "sha": "def456...",
      "size": 1234
    },
    {
      "name": "utils",
      "path": "src/utils",
      "type": "tree",
      "mode": "040000",
      "sha": "ghi789..."
    }
  ]
}
```

### Blob API

```
GET /api/v1/repos/{owner}/{repo}/git/blobs/{ref}/{path}

Response:
{
  "sha": "abc123...",
  "path": "src/main.go",
  "name": "main.go",
  "size": 1234,
  "content": "package main\n...",
  "encoding": "utf-8",
  "is_binary": false,
  "language": "Go",
  "lines": 50,
  "sloc": 42
}
```

### References API

```
GET /api/v1/repos/{owner}/{repo}/branches
GET /api/v1/repos/{owner}/{repo}/tags

Response:
{
  "refs": [
    {
      "name": "main",
      "type": "branch",
      "sha": "abc123...",
      "is_default": true
    }
  ]
}
```

### Commits API

```
GET /api/v1/repos/{owner}/{repo}/commits?ref={ref}&path={path}

Response:
{
  "commits": [
    {
      "sha": "abc123...",
      "short_sha": "abc123f",
      "message": "Add new feature",
      "author": {
        "name": "John Doe",
        "email": "john@example.com",
        "date": "2024-01-15T10:30:00Z"
      },
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total_count": 142
}
```

## pkg/seed/github Implementation

### GitHub API Client

```go
package github

import (
    "context"
    "encoding/json"
    "net/http"
)

// Client is a GitHub API client
type Client struct {
    httpClient *http.Client
    baseURL    string
    token      string  // Optional personal access token
}

// NewClient creates a new GitHub client
func NewClient(token string) *Client

// Repository represents a GitHub repository
type Repository struct {
    ID              int64     `json:"id"`
    Name            string    `json:"name"`
    FullName        string    `json:"full_name"`
    Description     string    `json:"description"`
    Private         bool      `json:"private"`
    Fork            bool      `json:"fork"`
    Language        string    `json:"language"`
    StargazersCount int       `json:"stargazers_count"`
    ForksCount      int       `json:"forks_count"`
    WatchersCount   int       `json:"watchers_count"`
    OpenIssuesCount int       `json:"open_issues_count"`
    DefaultBranch   string    `json:"default_branch"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    PushedAt        time.Time `json:"pushed_at"`
}

// Issue represents a GitHub issue
type Issue struct {
    ID        int64     `json:"id"`
    Number    int       `json:"number"`
    Title     string    `json:"title"`
    Body      string    `json:"body"`
    State     string    `json:"state"`
    User      User      `json:"user"`
    Labels    []Label   `json:"labels"`
    Assignees []User    `json:"assignees"`
    Milestone *Milestone `json:"milestone"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    ClosedAt  *time.Time `json:"closed_at"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
    ID        int64     `json:"id"`
    Number    int       `json:"number"`
    Title     string    `json:"title"`
    Body      string    `json:"body"`
    State     string    `json:"state"`
    User      User      `json:"user"`
    Head      Branch    `json:"head"`
    Base      Branch    `json:"base"`
    Merged    bool      `json:"merged"`
    MergedAt  *time.Time `json:"merged_at"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Comment represents a GitHub comment
type Comment struct {
    ID        int64     `json:"id"`
    Body      string    `json:"body"`
    User      User      `json:"user"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// User represents a GitHub user
type User struct {
    ID        int64  `json:"id"`
    Login     string `json:"login"`
    AvatarURL string `json:"avatar_url"`
}

// Label represents a GitHub label
type Label struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Color string `json:"color"`
}

// Milestone represents a GitHub milestone
type Milestone struct {
    ID          int64     `json:"id"`
    Number      int       `json:"number"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    State       string    `json:"state"`
    DueOn       *time.Time `json:"due_on"`
}

// Branch represents a git branch reference
type Branch struct {
    Ref  string `json:"ref"`
    SHA  string `json:"sha"`
    Repo struct {
        Name     string `json:"name"`
        FullName string `json:"full_name"`
    } `json:"repo"`
}
```

### Seeding Functions

```go
// Seed populates the database with data from a GitHub repository
func Seed(ctx context.Context, owner, repo string, stores Stores) error

// FetchRepository fetches repository metadata
func (c *Client) FetchRepository(ctx context.Context, owner, repo string) (*Repository, error)

// FetchIssues fetches all issues from a repository
func (c *Client) FetchIssues(ctx context.Context, owner, repo string) ([]Issue, error)

// FetchPullRequests fetches all pull requests from a repository
func (c *Client) FetchPullRequests(ctx context.Context, owner, repo string) ([]PullRequest, error)

// FetchComments fetches comments for an issue or PR
func (c *Client) FetchComments(ctx context.Context, owner, repo string, number int) ([]Comment, error)

// FetchLabels fetches all labels from a repository
func (c *Client) FetchLabels(ctx context.Context, owner, repo string) ([]Label, error)

// FetchMilestones fetches all milestones from a repository
func (c *Client) FetchMilestones(ctx context.Context, owner, repo string) ([]Milestone, error)
```

## Template Fixes Required

### Repository Model Updates

Add to `feature/repos/api.go`:

```go
type Repository struct {
    // ... existing fields ...

    // Language detection (computed from git analysis)
    Language      string `json:"language,omitempty"`
    LanguageColor string `json:"language_color,omitempty"`
}
```

### Template Variable Fixes

In `pages/home.html` line 88-98:
- Change `.Language` to `.Language` (after adding to model)
- Change `.Stars` to `.StarCount`

### View Models for Templates

```go
// RepoHomeView is the view model for repo home page
type RepoHomeView struct {
    User       *users.User
    Owner      *users.User
    Repository *repos.Repository
    IsStarred  bool
    CanEdit    bool

    // Code explorer additions
    Tree       *git.Tree
    LatestCommit *git.Commit
    CommitCount  int
    Branches     []git.Reference
    Tags         []git.Reference
    CurrentRef   string
    CurrentPath  string
    Readme       string  // Rendered README content
}

// RepoTreeView is the view model for tree (directory) view
type RepoTreeView struct {
    User       *users.User
    Owner      *users.User
    Repository *repos.Repository
    IsStarred  bool
    CanEdit    bool

    Tree          *git.Tree
    LatestCommit  *git.Commit
    FileCommits   map[string]*git.FileLastCommit
    CommitCount   int
    Branches      []git.Reference
    Tags          []git.Reference
    CurrentRef    string
    CurrentPath   string
    Breadcrumbs   []Breadcrumb
}

// RepoBlobView is the view model for blob (file) view
type RepoBlobView struct {
    User       *users.User
    Owner      *users.User
    Repository *repos.Repository
    IsStarred  bool
    CanEdit    bool

    Blob         *git.Blob
    LatestCommit *git.Commit
    Branches     []git.Reference
    Tags         []git.Reference
    CurrentRef   string
    CurrentPath  string
    Breadcrumbs  []Breadcrumb
    SyntaxHTML   string  // Pre-rendered syntax highlighted content
}

// Breadcrumb represents a navigation breadcrumb
type Breadcrumb struct {
    Name string
    Path string
    URL  string
}
```

## Testing Strategy

### Template Rendering Tests

```go
func TestTemplateRendering(t *testing.T) {
    tests := []struct {
        name     string
        template string
        data     interface{}
        wantErr  bool
    }{
        {
            name:     "home page unauthenticated",
            template: "home",
            data:     &HomeView{Repositories: []*repos.Repository{}},
        },
        {
            name:     "home page authenticated",
            template: "home",
            data:     &HomeView{User: &users.User{}, Repositories: []*repos.Repository{}},
        },
        {
            name:     "repo home page",
            template: "repo_home",
            data:     &RepoHomeView{...},
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            err := templates.Execute(&buf, tt.template, tt.data)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

### Git Package Tests

```go
func TestGitOperations(t *testing.T) {
    // Use the local mizu repository as test data
    repo, err := git.Open("/Users/apple/github/go-mizu/mizu")
    require.NoError(t, err)

    t.Run("list branches", func(t *testing.T) {
        branches, err := repo.ListBranches(context.Background())
        require.NoError(t, err)
        require.NotEmpty(t, branches)
    })

    t.Run("get tree", func(t *testing.T) {
        tree, err := repo.GetTree(context.Background(), "main", "")
        require.NoError(t, err)
        require.NotEmpty(t, tree.Entries)
    })

    t.Run("get blob", func(t *testing.T) {
        blob, err := repo.GetBlob(context.Background(), "main", "app.go")
        require.NoError(t, err)
        require.Equal(t, "app.go", blob.Name)
        require.Equal(t, "Go", blob.Language)
    })
}
```

## Implementation Order

1. **Fix Template Errors** (Priority: Critical)
   - Add `Language` and `LanguageColor` fields to Repository model
   - Fix `.Stars` -> `.StarCount` in templates
   - Add template validation tests

2. **Implement pkg/git** (Priority: High)
   - Repository opening and basic operations
   - Tree/blob reading
   - Commit history
   - Reference listing
   - Language detection

3. **Update Repository Home Page**
   - Add file tree to repo_home.html
   - Branch selector dropdown
   - Latest commit banner

4. **Create Tree and Blob Pages**
   - repo_tree.html for directory listings
   - repo_blob.html for file viewing
   - Syntax highlighting integration

5. **Add Routing**
   - /tree/{ref}/{path} routes
   - /blob/{ref}/{path} routes
   - /raw/{ref}/{path} routes

6. **Implement pkg/seed/github**
   - GitHub API client
   - Issue/PR fetching
   - Data transformation and storage

7. **Testing**
   - Template rendering tests
   - Git operations tests
   - Integration tests

## File Icons by Extension

```go
var fileIcons = map[string]string{
    // Directories
    "directory":  "file-directory-fill",

    // Code files
    ".go":        "file-code",
    ".js":        "file-code",
    ".ts":        "file-code",
    ".py":        "file-code",
    ".rb":        "file-code",
    ".java":      "file-code",
    ".c":         "file-code",
    ".cpp":       "file-code",
    ".h":         "file-code",
    ".rs":        "file-code",
    ".php":       "file-code",

    // Documentation
    ".md":        "file",
    ".txt":       "file",
    ".rst":       "file",

    // Config
    ".json":      "file-code",
    ".yaml":      "file-code",
    ".yml":       "file-code",
    ".toml":      "file-code",
    ".xml":       "file-code",

    // Media
    ".png":       "file-media",
    ".jpg":       "file-media",
    ".jpeg":      "file-media",
    ".gif":       "file-media",
    ".svg":       "file-media",
    ".ico":       "file-media",

    // Archives
    ".zip":       "file-zip",
    ".tar":       "file-zip",
    ".gz":        "file-zip",
    ".rar":       "file-zip",

    // Default
    "":           "file",
}
```

## Performance Considerations

1. **Tree Last Commits**: Fetching last commit for each file in a tree can be expensive. Consider:
   - Lazy loading via AJAX
   - Caching commit information
   - Limiting to visible entries with pagination

2. **Large Files**: Binary and very large text files should:
   - Show a "View Raw" link instead of inline content
   - Use streaming for raw downloads
   - Limit syntax highlighting to reasonable sizes (<1MB)

3. **Repository Caching**: Cache frequently accessed data:
   - Branch list
   - Default branch
   - Root tree structure
   - README content

## Security Considerations

1. **Path Traversal**: Validate all path parameters to prevent directory traversal attacks
2. **Symlink Following**: Be careful with symbolic links that might point outside the repository
3. **Private Repositories**: Ensure proper authentication for private repo access
4. **Rate Limiting**: Implement rate limiting for expensive git operations
