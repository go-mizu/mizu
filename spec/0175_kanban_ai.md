# Kanban AI - Modern UI Design Specification

## Overview

This specification defines a modern, Linear.app-inspired light theme UI for the Kanban application. The design prioritizes Developer Experience (DX) with fast interactions, keyboard shortcuts, and intuitive workflows.

## Design Philosophy

### Core Principles
1. **Speed First** - Instant feedback, optimistic updates, minimal loading states
2. **Keyboard-Centric** - All actions accessible via keyboard shortcuts
3. **Clean & Minimal** - Remove clutter, focus on content
4. **Consistent** - Unified visual language across all pages
5. **Accessible** - WCAG 2.1 AA compliant, proper contrast ratios

### Design Inspirations
- **Linear.app** - Clean issue tracking, keyboard navigation, status colors
- **shadcn/ui** - CSS variables, component patterns, light theme palette
- **ChatGPT** - Collapsible sidebar, clean typography, minimal chrome

---

## User Requirements

### Primary User Goals
1. Quickly create and manage issues
2. Visualize work progress on kanban board
3. Navigate between projects efficiently
4. Collaborate with team members
5. Track sprint/cycle progress

### Key DX Features
1. **Command Palette (Cmd+K)** - Global search and quick actions
2. **Keyboard Navigation** - j/k to move, Enter to open, Escape to close
3. **Quick Add** - Create issues without modals
4. **Inline Editing** - Click to edit titles directly
5. **Drag & Drop** - Intuitive card movement on board
6. **Real-time Updates** - Instant UI feedback

---

## Design System

### Color Palette (Light Theme)

```css
:root {
  /* Base Colors */
  --background: 0 0% 100%;           /* #ffffff - Page background */
  --foreground: 240 10% 3.9%;        /* #09090b - Primary text */

  /* Component Colors */
  --card: 0 0% 100%;                 /* #ffffff - Card background */
  --card-foreground: 240 10% 3.9%;  /* #09090b - Card text */
  --popover: 0 0% 100%;             /* #ffffff - Dropdown/modal bg */
  --popover-foreground: 240 10% 3.9%;

  /* Interactive Colors */
  --primary: 240 5.9% 10%;          /* #18181b - Primary buttons */
  --primary-foreground: 0 0% 98%;   /* #fafafa - Button text */
  --secondary: 240 4.8% 95.9%;      /* #f4f4f5 - Secondary bg */
  --secondary-foreground: 240 5.9% 10%;

  /* Utility Colors */
  --muted: 240 4.8% 95.9%;          /* #f4f4f5 - Muted backgrounds */
  --muted-foreground: 240 3.8% 46.1%; /* #71717a - Muted text */
  --accent: 240 4.8% 95.9%;         /* #f4f4f5 - Accent bg */
  --accent-foreground: 240 5.9% 10%;

  /* Borders & Focus */
  --border: 240 5.9% 90%;           /* #e4e4e7 - Borders */
  --input: 240 5.9% 90%;            /* #e4e4e7 - Input borders */
  --ring: 240 5.9% 10%;             /* #18181b - Focus ring */

  /* Semantic Colors */
  --destructive: 0 84.2% 60.2%;     /* #ef4444 - Danger/delete */
  --destructive-foreground: 0 0% 98%;

  /* Status Colors (Linear-inspired) */
  --status-backlog: 220 9% 46%;     /* #6b7280 - Gray */
  --status-todo: 217 91% 60%;       /* #3b82f6 - Blue */
  --status-in-progress: 45 93% 47%; /* #eab308 - Yellow */
  --status-done: 142 71% 45%;       /* #22c55e - Green */
  --status-canceled: 0 0% 60%;      /* #999999 - Gray */

  /* Priority Colors */
  --priority-urgent: 0 84% 60%;     /* Red */
  --priority-high: 25 95% 53%;      /* Orange */
  --priority-medium: 45 93% 47%;    /* Yellow */
  --priority-low: 217 91% 60%;      /* Blue */
  --priority-none: 220 9% 46%;      /* Gray */

  /* Spacing */
  --radius: 0.5rem;
  --sidebar-width: 240px;
  --sidebar-collapsed: 64px;
  --topbar-height: 56px;
}
```

### Typography

```css
/* Font Stack */
--font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
--font-mono: 'JetBrains Mono', 'Fira Code', monospace;

/* Font Sizes */
--text-xs: 0.75rem;    /* 12px */
--text-sm: 0.875rem;   /* 14px - Base */
--text-base: 1rem;     /* 16px */
--text-lg: 1.125rem;   /* 18px */
--text-xl: 1.25rem;    /* 20px */
--text-2xl: 1.5rem;    /* 24px */
--text-3xl: 1.875rem;  /* 30px */

/* Font Weights */
--font-normal: 400;
--font-medium: 500;
--font-semibold: 600;
--font-bold: 700;

/* Line Heights */
--leading-tight: 1.25;
--leading-normal: 1.5;
--leading-relaxed: 1.625;
```

### Spacing Scale

```css
--space-0: 0;
--space-1: 0.25rem;   /* 4px */
--space-2: 0.5rem;    /* 8px */
--space-3: 0.75rem;   /* 12px */
--space-4: 1rem;      /* 16px */
--space-5: 1.25rem;   /* 20px */
--space-6: 1.5rem;    /* 24px */
--space-8: 2rem;      /* 32px */
--space-10: 2.5rem;   /* 40px */
--space-12: 3rem;     /* 48px */
```

---

## Component Library

### Buttons

```html
<!-- Primary Button -->
<button class="btn btn-primary">Create Issue</button>

<!-- Secondary Button -->
<button class="btn btn-secondary">Cancel</button>

<!-- Ghost Button -->
<button class="btn btn-ghost">View All</button>

<!-- Icon Button -->
<button class="btn btn-icon btn-ghost">
  <svg>...</svg>
</button>

<!-- Destructive Button -->
<button class="btn btn-destructive">Delete</button>
```

**Button Styles:**
- Height: 36px (default), 32px (sm), 40px (lg)
- Padding: 0 16px
- Border radius: 6px
- Transitions: 150ms ease

### Inputs

```html
<!-- Text Input -->
<input type="text" class="input" placeholder="Search...">

<!-- With Label -->
<div class="form-group">
  <label for="title" class="label">Title</label>
  <input type="text" id="title" class="input">
</div>

<!-- Textarea -->
<textarea class="textarea" rows="4"></textarea>

<!-- Select -->
<select class="select">
  <option>Option 1</option>
</select>
```

**Input Styles:**
- Height: 36px
- Border: 1px solid var(--border)
- Border radius: 6px
- Focus: ring-2 ring-offset-2

### Cards

```html
<div class="card">
  <div class="card-header">
    <h3 class="card-title">Title</h3>
  </div>
  <div class="card-content">
    Content here
  </div>
</div>
```

**Card Styles:**
- Background: white
- Border: 1px solid var(--border)
- Border radius: 8px
- Shadow: 0 1px 3px rgba(0,0,0,0.1)

### Modals

```html
<div id="modal-id" class="modal hidden">
  <div class="modal-backdrop"></div>
  <div class="modal-content">
    <div class="modal-header">
      <h2 class="modal-title">Modal Title</h2>
      <button class="modal-close btn btn-icon btn-ghost">&times;</button>
    </div>
    <div class="modal-body">
      Content
    </div>
    <div class="modal-footer">
      <button class="btn btn-secondary">Cancel</button>
      <button class="btn btn-primary">Save</button>
    </div>
  </div>
</div>
```

### Dropdowns

```html
<div class="dropdown">
  <button class="dropdown-trigger btn btn-ghost btn-icon">
    <svg>...</svg>
  </button>
  <div class="dropdown-menu hidden">
    <button class="dropdown-item">Edit</button>
    <button class="dropdown-item">Delete</button>
  </div>
</div>
```

### Status Badges

```html
<span class="status-badge status-todo">Todo</span>
<span class="status-badge status-in-progress">In Progress</span>
<span class="status-badge status-done">Done</span>
```

### Tables

```html
<table class="table">
  <thead>
    <tr>
      <th>Key</th>
      <th>Title</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td class="issue-key">PROJ-1</td>
      <td>Issue title</td>
      <td><span class="status-badge">Todo</span></td>
    </tr>
  </tbody>
</table>
```

---

## Page Layouts

### Main Application Layout

```
+------------------+----------------------------------------+
|     Sidebar      |              Topbar                    |
|                  +----------------------------------------+
|   - Logo         |                                        |
|   - Navigation   |           Main Content                 |
|   - Projects     |                                        |
|   - Teams        |                                        |
|                  |                                        |
|   [Collapse]     |                                        |
+------------------+----------------------------------------+
```

**Sidebar (Expanded: 240px, Collapsed: 64px)**
- Logo/brand at top
- Navigation links with icons
- Project list
- Collapse toggle at bottom
- User avatar with dropdown

**Topbar (Height: 56px)**
- Breadcrumb navigation
- Search input / Cmd+K trigger
- Action buttons
- User menu

### Auth Layout

```
+--------------------------------------------------+
|                                                  |
|                                                  |
|              +------------------+                |
|              |      Logo        |                |
|              +------------------+                |
|              |                  |                |
|              |   Form Fields    |                |
|              |                  |                |
|              +------------------+                |
|                                                  |
|                                                  |
+--------------------------------------------------+
```

---

## Page Specifications

### 1. Login Page (`/login`)

**Elements:**
- `.auth-logo` - Brand logo
- `#email` - Email input field
- `#password` - Password input field
- `button[type="submit"]` - Login button
- `.alert-error` - Error message container
- `a[href="/register"]` - Register link

**Interactions:**
- Form validation on blur
- Error display on invalid credentials
- Redirect to `/app` on success

### 2. Register Page (`/register`)

**Elements:**
- Form fields: display_name, email, username, password
- Password strength indicator
- Submit button
- Login link

### 3. Home/Dashboard (`/app`, `/w/{workspace}`)

**Elements:**
- `.sidebar` - Navigation sidebar
- `.topbar` - Top navigation bar
- `h1:has-text("Welcome back")` - Welcome message
- `.card` with `.text-2xl` - Stats cards
  - Open Issues count
  - In Progress count
  - Completed count
- `h2:has-text("Projects")` - Projects section
- `.card` links - Project cards
- `h2:has-text("Active Cycle")` - Active cycle section

**Stats Cards Layout:**
```
+------------+  +------------+  +------------+
| Open       |  | In Progress|  | Completed  |
| Issues     |  |            |  |            |
|    24      |  |     8      |  |     156    |
+------------+  +------------+  +------------+
```

### 4. Kanban Board (`/w/{workspace}/board/{projectId}`)

**Elements:**
- `#board` - Main board container
- `.board-column` - Column containers
- `.column-title` - Column name
- `.column-body` - Cards container
- `.issue-card` - Issue cards
- `.issue-key` - Issue identifier (e.g., "PROJ-1")
- `.quick-add-form input` - Quick add input
- `[data-modal="create-issue-modal"]` - Create issue trigger
- `#create-issue-modal` - Create issue modal
- `[data-modal="add-column-modal"]` - Add column trigger
- `#add-column-modal` - Add column modal

**Board Layout:**
```
+------------------+------------------+------------------+
| Backlog (5)  [v] | In Progress (3)  | Done (12)    [v] |
+------------------+------------------+------------------+
| +-------------+  | +-------------+  | +-------------+  |
| | PROJ-1      |  | | PROJ-4      |  | | PROJ-7      |  |
| | Task title  |  | | Task title  |  | | Task title  |  |
| | [avatar]    |  | | [avatar]    |  | | [avatar]    |  |
| +-------------+  | +-------------+  | +-------------+  |
| +-------------+  | +-------------+  |                  |
| | PROJ-2      |  | | PROJ-5      |  |                  |
| | Task title  |  | | Task title  |  |                  |
| +-------------+  | +-------------+  |                  |
+------------------+------------------+------------------+
| [+ Add issue]    | [+ Add issue]    | [+ Add issue]    |
+------------------+------------------+------------------+
```

**Issue Card:**
```html
<div class="issue-card" draggable="true" data-issue-id="xxx" data-position="1">
  <div class="issue-card-header">
    <span class="issue-key">PROJ-1</span>
    <button class="dropdown-trigger">...</button>
  </div>
  <div class="issue-card-title">Issue title here</div>
  <div class="issue-card-meta">
    <span class="priority-indicator priority-high"></span>
    <div class="avatar avatar-sm">JD</div>
  </div>
</div>
```

**Drag & Drop:**
- Native HTML5 drag events
- Visual drop indicator
- Column highlight on dragover
- Optimistic position update

### 5. Issues List (`/w/{workspace}/issues`)

**Elements:**
- `h1:has-text("Issues")` - Page title
- `p.text-muted:has-text("issues")` - Issue count
- `.table` - Issues table
- `.table tbody tr` - Issue rows
- `.issue-key` - Issue identifier
- `.status-badge` - Status indicator
- `[data-modal="create-issue-modal"]` - Create issue trigger
- `button:has-text("Filter")` - Filter button
- `.empty-state` - Empty state when no issues

**Table Columns:**
| Key | Title | Status | Assignee | Updated |
|-----|-------|--------|----------|---------|

### 6. Issue Detail (`/w/{workspace}/issue/{key}`)

**Elements:**
- `a.btn-ghost.btn-icon` - Back button
- `.issue-key` - Issue identifier
- `#issue-title` or `h1[contenteditable]` - Editable title
- `#issue-description` or `.prose[contenteditable]` - Description
- `#issue-status` - Status select
- `#issue-priority` - Priority select
- `#issue-cycle` - Cycle select
- `.card:has-text("Assignees")` - Assignees section
- `[data-modal="assign-modal"]` - Add assignee trigger
- `textarea[name="body"]` - Comment input
- `button:has-text("Comment")` - Submit comment
- `.card:has-text("Activity")` - Comments section
- `.dropdown button` - More options menu
- `button:has-text("Delete")` - Delete issue

**Layout:**
```
+------------------------------------------+-------------+
| [<] Back                                 |             |
|                                          |  Status     |
| PROJ-1                                   |  [Select]   |
| ========================================= |             |
| Issue Title (editable)                   |  Priority   |
|                                          |  [Select]   |
| Description text here...                 |             |
|                                          |  Cycle      |
|                                          |  [Select]   |
|                                          |             |
+------------------------------------------+  Assignees  |
| Activity                                 |  [+] Add    |
| +--------------------------------------+ |             |
| | Comment input                        | +-------------+
| +--------------------------------------+ |
| | [Comment] button                     | |
| +--------------------------------------+ |
|                                          |
| John - 2h ago                            |
| Great progress on this!                  |
|                                          |
+------------------------------------------+
```

### 7. Cycles Page (`/w/{workspace}/cycles`)

**Elements:**
- Page title
- Create cycle button
- Cycle cards/list
- Status badges (planning/active/completed)
- Progress indicators
- Date ranges

**Cycle Card:**
```html
<div class="card cycle-card">
  <div class="card-header">
    <h3>Sprint 5</h3>
    <span class="status-badge status-active">Active</span>
  </div>
  <div class="card-content">
    <div class="progress-bar">
      <div class="progress-fill" style="width: 65%"></div>
    </div>
    <p class="text-muted">8 of 12 issues completed</p>
    <p class="text-sm">Jan 15 - Jan 29</p>
  </div>
</div>
```

### 8. Team Page (`/w/{workspace}/team/{teamId}`)

**Elements:**
- Team name header
- Members table
- Role badges (lead/member)
- Add member button
- Member avatars
- Role management dropdowns

---

## Interactions & Animations

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + K` | Open command palette |
| `Escape` | Close modal/dropdown |
| `j` / `k` | Move down/up in lists |
| `Enter` | Open selected item |
| `c` | Create new issue |
| `?` | Show keyboard shortcuts |

### Command Palette

```html
<div id="command-palette" class="modal hidden">
  <div class="modal-backdrop"></div>
  <div class="command-palette-content">
    <input type="text" class="command-input" placeholder="Type a command or search...">
    <div class="command-results">
      <div class="command-group">
        <div class="command-group-title">Issues</div>
        <button class="command-item">
          <span class="command-icon">+</span>
          <span class="command-label">Create new issue</span>
          <span class="command-shortcut">C</span>
        </button>
      </div>
      <div class="command-group">
        <div class="command-group-title">Navigation</div>
        <button class="command-item">
          <span class="command-icon">H</span>
          <span class="command-label">Go to Home</span>
        </button>
      </div>
    </div>
  </div>
</div>
```

### Transitions

```css
/* Default transition */
.transition {
  transition: all 150ms ease;
}

/* Hover states */
.btn:hover {
  opacity: 0.9;
}

.card:hover {
  border-color: var(--ring);
}

/* Modal animations */
.modal-content {
  animation: modal-in 150ms ease;
}

@keyframes modal-in {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

/* Sidebar collapse */
.sidebar {
  transition: width 200ms ease;
}
```

### Drag & Drop Visual Feedback

```css
.issue-card[dragging] {
  opacity: 0.5;
  transform: rotate(2deg);
}

.column-body.drag-over {
  background: var(--accent);
  border: 2px dashed var(--primary);
}

.drop-indicator {
  height: 2px;
  background: var(--primary);
  margin: 4px 0;
}
```

---

## API Integration

### Endpoints Used

```javascript
// Auth
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me

// Projects
GET  /api/v1/teams/{teamId}/projects
POST /api/v1/teams/{teamId}/projects
GET  /api/v1/projects/{id}

// Columns
GET  /api/v1/projects/{projectId}/columns
POST /api/v1/projects/{projectId}/columns
POST /api/v1/columns/{id}/position

// Issues
GET  /api/v1/projects/{projectId}/issues
POST /api/v1/projects/{projectId}/issues
PATCH /api/v1/issues/{key}
POST /api/v1/issues/{key}/move

// Cycles
GET  /api/v1/teams/{teamId}/cycles
POST /api/v1/teams/{teamId}/cycles

// Comments
GET  /api/v1/issues/{issueId}/comments
POST /api/v1/issues/{issueId}/comments
```

### API Helper Functions

```javascript
const api = {
  async request(method, path, data) {
    const res = await fetch(`/api/v1${path}`, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: data ? JSON.stringify(data) : undefined,
      credentials: 'include'
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },
  get: (path) => api.request('GET', path),
  post: (path, data) => api.request('POST', path, data),
  patch: (path, data) => api.request('PATCH', path, data),
  delete: (path) => api.request('DELETE', path)
};
```

---

## E2E Test Selectors Reference

All selectors required by existing e2e tests:

```
Navigation:
- .sidebar
- .topbar
- .avatar

Board:
- #board
- .board-column
- .column-title
- .column-body
- .issue-card
- .issue-key
- .quick-add-form input
- #create-issue-modal
- #add-column-modal
- [data-modal="create-issue-modal"]
- [data-modal="add-column-modal"]

Issues:
- h1:has-text("Issues")
- p.text-muted:has-text("issues")
- .table
- .table tbody tr
- .status-badge
- .empty-state

Issue Detail:
- .issue-key
- #issue-title, h1[contenteditable]
- #issue-description, .prose[contenteditable]
- #issue-status
- #issue-priority
- #issue-cycle
- .card:has-text("Assignees")
- [data-modal="assign-modal"]
- textarea[name="body"]
- button:has-text("Comment")
- .card:has-text("Activity")
- a.btn-ghost.btn-icon (back button)
- .dropdown button
- button:has-text("Delete")

Home:
- h1:has-text("Welcome back")
- .card with .text-2xl (stats)
- .card:has-text("Open Issues")
- .card:has-text("In Progress")
- .card:has-text("Completed")
- h2:has-text("Projects")
- h2:has-text("Active Cycle")
- button:has-text("New Issue")

Auth:
- #email
- #password
- button[type="submit"]
- .alert-error
- .auth-logo
- a[href="/register"]

Forms:
- #issue-title (in modal)
- #issue-description (in modal)
- #issue-column (in modal)
- #column-name
- button:has-text("Create Issue")
- button:has-text("Add Column")
```

---

## File Structure

```
blueprints/kanban/assets/
├── static/
│   ├── css/
│   │   └── default.css      # Main stylesheet
│   └── js/
│       └── app.js           # JavaScript interactions
└── views/
    └── default/
        ├── layouts/
        │   ├── default.html  # Main app layout
        │   └── auth.html     # Auth pages layout
        └── pages/
            ├── home.html     # Dashboard
            ├── board.html    # Kanban board
            ├── issues.html   # Issues list
            ├── issue.html    # Issue detail
            ├── cycles.html   # Cycles management
            ├── team.html     # Team management
            ├── login.html    # Login form
            └── register.html # Registration form
```

---

## Implementation Checklist

- [ ] CSS Design System with all variables
- [ ] Component styles (buttons, inputs, cards, modals, tables)
- [ ] Layout styles (sidebar, topbar, main content)
- [ ] Status and priority badge styles
- [ ] JavaScript utilities (API, modals, dropdowns)
- [ ] Drag-and-drop implementation
- [ ] Command palette (Cmd+K)
- [ ] Keyboard navigation
- [ ] Auth layout (centered card design)
- [ ] Main layout (collapsible sidebar)
- [ ] Login page
- [ ] Register page
- [ ] Home/Dashboard page
- [ ] Board page with all interactions
- [ ] Issues list page
- [ ] Issue detail page
- [ ] Cycles page
- [ ] Team page
- [ ] E2E test compatibility verification
