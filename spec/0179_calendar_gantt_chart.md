# RFC 0179: Calendar and Gantt Chart Views

## Summary

Add "Calendar" and "Gantt Chart" views to the Issues page, providing timeline-based visualization of work items. These views complement the existing Board (Kanban) and List views.

## Motivation

The current issue tracking focuses on status-based organization (Kanban board) and flat listing (Issues page). Users need:

1. **Temporal planning** - Visualize work across dates and deadlines
2. **Resource scheduling** - See workload distribution over time
3. **Timeline visibility** - Understand duration and overlap of work items
4. **Cycle integration** - View issues within their planning cycles

## Design Principles

### User Needs

1. **"When is this due?"** - Calendar view shows issues by due date
2. **"How long will this take?"** - Gantt chart shows duration from start to end
3. **"What's happening this week?"** - Quick date-range navigation
4. **"How does work overlap?"** - Visual timeline bars showing concurrent work

### Developer Experience

1. Minimal schema changes (add date fields to issues)
2. Reuse existing issue data and API endpoints
3. Progressive enhancement - works without JavaScript for basic display
4. Consistent with existing view patterns

## Specification

### 1. Schema Changes

Add date fields to the `issues` table:

```sql
ALTER TABLE issues ADD COLUMN due_date DATE;
ALTER TABLE issues ADD COLUMN start_date DATE;
ALTER TABLE issues ADD COLUMN end_date DATE;
```

Updated schema:

```sql
CREATE TABLE IF NOT EXISTS issues (
    id         VARCHAR PRIMARY KEY,
    project_id VARCHAR NOT NULL REFERENCES projects(id),
    number     INTEGER NOT NULL,
    key        VARCHAR NOT NULL,
    title      VARCHAR NOT NULL,
    column_id  VARCHAR NOT NULL REFERENCES columns(id),
    position   INTEGER NOT NULL DEFAULT 0,
    creator_id VARCHAR NOT NULL REFERENCES users(id),
    cycle_id   VARCHAR REFERENCES cycles(id),
    due_date   DATE,           -- NEW: When the issue is due
    start_date DATE,           -- NEW: When work begins
    end_date   DATE,           -- NEW: When work ends
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, number),
    UNIQUE (project_id, key)
);
```

### 2. Route Structure

Add view query parameter to Issues page and new dedicated routes:

```
GET /w/{workspace}/issues                -> Issues list (default view)
GET /w/{workspace}/issues?view=list      -> Issues list view
GET /w/{workspace}/issues?view=calendar  -> Calendar view
GET /w/{workspace}/issues?view=gantt     -> Gantt chart view
```

### 3. Data Models

#### 3.1 Issue Model Update

```go
// Issue represents a work item (updated with date fields)
type Issue struct {
    ID        string     `json:"id"`
    ProjectID string     `json:"project_id"`
    Number    int        `json:"number"`
    Key       string     `json:"key"`
    Title     string     `json:"title"`
    ColumnID  string     `json:"column_id"`
    Position  int        `json:"position"`
    CreatorID string     `json:"creator_id"`
    CycleID   *string    `json:"cycle_id,omitempty"`
    DueDate   *time.Time `json:"due_date,omitempty"`   // NEW
    StartDate *time.Time `json:"start_date,omitempty"` // NEW
    EndDate   *time.Time `json:"end_date,omitempty"`   // NEW
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}
```

#### 3.2 Calendar View Data

```go
// CalendarDay represents a single day in the calendar
type CalendarDay struct {
    Date      time.Time
    Issues    []*CalendarIssue
    IsToday   bool
    IsWeekend bool
    IsOtherMonth bool // For days from prev/next month shown in grid
}

// CalendarIssue wraps an issue with calendar-specific metadata
type CalendarIssue struct {
    *issues.Issue
    Project     *projects.Project
    Column      *columns.Column
    Assignees   []*users.User
    DaysUntilDue int   // Negative if overdue
    IsOverdue   bool
}

// CalendarData holds data for the calendar page
type CalendarData struct {
    Title           string
    User            *users.User
    Workspace       *workspaces.Workspace
    Workspaces      []*workspaces.Workspace
    Teams           []*teams.Team
    Projects        []*projects.Project

    // Calendar-specific
    Year            int
    Month           time.Month
    MonthName       string
    Days            [][]CalendarDay // 6 weeks x 7 days grid
    PrevMonth       string          // URL param: "2024-01"
    NextMonth       string          // URL param: "2024-03"

    // For quick navigation
    Today           time.Time
    CurrentWeekStart time.Time
    CurrentWeekEnd   time.Time

    // Active view state
    ActiveView      string // "calendar"
    Columns         []*columns.Column
    DefaultProjectID string

    // Standard fields
    ActiveTeamID    string
    ActiveProjectID string
    ActiveNav       string
    Breadcrumbs     []Breadcrumb
}
```

#### 3.3 Gantt Chart Data

```go
// GanttIssue wraps an issue with Gantt-specific metadata
type GanttIssue struct {
    *issues.Issue
    Project       *projects.Project
    Column        *columns.Column
    Assignees     []*users.User

    // Gantt positioning (percentage-based for timeline)
    LeftOffset    float64 // 0-100% position on timeline
    Width         float64 // 0-100% width on timeline
    Row           int     // Vertical row position

    // Computed dates (fallback to created/updated if no explicit dates)
    EffectiveStart time.Time
    EffectiveEnd   time.Time
    HasExplicitDates bool
}

// GanttData holds data for the Gantt chart page
type GanttData struct {
    Title           string
    User            *users.User
    Workspace       *workspaces.Workspace
    Workspaces      []*workspaces.Workspace
    Teams           []*teams.Team
    Projects        []*projects.Project

    // Gantt-specific
    Issues          []*GanttIssue
    TimelineStart   time.Time
    TimelineEnd     time.Time
    TimelineDays    int
    TodayOffset     float64 // Position of today marker (0-100%)

    // Time scale
    Scale           string   // "day", "week", "month"
    HeaderDates     []GanttHeaderDate

    // Grouping
    GroupBy         string   // "project", "assignee", "column", "none"
    Groups          []*GanttGroup

    // Active view state
    ActiveView      string // "gantt"
    Columns         []*columns.Column
    DefaultProjectID string

    // Standard fields
    ActiveTeamID    string
    ActiveProjectID string
    ActiveNav       string
    Breadcrumbs     []Breadcrumb
}

// GanttHeaderDate represents a date marker in the timeline header
type GanttHeaderDate struct {
    Date    time.Time
    Label   string  // "Mon 15", "Week 3", "Feb"
    Offset  float64 // Position on timeline (0-100%)
    IsToday bool
    IsWeekend bool
}

// GanttGroup represents a group of issues (by project, assignee, etc.)
type GanttGroup struct {
    ID      string
    Name    string
    Issues  []*GanttIssue
}
```

### 4. UI Components

#### 4.1 View Switcher (shared across views)

Add a view switcher to the Issues page header:

```html
<div class="view-switcher">
  <a href="/w/{{.Workspace.Slug}}/issues?view=list"
     class="view-btn {{if eq .ActiveView "list"}}active{{end}}"
     title="List view">
    <svg><!-- list icon --></svg>
  </a>
  <a href="/w/{{.Workspace.Slug}}/board/{{.DefaultProjectID}}"
     class="view-btn {{if eq .ActiveView "board"}}active{{end}}"
     title="Board view">
    <svg><!-- kanban icon --></svg>
  </a>
  <a href="/w/{{.Workspace.Slug}}/issues?view=calendar"
     class="view-btn {{if eq .ActiveView "calendar"}}active{{end}}"
     title="Calendar view">
    <svg><!-- calendar icon --></svg>
  </a>
  <a href="/w/{{.Workspace.Slug}}/issues?view=gantt"
     class="view-btn {{if eq .ActiveView "gantt"}}active{{end}}"
     title="Gantt chart">
    <svg><!-- gantt-chart icon --></svg>
  </a>
</div>
```

#### 4.2 Calendar View Template

```html
{{define "content"}}
<div class="page-header flex items-center justify-between">
  <div class="flex items-center gap-4">
    <h1>Issues</h1>
    {{template "view-switcher" .}}
  </div>
  <div class="flex gap-2">
    <!-- Calendar navigation -->
    <div class="calendar-nav">
      <a href="?view=calendar&month={{.PrevMonth}}" class="btn btn-ghost btn-icon">
        <svg><!-- chevron-left --></svg>
      </a>
      <span class="calendar-month-label">{{.MonthName}} {{.Year}}</span>
      <a href="?view=calendar&month={{.NextMonth}}" class="btn btn-ghost btn-icon">
        <svg><!-- chevron-right --></svg>
      </a>
      <a href="?view=calendar" class="btn btn-secondary btn-sm">Today</a>
    </div>
    <button class="btn btn-primary" data-modal="create-issue-modal">
      <svg><!-- plus icon --></svg>
      New Issue
    </button>
  </div>
</div>

<div class="calendar-container">
  <!-- Day headers -->
  <div class="calendar-header">
    <div class="calendar-header-cell">Mon</div>
    <div class="calendar-header-cell">Tue</div>
    <div class="calendar-header-cell">Wed</div>
    <div class="calendar-header-cell">Thu</div>
    <div class="calendar-header-cell">Fri</div>
    <div class="calendar-header-cell weekend">Sat</div>
    <div class="calendar-header-cell weekend">Sun</div>
  </div>

  <!-- Calendar grid -->
  <div class="calendar-grid">
    {{range .Days}}
    <div class="calendar-week">
      {{range .}}
      <div class="calendar-day {{if .IsToday}}today{{end}} {{if .IsWeekend}}weekend{{end}} {{if .IsOtherMonth}}other-month{{end}}">
        <div class="calendar-day-header">
          <span class="calendar-day-number">{{.Date.Day}}</span>
          {{if .IsToday}}<span class="today-badge">Today</span>{{end}}
        </div>
        <div class="calendar-day-issues">
          {{range .Issues}}
          <a href="/w/{{$.Workspace.Slug}}/issue/{{.Key}}"
             class="calendar-issue {{if .IsOverdue}}overdue{{end}}"
             data-issue-key="{{.Key}}"
             draggable="true">
            <span class="calendar-issue-key">{{.Key}}</span>
            <span class="calendar-issue-title">{{.Title}}</span>
          </a>
          {{end}}
        </div>
        {{if gt (len .Issues) 3}}
        <button class="calendar-more-btn" data-date="{{.Date.Format "2006-01-02"}}">
          +{{sub (len .Issues) 3}} more
        </button>
        {{end}}
      </div>
      {{end}}
    </div>
    {{end}}
  </div>
</div>

<!-- Quick issue preview popover (shown on hover/click) -->
<div id="issue-popover" class="issue-popover hidden">
  <div class="issue-popover-header">
    <span class="issue-key-badge"></span>
    <span class="status-badge"></span>
  </div>
  <div class="issue-popover-title"></div>
  <div class="issue-popover-meta">
    <span class="issue-project"></span>
    <span class="issue-assignees"></span>
  </div>
</div>
{{end}}
```

#### 4.3 Gantt Chart View Template

```html
{{define "content"}}
<div class="page-header flex items-center justify-between">
  <div class="flex items-center gap-4">
    <h1>Issues</h1>
    {{template "view-switcher" .}}
  </div>
  <div class="flex gap-2">
    <!-- Gantt controls -->
    <div class="gantt-controls">
      <div class="dropdown">
        <button class="dropdown-trigger btn btn-secondary">
          Scale: {{.Scale}}
          <svg><!-- chevron-down --></svg>
        </button>
        <div class="dropdown-menu hidden">
          <a href="?view=gantt&scale=day" class="dropdown-item">Day</a>
          <a href="?view=gantt&scale=week" class="dropdown-item">Week</a>
          <a href="?view=gantt&scale=month" class="dropdown-item">Month</a>
        </div>
      </div>
      <div class="dropdown">
        <button class="dropdown-trigger btn btn-secondary">
          Group: {{.GroupBy}}
          <svg><!-- chevron-down --></svg>
        </button>
        <div class="dropdown-menu hidden">
          <a href="?view=gantt&group=none" class="dropdown-item">None</a>
          <a href="?view=gantt&group=project" class="dropdown-item">Project</a>
          <a href="?view=gantt&group=assignee" class="dropdown-item">Assignee</a>
          <a href="?view=gantt&group=column" class="dropdown-item">Status</a>
        </div>
      </div>
      <a href="?view=gantt" class="btn btn-secondary btn-sm">Today</a>
    </div>
    <button class="btn btn-primary" data-modal="create-issue-modal">
      <svg><!-- plus icon --></svg>
      New Issue
    </button>
  </div>
</div>

<div class="gantt-container">
  <!-- Fixed left column with issue names -->
  <div class="gantt-sidebar">
    <div class="gantt-sidebar-header">
      <span>Issue</span>
    </div>
    <div class="gantt-sidebar-body">
      {{range .Groups}}
      {{if .Name}}
      <div class="gantt-group-header">
        <span class="gantt-group-name">{{.Name}}</span>
        <span class="gantt-group-count">{{len .Issues}}</span>
      </div>
      {{end}}
      {{range .Issues}}
      <div class="gantt-row-label" data-issue-key="{{.Key}}">
        <span class="issue-key">{{.Key}}</span>
        <span class="issue-title truncate">{{.Title}}</span>
      </div>
      {{end}}
      {{end}}
    </div>
  </div>

  <!-- Scrollable timeline area -->
  <div class="gantt-timeline">
    <!-- Timeline header with date markers -->
    <div class="gantt-timeline-header">
      {{range .HeaderDates}}
      <div class="gantt-date-marker {{if .IsToday}}today{{end}} {{if .IsWeekend}}weekend{{end}}"
           style="left: {{.Offset}}%;">
        <span class="gantt-date-label">{{.Label}}</span>
      </div>
      {{end}}
    </div>

    <!-- Timeline body with bars -->
    <div class="gantt-timeline-body">
      <!-- Today line -->
      <div class="gantt-today-line" style="left: {{.TodayOffset}}%;"></div>

      <!-- Weekend columns (visual guides) -->
      {{range .HeaderDates}}
      {{if .IsWeekend}}
      <div class="gantt-weekend-column" style="left: {{.Offset}}%;"></div>
      {{end}}
      {{end}}

      <!-- Issue bars -->
      {{range .Groups}}
      {{if .Name}}
      <div class="gantt-group-spacer"></div>
      {{end}}
      {{range .Issues}}
      <div class="gantt-row">
        <a href="/w/{{$.Workspace.Slug}}/issue/{{.Key}}"
           class="gantt-bar {{if not .HasExplicitDates}}no-dates{{end}}"
           style="left: {{.LeftOffset}}%; width: {{.Width}}%;"
           data-issue-key="{{.Key}}"
           draggable="true">
          <span class="gantt-bar-title">{{.Title}}</span>
        </a>
      </div>
      {{end}}
      {{end}}
    </div>
  </div>
</div>

<script>
// Gantt drag to resize/move
const gantt = {
  init() {
    this.setupDragHandles();
    this.setupTimelineScroll();
  },

  setupDragHandles() {
    document.querySelectorAll('.gantt-bar').forEach(bar => {
      // Add resize handles
      const leftHandle = document.createElement('div');
      leftHandle.className = 'gantt-resize-handle left';
      bar.appendChild(leftHandle);

      const rightHandle = document.createElement('div');
      rightHandle.className = 'gantt-resize-handle right';
      bar.appendChild(rightHandle);

      // Setup drag events
      this.setupBarDrag(bar);
      this.setupResizeHandles(bar, leftHandle, rightHandle);
    });
  },

  setupBarDrag(bar) {
    let startX, startLeft;

    bar.addEventListener('dragstart', (e) => {
      startX = e.clientX;
      startLeft = parseFloat(bar.style.left);
      e.dataTransfer.effectAllowed = 'move';
    });

    // ... drag implementation
  },

  setupResizeHandles(bar, left, right) {
    // ... resize implementation
  },

  setupTimelineScroll() {
    const timeline = document.querySelector('.gantt-timeline');
    const sidebar = document.querySelector('.gantt-sidebar-body');

    // Sync vertical scroll between sidebar and timeline
    timeline.addEventListener('scroll', () => {
      sidebar.scrollTop = timeline.scrollTop;
    });
  },

  async updateIssueDates(issueKey, startDate, endDate) {
    try {
      await app.api.patch(`/issues/${issueKey}`, {
        start_date: startDate,
        end_date: endDate,
      });
    } catch (error) {
      console.error('Failed to update dates:', error);
    }
  }
};

gantt.init();
</script>
{{end}}
```

### 5. CSS Additions

```css
/* ========================================
   View Switcher
   ======================================== */

.view-switcher {
  display: inline-flex;
  border: 1px solid hsl(var(--border));
  border-radius: 0;
}

.view-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  color: hsl(var(--muted-foreground));
  background: transparent;
  border-right: 1px solid hsl(var(--border));
  transition: all 150ms ease;
}

.view-btn:last-child {
  border-right: none;
}

.view-btn:hover {
  background: hsl(var(--muted));
  color: hsl(var(--foreground));
}

.view-btn.active {
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.view-btn svg {
  width: 16px;
  height: 16px;
}

/* ========================================
   Calendar View
   ======================================== */

.calendar-container {
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
}

.calendar-nav {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.calendar-month-label {
  font-size: 1rem;
  font-weight: 600;
  min-width: 140px;
  text-align: center;
}

.calendar-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  border-bottom: 1px solid hsl(var(--border));
}

.calendar-header-cell {
  padding: 0.75rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: hsl(var(--muted-foreground));
  text-align: center;
}

.calendar-header-cell.weekend {
  color: hsl(var(--muted-foreground) / 0.5);
}

.calendar-grid {
  display: flex;
  flex-direction: column;
}

.calendar-week {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  border-bottom: 1px solid hsl(var(--border));
}

.calendar-week:last-child {
  border-bottom: none;
}

.calendar-day {
  min-height: 120px;
  padding: 0.5rem;
  border-right: 1px solid hsl(var(--border));
  background: hsl(var(--background));
}

.calendar-day:last-child {
  border-right: none;
}

.calendar-day.weekend {
  background: hsl(var(--muted) / 0.3);
}

.calendar-day.other-month {
  opacity: 0.4;
}

.calendar-day.today {
  background: hsl(var(--primary) / 0.05);
}

.calendar-day-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.calendar-day-number {
  font-size: 0.875rem;
  font-weight: 500;
  color: hsl(var(--foreground));
}

.calendar-day.today .calendar-day-number {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  border-radius: 50%;
}

.today-badge {
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  color: hsl(var(--primary));
}

.calendar-day-issues {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.calendar-issue {
  display: block;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  background: hsl(var(--muted));
  border-left: 3px solid hsl(var(--primary));
  border-radius: 0;
  transition: all 150ms ease;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.calendar-issue:hover {
  background: hsl(var(--accent));
  border-left-color: hsl(var(--ring));
}

.calendar-issue.overdue {
  border-left-color: hsl(var(--destructive));
}

.calendar-issue-key {
  font-weight: 500;
  color: hsl(var(--muted-foreground));
  margin-right: 0.25rem;
}

.calendar-issue-title {
  color: hsl(var(--foreground));
}

.calendar-more-btn {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
  background: none;
  border: none;
  cursor: pointer;
}

.calendar-more-btn:hover {
  color: hsl(var(--foreground));
  text-decoration: underline;
}

/* Issue Popover */
.issue-popover {
  position: fixed;
  z-index: 100;
  width: 280px;
  padding: 1rem;
  background: hsl(var(--popover));
  border: 1px solid hsl(var(--border));
  box-shadow: 0 4px 12px rgb(0 0 0 / 0.1);
  animation: fade-in 150ms ease;
}

.issue-popover.hidden {
  display: none;
}

.issue-popover-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.issue-popover-title {
  font-size: 0.875rem;
  font-weight: 500;
  margin-bottom: 0.5rem;
}

.issue-popover-meta {
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
  display: flex;
  gap: 0.75rem;
}

/* ========================================
   Gantt Chart View
   ======================================== */

.gantt-container {
  display: flex;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  height: calc(100vh - var(--topbar-height) - 8rem);
  overflow: hidden;
}

.gantt-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Gantt Sidebar */
.gantt-sidebar {
  flex-shrink: 0;
  width: 280px;
  border-right: 1px solid hsl(var(--border));
  display: flex;
  flex-direction: column;
}

.gantt-sidebar-header {
  padding: 0.75rem 1rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted));
  border-bottom: 1px solid hsl(var(--border));
  height: 48px;
  display: flex;
  align-items: center;
}

.gantt-sidebar-body {
  flex: 1;
  overflow-y: auto;
}

.gantt-group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted) / 0.5);
  border-bottom: 1px solid hsl(var(--border));
}

.gantt-group-count {
  font-weight: 500;
}

.gantt-row-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  height: 36px;
  border-bottom: 1px solid hsl(var(--border));
  cursor: pointer;
}

.gantt-row-label:hover {
  background: hsl(var(--muted) / 0.5);
}

.gantt-row-label .issue-key {
  font-size: 0.75rem;
  font-weight: 500;
  color: hsl(var(--muted-foreground));
  flex-shrink: 0;
}

.gantt-row-label .issue-title {
  font-size: 0.875rem;
  color: hsl(var(--foreground));
  flex: 1;
  min-width: 0;
}

/* Gantt Timeline */
.gantt-timeline {
  flex: 1;
  overflow-x: auto;
  overflow-y: auto;
  position: relative;
}

.gantt-timeline-header {
  position: sticky;
  top: 0;
  height: 48px;
  background: hsl(var(--muted));
  border-bottom: 1px solid hsl(var(--border));
  display: flex;
  align-items: flex-end;
  z-index: 10;
  min-width: 100%;
}

.gantt-date-marker {
  position: absolute;
  bottom: 0;
  padding: 0 0.25rem 0.5rem;
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
  border-left: 1px solid hsl(var(--border));
  transform: translateX(-50%);
}

.gantt-date-marker.today {
  color: hsl(var(--primary));
  font-weight: 600;
}

.gantt-date-marker.weekend {
  color: hsl(var(--muted-foreground) / 0.5);
}

.gantt-date-label {
  white-space: nowrap;
}

.gantt-timeline-body {
  position: relative;
  min-width: 100%;
  min-height: 100%;
}

.gantt-today-line {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2px;
  background: hsl(var(--primary));
  z-index: 5;
}

.gantt-weekend-column {
  position: absolute;
  top: 0;
  bottom: 0;
  width: calc(100% / var(--timeline-days));
  background: hsl(var(--muted) / 0.3);
  pointer-events: none;
}

.gantt-group-spacer {
  height: 32px;
  border-bottom: 1px solid hsl(var(--border));
}

.gantt-row {
  position: relative;
  height: 36px;
  border-bottom: 1px solid hsl(var(--border));
}

.gantt-bar {
  position: absolute;
  top: 4px;
  height: 28px;
  display: flex;
  align-items: center;
  padding: 0 0.5rem;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  font-size: 0.75rem;
  border-radius: 0;
  cursor: pointer;
  transition: opacity 150ms ease;
  overflow: hidden;
}

.gantt-bar:hover {
  opacity: 0.9;
}

.gantt-bar.no-dates {
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  border: 1px dashed hsl(var(--border));
}

.gantt-bar-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Resize handles */
.gantt-resize-handle {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 8px;
  cursor: ew-resize;
  opacity: 0;
  transition: opacity 150ms ease;
}

.gantt-bar:hover .gantt-resize-handle {
  opacity: 1;
}

.gantt-resize-handle.left {
  left: 0;
  background: linear-gradient(to right, hsl(var(--background) / 0.5), transparent);
}

.gantt-resize-handle.right {
  right: 0;
  background: linear-gradient(to left, hsl(var(--background) / 0.5), transparent);
}

/* ========================================
   Date Input Styles (for issue forms)
   ======================================== */

.date-input-group {
  display: flex;
  gap: 1rem;
}

.date-input-group .form-group {
  flex: 1;
}

.date-display {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.875rem;
  color: hsl(var(--foreground));
}

.date-display.overdue {
  color: hsl(var(--destructive));
}

.date-display svg {
  width: 14px;
  height: 14px;
  color: hsl(var(--muted-foreground));
}

/* ========================================
   Responsive Calendar & Gantt
   ======================================== */

@media (max-width: 1024px) {
  .gantt-sidebar {
    width: 200px;
  }
}

@media (max-width: 768px) {
  .calendar-day {
    min-height: 80px;
    padding: 0.25rem;
  }

  .calendar-issue {
    padding: 0.125rem 0.25rem;
    font-size: 0.625rem;
  }

  .calendar-issue-key {
    display: none;
  }

  .gantt-sidebar {
    width: 120px;
  }

  .gantt-row-label .issue-key {
    display: none;
  }
}
```

### 6. API Additions

Update the issue PATCH endpoint to support date fields:

```go
// UpdateIn represents the input for updating an issue
type UpdateIn struct {
    Title     *string    `json:"title,omitempty"`
    CycleID   *string    `json:"cycle_id,omitempty"`
    DueDate   *time.Time `json:"due_date,omitempty"`   // NEW
    StartDate *time.Time `json:"start_date,omitempty"` // NEW
    EndDate   *time.Time `json:"end_date,omitempty"`   // NEW
}

// PATCH /api/v1/issues/{key}
func (h *Issue) Update(c *mizu.Ctx) error {
    key := c.Param("key")
    var in issues.UpdateIn
    if err := c.BindJSON(&in); err != nil {
        return BadRequest(c, err.Error())
    }

    issue, err := h.issues.Update(c.Request().Context(), key, &in)
    if err != nil {
        return handleError(c, err)
    }

    return OK(c, issue)
}
```

Add bulk date update endpoint for Gantt drag operations:

```go
// POST /api/v1/issues/bulk-dates
type BulkDatesIn struct {
    Updates []struct {
        Key       string     `json:"key"`
        StartDate *time.Time `json:"start_date,omitempty"`
        EndDate   *time.Time `json:"end_date,omitempty"`
    } `json:"updates"`
}

func (h *Issue) BulkUpdateDates(c *mizu.Ctx) error {
    var in BulkDatesIn
    if err := c.BindJSON(&in); err != nil {
        return BadRequest(c, err.Error())
    }

    ctx := c.Request().Context()
    for _, update := range in.Updates {
        h.issues.Update(ctx, update.Key, &issues.UpdateIn{
            StartDate: update.StartDate,
            EndDate:   update.EndDate,
        })
    }

    return OK(c, map[string]string{"status": "ok"})
}
```

### 7. Handler Changes

Add view routing to the Issues handler:

```go
// Issues renders the issues page with different view modes
func (h *Page) Issues(c *mizu.Ctx) error {
    view := c.Query("view")
    if view == "" {
        view = "list"
    }

    switch view {
    case "calendar":
        return h.IssuesCalendar(c)
    case "gantt":
        return h.IssuesGantt(c)
    default:
        return h.IssuesList(c)
    }
}

// IssuesCalendar renders the calendar view
func (h *Page) IssuesCalendar(c *mizu.Ctx) error {
    userID := h.getUserID(c)
    if userID == "" {
        http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
        return nil
    }

    ctx := c.Request().Context()
    workspaceSlug := c.Param("workspace")

    // Parse month parameter or use current
    monthStr := c.Query("month")
    var year int
    var month time.Month
    if monthStr != "" {
        t, err := time.Parse("2006-01", monthStr)
        if err == nil {
            year, month = t.Year(), t.Month()
        }
    }
    if year == 0 {
        now := time.Now()
        year, month = now.Year(), now.Month()
    }

    // Fetch common data
    user, _ := h.users.GetByID(ctx, userID)
    workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
    workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

    // Fetch issues with due dates
    var allIssues []*issues.Issue
    // ... collect issues from projects

    // Build calendar grid
    days := buildCalendarGrid(year, month, allIssues)

    // Calculate prev/next months
    prevMonth := time.Date(year, month-1, 1, 0, 0, 0, 0, time.UTC)
    nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)

    return render(h, c, "calendar", CalendarData{
        Title:      "Calendar",
        User:       user,
        Workspace:  workspace,
        Workspaces: workspaceList,
        Year:       year,
        Month:      month,
        MonthName:  month.String(),
        Days:       days,
        PrevMonth:  prevMonth.Format("2006-01"),
        NextMonth:  nextMonth.Format("2006-01"),
        Today:      time.Now(),
        ActiveView: "calendar",
        ActiveNav:  "issues",
    })
}

// IssuesGantt renders the Gantt chart view
func (h *Page) IssuesGantt(c *mizu.Ctx) error {
    // ... similar structure to IssuesCalendar
    // Build timeline from issues with start/end dates
    // Calculate bar positions based on date range
}
```

### 8. Helper Functions

```go
// buildCalendarGrid creates a 6x7 grid of days for the calendar
func buildCalendarGrid(year int, month time.Month, issues []*issues.Issue) [][]CalendarDay {
    // Get first day of month and calculate grid start
    firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
    weekday := int(firstOfMonth.Weekday())
    if weekday == 0 { // Sunday
        weekday = 7
    }
    gridStart := firstOfMonth.AddDate(0, 0, -(weekday - 1))

    // Index issues by due date for quick lookup
    issuesByDate := make(map[string][]*CalendarIssue)
    for _, issue := range issues {
        if issue.DueDate != nil {
            key := issue.DueDate.Format("2006-01-02")
            issuesByDate[key] = append(issuesByDate[key], &CalendarIssue{
                Issue:     issue,
                IsOverdue: issue.DueDate.Before(time.Now()),
            })
        }
    }

    // Build 6 weeks of days
    weeks := make([][]CalendarDay, 6)
    today := time.Now().Truncate(24 * time.Hour)

    for w := 0; w < 6; w++ {
        weeks[w] = make([]CalendarDay, 7)
        for d := 0; d < 7; d++ {
            date := gridStart.AddDate(0, 0, w*7+d)
            dateKey := date.Format("2006-01-02")

            weeks[w][d] = CalendarDay{
                Date:         date,
                Issues:       issuesByDate[dateKey],
                IsToday:      date.Equal(today),
                IsWeekend:    d >= 5,
                IsOtherMonth: date.Month() != month,
            }
        }
    }

    return weeks
}

// calculateGanttPosition computes the left offset and width for a Gantt bar
func calculateGanttPosition(start, end, timelineStart, timelineEnd time.Time) (leftOffset, width float64) {
    totalDays := timelineEnd.Sub(timelineStart).Hours() / 24

    startDays := start.Sub(timelineStart).Hours() / 24
    endDays := end.Sub(timelineStart).Hours() / 24

    leftOffset = (startDays / totalDays) * 100
    width = ((endDays - startDays) / totalDays) * 100

    // Clamp to visible range
    if leftOffset < 0 {
        width += leftOffset
        leftOffset = 0
    }
    if leftOffset+width > 100 {
        width = 100 - leftOffset
    }

    return leftOffset, max(width, 1) // Minimum 1% width for visibility
}
```

### 9. Icons

Add new icons for the view switcher:

```html
{{define "icon"}}
{{if eq . "calendar"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M8 2v4"/>
    <path d="M16 2v4"/>
    <rect width="18" height="18" x="3" y="4" rx="2"/>
    <path d="M3 10h18"/>
</svg>
{{else if eq . "gantt-chart"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M3 3v18h18"/>
    <rect x="7" y="6" width="10" height="3"/>
    <rect x="5" y="12" width="8" height="3"/>
    <rect x="9" y="18" width="6" height="3"/>
</svg>
{{else if eq . "list"}}
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <line x1="8" x2="21" y1="6" y2="6"/>
    <line x1="8" x2="21" y1="12" y2="12"/>
    <line x1="8" x2="21" y1="18" y2="18"/>
    <line x1="3" x2="3.01" y1="6" y2="6"/>
    <line x1="3" x2="3.01" y1="12" y2="12"/>
    <line x1="3" x2="3.01" y1="18" y2="18"/>
</svg>
{{end}}
{{end}}
```

### 10. JavaScript Additions

```javascript
// calendar.js - Add to app.js

const calendar = {
  init() {
    this.setupDragDrop();
    this.setupPopovers();
  },

  setupDragDrop() {
    // Make issues draggable between days
    document.querySelectorAll('.calendar-issue').forEach(issue => {
      issue.addEventListener('dragstart', (e) => {
        e.dataTransfer.setData('text/plain', issue.dataset.issueKey);
        e.dataTransfer.effectAllowed = 'move';
      });
    });

    document.querySelectorAll('.calendar-day').forEach(day => {
      day.addEventListener('dragover', (e) => {
        e.preventDefault();
        day.classList.add('drag-over');
      });

      day.addEventListener('dragleave', () => {
        day.classList.remove('drag-over');
      });

      day.addEventListener('drop', async (e) => {
        e.preventDefault();
        day.classList.remove('drag-over');

        const issueKey = e.dataTransfer.getData('text/plain');
        const newDate = day.querySelector('.calendar-day-number').textContent;
        // TODO: Get full date from day element

        try {
          await app.api.patch(`/issues/${issueKey}`, {
            due_date: newDate,
          });
          location.reload();
        } catch (error) {
          console.error('Failed to update due date:', error);
        }
      });
    });
  },

  setupPopovers() {
    const popover = document.getElementById('issue-popover');
    if (!popover) return;

    document.querySelectorAll('.calendar-issue').forEach(issue => {
      issue.addEventListener('mouseenter', async (e) => {
        const key = issue.dataset.issueKey;
        // Fetch issue details and show popover
        // Position popover near cursor
      });

      issue.addEventListener('mouseleave', () => {
        popover.classList.add('hidden');
      });
    });
  }
};

// Initialize on page load
if (document.querySelector('.calendar-container')) {
  calendar.init();
}
```

## Implementation Checklist

### Schema & API
- [ ] Add `due_date`, `start_date`, `end_date` columns to issues table
- [ ] Update Issue struct with new date fields
- [ ] Update issue store with date field handling
- [ ] Update PATCH /issues/{key} to accept date fields
- [ ] Add POST /issues/bulk-dates for Gantt bulk updates

### Calendar View
- [ ] Create `CalendarData` and `CalendarDay` types in page.go
- [ ] Implement `IssuesCalendar` handler
- [ ] Implement `buildCalendarGrid` helper
- [ ] Create `pages/calendar.html` template
- [ ] Add calendar CSS to default.css
- [ ] Add calendar JavaScript to app.js
- [ ] Add calendar icon to icon template

### Gantt View
- [ ] Create `GanttData` and `GanttIssue` types in page.go
- [ ] Implement `IssuesGantt` handler
- [ ] Implement `calculateGanttPosition` helper
- [ ] Create `pages/gantt.html` template
- [ ] Add Gantt CSS to default.css
- [ ] Add Gantt JavaScript (drag to resize/move)
- [ ] Add gantt-chart icon to icon template

### Shared Components
- [ ] Create view switcher partial template
- [ ] Add view switcher to Issues page header
- [ ] Update routes in server.go for view parameter
- [ ] Add date inputs to issue create/edit forms
- [ ] Update issue detail page with date fields

### Testing
- [ ] Add tests for calendar grid building
- [ ] Add tests for Gantt position calculation
- [ ] Add e2e tests for view switching
- [ ] Add e2e tests for date drag-drop

## Migration Notes

- New date columns are nullable (no migration required for existing issues)
- Issues without dates appear at the bottom of Gantt (shown with dashed border)
- Calendar only shows issues with due_date set
- Cycles provide default date ranges for issues attached to them
