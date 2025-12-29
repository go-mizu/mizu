# GitHub Repository Seeder Specification

## Overview

This specification details the implementation of `pkg/seed/github` for importing GitHub repository data (issues, pull requests, comments, labels, milestones, etc.) into GitHome's DuckDB store for offline viewing.

## Goals

1. **Complete Data Import**: Import all relevant repository metadata from GitHub API
2. **Offline Viewing**: Enable full offline access to repository issues, PRs, comments
3. **Rate Limit Handling**: Graceful handling of GitHub API rate limits
4. **Incremental Sync**: Support for updating existing data (future enhancement)
5. **CLI Integration**: Add `githome seed github` command

## Data to Import

### Primary Entities
| Entity | GitHub API Endpoint | GitHome Store |
|--------|---------------------|---------------|
| Repository | `GET /repos/{owner}/{repo}` | `repos.Repository` |
| Issues | `GET /repos/{owner}/{repo}/issues` | `issues.Issue` |
| Pull Requests | `GET /repos/{owner}/{repo}/pulls` | `pulls.PullRequest` |
| Issue Comments | `GET /repos/{owner}/{repo}/issues/{number}/comments` | `comments.IssueComment` |
| PR Review Comments | `GET /repos/{owner}/{repo}/pulls/{number}/comments` | `pulls.ReviewComment` |
| Labels | `GET /repos/{owner}/{repo}/labels` | `labels.Label` |
| Milestones | `GET /repos/{owner}/{repo}/milestones` | `milestones.Milestone` |

### Secondary Entities (imported as referenced)
| Entity | Source | GitHome Store |
|--------|--------|---------------|
| Users | From issues/PRs/comments | `users.User` |
| Organizations | From repository owner | `orgs.Organization` |

## Package Structure

```
pkg/seed/github/
├── seeder.go       # Main seeder orchestration
├── client.go       # GitHub API client wrapper
├── mappers.go      # GitHub -> GitHome type mappers
├── config.go       # Configuration types
└── seeder_test.go  # Comprehensive tests
```

## Configuration

```go
// Config contains GitHub seeder configuration
type Config struct {
    // Required
    Owner       string  // Repository owner (user or org)
    Repo        string  // Repository name
    Token       string  // GitHub personal access token (optional, higher rate limits)

    // Optional
    BaseURL     string  // GitHub API base URL (for GitHub Enterprise)
    AdminUserID int64   // Admin user ID for ownership (defaults to 1)
    IsPublic    bool    // Make imported repo public (default: true)

    // Import options
    ImportIssues    bool  // Import issues (default: true)
    ImportPRs       bool  // Import pull requests (default: true)
    ImportComments  bool  // Import comments (default: true)
    ImportLabels    bool  // Import labels (default: true)
    ImportMilestones bool // Import milestones (default: true)

    // Limits
    MaxIssues       int   // Max issues to import (0 = all)
    MaxPRs          int   // Max PRs to import (0 = all)
    MaxCommentsPerItem int // Max comments per issue/PR (0 = all)
}
```

## Result Structure

```go
// Result contains the result of a GitHub seeding operation
type Result struct {
    // Counts
    RepoCreated        bool
    OrgCreated         bool
    UsersCreated       int
    IssuesCreated      int
    PRsCreated         int
    CommentsCreated    int
    LabelsCreated      int
    MilestonesCreated  int

    // Skipped (already exist)
    IssuesSkipped      int
    PRsSkipped         int
    CommentsSkipped    int

    // Errors
    Errors             []error

    // Rate limit info
    RateLimitRemaining int
    RateLimitReset     time.Time
}
```

## Implementation Details

### 1. GitHub API Client (`client.go`)

```go
// Client wraps GitHub API interactions
type Client struct {
    httpClient  *http.Client
    baseURL     string
    token       string
    rateLimiter *rate.Limiter
}

// Key methods:
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
func (c *Client) ListIssues(ctx context.Context, owner, repo string, opts *ListOptions) ([]*Issue, error)
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, opts *ListOptions) ([]*PullRequest, error)
func (c *Client) ListIssueComments(ctx context.Context, owner, repo string, number int, opts *ListOptions) ([]*Comment, error)
func (c *Client) ListPRComments(ctx context.Context, owner, repo string, number int, opts *ListOptions) ([]*ReviewComment, error)
func (c *Client) ListLabels(ctx context.Context, owner, repo string, opts *ListOptions) ([]*Label, error)
func (c *Client) ListMilestones(ctx context.Context, owner, repo string, opts *ListOptions) ([]*Milestone, error)
```

### 2. Type Mappers (`mappers.go`)

Convert GitHub API types to GitHome types:

```go
func mapRepository(gh *ghRepo, ownerID int64) *repos.Repository
func mapIssue(gh *ghIssue, repoID int64, creatorID int64) *issues.Issue
func mapPullRequest(gh *ghPR, repoID int64, creatorID int64) *pulls.PullRequest
func mapIssueComment(gh *ghComment, issueID, repoID, creatorID int64) *comments.IssueComment
func mapLabel(gh *ghLabel, repoID int64) *labels.Label
func mapMilestone(gh *ghMilestone, repoID int64, creatorID int64) *milestones.Milestone
func mapUser(gh *ghUser) *users.User
```

### 3. Main Seeder (`seeder.go`)

```go
type Seeder struct {
    db         *sql.DB
    client     *Client
    config     Config

    // Stores
    usersStore      *duckdb.UsersStore
    orgsStore       *duckdb.OrgsStore
    reposStore      *duckdb.ReposStore
    issuesStore     *duckdb.IssuesStore
    pullsStore      *duckdb.PullsStore
    commentsStore   *duckdb.CommentsStore
    labelsStore     *duckdb.LabelsStore
    milestonesStore *duckdb.MilestonesStore

    // Cache
    userCache map[string]int64  // login -> id
    labelCache map[string]int64 // name -> id
}

func NewSeeder(db *sql.DB, config Config) *Seeder
func (s *Seeder) Seed(ctx context.Context) (*Result, error)
```

### Seeding Process

1. **Initialize**: Create HTTP client, set up rate limiter
2. **Fetch Repository**: Get repository metadata
3. **Ensure Owner**: Create user/org if needed
4. **Create Repository**: Create repo in GitHome
5. **Import Labels**: Fetch and create all labels
6. **Import Milestones**: Fetch and create all milestones
7. **Import Issues**:
   - Fetch issues (both open and closed)
   - Map users (create if needed)
   - Create issues with labels and milestone references
   - Fetch and import comments for each issue
8. **Import Pull Requests**:
   - Fetch PRs (both open and closed)
   - Map users (create if needed)
   - Create PRs with labels and milestone references
   - Fetch and import review comments for each PR
9. **Return Result**: Aggregate counts and errors

### Rate Limiting

GitHub API has the following rate limits:
- **Unauthenticated**: 60 requests/hour
- **Authenticated**: 5,000 requests/hour

Strategy:
- Use `golang.org/x/time/rate` for client-side limiting
- Check `X-RateLimit-Remaining` header
- Wait for reset if approaching limit
- Log warning when < 100 requests remaining

### Error Handling

- Log and continue on individual item failures
- Aggregate errors in result
- Abort on critical failures (auth, repo not found)
- Support partial completion

## CLI Integration

### Command: `githome seed github`

```bash
githome seed github <owner>/<repo> [flags]

Flags:
  --token string     GitHub personal access token (or GITHUB_TOKEN env)
  --base-url string  GitHub API base URL (for GitHub Enterprise)
  --public           Make repository public (default: true)
  --max-issues int   Maximum issues to import (0 = all)
  --max-prs int      Maximum PRs to import (0 = all)
  --no-comments      Skip importing comments
  --no-prs           Skip importing pull requests

Examples:
  githome seed github golang/go
  githome seed github golang/go --token $GITHUB_TOKEN
  githome seed github golang/go --max-issues 100 --max-prs 50
  githome seed github mycompany/private-repo --base-url https://github.mycompany.com/api/v3
```

### CLI Implementation (`cli/seed.go`)

Add `newSeedGitHubCmd()` following the existing pattern from `newSeedLocalCmd()`.

## Testing Strategy

### Unit Tests
- Test type mappers with fixtures
- Test pagination handling
- Test error scenarios

### Integration Tests
Using `golang/go` as test repository:
- Test fetching real repository metadata
- Test fetching issues with pagination
- Test fetching PRs
- Test rate limit handling
- Test with and without authentication

### Test Cases

```go
func TestSeeder_SeedGolangGo(t *testing.T)
func TestSeeder_SeedWithMaxLimits(t *testing.T)
func TestSeeder_SeedLabelsOnly(t *testing.T)
func TestSeeder_HandleRateLimit(t *testing.T)
func TestSeeder_ResumeAfterError(t *testing.T)
func TestMapper_Issue(t *testing.T)
func TestMapper_PullRequest(t *testing.T)
func TestMapper_Comment(t *testing.T)
func TestClient_Pagination(t *testing.T)
```

## GitHub API Response Types

### Repository
```json
{
  "id": 1234567,
  "node_id": "MDEwOlJlcG9zaXRvcnkx",
  "name": "repo-name",
  "full_name": "owner/repo-name",
  "owner": { "login": "owner", "id": 1, "type": "Organization" },
  "private": false,
  "description": "Description",
  "fork": false,
  "default_branch": "main",
  "has_issues": true,
  "has_projects": true,
  "has_wiki": true,
  "open_issues_count": 100,
  "created_at": "2020-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Issue
```json
{
  "id": 123,
  "number": 1,
  "title": "Issue title",
  "body": "Issue body",
  "state": "open",
  "user": { "login": "username", "id": 1 },
  "labels": [{ "id": 1, "name": "bug", "color": "ff0000" }],
  "milestone": { "id": 1, "number": 1, "title": "v1.0" },
  "assignees": [{ "login": "assignee", "id": 2 }],
  "comments": 5,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T00:00:00Z",
  "closed_at": null
}
```

### Pull Request
```json
{
  "id": 456,
  "number": 10,
  "title": "PR title",
  "body": "PR description",
  "state": "open",
  "user": { "login": "username", "id": 1 },
  "head": { "ref": "feature-branch", "sha": "abc123" },
  "base": { "ref": "main", "sha": "def456" },
  "draft": false,
  "merged": false,
  "mergeable": true,
  "comments": 3,
  "review_comments": 5,
  "commits": 2,
  "additions": 100,
  "deletions": 50,
  "changed_files": 5,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T00:00:00Z"
}
```

## Future Enhancements

1. **Incremental Sync**: Update existing data based on `updated_at`
2. **Webhook Support**: Real-time updates via GitHub webhooks
3. **PR Reviews**: Import full review data (currently only comments)
4. **Reactions**: Import emoji reactions on issues/comments
5. **Events**: Import issue/PR events timeline
6. **GraphQL API**: Use GraphQL for more efficient data fetching

## Dependencies

```go
import (
    "golang.org/x/time/rate"  // Rate limiting
    "encoding/json"           // JSON parsing
    "net/http"               // HTTP client
    "net/url"                // URL handling
)
```

No additional external dependencies required - using standard library for HTTP and JSON.
