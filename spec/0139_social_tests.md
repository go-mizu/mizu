# 0139: Social Blueprint Comprehensive Tests

## Overview

This specification defines comprehensive tests for the social blueprint covering:
1. **Store Tests** - Unit tests for all DuckDB store implementations (real database, not mocks)
2. **E2E Tests** - End-to-end HTTP API tests for the web server
3. **CLI Tests** - Tests for all CLI commands

## Test Structure

### 1. Store Tests (`store/duckdb/*_test.go`)

#### 1.1 Core Store Test (`store_test.go`)

Test helpers and utilities:
- `setupTestStore(t)` - Creates in-memory DuckDB for testing
- `newTestID()` - Generates ULID for test data
- `testTime()` - Returns fixed time for deterministic tests
- `ptr[T](v T) *T` - Generic pointer helper

#### 1.2 Accounts Store (`accounts_store_test.go`)

| Test | Description |
|------|-------------|
| `TestAccountsStore_Insert` | Insert new account and verify all fields |
| `TestAccountsStore_Insert_DuplicateUsername` | Error on duplicate username |
| `TestAccountsStore_Insert_DuplicateEmail` | Error on duplicate email |
| `TestAccountsStore_GetByID` | Retrieve account by ID |
| `TestAccountsStore_GetByID_NotFound` | ErrNotFound for missing account |
| `TestAccountsStore_GetByIDs` | Batch retrieve multiple accounts |
| `TestAccountsStore_GetByUsername` | Case-insensitive username lookup |
| `TestAccountsStore_GetByEmail` | Case-insensitive email lookup |
| `TestAccountsStore_Update` | Partial update with dynamic fields |
| `TestAccountsStore_ExistsUsername` | Check username availability |
| `TestAccountsStore_ExistsEmail` | Check email availability |
| `TestAccountsStore_GetPasswordHash` | Retrieve password hash for auth |
| `TestAccountsStore_List` | Paginated list with offset/limit |
| `TestAccountsStore_Search` | Search by username/display name |
| `TestAccountsStore_SetVerified` | Toggle verified status |
| `TestAccountsStore_SetSuspended` | Toggle suspended status |
| `TestAccountsStore_SetAdmin` | Toggle admin status |
| `TestAccountsStore_GetFollowersCount` | Count followers |
| `TestAccountsStore_GetFollowingCount` | Count following |
| `TestAccountsStore_GetPostsCount` | Count posts |
| `TestAccountsStore_Session_CRUD` | Create, get, delete session |
| `TestAccountsStore_DeleteExpiredSessions` | Clean up expired sessions |

#### 1.3 Posts Store (`posts_store_test.go`)

| Test | Description |
|------|-------------|
| `TestPostsStore_Insert` | Insert post with all fields |
| `TestPostsStore_GetByID` | Retrieve post by ID |
| `TestPostsStore_GetByID_NotFound` | ErrNotFound for missing post |
| `TestPostsStore_GetByIDs` | Batch retrieve multiple posts |
| `TestPostsStore_Update` | Update content, warning, sensitivity |
| `TestPostsStore_Delete` | Delete post |
| `TestPostsStore_List` | List with filters (accountID, excludeReplies) |
| `TestPostsStore_GetReplies` | Get replies to a post |
| `TestPostsStore_GetAncestors` | Get thread ancestors |
| `TestPostsStore_GetDescendants` | Get thread descendants |
| `TestPostsStore_IncrementRepliesCount` | Increment replies counter |
| `TestPostsStore_DecrementRepliesCount` | Decrement replies counter |
| `TestPostsStore_IncrementQuotesCount` | Increment quotes counter |
| `TestPostsStore_Media_CRUD` | Insert, get, delete media |
| `TestPostsStore_Hashtags` | Upsert hashtag, link to post, get |
| `TestPostsStore_Mentions` | Insert and get mentions |
| `TestPostsStore_EditHistory` | Insert edit history |

#### 1.4 Interactions Store (`interactions_store_test.go`)

| Test | Description |
|------|-------------|
| `TestInteractionsStore_Like` | Insert like |
| `TestInteractionsStore_Unlike` | Delete like |
| `TestInteractionsStore_ExistsLike` | Check if liked |
| `TestInteractionsStore_GetLikedBy` | Get accounts who liked |
| `TestInteractionsStore_GetLikedPosts` | Get user's liked posts |
| `TestInteractionsStore_LikesCount` | Increment/decrement likes |
| `TestInteractionsStore_Repost` | Insert repost |
| `TestInteractionsStore_Unrepost` | Delete repost |
| `TestInteractionsStore_ExistsRepost` | Check if reposted |
| `TestInteractionsStore_GetRepostedBy` | Get accounts who reposted |
| `TestInteractionsStore_RepostsCount` | Increment/decrement reposts |
| `TestInteractionsStore_Bookmark` | Insert bookmark |
| `TestInteractionsStore_Unbookmark` | Delete bookmark |
| `TestInteractionsStore_ExistsBookmark` | Check if bookmarked |
| `TestInteractionsStore_GetBookmarkedPosts` | Get user's bookmarks |
| `TestInteractionsStore_GetPostState` | Get viewer's interaction state |
| `TestInteractionsStore_GetPostStates` | Batch get states |

#### 1.5 Relationships Store (`relationships_store_test.go`)

| Test | Description |
|------|-------------|
| `TestRelationshipsStore_InsertFollow` | Create follow |
| `TestRelationshipsStore_DeleteFollow` | Remove follow |
| `TestRelationshipsStore_GetFollow` | Get follow record |
| `TestRelationshipsStore_SetFollowPending` | Toggle pending status |
| `TestRelationshipsStore_GetFollowers` | List followers |
| `TestRelationshipsStore_GetFollowing` | List following |
| `TestRelationshipsStore_GetPendingFollowers` | List pending requests |
| `TestRelationshipsStore_ExistsFollow` | Check if following |
| `TestRelationshipsStore_InsertBlock` | Create block |
| `TestRelationshipsStore_DeleteBlock` | Remove block |
| `TestRelationshipsStore_GetBlocks` | List blocked accounts |
| `TestRelationshipsStore_ExistsBlock` | Check if blocking |
| `TestRelationshipsStore_ExistsBlockEither` | Bidirectional block check |
| `TestRelationshipsStore_InsertMute` | Create mute with expiration |
| `TestRelationshipsStore_DeleteMute` | Remove mute |
| `TestRelationshipsStore_GetMutes` | List muted accounts |
| `TestRelationshipsStore_ExistsMute` | Check if muting |
| `TestRelationshipsStore_GetRelationship` | Get full relationship state |

#### 1.6 Lists Store (`lists_store_test.go`)

| Test | Description |
|------|-------------|
| `TestListsStore_Insert` | Create list |
| `TestListsStore_GetByID` | Retrieve list by ID |
| `TestListsStore_GetByAccount` | Get user's lists |
| `TestListsStore_Update` | Update list title |
| `TestListsStore_Delete` | Delete list (cascades) |
| `TestListsStore_InsertMember` | Add member to list |
| `TestListsStore_DeleteMember` | Remove member |
| `TestListsStore_GetMembers` | List members |
| `TestListsStore_ExistsMember` | Check membership |
| `TestListsStore_GetMemberCount` | Count members |
| `TestListsStore_GetListsContaining` | Find lists containing account |

#### 1.7 Notifications Store (`notifications_store_test.go`)

| Test | Description |
|------|-------------|
| `TestNotificationsStore_Insert` | Create notification |
| `TestNotificationsStore_GetByID` | Retrieve notification |
| `TestNotificationsStore_List` | List with filters (types, exclude) |
| `TestNotificationsStore_List_Pagination` | maxID/sinceID pagination |
| `TestNotificationsStore_MarkRead` | Mark single as read |
| `TestNotificationsStore_MarkAllRead` | Mark all as read |
| `TestNotificationsStore_Delete` | Delete notification |
| `TestNotificationsStore_DeleteAll` | Delete all for account |
| `TestNotificationsStore_UnreadCount` | Count unread |
| `TestNotificationsStore_Exists` | Deduplication check |

#### 1.8 Timelines Store (`timelines_store_test.go`)

| Test | Description |
|------|-------------|
| `TestTimelinesStore_GetHomeFeed` | Posts from self + following |
| `TestTimelinesStore_GetHomeFeed_Pagination` | maxID/minID pagination |
| `TestTimelinesStore_GetPublicFeed` | Public posts only |
| `TestTimelinesStore_GetPublicFeed_OnlyMedia` | Filter media posts |
| `TestTimelinesStore_GetUserFeed` | User's public posts |
| `TestTimelinesStore_GetUserFeed_IncludeReplies` | Include replies |
| `TestTimelinesStore_GetHashtagFeed` | Posts with hashtag |
| `TestTimelinesStore_GetListFeed` | Posts from list members |
| `TestTimelinesStore_GetBookmarksFeed` | User's bookmarks |
| `TestTimelinesStore_GetLikesFeed` | User's liked posts |

#### 1.9 Search Store (`search_store_test.go`)

| Test | Description |
|------|-------------|
| `TestSearchStore_SearchAccounts` | Search discoverable accounts |
| `TestSearchStore_SearchAccounts_ExcludesSuspended` | Excludes suspended |
| `TestSearchStore_SearchPosts` | Full-text post search |
| `TestSearchStore_SearchPosts_Filters` | minLikes, minReposts, hasMedia |
| `TestSearchStore_SearchHashtags` | Hashtag search |
| `TestSearchStore_SuggestHashtags` | Autocomplete prefix |

#### 1.10 Trending Store (`trending_store_test.go`)

| Test | Description |
|------|-------------|
| `TestTrendingStore_GetTrendingTags` | Tags by post count |
| `TestTrendingStore_GetTrendingPosts` | Posts by engagement |
| `TestTrendingStore_ComputeTrendingTags` | Compute with time window |
| `TestTrendingStore_ComputeTrendingPosts` | Compute with time window |

### 2. E2E Tests (`app/web/server_e2e_test.go`)

Build tag: `//go:build e2e`
Skip condition: `E2E_TEST=1`

#### Test Helpers
- `setupTestServer(t)` - Create test server with temp database
- `createTestUser(t, store, username)` - Create user via service
- `loginUser(t, ts, username, password)` - Login and get token
- `authRequest(t, method, url, token, body)` - Authenticated request
- `get(t, url)` - Simple GET request
- `assertStatus(t, resp, want)` - Status code assertion
- `assertContains(t, body, substr)` - Body content assertion

#### Test Groups

| Group | Tests |
|-------|-------|
| **Auth** | Register, Login, Login_InvalidPassword, Logout |
| **Accounts** | VerifyCredentials, UpdateCredentials, GetAccount, GetAccountByUsername, GetAccountPosts, GetAccountFollowers, GetAccountFollowing, Search |
| **Posts** | Create, Get, Update, Delete, GetContext |
| **Interactions** | Like, Unlike, Repost, Unrepost, Bookmark, Unbookmark, LikedBy, RepostedBy |
| **Relationships** | Follow, Unfollow, Block, Unblock, Mute, Unmute, GetPendingFollowers, AcceptFollow, RejectFollow |
| **Timelines** | Home, Public, Hashtag, List, Bookmarks |
| **Notifications** | List, UnreadCount, Clear, Dismiss |
| **Search** | Search, TrendingTags, TrendingPosts |
| **Lists** | Create, Get, Update, Delete, GetMembers, AddMember, RemoveMember |
| **HTMLPages** | Home, Login, Register, Explore, Search, Notifications, Bookmarks, Profile, Post, Followers, Following, Tags, Settings |
| **UserJourney** | Complete user flow from register to logout |
| **Unauthorized** | All protected endpoints return 401 |

### 3. CLI Tests (`cli/*_test.go`)

#### 3.1 Root Tests (`root_test.go`)

| Test | Description |
|------|-------------|
| `TestVersionVariable` | Default version is set |
| `TestExecute` | Execute doesn't panic |

#### 3.2 Init Tests (`init_test.go`)

| Test | Description |
|------|-------------|
| `TestInitCmd_Use` | Command name is "init" |
| `TestInitCmd_Short` | Has description |
| `TestRunInit` | Creates database file |
| `TestRunInit_InvalidPath` | Errors on invalid path |

#### 3.3 Serve Tests (`serve_test.go`)

| Test | Description |
|------|-------------|
| `TestServeCmd_Use` | Command name is "serve" |
| `TestServeCmd_Short` | Has description |
| `TestRunServe_InvalidPath` | Errors on invalid path |
| `TestRunServe_ContextCancel` | Graceful shutdown on cancel |

#### 3.4 Seed Tests (`seed_test.go`)

| Test | Description |
|------|-------------|
| `TestSeedCmd_Use` | Command name is "seed" |
| `TestSeedCmd_Short` | Has description |
| `TestSeedCmd_Flags` | Has --users and --posts flags |
| `TestRunSeed` | Creates users and posts |
| `TestRunSeed_InvalidPath` | Errors on invalid path |

## Running Tests

```bash
# Run store tests
go test ./store/duckdb/...

# Run E2E tests
E2E_TEST=1 go test -tags=e2e ./app/web/...

# Run CLI tests
go test ./cli/...

# Run all tests
E2E_TEST=1 go test -tags=e2e ./...
```

## Implementation Notes

1. **In-memory DuckDB**: Store tests use `sql.Open("duckdb", "")` for in-memory database
2. **Test isolation**: Each test gets fresh database via `setupTestStore(t)`
3. **Cleanup**: Use `t.Cleanup()` for resource cleanup
4. **Helpers**: Use `t.Helper()` in helper functions for better error locations
5. **Assertions**: Use `t.Errorf` for soft failures, `t.Fatalf` for setup failures
6. **E2E skip**: E2E tests skip unless `E2E_TEST=1` environment variable is set
7. **HTTP testing**: Use `httptest.NewServer` for E2E tests
