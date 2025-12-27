# RFC 0178: Inbox Feature

## Summary

Replace the "Home" dashboard with an "Inbox" feature that provides a unified view of issues requiring user attention, with a permanent "Create Issue" box for quick capture of new work items.

## Motivation

The current Home page shows aggregate statistics and project lists, but doesn't provide a task-focused view of work items requiring attention. Users need:

1. **Quick issue capture** - A permanently visible input for creating issues without modal friction
2. **Personal task focus** - Issues assigned to them, created by them, or requiring their review
3. **Temporal grouping** - Issues organized by recency (Today, Yesterday, This Week, Older)
4. **Quick triage** - Ability to quickly assign status, project, and priority during creation

## Design Principles

### User Needs

1. **"What should I work on?"** - Show issues assigned to me sorted by priority/recency
2. **"Quick thought capture"** - Create issues with minimal friction (just title, smart defaults)
3. **"Where did I leave off?"** - Show recently created/updated issues
4. **"What needs my attention?"** - Filter by status, assignee, priority

### Developer Experience

1. Single endpoint for inbox data (avoids multiple API calls)
2. Progressive enhancement - works without JavaScript
3. Consistent with existing template patterns
4. Minimal new database queries

## Specification

### 1. Navigation Change

Replace "Home" with "Inbox" in the sidebar:

```diff
- <a href="/w/{{.Workspace.Slug}}" class="nav-item">
-   <svg><!-- home icon --></svg>
-   <span>Home</span>
- </a>
+ <a href="/w/{{.Workspace.Slug}}/inbox" class="nav-item">
+   <svg><!-- inbox icon --></svg>
+   <span>Inbox</span>
+ </a>
```

### 2. Route Structure

```
GET /w/{workspace}/inbox          -> Inbox page (default: assigned to me)
GET /w/{workspace}/inbox?tab=assigned  -> Issues assigned to me
GET /w/{workspace}/inbox?tab=created   -> Issues created by me
GET /w/{workspace}/inbox?tab=all       -> All workspace issues
```

### 3. Data Model

No schema changes required. Use existing `issues`, `assignees`, and `projects` tables.

```go
// InboxData holds data for the inbox page
type InboxData struct {
    Title           string
    User            *users.User
    Workspace       *workspaces.Workspace
    Workspaces      []*workspaces.Workspace
    Teams           []*teams.Team
    Projects        []*projects.Project

    // Inbox-specific
    Issues          []*InboxIssue
    ActiveTab       string // "assigned", "created", "all"
    IssueGroups     []*IssueGroup

    // For create issue form
    Columns         []*columns.Column
    DefaultProjectID string
    TeamMembers     []*users.User
    Cycles          []*cycles.Cycle

    // Standard fields
    ActiveTeamID    string
    ActiveProjectID string
    ActiveNav       string
    Breadcrumbs     []Breadcrumb
}

// InboxIssue wraps an issue with inbox-specific metadata
type InboxIssue struct {
    *issues.Issue
    Project     *projects.Project
    Column      *columns.Column
    Assignees   []*users.User
    TimeGroup   string // "today", "yesterday", "this_week", "older"
    IsAssigned  bool   // Current user is assignee
    IsCreator   bool   // Current user created it
}

// IssueGroup groups issues by time
type IssueGroup struct {
    Label  string        // "Today", "Yesterday", "This Week", "Older"
    Key    string        // "today", "yesterday", "this_week", "older"
    Issues []*InboxIssue
}
```

### 4. UI Components

#### 4.1 Permanent Create Issue Box

A permanently visible issue creation form at the top of the Inbox page:

```html
<div class="inbox-create-box">
    <div class="create-box-header">
        <span class="create-box-project">
            <svg><!-- folder icon --></svg>
            {{.DefaultProject.Name}}
            <svg><!-- chevron-down icon --></svg>
        </span>
        <span class="breadcrumb-separator">></span>
        <span>New issue</span>
    </div>

    <form class="create-issue-form" id="inbox-create-form">
        <input
            type="text"
            name="title"
            class="create-box-title"
            placeholder="Issue title"
            autocomplete="off"
            required
        >
        <textarea
            name="description"
            class="create-box-description"
            placeholder="Add description..."
            rows="2"
        ></textarea>

        <div class="create-box-properties">
            <button type="button" class="property-chip" data-property="status">
                <svg><!-- circle-dot icon --></svg>
                Backlog
            </button>
            <button type="button" class="property-chip" data-property="priority">
                <svg><!-- signal icon --></svg>
                Priority
            </button>
            <button type="button" class="property-chip" data-property="assignee">
                <svg><!-- user icon --></svg>
                Assignee
            </button>
            <button type="button" class="property-chip" data-property="project">
                <svg><!-- folder icon --></svg>
                Project
            </button>
            <button type="button" class="property-chip" data-property="cycle">
                <svg><!-- refresh-cw icon --></svg>
                Cycle
            </button>
        </div>

        <div class="create-box-footer">
            <button type="button" class="btn btn-ghost btn-sm">
                <svg><!-- paperclip icon --></svg>
            </button>
            <div class="create-box-actions">
                <label class="create-more-toggle">
                    <input type="checkbox" name="create_more" id="create-more">
                    <span>Create more</span>
                </label>
                <button type="submit" class="btn btn-primary">
                    Create issue
                </button>
            </div>
        </div>
    </form>
</div>
```

#### 4.2 Tab Navigation

```html
<div class="inbox-tabs">
    <a href="?tab=assigned" class="inbox-tab {{if eq .ActiveTab "assigned"}}active{{end}}">
        Assigned to me
        <span class="tab-count">{{.AssignedCount}}</span>
    </a>
    <a href="?tab=created" class="inbox-tab {{if eq .ActiveTab "created"}}active{{end}}">
        Created by me
        <span class="tab-count">{{.CreatedCount}}</span>
    </a>
    <a href="?tab=all" class="inbox-tab {{if eq .ActiveTab "all"}}active{{end}}">
        All issues
    </a>
</div>
```

#### 4.3 Issue List with Time Groups

```html
{{range .IssueGroups}}
{{if .Issues}}
<div class="issue-group">
    <div class="group-header">{{.Label}}</div>
    {{range .Issues}}
    <a href="/w/{{$.Workspace.Slug}}/issue/{{.Key}}" class="inbox-issue">
        <div class="issue-row">
            <span class="issue-key">{{.Key}}</span>
            <span class="issue-title">{{.Title}}</span>
            <div class="issue-meta">
                {{if .Column}}
                <span class="status-badge {{lower .Column.Name}}">{{.Column.Name}}</span>
                {{end}}
                <span class="issue-project">{{.Project.Name}}</span>
                <span class="issue-date">{{.UpdatedAt | timeAgo}}</span>
            </div>
        </div>
    </a>
    {{end}}
</div>
{{end}}
{{end}}
```

### 5. CSS Additions

```css
/* Inbox Create Box */
.inbox-create-box {
    background: hsl(var(--card));
    border: 1px solid hsl(var(--border));
    margin-bottom: 1.5rem;
}

.create-box-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    color: hsl(var(--muted-foreground));
    border-bottom: 1px solid hsl(var(--border));
}

.create-box-project {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    font-weight: 500;
    color: hsl(var(--foreground));
}

.create-box-title {
    width: 100%;
    padding: 1rem;
    font-size: 1.125rem;
    font-weight: 500;
    border: none;
    background: transparent;
    outline: none;
}

.create-box-title::placeholder {
    color: hsl(var(--muted-foreground));
}

.create-box-description {
    width: 100%;
    padding: 0 1rem 1rem;
    font-size: 0.875rem;
    border: none;
    background: transparent;
    outline: none;
    resize: none;
}

.create-box-properties {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    padding: 0 1rem 1rem;
}

.property-chip {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.75rem;
    font-size: 0.8125rem;
    color: hsl(var(--muted-foreground));
    background: transparent;
    border: 1px dashed hsl(var(--border));
    cursor: pointer;
    transition: all 150ms ease;
}

.property-chip:hover {
    border-style: solid;
    background: hsl(var(--muted));
}

.property-chip.selected {
    border-style: solid;
    background: hsl(var(--muted));
    color: hsl(var(--foreground));
}

.property-chip svg {
    width: 14px;
    height: 14px;
}

.create-box-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1rem;
    border-top: 1px solid hsl(var(--border));
}

.create-box-actions {
    display: flex;
    align-items: center;
    gap: 1rem;
}

.create-more-toggle {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    color: hsl(var(--muted-foreground));
    cursor: pointer;
}

.create-more-toggle input {
    width: 1rem;
    height: 1rem;
    accent-color: hsl(var(--primary));
}

/* Inbox Tabs */
.inbox-tabs {
    display: flex;
    gap: 0;
    border-bottom: 1px solid hsl(var(--border));
    margin-bottom: 1rem;
}

.inbox-tab {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: hsl(var(--muted-foreground));
    border-bottom: 2px solid transparent;
    transition: all 150ms ease;
}

.inbox-tab:hover {
    color: hsl(var(--foreground));
}

.inbox-tab.active {
    color: hsl(var(--foreground));
    border-bottom-color: hsl(var(--primary));
}

.tab-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 1.25rem;
    height: 1.25rem;
    padding: 0 0.375rem;
    font-size: 0.75rem;
    background: hsl(var(--muted));
    color: hsl(var(--muted-foreground));
}

/* Issue Groups */
.issue-group {
    margin-bottom: 1.5rem;
}

.group-header {
    padding: 0.5rem 0;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: hsl(var(--muted-foreground));
}

/* Inbox Issue Row */
.inbox-issue {
    display: block;
    padding: 0.75rem 0;
    border-bottom: 1px solid hsl(var(--border));
    transition: background 150ms ease;
}

.inbox-issue:hover {
    background: hsl(var(--muted) / 0.5);
}

.inbox-issue:last-child {
    border-bottom: none;
}

.issue-row {
    display: grid;
    grid-template-columns: 80px 1fr auto;
    gap: 1rem;
    align-items: center;
}

.issue-key {
    font-size: 0.75rem;
    font-weight: 500;
    color: hsl(var(--muted-foreground));
}

.issue-title {
    font-size: 0.875rem;
    font-weight: 500;
    color: hsl(var(--foreground));
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.issue-meta {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 0.75rem;
    color: hsl(var(--muted-foreground));
}

.issue-project {
    font-weight: 500;
}

.issue-date {
    opacity: 0.7;
}
```

### 6. Icon System Update

Replace inline SVGs with a consistent Lucide icon system. Create a partial template:

```html
{{define "icon"}}
{{if eq . "inbox"}}
<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/>
    <path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>
</svg>
{{else if eq . "plus"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M5 12h14"/>
    <path d="M12 5v14"/>
</svg>
{{else if eq . "folder"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"/>
</svg>
{{else if eq . "chevron-down"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="m6 9 6 6 6-6"/>
</svg>
{{else if eq . "circle-dot"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="10"/>
    <circle cx="12" cy="12" r="1"/>
</svg>
{{else if eq . "signal"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M2 20h.01"/>
    <path d="M7 20v-4"/>
    <path d="M12 20v-8"/>
    <path d="M17 20V8"/>
    <path d="M22 4v16"/>
</svg>
{{else if eq . "user"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/>
    <circle cx="12" cy="7" r="4"/>
</svg>
{{else if eq . "refresh-cw"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/>
    <path d="M21 3v5h-5"/>
    <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/>
    <path d="M8 16H3v5"/>
</svg>
{{else if eq . "paperclip"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"/>
</svg>
{{else if eq . "kanban"}}
<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <rect width="6" height="14" x="4" y="5" rx="2"/>
    <rect width="6" height="10" x="14" y="7" rx="2"/>
</svg>
{{else if eq . "users"}}
<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/>
    <circle cx="9" cy="7" r="4"/>
    <path d="M22 21v-2a4 4 0 0 0-3-3.87"/>
    <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
</svg>
{{else if eq . "refresh-ccw"}}
<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
    <path d="M3 3v5h5"/>
    <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/>
    <path d="M16 16h5v5"/>
</svg>
{{else if eq . "settings"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
    <circle cx="12" cy="12" r="3"/>
</svg>
{{else if eq . "search"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="11" cy="11" r="8"/>
    <path d="m21 21-4.3-4.3"/>
</svg>
{{else if eq . "log-out"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
    <polyline points="16 17 21 12 16 7"/>
    <line x1="21" x2="9" y1="12" y2="12"/>
</svg>
{{else if eq . "panel-left"}}
<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <rect width="18" height="18" x="3" y="3" rx="2"/>
    <path d="M9 3v18"/>
</svg>
{{else if eq . "circle-alert"}}
<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="10"/>
    <line x1="12" x2="12" y1="8" y2="12"/>
    <line x1="12" x2="12.01" y1="16" y2="16"/>
</svg>
{{else if eq . "filter"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/>
</svg>
{{else if eq . "trash-2"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M3 6h18"/>
    <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
    <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
    <line x1="10" x2="10" y1="11" y2="17"/>
    <line x1="14" x2="14" y1="11" y2="17"/>
</svg>
{{else if eq . "more-vertical"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="1"/>
    <circle cx="12" cy="5" r="1"/>
    <circle cx="12" cy="19" r="1"/>
</svg>
{{else if eq . "external-link"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
    <polyline points="15 3 21 3 21 9"/>
    <line x1="10" x2="21" y1="14" y2="3"/>
</svg>
{{else if eq . "arrow-left"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="m12 19-7-7 7-7"/>
    <path d="M19 12H5"/>
</svg>
{{else if eq . "layout-grid"}}
<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <rect width="7" height="7" x="3" y="3" rx="1"/>
    <rect width="7" height="7" x="14" y="3" rx="1"/>
    <rect width="7" height="7" x="14" y="14" rx="1"/>
    <rect width="7" height="7" x="3" y="14" rx="1"/>
</svg>
{{end}}
{{end}}
```

### 7. JavaScript Additions

```javascript
// inbox.js - Add to app.js

const inbox = {
    init() {
        this.setupCreateForm();
        this.setupPropertyChips();
    },

    setupCreateForm() {
        const form = document.getElementById('inbox-create-form');
        if (!form) return;

        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const data = {
                title: form.title.value,
                description: form.description?.value || '',
                project_id: form.project_id?.value || this.selectedProjectId,
                column_id: this.selectedColumnId,
                cycle_id: this.selectedCycleId,
            };

            // Get assignee if selected
            if (this.selectedAssigneeId) {
                data.assignee_ids = [this.selectedAssigneeId];
            }

            try {
                const issue = await app.api.post(`/projects/${data.project_id}/issues`, data);

                // Check if "Create more" is enabled
                const createMore = document.getElementById('create-more')?.checked;
                if (createMore) {
                    // Clear form and focus title
                    form.reset();
                    form.title.focus();
                    this.showSuccessToast(`Created ${issue.key}`);
                } else {
                    // Navigate to issue
                    location.href = `/w/${workspaceSlug}/issue/${issue.key}`;
                }
            } catch (error) {
                this.showErrorToast(error.message || 'Failed to create issue');
            }
        });
    },

    setupPropertyChips() {
        document.querySelectorAll('.property-chip').forEach(chip => {
            chip.addEventListener('click', () => {
                const property = chip.dataset.property;
                this.openPropertySelector(property, chip);
            });
        });
    },

    openPropertySelector(property, targetChip) {
        // Create dropdown for property selection
        const dropdown = document.createElement('div');
        dropdown.className = 'property-dropdown';
        // ... populate based on property type
    },

    showSuccessToast(message) {
        // Simple toast notification
        console.log('Success:', message);
    },

    showErrorToast(message) {
        alert(message);
    },

    selectedProjectId: null,
    selectedColumnId: null,
    selectedCycleId: null,
    selectedAssigneeId: null,
};

// Initialize on page load
if (document.querySelector('.inbox-create-box')) {
    inbox.init();
}
```

### 8. Handler Changes

Modify `page.go`:

```go
// Inbox renders the inbox page (replaces Home)
func (h *Page) Inbox(c *mizu.Ctx) error {
    userID := h.getUserID(c)
    if userID == "" {
        http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
        return nil
    }

    ctx := c.Request().Context()
    tab := c.Query("tab")
    if tab == "" {
        tab = "assigned"
    }

    user, _ := h.users.GetByID(ctx, userID)
    workspaceSlug := c.Param("workspace")
    workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
    workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

    // Get teams and projects
    var teamList []*teams.Team
    var projectList []*projects.Project
    var defaultProjectID string
    var defaultTeamID string

    if workspace != nil {
        teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
        if len(teamList) > 0 {
            defaultTeamID = teamList[0].ID
            projectList, _ = h.projects.ListByTeam(ctx, teamList[0].ID)
            if len(projectList) > 0 {
                defaultProjectID = projectList[0].ID
            }
        }
    }

    // Fetch issues based on tab
    var allIssues []*issues.Issue
    // ... implementation depends on tab filter

    // Group issues by time
    issueGroups := h.groupIssuesByTime(allIssues, userID)

    // Get columns for default project (for create form)
    var columnList []*columns.Column
    if defaultProjectID != "" {
        columnList, _ = h.columns.ListByProject(ctx, defaultProjectID)
    }

    return render(h, c, "inbox", InboxData{
        Title:            "Inbox",
        User:             user,
        Workspace:        workspace,
        Workspaces:       workspaceList,
        Teams:            teamList,
        Projects:         projectList,
        IssueGroups:      issueGroups,
        ActiveTab:        tab,
        Columns:          columnList,
        DefaultProjectID: defaultProjectID,
        DefaultTeamID:    defaultTeamID,
        ActiveNav:        "inbox",
    })
}

// groupIssuesByTime groups issues into time-based sections
func (h *Page) groupIssuesByTime(issues []*issues.Issue, userID string) []*IssueGroup {
    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    yesterday := today.AddDate(0, 0, -1)
    weekAgo := today.AddDate(0, 0, -7)

    groups := map[string]*IssueGroup{
        "today":     {Label: "Today", Key: "today"},
        "yesterday": {Label: "Yesterday", Key: "yesterday"},
        "this_week": {Label: "This Week", Key: "this_week"},
        "older":     {Label: "Older", Key: "older"},
    }

    for _, issue := range issues {
        var group string
        if issue.UpdatedAt.After(today) || issue.UpdatedAt.Equal(today) {
            group = "today"
        } else if issue.UpdatedAt.After(yesterday) || issue.UpdatedAt.Equal(yesterday) {
            group = "yesterday"
        } else if issue.UpdatedAt.After(weekAgo) {
            group = "this_week"
        } else {
            group = "older"
        }

        inboxIssue := &InboxIssue{
            Issue:     issue,
            TimeGroup: group,
            IsCreator: issue.CreatorID == userID,
        }
        groups[group].Issues = append(groups[group].Issues, inboxIssue)
    }

    // Return in order
    return []*IssueGroup{
        groups["today"],
        groups["yesterday"],
        groups["this_week"],
        groups["older"],
    }
}
```

### 9. Route Registration

Update `server.go`:

```go
// Replace Home route
// router.GET("/w/{workspace}", page.Home)
router.GET("/w/{workspace}", func(c *mizu.Ctx) error {
    // Redirect to inbox
    workspace := c.Param("workspace")
    http.Redirect(c.Writer(), c.Request(), "/w/"+workspace+"/inbox", http.StatusFound)
    return nil
})
router.GET("/w/{workspace}/inbox", page.Inbox)
```

## Implementation Checklist

1. [ ] Create `pages/inbox.html` template
2. [ ] Add InboxData and InboxIssue types to `page.go`
3. [ ] Implement `Inbox` handler
4. [ ] Implement `groupIssuesByTime` helper
5. [ ] Update routes in `server.go`
6. [ ] Update sidebar navigation in `layouts/default.html`
7. [ ] Add inbox CSS to `default.css`
8. [ ] Add inbox JavaScript to `app.js`
9. [ ] Update `ActiveNav` handling throughout
10. [ ] Test all inbox tabs and create functionality

## Migration Notes

- The `/w/{workspace}` route now redirects to `/w/{workspace}/inbox`
- Old "Home" bookmarks will redirect automatically
- No database schema changes required
- Existing issues API endpoints remain unchanged
