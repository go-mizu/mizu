# Spec 0187: Issue Activity Tracking

## Overview

This feature adds comprehensive activity tracking for issues in the Kanban blueprint. Every action related to an issue is captured and stored, enabling users to see a complete audit trail of changes. Activities are displayed both in a dedicated Activities page (accessible from the sidebar) and within each issue's detail page alongside comments.

## Goals

1. Capture all issue-related actions automatically
2. Display activities in a compact, readable format
3. Provide a global Activities page for workspace-wide activity view
4. Integrate activities seamlessly with existing comments in issue detail view

## Activity Types

The following actions will be tracked:

| Action Type | Description | Example |
|-------------|-------------|---------|
| `issue_created` | Issue was created | "created the issue" |
| `issue_updated` | Issue field(s) updated | "changed the title" |
| `status_changed` | Issue moved to different column | "changed status from Todo to In Progress" |
| `priority_changed` | Issue priority changed | "changed priority from None to High" |
| `assignee_added` | Assignee added to issue | "assigned John Doe" |
| `assignee_removed` | Assignee removed from issue | "unassigned John Doe" |
| `cycle_attached` | Issue attached to sprint | "added to Sprint 1" |
| `cycle_detached` | Issue removed from sprint | "removed from Sprint 1" |
| `start_date_changed` | Start date set/changed | "set start date to Jan 15, 2025" |
| `due_date_changed` | Due date set/changed | "set due date to Jan 20, 2025" |
| `comment_added` | Comment added to issue | "added a comment" |

## Data Model

### Activity Entity

```go
type Activity struct {
    ID         string    `json:"id"`
    IssueID    string    `json:"issue_id"`
    ActorID    string    `json:"actor_id"`     // User who performed the action
    Action     string    `json:"action"`       // Action type (see above)
    OldValue   string    `json:"old_value"`    // Previous value (JSON or string)
    NewValue   string    `json:"new_value"`    // New value (JSON or string)
    Metadata   string    `json:"metadata"`     // Additional context (JSON)
    CreatedAt  time.Time `json:"created_at"`
}
```

### Database Schema

```sql
CREATE TABLE IF NOT EXISTS activities (
    id         VARCHAR PRIMARY KEY,
    issue_id   VARCHAR NOT NULL,
    actor_id   VARCHAR NOT NULL,
    action     VARCHAR NOT NULL,
    old_value  VARCHAR,
    new_value  VARCHAR,
    metadata   VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## API Design

### Feature Contract (`feature/activities/api.go`)

```go
type API interface {
    Create(ctx context.Context, issueID, actorID string, in *CreateIn) (*Activity, error)
    GetByID(ctx context.Context, id string) (*Activity, error)
    ListByIssue(ctx context.Context, issueID string) ([]*Activity, error)
    ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*Activity, error)
    Delete(ctx context.Context, id string) error
}

type Store interface {
    Create(ctx context.Context, a *Activity) error
    GetByID(ctx context.Context, id string) (*Activity, error)
    ListByIssue(ctx context.Context, issueID string) ([]*Activity, error)
    ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*Activity, error)
    Delete(ctx context.Context, id string) error
    CountByIssue(ctx context.Context, issueID string) (int, error)
}

type CreateIn struct {
    Action   string `json:"action"`
    OldValue string `json:"old_value,omitempty"`
    NewValue string `json:"new_value,omitempty"`
    Metadata string `json:"metadata,omitempty"`
}
```

### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/issues/{issueID}/activities` | List activities for an issue |
| GET | `/api/v1/workspaces/{slug}/activities` | List activities for a workspace |

## UI Design

### 1. Sidebar Navigation (Default Theme)

Add "Activities" link after "Sprints" in the sidebar navigation:

```html
<!-- After Sprints link -->
<a href="/w/{{.Workspace.Slug}}/activities" class="flex items-center gap-2 px-2 py-1.5 text-sm rounded transition-colors ...">
  <svg><!-- Activity/Lightning icon --></svg>
  <span>Activities</span>
</a>
```

### 2. Activities Page (`/w/{workspace}/activities`)

A compact list view showing recent activities across all issues:

```
+------------------------------------------+
| Activities                               |
+------------------------------------------+
| [avatar] John changed status on PRJ-123  |
|          Todo -> In Progress             |
|          2 minutes ago                   |
+------------------------------------------+
| [avatar] Jane created PRJ-124            |
|          "Implement dark mode"           |
|          5 minutes ago                   |
+------------------------------------------+
| [avatar] John assigned Jane to PRJ-123   |
|          15 minutes ago                  |
+------------------------------------------+
```

Features:
- Compact row design with avatar, action text, and relative timestamp
- Issue key links to issue detail
- User names link to user profile (future)
- Infinite scroll or "Load more" pagination

### 3. Issue Detail Page Activity Section

Merge activities with comments in a unified timeline:

```
+------------------------------------------+
| Activity                                 |
+------------------------------------------+
| [avatar] John created the issue          |
|          Jan 15, 2025 3:04 PM            |
+------------------------------------------+
| [avatar] Jane changed priority           |
|          None -> High                    |
|          Jan 15, 2025 3:10 PM            |
+------------------------------------------+
| [avatar] John commented:                 |
|          "This looks good to me!"        |
|          Jan 15, 2025 4:00 PM            |
+------------------------------------------+
| [avatar] Jane changed status             |
|          Todo -> In Progress             |
|          Jan 16, 2025 9:00 AM            |
+------------------------------------------+
```

The timeline merges both activities and comments, sorted by creation time.

## Implementation Plan

### Phase 1: Core Infrastructure

1. **Create feature package** (`feature/activities/`)
   - `api.go` - Interface definitions and types
   - `service.go` - Service implementation

2. **Create store** (`store/duckdb/activities_store.go`)
   - SQL queries for CRUD operations
   - Workspace-scoped queries via JOINs

3. **Add schema** (`store/duckdb/schema.sql`)
   - Add activities table

4. **Write tests** (`store/duckdb/activities_store_test.go`)
   - Test all store methods

### Phase 2: API Layer

5. **Create API handler** (`app/web/handler/api/activity.go`)
   - List by issue
   - List by workspace

6. **Wire into server** (`app/web/server.go`)
   - Create store, service, handler
   - Register routes

### Phase 3: Activity Logging Integration

7. **Modify issues service** (`feature/issues/service.go`)
   - Inject activities service
   - Log activities on create, update, move, cycle operations

8. **Modify assignees service** (`feature/assignees/service.go`)
   - Inject activities service
   - Log activities on add/remove

9. **Modify comments service** (`feature/comments/service.go`)
   - Inject activities service
   - Log activities on comment create

### Phase 4: UI Implementation

10. **Add sidebar link** (`assets/views/default/layouts/default.html`)
    - Add Activities nav item after Sprints

11. **Create activities page** (`assets/views/default/pages/activities.html`)
    - Compact activity list view
    - Load more/pagination

12. **Update issue detail page** (`assets/views/default/pages/issue.html`)
    - Merge activities with comments
    - Unified timeline display

13. **Add page handler** (`app/web/handler/page.go`)
    - Add Activities handler method
    - Add ActivitiesData struct

## Activity Message Templates

For UI display, activities are formatted with these templates:

| Action | Template |
|--------|----------|
| `issue_created` | "{actor} created the issue" |
| `issue_updated` | "{actor} updated the {field}" |
| `status_changed` | "{actor} changed status from {old} to {new}" |
| `priority_changed` | "{actor} changed priority from {old} to {new}" |
| `assignee_added` | "{actor} assigned {assignee}" |
| `assignee_removed` | "{actor} unassigned {assignee}" |
| `cycle_attached` | "{actor} added to {cycle}" |
| `cycle_detached` | "{actor} removed from {cycle}" |
| `start_date_changed` | "{actor} set start date to {date}" |
| `due_date_changed` | "{actor} set due date to {date}" |
| `comment_added` | "{actor} commented" |

## Testing

1. **Store tests** - Verify CRUD operations work correctly
2. **Service tests** - Verify activity creation and listing
3. **Integration tests** - Verify activities are logged on issue operations
4. **E2E tests** - Verify UI displays activities correctly

## Future Enhancements

- Activity filtering by action type
- Activity search
- Email/webhook notifications for activities
- Activity export (CSV, JSON)
- Aggregate activity stats (e.g., "5 activities in the last hour")
