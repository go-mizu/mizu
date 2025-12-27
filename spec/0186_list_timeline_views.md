# Feature Spec: Trello List View and Timeline View

**Feature ID:** 0186
**Created:** 2025-12-27
**Status:** Implementation Ready

## Overview

Add two new view modes to the Trello-style Kanban board:
1. **List View** - A table-based view showing all cards in a sortable, filterable list format
2. **Timeline View** - A Gantt-style timeline showing cards with dates across a horizontal time axis

Users can switch between Kanban, List, and Timeline views using a view switcher in the board header.

## Goals

- Provide alternative ways to visualize board data beyond the traditional Kanban columns
- Match Trello's native List and Timeline view functionality and interactivity
- Enable seamless switching between views without page reload
- Maintain full CRUD capabilities in all views (create, edit, move, delete cards)
- Support drag-and-drop interactions where applicable

## User Stories

1. As a user, I want to see all my cards in a table format so I can quickly scan and sort by different properties
2. As a user, I want to see my cards on a timeline so I can visualize project schedules and deadlines
3. As a user, I want to switch between Kanban, List, and Timeline views without losing my place
4. As a user, I want to drag cards to different dates in Timeline view to update their schedule
5. As a user, I want to inline edit card properties in List view

## Technical Design

### 1. View Switcher Component

**Location:** Board header, next to the board title

```html
<div class="view-switcher">
  <button class="view-btn active" data-view="kanban">
    <svg><!-- Kanban icon --></svg>
    Board
  </button>
  <button class="view-btn" data-view="list">
    <svg><!-- List icon --></svg>
    List
  </button>
  <button class="view-btn" data-view="timeline">
    <svg><!-- Timeline icon --></svg>
    Timeline
  </button>
</div>
```

**Behavior:**
- Active view is highlighted with background color
- Clicking a view button switches the view instantly (no page reload)
- View preference is persisted in localStorage per board
- URL query param `?view=list|timeline|kanban` for shareable links

### 2. List View

**Layout:**
- Full-width table with sortable columns
- Sticky header row
- Rows grouped by list (column) with collapsible sections
- Quick add row at bottom of each section

**Columns:**
| Column | Width | Sortable | Description |
|--------|-------|----------|-------------|
| Checkbox | 40px | No | Multi-select for bulk actions |
| Card | flex | Yes | Card title with inline editing |
| List | 150px | Yes | Current list/status with dropdown |
| Labels | 120px | No | Color labels with hover expand |
| Members | 100px | No | Avatar stack |
| Due Date | 120px | Yes | Date with status indicator |
| Created | 100px | Yes | Creation timestamp |

**Interactions:**
- Click row to open card detail modal
- Click checkbox to select (enables bulk actions toolbar)
- Click list cell to show dropdown for moving card
- Hover row to show quick actions (open, archive, delete)
- Drag row handle to reorder within list
- Drag row to different list section to move card
- Press Enter on focused row to open card
- Inline edit title by clicking on it

**Filtering & Sorting:**
- Click column header to sort (asc/desc/none)
- Filter bar above table with filters for:
  - Labels
  - Members
  - Due date (overdue, due soon, no date)
  - List/status

### 3. Timeline View

**Layout:**
- Left sidebar (250px): Card list grouped by list
- Main area: Horizontal timeline with scrollable date axis
- Header: Date range selector, zoom controls, today button

**Time Scales:**
- Day view (default): Shows 14 days
- Week view: Shows 8 weeks
- Month view: Shows 6 months

**Card Representation:**
- Cards with dates shown as horizontal bars on the timeline
- Bar start = start_date (or created_at if no start)
- Bar end = due_date (or start + 1 day if no due)
- Bar color = list color or priority color
- Cards without dates appear in "Unscheduled" section at top

**Interactions:**
- Drag card bar left edge to change start date
- Drag card bar right edge to change due date
- Drag entire bar to shift both dates
- Click card bar to open card detail modal
- Drag card from sidebar to timeline to set dates
- Right-click card bar for context menu
- Scroll horizontally to navigate time
- Zoom in/out with controls or Ctrl+scroll

**Visual Indicators:**
- Today line: Vertical red dashed line
- Weekends: Light gray background
- Overdue: Red bar background
- Due soon (within 24h): Yellow bar background
- Completed (in Done list): Green bar with checkmark

### 4. Data Requirements

**Existing fields used:**
- `start_date` - Timeline bar start
- `due_date` - Timeline bar end / List due date column
- `column_id` - List grouping / Timeline sidebar grouping
- `position` - Order within list
- `created_at` - Fallback for dates, List created column
- `title` - Display name
- `priority` - Optional bar coloring

**No new database fields required.**

### 5. API Endpoints

**Existing endpoints used:**
- `GET /api/v1/projects/{id}/issues` - Load all cards
- `PATCH /api/v1/issues/{key}` - Update card (title, dates)
- `POST /api/v1/issues/{key}/move` - Move to different list
- `GET /api/v1/projects/{id}/columns` - Load lists

**No new API endpoints required.**

### 6. State Management

**Client-side state:**
```javascript
const viewState = {
  currentView: 'kanban', // 'kanban' | 'list' | 'timeline'
  listView: {
    sortColumn: 'position',
    sortDirection: 'asc',
    filters: {
      labels: [],
      members: [],
      dueDate: null
    },
    collapsedLists: [],
    selectedCards: []
  },
  timelineView: {
    scale: 'day', // 'day' | 'week' | 'month'
    startDate: new Date(), // viewport start
    collapsedLists: []
  }
};
```

**localStorage keys:**
- `kanban-view-{boardId}`: Last used view
- `kanban-list-sort-{boardId}`: List view sort preferences
- `kanban-timeline-scale-{boardId}`: Timeline zoom level

### 7. Keyboard Shortcuts

**List View:**
- `↑/↓` - Navigate rows
- `Enter` - Open selected card
- `Space` - Toggle selection
- `Ctrl+A` - Select all
- `Escape` - Clear selection
- `Delete` - Archive selected (with confirmation)

**Timeline View:**
- `←/→` - Scroll timeline
- `+/-` - Zoom in/out
- `T` - Jump to today
- `Enter` - Open hovered card

### 8. CSS Architecture

**New CSS files/sections:**
```css
/* List View */
.list-view { }
.list-view-header { }
.list-view-table { }
.list-view-row { }
.list-view-cell { }
.list-view-group { }
.list-view-group-header { }

/* Timeline View */
.timeline-view { }
.timeline-sidebar { }
.timeline-main { }
.timeline-header { }
.timeline-grid { }
.timeline-card-bar { }
.timeline-today-line { }
.timeline-date-axis { }
```

### 9. Implementation Files

**Modified files:**
- `assets/views/trello/pages/board.html` - Add view switcher and view containers
- `assets/views/trello/layouts/default.html` - Add view-specific CSS

**New sections in board.html:**
- View switcher component
- List view container with table
- Timeline view container with grid
- JavaScript for view switching, list interactions, timeline interactions

## UI Mockups

### View Switcher
```
┌─────────────────────────────────────────────────────────────┐
│  📋 Board Name    [★]  │  [Board] [List] [Timeline]  │  👤👤 │
└─────────────────────────────────────────────────────────────┘
```

### List View
```
┌─────────────────────────────────────────────────────────────┐
│ □  Card                        │ List       │ Due      │ 👤  │
├─────────────────────────────────────────────────────────────┤
│ ▼ To Do (3)                                                 │
├─────────────────────────────────────────────────────────────┤
│ □  Design login page           │ To Do      │ Dec 28   │ JD  │
│ □  Write API docs              │ To Do      │ Dec 30   │ AS  │
│ □  Fix navigation bug          │ To Do      │ -        │ -   │
│ + Add a card                                                │
├─────────────────────────────────────────────────────────────┤
│ ▼ In Progress (2)                                           │
├─────────────────────────────────────────────────────────────┤
│ □  Implement auth flow         │ In Progr.. │ Dec 27   │ JD  │
│ □  Database schema             │ In Progr.. │ Jan 2    │ AS  │
│ + Add a card                                                │
└─────────────────────────────────────────────────────────────┘
```

### Timeline View
```
┌────────────┬────────────────────────────────────────────────┐
│            │  Dec 25  │  Dec 26  │  Dec 27  │  Dec 28  │... │
├────────────┼────────────────────────────────────────────────┤
│ ▼ To Do    │                      │░░░░░░░░░░░░│            │ Design login
│            │          │░░░░░░░░░░░░░░░░░░░░░░░│            │ Write docs
├────────────┼────────────────────────────────────────────────┤
│ ▼ In Prog  │░░░░░░░░░░░░░░░░░░░░░░|            │            │ Auth flow
│            │                      │░░░░░░░░░░░░░░░░░░░░░░░░│ DB schema
├────────────┼────────────────────────────────────────────────┤
│ Unscheduled│  • Fix navigation bug                          │
└────────────┴────────────────────────────────────────────────┘
                                    ↑ Today line
```

## Testing Plan

### Unit Tests
- View state management
- Date calculations for timeline
- Sort/filter logic for list view

### E2E Tests
1. Switch between all three views
2. Create card in List view
3. Move card via List view dropdown
4. Drag card in Timeline view to change dates
5. Inline edit card title in List view
6. Sort List view by different columns
7. Filter List view by label/member/date
8. Timeline zoom and scroll
9. Keyboard navigation in both views

### Manual Testing
- Cross-browser compatibility (Chrome, Firefox, Safari)
- Responsive behavior on tablet
- Performance with 100+ cards
- Drag-and-drop smoothness

## Rollout Plan

1. Implement view switcher component
2. Implement List view with basic table
3. Add List view interactivity (sort, filter, inline edit)
4. Implement Timeline view layout
5. Add Timeline view interactivity (drag dates, zoom)
6. QA and bug fixes
7. Release

## Success Metrics

- View switch latency < 100ms
- Timeline renders 100 cards < 500ms
- User engagement with new views (track view switches)
- Reduction in "need different view" feedback

## Open Questions

1. Should we persist view preference per-board or globally? **Decision: Per-board**
2. Should Timeline show hours for same-day items? **Decision: No, day granularity only**
3. Should we add print/export from List view? **Decision: Future enhancement**

## Appendix: Trello Reference

**Trello List View Features:**
- Table with columns: Card, List, Labels, Members, Due date
- Click row to open card
- Sortable columns
- Inline editing
- Bulk selection and actions
- Filter by labels, members, due date

**Trello Timeline View Features:**
- Horizontal time axis
- Cards as bars based on dates
- Drag to resize dates
- Drag to reposition
- Zoom levels (day/week/month)
- Today marker
- Group by list in sidebar
- Unscheduled section
