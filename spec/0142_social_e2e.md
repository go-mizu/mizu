# Spec 0142: Social App TikTok Redesign & E2E Test Suite

## Overview

This specification covers the complete redesign of the social app from Twitter/X-style to TikTok-style, implementing a light theme with dark mode toggle, and creating a comprehensive E2E test suite to verify all user flows.

## Current State

- **Theme**: Dark theme only
- **Design**: Twitter/X-style 3-column layout
- **Tests**: Unit tests for stores, basic server tests
- **Functionality**: Full-featured but untested end-to-end

## Goals

1. **Theme System**: Light theme default with dark mode toggle (persisted)
2. **TikTok-Style Redesign**: Vertical content-first feed, side action buttons, bottom navigation
3. **E2E Test Suite**: Complete coverage of all user flows with real data
4. **Production Ready**: All features verified working end-to-end

---

## Part 1: Theme System

### 1.1 Light Theme (Default)

```css
/* Light Theme Colors */
--bg-primary: #ffffff;
--bg-secondary: #f5f5f5;
--bg-tertiary: #e5e5e5;
--text-primary: #0a0a0a;
--text-secondary: #525252;
--text-tertiary: #737373;
--border-color: #e5e5e5;
--accent-primary: #fe2c55;  /* TikTok red */
--accent-secondary: #25f4ee; /* TikTok cyan */
```

### 1.2 Dark Theme (Toggle)

```css
/* Dark Theme Colors */
--bg-primary: #000000;
--bg-secondary: #121212;
--bg-tertiary: #1f1f1f;
--text-primary: #ffffff;
--text-secondary: #a3a3a3;
--text-tertiary: #737373;
--border-color: #2a2a2a;
```

### 1.3 Theme Toggle Implementation

- Store preference in `localStorage`
- Toggle button in header/settings
- CSS custom properties for seamless switching
- Respect `prefers-color-scheme` on first visit

---

## Part 2: TikTok-Style Redesign

### 2.1 Design Principles

1. **Content-First**: Maximize content visibility, minimize chrome
2. **Vertical Scroll**: Full-height content cards for immersive experience
3. **Side Actions**: Like/comment/share buttons on right side of content
4. **Bottom Navigation**: Mobile-style nav for all screen sizes
5. **Creator Focus**: Prominent creator info, easy follow actions
6. **Clean Typography**: Bold headers, readable body text

### 2.2 Layout Structure

```
┌─────────────────────────────────────────────────┐
│ Header: Logo, Search, Theme Toggle, User Menu   │
├────────────────────┬────────────────────────────┤
│                    │                            │
│   For You │        │    ┌─────────┐             │
│   Following        │    │ Avatar  │             │
│                    │    │ @user   │             │
│                    │    │ Follow  │             │
│  ┌─────────────┐   │    ├─────────┤             │
│  │             │   │    │ ❤ 1.2k  │             │
│  │   Content   │   │    │ 💬 234  │             │
│  │   Card      │   │    │ ↗ Share │             │
│  │             │   │    │ ★ Save  │             │
│  └─────────────┘   │    └─────────┘             │
│                    │                            │
│  Caption & tags    │                            │
│                    │                            │
├────────────────────┴────────────────────────────┤
│ Bottom Nav: Home | Explore | Create | Inbox | Me│
└─────────────────────────────────────────────────┘
```

### 2.3 Page Redesigns

#### Home Page (`/`)
- **For You / Following** tab switcher at top
- Full-width content cards with large text/media area
- Side action bar (like, comment, share, bookmark)
- Creator info overlay at bottom of card
- Floating compose button (+ icon)

#### Explore Page (`/explore`)
- Search bar prominent at top
- Trending hashtags as horizontal scroll chips
- Grid of trending posts (2-3 columns)
- Category filters (optional)

#### Profile Page (`/u/:username`)
- Large header with avatar centered
- Stats row (posts, followers, following)
- Bio section
- Content grid (3 columns, square thumbnails)
- Tab bar: Posts | Likes | Saved

#### Post Detail (`/u/:username/post/:id`)
- Full content display
- Thread/replies below
- Side action bar
- Related posts suggestion

#### Notifications (`/notifications`)
- Clean list with avatar, action, preview
- Tab filters: All | Mentions | Likes | Follows

#### Settings (`/settings`)
- Clean form sections
- Theme toggle prominent
- Account management

### 2.4 Components

1. **ContentCard**: Full-width post display with overlay creator info
2. **ActionBar**: Vertical side bar with interaction buttons
3. **BottomNav**: Fixed bottom navigation with icons
4. **TabSwitcher**: For You / Following tabs
5. **UserChip**: Small avatar + username for mentions
6. **TrendingChip**: Hashtag with post count
7. **ProfileHeader**: Centered avatar, stats, bio
8. **ContentGrid**: 3-column square thumbnail grid
9. **ThemeToggle**: Sun/Moon icon button

---

## Part 3: E2E Test Suite

### 3.1 Test Infrastructure

```go
// e2e/helpers.go
type E2ETestSuite struct {
    server   *httptest.Server
    client   *http.Client
    db       *duckdb.Store
    cleanup  func()
}

func NewE2ETestSuite(t *testing.T) *E2ETestSuite
func (s *E2ETestSuite) CreateUser(username, email, password string) *Account
func (s *E2ETestSuite) LoginAs(username, password string) *http.Cookie
func (s *E2ETestSuite) CreatePost(cookie *http.Cookie, content string) *Post
func (s *E2ETestSuite) GET(path string, cookie *http.Cookie) *http.Response
func (s *E2ETestSuite) POST(path string, body any, cookie *http.Cookie) *http.Response
func (s *E2ETestSuite) AssertHTML(resp *http.Response, selector string, contains string)
func (s *E2ETestSuite) AssertJSON(resp *http.Response, path string, expected any)
```

### 3.2 Test Seed Data

```go
// e2e/seed.go
type SeedData struct {
    Users    []*Account  // 10 users with varied profiles
    Posts    []*Post     // 100 posts with hashtags, mentions
    Follows  []Follow    // Social graph connections
    Likes    []Like      // Interaction data
    Reposts  []Repost
    Lists    []*List     // Curated lists
}

func GenerateSeedData() *SeedData
func (s *SeedData) LoadInto(db *Store)
```

### 3.3 Test Categories

#### Authentication Tests (`e2e/auth_test.go`)
```go
func TestE2E_Registration_Success(t *testing.T)
func TestE2E_Registration_DuplicateUsername(t *testing.T)
func TestE2E_Registration_InvalidEmail(t *testing.T)
func TestE2E_Login_Success(t *testing.T)
func TestE2E_Login_WrongPassword(t *testing.T)
func TestE2E_Login_NonexistentUser(t *testing.T)
func TestE2E_Logout_ClearsSession(t *testing.T)
func TestE2E_SessionPersistence(t *testing.T)
func TestE2E_ProtectedRoutes_RequireAuth(t *testing.T)
```

#### Post Tests (`e2e/posts_test.go`)
```go
func TestE2E_CreatePost_Success(t *testing.T)
func TestE2E_CreatePost_WithHashtags(t *testing.T)
func TestE2E_CreatePost_WithMentions(t *testing.T)
func TestE2E_CreatePost_CharacterLimit(t *testing.T)
func TestE2E_EditPost_Success(t *testing.T)
func TestE2E_DeletePost_Success(t *testing.T)
func TestE2E_DeletePost_OnlyOwner(t *testing.T)
func TestE2E_ViewPost_SinglePost(t *testing.T)
func TestE2E_ViewPost_WithReplies(t *testing.T)
func TestE2E_ReplyToPost_Success(t *testing.T)
func TestE2E_QuotePost_Success(t *testing.T)
```

#### Timeline Tests (`e2e/timeline_test.go`)
```go
func TestE2E_HomeTimeline_ShowsFollowedPosts(t *testing.T)
func TestE2E_HomeTimeline_ExcludesUnfollowed(t *testing.T)
func TestE2E_HomeTimeline_IncludesReposts(t *testing.T)
func TestE2E_PublicTimeline_ShowsAll(t *testing.T)
func TestE2E_UserTimeline_ShowsUserPosts(t *testing.T)
func TestE2E_HashtagTimeline_FiltersByTag(t *testing.T)
func TestE2E_ListTimeline_ShowsListMembers(t *testing.T)
func TestE2E_Timeline_Pagination(t *testing.T)
```

#### Interaction Tests (`e2e/interactions_test.go`)
```go
func TestE2E_LikePost_Success(t *testing.T)
func TestE2E_UnlikePost_Success(t *testing.T)
func TestE2E_LikePost_UpdatesCount(t *testing.T)
func TestE2E_RepostPost_Success(t *testing.T)
func TestE2E_UnrepostPost_Success(t *testing.T)
func TestE2E_BookmarkPost_Success(t *testing.T)
func TestE2E_UnbookmarkPost_Success(t *testing.T)
func TestE2E_Bookmarks_ShowsBookmarked(t *testing.T)
func TestE2E_WhoLiked_ShowsLikers(t *testing.T)
func TestE2E_WhoReposted_ShowsReposters(t *testing.T)
```

#### Relationship Tests (`e2e/relationships_test.go`)
```go
func TestE2E_Follow_Success(t *testing.T)
func TestE2E_Unfollow_Success(t *testing.T)
func TestE2E_Follow_UpdatesCounts(t *testing.T)
func TestE2E_Block_HidesContent(t *testing.T)
func TestE2E_Unblock_Success(t *testing.T)
func TestE2E_Mute_HidesFromTimeline(t *testing.T)
func TestE2E_Unmute_Success(t *testing.T)
func TestE2E_Followers_ListsFollowers(t *testing.T)
func TestE2E_Following_ListsFollowing(t *testing.T)
func TestE2E_RelationshipStatus_Accurate(t *testing.T)
```

#### Notification Tests (`e2e/notifications_test.go`)
```go
func TestE2E_Notification_OnFollow(t *testing.T)
func TestE2E_Notification_OnLike(t *testing.T)
func TestE2E_Notification_OnRepost(t *testing.T)
func TestE2E_Notification_OnMention(t *testing.T)
func TestE2E_Notification_OnReply(t *testing.T)
func TestE2E_Notification_MarkRead(t *testing.T)
func TestE2E_Notification_ClearAll(t *testing.T)
func TestE2E_Notification_UnreadCount(t *testing.T)
```

#### Search Tests (`e2e/search_test.go`)
```go
func TestE2E_Search_Posts(t *testing.T)
func TestE2E_Search_Accounts(t *testing.T)
func TestE2E_Search_Hashtags(t *testing.T)
func TestE2E_Search_EmptyResults(t *testing.T)
func TestE2E_Trending_Tags(t *testing.T)
func TestE2E_Trending_Posts(t *testing.T)
```

#### List Tests (`e2e/lists_test.go`)
```go
func TestE2E_CreateList_Success(t *testing.T)
func TestE2E_UpdateList_Success(t *testing.T)
func TestE2E_DeleteList_Success(t *testing.T)
func TestE2E_AddMember_Success(t *testing.T)
func TestE2E_RemoveMember_Success(t *testing.T)
func TestE2E_ListTimeline_ShowsMemberPosts(t *testing.T)
```

#### Profile Tests (`e2e/profile_test.go`)
```go
func TestE2E_ViewProfile_Success(t *testing.T)
func TestE2E_UpdateProfile_Bio(t *testing.T)
func TestE2E_UpdateProfile_DisplayName(t *testing.T)
func TestE2E_UpdateProfile_Avatar(t *testing.T)
func TestE2E_Profile_ShowsStats(t *testing.T)
func TestE2E_Profile_ShowsPosts(t *testing.T)
```

#### UI Flow Tests (`e2e/ui_flows_test.go`)
```go
func TestE2E_Flow_NewUserOnboarding(t *testing.T)
func TestE2E_Flow_CreateAndSharePost(t *testing.T)
func TestE2E_Flow_DiscoverAndFollow(t *testing.T)
func TestE2E_Flow_EngageWithContent(t *testing.T)
func TestE2E_Flow_ManageLists(t *testing.T)
func TestE2E_Flow_SearchAndExplore(t *testing.T)
```

### 3.4 Test Execution

```bash
# Run all E2E tests
E2E_TEST=1 go test ./e2e/... -v

# Run specific test category
E2E_TEST=1 go test ./e2e/... -run TestE2E_Auth -v

# Run with coverage
E2E_TEST=1 go test ./e2e/... -coverprofile=coverage.out

# Run with seed data only (for manual testing)
go run ./cmd/social seed --users 20 --posts 200
```

---

## Part 4: Implementation Checklist

### Phase 1: Theme System
- [ ] Add CSS custom properties for theming
- [ ] Implement light theme as default
- [ ] Keep dark theme accessible via toggle
- [ ] Add theme toggle component to header
- [ ] Persist theme preference in localStorage
- [ ] Respect `prefers-color-scheme` on first visit

### Phase 2: Layout Redesign
- [ ] Redesign base layout template
- [ ] Implement bottom navigation component
- [ ] Create vertical action bar component
- [ ] Update header with search and theme toggle
- [ ] Remove 3-column Twitter layout
- [ ] Implement content-first single column

### Phase 3: Page Redesigns
- [ ] Home page with For You / Following tabs
- [ ] Explore page with search and trending grid
- [ ] Profile page with centered header and content grid
- [ ] Post detail page with replies
- [ ] Notifications page
- [ ] Bookmarks page
- [ ] Lists pages
- [ ] Settings page
- [ ] Login/Register pages
- [ ] 404 page

### Phase 4: E2E Test Infrastructure
- [ ] Create e2e/ directory structure
- [ ] Implement test helpers
- [ ] Create seed data generator
- [ ] Set up test database isolation

### Phase 5: E2E Test Implementation
- [ ] Auth tests (registration, login, logout, sessions)
- [ ] Post tests (CRUD, hashtags, mentions)
- [ ] Timeline tests (home, public, user, hashtag, list)
- [ ] Interaction tests (like, repost, bookmark)
- [ ] Relationship tests (follow, block, mute)
- [ ] Notification tests
- [ ] Search tests
- [ ] List tests
- [ ] Profile tests
- [ ] UI flow tests

### Phase 6: Verification
- [ ] All E2E tests pass
- [ ] Manual testing of all flows
- [ ] Responsive design verification
- [ ] Theme toggle works correctly
- [ ] No regressions in functionality

---

## Appendix: File Changes

### New Files
- `e2e/helpers_test.go` - Test infrastructure
- `e2e/seed_test.go` - Seed data generator
- `e2e/auth_test.go` - Auth tests
- `e2e/posts_test.go` - Post tests
- `e2e/timeline_test.go` - Timeline tests
- `e2e/interactions_test.go` - Interaction tests
- `e2e/relationships_test.go` - Relationship tests
- `e2e/notifications_test.go` - Notification tests
- `e2e/search_test.go` - Search tests
- `e2e/lists_test.go` - List tests
- `e2e/profile_test.go` - Profile tests
- `e2e/ui_flows_test.go` - UI flow tests

### Modified Files
- `assets/static/css/style.css` - Complete redesign
- `assets/static/js/app.js` - Theme toggle, new interactions
- `assets/views/default/layouts/base.html` - New layout structure
- `assets/views/default/pages/*.html` - All page templates
- `assets/views/default/components/*.html` - New components

### Deleted Files
- Previous page templates (replaced with new designs)
