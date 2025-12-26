# Blueprint: Kanban - Project Management System

## Overview

A full-featured project management system inspired by Linear, Trello, and Jira. Features a clean, modern UI with elegant themes inspired by shadcn and ChatGPT's design language. Built for teams to organize work with boards, issues, sprints, and real-time collaboration.

## Design Philosophy

- **Linear-inspired**: Fast, keyboard-driven, minimal UI with focus on content
- **shadcn aesthetics**: Subtle gradients, refined spacing, muted colors with high contrast text
- **ChatGPT elegance**: Clean typography, generous whitespace, smooth animations
- **Dark-first**: Default dark theme with light mode option

## Core Features

### 1. Workspaces
- Multi-tenant workspace isolation
- Workspace settings and branding
- Member management with roles (Owner, Admin, Member, Guest)
- Workspace-level integrations

### 2. Projects
- Multiple projects per workspace
- Project-specific settings
- Project leads and team assignment
- Project templates
- Archive/restore functionality

### 3. Issues (Tasks/Tickets)
- Rich text description with markdown support
- Priority levels (Urgent, High, Medium, Low, None)
- Status workflow (Backlog, Todo, In Progress, In Review, Done, Cancelled)
- Labels/tags with custom colors
- Assignees (multiple)
- Due dates and time estimates
- Parent/child relationships (epics, stories, tasks, subtasks)
- Issue linking (blocks, blocked by, relates to, duplicates)
- Attachments
- Comments with mentions
- Activity history
- Custom fields

### 4. Board Views
- **Kanban Board**: Drag-and-drop columns by status
- **List View**: Traditional table/list with sorting and grouping
- **Calendar View**: Issues by due date
- **Timeline View**: Gantt-style chart (optional)

### 5. Sprints & Cycles
- Sprint planning with capacity
- Sprint backlog management
- Sprint burndown tracking
- Cycle analytics

### 6. Search & Filters
- Global search across issues
- Advanced filtering (status, assignee, priority, labels, dates)
- Saved filters/views
- Recent items

### 7. Notifications
- In-app notifications
- Mention notifications
- Assignment notifications
- Due date reminders

### 8. User Management
- User profiles
- User preferences
- Keyboard shortcuts

## Data Models

### Workspace
```go
type Workspace struct {
    ID          string    `json:"id"`
    Slug        string    `json:"slug"`        // URL-friendly identifier
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    AvatarURL   string    `json:"avatar_url,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### WorkspaceMember
```go
type WorkspaceMember struct {
    ID          string    `json:"id"`
    WorkspaceID string    `json:"workspace_id"`
    UserID      string    `json:"user_id"`
    Role        string    `json:"role"` // owner, admin, member, guest
    JoinedAt    time.Time `json:"joined_at"`
}
```

### Project
```go
type Project struct {
    ID          string    `json:"id"`
    WorkspaceID string    `json:"workspace_id"`
    Key         string    `json:"key"`         // e.g., "PROJ" for issue prefixes
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    Color       string    `json:"color"`       // Project accent color
    LeadID      string    `json:"lead_id,omitempty"`
    Status      string    `json:"status"`      // active, archived, completed
    StartDate   *time.Time `json:"start_date,omitempty"`
    TargetDate  *time.Time `json:"target_date,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Issue
```go
type Issue struct {
    ID          string     `json:"id"`
    ProjectID   string     `json:"project_id"`
    Number      int        `json:"number"`      // Auto-increment per project
    Key         string     `json:"key"`         // e.g., "PROJ-123"
    Title       string     `json:"title"`
    Description string     `json:"description,omitempty"`
    Type        string     `json:"type"`        // epic, story, task, bug, subtask
    Status      string     `json:"status"`      // backlog, todo, in_progress, in_review, done, cancelled
    Priority    string     `json:"priority"`    // urgent, high, medium, low, none
    ParentID    string     `json:"parent_id,omitempty"`
    CreatorID   string     `json:"creator_id"`
    AssigneeIDs []string   `json:"assignee_ids,omitempty"`
    LabelIDs    []string   `json:"label_ids,omitempty"`
    SprintID    string     `json:"sprint_id,omitempty"`
    DueDate     *time.Time `json:"due_date,omitempty"`
    Estimate    *int       `json:"estimate,omitempty"` // Story points or hours
    Position    int        `json:"position"`    // For ordering within status
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}
```

### Label
```go
type Label struct {
    ID          string `json:"id"`
    ProjectID   string `json:"project_id"`
    Name        string `json:"name"`
    Color       string `json:"color"`
    Description string `json:"description,omitempty"`
}
```

### Comment
```go
type Comment struct {
    ID        string    `json:"id"`
    IssueID   string    `json:"issue_id"`
    AuthorID  string    `json:"author_id"`
    Content   string    `json:"content"`  // Markdown
    EditedAt  *time.Time `json:"edited_at,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Sprint
```go
type Sprint struct {
    ID          string     `json:"id"`
    ProjectID   string     `json:"project_id"`
    Name        string     `json:"name"`
    Goal        string     `json:"goal,omitempty"`
    Status      string     `json:"status"` // planning, active, completed
    StartDate   *time.Time `json:"start_date,omitempty"`
    EndDate     *time.Time `json:"end_date,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
}
```

### IssueLink
```go
type IssueLink struct {
    ID            string `json:"id"`
    SourceIssueID string `json:"source_issue_id"`
    TargetIssueID string `json:"target_issue_id"`
    LinkType      string `json:"link_type"` // blocks, blocked_by, relates_to, duplicates
}
```

### Activity
```go
type Activity struct {
    ID        string    `json:"id"`
    IssueID   string    `json:"issue_id"`
    ActorID   string    `json:"actor_id"`
    Action    string    `json:"action"`     // created, updated, commented, etc.
    Field     string    `json:"field,omitempty"`
    OldValue  string    `json:"old_value,omitempty"`
    NewValue  string    `json:"new_value,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}
```

### User
```go
type User struct {
    ID          string    `json:"id"`
    Email       string    `json:"email"`
    Username    string    `json:"username"`
    DisplayName string    `json:"display_name"`
    AvatarURL   string    `json:"avatar_url,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Notification
```go
type Notification struct {
    ID         string    `json:"id"`
    UserID     string    `json:"user_id"`
    Type       string    `json:"type"`       // mention, assignment, due_date, comment
    IssueID    string    `json:"issue_id,omitempty"`
    ActorID    string    `json:"actor_id,omitempty"`
    Content    string    `json:"content"`
    ReadAt     *time.Time `json:"read_at,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
}
```

## API Endpoints

### Authentication
```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
GET    /api/v1/auth/me
```

### Workspaces
```
GET    /api/v1/workspaces
POST   /api/v1/workspaces
GET    /api/v1/workspaces/{slug}
PATCH  /api/v1/workspaces/{slug}
DELETE /api/v1/workspaces/{slug}
GET    /api/v1/workspaces/{slug}/members
POST   /api/v1/workspaces/{slug}/members
DELETE /api/v1/workspaces/{slug}/members/{id}
```

### Projects
```
GET    /api/v1/workspaces/{slug}/projects
POST   /api/v1/workspaces/{slug}/projects
GET    /api/v1/workspaces/{slug}/projects/{key}
PATCH  /api/v1/workspaces/{slug}/projects/{key}
DELETE /api/v1/workspaces/{slug}/projects/{key}
```

### Issues
```
GET    /api/v1/projects/{key}/issues
POST   /api/v1/projects/{key}/issues
GET    /api/v1/issues/{key}
PATCH  /api/v1/issues/{key}
DELETE /api/v1/issues/{key}
POST   /api/v1/issues/{key}/move          # Change status/position
GET    /api/v1/issues/{key}/comments
POST   /api/v1/issues/{key}/comments
GET    /api/v1/issues/{key}/activity
POST   /api/v1/issues/{key}/links
DELETE /api/v1/issues/{key}/links/{id}
```

### Labels
```
GET    /api/v1/projects/{key}/labels
POST   /api/v1/projects/{key}/labels
PATCH  /api/v1/projects/{key}/labels/{id}
DELETE /api/v1/projects/{key}/labels/{id}
```

### Sprints
```
GET    /api/v1/projects/{key}/sprints
POST   /api/v1/projects/{key}/sprints
PATCH  /api/v1/projects/{key}/sprints/{id}
DELETE /api/v1/projects/{key}/sprints/{id}
POST   /api/v1/projects/{key}/sprints/{id}/start
POST   /api/v1/projects/{key}/sprints/{id}/complete
```

### Search & Filters
```
GET    /api/v1/search                     # Global search
GET    /api/v1/projects/{key}/issues/filter
```

### Notifications
```
GET    /api/v1/notifications
POST   /api/v1/notifications/read
POST   /api/v1/notifications/read-all
```

### Users
```
GET    /api/v1/users/{id}
PATCH  /api/v1/users/{id}
GET    /api/v1/workspaces/{slug}/users   # Members with user details
```

## Web Routes (Pages)

```
GET    /                              # Landing/Dashboard
GET    /login                         # Login page
GET    /register                      # Registration page
GET    /{workspace}                   # Workspace dashboard
GET    /{workspace}/projects          # Projects list
GET    /{workspace}/projects/{key}    # Project board (kanban default)
GET    /{workspace}/projects/{key}/list      # List view
GET    /{workspace}/projects/{key}/calendar  # Calendar view
GET    /{workspace}/projects/{key}/backlog   # Backlog view
GET    /{workspace}/projects/{key}/sprints   # Sprint management
GET    /{workspace}/projects/{key}/settings  # Project settings
GET    /{workspace}/issue/{key}       # Issue detail modal/page
GET    /{workspace}/settings          # Workspace settings
GET    /{workspace}/members           # Team members
GET    /notifications                 # Notifications page
GET    /settings                      # User settings
```

## UI Design Specification

### Color Palette (Dark Theme - Default)

```css
:root {
    /* Backgrounds */
    --bg-primary: #0a0a0b;          /* Main background */
    --bg-secondary: #111113;         /* Cards, panels */
    --bg-tertiary: #18181b;          /* Elevated surfaces */
    --bg-hover: #1f1f23;             /* Hover states */
    --bg-active: #27272a;            /* Active states */

    /* Borders */
    --border-subtle: #27272a;        /* Subtle dividers */
    --border-default: #3f3f46;       /* Default borders */
    --border-strong: #52525b;        /* Emphasized borders */

    /* Text */
    --text-primary: #fafafa;         /* Primary text */
    --text-secondary: #a1a1aa;       /* Secondary text */
    --text-muted: #71717a;           /* Muted text */
    --text-disabled: #52525b;        /* Disabled text */

    /* Accent Colors */
    --accent-primary: #6366f1;       /* Indigo - primary actions */
    --accent-primary-hover: #818cf8;
    --accent-secondary: #8b5cf6;     /* Purple - secondary */

    /* Status Colors */
    --status-backlog: #71717a;       /* Gray */
    --status-todo: #f59e0b;          /* Amber */
    --status-progress: #3b82f6;      /* Blue */
    --status-review: #8b5cf6;        /* Purple */
    --status-done: #22c55e;          /* Green */
    --status-cancelled: #ef4444;     /* Red */

    /* Priority Colors */
    --priority-urgent: #ef4444;      /* Red */
    --priority-high: #f97316;        /* Orange */
    --priority-medium: #eab308;      /* Yellow */
    --priority-low: #3b82f6;         /* Blue */
    --priority-none: #71717a;        /* Gray */

    /* Issue Type Colors */
    --type-epic: #8b5cf6;            /* Purple */
    --type-story: #22c55e;           /* Green */
    --type-task: #3b82f6;            /* Blue */
    --type-bug: #ef4444;             /* Red */
    --type-subtask: #71717a;         /* Gray */
}
```

### Color Palette (Light Theme)

```css
[data-theme="light"] {
    --bg-primary: #ffffff;
    --bg-secondary: #fafafa;
    --bg-tertiary: #f4f4f5;
    --bg-hover: #f4f4f5;
    --bg-active: #e4e4e7;

    --border-subtle: #f4f4f5;
    --border-default: #e4e4e7;
    --border-strong: #d4d4d8;

    --text-primary: #09090b;
    --text-secondary: #52525b;
    --text-muted: #71717a;
    --text-disabled: #a1a1aa;
}
```

### Typography

```css
:root {
    --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
    --font-mono: 'JetBrains Mono', 'Fira Code', monospace;

    --text-xs: 0.6875rem;     /* 11px */
    --text-sm: 0.8125rem;     /* 13px */
    --text-base: 0.875rem;    /* 14px */
    --text-lg: 1rem;          /* 16px */
    --text-xl: 1.125rem;      /* 18px */
    --text-2xl: 1.25rem;      /* 20px */
    --text-3xl: 1.5rem;       /* 24px */

    --font-normal: 400;
    --font-medium: 500;
    --font-semibold: 600;
    --font-bold: 700;

    --leading-tight: 1.25;
    --leading-normal: 1.5;
    --leading-relaxed: 1.625;
}
```

### Spacing

```css
:root {
    --space-0: 0;
    --space-1: 0.25rem;    /* 4px */
    --space-2: 0.5rem;     /* 8px */
    --space-3: 0.75rem;    /* 12px */
    --space-4: 1rem;       /* 16px */
    --space-5: 1.25rem;    /* 20px */
    --space-6: 1.5rem;     /* 24px */
    --space-8: 2rem;       /* 32px */
    --space-10: 2.5rem;    /* 40px */
    --space-12: 3rem;      /* 48px */
    --space-16: 4rem;      /* 64px */
}
```

### Border Radius

```css
:root {
    --radius-sm: 0.25rem;   /* 4px */
    --radius-md: 0.375rem;  /* 6px */
    --radius-lg: 0.5rem;    /* 8px */
    --radius-xl: 0.75rem;   /* 12px */
    --radius-2xl: 1rem;     /* 16px */
    --radius-full: 9999px;
}
```

### Shadows

```css
:root {
    --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.3);
    --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.4);
    --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.5);
    --shadow-xl: 0 20px 25px -5px rgba(0, 0, 0, 0.6);
}
```

### Component Specifications

#### Sidebar Navigation
- Width: 220px (collapsed: 56px)
- Fixed position on desktop
- Contains: workspace selector, navigation items, quick actions
- Keyboard shortcut: `[` to toggle

#### Board View
- Columns for each status
- Column header with count
- Issue cards with:
  - Issue key/number
  - Title (truncated)
  - Priority indicator
  - Assignee avatar
  - Labels (collapsed)
  - Due date indicator
- Drag & drop between columns
- Quick add button in each column

#### Issue Card
- Compact design: ~60px height
- Shows: key, title, priority, assignee
- Hover: reveals more actions
- Click: opens issue detail

#### Issue Detail Panel
- Slide-in panel from right (720px width)
- Or full page for mobile
- Sections:
  - Header: key, title, status selector
  - Properties: assignee, priority, labels, dates
  - Description: markdown editor/viewer
  - Subtasks
  - Comments
  - Activity timeline

#### Command Palette
- Triggered by Cmd/Ctrl + K
- Quick actions: create issue, search, navigate
- Fuzzy search for issues

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `c` | Create new issue |
| `g b` | Go to board |
| `g l` | Go to list view |
| `g s` | Go to sprints |
| `g n` | Go to notifications |
| `Cmd/Ctrl + K` | Command palette |
| `Cmd/Ctrl + /` | Keyboard shortcuts help |
| `j` / `k` | Navigate issues |
| `Enter` | Open selected issue |
| `Esc` | Close panel / Deselect |
| `[` | Toggle sidebar |
| `1-5` | Set priority on selected issue |
| `m` | Assign to me |
| `l` | Add label |

## Project Structure

```
kanban/
├── app/
│   └── web/
│       ├── handler/
│       │   ├── auth.go
│       │   ├── workspace.go
│       │   ├── project.go
│       │   ├── issue.go
│       │   ├── comment.go
│       │   ├── sprint.go
│       │   ├── label.go
│       │   ├── notification.go
│       │   ├── search.go
│       │   └── page.go
│       ├── middleware.go
│       └── server.go
├── assets/
│   ├── static/
│   │   ├── css/
│   │   │   └── app.css
│   │   ├── js/
│   │   │   ├── app.js
│   │   │   ├── board.js
│   │   │   └── shortcuts.js
│   │   └── img/
│   │       └── logo.svg
│   ├── views/
│   │   ├── layouts/
│   │   │   └── default.html
│   │   ├── pages/
│   │   │   ├── home.html
│   │   │   ├── login.html
│   │   │   ├── register.html
│   │   │   ├── workspace.html
│   │   │   ├── projects.html
│   │   │   ├── board.html
│   │   │   ├── list.html
│   │   │   ├── issue.html
│   │   │   ├── sprints.html
│   │   │   ├── settings.html
│   │   │   └── notifications.html
│   │   └── components/
│   │       ├── sidebar.html
│   │       ├── issue_card.html
│   │       ├── issue_detail.html
│   │       ├── command_palette.html
│   │       ├── dropdown.html
│   │       ├── modal.html
│   │       └── toast.html
│   └── assets.go
├── cli/
│   ├── root.go
│   ├── serve.go
│   ├── init.go
│   ├── migrate.go
│   └── user.go
├── cmd/
│   └── kanban/
│       └── main.go
├── feature/
│   ├── workspaces/
│   │   ├── api.go
│   │   └── service.go
│   ├── projects/
│   │   ├── api.go
│   │   └── service.go
│   ├── issues/
│   │   ├── api.go
│   │   └── service.go
│   ├── comments/
│   │   ├── api.go
│   │   └── service.go
│   ├── sprints/
│   │   ├── api.go
│   │   └── service.go
│   ├── labels/
│   │   ├── api.go
│   │   └── service.go
│   ├── notifications/
│   │   ├── api.go
│   │   └── service.go
│   ├── search/
│   │   ├── api.go
│   │   └── service.go
│   └── users/
│       ├── api.go
│       └── service.go
├── pkg/
│   ├── ulid/
│   │   └── ulid.go
│   └── password/
│       └── argon2.go
├── store/
│   └── duckdb/
│       ├── store.go
│       ├── schema.sql
│       ├── workspaces_store.go
│       ├── projects_store.go
│       ├── issues_store.go
│       ├── comments_store.go
│       ├── sprints_store.go
│       ├── labels_store.go
│       ├── notifications_store.go
│       └── users_store.go
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Implementation Priorities

### Phase 1: Core Foundation
1. Database schema and migrations
2. User authentication
3. Workspace CRUD
4. Project CRUD

### Phase 2: Issue Management
1. Issue CRUD
2. Status workflow
3. Priority system
4. Labels
5. Comments

### Phase 3: Board Interface
1. Kanban board view
2. Drag & drop functionality
3. Issue detail panel
4. Quick actions

### Phase 4: Enhanced Features
1. Sprints
2. Search & filters
3. Notifications
4. Activity tracking

### Phase 5: Polish
1. Command palette
2. Keyboard shortcuts
3. Dark/light theme toggle
4. Performance optimization

## Technical Notes

### DuckDB Schema Considerations
- Use TEXT for all IDs (ULIDs)
- Use INTEGER for positions (for drag & drop ordering)
- Use VARCHAR for enum-like fields (status, priority, type)
- Create indexes on frequently queried columns

### Frontend Approach
- Server-side rendered HTML with Go templates
- Minimal JavaScript for interactivity
- Use native drag & drop API
- Progressive enhancement

### Performance Targets
- Board render: < 100ms
- Issue create/update: < 50ms
- Search results: < 200ms
