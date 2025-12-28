# GitHome Services - Implementation Status and Test Plan

This document provides a comprehensive analysis of all services in `feature/*`, their implementation status, and test plans.

## Service Implementation Status

### Production-Ready Services (Fully Implemented)

| Service | Methods | Status | Notes |
|---------|---------|--------|-------|
| **users** | 15 | Production | Full CRUD, auth, follow/unfollow, password management |
| **repos** | 24 | Production | Full CRUD, fork, transfer, topics, languages |
| **git** | 17 | Production | Blob, commit, ref, tree, tag operations (has tests) |
| **issues** | 16 | Production | Full CRUD, lock/unlock, assignees, events |
| **orgs** | 11 | Production | Full org management, membership |
| **branches** | 8 | Production | Branch CRUD, protection rules |
| **labels** | 7 | Production | Label CRUD per repository |
| **milestones** | 7 | Production | Milestone CRUD with progress tracking |
| **comments** | 12 | Production | Issue/PR/commit comments |
| **stars** | 8 | Production | Star/unstar with counter management |
| **watches** | 7 | Production | Watch subscription management |
| **releases** | 14 | Production | Release and asset management |
| **reactions** | 18 | Production | Reactions for issues, comments, PRs, releases |
| **webhooks** | 20 | Production | Webhook CRUD, dispatch, delivery tracking |
| **teams** | 18 | Production | Team management, membership, repo permissions |
| **collaborators** | 12 | Production | Collaborator and invitation management |
| **activities** | 11 | Production | Event feed management |
| **notifications** | 11 | Production | Notification thread management |
| **pulls** | 15 | Production | Pull request management, reviews |
| **commits** | 4 | Production | Commit listing and comparison |

### Delegating Services (Minimal Logic)

| Service | Methods | Status | Notes |
|---------|---------|--------|-------|
| **search** | 7 | Delegate | Delegates directly to store, pagination only |

---

## Test Plan by Service

All tests use real in-memory DuckDB via `store/duckdb` package - no mocking.

### 1. Users Service (`feature/users/service_test.go`)

#### Test Setup
```go
func setupTestService(t *testing.T) (*users.Service, *duckdb.Store, func())
```

#### Test Cases

**User Creation**
- `TestService_Create_Success` - Create user with valid input
- `TestService_Create_DuplicateLogin` - Reject duplicate login
- `TestService_Create_DuplicateEmail` - Reject duplicate email
- `TestService_Create_PasswordHashed` - Verify password is hashed

**Authentication**
- `TestService_Authenticate_ByLogin` - Auth by login
- `TestService_Authenticate_ByEmail` - Auth by email
- `TestService_Authenticate_WrongPassword` - Reject wrong password
- `TestService_Authenticate_NonexistentUser` - Reject unknown user

**User Retrieval**
- `TestService_GetByID_Success` - Get existing user
- `TestService_GetByID_NotFound` - 404 for unknown ID
- `TestService_GetByLogin_Success` - Get by username
- `TestService_GetByEmail_Success` - Get by email
- `TestService_List_Pagination` - Verify pagination

**User Updates**
- `TestService_Update_Name` - Update name field
- `TestService_Update_Bio` - Update bio field
- `TestService_Delete_Success` - Delete user

**Password Management**
- `TestService_UpdatePassword_Success` - Change password
- `TestService_UpdatePassword_WrongOld` - Reject wrong old password

**Follow System**
- `TestService_Follow_Success` - Follow user, counters updated
- `TestService_Follow_AlreadyFollowing` - Idempotent follow
- `TestService_Unfollow_Success` - Unfollow, counters decremented
- `TestService_Unfollow_NotFollowing` - Idempotent unfollow
- `TestService_IsFollowing_True` - Check following relationship
- `TestService_IsFollowing_False` - Check not following
- `TestService_ListFollowers` - List followers with pagination
- `TestService_ListFollowing` - List following with pagination

---

### 2. Repos Service (`feature/repos/service_test.go`)

#### Test Cases

**Repository Creation**
- `TestService_Create_Success` - Create repo for user
- `TestService_Create_DuplicateName` - Reject duplicate name
- `TestService_Create_VisibilityPublic` - Default visibility
- `TestService_Create_VisibilityPrivate` - Private repo
- `TestService_CreateForOrg_Success` - Create org repo

**Repository Retrieval**
- `TestService_Get_Success` - Get by owner/name
- `TestService_Get_NotFound` - 404 for unknown repo
- `TestService_GetByID_Success` - Get by ID
- `TestService_ListForUser` - List user repos
- `TestService_ListForOrg` - List org repos
- `TestService_ListForAuthenticatedUser` - List own repos

**Repository Updates**
- `TestService_Update_Description` - Update description
- `TestService_Update_Rename` - Rename repository
- `TestService_Update_Visibility` - Change visibility
- `TestService_Delete_Success` - Delete repo

**Repository Transfer**
- `TestService_Transfer_ToUser` - Transfer to user
- `TestService_Transfer_ToOrg` - Transfer to org
- `TestService_Transfer_WithRename` - Transfer with new name

**Forking**
- `TestService_CreateFork_Success` - Fork repo
- `TestService_CreateFork_IncrementsForks` - Fork count updated
- `TestService_ListForks` - List forks

**Topics & Languages**
- `TestService_ListTopics` - Get topics
- `TestService_ReplaceTopics` - Replace all topics
- `TestService_ListLanguages` - Get language stats

**Counter Management**
- `TestService_IncrementOpenIssues` - Issue counter
- `TestService_IncrementStargazers` - Star counter
- `TestService_IncrementWatchers` - Watch counter

---

### 3. Issues Service (`feature/issues/service_test.go`)

#### Test Cases

**Issue Creation**
- `TestService_Create_Success` - Create issue
- `TestService_Create_WithAssignees` - Create with assignees
- `TestService_Create_IncrementsOpenIssues` - Counter updated
- `TestService_Create_RepoNotFound` - 404 for unknown repo

**Issue Retrieval**
- `TestService_Get_Success` - Get by number
- `TestService_Get_NotFound` - 404 for unknown issue
- `TestService_GetByID_Success` - Get by ID
- `TestService_ListForRepo` - List with pagination
- `TestService_ListForRepo_FilterByState` - Filter open/closed/all
- `TestService_ListForUser` - User's issues

**Issue Updates**
- `TestService_Update_Title` - Update title
- `TestService_Update_Body` - Update body
- `TestService_Update_Close` - Close issue, counter decremented
- `TestService_Update_Reopen` - Reopen issue, counter incremented

**Locking**
- `TestService_Lock_Success` - Lock issue
- `TestService_Lock_WithReason` - Lock with reason
- `TestService_Unlock_Success` - Unlock issue

**Assignees**
- `TestService_AddAssignees` - Add assignees
- `TestService_RemoveAssignees` - Remove assignees
- `TestService_CheckAssignee` - Verify assignee eligibility

**Events**
- `TestService_ListEvents` - List issue events
- `TestService_CreateEvent` - Create event

---

### 4. Stars Service (`feature/stars/service_test.go`)

#### Test Cases

- `TestService_Star_Success` - Star repo, counter incremented
- `TestService_Star_AlreadyStarred` - Idempotent star
- `TestService_Star_RepoNotFound` - 404 for unknown repo
- `TestService_Unstar_Success` - Unstar, counter decremented
- `TestService_Unstar_NotStarred` - Idempotent unstar
- `TestService_IsStarred_True` - Check starred
- `TestService_IsStarred_False` - Check not starred
- `TestService_ListStargazers` - List with pagination
- `TestService_ListStargazersWithTimestamps` - Include timestamps
- `TestService_ListForUser` - User's starred repos
- `TestService_ListForAuthenticatedUser` - Own starred repos

---

### 5. Watches Service (`feature/watches/service_test.go`)

#### Test Cases

- `TestService_SetSubscription_Subscribe` - Subscribe to repo
- `TestService_SetSubscription_Ignore` - Set ignored
- `TestService_SetSubscription_Update` - Update existing
- `TestService_GetSubscription_Success` - Get subscription
- `TestService_GetSubscription_NotFound` - 404 if not subscribed
- `TestService_DeleteSubscription_Success` - Unsubscribe
- `TestService_DeleteSubscription_NotSubscribed` - Idempotent
- `TestService_ListWatchers` - List repo watchers
- `TestService_ListForUser` - User's watched repos

---

### 6. Labels Service (`feature/labels/service_test.go`)

#### Test Cases

- `TestService_Create_Success` - Create label
- `TestService_Create_DuplicateName` - Reject duplicate
- `TestService_Get_Success` - Get by name
- `TestService_Get_NotFound` - 404 for unknown
- `TestService_List` - List repo labels
- `TestService_Update_Name` - Rename label
- `TestService_Update_Color` - Change color
- `TestService_Delete_Success` - Delete label

---

### 7. Milestones Service (`feature/milestones/service_test.go`)

#### Test Cases

- `TestService_Create_Success` - Create milestone
- `TestService_Create_WithDueDate` - With due date
- `TestService_Get_Success` - Get by number
- `TestService_Get_NotFound` - 404 for unknown
- `TestService_List` - List milestones
- `TestService_List_FilterByState` - Filter open/closed/all
- `TestService_Update_Title` - Update title
- `TestService_Update_Close` - Close milestone
- `TestService_Delete_Success` - Delete milestone

---

### 8. Comments Service (`feature/comments/service_test.go`)

#### Test Cases

**Issue Comments**
- `TestService_CreateIssueComment_Success` - Create comment
- `TestService_GetIssueComment_Success` - Get by ID
- `TestService_ListIssueComments` - List for issue
- `TestService_UpdateIssueComment` - Update comment
- `TestService_DeleteIssueComment` - Delete comment

**Commit Comments**
- `TestService_CreateCommitComment_Success` - Create comment
- `TestService_GetCommitComment_Success` - Get by ID
- `TestService_ListCommitComments` - List for commit
- `TestService_UpdateCommitComment` - Update comment
- `TestService_DeleteCommitComment` - Delete comment

---

### 9. Collaborators Service (`feature/collaborators/service_test.go`)

#### Test Cases

- `TestService_Add_NewCollaborator` - Add creates invitation
- `TestService_Add_UpdatePermission` - Update existing permission
- `TestService_IsCollaborator_True` - Check is collaborator
- `TestService_IsCollaborator_False` - Check not collaborator
- `TestService_GetPermission_Owner` - Owner has admin
- `TestService_GetPermission_Collaborator` - Check permission level
- `TestService_GetPermission_PublicRead` - Public repo read access
- `TestService_Remove_Success` - Remove collaborator
- `TestService_ListInvitations` - List pending invitations
- `TestService_AcceptInvitation` - Accept becomes collaborator
- `TestService_DeclineInvitation` - Decline removes invitation
- `TestService_UpdateInvitation` - Change invitation permission

---

### 10. Teams Service (`feature/teams/service_test.go`)

#### Test Cases

**Team CRUD**
- `TestService_Create_Success` - Create team
- `TestService_Create_WithMaintainers` - Create with initial maintainers
- `TestService_Create_DuplicateSlug` - Reject duplicate
- `TestService_GetBySlug_Success` - Get by slug
- `TestService_GetByID_Success` - Get by ID
- `TestService_List` - List org teams
- `TestService_Update_Name` - Update name (changes slug)
- `TestService_Delete_Success` - Delete team

**Membership**
- `TestService_AddMembership_NewMember` - Add member
- `TestService_AddMembership_UpdateRole` - Update existing role
- `TestService_GetMembership_Success` - Get membership
- `TestService_GetMembership_NotMember` - 404 if not member
- `TestService_RemoveMembership` - Remove from team
- `TestService_ListMembers` - List team members

**Repository Access**
- `TestService_AddRepo_Success` - Add repo to team
- `TestService_AddRepo_UpdatePermission` - Update repo permission
- `TestService_CheckRepoPermission` - Check permission level
- `TestService_RemoveRepo` - Remove repo from team
- `TestService_ListRepos` - List team repos

**Hierarchy**
- `TestService_ListChildren` - List child teams

---

### 11. Releases Service (`feature/releases/service_test.go`)

#### Test Cases

**Release CRUD**
- `TestService_Create_Success` - Create release
- `TestService_Create_Draft` - Create draft release
- `TestService_Create_Prerelease` - Create prerelease
- `TestService_Create_DuplicateTag` - Reject duplicate tag
- `TestService_Get_Success` - Get by ID
- `TestService_GetLatest` - Get latest non-draft
- `TestService_GetByTag` - Get by tag name
- `TestService_List` - List releases
- `TestService_Update_MakePublic` - Publish draft
- `TestService_Delete_Success` - Delete release

**Assets**
- `TestService_UploadAsset` - Upload file
- `TestService_GetAsset` - Get asset
- `TestService_ListAssets` - List release assets
- `TestService_UpdateAsset` - Rename asset
- `TestService_DeleteAsset` - Delete asset
- `TestService_DownloadAsset` - Download and increment count

**Notes**
- `TestService_GenerateNotes` - Generate release notes

---

### 12. Reactions Service (`feature/reactions/service_test.go`)

#### Test Cases

**Issue Reactions**
- `TestService_CreateForIssue_Success` - Add reaction
- `TestService_CreateForIssue_Idempotent` - Return existing
- `TestService_CreateForIssue_InvalidContent` - Reject invalid
- `TestService_ListForIssue` - List reactions
- `TestService_DeleteForIssue` - Remove reaction

**Comment Reactions**
- `TestService_CreateForIssueComment` - Add to comment
- `TestService_ListForIssueComment` - List reactions
- `TestService_DeleteForIssueComment` - Remove reaction

**Release Reactions**
- `TestService_CreateForRelease` - Add to release
- `TestService_ListForRelease` - List reactions

**Rollup**
- `TestService_GetRollup` - Get reaction counts

---

### 13. Webhooks Service (`feature/webhooks/service_test.go`)

#### Test Cases

**Repo Webhooks**
- `TestService_CreateForRepo_Success` - Create webhook
- `TestService_CreateForRepo_DefaultEvents` - Default to push
- `TestService_GetForRepo_Success` - Get by ID
- `TestService_ListForRepo` - List webhooks
- `TestService_UpdateForRepo` - Update config
- `TestService_DeleteForRepo` - Delete webhook

**Org Webhooks**
- `TestService_CreateForOrg` - Create org webhook
- `TestService_GetForOrg` - Get org webhook
- `TestService_ListForOrg` - List org webhooks

**Dispatch & Delivery**
- `TestService_Dispatch_Success` - Dispatch event
- `TestService_Dispatch_Inactive` - Skip inactive webhook
- `TestService_Dispatch_WithSignature` - Sign payload
- `TestService_ListDeliveriesForRepo` - List deliveries
- `TestService_GetDeliveryForRepo` - Get delivery
- `TestService_RedeliverForRepo` - Redeliver event

**Ping**
- `TestService_PingRepo` - Send ping event
- `TestService_TestRepo` - Test webhook

---

### 14. Notifications Service (`feature/notifications/service_test.go`)

#### Test Cases

- `TestService_Create_Success` - Create notification
- `TestService_List` - List user notifications
- `TestService_List_FilterUnread` - Filter unread only
- `TestService_ListForRepo` - Notifications for repo
- `TestService_GetThread` - Get thread
- `TestService_MarkAsRead` - Mark all read
- `TestService_MarkRepoAsRead` - Mark repo notifications read
- `TestService_MarkThreadAsRead` - Mark single thread read
- `TestService_MarkThreadAsDone` - Remove notification
- `TestService_GetThreadSubscription` - Get subscription
- `TestService_SetThreadSubscription_Ignore` - Ignore thread
- `TestService_DeleteThreadSubscription` - Remove subscription

---

### 15. Activities Service (`feature/activities/service_test.go`)

#### Test Cases

- `TestService_Create_Success` - Create event
- `TestService_ListPublic` - List public events
- `TestService_ListForRepo` - Events for repo
- `TestService_ListForOrg` - Events for org
- `TestService_ListForUser` - Events by user
- `TestService_ListReceivedEvents` - Events received by user
- `TestService_GetFeeds` - Get feed URLs

---

### 16. Orgs Service (`feature/orgs/service_test.go`)

#### Test Cases

- `TestService_Create_Success` - Create org
- `TestService_Create_DuplicateLogin` - Reject duplicate
- `TestService_Get_Success` - Get by login
- `TestService_GetByID_Success` - Get by ID
- `TestService_List` - List all orgs
- `TestService_Update_Description` - Update org
- `TestService_Delete_Success` - Delete org
- `TestService_ListMembers` - List org members
- `TestService_GetMembership` - Get user membership
- `TestService_AddMember` - Add member
- `TestService_RemoveMember` - Remove member

---

## Test Infrastructure

### Shared Test Helper

Create `feature/testutil/testutil.go`:

```go
package testutil

import (
    "context"
    "database/sql"
    "testing"

    _ "github.com/duckdb/duckdb-go/v2"
    "github.com/go-mizu/blueprints/githome/store/duckdb"
)

// Setup creates an in-memory DuckDB store for testing
func Setup(t *testing.T) (*duckdb.Store, func()) {
    t.Helper()

    db, err := sql.Open("duckdb", "")
    if err != nil {
        t.Fatalf("failed to open duckdb: %v", err)
    }

    store, err := duckdb.New(db)
    if err != nil {
        db.Close()
        t.Fatalf("failed to create store: %v", err)
    }

    if err := store.Ensure(context.Background()); err != nil {
        store.Close()
        t.Fatalf("failed to ensure schema: %v", err)
    }

    cleanup := func() {
        store.Close()
    }

    return store, cleanup
}
```

### Test Conventions

1. Each test file follows pattern: `feature/{name}/service_test.go`
2. Tests use real DuckDB - no mocks
3. Each test creates fresh store via `testutil.Setup()`
4. Test functions follow `Test{Service}_{Method}_{Scenario}` naming
5. Cleanup functions ensure resource cleanup

---

## Implementation Order

1. **Phase 1**: Core entities
   - users, repos, orgs

2. **Phase 2**: Issue tracking
   - issues, labels, milestones, comments

3. **Phase 3**: Social features
   - stars, watches, reactions

4. **Phase 4**: Collaboration
   - collaborators, teams

5. **Phase 5**: Releases & automation
   - releases, webhooks

6. **Phase 6**: Activity & notifications
   - activities, notifications

---

## Notes

- The `git` service already has comprehensive tests in `feature/git/service_test.go`
- The `search` service delegates to store and has minimal logic to test
- The `pulls` and `commits` services share patterns with `issues`
- All services follow dependency injection pattern with store interfaces
