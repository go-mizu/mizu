# Forum Blueprint - Testing & CLI Enhancement Plan

## Overview

This spec details the implementation plan for comprehensive testing and CLI enhancement of the Forum blueprint. The forum is a full-featured discussion platform inspired by Reddit/Discourse.

## Current State

- **7 Store Files**: `accounts_store.go`, `boards_store.go`, `threads_store.go`, `comments_store.go`, `votes_store.go`, `bookmarks_store.go`, `notifications_store.go`
- **7 Feature APIs**: accounts, boards, threads, comments, votes, bookmarks, notifications
- **CLI**: Basic Cobra setup with `serve`, `init`, `seed` commands
- **Web Server**: Full implementation with HTML pages and API routes
- **Schema**: 12 tables (accounts, sessions, boards, board_members, board_moderators, threads, comments, votes, bookmarks, notifications, mod_actions, bans, reports)

## Implementation Plan

---

## Phase 1: Makefile

Create `forum/Makefile` with targets for building, running, testing, and development.

```makefile
# Key targets:
- build       # Build to $HOME/bin/forum
- run         # Run CLI with ARGS
- serve       # Run serve command
- dev         # Run serve --dev
- init        # Initialize database
- seed        # Seed sample data
- test        # Run all tests
- test-store  # Run store tests only
- test-e2e    # Run e2e tests (requires E2E_TEST=1)
- test-cli    # Run CLI tests
- clean       # Remove binary
- clean-data  # Remove data directory
```

---

## Phase 2: Store Tests

### 2.1 Test Infrastructure (`store/duckdb/store_test.go`)

Shared test utilities:
- `setupTestStore(t)` - Creates in-memory DuckDB store for testing
- `randomID()` - Generates random ULID for test data
- Test fixtures for common data patterns

### 2.2 AccountsStore Tests (`accounts_store_test.go`)

| Test Case | Description |
|-----------|-------------|
| `TestAccountsStore_Create` | Create account, verify fields |
| `TestAccountsStore_Create_DuplicateUsername` | Reject duplicate usernames |
| `TestAccountsStore_Create_DuplicateEmail` | Reject duplicate emails |
| `TestAccountsStore_GetByID` | Retrieve by ID |
| `TestAccountsStore_GetByID_NotFound` | Return ErrNotFound |
| `TestAccountsStore_GetByUsername` | Case-insensitive username lookup |
| `TestAccountsStore_GetByEmail` | Case-insensitive email lookup |
| `TestAccountsStore_Update` | Update account fields |
| `TestAccountsStore_Delete` | Delete account |
| `TestAccountsStore_Session_CRUD` | Create, get, delete sessions |
| `TestAccountsStore_Session_Expiry` | Clean expired sessions |
| `TestAccountsStore_List` | List with ordering |
| `TestAccountsStore_Search` | Search by username/display name |

### 2.3 BoardsStore Tests (`boards_store_test.go`)

| Test Case | Description |
|-----------|-------------|
| `TestBoardsStore_Create` | Create board, verify fields |
| `TestBoardsStore_Create_DuplicateName` | Reject duplicate names |
| `TestBoardsStore_GetByName` | Case-insensitive name lookup |
| `TestBoardsStore_GetByID` | Retrieve by ID |
| `TestBoardsStore_Update` | Update board fields |
| `TestBoardsStore_Delete` | Delete board |
| `TestBoardsStore_Members` | Add/remove/list members |
| `TestBoardsStore_Members_Idempotent` | Adding member twice is no-op |
| `TestBoardsStore_Moderators` | Add/remove/list moderators |
| `TestBoardsStore_Moderators_Permissions` | JSON permissions stored correctly |
| `TestBoardsStore_ListJoinedBoards` | List boards user has joined |
| `TestBoardsStore_ListModeratedBoards` | List boards user moderates |
| `TestBoardsStore_List` | List public boards by member count |
| `TestBoardsStore_Search` | Search by name/title |
| `TestBoardsStore_ListPopular` | List by member count |
| `TestBoardsStore_ListNew` | List by creation date |

### 2.4 ThreadsStore Tests (`threads_store_test.go`)

| Test Case | Description |
|-----------|-------------|
| `TestThreadsStore_Create` | Create thread, verify fields |
| `TestThreadsStore_GetByID` | Retrieve thread by ID |
| `TestThreadsStore_GetByID_NotFound` | Return ErrNotFound |
| `TestThreadsStore_Update` | Update thread fields |
| `TestThreadsStore_Delete` | Delete thread |
| `TestThreadsStore_List_Hot` | List sorted by hot score |
| `TestThreadsStore_List_New` | List sorted by creation date |
| `TestThreadsStore_List_Top` | List sorted by score |
| `TestThreadsStore_List_TimeRange` | Filter by time range (day/week/month/year) |
| `TestThreadsStore_ListByBoard` | List threads in specific board |
| `TestThreadsStore_ListByAuthor` | List threads by author |
| `TestThreadsStore_UpdateHotScores` | Batch hot score recalculation |
| `TestThreadsStore_Pinned` | Pinned threads appear first |
| `TestThreadsStore_Removed_Excluded` | Removed threads excluded from lists |

### 2.5 CommentsStore Tests (`comments_store_test.go`)

| Test Case | Description |
|-----------|-------------|
| `TestCommentsStore_Create` | Create comment with path |
| `TestCommentsStore_Create_Nested` | Create nested comment (depth, path) |
| `TestCommentsStore_GetByID` | Retrieve comment by ID |
| `TestCommentsStore_GetByID_NotFound` | Return ErrNotFound |
| `TestCommentsStore_Update` | Update comment content |
| `TestCommentsStore_Delete` | Delete comment |
| `TestCommentsStore_ListByThread_Best` | List sorted by Wilson score |
| `TestCommentsStore_ListByThread_Top` | List sorted by score |
| `TestCommentsStore_ListByThread_New` | List sorted by creation date |
| `TestCommentsStore_ListByThread_Old` | List sorted by creation date ASC |
| `TestCommentsStore_ListByThread_Controversial` | List sorted by controversy |
| `TestCommentsStore_ListByParent` | List direct children |
| `TestCommentsStore_ListByPath` | List by path prefix |
| `TestCommentsStore_ListByAuthor` | List by author |
| `TestCommentsStore_IncrementChildCount` | Update child counts |
| `TestCommentsStore_Removed_Excluded` | Removed comments excluded |

### 2.6 VotesStore Tests (`votes_store_test.go`)

| Test Case | Description |
|-----------|-------------|
| `TestVotesStore_Create` | Create vote |
| `TestVotesStore_GetByTarget` | Retrieve vote by target |
| `TestVotesStore_GetByTarget_NotFound` | Return ErrNotFound |
| `TestVotesStore_Update` | Update vote value |
| `TestVotesStore_Delete` | Delete vote |
| `TestVotesStore_GetByTargets` | Batch retrieve votes |
| `TestVotesStore_GetByTargets_Empty` | Handle empty target list |
| `TestVotesStore_CountByTarget` | Count up/down votes |

### 2.7 BookmarksStore Tests (`bookmarks_store_test.go`)

| Test Case | Description |
|-----------|-------------|
| `TestBookmarksStore_Create` | Create bookmark |
| `TestBookmarksStore_GetByTarget` | Retrieve bookmark |
| `TestBookmarksStore_GetByTarget_NotFound` | Return ErrNotFound |
| `TestBookmarksStore_Delete` | Delete bookmark |
| `TestBookmarksStore_List` | List by account and type |
| `TestBookmarksStore_GetByTargets` | Batch retrieve bookmarks |

### 2.8 NotificationsStore Tests (`notifications_store_test.go`)

| Test Case | Description |
|-----------|-------------|
| `TestNotificationsStore_Create` | Create notification |
| `TestNotificationsStore_GetByID` | Retrieve by ID |
| `TestNotificationsStore_GetByID_NotFound` | Return ErrNotFound |
| `TestNotificationsStore_List` | List by account |
| `TestNotificationsStore_List_Unread` | Filter unread only |
| `TestNotificationsStore_MarkRead` | Mark notifications as read |
| `TestNotificationsStore_MarkAllRead` | Mark all as read for account |
| `TestNotificationsStore_Delete` | Delete notification |
| `TestNotificationsStore_DeleteBefore` | Delete old notifications |
| `TestNotificationsStore_CountUnread` | Count unread |

---

## Phase 3: E2E Web Tests

### `app/web/server_e2e_test.go`

Full integration tests with real DuckDB database.

#### Test Setup

```go
//go:build e2e

func setupTestServer(t *testing.T) (*httptest.Server, *duckdb.Store)
func createTestUser(t *testing.T, store *duckdb.Store) *accounts.Account
func createTestBoard(t *testing.T, store *duckdb.Store, owner string) *boards.Board
func createTestThread(t *testing.T, store *duckdb.Store, board, author string) *threads.Thread
func loginUser(t *testing.T, ts *httptest.Server, username, password string) string // returns session cookie
```

#### API Tests

| Test Group | Tests |
|------------|-------|
| **Auth API** | Register, Login, Logout, Me |
| **Boards API** | List, Create, Get, Update, Join, Leave, Moderators |
| **Threads API** | Create, Get, Update, Delete, List, ListByBoard |
| **Comments API** | Create, Get, Update, Delete, List, Nested |
| **Votes API** | Vote thread, Unvote thread, Vote comment, Unvote comment |
| **Bookmarks API** | Bookmark thread, Unbookmark, Bookmark comment |
| **Users API** | Get profile, List threads, List comments |
| **Notifications API** | List, MarkRead, MarkAllRead |

#### HTML Page Tests

| Test | Description |
|------|-------------|
| `TestHTMLPages_Home` | Homepage loads, shows threads |
| `TestHTMLPages_Board` | Board page with threads |
| `TestHTMLPages_Thread` | Thread page with comments |
| `TestHTMLPages_User` | User profile page |
| `TestHTMLPages_Login` | Login form renders |
| `TestHTMLPages_Register` | Register form renders |
| `TestHTMLPages_AuthRequired` | Protected pages redirect |

#### Integration Scenarios

| Scenario | Description |
|----------|-------------|
| `TestScenario_UserJourney` | Register -> Login -> Create board -> Post thread -> Comment -> Vote |
| `TestScenario_Moderation` | Create thread -> Report -> Mod removes -> Verify hidden |
| `TestScenario_Voting` | Vote -> Change vote -> Remove vote -> Verify counts |
| `TestScenario_Notifications` | Post reply -> Verify notification created -> Mark read |

---

## Phase 4: CLI Enhancement with Fang

### 4.1 Dependencies

Add to `go.mod`:
```
github.com/charmbracelet/fang v0.x.x
github.com/charmbracelet/lipgloss v0.x.x
```

### 4.2 UI Module (`cli/ui.go`)

Create styled UI helper matching microblog patterns:

```go
type UI struct {
    mu       sync.Mutex
    spinning bool
    spinMsg  string
    spinDone chan struct{}
}

// Methods
func (u *UI) Header(icon, title string)
func (u *UI) Info(label, value string)
func (u *UI) Success(message string)
func (u *UI) Error(message string)
func (u *UI) Warn(message string)
func (u *UI) Hint(message string)
func (u *UI) Blank()
func (u *UI) StartSpinner(message string)
func (u *UI) UpdateSpinner(message string)
func (u *UI) StopSpinner(message string, duration time.Duration)
func (u *UI) StopSpinnerError(message string)
func (u *UI) Table(headers []string, rows [][]string)
```

### 4.3 Enhanced Commands

#### `root.go` - Fang Integration

```go
func Execute(ctx context.Context) error {
    root := &cobra.Command{...}

    root.AddCommand(
        NewServe(),
        NewInit(),
        NewSeed(),
        NewUser(),    // New
        NewBoard(),   // New
        NewStats(),   // New
    )

    return fang.Execute(ctx, root,
        fang.WithVersion(Version),
        fang.WithCommit(Commit),
    )
}
```

#### `serve.go` - Enhanced Output

- Show styled header with icon
- Display config info (addr, data dir, mode)
- Spinner during startup
- Success message with clickable URL
- Graceful shutdown message

#### `init.go` - Enhanced Output

- Progress spinner for each step
- Show table counts after init
- Duration for each step

#### `seed.go` - Enhanced Output

- Progress for each entity type
- Summary table at end
- Duration tracking

#### `user.go` - New User Management

```bash
forum user create <username>    # Create user
forum user list                 # List all users
forum user get <username>       # Show user details
forum user suspend <username>   # Suspend user
forum user unsuspend <username> # Unsuspend user
forum user admin <username>     # Toggle admin status
```

#### `board.go` - New Board Management

```bash
forum board create <name>       # Create board
forum board list                # List all boards
forum board get <name>          # Show board details
forum board archive <name>      # Archive board
```

#### `stats.go` - New Statistics

```bash
forum stats                     # Show database statistics
```

Shows:
- Total users, threads, comments
- Most active boards
- Recent activity

---

## Phase 5: CLI Tests

### 5.1 Test Infrastructure (`cli/cli_test.go`)

```go
func setupTestCLI(t *testing.T) (dataDir string, cleanup func())
func runCommand(t *testing.T, args ...string) (stdout, stderr string, err error)
```

### 5.2 Command Tests

| Test File | Tests |
|-----------|-------|
| `init_test.go` | Init creates database, tables exist |
| `seed_test.go` | Seed creates sample data |
| `serve_test.go` | Serve starts server (quick check) |
| `user_test.go` | CRUD user operations |
| `board_test.go` | CRUD board operations |
| `stats_test.go` | Stats output format |

---

## File Structure After Implementation

```
forum/
├── Makefile                        # NEW
├── cli/
│   ├── root.go                     # ENHANCED (Fang)
│   ├── serve.go                    # ENHANCED (UI)
│   ├── init.go                     # ENHANCED (UI)
│   ├── seed.go                     # ENHANCED (UI)
│   ├── user.go                     # NEW
│   ├── board.go                    # NEW
│   ├── stats.go                    # NEW
│   ├── ui.go                       # NEW
│   ├── init_test.go                # NEW
│   ├── seed_test.go                # NEW
│   ├── serve_test.go               # NEW
│   ├── user_test.go                # NEW
│   ├── board_test.go               # NEW
│   └── stats_test.go               # NEW
├── store/duckdb/
│   ├── store_test.go               # NEW (shared helpers)
│   ├── accounts_store_test.go      # NEW
│   ├── boards_store_test.go        # NEW
│   ├── threads_store_test.go       # NEW
│   ├── comments_store_test.go      # NEW
│   ├── votes_store_test.go         # NEW
│   ├── bookmarks_store_test.go     # NEW
│   └── notifications_store_test.go # NEW
└── app/web/
    └── server_e2e_test.go          # NEW
```

---

## Testing Commands

```bash
# Run all store tests
make test-store

# Run specific store test
go test -v ./store/duckdb -run TestAccountsStore

# Run e2e tests (requires data)
make test-e2e

# Run CLI tests
make test-cli

# Run all tests
make test
```

---

## Dependencies to Add

```go
// go.mod additions
require (
    github.com/charmbracelet/fang v0.x.x
    github.com/charmbracelet/lipgloss v0.x.x
    github.com/duckdb/duckdb-go/v2 v1.x.x  // For in-memory testing
)
```

---

## Implementation Order

1. **Makefile** - Foundation for all other work
2. **Store test helpers** - Shared infrastructure
3. **AccountsStore tests** - Core authentication
4. **BoardsStore tests** - Core community structure
5. **ThreadsStore tests** - Main content
6. **CommentsStore tests** - Nested replies
7. **VotesStore tests** - Voting mechanics
8. **BookmarksStore tests** - User preferences
9. **NotificationsStore tests** - User alerts
10. **E2E tests** - Full integration
11. **CLI UI module** - Fang/Lipgloss setup
12. **CLI enhancements** - Improved commands
13. **CLI tests** - Command validation

---

## Success Criteria

- [ ] All store tests pass with `make test-store`
- [ ] E2E tests pass with `make test-e2e`
- [ ] CLI tests pass with `make test-cli`
- [ ] CLI shows styled output with spinners
- [ ] All commands work: `init`, `seed`, `serve`, `user`, `board`, `stats`
- [ ] No regressions in existing functionality
