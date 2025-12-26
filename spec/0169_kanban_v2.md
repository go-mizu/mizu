# 0169 Kanban V2 - Complete Refactoring Specification

## Overview

This specification documents the complete refactoring of the Kanban blueprint to align with the new minimal schema design. The key changes include:

1. **Teams** as the primary organizing unit (Linear-like)
2. **Columns** for Kanban board state (instead of status field)
3. **Cycles** for planning periods (team-scoped, replacing project-scoped sprints)
4. **Fields + Values** for extensible custom attributes (monday/GitHub style)
5. Simplified data model with minimal required columns

## Schema Design Principles

- Minimal required columns only
- Everything optional becomes a Field + Value
- Typed values are AI/analytics-friendly
- No secondary indexes (DuckDB columnar scans + zone maps)
- One consistent vocabulary across UI/API/service/store/DB

## Core Mental Model

```
Workspace → Team → Project (Board) → Columns → Cards (Issues)
                 → Cycles → Cards (optional attachment)

Cards can have Fields, and each card stores Values for those fields.
```

---

## Database Schema

### Core Tables

#### users
```sql
CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR PRIMARY KEY,
    email         VARCHAR UNIQUE NOT NULL,
    username      VARCHAR UNIQUE NOT NULL,
    display_name  VARCHAR NOT NULL,
    password_hash VARCHAR NOT NULL
);
```

#### sessions
```sql
CREATE TABLE IF NOT EXISTS sessions (
    id         VARCHAR PRIMARY KEY,
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### workspaces
```sql
CREATE TABLE IF NOT EXISTS workspaces (
    id   VARCHAR PRIMARY KEY,
    slug VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL
);
```

#### workspace_members
```sql
CREATE TABLE IF NOT EXISTS workspace_members (
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         VARCHAR NOT NULL DEFAULT 'member', -- owner, admin, member, guest
    joined_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, user_id)
);
```

#### teams
```sql
CREATE TABLE IF NOT EXISTS teams (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key          VARCHAR NOT NULL, -- short code like "ENG"
    name         VARCHAR NOT NULL,
    UNIQUE (workspace_id, key),
    UNIQUE (workspace_id, name)
);
```

#### team_members
```sql
CREATE TABLE IF NOT EXISTS team_members (
    team_id   VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id   VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      VARCHAR NOT NULL DEFAULT 'member', -- lead, member
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id)
);
```

#### projects
```sql
CREATE TABLE IF NOT EXISTS projects (
    id            VARCHAR PRIMARY KEY,
    team_id       VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    key           VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,
    issue_counter INTEGER NOT NULL DEFAULT 0,
    UNIQUE (team_id, key)
);
```

#### columns
```sql
CREATE TABLE IF NOT EXISTS columns (
    id          VARCHAR PRIMARY KEY,
    project_id  VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        VARCHAR NOT NULL,
    position    INTEGER NOT NULL DEFAULT 0,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (project_id, name)
);
```

#### cycles
```sql
CREATE TABLE IF NOT EXISTS cycles (
    id         VARCHAR PRIMARY KEY,
    team_id    VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    number     INTEGER NOT NULL,
    name       VARCHAR NOT NULL,
    status     VARCHAR NOT NULL DEFAULT 'planning', -- planning, active, completed
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (team_id, number)
);
```

#### issues
```sql
CREATE TABLE IF NOT EXISTS issues (
    id         VARCHAR PRIMARY KEY,
    project_id VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number     INTEGER NOT NULL,
    key        VARCHAR NOT NULL,
    title      VARCHAR NOT NULL,
    column_id  VARCHAR NOT NULL REFERENCES columns(id) ON DELETE RESTRICT,
    position   INTEGER NOT NULL DEFAULT 0,
    creator_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cycle_id   VARCHAR REFERENCES cycles(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, number),
    UNIQUE (project_id, key)
);
```

#### issue_assignees
```sql
CREATE TABLE IF NOT EXISTS issue_assignees (
    issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id  VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, user_id)
);
```

#### comments
```sql
CREATE TABLE IF NOT EXISTS comments (
    id         VARCHAR PRIMARY KEY,
    issue_id   VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    author_id  VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    VARCHAR NOT NULL,
    edited_at  TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### fields
```sql
CREATE TABLE IF NOT EXISTS fields (
    id            VARCHAR PRIMARY KEY,
    project_id    VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key           VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,
    kind          VARCHAR NOT NULL, -- text, number, bool, date, ts, select, user, json
    position      INTEGER NOT NULL DEFAULT 0,
    is_required   BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived   BOOLEAN NOT NULL DEFAULT FALSE,
    settings_json VARCHAR,
    UNIQUE (project_id, key),
    UNIQUE (project_id, name)
);
```

#### values
```sql
CREATE TABLE IF NOT EXISTS values (
    issue_id   VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    field_id   VARCHAR NOT NULL REFERENCES fields(id) ON DELETE CASCADE,
    value_text VARCHAR,
    value_num  DOUBLE,
    value_bool BOOLEAN,
    value_date DATE,
    value_ts   TIMESTAMP,
    value_ref  VARCHAR,
    value_json VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (issue_id, field_id)
);
```

---

## Store Layer (store/duckdb/*)

### File Structure

| File | Description |
|------|-------------|
| `store.go` | DuckDB handle, schema migration, transaction helpers |
| `users_store.go` | Users and sessions CRUD |
| `workspaces_store.go` | Workspace CRUD; membership CRUD and role management |
| `teams_store.go` | Team CRUD; team membership CRUD |
| `projects_store.go` | Project CRUD within a team; issue_counter allocation |
| `columns_store.go` | Column CRUD; reorder; set default; archive/unarchive |
| `cycles_store.go` | Cycle CRUD per team; status transitions |
| `issues_store.go` | Issue CRUD; move between columns; reorder; cycle attachment |
| `assignees_store.go` | Issue assignees add/remove/list |
| `comments_store.go` | Comment CRUD; list by issue |
| `fields_store.go` | Field definitions per project |
| `values_store.go` | Typed field values per issue |

### Store Interfaces

#### UsersStore
```go
type UsersStore interface {
    Create(ctx context.Context, u *users.User) error
    GetByID(ctx context.Context, id string) (*users.User, error)
    GetByEmail(ctx context.Context, email string) (*users.User, error)
    GetByUsername(ctx context.Context, username string) (*users.User, error)
    Update(ctx context.Context, id string, in *users.UpdateIn) error
    UpdatePassword(ctx context.Context, id string, passwordHash string) error
    CreateSession(ctx context.Context, sess *users.Session) error
    GetSession(ctx context.Context, id string) (*users.Session, error)
    DeleteSession(ctx context.Context, id string) error
    DeleteExpiredSessions(ctx context.Context) error
}
```

#### WorkspacesStore
```go
type WorkspacesStore interface {
    Create(ctx context.Context, w *workspaces.Workspace) error
    GetByID(ctx context.Context, id string) (*workspaces.Workspace, error)
    GetBySlug(ctx context.Context, slug string) (*workspaces.Workspace, error)
    ListByUser(ctx context.Context, userID string) ([]*workspaces.Workspace, error)
    Update(ctx context.Context, id string, in *workspaces.UpdateIn) error
    Delete(ctx context.Context, id string) error
    AddMember(ctx context.Context, m *workspaces.Member) error
    GetMember(ctx context.Context, workspaceID, userID string) (*workspaces.Member, error)
    ListMembers(ctx context.Context, workspaceID string) ([]*workspaces.Member, error)
    UpdateMemberRole(ctx context.Context, workspaceID, userID, role string) error
    RemoveMember(ctx context.Context, workspaceID, userID string) error
}
```

#### TeamsStore
```go
type TeamsStore interface {
    Create(ctx context.Context, t *teams.Team) error
    GetByID(ctx context.Context, id string) (*teams.Team, error)
    GetByKey(ctx context.Context, workspaceID, key string) (*teams.Team, error)
    ListByWorkspace(ctx context.Context, workspaceID string) ([]*teams.Team, error)
    Update(ctx context.Context, id string, in *teams.UpdateIn) error
    Delete(ctx context.Context, id string) error
    AddMember(ctx context.Context, m *teams.Member) error
    GetMember(ctx context.Context, teamID, userID string) (*teams.Member, error)
    ListMembers(ctx context.Context, teamID string) ([]*teams.Member, error)
    UpdateMemberRole(ctx context.Context, teamID, userID, role string) error
    RemoveMember(ctx context.Context, teamID, userID string) error
}
```

#### ProjectsStore
```go
type ProjectsStore interface {
    Create(ctx context.Context, p *projects.Project) error
    GetByID(ctx context.Context, id string) (*projects.Project, error)
    GetByKey(ctx context.Context, teamID, key string) (*projects.Project, error)
    ListByTeam(ctx context.Context, teamID string) ([]*projects.Project, error)
    Update(ctx context.Context, id string, in *projects.UpdateIn) error
    Delete(ctx context.Context, id string) error
    IncrementIssueCounter(ctx context.Context, id string) (int, error)
}
```

#### ColumnsStore
```go
type ColumnsStore interface {
    Create(ctx context.Context, c *columns.Column) error
    GetByID(ctx context.Context, id string) (*columns.Column, error)
    ListByProject(ctx context.Context, projectID string) ([]*columns.Column, error)
    Update(ctx context.Context, id string, in *columns.UpdateIn) error
    UpdatePosition(ctx context.Context, id string, position int) error
    SetDefault(ctx context.Context, projectID, columnID string) error
    Archive(ctx context.Context, id string) error
    Unarchive(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error
    GetDefault(ctx context.Context, projectID string) (*columns.Column, error)
}
```

#### CyclesStore
```go
type CyclesStore interface {
    Create(ctx context.Context, c *cycles.Cycle) error
    GetByID(ctx context.Context, id string) (*cycles.Cycle, error)
    GetByNumber(ctx context.Context, teamID string, number int) (*cycles.Cycle, error)
    ListByTeam(ctx context.Context, teamID string) ([]*cycles.Cycle, error)
    GetActive(ctx context.Context, teamID string) (*cycles.Cycle, error)
    Update(ctx context.Context, id string, in *cycles.UpdateIn) error
    UpdateStatus(ctx context.Context, id, status string) error
    Delete(ctx context.Context, id string) error
    GetNextNumber(ctx context.Context, teamID string) (int, error)
}
```

#### IssuesStore
```go
type IssuesStore interface {
    Create(ctx context.Context, i *issues.Issue) error
    GetByID(ctx context.Context, id string) (*issues.Issue, error)
    GetByKey(ctx context.Context, key string) (*issues.Issue, error)
    ListByProject(ctx context.Context, projectID string) ([]*issues.Issue, error)
    ListByColumn(ctx context.Context, columnID string) ([]*issues.Issue, error)
    ListByCycle(ctx context.Context, cycleID string) ([]*issues.Issue, error)
    Update(ctx context.Context, id string, in *issues.UpdateIn) error
    Move(ctx context.Context, id, columnID string, position int) error
    AttachCycle(ctx context.Context, id, cycleID string) error
    DetachCycle(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, projectID, query string, limit int) ([]*issues.Issue, error)
}
```

#### AssigneesStore
```go
type AssigneesStore interface {
    Add(ctx context.Context, issueID, userID string) error
    Remove(ctx context.Context, issueID, userID string) error
    List(ctx context.Context, issueID string) ([]string, error)
    ListByUser(ctx context.Context, userID string) ([]string, error)
}
```

#### CommentsStore
```go
type CommentsStore interface {
    Create(ctx context.Context, c *comments.Comment) error
    GetByID(ctx context.Context, id string) (*comments.Comment, error)
    ListByIssue(ctx context.Context, issueID string) ([]*comments.Comment, error)
    Update(ctx context.Context, id, content string) error
    Delete(ctx context.Context, id string) error
    CountByIssue(ctx context.Context, issueID string) (int, error)
}
```

#### FieldsStore
```go
type FieldsStore interface {
    Create(ctx context.Context, f *fields.Field) error
    GetByID(ctx context.Context, id string) (*fields.Field, error)
    GetByKey(ctx context.Context, projectID, key string) (*fields.Field, error)
    ListByProject(ctx context.Context, projectID string) ([]*fields.Field, error)
    Update(ctx context.Context, id string, in *fields.UpdateIn) error
    UpdatePosition(ctx context.Context, id string, position int) error
    Archive(ctx context.Context, id string) error
    Unarchive(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error
}
```

#### ValuesStore
```go
type ValuesStore interface {
    Set(ctx context.Context, v *values.Value) error
    Get(ctx context.Context, issueID, fieldID string) (*values.Value, error)
    ListByIssue(ctx context.Context, issueID string) ([]*values.Value, error)
    ListByField(ctx context.Context, fieldID string) ([]*values.Value, error)
    Delete(ctx context.Context, issueID, fieldID string) error
    DeleteByIssue(ctx context.Context, issueID string) error
    BulkSet(ctx context.Context, vs []*values.Value) error
    BulkGetByIssues(ctx context.Context, issueIDs []string) (map[string][]*values.Value, error)
}
```

---

## Feature Layer (feature/*)

### Data Models

#### users
```go
type User struct {
    ID           string    `json:"id"`
    Email        string    `json:"email"`
    Username     string    `json:"username"`
    DisplayName  string    `json:"display_name"`
    PasswordHash string    `json:"-"`
}

type Session struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    ExpiresAt time.Time `json:"expires_at"`
    CreatedAt time.Time `json:"created_at"`
}
```

#### workspaces
```go
type Workspace struct {
    ID   string `json:"id"`
    Slug string `json:"slug"`
    Name string `json:"name"`
}

type Member struct {
    WorkspaceID string    `json:"workspace_id"`
    UserID      string    `json:"user_id"`
    Role        string    `json:"role"` // owner, admin, member, guest
    JoinedAt    time.Time `json:"joined_at"`
}
```

#### teams
```go
type Team struct {
    ID          string `json:"id"`
    WorkspaceID string `json:"workspace_id"`
    Key         string `json:"key"`
    Name        string `json:"name"`
}

type Member struct {
    TeamID   string    `json:"team_id"`
    UserID   string    `json:"user_id"`
    Role     string    `json:"role"` // lead, member
    JoinedAt time.Time `json:"joined_at"`
}
```

#### projects
```go
type Project struct {
    ID           string `json:"id"`
    TeamID       string `json:"team_id"`
    Key          string `json:"key"`
    Name         string `json:"name"`
    IssueCounter int    `json:"issue_counter"`
}
```

#### columns
```go
type Column struct {
    ID         string `json:"id"`
    ProjectID  string `json:"project_id"`
    Name       string `json:"name"`
    Position   int    `json:"position"`
    IsDefault  bool   `json:"is_default"`
    IsArchived bool   `json:"is_archived"`
}
```

#### cycles
```go
type Cycle struct {
    ID        string    `json:"id"`
    TeamID    string    `json:"team_id"`
    Number    int       `json:"number"`
    Name      string    `json:"name"`
    Status    string    `json:"status"` // planning, active, completed
    StartDate time.Time `json:"start_date"`
    EndDate   time.Time `json:"end_date"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

#### issues
```go
type Issue struct {
    ID        string    `json:"id"`
    ProjectID string    `json:"project_id"`
    Number    int       `json:"number"`
    Key       string    `json:"key"`
    Title     string    `json:"title"`
    ColumnID  string    `json:"column_id"`
    Position  int       `json:"position"`
    CreatorID string    `json:"creator_id"`
    CycleID   string    `json:"cycle_id,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

#### assignees
```go
// Simple many-to-many relation, no separate struct needed
// Operations: Add(issueID, userID), Remove(issueID, userID), List(issueID)
```

#### comments
```go
type Comment struct {
    ID        string     `json:"id"`
    IssueID   string     `json:"issue_id"`
    AuthorID  string     `json:"author_id"`
    Content   string     `json:"content"`
    EditedAt  *time.Time `json:"edited_at,omitempty"`
    CreatedAt time.Time  `json:"created_at"`
}
```

#### fields
```go
type Field struct {
    ID           string `json:"id"`
    ProjectID    string `json:"project_id"`
    Key          string `json:"key"`
    Name         string `json:"name"`
    Kind         string `json:"kind"` // text, number, bool, date, ts, select, user, json
    Position     int    `json:"position"`
    IsRequired   bool   `json:"is_required"`
    IsArchived   bool   `json:"is_archived"`
    SettingsJSON string `json:"settings_json,omitempty"`
}

// Kind constants
const (
    KindText   = "text"
    KindNumber = "number"
    KindBool   = "bool"
    KindDate   = "date"
    KindTS     = "ts"
    KindSelect = "select"
    KindUser   = "user"
    KindJSON   = "json"
)
```

#### values
```go
type Value struct {
    IssueID   string     `json:"issue_id"`
    FieldID   string     `json:"field_id"`
    ValueText *string    `json:"value_text,omitempty"`
    ValueNum  *float64   `json:"value_num,omitempty"`
    ValueBool *bool      `json:"value_bool,omitempty"`
    ValueDate *time.Time `json:"value_date,omitempty"`
    ValueTS   *time.Time `json:"value_ts,omitempty"`
    ValueRef  *string    `json:"value_ref,omitempty"`
    ValueJSON *string    `json:"value_json,omitempty"`
    UpdatedAt time.Time  `json:"updated_at"`
}
```

---

## Migration from V1

### Removed Tables/Features
- `labels` table (use fields with kind=select or kind=json)
- `sprints` table (replaced by team-scoped cycles)
- `activities` table (can be re-added as needed)
- `notifications` table (can be re-added as needed)
- `issue_labels` junction table
- `issue_links` table

### Changed Tables
- `projects`: now belongs to team (team_id) instead of workspace (workspace_id)
- `issues`: column_id instead of status field, cycle_id instead of sprint_id
- `workspace_members`: removed id field, uses (workspace_id, user_id) as primary key

### New Tables
- `teams`
- `team_members`
- `columns`
- `cycles`
- `fields`
- `values`

### Data Migration Notes
- Issue status values should be migrated to columns
- Sprint data should be migrated to cycles (may require team assignment)
- Labels should be migrated to fields with kind=select or kind=json
- Issue description, priority, type, estimate, due_date should become field values

---

## Files to Delete

### store/duckdb/
- `labels_store.go`
- `sprints_store.go`
- `notifications_store.go`

### feature/
- `labels/` (entire directory)
- `sprints/` (entire directory)
- `notifications/` (entire directory)
- `search/` (entire directory - search can be added back later)

---

## Files to Create

### store/duckdb/
- `teams_store.go`
- `columns_store.go`
- `cycles_store.go`
- `assignees_store.go`
- `fields_store.go`
- `values_store.go`

### feature/
- `teams/api.go`
- `teams/service.go`
- `columns/api.go`
- `columns/service.go`
- `cycles/api.go`
- `cycles/service.go`
- `assignees/api.go`
- `assignees/service.go`
- `fields/api.go`
- `fields/service.go`
- `values/api.go`
- `values/service.go`

---

## Files to Modify

### store/duckdb/
- `schema.sql` - Complete replacement with new schema
- `store.go` - Update Stats() to use new table names
- `users_store.go` - Remove avatar_url, created_at, updated_at from schema
- `workspaces_store.go` - Remove description, avatar_url, created_at, updated_at; update member primary key
- `projects_store.go` - Change workspace_id to team_id; remove extra fields
- `issues_store.go` - Complete rewrite for new schema
- `comments_store.go` - Minor cleanup (already mostly compatible)

### feature/
- `users/api.go` - Simplify User struct
- `workspaces/api.go` - Simplify Workspace and Member structs
- `projects/api.go` - Change to team_id; remove extra fields
- `issues/api.go` - Complete rewrite for new schema
- `comments/api.go` - Already mostly compatible

---

## Implementation Order

1. Update `schema.sql` with new schema
2. Update feature data models (`api.go` files)
3. Update existing stores
4. Create new stores
5. Update services
6. Write comprehensive tests
7. Update web handlers (if in scope)
