# Commit UI Specification

**Status**: Draft
**Version**: 1.0
**Date**: 2025-12-29

## Overview

This specification defines the UI implementation for commit list and single commit view pages in GitHome, designed to match GitHub's commit UI exactly.

## Reference

- GitHub Primer Design System: https://primer.style/
- GitHub Octicons: https://primer.style/foundations/icons
- Sample Repository: https://github.com/golang/go

## 1. Commit List Page (`/:owner/:repo/commits/:ref`)

### 1.1 Page Structure

```
+------------------------------------------------------------------+
| [Repo Header]                                                     |
+------------------------------------------------------------------+
| [Navigation Tabs: Code | Issues | Pull Requests | ...]          |
+------------------------------------------------------------------+
| [Container]                                                       |
|   [Branch Selector] [Commits on master]                          |
|   +--------------------------------------------------------------+
|   | Commits on Dec 27, 2025                                      |
|   +--------------------------------------------------------------+
|   | [Avatar] [Commit Message]                 [SHA] [Browse code]|
|   |          Author · authored and Committer · committed · time  |
|   +--------------------------------------------------------------+
|   | [Avatar] [Commit Message]                 [SHA] [Browse code]|
|   |          Author · authored · time                            |
|   +--------------------------------------------------------------+
|   | ...                                                          |
|   +--------------------------------------------------------------+
|   | Commits on Dec 26, 2025                                      |
|   +--------------------------------------------------------------+
|   | ...                                                          |
|   +--------------------------------------------------------------+
|   [< Older]                                    [Newer >]         |
+------------------------------------------------------------------+
```

### 1.2 Components

#### 1.2.1 Branch/Ref Selector
- Button with branch icon (octicon-git-branch)
- Shows current branch/tag name
- Dropdown with branches and tags list

#### 1.2.2 Date Group Header
- Format: "Commits on Dec 27, 2025"
- Light gray background (`--color-bg-subtle`)
- Font: 14px, semi-bold
- Padding: 16px horizontal, 8px vertical

#### 1.2.3 Commit Entry
Each commit row contains:

**Left Section:**
- Author avatar (20px round, stacked if author != committer)
- Commit message (primary link, bold, dark text)
- Metadata line (muted text, 12px):
  - If author == committer: `{author} committed {time ago}`
  - If author != committer: `{author} authored and {committer} committed {time ago}`

**Right Section:**
- Short SHA (7 chars, monospace, link to commit)
- Copy SHA button (clipboard icon, appears on hover)
- Browse code button (code icon, links to tree at commit)

### 1.3 CSS Classes

```css
/* Commit list container */
.commits-list-container { }

/* Date group header */
.TimelineItem-header {
  padding: 8px 16px;
  background-color: var(--color-bg-subtle);
  border: 1px solid var(--color-border-default);
  border-bottom: 0;
  font-weight: 600;
  font-size: 14px;
}

/* Commit entry */
.commit-entry {
  display: flex;
  align-items: flex-start;
  padding: 16px;
  border: 1px solid var(--color-border-default);
  border-top: 0;
}

.commit-entry:first-child {
  border-top: 1px solid var(--color-border-default);
}

.commit-entry:hover {
  background-color: var(--color-bg-subtle);
}

/* Avatar stack for author/committer */
.AvatarStack {
  display: flex;
  flex-shrink: 0;
  margin-right: 16px;
}

.AvatarStack .Avatar {
  width: 20px;
  height: 20px;
  border: 2px solid var(--color-bg-default);
  margin-right: -8px;
}

/* Commit message link */
.commit-message {
  font-weight: 600;
  color: var(--color-fg-default);
  word-break: break-word;
}

.commit-message:hover {
  color: var(--color-accent-fg);
  text-decoration: underline;
}

/* Commit metadata */
.commit-meta {
  font-size: 12px;
  color: var(--color-fg-muted);
  margin-top: 4px;
}

.commit-meta a {
  color: var(--color-fg-muted);
}

.commit-meta a:hover {
  color: var(--color-accent-fg);
}

/* SHA and actions */
.commit-actions {
  display: flex;
  align-items: center;
  margin-left: auto;
  flex-shrink: 0;
  gap: 8px;
}

.commit-sha {
  font-family: var(--font-family-mono);
  font-size: 12px;
  padding: 4px 8px;
  background-color: var(--color-bg-subtle);
  border-radius: var(--border-radius-sm);
}

.commit-sha:hover {
  color: var(--color-accent-fg);
  background-color: var(--color-accent-subtle);
}

/* Pagination */
.commits-pagination {
  display: flex;
  justify-content: space-between;
  padding: 16px 0;
}
```

### 1.4 Octicons Used

| Icon | Name | Usage |
|------|------|-------|
| <svg>...</svg> | git-branch | Branch selector |
| <svg>...</svg> | copy | Copy SHA button |
| <svg>...</svg> | code | Browse code button |
| <svg>...</svg> | git-commit | Commit indicator |

#### Octicon SVG Definitions

```html
<!-- git-commit -->
<svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
  <path d="M11.93 8.5a4.002 4.002 0 0 1-7.86 0H.75a.75.75 0 0 1 0-1.5h3.32a4.002 4.002 0 0 1 7.86 0h3.32a.75.75 0 0 1 0 1.5Zm-1.43-.75a2.5 2.5 0 1 0-5 0 2.5 2.5 0 0 0 5 0Z"/>
</svg>

<!-- copy -->
<svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
  <path d="M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 0 1 0 1.5h-1.5a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-1.5a.75.75 0 0 1 1.5 0v1.5A1.75 1.75 0 0 1 9.25 16h-7.5A1.75 1.75 0 0 1 0 14.25Z"/>
  <path d="M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0 1 14.25 11h-7.5A1.75 1.75 0 0 1 5 9.25Zm1.75-.25a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-7.5a.25.25 0 0 0-.25-.25Z"/>
</svg>

<!-- code -->
<svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
  <path d="m11.28 3.22 4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.749.749 0 0 1-1.275-.326.749.749 0 0 1 .215-.734L13.94 8l-3.72-3.72a.749.749 0 0 1 .326-1.275.749.749 0 0 1 .734.215Zm-6.56 0a.751.751 0 0 1 1.042.018.751.751 0 0 1 .018 1.042L2.06 8l3.72 3.72a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L.47 8.53a.75.75 0 0 1 0-1.06Z"/>
</svg>
```

### 1.5 Data Structure

```go
// RepoCommitsData holds data for commits list page
type RepoCommitsData struct {
    Title         string
    User          *users.User
    Repo          *RepoView
    CommitGroups  []*CommitGroup    // Grouped by date
    CurrentBranch string
    Branches      []*branches.Branch
    Page          int
    HasNext       bool
    HasPrev       bool
    NextCursor    string            // For cursor-based pagination
    PrevCursor    string
    Breadcrumbs   []Breadcrumb
    UnreadCount   int
    ActiveNav     string
}

// CommitGroup represents commits grouped by date
type CommitGroup struct {
    Date    string           // "Dec 27, 2025"
    Commits []*CommitViewItem
}

// CommitViewItem represents a commit for display
type CommitViewItem struct {
    SHA              string
    ShortSHA         string          // First 7 chars
    Message          string
    MessageTitle     string          // First line only
    MessageBody      string          // Rest of message
    Author           *UserView
    Committer        *UserView
    AuthorDate       time.Time
    CommitterDate    time.Time
    TimeAgo          string
    IsSameAuthor     bool            // Author == Committer
    ParentSHAs       []string
    TreeURL          string          // Link to browse code
    CommitURL        string          // Link to commit detail
    Verified         bool
    VerificationLabel string
}

// UserView for display
type UserView struct {
    Login     string
    Name      string
    Email     string
    AvatarURL string
    HTMLURL   string
}
```

---

## 2. Single Commit View Page (`/:owner/:repo/commit/:sha`)

### 2.1 Page Structure

```
+------------------------------------------------------------------+
| [Repo Header]                                                     |
+------------------------------------------------------------------+
| [Navigation Tabs]                                                 |
+------------------------------------------------------------------+
| [Container]                                                       |
|   +--------------------------------------------------------------+
|   | [Commit Icon] Commit {short_sha}                             |
|   +--------------------------------------------------------------+
|   |                                                              |
|   | [Commit Title - First line of message]                       |
|   |                                                              |
|   | [Author Avatar] {author} authored and {committer} committed  |
|   |                 {time_ago} · commit {full_sha}               |
|   |                                                              |
|   | [Expand icon] {n} parent {parent_sha}                        |
|   |                                                              |
|   | [Full commit message if multiline...]                        |
|   |                                                              |
|   +--------------------------------------------------------------+
|                                                                   |
|   +--------------------------------------------------------------+
|   | Showing {n} changed files with {additions} additions and     |
|   | {deletions} deletions.                                       |
|   |                                                              |
|   | [Unified] [Split] view buttons                               |
|   +--------------------------------------------------------------+
|                                                                   |
|   [File Tree Sidebar (optional)]                                  |
|                                                                   |
|   +--------------------------------------------------------------+
|   | [File Icon] {filename}                          [+{add}/-{del}] |
|   +--------------------------------------------------------------+
|   | [Diff content with line numbers and highlighting]            |
|   | @@ -10,5 +10,8 @@                                            |
|   |   context line                                               |
|   | - deleted line                                               |
|   | + added line                                                 |
|   +--------------------------------------------------------------+
|   | [Next file...]                                               |
|   +--------------------------------------------------------------+
+------------------------------------------------------------------+
```

### 2.2 Components

#### 2.2.1 Commit Header
- Commit icon + "Commit {short_sha}" title
- Border box with subtle background

#### 2.2.2 Commit Info Card
- Commit title (first line, large, bold)
- Author/Committer info with avatars
- Verified badge (if GPG signed)
- Parent commit links
- Browse files button

#### 2.2.3 Stats Summary Bar
- "Showing X changed files with Y additions and Z deletions"
- Green for additions, red for deletions
- View mode toggle (Unified/Split)

#### 2.2.4 File Change List
Each file shows:
- File status icon (added/modified/deleted/renamed)
- File path (clickable to jump)
- +additions / -deletions count
- Expand/collapse toggle

#### 2.2.5 Diff View
- Line numbers (old on left, new on right for split view)
- Hunk headers (@@ -x,y +a,b @@)
- Context lines (no highlight)
- Added lines (green background)
- Deleted lines (red background)
- Syntax highlighting

### 2.3 CSS Classes

```css
/* Commit header */
.commit-header-box {
  padding: 16px;
  background-color: var(--color-bg-subtle);
  border: 1px solid var(--color-border-default);
  border-radius: var(--border-radius-md);
  margin-bottom: 16px;
}

.commit-header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-fg-muted);
  margin-bottom: 16px;
}

/* Commit title */
.commit-title {
  font-size: 24px;
  font-weight: 600;
  line-height: 1.25;
  margin-bottom: 16px;
  word-break: break-word;
}

/* Author info */
.commit-author-info {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 14px;
  color: var(--color-fg-muted);
}

.commit-author-info .Avatar {
  width: 20px;
  height: 20px;
}

.commit-author-info a {
  color: var(--color-fg-default);
  font-weight: 600;
}

/* Verified badge */
.commit-verified {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-success-fg);
  background-color: var(--color-success-subtle);
  border: 1px solid var(--color-success-emphasis);
  border-radius: 2em;
}

/* Parent commits */
.commit-parents {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border-muted);
  font-size: 14px;
  color: var(--color-fg-muted);
}

.commit-parent-sha {
  font-family: var(--font-family-mono);
  font-size: 12px;
  padding: 2px 6px;
  background-color: var(--color-bg-subtle);
  border-radius: var(--border-radius-sm);
}

/* Commit message body */
.commit-body {
  margin-top: 16px;
  padding: 16px;
  font-family: var(--font-family-mono);
  font-size: 14px;
  white-space: pre-wrap;
  background-color: var(--color-bg-subtle);
  border: 1px solid var(--color-border-default);
  border-radius: var(--border-radius-md);
}

/* Stats summary */
.diffstat-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background-color: var(--color-bg-subtle);
  border: 1px solid var(--color-border-default);
  border-radius: var(--border-radius-md);
  margin-bottom: 16px;
}

.diffstat-text {
  font-size: 14px;
  color: var(--color-fg-default);
}

.diffstat-add {
  color: var(--color-success-fg);
  font-weight: 600;
}

.diffstat-del {
  color: var(--color-danger-fg);
  font-weight: 600;
}

/* File header */
.file-header {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  background-color: var(--color-bg-subtle);
  border: 1px solid var(--color-border-default);
  border-radius: var(--border-radius-md) var(--border-radius-md) 0 0;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.file-icon {
  flex-shrink: 0;
}

/* File status icons */
.file-status-added { color: var(--color-success-fg); }
.file-status-modified { color: var(--color-warning-fg); }
.file-status-removed { color: var(--color-danger-fg); }
.file-status-renamed { color: var(--color-fg-muted); }

.file-path {
  font-family: var(--font-family-mono);
  font-size: 12px;
  color: var(--color-fg-default);
  text-overflow: ellipsis;
  overflow: hidden;
  white-space: nowrap;
}

.file-path a {
  color: var(--color-fg-default);
}

.file-path a:hover {
  color: var(--color-accent-fg);
  text-decoration: underline;
}

.file-stats {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  margin-left: auto;
}

/* Diff table */
.diff-table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-family-mono);
  font-size: 12px;
  line-height: 20px;
  table-layout: fixed;
}

.diff-table td {
  padding: 0 10px;
  vertical-align: top;
}

/* Line numbers */
.diff-line-num {
  width: 1%;
  min-width: 50px;
  padding: 0 10px;
  text-align: right;
  color: var(--color-fg-muted);
  background-color: var(--color-bg-subtle);
  user-select: none;
  vertical-align: top;
  border-right: 1px solid var(--color-border-muted);
}

.diff-line-num:hover {
  color: var(--color-accent-fg);
  cursor: pointer;
}

/* Line content */
.diff-line-code {
  padding: 0 10px;
  white-space: pre;
  overflow-x: auto;
  word-wrap: normal;
}

/* Addition line */
.diff-addition {
  background-color: var(--color-success-subtle);
}

.diff-addition .diff-line-num {
  background-color: rgba(46, 160, 67, 0.15);
  border-color: rgba(46, 160, 67, 0.3);
}

.diff-addition .diff-line-code {
  background-color: rgba(46, 160, 67, 0.15);
}

/* Deletion line */
.diff-deletion {
  background-color: var(--color-danger-subtle);
}

.diff-deletion .diff-line-num {
  background-color: rgba(248, 81, 73, 0.15);
  border-color: rgba(248, 81, 73, 0.3);
}

.diff-deletion .diff-line-code {
  background-color: rgba(248, 81, 73, 0.15);
}

/* Hunk header */
.diff-hunk {
  color: var(--color-fg-muted);
  background-color: var(--color-accent-subtle);
}

.diff-hunk .diff-line-num {
  background-color: var(--color-accent-subtle);
}

.diff-hunk .diff-line-code {
  background-color: var(--color-accent-subtle);
  color: var(--color-accent-fg);
}

/* Diff marker (+/-) */
.diff-marker {
  display: inline-block;
  width: 1ch;
  user-select: none;
}

.diff-marker-add {
  color: var(--color-success-fg);
}

.diff-marker-del {
  color: var(--color-danger-fg);
}

/* File container */
.file-diff-container {
  margin-bottom: 16px;
  border: 1px solid var(--color-border-default);
  border-radius: var(--border-radius-md);
  overflow: hidden;
}

.file-diff-container .file-header {
  border: none;
  border-radius: 0;
  border-bottom: 1px solid var(--color-border-default);
}

/* Expand/collapse */
.diff-expander {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  background-color: var(--color-bg-subtle);
  border-top: 1px solid var(--color-border-muted);
  border-bottom: 1px solid var(--color-border-muted);
  color: var(--color-fg-muted);
  font-size: 12px;
  cursor: pointer;
}

.diff-expander:hover {
  background-color: var(--color-accent-subtle);
  color: var(--color-accent-fg);
}
```

### 2.4 Octicons Used

| Icon | Name | Usage |
|------|------|-------|
| git-commit | Commit header icon |
| file | Regular file |
| file-diff | Modified file |
| diff-added | Added file |
| diff-removed | Removed file |
| diff-renamed | Renamed file |
| chevron-down/up | Expand/collapse |
| verified | Verified signature |
| copy | Copy SHA |
| code | Browse files |

#### Additional Octicon Definitions

```html
<!-- file-diff (for modified files) -->
<svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
  <path d="M2.75 1.5a.25.25 0 0 0-.25.25v12.5c0 .138.112.25.25.25h10.5a.25.25 0 0 0 .25-.25V4.664a.25.25 0 0 0-.073-.177l-2.914-2.914a.25.25 0 0 0-.177-.073ZM2.75 0h6.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 12.25 16H2.75A1.75 1.75 0 0 1 1 14.25V1.75C1 .784 1.784 0 2.75 0Zm4.5 8.5a.75.75 0 0 0 0 1.5h2a.75.75 0 0 0 0-1.5Zm-2.72 1.28a.749.749 0 0 1 0-1.06.749.749 0 0 1 1.06 0l.72.72.72-.72a.749.749 0 0 1 1.06 1.06l-.72.72.72.72a.749.749 0 0 1-1.06 1.06l-.72-.72-.72.72a.749.749 0 0 1-1.06-1.06l.72-.72Z"/>
</svg>

<!-- diff-added -->
<svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
  <path d="M2.75 1.5a.25.25 0 0 0-.25.25v12.5c0 .138.112.25.25.25h10.5a.25.25 0 0 0 .25-.25V4.664a.25.25 0 0 0-.073-.177l-2.914-2.914a.25.25 0 0 0-.177-.073ZM2.75 0h6.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 12.25 16H2.75A1.75 1.75 0 0 1 1 14.25V1.75C1 .784 1.784 0 2.75 0Zm4.5 8.75a.75.75 0 0 1 .75-.75h1.5a.75.75 0 0 1 0 1.5H8v1.5a.75.75 0 0 1-1.5 0V9.5H5a.75.75 0 0 1 0-1.5h1.5V6.5a.75.75 0 0 1 1.5 0v1.5h1.5Z"/>
</svg>

<!-- diff-removed -->
<svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
  <path d="M2.75 1.5a.25.25 0 0 0-.25.25v12.5c0 .138.112.25.25.25h10.5a.25.25 0 0 0 .25-.25V4.664a.25.25 0 0 0-.073-.177l-2.914-2.914a.25.25 0 0 0-.177-.073ZM2.75 0h6.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 12.25 16H2.75A1.75 1.75 0 0 1 1 14.25V1.75C1 .784 1.784 0 2.75 0ZM5 9.25a.75.75 0 0 1 .75-.75h4.5a.75.75 0 0 1 0 1.5h-4.5a.75.75 0 0 1-.75-.75Z"/>
</svg>

<!-- verified -->
<svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
  <path d="m9.585.52.929.68c.153.112.331.186.518.215l1.138.175a2.678 2.678 0 0 1 2.24 2.24l.174 1.139c.029.187.103.365.215.518l.68.928a2.677 2.677 0 0 1 0 3.17l-.68.928a1.174 1.174 0 0 0-.215.518l-.175 1.139a2.678 2.678 0 0 1-2.241 2.24l-1.138.175a1.17 1.17 0 0 0-.518.215l-.928.68a2.677 2.677 0 0 1-3.17 0l-.928-.68a1.174 1.174 0 0 0-.518-.215L3.83 14.41a2.678 2.678 0 0 1-2.24-2.24l-.175-1.138a1.17 1.17 0 0 0-.215-.518l-.68-.928a2.677 2.677 0 0 1 0-3.17l.68-.928c.112-.153.186-.331.215-.518l.175-1.14a2.678 2.678 0 0 1 2.24-2.24l1.139-.175c.187-.029.365-.103.518-.215l.928-.68a2.677 2.677 0 0 1 3.17 0ZM7.303 11.03l3.384-3.384a.75.75 0 0 0-1.061-1.06L6.772 9.438l-1.09-1.09a.75.75 0 0 0-1.061 1.061l1.621 1.621a.75.75 0 0 0 1.061 0Z"/>
</svg>
```

### 2.5 Data Structure

```go
// CommitDetailData holds data for single commit view
type CommitDetailData struct {
    Title            string
    User             *users.User
    Repo             *RepoView
    Commit           *CommitViewDetail
    Files            []*FileChangeView
    Stats            *StatsView
    Branches         []*branches.Branch
    Tags             []string
    Breadcrumbs      []Breadcrumb
    UnreadCount      int
    ActiveNav        string
}

// CommitViewDetail represents full commit details
type CommitViewDetail struct {
    SHA              string
    ShortSHA         string
    Message          string
    MessageTitle     string
    MessageBody      string          // Everything after first line
    Author           *UserView
    Committer        *UserView
    AuthorDate       time.Time
    CommitterDate    time.Time
    TimeAgo          string
    IsSameAuthor     bool
    Parents          []*ParentCommit
    TreeSHA          string
    TreeURL          string
    Verified         bool
    VerifiedReason   string
    HTMLURL          string
    URL              string
}

// ParentCommit represents a parent reference
type ParentCommit struct {
    SHA      string
    ShortSHA string
    HTMLURL  string
}

// FileChangeView represents a changed file
type FileChangeView struct {
    SHA              string
    Filename         string
    PreviousFilename string          // For renames
    Status           string          // added, removed, modified, renamed
    StatusIcon       string          // Octicon name
    StatusClass      string          // CSS class
    Additions        int
    Deletions        int
    Changes          int
    BlobURL          string
    RawURL           string
    Patch            string          // Raw patch content
    DiffLines        []*DiffLine     // Parsed diff lines
    IsBinary         bool
    IsCollapsed      bool            // For large files
    TooLarge         bool            // File too large to display
}

// DiffLine represents a single line in a diff
type DiffLine struct {
    Type         string    // context, addition, deletion, hunk
    OldLineNum   int       // Line number in old file (0 if N/A)
    NewLineNum   int       // Line number in new file (0 if N/A)
    Content      string    // Line content (without +/- prefix)
    HTMLContent  template.HTML // Syntax highlighted
}

// StatsView represents commit statistics
type StatsView struct {
    FilesChanged int
    Additions    int
    Deletions    int
    Total        int
}
```

---

## 3. Routes

### 3.1 URL Patterns

| Route | Handler | Description |
|-------|---------|-------------|
| `GET /:owner/:repo/commits` | RepoCommits | Commit list (default branch) |
| `GET /:owner/:repo/commits/:ref` | RepoCommits | Commit list for ref |
| `GET /:owner/:repo/commits/:ref/*path` | RepoCommits | Commit list for path |
| `GET /:owner/:repo/commit/:sha` | CommitDetail | Single commit view |

### 3.2 Query Parameters

**Commit List:**
- `page` - Page number (default: 1)
- `per_page` - Items per page (default: 35, max: 100)
- `author` - Filter by author login
- `committer` - Filter by committer login
- `since` - Filter commits after date (ISO 8601)
- `until` - Filter commits before date (ISO 8601)

---

## 4. Template Files

### 4.1 Commits List Template

Create: `assets/views/default/pages/repo_commits.html`

### 4.2 Commit Detail Template

Create: `assets/views/default/pages/commit_detail.html`

---

## 5. Handler Methods

### 5.1 RepoCommits Handler

```go
// RepoCommits renders the commits list page
func (h *Page) RepoCommits(c *mizu.Ctx) error {
    owner := c.Param("owner")
    repoName := c.Param("repo")
    ref := c.Param("ref")
    path := c.Param("path")

    // Default to main branch if not specified
    // Get commits with pagination
    // Group commits by date
    // Build view data
    // Render template
}
```

### 5.2 CommitDetail Handler

```go
// CommitDetail renders a single commit view
func (h *Page) CommitDetail(c *mizu.Ctx) error {
    owner := c.Param("owner")
    repoName := c.Param("repo")
    sha := c.Param("sha")

    // Get commit with files and stats
    // Parse diff patches into structured view
    // Handle syntax highlighting
    // Build view data
    // Render template
}
```

---

## 6. JavaScript Interactions

### 6.1 Copy SHA to Clipboard

```javascript
function copySHA(sha, button) {
    navigator.clipboard.writeText(sha).then(() => {
        const icon = button.querySelector('.octicon');
        // Show checkmark temporarily
        const originalHTML = icon.innerHTML;
        icon.innerHTML = '...checkmark path...';
        setTimeout(() => {
            icon.innerHTML = originalHTML;
        }, 2000);
    });
}
```

### 6.2 Expand/Collapse Commit Message

```javascript
function toggleCommitBody(button) {
    const body = document.querySelector('.commit-body-full');
    const isExpanded = body.style.display !== 'none';
    body.style.display = isExpanded ? 'none' : 'block';
    button.textContent = isExpanded ? 'Show more' : 'Show less';
}
```

### 6.3 File Diff Toggle

```javascript
function toggleFileDiff(fileId) {
    const diff = document.getElementById('diff-' + fileId);
    const toggle = document.getElementById('toggle-' + fileId);
    const isHidden = diff.style.display === 'none';
    diff.style.display = isHidden ? 'block' : 'none';
    toggle.classList.toggle('expanded', isHidden);
}
```

---

## 7. API Endpoints (for AJAX)

These existing API endpoints support the UI:

| Endpoint | Usage |
|----------|-------|
| `GET /api/v3/repos/:owner/:repo/commits` | Fetch commits list |
| `GET /api/v3/repos/:owner/:repo/commits/:ref` | Fetch single commit |

---

## 8. Testing with golang/go

### 8.1 Test Cases

1. **Commit List Page**
   - Visit `/golang/go/commits/master`
   - Verify date grouping matches GitHub
   - Verify avatar display for authors/committers
   - Verify commit message truncation
   - Verify pagination works

2. **Single Commit View**
   - Visit a specific commit
   - Verify file changes display
   - Verify diff coloring (green/red)
   - Verify line numbers
   - Verify expand/collapse
   - Verify copy SHA functionality

### 8.2 Visual Comparison Checklist

- [ ] Date group headers match GitHub style
- [ ] Avatar sizing and positioning match
- [ ] Commit message font and color match
- [ ] SHA display style matches
- [ ] Button hover states match
- [ ] Diff colors match exactly
- [ ] Line number styling matches
- [ ] File header styling matches
- [ ] Stats bar styling matches
- [ ] Pagination styling matches

---

## 9. Implementation Order

1. Add CSS styles to `main.css`
2. Create `repo_commits.html` template
3. Create `commit_detail.html` template
4. Add handler methods to `page.go`
5. Add routes to `server.go`
6. Add diff parsing utilities
7. Test with golang/go repository
8. Fine-tune styling to match GitHub exactly

---

## 10. Color Reference

### Diff Colors

| Element | Light Mode | Dark Mode |
|---------|------------|-----------|
| Addition bg | `rgba(46, 160, 67, 0.15)` | `rgba(46, 160, 67, 0.15)` |
| Addition strong | `rgba(46, 160, 67, 0.4)` | `rgba(46, 160, 67, 0.4)` |
| Deletion bg | `rgba(248, 81, 73, 0.15)` | `rgba(248, 81, 73, 0.15)` |
| Deletion strong | `rgba(248, 81, 73, 0.4)` | `rgba(248, 81, 73, 0.4)` |
| Hunk header bg | `var(--color-accent-subtle)` | `var(--color-accent-subtle)` |

### File Status Colors

| Status | Color |
|--------|-------|
| Added | `--color-success-fg` (#1a7f37) |
| Modified | `--color-warning-fg` (#9a6700) |
| Removed | `--color-danger-fg` (#d1242f) |
| Renamed | `--color-fg-muted` (#656d76) |
