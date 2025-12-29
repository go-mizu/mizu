# Pull Requests UI Implementation Plan

## Overview

Implement Pull Requests UI that renders exactly like GitHub, including:
- PR list page at `/{owner}/{repo}/pulls`
- PR detail page at `/{owner}/{repo}/pulls/{number}`
- 100% GitHub-compatible API for serving PR data
- Test with real PR: https://github.com/golang/go/pull/75624

## Reference Screenshot Analysis

From the GitHub PR #75624 screenshot:

### Header Section
- PR title with number: "compress/flate: improve compression speed #75624"
- Status badge: "Open" (green) with git-pull-request icon
- Merge info: "klauspost wants to merge 16 commits into golang:master from klauspost:deflate-improve-comp"
- Code dropdown button (right side)

### Tab Navigation
- Conversation (with comment count: 114)
- Commits (count: 16)
- Checks (count: 1)
- Files changed (count: 45)
- Diff stats on right: +3,571 -1,188 with colored bars

### Main Content Area (Left Column)
- Original comment with:
  - User avatar (circular)
  - Username with "commented" text and date
  - "Contributor" badge
  - "..." menu button
  - Comment body (rendered markdown with links, lists)
  - Reactions section

### Sidebar (Right Column)
- Reviewers section: "No reviews" with info link
- Assignees section: "No one assigned"
- Labels section: "None yet"
- Projects section: "None yet"
- Milestone section: "No milestone"
- Development section: Shows linked issues
- Notifications section with "Unsubscribe" button
- Participants section (not visible in screenshot)

## Implementation Steps

### 1. Database Schema (Already Exists)

The `pull_requests` table already exists in `store/duckdb/schema.sql` with all required fields:
- Basic: id, node_id, repo_id, number, state, locked, title, body
- Branch info: head_ref, head_sha, head_repo_id, head_label, base_ref, base_sha, base_label
- Merge status: draft, merged, mergeable, rebaseable, mergeable_state, merge_commit_sha, merged_at, merged_by_id
- Stats: comments, review_comments, commits, additions, deletions, changed_files
- Relations: creator_id, milestone_id
- Timestamps: created_at, updated_at, closed_at

Related tables:
- pr_assignees
- pr_labels
- pr_reviews
- pr_review_comments
- pr_requested_reviewers
- pr_requested_teams

### 2. Domain Models (Already Exists)

In `feature/pulls/api.go`:
- PullRequest struct with all GitHub-compatible fields
- PRBranch for head/base branch info
- Review, ReviewComment structs
- API interface with List, Get, Create, Update, etc.
- Store interface for persistence

### 3. Service Layer (Already Exists)

In `feature/pulls/service.go`:
- Full implementation of pulls.API interface
- URL population for GitHub API compatibility
- User info population

### 4. API Handlers (Already Exists)

In `app/web/handler/api/pull.go`:
- All 25+ REST API endpoints implemented
- GitHub-compatible JSON responses

### 5. Seeding (Already Exists)

In `pkg/seed/github/seeder.go`:
- importPullRequests() function
- mapPullRequest() mapper in mappers.go
- Support for importing merged PRs, assignees, review comments

### 6. Page Routes (TO IMPLEMENT)

Add to `app/web/server.go` in the pages section:
```go
pages.Get("/{owner}/{repo}/pulls", pageHandler.RepoPulls)
pages.Get("/{owner}/{repo}/pulls/{number}", pageHandler.PullDetail)
```

### 7. Page Handlers (TO IMPLEMENT)

Add to `app/web/handler/page.go`:

#### RepoPulls Handler
```go
func (h *Page) RepoPulls(c *mizu.Ctx) error {
    // Similar to RepoIssues but for pull requests
    // - Get repo
    // - Build repo view with "pulls" active tab
    // - List PRs with pagination
    // - Get labels for filtering
    // - Calculate open/closed counts
}
```

#### PullDetail Handler
```go
func (h *Page) PullDetail(c *mizu.Ctx) error {
    // Similar to IssueDetail but for pull requests
    // - Get repo and PR
    // - Build repo view
    // - Get commits, files, reviews
    // - Get comments
    // - Build participants list
    // - Render markdown body
}
```

### 8. View Data Structs (TO IMPLEMENT)

Already partially defined in page.go:
- PullView struct (exists)
- RepoPullsData struct (exists)
- PullDetailData struct (exists)

Need to ensure all fields match template requirements.

### 9. HTML Templates (TO IMPLEMENT)

#### repo_pulls.html
Based on repo_issues.html structure:
- Repo header with tabs
- Search/filter toolbar
- State toggle (Open/Closed counts)
- Filter dropdowns (Author, Labels, Milestones, Reviewers, Sort)
- PR list items showing:
  - Status icon (open/closed/merged/draft)
  - Title with labels
  - Meta info: #number, author, created date
  - Review status indicators
  - Comment count
  - Assignee avatars

#### pull_view.html
Based on issue_view.html but with PR-specific elements:
- PR header (title, status, merge info)
- Tab navigation (Conversation/Commits/Checks/Files)
- Stats bar (additions/deletions with colored bars)
- Main timeline with:
  - Original comment (PR body)
  - Review comments
  - Status events
  - New comment form
- Sidebar with:
  - Reviewers
  - Assignees
  - Labels
  - Projects
  - Milestone
  - Development (linked issues)
  - Notifications
  - Participants

### 10. CSS Styles

Reuse existing GitHub-like styles from main.css:
- State badges (.State, .State--open, .State--merged)
- Timeline items
- Comment boxes
- Sidebar sections
- Label pills
- Avatar styles
- Diff stats bars

Add PR-specific styles:
- Merge status indicators
- Review status badges
- Diff stat bars (green/red proportional)

## Testing Plan

1. Build the application
2. Seed golang/go repository with PR #75624:
   ```bash
   githome seed github golang/go --import-prs --single-pr 75624 --import-comments
   ```
3. Start server and navigate to:
   - http://localhost:8080/golang/go/pulls (list view)
   - http://localhost:8080/golang/go/pulls/75624 (detail view)
4. Compare side-by-side with real GitHub page
5. Verify:
   - Header renders correctly (title, status, merge info)
   - Tab navigation works
   - Body markdown renders correctly
   - Comments display properly
   - Sidebar sections populated
   - Stats bar shows correct additions/deletions
   - Mobile responsive layout

## File Modifications Summary

### Files to Modify
1. `app/web/server.go` - Add page routes
2. `app/web/handler/page.go` - Add handler methods

### Files to Create
1. `assets/views/default/pages/repo_pulls.html` - PR list template
2. `assets/views/default/pages/pull_view.html` - PR detail template

### Files Already Complete (No Changes Needed)
- `store/duckdb/schema.sql` - Schema exists
- `store/duckdb/pulls_store.go` - Store implementation exists
- `feature/pulls/api.go` - Domain models exist
- `feature/pulls/service.go` - Service implementation exists
- `app/web/handler/api/pull.go` - API handlers exist
- `pkg/seed/github/seeder.go` - Seeding exists
- `pkg/seed/github/mappers.go` - Mappers exist
