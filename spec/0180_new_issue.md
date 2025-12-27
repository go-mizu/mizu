# 0180: Unified Create Issue Form

## Status
**Implemented**

## Summary
This spec describes the unified "Create Issue" form design that consolidates the previously separate implementations in Inbox and Issues pages into a single, consistent component.

## Problem Statement

Previously, there were two different "Create Issue" implementations:

1. **Inbox Page** (`inbox.html`): A rich inline form with property chips for status, project, assignee, and cycle
2. **Issues Page** (`issues.html`): A simpler modal with just title, description, and project selector

This caused:
- Inconsistent user experience across pages
- Duplicate code and maintenance burden
- The "undefined" project ID bug when `DefaultProjectID` was empty
- Missing description field in the Issue model causing "invalid request body" errors

## Solution

### 1. Database Schema Changes

Added `description` column to the issues table:

```sql
CREATE TABLE IF NOT EXISTS issues (
    id          VARCHAR PRIMARY KEY,
    project_id  VARCHAR NOT NULL REFERENCES projects(id),
    number      INTEGER NOT NULL,
    key         VARCHAR NOT NULL,
    title       VARCHAR NOT NULL,
    description VARCHAR DEFAULT '',  -- NEW FIELD
    column_id   VARCHAR NOT NULL REFERENCES columns(id),
    -- ... rest of columns
);
```

### 2. API Changes

#### CreateIn Struct
```go
type CreateIn struct {
    Title       string `json:"title"`
    Description string `json:"description,omitempty"`  // NEW FIELD
    ColumnID    string `json:"column_id,omitempty"`
    CycleID     string `json:"cycle_id,omitempty"`
}
```

#### UpdateIn Struct
```go
type UpdateIn struct {
    Title       *string    `json:"title,omitempty"`
    Description *string    `json:"description,omitempty"`  // NEW FIELD
    CycleID     *string    `json:"cycle_id,omitempty"`
    DueDate     *time.Time `json:"due_date,omitempty"`
    StartDate   *time.Time `json:"start_date,omitempty"`
    EndDate     *time.Time `json:"end_date,omitempty"`
}
```

#### Validation
- Title is now **required** - empty titles return a 500 error with "title is required" message

### 3. Unified Modal Design

The new unified modal includes:

```html
<div id="create-issue-modal" class="modal hidden">
  <div class="modal-content modal-lg">
    <form id="create-issue-form">
      <!-- Title (required) -->
      <input type="text" name="title" required>

      <!-- Description (optional) -->
      <textarea name="description"></textarea>

      <!-- Project selector (required) -->
      <select name="project_id" required>
        <option value="">Select project...</option>
        <!-- Populated from server -->
      </select>

      <!-- Status/Column selector (optional) -->
      <select name="column_id">
        <option value="">Default (Backlog)</option>
        <!-- Populated from server -->
      </select>

      <!-- Create another checkbox -->
      <label class="create-more-toggle">
        <input type="checkbox" name="create_more">
        <span>Create another</span>
      </label>
    </form>
  </div>
</div>
```

### 4. JavaScript Validation

Before submitting:
1. Validate project ID is selected (not empty or undefined)
2. Validate title is not empty
3. Disable submit button during API call
4. Show loading state

```javascript
// Validate project ID
if (!projectId) {
  alert('Please select a project');
  return;
}

// Validate title
if (!data.title) {
  alert('Please enter a title');
  form.title.focus();
  return;
}
```

### 5. "Create Another" Feature

When "Create another" is checked:
- Form is reset after successful creation
- Toast notification shows created issue key
- Focus returns to title input
- Modal stays open for next issue

### 6. CSS Additions

```css
.modal-content.modal-lg {
  max-width: 640px;
}

.form-row {
  display: flex;
  gap: 1rem;
}

.form-group-half {
  flex: 1;
}

.text-error {
  color: hsl(var(--destructive));
}

.create-more-toggle {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: hsl(var(--muted-foreground));
  cursor: pointer;
}

.toast-notification {
  /* Toast styles for success messages */
}
```

## API Endpoints

### Create Issue
```
POST /api/v1/projects/{projectId}/issues
Content-Type: application/json

{
  "title": "Issue title",           // Required
  "description": "Details...",      // Optional
  "column_id": "col-uuid",          // Optional, defaults to project default column
  "cycle_id": "cycle-uuid"          // Optional
}
```

### Response
```json
{
  "success": true,
  "data": {
    "id": "issue-uuid",
    "project_id": "project-uuid",
    "number": 1,
    "key": "PROJ-1",
    "title": "Issue title",
    "description": "Details...",
    "column_id": "column-uuid",
    "position": 0,
    "creator_id": "user-uuid",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

### Error Responses

#### Missing Title
```json
{
  "success": false,
  "error": "title is required"
}
```

#### Invalid Project
```json
{
  "success": false,
  "error": "project not found"
}
```

## Test Coverage

Added comprehensive e2e tests:

1. `TestIssue_Create` - Basic creation
2. `TestIssue_CreateWithDescription` - Creation with description
3. `TestIssue_CreateWithColumnID` - Creation with specific column
4. `TestIssue_CreateMissingTitle` - Validation error
5. `TestIssue_CreateInvalidProjectID` - Invalid project handling
6. `TestIssue_Update` - Update with description
7. `TestIssue_Move` - Moving between columns
8. `TestIssue_Delete` - Deletion

## Seed Data

Updated seed data to:
1. Include descriptions in all seeded issues
2. Add a second project ("Documentation") for testing project selector
3. Create proper column structure for both projects

## Migration Notes

For existing installations:
1. Run schema migration to add `description` column
2. Existing issues will have empty descriptions (default value)
3. No data migration required

## Files Changed

### Backend
- `store/duckdb/schema.sql` - Added description column
- `feature/issues/api.go` - Added Description to Issue, CreateIn, UpdateIn
- `feature/issues/service.go` - Pass description to store, add title validation
- `store/duckdb/issues_store.go` - Update all queries to include description
- `cli/seed.go` - Add descriptions to seed data, add second project

### Frontend
- `assets/views/default/pages/issues.html` - Unified modal design
- `assets/views/default/pages/inbox.html` - Updated validation
- `assets/static/css/default.css` - Added new styles

### Tests
- `app/web/server_e2e_test.go` - Added issue creation tests
