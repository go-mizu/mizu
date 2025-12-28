# GitHome Full Features Implementation Plan

## Overview

This document outlines the complete implementation plan for GitHome, a GitHub-like application blueprint. The goal is to implement all 14 missing features and enhance the 3 existing features with proper Store interfaces in DuckDB.

## Current State Analysis

### Implemented Features (3)
| Feature | api.go | service.go | DuckDB Store | Tests |
|---------|--------|------------|--------------|-------|
| users   | ✓      | ✓          | ✓            | ✓     |
| repos   | ✓      | ✓          | ✓            | ✓     |
| issues  | ✓      | ✓          | ✓            | ✓     |

### Missing Features (14 stub directories)
| Feature       | Status | Priority | Description                    |
|---------------|--------|----------|--------------------------------|
| labels        | TODO   | High     | Issue/PR label management      |
| milestones    | TODO   | High     | Release milestone tracking     |
| comments      | TODO   | High     | Generic comments (issues/PRs)  |
| activities    | TODO   | Medium   | Activity feed                  |
| notifications | TODO   | Medium   | User notifications             |
| stars         | TODO   | Medium   | Standalone star management     |
| watches       | TODO   | Medium   | Watch/unwatch repos            |
| orgs          | TODO   | Medium   | Organization management        |
| teams         | TODO   | Medium   | Team management within orgs    |
| collaborators | TODO   | Medium   | Repo collaborator management   |
| pulls         | TODO   | High     | Pull request management        |
| releases      | TODO   | Medium   | Release/tag management         |
| webhooks      | TODO   | Low      | Webhook management             |
| reactions     | TODO   | Low      | Emoji reactions (future)       |

---

## Part 1: Enhance Existing Features

### 1.1 Users Feature Enhancements

**Missing Store Methods:**
- SSH Key management: `CreateSSHKey`, `GetSSHKey`, `ListSSHKeys`, `DeleteSSHKey`, `UpdateSSHKeyUsage`
- API Token management: `CreateAPIToken`, `GetAPIToken`, `GetAPITokenByHash`, `ListAPITokens`, `DeleteAPIToken`, `UpdateAPITokenUsage`

**API Additions:**
```go
// SSH Keys
CreateSSHKey(ctx, userID string, in *CreateSSHKeyIn) (*SSHKey, error)
GetSSHKey(ctx, id string) (*SSHKey, error)
ListSSHKeys(ctx, userID string) ([]*SSHKey, error)
DeleteSSHKey(ctx, id string) error

// API Tokens
CreateAPIToken(ctx, userID string, in *CreateAPITokenIn) (*APIToken, string, error)
ListAPITokens(ctx, userID string) ([]*APIToken, error)
DeleteAPIToken(ctx, id string) error
ValidateAPIToken(ctx, token string) (*User, error)
```

### 1.2 Repos Feature Enhancements

**Missing Store Methods:**
- `ListForks(ctx, repoID string, limit, offset int) ([]*Repository, error)`
- `CountByOwner(ctx, ownerID, ownerType string) (int, error)`

**API Additions:**
- `Transfer(ctx, repoID, newOwnerID string) (*Repository, error)`
- `GetStats(ctx, repoID string) (*RepoStats, error)`

### 1.3 Issues Feature Enhancements

**Missing Store Methods (Comments):**
```go
CreateComment(ctx, c *Comment) error
GetComment(ctx, id string) (*Comment, error)
UpdateComment(ctx, c *Comment) error
DeleteComment(ctx, id string) error
ListComments(ctx, targetType, targetID string, limit, offset int) ([]*Comment, int, error)
```

---

## Part 2: New Features Implementation

### 2.1 Labels Feature

**Package:** `feature/labels`

**Models:**
```go
type Label struct {
    ID          string    `json:"id"`
    RepoID      string    `json:"repo_id"`
    Name        string    `json:"name"`
    Color       string    `json:"color"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
}

type CreateIn struct {
    Name        string `json:"name"`
    Color       string `json:"color"`
    Description string `json:"description"`
}

type UpdateIn struct {
    Name        *string `json:"name,omitempty"`
    Color       *string `json:"color,omitempty"`
    Description *string `json:"description,omitempty"`
}
```

**API Interface:**
```go
type API interface {
    Create(ctx, repoID string, in *CreateIn) (*Label, error)
    GetByID(ctx, id string) (*Label, error)
    GetByName(ctx, repoID, name string) (*Label, error)
    Update(ctx, id string, in *UpdateIn) (*Label, error)
    Delete(ctx, id string) error
    List(ctx, repoID string) ([]*Label, error)
}
```

**Store Interface:**
```go
type Store interface {
    Create(ctx, l *Label) error
    GetByID(ctx, id string) (*Label, error)
    GetByName(ctx, repoID, name string) (*Label, error)
    Update(ctx, l *Label) error
    Delete(ctx, id string) error
    List(ctx, repoID string) ([]*Label, error)
    ListByIDs(ctx, ids []string) ([]*Label, error)
}
```

### 2.2 Milestones Feature

**Package:** `feature/milestones`

**Models:**
```go
type Milestone struct {
    ID            string     `json:"id"`
    RepoID        string     `json:"repo_id"`
    Number        int        `json:"number"`
    Title         string     `json:"title"`
    Description   string     `json:"description"`
    State         string     `json:"state"` // open, closed
    DueDate       *time.Time `json:"due_date,omitempty"`
    OpenIssues    int        `json:"open_issues"`
    ClosedIssues  int        `json:"closed_issues"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    ClosedAt      *time.Time `json:"closed_at,omitempty"`
}
```

**API Interface:**
```go
type API interface {
    Create(ctx, repoID string, in *CreateIn) (*Milestone, error)
    GetByID(ctx, id string) (*Milestone, error)
    GetByNumber(ctx, repoID string, number int) (*Milestone, error)
    Update(ctx, id string, in *UpdateIn) (*Milestone, error)
    Delete(ctx, id string) error
    List(ctx, repoID string, state string) ([]*Milestone, error)
    Close(ctx, id string) error
    Reopen(ctx, id string) error
}
```

### 2.3 Comments Feature

**Package:** `feature/comments`

**Models:**
```go
type Comment struct {
    ID         string    `json:"id"`
    TargetType string    `json:"target_type"` // issue, pull_request
    TargetID   string    `json:"target_id"`
    UserID     string    `json:"user_id"`
    Body       string    `json:"body"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

**API Interface:**
```go
type API interface {
    Create(ctx, targetType, targetID, userID string, in *CreateIn) (*Comment, error)
    GetByID(ctx, id string) (*Comment, error)
    Update(ctx, id string, in *UpdateIn) (*Comment, error)
    Delete(ctx, id string) error
    List(ctx, targetType, targetID string, limit, offset int) ([]*Comment, int, error)
}
```

### 2.4 Activities Feature

**Package:** `feature/activities`

**Models:**
```go
type Activity struct {
    ID         string    `json:"id"`
    ActorID    string    `json:"actor_id"`
    EventType  string    `json:"event_type"`
    RepoID     string    `json:"repo_id,omitempty"`
    TargetType string    `json:"target_type,omitempty"`
    TargetID   string    `json:"target_id,omitempty"`
    Ref        string    `json:"ref,omitempty"`
    RefType    string    `json:"ref_type,omitempty"`
    Payload    string    `json:"payload"`
    IsPublic   bool      `json:"is_public"`
    CreatedAt  time.Time `json:"created_at"`
}

// Event types
const (
    EventPush         = "push"
    EventCreate       = "create"
    EventDelete       = "delete"
    EventFork         = "fork"
    EventStar         = "star"
    EventWatch        = "watch"
    EventIssueOpen    = "issue_open"
    EventIssueClose   = "issue_close"
    EventIssueComment = "issue_comment"
    EventPROpen       = "pr_open"
    EventPRClose      = "pr_close"
    EventPRMerge      = "pr_merge"
    EventPRComment    = "pr_comment"
    EventRelease      = "release"
)
```

**API Interface:**
```go
type API interface {
    Record(ctx, a *Activity) error
    ListByUser(ctx, userID string, limit, offset int) ([]*Activity, error)
    ListByRepo(ctx, repoID string, limit, offset int) ([]*Activity, error)
    ListPublic(ctx, limit, offset int) ([]*Activity, error)
    ListFeed(ctx, userID string, limit, offset int) ([]*Activity, error)
}
```

### 2.5 Notifications Feature

**Package:** `feature/notifications`

**Models:**
```go
type Notification struct {
    ID         string     `json:"id"`
    UserID     string     `json:"user_id"`
    RepoID     string     `json:"repo_id,omitempty"`
    Type       string     `json:"type"`
    ActorID    string     `json:"actor_id,omitempty"`
    TargetType string     `json:"target_type"`
    TargetID   string     `json:"target_id"`
    Title      string     `json:"title"`
    Reason     string     `json:"reason"`
    Unread     bool       `json:"unread"`
    CreatedAt  time.Time  `json:"created_at"`
    UpdatedAt  time.Time  `json:"updated_at"`
    LastReadAt *time.Time `json:"last_read_at,omitempty"`
}

// Reasons
const (
    ReasonAssigned    = "assigned"
    ReasonMentioned   = "mentioned"
    ReasonSubscribed  = "subscribed"
    ReasonAuthor      = "author"
    ReasonReviewReq   = "review_requested"
)
```

**API Interface:**
```go
type API interface {
    Create(ctx, n *Notification) error
    List(ctx, userID string, unreadOnly bool, limit, offset int) ([]*Notification, int, error)
    MarkAsRead(ctx, id string) error
    MarkAllAsRead(ctx, userID string) error
    Delete(ctx, id string) error
    GetUnreadCount(ctx, userID string) (int, error)
}
```

### 2.6 Stars Feature (Standalone)

**Package:** `feature/stars`

**Models:**
```go
type Star struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    RepoID    string    `json:"repo_id"`
    CreatedAt time.Time `json:"created_at"`
}
```

**API Interface:**
```go
type API interface {
    Star(ctx, userID, repoID string) error
    Unstar(ctx, userID, repoID string) error
    IsStarred(ctx, userID, repoID string) (bool, error)
    ListStargazers(ctx, repoID string, limit, offset int) ([]string, int, error)
    ListStarred(ctx, userID string, limit, offset int) ([]string, int, error)
    GetCount(ctx, repoID string) (int, error)
}
```

### 2.7 Watches Feature

**Package:** `feature/watches`

**Models:**
```go
type Watch struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    RepoID    string    `json:"repo_id"`
    Level     string    `json:"level"` // watching, releases_only, ignoring
    CreatedAt time.Time `json:"created_at"`
}

const (
    LevelWatching     = "watching"
    LevelReleasesOnly = "releases_only"
    LevelIgnoring     = "ignoring"
)
```

**API Interface:**
```go
type API interface {
    Watch(ctx, userID, repoID string, level string) error
    Unwatch(ctx, userID, repoID string) error
    GetWatchStatus(ctx, userID, repoID string) (*Watch, error)
    ListWatchers(ctx, repoID string, limit, offset int) ([]string, int, error)
    ListWatching(ctx, userID string, limit, offset int) ([]string, int, error)
    GetCount(ctx, repoID string) (int, error)
}
```

### 2.8 Organizations Feature

**Package:** `feature/orgs`

**Models:**
```go
type Organization struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Slug        string    `json:"slug"`
    DisplayName string    `json:"display_name"`
    Description string    `json:"description"`
    AvatarURL   string    `json:"avatar_url"`
    Location    string    `json:"location"`
    Website     string    `json:"website"`
    Email       string    `json:"email"`
    IsVerified  bool      `json:"is_verified"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Member struct {
    ID        string    `json:"id"`
    OrgID     string    `json:"org_id"`
    UserID    string    `json:"user_id"`
    Role      string    `json:"role"` // owner, admin, member
    CreatedAt time.Time `json:"created_at"`
}

const (
    RoleOwner  = "owner"
    RoleAdmin  = "admin"
    RoleMember = "member"
)
```

**API Interface:**
```go
type API interface {
    Create(ctx, creatorID string, in *CreateIn) (*Organization, error)
    GetByID(ctx, id string) (*Organization, error)
    GetBySlug(ctx, slug string) (*Organization, error)
    Update(ctx, id string, in *UpdateIn) (*Organization, error)
    Delete(ctx, id string) error
    List(ctx, limit, offset int) ([]*Organization, error)

    // Members
    AddMember(ctx, orgID, userID string, role string) error
    RemoveMember(ctx, orgID, userID string) error
    UpdateMemberRole(ctx, orgID, userID string, role string) error
    GetMember(ctx, orgID, userID string) (*Member, error)
    ListMembers(ctx, orgID string, limit, offset int) ([]*Member, error)
    ListUserOrgs(ctx, userID string) ([]*Organization, error)
}
```

### 2.9 Teams Feature

**Package:** `feature/teams`

**Models:**
```go
type Team struct {
    ID          string    `json:"id"`
    OrgID       string    `json:"org_id"`
    Name        string    `json:"name"`
    Slug        string    `json:"slug"`
    Description string    `json:"description"`
    Permission  string    `json:"permission"` // read, triage, write, maintain, admin
    ParentID    string    `json:"parent_id,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type TeamMember struct {
    ID        string    `json:"id"`
    TeamID    string    `json:"team_id"`
    UserID    string    `json:"user_id"`
    Role      string    `json:"role"` // maintainer, member
    CreatedAt time.Time `json:"created_at"`
}

type TeamRepo struct {
    ID         string    `json:"id"`
    TeamID     string    `json:"team_id"`
    RepoID     string    `json:"repo_id"`
    Permission string    `json:"permission"`
    CreatedAt  time.Time `json:"created_at"`
}
```

**API Interface:**
```go
type API interface {
    Create(ctx, orgID string, in *CreateIn) (*Team, error)
    GetByID(ctx, id string) (*Team, error)
    GetBySlug(ctx, orgID, slug string) (*Team, error)
    Update(ctx, id string, in *UpdateIn) (*Team, error)
    Delete(ctx, id string) error
    List(ctx, orgID string) ([]*Team, error)

    // Members
    AddMember(ctx, teamID, userID string, role string) error
    RemoveMember(ctx, teamID, userID string) error
    ListMembers(ctx, teamID string) ([]*TeamMember, error)

    // Repos
    AddRepo(ctx, teamID, repoID string, permission string) error
    RemoveRepo(ctx, teamID, repoID string) error
    ListRepos(ctx, teamID string) ([]*TeamRepo, error)
}
```

### 2.10 Collaborators Feature (Standalone)

**Package:** `feature/collaborators`

Extracted from repos for standalone management.

**API Interface:**
```go
type API interface {
    Add(ctx, repoID, userID string, permission string) error
    Remove(ctx, repoID, userID string) error
    Update(ctx, repoID, userID string, permission string) error
    Get(ctx, repoID, userID string) (*Collaborator, error)
    List(ctx, repoID string, limit, offset int) ([]*Collaborator, error)
    ListUserRepos(ctx, userID string, limit, offset int) ([]string, error)
    GetPermission(ctx, repoID, userID string) (string, error)
}
```

### 2.11 Pull Requests Feature

**Package:** `feature/pulls`

**Models:**
```go
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
    State          string     `json:"state"` // open, closed, merged
    IsDraft        bool       `json:"is_draft"`
    IsLocked       bool       `json:"is_locked"`
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

    Labels    []*labels.Label `json:"labels,omitempty"`
    Assignees []string        `json:"assignees,omitempty"`
    Reviewers []string        `json:"reviewers,omitempty"`
}

type Review struct {
    ID          string     `json:"id"`
    PRID        string     `json:"pr_id"`
    UserID      string     `json:"user_id"`
    Body        string     `json:"body"`
    State       string     `json:"state"` // pending, approved, changes_requested, commented
    CommitSHA   string     `json:"commit_sha"`
    CreatedAt   time.Time  `json:"created_at"`
    SubmittedAt *time.Time `json:"submitted_at,omitempty"`
}

type ReviewComment struct {
    ID               string    `json:"id"`
    ReviewID         string    `json:"review_id"`
    UserID           string    `json:"user_id"`
    Path             string    `json:"path"`
    Position         int       `json:"position,omitempty"`
    OriginalPosition int       `json:"original_position,omitempty"`
    DiffHunk         string    `json:"diff_hunk,omitempty"`
    Line             int       `json:"line,omitempty"`
    OriginalLine     int       `json:"original_line,omitempty"`
    Side             string    `json:"side"` // LEFT, RIGHT
    Body             string    `json:"body"`
    InReplyToID      string    `json:"in_reply_to_id,omitempty"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

**API Interface:**
```go
type API interface {
    // CRUD
    Create(ctx, repoID, authorID string, in *CreateIn) (*PullRequest, error)
    GetByID(ctx, id string) (*PullRequest, error)
    GetByNumber(ctx, repoID string, number int) (*PullRequest, error)
    Update(ctx, id string, in *UpdateIn) (*PullRequest, error)
    List(ctx, repoID string, opts *ListOpts) ([]*PullRequest, int, error)

    // State
    Close(ctx, id string) error
    Reopen(ctx, id string) error
    Merge(ctx, id, userID string, method string) error
    MarkReady(ctx, id string) error

    // Labels
    AddLabels(ctx, id string, labelIDs []string) error
    RemoveLabel(ctx, id, labelID string) error

    // Assignees
    AddAssignees(ctx, id string, userIDs []string) error
    RemoveAssignees(ctx, id string, userIDs []string) error

    // Reviewers
    RequestReview(ctx, id string, userIDs []string) error
    RemoveReviewRequest(ctx, id string, userIDs []string) error

    // Reviews
    CreateReview(ctx, prID, userID string, in *CreateReviewIn) (*Review, error)
    SubmitReview(ctx, reviewID string, state string) (*Review, error)
    ListReviews(ctx, prID string) ([]*Review, error)

    // Review Comments
    CreateReviewComment(ctx, reviewID string, in *CreateReviewCommentIn) (*ReviewComment, error)
    ListReviewComments(ctx, prID string) ([]*ReviewComment, error)
}
```

### 2.12 Releases Feature

**Package:** `feature/releases`

**Models:**
```go
type Release struct {
    ID              string     `json:"id"`
    RepoID          string     `json:"repo_id"`
    TagName         string     `json:"tag_name"`
    TargetCommitish string     `json:"target_commitish"`
    Name            string     `json:"name"`
    Body            string     `json:"body"`
    IsDraft         bool       `json:"is_draft"`
    IsPrerelease    bool       `json:"is_prerelease"`
    AuthorID        string     `json:"author_id"`
    CreatedAt       time.Time  `json:"created_at"`
    PublishedAt     *time.Time `json:"published_at,omitempty"`

    Assets []*Asset `json:"assets,omitempty"`
}

type Asset struct {
    ID            string    `json:"id"`
    ReleaseID     string    `json:"release_id"`
    Name          string    `json:"name"`
    Label         string    `json:"label"`
    ContentType   string    `json:"content_type"`
    SizeBytes     int64     `json:"size_bytes"`
    DownloadCount int       `json:"download_count"`
    UploaderID    string    `json:"uploader_id"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

**API Interface:**
```go
type API interface {
    Create(ctx, repoID, authorID string, in *CreateIn) (*Release, error)
    GetByID(ctx, id string) (*Release, error)
    GetByTag(ctx, repoID, tagName string) (*Release, error)
    GetLatest(ctx, repoID string) (*Release, error)
    Update(ctx, id string, in *UpdateIn) (*Release, error)
    Delete(ctx, id string) error
    List(ctx, repoID string, limit, offset int) ([]*Release, error)
    Publish(ctx, id string) (*Release, error)

    // Assets
    UploadAsset(ctx, releaseID, uploaderID string, in *UploadAssetIn) (*Asset, error)
    GetAsset(ctx, id string) (*Asset, error)
    DeleteAsset(ctx, id string) error
    ListAssets(ctx, releaseID string) ([]*Asset, error)
    IncrementDownload(ctx, assetID string) error
}
```

### 2.13 Webhooks Feature

**Package:** `feature/webhooks`

**Models:**
```go
type Webhook struct {
    ID               string     `json:"id"`
    RepoID           string     `json:"repo_id,omitempty"`
    OrgID            string     `json:"org_id,omitempty"`
    URL              string     `json:"url"`
    Secret           string     `json:"-"`
    ContentType      string     `json:"content_type"`
    Events           []string   `json:"events"`
    Active           bool       `json:"active"`
    InsecureSSL      bool       `json:"insecure_ssl"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
    LastResponseCode int        `json:"last_response_code,omitempty"`
    LastResponseAt   *time.Time `json:"last_response_at,omitempty"`
}

type Delivery struct {
    ID              string    `json:"id"`
    WebhookID       string    `json:"webhook_id"`
    Event           string    `json:"event"`
    GUID            string    `json:"guid"`
    Payload         string    `json:"payload"`
    RequestHeaders  string    `json:"request_headers"`
    ResponseHeaders string    `json:"response_headers"`
    ResponseBody    string    `json:"response_body"`
    StatusCode      int       `json:"status_code"`
    Delivered       bool      `json:"delivered"`
    DurationMS      int       `json:"duration_ms"`
    CreatedAt       time.Time `json:"created_at"`
}
```

**API Interface:**
```go
type API interface {
    Create(ctx, in *CreateIn) (*Webhook, error)
    GetByID(ctx, id string) (*Webhook, error)
    Update(ctx, id string, in *UpdateIn) (*Webhook, error)
    Delete(ctx, id string) error
    ListByRepo(ctx, repoID string) ([]*Webhook, error)
    ListByOrg(ctx, orgID string) ([]*Webhook, error)
    Ping(ctx, id string) error

    // Deliveries
    RecordDelivery(ctx, d *Delivery) error
    GetDelivery(ctx, id string) (*Delivery, error)
    ListDeliveries(ctx, webhookID string, limit, offset int) ([]*Delivery, error)
    RedeliverDelivery(ctx, id string) error
}
```

---

## Part 3: DuckDB Store Implementation

### 3.1 Store Structure

Each feature will have its own store file in `store/duckdb/`:

```
store/duckdb/
├── store.go              # Core store (existing)
├── schema.sql            # Database schema (existing)
├── users_store.go        # Users store (existing, enhance)
├── repos_store.go        # Repos store (existing, enhance)
├── issues_store.go       # Issues store (existing, enhance)
├── labels_store.go       # NEW
├── milestones_store.go   # NEW
├── comments_store.go     # NEW
├── activities_store.go   # NEW
├── notifications_store.go # NEW
├── stars_store.go        # NEW
├── watches_store.go      # NEW
├── orgs_store.go         # NEW
├── teams_store.go        # NEW
├── collaborators_store.go # NEW (extract from repos)
├── pulls_store.go        # NEW
├── releases_store.go     # NEW
├── webhooks_store.go     # NEW
└── *_store_test.go       # Tests for each store
```

### 3.2 Implementation Pattern

Each store follows this pattern:

```go
package duckdb

import (
    "context"
    "database/sql"

    "github.com/go-mizu/blueprints/githome/feature/XXX"
)

type XXXStore struct {
    db *sql.DB
}

func NewXXXStore(db *sql.DB) *XXXStore {
    return &XXXStore{db: db}
}

// Implement all Store interface methods
```

### 3.3 Test Coverage Requirements

Each store test file must include:
1. Create/Insert tests
2. Read/Get tests (by ID, by other fields)
3. Update tests
4. Delete tests
5. List tests with pagination
6. Edge cases (not found, duplicates, etc.)

---

## Part 4: Implementation Order

### Phase 1: Core Features (High Priority)
1. Labels feature + store + tests
2. Milestones feature + store + tests
3. Comments feature + store + tests
4. Enhance existing Issues store with comments

### Phase 2: Social Features
5. Stars feature (standalone) + store + tests
6. Watches feature + store + tests
7. Activities feature + store + tests
8. Notifications feature + store + tests

### Phase 3: Organization Features
9. Orgs feature + store + tests
10. Teams feature + store + tests
11. Collaborators feature (standalone) + store + tests

### Phase 4: Advanced Features
12. Pull Requests feature + store + tests
13. Releases feature + store + tests
14. Webhooks feature + store + tests

### Phase 5: Enhancements
15. Enhance Users store (SSH Keys, API Tokens)
16. Enhance Repos store (ListForks, Transfer)

---

## File Summary

### New Feature Files (28 files)
- `feature/labels/api.go`
- `feature/labels/service.go`
- `feature/milestones/api.go`
- `feature/milestones/service.go`
- `feature/comments/api.go`
- `feature/comments/service.go`
- `feature/activities/api.go`
- `feature/activities/service.go`
- `feature/notifications/api.go`
- `feature/notifications/service.go`
- `feature/stars/api.go`
- `feature/stars/service.go`
- `feature/watches/api.go`
- `feature/watches/service.go`
- `feature/orgs/api.go`
- `feature/orgs/service.go`
- `feature/teams/api.go`
- `feature/teams/service.go`
- `feature/collaborators/api.go`
- `feature/collaborators/service.go`
- `feature/pulls/api.go`
- `feature/pulls/service.go`
- `feature/releases/api.go`
- `feature/releases/service.go`
- `feature/webhooks/api.go`
- `feature/webhooks/service.go`

### New Store Files (28 files)
- `store/duckdb/labels_store.go`
- `store/duckdb/labels_store_test.go`
- `store/duckdb/milestones_store.go`
- `store/duckdb/milestones_store_test.go`
- `store/duckdb/comments_store.go`
- `store/duckdb/comments_store_test.go`
- `store/duckdb/activities_store.go`
- `store/duckdb/activities_store_test.go`
- `store/duckdb/notifications_store.go`
- `store/duckdb/notifications_store_test.go`
- `store/duckdb/stars_store.go`
- `store/duckdb/stars_store_test.go`
- `store/duckdb/watches_store.go`
- `store/duckdb/watches_store_test.go`
- `store/duckdb/orgs_store.go`
- `store/duckdb/orgs_store_test.go`
- `store/duckdb/teams_store.go`
- `store/duckdb/teams_store_test.go`
- `store/duckdb/collaborators_store.go`
- `store/duckdb/collaborators_store_test.go`
- `store/duckdb/pulls_store.go`
- `store/duckdb/pulls_store_test.go`
- `store/duckdb/releases_store.go`
- `store/duckdb/releases_store_test.go`
- `store/duckdb/webhooks_store.go`
- `store/duckdb/webhooks_store_test.go`

### Enhanced Existing Files (6 files)
- `feature/users/api.go` - Add SSH Key and API Token types
- `feature/users/service.go` - Add SSH Key and API Token methods
- `store/duckdb/users_store.go` - Add SSH Key and API Token store methods
- `store/duckdb/users_store_test.go` - Add tests
- `store/duckdb/repos_store.go` - Add ListForks method
- `store/duckdb/repos_store_test.go` - Add tests

---

## Notes

1. The schema.sql already contains all required tables
2. Follow existing code patterns for consistency
3. Use ULID for all ID generation
4. Handle nullable fields properly with sql.NullXXX types
5. All timestamps in UTC
6. Implement proper pagination (limit/offset)
7. Write comprehensive tests for all store methods
