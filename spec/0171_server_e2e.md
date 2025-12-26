# 0171 - Kanban Blueprint Server E2E Test Specification

## Overview

This specification defines the end-to-end test requirements for the Kanban Blueprint API server. Tests validate all business cases through HTTP API endpoints using `httptest`.

## Test Infrastructure

### Test Setup Pattern

```go
func testServer(t *testing.T) (*testEnv, func()) {
    t.Helper()

    db, err := sql.Open("duckdb", ":memory:")
    // ... setup stores and services

    return env, cleanup
}

type testEnv struct {
    t          *testing.T
    db         *sql.DB
    users      users.API
    workspaces workspaces.API
    teams      teams.API
    projects   projects.API
    columns    columns.API
    issues     issues.API
    cycles     cycles.API
    comments   comments.API
    fields     fields.API
    values     values.API
    assignees  assignees.API
}
```

### API Response Format

```go
type apiResponse struct {
    Success bool            `json:"success"`
    Data    json.RawMessage `json:"data,omitempty"`
    Error   string          `json:"error,omitempty"`
}
```

---

## Test Groups

### 1. Server Lifecycle Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestServer_New` | Create server with temp dir config | Server created, app/db not nil |
| `TestServer_Handler` | Get HTTP handler | Handler not nil |
| `TestServer_Close` | Graceful shutdown | No error, DB closed |

---

### 2. Authentication Tests (Auth)

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestAuth_Register_Success` | Register with valid data | 201, user + session returned |
| `TestAuth_Register_MissingUsername` | Register without username | 400 |
| `TestAuth_Register_MissingEmail` | Register without email | 400 |
| `TestAuth_Register_MissingPassword` | Register without password | 400 |
| `TestAuth_Register_PasswordTooShort` | Password < 8 chars | 400 |
| `TestAuth_Register_DuplicateUsername` | Username already exists | 400 |
| `TestAuth_Register_DuplicateEmail` | Email already exists | 400 |
| `TestAuth_Login_WithUsername` | Login with username | 200, session token |
| `TestAuth_Login_WithEmail` | Login with email | 200, session token |
| `TestAuth_Login_InvalidPassword` | Wrong password | 401 |
| `TestAuth_Login_NonexistentUser` | User doesn't exist | 401 |
| `TestAuth_Me_WithBearerToken` | Get current user | 200, user data |
| `TestAuth_Me_WithSessionCookie` | Get current user via cookie | 200, user data |
| `TestAuth_Me_Unauthorized` | No auth header | 401 |
| `TestAuth_UpdateMe` | Update display name/status | 200, updated user |
| `TestAuth_Logout` | Logout and clear session | 200, cookie cleared |

---

### 3. Workspace Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestWorkspace_Create` | Create workspace | 201, workspace returned |
| `TestWorkspace_Create_DuplicateSlug` | Duplicate slug | 400 |
| `TestWorkspace_List` | List user's workspaces | 200, array |
| `TestWorkspace_Get` | Get by slug | 200, workspace |
| `TestWorkspace_Get_NotFound` | Non-existent slug | 404 |
| `TestWorkspace_Update` | Update name | 200, updated |
| `TestWorkspace_Delete` | Delete workspace | 200 |
| `TestWorkspace_ListMembers` | List workspace members | 200, array |
| `TestWorkspace_AddMember` | Add user to workspace | 201 |
| `TestWorkspace_RemoveMember` | Remove user | 200 |
| `TestWorkspace_Unauthorized` | Access without auth | 401 |

---

### 4. Team Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestTeam_Create` | Create team in workspace | 201, team returned |
| `TestTeam_Create_DuplicateKey` | Duplicate key in workspace | 400 |
| `TestTeam_List` | List teams in workspace | 200, array |
| `TestTeam_Get` | Get by ID | 200, team |
| `TestTeam_Get_NotFound` | Non-existent ID | 404 |
| `TestTeam_Update` | Update name | 200, updated |
| `TestTeam_Delete` | Delete team | 200 |
| `TestTeam_ListMembers` | List team members | 200, array |
| `TestTeam_AddMember` | Add user to team | 201 |
| `TestTeam_AddMember_AlreadyMember` | Add existing member | 400 |
| `TestTeam_UpdateMemberRole` | Change role (member → lead) | 200 |
| `TestTeam_RemoveMember` | Remove user | 200 |

---

### 5. Project Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestProject_Create` | Create project in team | 201, project returned |
| `TestProject_Create_DuplicateKey` | Duplicate key in team | 400 |
| `TestProject_Create_KeyUppercase` | Key auto-uppercased | key = "ABC" |
| `TestProject_List` | List projects in team | 200, array |
| `TestProject_Get` | Get by key | 200, project |
| `TestProject_Get_NotFound` | Non-existent key | 404 |
| `TestProject_Update` | Update name | 200, updated |
| `TestProject_Delete` | Delete project | 200 |

---

### 6. Column Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestColumn_Create` | Create column | 201, column returned |
| `TestColumn_Create_FirstIsDefault` | First column is default | is_default = true |
| `TestColumn_List` | List columns by project | 200, ordered array |
| `TestColumn_Update` | Update name | 200, updated |
| `TestColumn_UpdatePosition` | Reorder column | 200 |
| `TestColumn_SetDefault` | Change default column | 200, old unset |
| `TestColumn_Archive` | Archive column | 200, is_archived = true |
| `TestColumn_Unarchive` | Unarchive column | 200, is_archived = false |
| `TestColumn_Delete` | Delete column | 200 |
| `TestColumn_Delete_WithIssues` | Delete column with issues | Error or cascade |

---

### 7. Issue Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestIssue_Create` | Create issue | 201, issue + key |
| `TestIssue_Create_KeyGeneration` | Key = PROJECT-N | key = "ABC-1" |
| `TestIssue_Create_AutoColumn` | Issue in default column | column_id set |
| `TestIssue_List_ByProject` | List by project | 200, array |
| `TestIssue_List_ByColumn` | List by column | 200, array |
| `TestIssue_Get_ByKey` | Get by key | 200, issue |
| `TestIssue_Get_NotFound` | Non-existent key | 404 |
| `TestIssue_Update` | Update title/description | 200, updated |
| `TestIssue_Move_ToColumn` | Move to different column | 200, column_id changed |
| `TestIssue_Move_Position` | Reorder within column | 200, position changed |
| `TestIssue_AttachCycle` | Attach to cycle | 200, cycle_id set |
| `TestIssue_DetachCycle` | Remove from cycle | 200, cycle_id null |
| `TestIssue_Delete` | Delete issue | 200 |
| `TestIssue_Search` | Search by title/key | 200, matching issues |

---

### 8. Cycle Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestCycle_Create` | Create cycle | 201, cycle + number |
| `TestCycle_Create_NumberAutoIncrement` | Number auto-assigned | number = 1, 2, 3... |
| `TestCycle_List` | List by team | 200, array |
| `TestCycle_Get_ByID` | Get by ID | 200, cycle |
| `TestCycle_Get_ByNumber` | Get by team + number | 200, cycle |
| `TestCycle_Get_NotFound` | Non-existent | 404 |
| `TestCycle_Update` | Update name/dates | 200, updated |
| `TestCycle_UpdateStatus_Planning` | Status = planning | 200 |
| `TestCycle_UpdateStatus_Active` | Status = active | 200 |
| `TestCycle_UpdateStatus_Completed` | Status = completed | 200 |
| `TestCycle_GetActive` | Get active cycle for team | 200 or 404 |
| `TestCycle_Delete` | Delete cycle | 200 |
| `TestCycle_Delete_WithIssues` | Delete cycle with attached issues | Issues detached |

---

### 9. Comment Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestComment_Create` | Create comment on issue | 201, comment |
| `TestComment_Create_Empty` | Empty content | 400 |
| `TestComment_List` | List by issue | 200, ordered array |
| `TestComment_Update` | Update content | 200, edited_at set |
| `TestComment_Delete` | Delete comment | 200 |
| `TestComment_Delete_NotOwner` | Delete other's comment | 403 |

---

### 10. Assignee Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestAssignee_Add` | Assign user to issue | 201 |
| `TestAssignee_Add_AlreadyAssigned` | Assign twice | 400 or 200 |
| `TestAssignee_List` | List assignees | 200, user IDs |
| `TestAssignee_Remove` | Unassign user | 200 |
| `TestAssignee_Remove_NotAssigned` | Remove non-assignee | 404 or 200 |
| `TestAssignee_ListByUser` | List issues assigned to user | 200, issue IDs |

---

### 11. Field Tests (Custom Fields)

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestField_Create` | Create custom field | 201, field |
| `TestField_Create_DuplicateKey` | Duplicate key in project | 400 |
| `TestField_List` | List by project | 200, array |
| `TestField_Get_ByID` | Get by ID | 200, field |
| `TestField_Get_ByKey` | Get by project + key | 200, field |
| `TestField_Update` | Update name/required | 200, updated |
| `TestField_UpdatePosition` | Reorder field | 200 |
| `TestField_Archive` | Archive field | 200, is_archived = true |
| `TestField_Unarchive` | Unarchive field | 200, is_archived = false |
| `TestField_Delete` | Delete field | 200 |

---

### 12. Value Tests (Field Values)

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestValue_Set_Text` | Set text value | 200, value |
| `TestValue_Set_Number` | Set number value | 200, value |
| `TestValue_Set_Bool` | Set boolean value | 200, value |
| `TestValue_Set_Date` | Set date value | 200, value |
| `TestValue_Set_Timestamp` | Set timestamp value | 200, value |
| `TestValue_Set_Ref` | Set reference value | 200, value |
| `TestValue_Set_JSON` | Set JSON value | 200, value |
| `TestValue_Get` | Get value by issue + field | 200, value |
| `TestValue_Get_NotSet` | Get unset value | 404 or null |
| `TestValue_ListByIssue` | List all values for issue | 200, array |
| `TestValue_Delete` | Delete value | 200 |
| `TestValue_BulkSet` | Set multiple values at once | 200 |
| `TestValue_BulkGet` | Get values for multiple issues | 200, map |

---

### 13. Page Tests (HTML)

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestPage_Home` | GET / | 200, HTML |
| `TestPage_Login` | GET /login | 200, HTML |
| `TestPage_Register` | GET /register | 200, HTML |
| `TestPage_Workspace` | GET /{workspace} | 200 or redirect |
| `TestPage_Projects` | GET /{workspace}/projects | 200 |
| `TestPage_Board` | GET /{workspace}/projects/{key} | 200 |
| `TestPage_Issue` | GET /{workspace}/issue/{key} | 200 |
| `TestPage_Settings` | GET /settings | 200 |

---

### 14. Static File Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestStatic_CSS` | GET /static/css/app.css | 200, CSS mime |
| `TestStatic_JS` | GET /static/js/app.js | 200, JS mime |
| `TestStatic_NotFound` | GET /static/nonexistent | 404 |

---

### 15. Security Tests

| Test Name | Description | Expected |
|-----------|-------------|----------|
| `TestSecurity_AllAPIRequireAuth` | All /api/v1/* except register/login | 401 without auth |
| `TestSecurity_CORS` | CORS headers present | Headers set |
| `TestSecurity_SessionExpiry` | Expired session rejected | 401 |
| `TestSecurity_InvalidToken` | Malformed token | 401 |

---

## Business Logic Test Cases

### Workspace → Team → Project Hierarchy

1. User creates workspace, becomes owner
2. Owner creates team within workspace
3. Team lead creates project within team
4. Project has default column created automatically
5. Issues created in default column

### Issue Lifecycle

1. Create issue → key generated (PROJECT-1)
2. Issue placed in default column
3. Move issue between columns (Kanban flow)
4. Attach issue to cycle (sprint planning)
5. Add assignees (multiple)
6. Set custom field values
7. Add comments
8. Complete issue (move to done column)
9. Detach from cycle (cycle completed)

### Cycle Lifecycle

1. Create cycle (planning status)
2. Attach issues to cycle
3. Activate cycle (status = active)
4. Only one active cycle per team
5. Complete cycle (status = completed)
6. Issues remain attached for history

### Custom Fields

1. Create field definitions per project
2. Field types: text, number, bool, date, ts, select, user, json
3. Set values on issues
4. Bulk operations for efficiency
5. Archive fields (soft delete)

---

## Test Data Factory Functions

```go
func (e *testEnv) createTestUser(username, email, password string) (*users.User, string)
func (e *testEnv) createTestWorkspace(userID, slug, name string) *workspaces.Workspace
func (e *testEnv) createTestTeam(workspaceID, key, name string) *teams.Team
func (e *testEnv) createTestProject(teamID, key, name string) *projects.Project
func (e *testEnv) createTestColumn(projectID, name string, position int) *columns.Column
func (e *testEnv) createTestIssue(projectID, creatorID, title string) *issues.Issue
func (e *testEnv) createTestCycle(teamID, name string) *cycles.Cycle
func (e *testEnv) createTestComment(issueID, authorID, content string) *comments.Comment
func (e *testEnv) createTestField(projectID, key, name, kind string) *fields.Field
```

---

## Implementation Notes

1. All tests use in-memory DuckDB (`:memory:`)
2. Each test gets fresh database via `t.TempDir()`
3. Tests are independent and can run in parallel
4. Use `httptest.NewRecorder` for HTTP testing
5. Bearer token OR session cookie for authentication
6. JSON responses follow standard format: `{success, data, error}`
7. HTTP status codes follow REST conventions:
   - 200: Success
   - 201: Created
   - 400: Bad Request
   - 401: Unauthorized
   - 403: Forbidden
   - 404: Not Found
   - 500: Internal Server Error
