# Enhance Pull Request View to Match GitHub 100%

## Reference PR
- **URL**: https://github.com/golang/go/pull/75624
- **Title**: "compress/flate: improve compression speed"
- **PR Number**: #75624
- **State**: Open
- **Author**: klauspost
- **Expected Data**:
  - Conversation comments: 114
  - Commits: 16
  - Files changed: 45
  - Lines: +3,571 −1,188

## Current Issues

1. **PR body not rendering correctly** - Body HTML not displaying properly
2. **Missing conversation count** - Comments count not shown in tabs (should be 114)
3. **Wrong commits/files counts** - Data mismatch with GitHub
4. **Missing proper timeline events** - Reviews, commits, status checks not in timeline
5. **Author not showing correctly** - Author info and dates broken
6. **Missing PR-specific seeding** - Need to seed PR commits, files, review comments

## Implementation Plan

### Phase 1: Enhance GitHub API Client & Seeder

#### 1.1 Add Missing GitHub API Endpoints

**File**: `pkg/seed/github/client.go`

Add new methods:
```go
// GetPullRequest fetches a single PR with full details
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*ghPullRequest, *RateLimitInfo, error)

// ListPRCommits fetches commits for a pull request
func (c *Client) ListPRCommits(ctx context.Context, owner, repo string, number int, opts *ListOptions) ([]*ghCommit, *RateLimitInfo, error)

// ListPRFiles fetches files changed in a pull request
func (c *Client) ListPRFiles(ctx context.Context, owner, repo string, number int, opts *ListOptions) ([]*ghPRFile, *RateLimitInfo, error)

// ListPRReviews fetches reviews for a pull request
func (c *Client) ListPRReviews(ctx context.Context, owner, repo string, number int, opts *ListOptions) ([]*ghReview, *RateLimitInfo, error)

// ListTimelineEvents fetches timeline events for an issue/PR
func (c *Client) ListTimelineEvents(ctx context.Context, owner, repo string, number int, opts *ListOptions) ([]*ghTimelineEvent, *RateLimitInfo, error)
```

Add new types:
```go
type ghCommit struct {
    SHA       string          `json:"sha"`
    NodeID    string          `json:"node_id"`
    Commit    *ghCommitData   `json:"commit"`
    URL       string          `json:"url"`
    HTMLURL   string          `json:"html_url"`
    Author    *ghUser         `json:"author"`
    Committer *ghUser         `json:"committer"`
    Parents   []*ghCommitRef  `json:"parents"`
}

type ghCommitData struct {
    Author    *ghCommitAuthor `json:"author"`
    Committer *ghCommitAuthor `json:"committer"`
    Message   string          `json:"message"`
    Tree      *ghCommitRef    `json:"tree"`
}

type ghCommitAuthor struct {
    Name  string    `json:"name"`
    Email string    `json:"email"`
    Date  time.Time `json:"date"`
}

type ghCommitRef struct {
    SHA string `json:"sha"`
    URL string `json:"url"`
}

type ghPRFile struct {
    SHA         string `json:"sha"`
    Filename    string `json:"filename"`
    Status      string `json:"status"`
    Additions   int    `json:"additions"`
    Deletions   int    `json:"deletions"`
    Changes     int    `json:"changes"`
    BlobURL     string `json:"blob_url"`
    RawURL      string `json:"raw_url"`
    ContentsURL string `json:"contents_url"`
    Patch       string `json:"patch"`
}

type ghReview struct {
    ID                int64     `json:"id"`
    NodeID            string    `json:"node_id"`
    User              *ghUser   `json:"user"`
    Body              string    `json:"body"`
    State             string    `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING, DISMISSED
    HTMLURL           string    `json:"html_url"`
    CommitID          string    `json:"commit_id"`
    SubmittedAt       time.Time `json:"submitted_at"`
    AuthorAssociation string    `json:"author_association"`
}

type ghTimelineEvent struct {
    ID        int64     `json:"id"`
    NodeID    string    `json:"node_id"`
    URL       string    `json:"url"`
    Actor     *ghUser   `json:"actor"`
    Event     string    `json:"event"` // committed, reviewed, commented, labeled, assigned, etc.
    CommitID  string    `json:"commit_id,omitempty"`
    CommitURL string    `json:"commit_url,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    // Additional fields based on event type
    Label     *ghLabel  `json:"label,omitempty"`
    Assignee  *ghUser   `json:"assignee,omitempty"`
    Review    *ghReview `json:"review,omitempty"`
}
```

#### 1.2 Enhance PR Seeding

**File**: `pkg/seed/github/seeder.go`

Modify `seedPullRequests` to:
1. Fetch full PR details (not just list API)
2. Seed PR commits
3. Seed PR files
4. Seed PR reviews
5. Seed PR review comments
6. Seed issue comments for PR (PR conversation uses issue comment API)

```go
func (s *Seeder) seedPullRequest(ctx context.Context, repo *repos.Repository, prNumber int) error {
    // 1. Get full PR details
    ghPR, _, err := s.client.GetPullRequest(ctx, owner, repoName, prNumber)

    // 2. Get PR commits
    commits, _, err := s.client.ListPRCommits(ctx, owner, repoName, prNumber, &ListOptions{PerPage: 100})

    // 3. Get PR files
    files, _, err := s.client.ListPRFiles(ctx, owner, repoName, prNumber, &ListOptions{PerPage: 100})

    // 4. Get PR reviews
    reviews, _, err := s.client.ListPRReviews(ctx, owner, repoName, prNumber, &ListOptions{PerPage: 100})

    // 5. Get PR review comments
    reviewComments, _, err := s.client.ListPRComments(ctx, owner, repoName, prNumber, &ListOptions{PerPage: 100})

    // 6. Get issue comments (PR conversation)
    issueComments, _, err := s.client.ListIssueComments(ctx, owner, repoName, prNumber, &ListOptions{PerPage: 100})

    // Store all data
    return nil
}
```

### Phase 2: Database Schema & Store Updates

#### 2.1 Add PR Commits Store

**File**: `store/duckdb/pr_commits_store.go` (new file)

```go
type PRCommitsStore struct {
    db *sql.DB
}

// Schema
const createPRCommitsTable = `
CREATE TABLE IF NOT EXISTS pr_commits (
    id BIGINT PRIMARY KEY,
    pr_id BIGINT NOT NULL,
    sha VARCHAR NOT NULL,
    message TEXT,
    author_name VARCHAR,
    author_email VARCHAR,
    author_date TIMESTAMP,
    committer_name VARCHAR,
    committer_email VARCHAR,
    committer_date TIMESTAMP,
    author_user_id BIGINT,
    committer_user_id BIGINT,
    url VARCHAR,
    html_url VARCHAR,
    UNIQUE(pr_id, sha)
)
`

func (s *PRCommitsStore) Create(ctx context.Context, commit *pulls.Commit) error
func (s *PRCommitsStore) ListByPR(ctx context.Context, prID int64) ([]*pulls.Commit, error)
```

#### 2.2 Add PR Files Store

**File**: `store/duckdb/pr_files_store.go` (new file)

```go
type PRFilesStore struct {
    db *sql.DB
}

// Schema
const createPRFilesTable = `
CREATE TABLE IF NOT EXISTS pr_files (
    id BIGSERIAL PRIMARY KEY,
    pr_id BIGINT NOT NULL,
    sha VARCHAR,
    filename VARCHAR NOT NULL,
    status VARCHAR,
    additions INT DEFAULT 0,
    deletions INT DEFAULT 0,
    changes INT DEFAULT 0,
    blob_url VARCHAR,
    raw_url VARCHAR,
    patch TEXT,
    UNIQUE(pr_id, filename)
)
`

func (s *PRFilesStore) Create(ctx context.Context, prID int64, file *pulls.PRFile) error
func (s *PRFilesStore) ListByPR(ctx context.Context, prID int64) ([]*pulls.PRFile, error)
```

#### 2.3 Add PR Reviews Store

**File**: `store/duckdb/pr_reviews_store.go` (new file)

```go
type PRReviewsStore struct {
    db *sql.DB
}

// Schema
const createPRReviewsTable = `
CREATE TABLE IF NOT EXISTS pr_reviews (
    id BIGINT PRIMARY KEY,
    pr_id BIGINT NOT NULL,
    user_id BIGINT,
    body TEXT,
    state VARCHAR,
    commit_id VARCHAR,
    submitted_at TIMESTAMP,
    author_association VARCHAR,
    html_url VARCHAR
)
`

func (s *PRReviewsStore) Create(ctx context.Context, review *pulls.Review) error
func (s *PRReviewsStore) ListByPR(ctx context.Context, prID int64) ([]*pulls.Review, error)
```

### Phase 3: Enhance Pulls Service

**File**: `feature/pulls/service.go`

Update service to populate PR with related data:

```go
func (s *Service) Get(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
    // Get PR from store
    pr, err := s.store.GetByNumber(ctx, repoID, number)

    // Populate user info
    if pr.CreatorID > 0 {
        pr.User, _ = s.users.GetSimpleByID(ctx, pr.CreatorID)
    }

    // Populate labels
    pr.Labels, _ = s.labels.ListForPR(ctx, pr.ID)

    // Populate milestone
    if pr.MilestoneID > 0 {
        pr.Milestone, _ = s.milestones.GetByID(ctx, pr.MilestoneID)
    }

    // Populate assignees
    pr.Assignees, _ = s.assignees.ListForPR(ctx, pr.ID)

    // Populate requested reviewers
    pr.RequestedReviewers, _ = s.store.ListRequestedReviewers(ctx, pr.ID)

    return pr, nil
}

// Add new methods
func (s *Service) ListCommits(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Commit, error)
func (s *Service) ListFiles(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*PRFile, error)
func (s *Service) ListReviews(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Review, error)
func (s *Service) GetConversationCount(ctx context.Context, owner, repo string, number int) (int, error)
```

### Phase 4: Enhance Page Handler

**File**: `app/web/handler/page.go`

#### 4.1 Update PullDetailData

```go
type PullDetailData struct {
    // Existing fields...

    // Tab counts for accurate display
    ConversationCount int // Issue comments + PR review comments
    CommitsCount      int
    ChecksCount       int
    FilesCount        int

    // Aggregated stats
    TotalAdditions    int
    TotalDeletions    int

    // Timeline events (combined comments, reviews, commits)
    Timeline          []*PRTimelineEvent

    // Reviews grouped by state
    ApprovedReviews   []*ReviewView
    ChangesRequested  []*ReviewView
    CommentedReviews  []*ReviewView
}

type PRTimelineEvent struct {
    Type          string // "comment", "review", "commit", "labeled", "assigned", etc.
    Actor         *users.SimpleUser
    TimeAgo       string
    CreatedAt     time.Time
    // Type-specific data
    Comment       *CommentView
    Review        *ReviewView
    Commit        *CommitViewItem
    Label         *labels.Label
    Assignee      *users.SimpleUser
}

type ReviewView struct {
    *pulls.Review
    TimeAgo       string
    BodyHTML      template.HTML
    StateIcon     string
    StateColor    string
    Comments      []*ReviewCommentView
}

type ReviewCommentView struct {
    *pulls.ReviewComment
    TimeAgo  string
    BodyHTML template.HTML
}
```

#### 4.2 Update PullDetail Handler

```go
func (h *Page) PullDetail(c *mizu.Ctx) error {
    // ... existing code ...

    // Get commits count
    commitList, _ := h.pulls.ListCommits(ctx, owner, repoName, number, nil)
    commitsCount := len(commitList)

    // Get files count and calculate totals
    fileList, _ := h.pulls.ListFiles(ctx, owner, repoName, number, nil)
    filesCount := len(fileList)
    var totalAdditions, totalDeletions int
    for _, f := range fileList {
        totalAdditions += f.Additions
        totalDeletions += f.Deletions
    }

    // Get reviews
    reviewList, _ := h.pulls.ListReviews(ctx, owner, repoName, number, nil)

    // Get issue comments (PR conversation)
    issueCommentList, _ := h.comments.ListForIssue(ctx, owner, repoName, number, &comments.ListOpts{PerPage: 200})

    // Get review comments (inline code comments)
    reviewCommentList, _ := h.pulls.ListReviewComments(ctx, owner, repoName, number, nil)

    // Calculate conversation count (issue comments + review comments + reviews with body)
    conversationCount := len(issueCommentList) + len(reviewCommentList)
    for _, r := range reviewList {
        if r.Body != "" {
            conversationCount++
        }
    }

    // Build unified timeline
    timeline := h.buildPRTimeline(pr, issueCommentList, reviewList, commitList, reviewCommentList)

    return render(h, c, "pull_view", PullDetailData{
        // ... existing fields ...
        ConversationCount: conversationCount,
        CommitsCount:      commitsCount,
        FilesCount:        filesCount,
        TotalAdditions:    totalAdditions,
        TotalDeletions:    totalDeletions,
        Timeline:          timeline,
    })
}

func (h *Page) buildPRTimeline(
    pr *pulls.PullRequest,
    comments []*comments.IssueComment,
    reviews []*pulls.Review,
    commits []*pulls.Commit,
    reviewComments []*pulls.ReviewComment,
) []*PRTimelineEvent {
    var events []*PRTimelineEvent

    // Add comments to timeline
    for _, c := range comments {
        events = append(events, &PRTimelineEvent{
            Type:      "comment",
            Actor:     c.User,
            CreatedAt: c.CreatedAt,
            TimeAgo:   formatTimeAgo(c.CreatedAt),
            Comment:   h.toCommentView(c),
        })
    }

    // Add reviews to timeline
    for _, r := range reviews {
        events = append(events, &PRTimelineEvent{
            Type:      "review",
            Actor:     r.User,
            CreatedAt: r.SubmittedAt,
            TimeAgo:   formatTimeAgo(r.SubmittedAt),
            Review:    h.toReviewView(r),
        })
    }

    // Add commits to timeline (optional, GitHub shows these separately)
    // ...

    // Sort by created_at
    sort.Slice(events, func(i, j int) bool {
        return events[i].CreatedAt.Before(events[j].CreatedAt)
    })

    return events
}
```

### Phase 5: Enhance PR Template

**File**: `assets/views/default/pages/pull_view.html`

#### 5.1 GitHub-Accurate Header

```html
<!-- PR Header - exactly like GitHub -->
<div class="gh-header">
    <div class="gh-header-show">
        <h1 class="gh-header-title">
            <span class="js-issue-title markdown-title">{{.Pull.Title}}</span>
            <span class="f1-light color-fg-muted">#{{.Pull.Number}}</span>
        </h1>
    </div>

    <div class="gh-header-meta d-flex flex-items-center flex-wrap mt-2 pb-3 border-bottom color-border-muted">
        <!-- State Badge -->
        {{if .Pull.Merged}}
        <span class="State State--merged d-inline-flex flex-items-center">
            <svg class="octicon octicon-git-merge" viewBox="0 0 16 16" width="16" height="16">...</svg>
            <span class="ml-1">Merged</span>
        </span>
        {{else if eq .Pull.State "open"}}
        {{if .Pull.Draft}}
        <span class="State State--draft d-inline-flex flex-items-center">
            <svg class="octicon octicon-git-pull-request-draft" viewBox="0 0 16 16" width="16" height="16">...</svg>
            <span class="ml-1">Draft</span>
        </span>
        {{else}}
        <span class="State State--open d-inline-flex flex-items-center">
            <svg class="octicon octicon-git-pull-request" viewBox="0 0 16 16" width="16" height="16">...</svg>
            <span class="ml-1">Open</span>
        </span>
        {{end}}
        {{else}}
        <span class="State State--closed d-inline-flex flex-items-center">
            <svg class="octicon octicon-git-pull-request-closed" viewBox="0 0 16 16" width="16" height="16">...</svg>
            <span class="ml-1">Closed</span>
        </span>
        {{end}}

        <span class="color-fg-muted ml-2">
            <a href="/{{.Pull.User.Login}}" class="author Link--secondary text-bold">{{.Pull.User.Login}}</a>
            wants to merge {{.CommitsCount}} commit{{if ne .CommitsCount 1}}s{{end}} into
            <span class="commit-ref">
                <a href="/{{.Repo.FullName}}/tree/{{.Pull.Base.Ref}}">{{.Pull.Base.Label}}</a>
            </span>
            from
            <span class="commit-ref">
                <a href="/{{.Repo.FullName}}/tree/{{.Pull.Head.Ref}}">{{.Pull.Head.Label}}</a>
            </span>
        </span>
    </div>
</div>
```

#### 5.2 GitHub-Style Tabs with Accurate Counts

```html
<!-- PR Tabs - exactly like GitHub -->
<div class="tabnav mt-3">
    <nav class="tabnav-tabs" aria-label="Pull request tabs">
        <a href="#conversation" class="tabnav-tab selected" aria-current="page">
            <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                <path d="M1.5 2.75c0-.69.56-1.25 1.25-1.25h10.5c.69 0 1.25.56 1.25 1.25v7.5c0 .69-.56 1.25-1.25 1.25h-4.5l-2.573 2.573a1.25 1.25 0 0 1-1.927-.328l-.646-1.245H2.75c-.69 0-1.25-.56-1.25-1.25v-7.5Z"/>
            </svg>
            <span>Conversation</span>
            {{if gt .ConversationCount 0}}
            <span class="Counter">{{.ConversationCount}}</span>
            {{end}}
        </a>
        <a href="#commits" class="tabnav-tab">
            <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                <path d="M11.93 8.5a4.002 4.002 0 0 1-7.86 0H.75a.75.75 0 0 1 0-1.5h3.32a4.002 4.002 0 0 1 7.86 0h3.32a.75.75 0 0 1 0 1.5Zm-1.43-.75a2.5 2.5 0 1 0-5 0 2.5 2.5 0 0 0 5 0Z"/>
            </svg>
            <span>Commits</span>
            <span class="Counter">{{.CommitsCount}}</span>
        </a>
        <a href="#checks" class="tabnav-tab">
            <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                <path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"/>
            </svg>
            <span>Checks</span>
            {{if gt .ChecksCount 0}}
            <span class="Counter">{{.ChecksCount}}</span>
            {{end}}
        </a>
        <a href="#files-changed" class="tabnav-tab">
            <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                <path d="M2 1.75C2 .784 2.784 0 3.75 0h6.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 13.25 16h-9.5A1.75 1.75 0 0 1 2 14.25V1.75Z"/>
            </svg>
            <span>Files changed</span>
            <span class="Counter">{{.FilesCount}}</span>
        </a>
    </nav>

    <!-- Diff Stats Badge -->
    <div class="diffstat ml-auto">
        <span class="color-fg-success text-bold">+{{formatNumber .TotalAdditions}}</span>
        <span class="color-fg-danger text-bold ml-1">-{{formatNumber .TotalDeletions}}</span>
    </div>
</div>
```

#### 5.3 Timeline with Comments, Reviews, and Events

```html
<!-- Conversation Timeline -->
<div id="conversation" class="js-discussion">
    <!-- PR Description (first comment) -->
    <div class="TimelineItem pt-0 pb-2">
        <div class="TimelineItem-avatar">
            <a href="/{{.Pull.User.Login}}">
                <img src="{{.Pull.User.AvatarURL}}" class="avatar avatar-user" width="40" height="40">
            </a>
        </div>
        <div class="TimelineItem-body">
            <div class="timeline-comment-group">
                <div class="timeline-comment">
                    <div class="timeline-comment-header d-flex flex-items-center color-bg-subtle px-3 py-2">
                        <h3 class="timeline-comment-header-text f5 text-normal">
                            <a href="/{{.Pull.User.Login}}" class="author Link--primary text-bold">{{.Pull.User.Login}}</a>
                            <span class="color-fg-muted">commented {{.Pull.TimeAgo}}</span>
                        </h3>
                        {{if .Pull.AuthorAssociation}}
                        {{if ne .Pull.AuthorAssociation "NONE"}}
                        <span class="Label Label--secondary ml-2">{{.Pull.AuthorAssociation}}</span>
                        {{end}}
                        {{end}}
                    </div>
                    <div class="comment-body markdown-body p-3">
                        {{if .Pull.BodyHTML}}
                        {{.Pull.BodyHTML}}
                        {{else}}
                        <p class="color-fg-muted"><em>No description provided.</em></p>
                        {{end}}
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Timeline Events -->
    {{range .Timeline}}
    {{if eq .Type "comment"}}
    <!-- Comment Event -->
    <div class="TimelineItem">
        <div class="TimelineItem-avatar">
            <a href="/{{.Actor.Login}}">
                <img src="{{.Actor.AvatarURL}}" class="avatar avatar-user" width="40" height="40">
            </a>
        </div>
        <div class="TimelineItem-body">
            <div class="timeline-comment">
                <div class="timeline-comment-header d-flex flex-items-center color-bg-subtle px-3 py-2">
                    <a href="/{{.Actor.Login}}" class="author Link--primary text-bold">{{.Actor.Login}}</a>
                    <span class="color-fg-muted ml-1">commented {{.TimeAgo}}</span>
                    {{if .Comment.AuthorAssociation}}
                    {{if ne .Comment.AuthorAssociation "NONE"}}
                    <span class="Label Label--secondary ml-2">{{.Comment.AuthorAssociation}}</span>
                    {{end}}
                    {{end}}
                </div>
                <div class="comment-body markdown-body p-3">
                    {{.Comment.BodyHTML}}
                </div>
            </div>
        </div>
    </div>
    {{else if eq .Type "review"}}
    <!-- Review Event -->
    <div class="TimelineItem">
        <div class="TimelineItem-badge {{.Review.StateColor}}">
            <!-- Review state icon -->
            {{if eq .Review.State "APPROVED"}}
            <svg class="octicon octicon-check" viewBox="0 0 16 16" width="16" height="16">
                <path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"/>
            </svg>
            {{else if eq .Review.State "CHANGES_REQUESTED"}}
            <svg class="octicon octicon-file-diff" viewBox="0 0 16 16" width="16" height="16">
                <path d="M2 1.75C2 .784 2.784 0 3.75 0h6.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 13.25 16h-9.5A1.75 1.75 0 0 1 2 14.25V1.75Z"/>
            </svg>
            {{else}}
            <svg class="octicon octicon-eye" viewBox="0 0 16 16" width="16" height="16">
                <path d="M8 2c1.981 0 3.671.992 4.933 2.078 1.27 1.091 2.187 2.345 2.637 3.023a1.62 1.62 0 0 1 0 1.798c-.45.678-1.367 1.932-2.637 3.023C11.67 13.008 9.981 14 8 14c-1.981 0-3.671-.992-4.933-2.078C1.797 10.83.88 9.576.43 8.898a1.62 1.62 0 0 1 0-1.798c.45-.677 1.367-1.931 2.637-3.022C4.33 2.992 6.019 2 8 2ZM1.679 7.932a.12.12 0 0 0 0 .136c.411.622 1.241 1.75 2.366 2.717C5.176 11.758 6.527 12.5 8 12.5c1.473 0 2.825-.742 3.955-1.715 1.124-.967 1.954-2.096 2.366-2.717a.12.12 0 0 0 0-.136c-.412-.621-1.242-1.75-2.366-2.717C10.824 4.242 9.473 3.5 8 3.5c-1.473 0-2.824.742-3.955 1.715-1.124.967-1.954 2.096-2.366 2.717ZM8 10a2 2 0 1 1-.001-3.999A2 2 0 0 1 8 10Z"/>
            </svg>
            {{end}}
        </div>
        <div class="TimelineItem-body">
            <a href="/{{.Actor.Login}}" class="text-bold Link--primary">{{.Actor.Login}}</a>
            {{if eq .Review.State "APPROVED"}}
            <span class="color-fg-success">approved these changes</span>
            {{else if eq .Review.State "CHANGES_REQUESTED"}}
            <span class="color-fg-danger">requested changes</span>
            {{else}}
            <span class="color-fg-muted">reviewed</span>
            {{end}}
            <span class="color-fg-muted">{{.TimeAgo}}</span>

            {{if .Review.BodyHTML}}
            <div class="review-body markdown-body mt-2 p-3 color-bg-subtle rounded-2">
                {{.Review.BodyHTML}}
            </div>
            {{end}}
        </div>
    </div>
    {{else if eq .Type "committed"}}
    <!-- Commit Event -->
    <div class="TimelineItem TimelineItem--condensed">
        <div class="TimelineItem-badge">
            <svg class="octicon octicon-git-commit" viewBox="0 0 16 16" width="16" height="16">
                <path d="M11.93 8.5a4.002 4.002 0 0 1-7.86 0H.75a.75.75 0 0 1 0-1.5h3.32a4.002 4.002 0 0 1 7.86 0h3.32a.75.75 0 0 1 0 1.5Zm-1.43-.75a2.5 2.5 0 1 0-5 0 2.5 2.5 0 0 0 5 0Z"/>
            </svg>
        </div>
        <div class="TimelineItem-body">
            <img src="{{.Commit.Author.AvatarURL}}" class="avatar avatar-small mr-1" width="16" height="16">
            <a href="/{{.Commit.Author.Login}}" class="text-bold Link--primary">{{.Commit.Author.Login}}</a>
            <span class="color-fg-muted">added a commit {{.TimeAgo}}</span>
            <a href="/{{$.Repo.FullName}}/commit/{{.Commit.SHA}}" class="Link--secondary ml-2">
                <code class="text-mono f6">{{.Commit.ShortSHA}}</code>
            </a>
        </div>
    </div>
    {{end}}
    {{end}}
</div>
```

### Phase 6: CSS Styling

Add GitHub-accurate CSS for:

```css
/* State badges */
.State--draft {
    background-color: #6e7781;
}

/* Review states */
.TimelineItem-badge.color-bg-success-emphasis {
    background-color: #1a7f37;
    color: white;
}

.TimelineItem-badge.color-bg-danger-emphasis {
    background-color: #cf222e;
    color: white;
}

/* Commit ref styling */
.commit-ref {
    display: inline-flex;
    align-items: center;
    padding: 2px 6px;
    font-size: 12px;
    font-family: ui-monospace, SFMono-Regular, monospace;
    background-color: var(--color-accent-subtle);
    border-radius: 6px;
}

/* Diff stats in tab bar */
.diffstat {
    font-size: 12px;
    font-family: ui-monospace, SFMono-Regular, monospace;
}

/* Timeline styling */
.TimelineItem {
    position: relative;
    display: flex;
    padding: 16px 0;
}

.TimelineItem::before {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 20px;
    display: block;
    width: 2px;
    content: "";
    background-color: var(--color-border-muted);
}

.TimelineItem:first-child::before {
    top: 40px;
}

.TimelineItem:last-child::before {
    bottom: auto;
    height: 20px;
}

.TimelineItem-avatar {
    position: relative;
    z-index: 1;
    flex-shrink: 0;
    width: 40px;
    margin-right: 16px;
}

.TimelineItem-badge {
    position: relative;
    z-index: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    margin-right: 16px;
    margin-left: 4px;
    color: var(--color-fg-muted);
    background-color: var(--color-canvas-default);
    border: 2px solid var(--color-border-default);
    border-radius: 50%;
}

.TimelineItem--condensed {
    padding: 8px 0;
}

.TimelineItem--condensed .TimelineItem-badge {
    width: 16px;
    height: 16px;
    margin-left: 12px;
}
```

### Phase 7: Template Helper Functions

**File**: `app/web/server.go` (template functions)

Add/update template functions:

```go
template.FuncMap{
    "formatNumber": func(n int) string {
        if n >= 1000 {
            return fmt.Sprintf("%.1fk", float64(n)/1000)
        }
        return strconv.Itoa(n)
    },
    "formatTimeAgo": formatTimeAgo,
    "truncate": func(s string, n int) string {
        if len(s) <= n {
            return s
        }
        return s[:n] + "..."
    },
    "firstChar": func(s string) string {
        if len(s) == 0 {
            return "?"
        }
        return strings.ToUpper(s[:1])
    },
    "contrastColor": func(hexColor string) string {
        // Calculate contrast color for label backgrounds
        // Returns "white" or "black" based on luminance
    },
    "toFloat": func(n int) float64 {
        return float64(n)
    },
    "div": func(a, b float64) float64 {
        if b == 0 {
            return 0
        }
        return a / b
    },
    "mul": func(a, b float64) float64 {
        return a * b
    },
    "add": func(a, b int) int {
        return a + b
    },
}
```

### Phase 8: Testing

#### 8.1 Seed Test Data

```bash
# Seed the golang/go PR #75624
./githome seed github golang/go --pr 75624
```

#### 8.2 Verify Data

1. Check conversation count = 114
2. Check commits count = 16
3. Check files changed = 45 (Note: current template shows different, need to verify API)
4. Check +3,571 / -1,188 lines
5. Verify author is klauspost
6. Verify PR state is Open

#### 8.3 Visual Comparison

Compare screenshots:
1. GitHub PR #75624 header vs GitHome header
2. GitHub PR tabs vs GitHome tabs
3. GitHub PR timeline vs GitHome timeline
4. GitHub PR sidebar vs GitHome sidebar

### Implementation Order

1. **Phase 1**: GitHub API client enhancements (~2 hours)
2. **Phase 2**: Database schema updates (~1 hour)
3. **Phase 3**: Pulls service updates (~1 hour)
4. **Phase 4**: Page handler updates (~2 hours)
5. **Phase 5**: Template updates (~3 hours)
6. **Phase 6**: CSS styling (~1 hour)
7. **Phase 7**: Template helpers (~30 mins)
8. **Phase 8**: Testing and refinement (~2 hours)

### Files to Modify

1. `pkg/seed/github/client.go` - Add new API methods
2. `pkg/seed/github/seeder.go` - Enhance PR seeding
3. `pkg/seed/github/mappers.go` - Add new mappers
4. `store/duckdb/pr_commits_store.go` - New file
5. `store/duckdb/pr_files_store.go` - New file
6. `store/duckdb/pr_reviews_store.go` - New file
7. `store/duckdb/migrations.go` - Add new tables
8. `feature/pulls/service.go` - Enhance service
9. `app/web/handler/page.go` - Update PullDetail handler
10. `app/web/server.go` - Add template functions
11. `assets/views/default/pages/pull_view.html` - Complete rewrite
12. `assets/views/default/base.html` - Add PR-specific CSS

### Success Criteria

- [ ] PR page renders exactly like GitHub
- [ ] Conversation tab shows 114 (matching GitHub)
- [ ] Commits tab shows 16 (matching GitHub)
- [ ] Files changed shows 45 (matching GitHub)
- [ ] Stats show +3,571 -1,188 (matching GitHub)
- [ ] Author shows klauspost with avatar
- [ ] State badge shows "Open" in green
- [ ] Branch refs show correct labels
- [ ] Timeline shows all comments and reviews
- [ ] Sidebar shows correct metadata
- [ ] Responsive on mobile
