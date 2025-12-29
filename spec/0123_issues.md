# Spec 0123: GitHub-like Issues View

## Overview

Implement an issues view that matches GitHub's issues interface 100% for visual accuracy and functionality. This includes the issues list page, single issue view, filters, search, and all associated UI elements.

## Goals

1. Visual parity with GitHub's issues interface
2. Full functionality for filtering, sorting, searching issues
3. Support for labels, milestones, assignees
4. Responsive design matching GitHub's breakpoints
5. Accessible and keyboard-navigable

## GitHub Issues UI Analysis

### Issues List Page (`/{owner}/{repo}/issues`)

#### Header Section
- Repository breadcrumb: `owner / repo`
- Repository navigation tabs (Code, Issues, Pull requests, etc.)
- Issues tab shows count badge

#### Issues Toolbar
```
[Filters ▼] [Search box: is:issue is:open          ] [Labels] [Milestones] [New issue]
```

- **Filters dropdown**: Quick filters (Open issues, Your issues, Assigned, etc.)
- **Search input**: Full-text search with qualifier support (`is:issue`, `is:open`, `label:bug`, etc.)
- **Labels button**: Opens label filter dropdown
- **Milestones button**: Opens milestone filter dropdown
- **New issue button**: Green primary button (requires auth)

#### State Toggle
```
[● Open (5,234)]  [✓ Closed (82,145)]
```
- Open issues icon: Circle with dot
- Closed issues icon: Circle with checkmark
- Selected state has bottom border accent
- Shows count for each state

#### Filter Bar (when filters active)
```
Clear current search query, filters, and sorts
```

#### Issues List Box
Each issue row contains:
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ ● Issue Title Here [label1] [label2] [label3]                    💬 23     │
│   #12345 opened 2 hours ago by username                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

- **Status icon**: Green circle-dot for open, purple check-circle for closed
- **Title**: Bold, clickable link
- **Labels**: Colored badges with label name, display inline after title
- **Comment count**: Speech bubble icon + count (right-aligned)
- **Meta line**: Issue number, "opened/closed" + relative time + "by" + author link
- **Milestone badge**: If assigned, shows milestone name
- **Assignee avatars**: Small circular avatars on the right (up to 3)

#### Pagination
```
[← Previous]                                                      [Next →]
```

### Issue Labels

Labels are displayed as:
- Rounded pill shape (border-radius: 2em)
- Background color from label.color
- Text color: white or black depending on contrast
- Font size: 12px
- Padding: 0 7px
- Font weight: 500

### Empty State

When no issues match:
```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│                            [Issue icon - large]                             │
│                                                                             │
│                    Welcome to issues!                                       │
│                                                                             │
│          Issues are used to track todos, bugs, feature requests,            │
│                          and more.                                          │
│                                                                             │
│                         [New issue button]                                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Single Issue View (`/{owner}/{repo}/issues/{number}`)

#### Issue Header
```
Issue Title Here #12345
[Open] username opened this issue 2 hours ago · 23 comments
```

- Title: Large heading (20px+)
- Issue number: Muted color
- State badge: Green "Open" or Purple "Closed" pill
- Meta: Author link + relative time + comment count

#### Issue Body
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ [Avatar] username commented 2 hours ago                        [···] Edit  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ Markdown rendered body content here...                                      │
│                                                                             │
│ - Lists                                                                     │
│ - Code blocks                                                               │
│ - Images                                                                    │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ 👍 12  ❤️ 5  🎉 3                                          Add reaction    │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Sidebar (Right)
```
┌─────────────────────────┐
│ Assignees               │
│ No one—assign yourself  │
├─────────────────────────┤
│ Labels                  │
│ None yet                │
├─────────────────────────┤
│ Projects                │
│ None yet                │
├─────────────────────────┤
│ Milestone               │
│ No milestone            │
├─────────────────────────┤
│ Development             │
│ No branches or PRs      │
├─────────────────────────┤
│ Notifications           │
│ [Subscribe] [Unsubscribe]│
├─────────────────────────┤
│ 2 participants          │
│ [avatar] [avatar]       │
└─────────────────────────┘
```

#### Timeline Events
```
[avatar] username added the bug label 1 hour ago
[avatar] username added this to the v1.0 milestone 1 hour ago
[avatar] username self-assigned this 30 minutes ago
```

#### Comments
Same format as issue body, with timeline position

#### New Comment Form
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ [Avatar] [Write] [Preview]                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────────────────────┐ │
│ │ Leave a comment                                                         │ │
│ │                                                                         │ │
│ │                                                                         │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ [Attach files by dragging...]              [Close issue] [Comment]         │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Implementation Plan

### 1. Update CSS (`assets/static/css/main.css`)

Add/update the following classes:

```css
/* Issues list toolbar */
.issues-toolbar { ... }
.issues-search-input { ... }
.issues-filter-dropdown { ... }

/* Issues state toggle */
.issues-state-toggle { ... }
.issues-state-toggle-item { ... }
.issues-state-toggle-item.selected { ... }

/* Issues list */
.issues-list { ... }
.issues-list-item { ... }
.issues-list-item:hover { ... }
.issues-list-item-icon { ... }
.issues-list-item-content { ... }
.issues-list-item-title { ... }
.issues-list-item-meta { ... }
.issues-list-item-actions { ... }

/* Issue labels */
.IssueLabel { ... }
.IssueLabel--big { ... }

/* Issue state badge */
.State { ... }
.State--open { ... }
.State--closed { ... }

/* Issue detail */
.issue-header { ... }
.issue-title { ... }
.issue-meta { ... }

/* Timeline comment */
.timeline-comment { ... }
.timeline-comment-header { ... }
.timeline-comment-body { ... }

/* Issue sidebar */
.issue-sidebar { ... }
.issue-sidebar-section { ... }

/* Empty state */
.issues-blankslate { ... }
```

### 2. Update Issues List Template (`repo_issues.html`)

Key changes:
- Add search/filter toolbar
- Add state toggle (Open/Closed) with icons and counts
- Enhance issue row with all GitHub elements
- Add assignee avatars
- Add comment count with icon
- Add empty state for no issues
- Add pagination

### 3. Update Issue Detail Template (`issue_view.html`)

Key changes:
- Add issue header with state badge
- Add sidebar layout (main content + sidebar)
- Add timeline events
- Add comment form
- Add reactions display
- Add markdown rendering for body

### 4. Update Page Handler (`app/web/handler/page.go`)

Enhance `RepoIssuesData` to include:
- Total open count
- Total closed count
- Current filters
- Pagination info
- Search query

Enhance `IssueDetailData` to include:
- Timeline events
- Participants list
- Subscription status

### 5. Update Issues Store (if needed)

May need to add:
- `CountByState(ctx, repoID, state string) (int, error)`
- `ListWithFilters(ctx, repoID, opts *ListOpts) ([]*Issue, error)`

## Color Reference (GitHub Primer)

```css
/* Issue states */
--color-open-fg: #1a7f37;        /* Green for open */
--color-open-emphasis: #1f883d;
--color-closed-fg: #8250df;      /* Purple for closed */
--color-closed-emphasis: #8250df;

/* Labels - computed based on background */
/* Light backgrounds get dark text */
/* Dark backgrounds get white text */
```

## Verification Checklist

### Issues List Page
- [ ] Repository header with tabs matches GitHub
- [ ] "New issue" button is green and positioned correctly
- [ ] Open/Closed toggle with correct icons and counts
- [ ] Issue rows show: icon, title, labels, comment count
- [ ] Issue meta shows: number, "opened/closed X ago by author"
- [ ] Labels are colored pills with correct contrast text
- [ ] Assignee avatars show on right side
- [ ] Hover state on issue rows
- [ ] Empty state when no issues
- [ ] Pagination controls

### Single Issue Page
- [ ] Issue title with number in header
- [ ] State badge (Open/Closed) with correct colors
- [ ] Author and timestamp in meta
- [ ] Issue body rendered as markdown
- [ ] Sidebar with Assignees, Labels, Milestone sections
- [ ] Timeline events for state changes, labels, etc.
- [ ] Comments with author avatars and timestamps
- [ ] Comment form at bottom
- [ ] Close/Reopen button (for authorized users)

### Labels
- [ ] Colored background from label.color
- [ ] Automatic contrast text color (white/black)
- [ ] Pill shape with correct border-radius
- [ ] Hover state shows label description

### Responsive
- [ ] Desktop layout (sidebar on right)
- [ ] Tablet layout (sidebar moves below)
- [ ] Mobile layout (simplified, stacked)

## Testing

1. Seed golang/go issues:
   ```bash
   githome seed github golang/go --max-issues 100 --max-comments 10
   ```

2. Start server:
   ```bash
   githome serve
   ```

3. Visit http://localhost:3000/golang/go/issues

4. Compare side-by-side with https://github.com/golang/go/issues

## Files Modified

1. `assets/static/css/main.css` - Added comprehensive issues view CSS (~200 lines)
2. `assets/views/default/pages/repo_issues.html` - Complete rewrite for GitHub-like design
3. `assets/views/default/pages/issue_view.html` - Complete rewrite with sidebar and comments
4. `assets/embed.go` - Added `contrastColor` template function
5. `app/web/handler/page.go` - Added Query field, pagination, accurate counts
6. `feature/issues/api.go` - Added `CountByState` to API interface
7. `feature/issues/service.go` - Implemented `CountByState` method
8. `store/duckdb/issues_store.go` - Added `CountByState` store method

## Verification Checklist

### Issues List Page (`/{owner}/{repo}/issues`)
- [x] Repository header with owner/repo breadcrumb
- [x] Repository navigation tabs (Code, Issues, Pull requests, etc.)
- [x] "New issue" green primary button
- [x] Search input with placeholder "is:issue is:open/closed"
- [x] Labels and Milestones filter buttons
- [x] Open/Closed toggle tabs with accurate counts
- [x] Open icon: Circle with dot (green)
- [x] Closed icon: Circle with checkmark (purple)
- [x] Sort dropdown (Newest, Oldest, Most commented, Recently updated)
- [x] Issue rows showing:
  - [x] State icon (open/closed)
  - [x] Title as clickable link
  - [x] Labels with colored backgrounds and contrast text
  - [x] Comment count with speech bubble icon
  - [x] Meta line: #number, opened/closed X ago by author
  - [x] Milestone badge (when assigned)
  - [x] Assignee avatars (right-aligned)
- [x] Hover state on issue rows
- [x] Empty state when no issues
- [x] Pagination (Previous/Next buttons)

### Single Issue Page (`/{owner}/{repo}/issues/{number}`)
- [x] Issue title with number in header
- [x] State badge (Open/Closed) with correct colors/icons
- [x] Author and timestamp in meta
- [x] Issue body in timeline comment box
- [x] Avatar with author name
- [x] Sidebar with sections:
  - [x] Assignees
  - [x] Labels
  - [x] Projects (placeholder)
  - [x] Milestone
  - [x] Development (placeholder)
  - [x] Notifications (for logged-in users)
  - [x] Participants
- [x] Comments with author avatars and timestamps
- [x] Comment form at bottom (for logged-in users)
- [x] Close/Reopen button (for authorized users)
- [x] Sign in prompt for unauthenticated users

### Labels
- [x] Colored background from label.color
- [x] Automatic contrast text color (white/black)
- [x] Pill shape with border-radius: 2em

### Responsive Design
- [x] Desktop layout (sidebar on right)
- [x] Tablet layout (actions hidden)
- [x] Mobile layout (simplified, issue icon hidden)

### CSS Classes Implemented
- `.issues-header-toolbar` - Top toolbar with search and filters
- `.issues-search-*` - Search input group
- `.issues-filter-*` - Filter buttons
- `.issues-state-toggle` - Open/Closed toggle
- `.issues-list` - Issues list container
- `.issues-list-item` - Individual issue row
- `.issues-list-item-*` - Issue row components
- `.IssueLabel` - Label pill styling
- `.State`, `.State--open`, `.State--closed` - State badges
- `.timeline-comment` - Comment box styling
- `.timeline-event` - Timeline event styling
- `.BorderGrid` - Sidebar grid styling
- `.blankslate` - Empty state styling
- `.SelectMenu` - Dropdown menu styling
- `.issue-title` - Issue title header
- `.issue-meta` - Issue metadata
- `.flash` - Flash messages

## Dependencies

- Existing Primer CSS design system (colors, spacing, typography)
- GitHub Octicons for icons (inline SVGs)
- goldmark for markdown rendering (already imported)
