# 0170 Store Tests Specification

## Overview

This document specifies the comprehensive E2E test cases for all DuckDB stores in the Kanban blueprint. All tests run against a real DuckDB v2 in-memory database to ensure full compatibility.

## Test Infrastructure

### Test Setup Pattern

```go
func setupTestStore(t *testing.T) (*Store, func()) {
    t.Helper()
    db, err := sql.Open("duckdb", "")
    require.NoError(t, err)

    store, err := New(db)
    require.NoError(t, err)

    err = store.Ensure(context.Background())
    require.NoError(t, err)

    return store, func() {
        db.Close()
    }
}
```

### Test Naming Convention

- `Test<Store>_<Method>` - Basic method test
- `Test<Store>_<Method>_<Scenario>` - Specific scenario test

---

## Store Test Cases

### users_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestUsersStore_Create` | Create a new user |
| `TestUsersStore_Create_DuplicateEmail` | Fail on duplicate email |
| `TestUsersStore_Create_DuplicateUsername` | Fail on duplicate username |
| `TestUsersStore_GetByID` | Get user by ID |
| `TestUsersStore_GetByID_NotFound` | Return nil for non-existent user |
| `TestUsersStore_GetByEmail` | Get user by email |
| `TestUsersStore_GetByEmail_NotFound` | Return nil for non-existent email |
| `TestUsersStore_GetByUsername` | Get user by username |
| `TestUsersStore_GetByUsername_NotFound` | Return nil for non-existent username |
| `TestUsersStore_Update` | Update user display_name |
| `TestUsersStore_UpdatePassword` | Update user password hash |
| `TestUsersStore_CreateSession` | Create a new session |
| `TestUsersStore_GetSession` | Get session by ID |
| `TestUsersStore_GetSession_NotFound` | Return nil for non-existent session |
| `TestUsersStore_DeleteSession` | Delete a session |
| `TestUsersStore_DeleteExpiredSessions` | Delete all expired sessions |

### workspaces_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestWorkspacesStore_Create` | Create a new workspace |
| `TestWorkspacesStore_Create_DuplicateSlug` | Fail on duplicate slug |
| `TestWorkspacesStore_GetByID` | Get workspace by ID |
| `TestWorkspacesStore_GetByID_NotFound` | Return nil for non-existent workspace |
| `TestWorkspacesStore_GetBySlug` | Get workspace by slug |
| `TestWorkspacesStore_GetBySlug_NotFound` | Return nil for non-existent slug |
| `TestWorkspacesStore_ListByUser` | List workspaces for a user |
| `TestWorkspacesStore_ListByUser_Empty` | Return empty list for user with no workspaces |
| `TestWorkspacesStore_Update` | Update workspace name |
| `TestWorkspacesStore_Delete` | Delete a workspace |
| `TestWorkspacesStore_Delete_CascadeMembers` | Deleting workspace removes members |
| `TestWorkspacesStore_AddMember` | Add a member to workspace |
| `TestWorkspacesStore_AddMember_Duplicate` | Handle duplicate member add |
| `TestWorkspacesStore_GetMember` | Get workspace member |
| `TestWorkspacesStore_GetMember_NotFound` | Return nil for non-member |
| `TestWorkspacesStore_ListMembers` | List all workspace members |
| `TestWorkspacesStore_UpdateMemberRole` | Update member role |
| `TestWorkspacesStore_RemoveMember` | Remove a member |

### teams_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestTeamsStore_Create` | Create a new team |
| `TestTeamsStore_Create_DuplicateKey` | Fail on duplicate key in workspace |
| `TestTeamsStore_Create_DuplicateName` | Fail on duplicate name in workspace |
| `TestTeamsStore_GetByID` | Get team by ID |
| `TestTeamsStore_GetByID_NotFound` | Return nil for non-existent team |
| `TestTeamsStore_GetByKey` | Get team by workspace ID and key |
| `TestTeamsStore_GetByKey_NotFound` | Return nil for non-existent key |
| `TestTeamsStore_ListByWorkspace` | List teams in a workspace |
| `TestTeamsStore_ListByWorkspace_Empty` | Return empty list for workspace with no teams |
| `TestTeamsStore_Update` | Update team key and name |
| `TestTeamsStore_Delete` | Delete a team |
| `TestTeamsStore_Delete_CascadeProjects` | Deleting team removes projects |
| `TestTeamsStore_AddMember` | Add a member to team |
| `TestTeamsStore_AddMember_Duplicate` | Handle duplicate member add |
| `TestTeamsStore_GetMember` | Get team member |
| `TestTeamsStore_GetMember_NotFound` | Return nil for non-member |
| `TestTeamsStore_ListMembers` | List all team members |
| `TestTeamsStore_UpdateMemberRole` | Update member role |
| `TestTeamsStore_RemoveMember` | Remove a member |

### projects_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestProjectsStore_Create` | Create a new project |
| `TestProjectsStore_Create_DuplicateKey` | Fail on duplicate key in team |
| `TestProjectsStore_GetByID` | Get project by ID |
| `TestProjectsStore_GetByID_NotFound` | Return nil for non-existent project |
| `TestProjectsStore_GetByKey` | Get project by team ID and key |
| `TestProjectsStore_GetByKey_NotFound` | Return nil for non-existent key |
| `TestProjectsStore_ListByTeam` | List projects in a team |
| `TestProjectsStore_ListByTeam_Empty` | Return empty list for team with no projects |
| `TestProjectsStore_Update` | Update project name |
| `TestProjectsStore_Delete` | Delete a project |
| `TestProjectsStore_Delete_CascadeIssues` | Deleting project removes issues |
| `TestProjectsStore_IncrementIssueCounter` | Increment and return counter |
| `TestProjectsStore_IncrementIssueCounter_Multiple` | Multiple increments are sequential |

### columns_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestColumnsStore_Create` | Create a new column |
| `TestColumnsStore_Create_DuplicateName` | Fail on duplicate name in project |
| `TestColumnsStore_GetByID` | Get column by ID |
| `TestColumnsStore_GetByID_NotFound` | Return nil for non-existent column |
| `TestColumnsStore_ListByProject` | List columns ordered by position |
| `TestColumnsStore_ListByProject_Empty` | Return empty list for project with no columns |
| `TestColumnsStore_ListByProject_ExcludesArchived` | Archived columns excluded by default |
| `TestColumnsStore_Update` | Update column name |
| `TestColumnsStore_UpdatePosition` | Update column position |
| `TestColumnsStore_SetDefault` | Set column as default |
| `TestColumnsStore_SetDefault_ClearsOther` | Setting default clears previous default |
| `TestColumnsStore_Archive` | Archive a column |
| `TestColumnsStore_Unarchive` | Unarchive a column |
| `TestColumnsStore_Delete` | Delete a column |
| `TestColumnsStore_Delete_HasIssues` | Fail when column has issues |
| `TestColumnsStore_GetDefault` | Get default column for project |
| `TestColumnsStore_GetDefault_NotFound` | Return nil when no default |

### cycles_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestCyclesStore_Create` | Create a new cycle |
| `TestCyclesStore_Create_DuplicateNumber` | Fail on duplicate number in team |
| `TestCyclesStore_GetByID` | Get cycle by ID |
| `TestCyclesStore_GetByID_NotFound` | Return nil for non-existent cycle |
| `TestCyclesStore_GetByNumber` | Get cycle by team ID and number |
| `TestCyclesStore_GetByNumber_NotFound` | Return nil for non-existent number |
| `TestCyclesStore_ListByTeam` | List cycles in a team |
| `TestCyclesStore_ListByTeam_Empty` | Return empty list for team with no cycles |
| `TestCyclesStore_ListByTeam_OrderedByNumber` | Cycles ordered by number descending |
| `TestCyclesStore_GetActive` | Get active cycle for team |
| `TestCyclesStore_GetActive_NotFound` | Return nil when no active cycle |
| `TestCyclesStore_Update` | Update cycle name and dates |
| `TestCyclesStore_UpdateStatus` | Update cycle status |
| `TestCyclesStore_UpdateStatus_Planning` | Transition to planning |
| `TestCyclesStore_UpdateStatus_Active` | Transition to active |
| `TestCyclesStore_UpdateStatus_Completed` | Transition to completed |
| `TestCyclesStore_Delete` | Delete a cycle |
| `TestCyclesStore_Delete_DetachesIssues` | Deleting cycle nullifies issue.cycle_id |
| `TestCyclesStore_GetNextNumber` | Get next cycle number for team |
| `TestCyclesStore_GetNextNumber_FirstCycle` | Return 1 for first cycle |

### issues_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestIssuesStore_Create` | Create a new issue |
| `TestIssuesStore_Create_DuplicateNumber` | Fail on duplicate number in project |
| `TestIssuesStore_Create_DuplicateKey` | Fail on duplicate key in project |
| `TestIssuesStore_GetByID` | Get issue by ID |
| `TestIssuesStore_GetByID_NotFound` | Return nil for non-existent issue |
| `TestIssuesStore_GetByKey` | Get issue by key |
| `TestIssuesStore_GetByKey_NotFound` | Return nil for non-existent key |
| `TestIssuesStore_ListByProject` | List all issues in a project |
| `TestIssuesStore_ListByProject_Empty` | Return empty list for project with no issues |
| `TestIssuesStore_ListByProject_OrderedByPosition` | Issues ordered by position |
| `TestIssuesStore_ListByColumn` | List issues in a column |
| `TestIssuesStore_ListByColumn_Empty` | Return empty list for empty column |
| `TestIssuesStore_ListByCycle` | List issues in a cycle |
| `TestIssuesStore_ListByCycle_Empty` | Return empty list when no issues in cycle |
| `TestIssuesStore_Update` | Update issue title |
| `TestIssuesStore_Move` | Move issue to different column |
| `TestIssuesStore_Move_SameColumn` | Reorder within same column |
| `TestIssuesStore_AttachCycle` | Attach issue to a cycle |
| `TestIssuesStore_DetachCycle` | Detach issue from cycle |
| `TestIssuesStore_Delete` | Delete an issue |
| `TestIssuesStore_Delete_CascadeComments` | Deleting issue removes comments |
| `TestIssuesStore_Delete_CascadeAssignees` | Deleting issue removes assignees |
| `TestIssuesStore_Delete_CascadeValues` | Deleting issue removes field values |
| `TestIssuesStore_Search` | Search issues by title |
| `TestIssuesStore_Search_ByKey` | Search issues by key |
| `TestIssuesStore_Search_NoResults` | Return empty list when no matches |
| `TestIssuesStore_Search_Limit` | Respect search limit |

### assignees_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestAssigneesStore_Add` | Add assignee to issue |
| `TestAssigneesStore_Add_Duplicate` | Handle duplicate add (no error) |
| `TestAssigneesStore_Remove` | Remove assignee from issue |
| `TestAssigneesStore_Remove_NotAssigned` | Handle remove of non-assignee (no error) |
| `TestAssigneesStore_List` | List assignees for an issue |
| `TestAssigneesStore_List_Empty` | Return empty list for issue with no assignees |
| `TestAssigneesStore_List_Multiple` | List multiple assignees |
| `TestAssigneesStore_ListByUser` | List issues assigned to a user |
| `TestAssigneesStore_ListByUser_Empty` | Return empty list for user with no assignments |

### comments_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestCommentsStore_Create` | Create a new comment |
| `TestCommentsStore_GetByID` | Get comment by ID |
| `TestCommentsStore_GetByID_NotFound` | Return nil for non-existent comment |
| `TestCommentsStore_ListByIssue` | List comments for an issue |
| `TestCommentsStore_ListByIssue_Empty` | Return empty list for issue with no comments |
| `TestCommentsStore_ListByIssue_OrderedByCreatedAt` | Comments ordered by created_at ascending |
| `TestCommentsStore_Update` | Update comment content |
| `TestCommentsStore_Update_SetsEditedAt` | Update sets edited_at timestamp |
| `TestCommentsStore_Delete` | Delete a comment |
| `TestCommentsStore_CountByIssue` | Count comments for an issue |
| `TestCommentsStore_CountByIssue_Zero` | Return 0 for issue with no comments |

### fields_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestFieldsStore_Create` | Create a new field |
| `TestFieldsStore_Create_DuplicateKey` | Fail on duplicate key in project |
| `TestFieldsStore_Create_DuplicateName` | Fail on duplicate name in project |
| `TestFieldsStore_GetByID` | Get field by ID |
| `TestFieldsStore_GetByID_NotFound` | Return nil for non-existent field |
| `TestFieldsStore_GetByKey` | Get field by project ID and key |
| `TestFieldsStore_GetByKey_NotFound` | Return nil for non-existent key |
| `TestFieldsStore_ListByProject` | List fields in a project |
| `TestFieldsStore_ListByProject_Empty` | Return empty list for project with no fields |
| `TestFieldsStore_ListByProject_OrderedByPosition` | Fields ordered by position |
| `TestFieldsStore_ListByProject_ExcludesArchived` | Archived fields excluded by default |
| `TestFieldsStore_Update` | Update field name |
| `TestFieldsStore_UpdatePosition` | Update field position |
| `TestFieldsStore_Archive` | Archive a field |
| `TestFieldsStore_Unarchive` | Unarchive a field |
| `TestFieldsStore_Delete` | Delete a field |
| `TestFieldsStore_Delete_CascadeValues` | Deleting field removes values |

### values_store_test.go

| Test Name | Description |
|-----------|-------------|
| `TestValuesStore_Set_Text` | Set text value |
| `TestValuesStore_Set_Number` | Set number value |
| `TestValuesStore_Set_Bool` | Set boolean value |
| `TestValuesStore_Set_Date` | Set date value |
| `TestValuesStore_Set_Timestamp` | Set timestamp value |
| `TestValuesStore_Set_Ref` | Set reference value |
| `TestValuesStore_Set_JSON` | Set JSON value |
| `TestValuesStore_Set_Update` | Update existing value |
| `TestValuesStore_Get` | Get value by issue and field ID |
| `TestValuesStore_Get_NotFound` | Return nil for non-existent value |
| `TestValuesStore_ListByIssue` | List all values for an issue |
| `TestValuesStore_ListByIssue_Empty` | Return empty list for issue with no values |
| `TestValuesStore_ListByField` | List all values for a field |
| `TestValuesStore_ListByField_Empty` | Return empty list for field with no values |
| `TestValuesStore_Delete` | Delete a value |
| `TestValuesStore_DeleteByIssue` | Delete all values for an issue |
| `TestValuesStore_BulkSet` | Set multiple values at once |
| `TestValuesStore_BulkSet_Update` | Bulk set updates existing values |
| `TestValuesStore_BulkGetByIssues` | Get values for multiple issues |
| `TestValuesStore_BulkGetByIssues_Empty` | Return empty map when no values |

---

## Test Helpers

### Common Test Data Factories

```go
func createTestUser(t *testing.T, store *UsersStore) *users.User {
    u := &users.User{
        ID:           ulid.Make().String(),
        Email:        "test@example.com",
        Username:     "testuser",
        DisplayName:  "Test User",
        PasswordHash: "hashed",
    }
    err := store.Create(context.Background(), u)
    require.NoError(t, err)
    return u
}

func createTestWorkspace(t *testing.T, store *WorkspacesStore) *workspaces.Workspace {
    w := &workspaces.Workspace{
        ID:   ulid.Make().String(),
        Slug: "test-workspace",
        Name: "Test Workspace",
    }
    err := store.Create(context.Background(), w)
    require.NoError(t, err)
    return w
}

func createTestTeam(t *testing.T, store *TeamsStore, workspaceID string) *teams.Team {
    team := &teams.Team{
        ID:          ulid.Make().String(),
        WorkspaceID: workspaceID,
        Key:         "TEST",
        Name:        "Test Team",
    }
    err := store.Create(context.Background(), team)
    require.NoError(t, err)
    return team
}

func createTestProject(t *testing.T, store *ProjectsStore, teamID string) *projects.Project {
    p := &projects.Project{
        ID:           ulid.Make().String(),
        TeamID:       teamID,
        Key:          "PROJ",
        Name:         "Test Project",
        IssueCounter: 0,
    }
    err := store.Create(context.Background(), p)
    require.NoError(t, err)
    return p
}

func createTestColumn(t *testing.T, store *ColumnsStore, projectID string) *columns.Column {
    c := &columns.Column{
        ID:         ulid.Make().String(),
        ProjectID:  projectID,
        Name:       "Todo",
        Position:   0,
        IsDefault:  true,
        IsArchived: false,
    }
    err := store.Create(context.Background(), c)
    require.NoError(t, err)
    return c
}

func createTestIssue(t *testing.T, store *IssuesStore, projectID, columnID, creatorID string, number int) *issues.Issue {
    i := &issues.Issue{
        ID:        ulid.Make().String(),
        ProjectID: projectID,
        Number:    number,
        Key:       fmt.Sprintf("PROJ-%d", number),
        Title:     fmt.Sprintf("Test Issue %d", number),
        ColumnID:  columnID,
        Position:  0,
        CreatorID: creatorID,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    err := store.Create(context.Background(), i)
    require.NoError(t, err)
    return i
}
```

---

## Running Tests

```bash
# Run all store tests
cd blueprints/kanban
make test

# Run specific store tests
go test -v ./store/duckdb/... -run TestUsersStore
go test -v ./store/duckdb/... -run TestTeamsStore
go test -v ./store/duckdb/... -run TestIssuesStore

# Run with race detection
go test -race ./store/duckdb/...

# Run with coverage
go test -cover ./store/duckdb/...
```

---

## Test Requirements

1. All tests MUST use real DuckDB v2 driver (not mocks)
2. All tests MUST use in-memory database for isolation
3. All tests MUST clean up after themselves (via defer)
4. All tests MUST use `context.Background()` for simplicity
5. All tests MUST use `require` for fatal assertions
6. All tests MUST use `assert` for non-fatal assertions
7. All foreign key dependencies MUST be created in test setup
8. All tests MUST be independent and not rely on execution order
