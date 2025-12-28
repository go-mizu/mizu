# GitHome Store Tests Specification

## Overview

This document describes the comprehensive test suite for GitHome's DuckDB store implementations. The stores implement the `feature/*` Store interfaces and provide data persistence for the GitHub-like application.

## Store Interface Coverage

### UsersStore (`users.Store`)

| Interface Method | Test Coverage |
|-----------------|---------------|
| `Create` | `TestUsersStore_Create`, `TestUsersStore_Create_WithAllFields`, `TestUsersStore_Create_DuplicateEmail`, `TestUsersStore_Create_DuplicateUsername` |
| `GetByID` | `TestUsersStore_GetByID`, `TestUsersStore_GetByID_NotFound` |
| `GetByUsername` | `TestUsersStore_GetByUsername`, `TestUsersStore_GetByUsername_NotFound` |
| `GetByEmail` | `TestUsersStore_GetByEmail`, `TestUsersStore_GetByEmail_NotFound` |
| `Update` | `TestUsersStore_Update`, `TestUsersStore_Update_UpdatesTimestamp`, `TestUsersStore_Update_AdminFlag` |
| `Delete` | `TestUsersStore_Delete`, `TestUsersStore_Delete_NonExistent` |
| `List` | `TestUsersStore_List`, `TestUsersStore_List_Pagination`, `TestUsersStore_List_Empty`, `TestUsersStore_List_OrderByCreatedAt` |
| `CreateSession` | `TestUsersStore_CreateSession` |
| `GetSession` | `TestUsersStore_GetSession_NotFound` |
| `DeleteSession` | `TestUsersStore_DeleteSession` |
| `DeleteUserSessions` | `TestUsersStore_DeleteUserSessions`, `TestUsersStore_DeleteUserSessions_PreservesOtherUsers` |
| `DeleteExpiredSessions` | `TestUsersStore_DeleteExpiredSessions` |
| `UpdateSessionActivity` | `TestUsersStore_UpdateSessionActivity` |

### ReposStore (`repos.Store`)

| Interface Method | Test Coverage |
|-----------------|---------------|
| `Create` | `TestReposStore_Create`, `TestReposStore_Create_WithTopics`, `TestReposStore_Create_WithFork` |
| `GetByID` | `TestReposStore_GetByID`, `TestReposStore_GetByID_NotFound` |
| `GetByOwnerAndName` | `TestReposStore_GetByOwnerAndName`, `TestReposStore_GetByOwnerAndName_NotFound` |
| `Update` | `TestReposStore_Update`, `TestReposStore_Update_UpdatesTimestamp` |
| `Delete` | `TestReposStore_Delete` |
| `ListByOwner` | `TestReposStore_ListByOwner`, `TestReposStore_ListByOwner_Pagination` |
| `ListPublic` | `TestReposStore_ListPublic` |
| `ListByIDs` | `TestReposStore_ListByIDs`, `TestReposStore_ListByIDs_Empty` |
| `AddCollaborator` | `TestReposStore_AddCollaborator` |
| `RemoveCollaborator` | `TestReposStore_RemoveCollaborator` |
| `GetCollaborator` | `TestReposStore_GetCollaborator_NotFound` |
| `ListCollaborators` | `TestReposStore_ListCollaborators`, `TestReposStore_ListCollaborators_Empty`, `TestReposStore_Collaborator_AllPermissionLevels` |
| `Star` | `TestReposStore_Star` |
| `Unstar` | `TestReposStore_Unstar` |
| `IsStarred` | `TestReposStore_IsStarred_NotStarred` |
| `ListStarredByUser` | `TestReposStore_ListStarredByUser`, `TestReposStore_ListStarredByUser_Empty`, `TestReposStore_ListStarredByUser_Pagination` |

### IssuesStore (`issues.Store`)

| Interface Method | Test Coverage |
|-----------------|---------------|
| `Create` | `TestIssuesStore_Create`, `TestIssuesStore_Create_WithAllFields` |
| `GetByID` | `TestIssuesStore_GetByID`, `TestIssuesStore_GetByID_NotFound` |
| `GetByNumber` | `TestIssuesStore_GetByNumber`, `TestIssuesStore_GetByNumber_NotFound`, `TestIssuesStore_GetByNumber_DifferentRepos` |
| `Update` | `TestIssuesStore_Update`, `TestIssuesStore_Update_UpdatesTimestamp`, `TestIssuesStore_Update_CloseIssue`, `TestIssuesStore_Update_LockIssue` |
| `Delete` | `TestIssuesStore_Delete`, `TestIssuesStore_Delete_NonExistent` |
| `List` | `TestIssuesStore_List`, `TestIssuesStore_List_FilterByState`, `TestIssuesStore_List_Pagination`, `TestIssuesStore_List_Empty`, `TestIssuesStore_List_OrderByCreatedAt` |
| `GetNextNumber` | `TestIssuesStore_GetNextNumber_Empty`, `TestIssuesStore_GetNextNumber_WithExisting`, `TestIssuesStore_GetNextNumber_WithGaps`, `TestIssuesStore_GetNextNumber_PerRepo` |
| `AddLabel` | `TestIssuesStore_AddLabel`, `TestIssuesStore_AddLabel_Multiple` |
| `RemoveLabel` | `TestIssuesStore_RemoveLabel` |
| `ListLabels` | `TestIssuesStore_ListLabels_Empty` |
| `AddAssignee` | `TestIssuesStore_AddAssignee`, `TestIssuesStore_AddAssignee_Multiple` |
| `RemoveAssignee` | `TestIssuesStore_RemoveAssignee` |
| `ListAssignees` | `TestIssuesStore_ListAssignees_Empty` |

## Test Categories

### 1. CRUD Operations
Basic Create, Read, Update, Delete operations for each entity type.

### 2. Uniqueness Constraints
- Duplicate email detection
- Duplicate username detection
- Repository name uniqueness per owner

### 3. Query Operations
- Lookup by various identifiers (ID, username, email, slug)
- List operations with pagination
- Filtering (e.g., issues by state)
- Sorting (e.g., by created_at DESC)

### 4. Relationship Operations
- Collaborator management (add, remove, list)
- Star/Unstar repositories
- Issue labels and assignees
- Fork relationships

### 5. Session Management
- Session creation and retrieval
- Session deletion (single, user-wide, expired)
- Session activity tracking

### 6. Edge Cases
- Not-found scenarios (return nil, not error)
- Empty list handling
- Pagination boundaries
- Issue number auto-increment per repository

## Integration Tests

### Repository Lifecycle
- `TestReposStore_DeleteRepoRemovesCollaborators` - Documents orphaned collaborator behavior
- `TestReposStore_UserWithMultipleRepos` - Multiple repos per user
- `TestReposStore_OrgTypeOwner` - Organization-owned repositories

### Issue Lifecycle
- `TestIssuesStore_IssueLifecycle` - Full create/label/assign/close cycle
- `TestIssuesStore_DeleteIssueRemovesLabels` - Orphan behavior
- `TestIssuesStore_DeleteIssueRemovesAssignees` - Orphan behavior

## Running Tests

```bash
# Run all store tests
make test-store

# Run with verbose output
GOWORK=off CGO_ENABLED=1 go test -v ./store/duckdb/...

# Run specific store tests
go test -v ./store/duckdb/... -run TestUsersStore
go test -v ./store/duckdb/... -run TestReposStore
go test -v ./store/duckdb/... -run TestIssuesStore
```

## Dependencies

- DuckDB driver: `github.com/duckdb/duckdb-go/v2`
- ULID generation: `github.com/oklog/ulid/v2`

## Technical Notes

### Database Setup
Tests use in-memory DuckDB databases (`sql.Open("duckdb", "")`) that are isolated per test function. Each test gets a fresh database via `setupTestStore()`.

### ID Generation
Test helpers use ULID for unique ID generation. Usernames and emails are derived from the ULID to ensure uniqueness across rapid test execution.

### Null Handling
- Not-found queries return `(nil, nil)` instead of `(nil, sql.ErrNoRows)`
- Nullable fields use `sql.NullString` and `sql.NullTime` for scanning
- Empty strings are stored as empty, not NULL (except for optional fields)

### Timestamp Behavior
- `CreatedAt` is set by the caller before insert
- `UpdatedAt` is automatically set to `time.Now()` in Update methods

## Known Limitations

1. **No CASCADE deletes**: Deleting a repository leaves orphaned collaborators, stars, and issues. A future enhancement could add cleanup logic.

2. **Issue numbers per repo**: The `GetNextNumber` function returns `MAX(number) + 1`, which doesn't reuse deleted issue numbers.

3. **Topics storage**: Repository topics are stored as comma-separated strings, limiting topic values that contain commas.

## Test Count Summary

| Store | Test Functions | Test Cases |
|-------|---------------|------------|
| Store | 3 | 3 |
| UsersStore | 18 | 18 |
| ReposStore | 24 | 24 |
| IssuesStore | 21 | 21 |
| **Total** | **66** | **66** |
