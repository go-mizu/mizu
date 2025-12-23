# Forum V2 Specification

**Version:** 2.0
**Status:** Implementation
**Author:** Mizu Team
**Created:** 2025-01-26

## 1. Overview

Forum V2 is a modern discussion platform inspired by Reddit and Discourse. It provides a full-featured community platform with boards (subreddit-like communities), threaded discussions, nested comments, voting, and comprehensive moderation tools.

### 1.1 Goals

1. **Scalable Architecture** - Handle large communities with millions of posts
2. **Great UX** - Fast, responsive, intuitive interface
3. **Moderation First** - Robust tools for community management
4. **Extensible** - Easy to add new features and integrations
5. **Privacy Conscious** - Minimal data collection, user control

### 1.2 Non-Goals

1. Real-time chat (separate feature)
2. Private messaging (v3)
3. Federation (future consideration)
4. Mobile apps (web-first)

## 2. System Architecture

### 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Client                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Browser │  │   API    │  │  Mobile  │  │   Bot    │    │
│  │   (SSR)  │  │  Client  │  │   Web    │  │  Client  │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
└───────┼─────────────┼─────────────┼─────────────┼───────────┘
        │             │             │             │
        └─────────────┼─────────────┼─────────────┘
                      │             │
┌─────────────────────┼─────────────┼─────────────────────────┐
│                     ▼             ▼                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   HTTP Server (Mizu)                   │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │  Middleware │  │   Handlers   │  │   Templates  │  │  │
│  │  │  (Auth, etc)│  │  (REST API)  │  │   (SSR)      │  │  │
│  │  └─────────────┘  └──────────────┘  └──────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                              │                               │
│  ┌───────────────────────────┼───────────────────────────┐  │
│  │                   Service Layer                        │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐      │  │
│  │  │Accounts │ │ Boards  │ │Threads  │ │Comments │      │  │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘      │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐      │  │
│  │  │  Votes  │ │Bookmarks│ │ Search  │ │Moderation      │  │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘      │  │
│  └───────────────────────────────────────────────────────┘  │
│                              │                               │
│  ┌───────────────────────────┼───────────────────────────┐  │
│  │                   Data Layer                           │  │
│  │  ┌─────────────────────────────────────────────────┐  │  │
│  │  │                   DuckDB                         │  │  │
│  │  │  (Accounts, Boards, Threads, Comments, Votes)   │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│                        Forum Server                          │
└──────────────────────────────────────────────────────────────┘
```

### 2.2 Directory Structure

```
forum/
├── cmd/forum/
│   └── main.go              # Entry point
├── cli/
│   ├── root.go              # Root command
│   ├── serve.go             # Serve command
│   ├── init.go              # Init command
│   └── seed.go              # Seed command
├── app/web/
│   ├── server.go            # Server orchestration
│   ├── routes.go            # Route definitions
│   ├── middleware.go        # Middleware definitions
│   ├── context.go           # Request context helpers
│   └── handler/
│       ├── auth.go          # Auth handlers
│       ├── board.go         # Board handlers
│       ├── thread.go        # Thread handlers
│       ├── comment.go       # Comment handlers
│       ├── vote.go          # Vote handlers
│       ├── user.go          # User handlers
│       ├── search.go        # Search handlers
│       ├── mod.go           # Moderation handlers
│       ├── page.go          # HTML page handlers
│       └── response.go      # Response helpers
├── feature/
│   ├── accounts/
│   │   ├── api.go           # Types and interface
│   │   └── service.go       # Business logic
│   ├── boards/
│   │   ├── api.go
│   │   └── service.go
│   ├── threads/
│   │   ├── api.go
│   │   └── service.go
│   ├── comments/
│   │   ├── api.go
│   │   └── service.go
│   ├── votes/
│   │   ├── api.go
│   │   └── service.go
│   ├── bookmarks/
│   │   ├── api.go
│   │   └── service.go
│   ├── notifications/
│   │   ├── api.go
│   │   └── service.go
│   ├── search/
│   │   ├── api.go
│   │   └── service.go
│   ├── moderation/
│   │   ├── api.go
│   │   └── service.go
│   └── tags/
│       ├── api.go
│       └── service.go
├── store/duckdb/
│   ├── store.go             # Core store
│   ├── schema.sql           # Database schema
│   ├── accounts_store.go
│   ├── boards_store.go
│   ├── threads_store.go
│   ├── comments_store.go
│   ├── votes_store.go
│   ├── bookmarks_store.go
│   ├── notifications_store.go
│   ├── search_store.go
│   └── moderation_store.go
├── assets/
│   ├── embed.go             # Embed directive
│   ├── static/
│   │   ├── css/
│   │   │   ├── app.css      # Main styles
│   │   │   └── components/  # Component styles
│   │   ├── js/
│   │   │   └── app.js       # Client JS
│   │   └── icons/           # SVG icons
│   └── views/
│       ├── layouts/
│       │   └── default.html
│       ├── pages/
│       │   ├── home.html
│       │   ├── board.html
│       │   ├── thread.html
│       │   ├── submit.html
│       │   ├── user.html
│       │   ├── search.html
│       │   ├── login.html
│       │   ├── register.html
│       │   ├── settings.html
│       │   ├── bookmarks.html
│       │   ├── notifications.html
│       │   └── mod/
│       │       ├── dashboard.html
│       │       ├── queue.html
│       │       └── log.html
│       └── components/
│           ├── thread_card.html
│           ├── comment.html
│           ├── vote_buttons.html
│           ├── board_sidebar.html
│           ├── user_card.html
│           ├── nav.html
│           ├── pagination.html
│           └── sort_tabs.html
└── pkg/
    ├── ulid/
    │   └── ulid.go          # ULID generation
    ├── password/
    │   └── password.go      # Password hashing
    ├── text/
    │   └── text.go          # Text utilities
    └── markdown/
        └── markdown.go      # Markdown rendering
```

## 3. Data Models

### 3.1 Account

```go
package accounts

import "time"

type Account struct {
    ID            string    `json:"id"`
    Username      string    `json:"username"`
    Email         string    `json:"email,omitempty"` // Hidden from public
    PasswordHash  string    `json:"-"`               // Never exposed
    DisplayName   string    `json:"display_name"`
    Bio           string    `json:"bio"`
    AvatarURL     string    `json:"avatar_url"`
    BannerURL     string    `json:"banner_url"`
    Karma         int64     `json:"karma"`
    PostKarma     int64     `json:"post_karma"`
    CommentKarma  int64     `json:"comment_karma"`
    IsAdmin       bool      `json:"is_admin"`
    IsModerator   bool      `json:"is_moderator"`   // Mods any board
    IsSuspended   bool      `json:"is_suspended"`
    SuspendReason string    `json:"suspend_reason,omitempty"`
    SuspendUntil  time.Time `json:"suspend_until,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`

    // Computed fields
    ThreadCount   int64     `json:"thread_count,omitempty"`
    CommentCount  int64     `json:"comment_count,omitempty"`
    CakeDay       bool      `json:"cake_day,omitempty"` // Anniversary
}

type Session struct {
    ID        string    `json:"id"`
    AccountID string    `json:"account_id"`
    Token     string    `json:"token"`
    UserAgent string    `json:"user_agent"`
    IP        string    `json:"ip"`
    ExpiresAt time.Time `json:"expires_at"`
    CreatedAt time.Time `json:"created_at"`
}

// Validation constants
const (
    UsernameMinLen = 3
    UsernameMaxLen = 20
    PasswordMinLen = 8
    BioMaxLen      = 500
)

var UsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
```

### 3.2 Board

```go
package boards

import "time"

type Board struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`          // URL slug
    Title         string    `json:"title"`         // Display name
    Description   string    `json:"description"`   // Short description
    Sidebar       string    `json:"sidebar"`       // Full description (Markdown)
    SidebarHTML   string    `json:"sidebar_html"`  // Rendered sidebar
    IconURL       string    `json:"icon_url"`
    BannerURL     string    `json:"banner_url"`
    PrimaryColor  string    `json:"primary_color"` // Hex color
    IsNSFW        bool      `json:"is_nsfw"`
    IsPrivate     bool      `json:"is_private"`
    IsArchived    bool      `json:"is_archived"`   // Read-only
    MemberCount   int64     `json:"member_count"`
    ThreadCount   int64     `json:"thread_count"`
    CreatedAt     time.Time `json:"created_at"`
    CreatedBy     string    `json:"created_by"`
    UpdatedAt     time.Time `json:"updated_at"`

    // Viewer state
    IsJoined      bool      `json:"is_joined,omitempty"`
    IsModerator   bool      `json:"is_moderator,omitempty"`
}

type BoardMember struct {
    BoardID   string    `json:"board_id"`
    AccountID string    `json:"account_id"`
    JoinedAt  time.Time `json:"joined_at"`
}

type BoardModerator struct {
    BoardID     string    `json:"board_id"`
    AccountID   string    `json:"account_id"`
    Permissions ModPerms  `json:"permissions"`
    AddedAt     time.Time `json:"added_at"`
    AddedBy     string    `json:"added_by"`

    // Relationship
    Account     *Account  `json:"account,omitempty"`
}

type ModPerms struct {
    ManagePosts    bool `json:"manage_posts"`
    ManageComments bool `json:"manage_comments"`
    ManageUsers    bool `json:"manage_users"`
    ManageMods     bool `json:"manage_mods"`
    ManageSettings bool `json:"manage_settings"`
}

// Validation constants
const (
    BoardNameMinLen  = 3
    BoardNameMaxLen  = 21
    BoardTitleMaxLen = 100
    BoardDescMaxLen  = 500
    SidebarMaxLen    = 10000
)

var BoardNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]+$`)
```

### 3.3 Thread

```go
package threads

import "time"

type Thread struct {
    ID            string      `json:"id"`
    BoardID       string      `json:"board_id"`
    AuthorID      string      `json:"author_id"`
    Title         string      `json:"title"`
    Content       string      `json:"content"`
    ContentHTML   string      `json:"content_html"`
    URL           string      `json:"url,omitempty"`       // For link posts
    Domain        string      `json:"domain,omitempty"`    // Extracted domain
    ThumbnailURL  string      `json:"thumbnail_url,omitempty"`
    Type          ThreadType  `json:"type"`
    Score         int64       `json:"score"`
    UpvoteCount   int64       `json:"upvote_count"`
    DownvoteCount int64       `json:"downvote_count"`
    CommentCount  int64       `json:"comment_count"`
    ViewCount     int64       `json:"view_count"`
    HotScore      float64     `json:"hot_score"`          // For sorting
    IsPinned      bool        `json:"is_pinned"`
    IsLocked      bool        `json:"is_locked"`
    IsRemoved     bool        `json:"is_removed"`
    IsNSFW        bool        `json:"is_nsfw"`
    IsSpoiler     bool        `json:"is_spoiler"`
    IsOC          bool        `json:"is_oc"`              // Original Content
    RemoveReason  string      `json:"remove_reason,omitempty"`
    CreatedAt     time.Time   `json:"created_at"`
    UpdatedAt     time.Time   `json:"updated_at"`
    EditedAt      *time.Time  `json:"edited_at,omitempty"`

    // Relationships
    Author        *Account    `json:"author,omitempty"`
    Board         *Board      `json:"board,omitempty"`
    Tags          []*Tag      `json:"tags,omitempty"`

    // Viewer state (computed per request)
    Vote          int         `json:"vote,omitempty"`     // -1, 0, 1
    IsBookmarked  bool        `json:"is_bookmarked,omitempty"`
    IsOwner       bool        `json:"is_owner,omitempty"`
    CanEdit       bool        `json:"can_edit,omitempty"`
    CanDelete     bool        `json:"can_delete,omitempty"`
}

type ThreadType string

const (
    ThreadTypeText  ThreadType = "text"
    ThreadTypeLink  ThreadType = "link"
    ThreadTypeImage ThreadType = "image"
    ThreadTypePoll  ThreadType = "poll"
)

// Sorting options
type SortBy string

const (
    SortHot           SortBy = "hot"
    SortNew           SortBy = "new"
    SortTop           SortBy = "top"
    SortRising        SortBy = "rising"
    SortControversial SortBy = "controversial"
)

type TimeRange string

const (
    TimeHour  TimeRange = "hour"
    TimeDay   TimeRange = "day"
    TimeWeek  TimeRange = "week"
    TimeMonth TimeRange = "month"
    TimeYear  TimeRange = "year"
    TimeAll   TimeRange = "all"
)

// Validation
const (
    TitleMinLen   = 1
    TitleMaxLen   = 300
    ContentMaxLen = 40000
    URLMaxLen     = 2000
)
```

### 3.4 Comment

```go
package comments

import "time"

type Comment struct {
    ID            string     `json:"id"`
    ThreadID      string     `json:"thread_id"`
    ParentID      string     `json:"parent_id,omitempty"` // Empty for top-level
    AuthorID      string     `json:"author_id"`
    Content       string     `json:"content"`
    ContentHTML   string     `json:"content_html"`
    Score         int64      `json:"score"`
    UpvoteCount   int64      `json:"upvote_count"`
    DownvoteCount int64      `json:"downvote_count"`
    Depth         int        `json:"depth"`               // 0 = top-level
    Path          string     `json:"path"`                // Materialized path
    ChildCount    int64      `json:"child_count"`
    IsRemoved     bool       `json:"is_removed"`
    IsDeleted     bool       `json:"is_deleted"`          // User-deleted
    RemoveReason  string     `json:"remove_reason,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    EditedAt      *time.Time `json:"edited_at,omitempty"`

    // Relationships
    Author        *Account   `json:"author,omitempty"`
    Parent        *Comment   `json:"parent,omitempty"`
    Children      []*Comment `json:"children,omitempty"`

    // Viewer state
    Vote          int        `json:"vote,omitempty"`
    IsBookmarked  bool       `json:"is_bookmarked,omitempty"`
    IsOwner       bool       `json:"is_owner,omitempty"`
    IsCollapsed   bool       `json:"is_collapsed,omitempty"`
    CanEdit       bool       `json:"can_edit,omitempty"`
    CanDelete     bool       `json:"can_delete,omitempty"`
}

// Materialized path format: /rootID/parentID/thisID
// Enables efficient tree queries with LIKE 'path/%'

// Sort options for comments
type CommentSort string

const (
    CommentSortBest          CommentSort = "best"    // Wilson score
    CommentSortTop           CommentSort = "top"     // Raw score
    CommentSortNew           CommentSort = "new"     // Most recent
    CommentSortOld           CommentSort = "old"     // Oldest first
    CommentSortControversial CommentSort = "controversial"
    CommentSortQA            CommentSort = "qa"      // OP replies first
)

const (
    ContentMaxLen     = 10000
    MaxDepth          = 10       // Max nesting depth
    DefaultCollapseAt = 5        // Auto-collapse at this depth
)
```

### 3.5 Vote

```go
package votes

import "time"

type Vote struct {
    ID         string    `json:"id"`
    AccountID  string    `json:"account_id"`
    TargetType string    `json:"target_type"` // "thread" or "comment"
    TargetID   string    `json:"target_id"`
    Value      int       `json:"value"`       // -1 or 1
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

// Composite key: (account_id, target_type, target_id)
```

### 3.6 Bookmark

```go
package bookmarks

import "time"

type Bookmark struct {
    ID         string    `json:"id"`
    AccountID  string    `json:"account_id"`
    TargetType string    `json:"target_type"` // "thread" or "comment"
    TargetID   string    `json:"target_id"`
    CreatedAt  time.Time `json:"created_at"`
}
```

### 3.7 Notification

```go
package notifications

import "time"

type Notification struct {
    ID         string           `json:"id"`
    AccountID  string           `json:"account_id"`   // Recipient
    Type       NotificationType `json:"type"`
    ActorID    string           `json:"actor_id"`     // Who triggered
    BoardID    string           `json:"board_id,omitempty"`
    ThreadID   string           `json:"thread_id,omitempty"`
    CommentID  string           `json:"comment_id,omitempty"`
    Message    string           `json:"message,omitempty"` // For mod actions
    Read       bool             `json:"read"`
    CreatedAt  time.Time        `json:"created_at"`

    // Relationships
    Actor      *Account         `json:"actor,omitempty"`
    Board      *Board           `json:"board,omitempty"`
    Thread     *Thread          `json:"thread,omitempty"`
    Comment    *Comment         `json:"comment,omitempty"`
}

type NotificationType string

const (
    NotifyReply         NotificationType = "reply"
    NotifyMention       NotificationType = "mention"
    NotifyThreadVote    NotificationType = "thread_vote"
    NotifyCommentVote   NotificationType = "comment_vote"
    NotifyFollow        NotificationType = "follow"
    NotifyMod           NotificationType = "mod"
    NotifyBoardInvite   NotificationType = "board_invite"
    NotifyAward         NotificationType = "award"
)
```

### 3.8 Tag

```go
package tags

import "time"

type Tag struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`       // URL slug
    Label     string    `json:"label"`      // Display name
    Color     string    `json:"color"`      // Hex color
    UseCount  int64     `json:"use_count"`
    CreatedAt time.Time `json:"created_at"`
}

type ThreadTag struct {
    ThreadID string `json:"thread_id"`
    TagID    string `json:"tag_id"`
}
```

### 3.9 Moderation

```go
package moderation

import "time"

type ModAction struct {
    ID          string        `json:"id"`
    BoardID     string        `json:"board_id"`
    ModeratorID string        `json:"moderator_id"`
    TargetType  string        `json:"target_type"` // thread, comment, account
    TargetID    string        `json:"target_id"`
    Action      ModActionType `json:"action"`
    Reason      string        `json:"reason"`
    Details     string        `json:"details"`      // JSON blob for extra data
    CreatedAt   time.Time     `json:"created_at"`

    // Relationships
    Moderator   *Account      `json:"moderator,omitempty"`
    Thread      *Thread       `json:"thread,omitempty"`
    Comment     *Comment      `json:"comment,omitempty"`
    Account     *Account      `json:"account,omitempty"`
}

type ModActionType string

const (
    ActionRemove    ModActionType = "remove"
    ActionApprove   ModActionType = "approve"
    ActionLock      ModActionType = "lock"
    ActionUnlock    ModActionType = "unlock"
    ActionPin       ModActionType = "pin"
    ActionUnpin     ModActionType = "unpin"
    ActionNSFW      ModActionType = "nsfw"
    ActionSpoiler   ModActionType = "spoiler"
    ActionBan       ModActionType = "ban"
    ActionUnban     ModActionType = "unban"
    ActionMute      ModActionType = "mute"
    ActionWarn      ModActionType = "warn"
)

type Ban struct {
    ID        string    `json:"id"`
    BoardID   string    `json:"board_id"` // Empty for site-wide
    AccountID string    `json:"account_id"`
    Reason    string    `json:"reason"`
    Message   string    `json:"message"`  // Message to user
    ModID     string    `json:"mod_id"`
    IsPerm    bool      `json:"is_permanent"`
    ExpiresAt time.Time `json:"expires_at,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}

type Report struct {
    ID         string     `json:"id"`
    ReporterID string     `json:"reporter_id"`
    BoardID    string     `json:"board_id"`
    TargetType string     `json:"target_type"`
    TargetID   string     `json:"target_id"`
    Reason     string     `json:"reason"`
    Details    string     `json:"details"`
    Status     ReportStatus `json:"status"`
    ResolvedBy string     `json:"resolved_by,omitempty"`
    ResolvedAt *time.Time `json:"resolved_at,omitempty"`
    CreatedAt  time.Time  `json:"created_at"`
}

type ReportStatus string

const (
    ReportPending   ReportStatus = "pending"
    ReportResolved  ReportStatus = "resolved"
    ReportDismissed ReportStatus = "dismissed"
)
```

## 4. Service Interfaces

### 4.1 Accounts API

```go
package accounts

import "context"

type API interface {
    // Account management
    Create(ctx context.Context, in CreateIn) (*Account, error)
    GetByID(ctx context.Context, id string) (*Account, error)
    GetByUsername(ctx context.Context, username string) (*Account, error)
    Update(ctx context.Context, id string, in UpdateIn) (*Account, error)
    UpdatePassword(ctx context.Context, id string, current, new string) error
    Delete(ctx context.Context, id string) error

    // Authentication
    Login(ctx context.Context, in LoginIn) (*Account, error)
    CreateSession(ctx context.Context, accountID, userAgent, ip string) (*Session, error)
    GetSession(ctx context.Context, token string) (*Session, error)
    DeleteSession(ctx context.Context, token string) error
    DeleteAllSessions(ctx context.Context, accountID string) error

    // Admin
    Suspend(ctx context.Context, id string, reason string, until time.Time) error
    Unsuspend(ctx context.Context, id string) error
    SetAdmin(ctx context.Context, id string, isAdmin bool) error

    // Lists
    List(ctx context.Context, opts ListOpts) ([]*Account, error)
    Search(ctx context.Context, query string, limit int) ([]*Account, error)
}

type CreateIn struct {
    Username string
    Email    string
    Password string
}

type UpdateIn struct {
    DisplayName *string
    Bio         *string
    AvatarURL   *string
    BannerURL   *string
}

type LoginIn struct {
    Username string
    Password string
}

type ListOpts struct {
    Limit   int
    Cursor  string
    OrderBy string
}
```

### 4.2 Boards API

```go
package boards

import "context"

type API interface {
    // Board management
    Create(ctx context.Context, creatorID string, in CreateIn) (*Board, error)
    GetByName(ctx context.Context, name string) (*Board, error)
    GetByID(ctx context.Context, id string) (*Board, error)
    Update(ctx context.Context, id string, in UpdateIn) (*Board, error)
    Delete(ctx context.Context, id string) error
    Archive(ctx context.Context, id string) error

    // Membership
    Join(ctx context.Context, boardID, accountID string) error
    Leave(ctx context.Context, boardID, accountID string) error
    IsMember(ctx context.Context, boardID, accountID string) (bool, error)
    ListMembers(ctx context.Context, boardID string, opts ListOpts) ([]*Account, error)

    // Moderation
    AddModerator(ctx context.Context, boardID, accountID, addedBy string, perms ModPerms) error
    RemoveModerator(ctx context.Context, boardID, accountID string) error
    IsModerator(ctx context.Context, boardID, accountID string) (bool, error)
    ListModerators(ctx context.Context, boardID string) ([]*BoardModerator, error)

    // Discovery
    List(ctx context.Context, opts ListOpts) ([]*Board, error)
    Search(ctx context.Context, query string, limit int) ([]*Board, error)
    ListPopular(ctx context.Context, limit int) ([]*Board, error)
    ListNew(ctx context.Context, limit int) ([]*Board, error)

    // User's boards
    ListJoined(ctx context.Context, accountID string) ([]*Board, error)
    ListModerated(ctx context.Context, accountID string) ([]*Board, error)
}

type CreateIn struct {
    Name        string
    Title       string
    Description string
    IsNSFW      bool
    IsPrivate   bool
}

type UpdateIn struct {
    Title        *string
    Description  *string
    Sidebar      *string
    IconURL      *string
    BannerURL    *string
    PrimaryColor *string
    IsNSFW       *bool
}
```

### 4.3 Threads API

```go
package threads

import "context"

type API interface {
    // Thread management
    Create(ctx context.Context, authorID string, in CreateIn) (*Thread, error)
    GetByID(ctx context.Context, id string, viewerID string) (*Thread, error)
    Update(ctx context.Context, id string, in UpdateIn) (*Thread, error)
    Delete(ctx context.Context, id string) error
    IncrementViews(ctx context.Context, id string) error

    // Listing
    List(ctx context.Context, opts ListOpts, viewerID string) ([]*Thread, error)
    ListByBoard(ctx context.Context, boardID string, opts ListOpts, viewerID string) ([]*Thread, error)
    ListByAuthor(ctx context.Context, authorID string, opts ListOpts, viewerID string) ([]*Thread, error)

    // Moderation
    Remove(ctx context.Context, id string, reason string) error
    Approve(ctx context.Context, id string) error
    Lock(ctx context.Context, id string) error
    Unlock(ctx context.Context, id string) error
    Pin(ctx context.Context, id string) error
    Unpin(ctx context.Context, id string) error
    SetNSFW(ctx context.Context, id string, nsfw bool) error
    SetSpoiler(ctx context.Context, id string, spoiler bool) error

    // Recalculation
    RecalculateScores(ctx context.Context) error
}

type CreateIn struct {
    BoardID   string
    Title     string
    Content   string
    URL       string
    Type      ThreadType
    Tags      []string
    IsNSFW    bool
    IsSpoiler bool
}

type UpdateIn struct {
    Content   *string
    IsNSFW    *bool
    IsSpoiler *bool
}

type ListOpts struct {
    Limit     int
    Cursor    string
    SortBy    SortBy
    TimeRange TimeRange
}
```

### 4.4 Comments API

```go
package comments

import "context"

type API interface {
    // Comment management
    Create(ctx context.Context, authorID string, in CreateIn) (*Comment, error)
    GetByID(ctx context.Context, id string, viewerID string) (*Comment, error)
    Update(ctx context.Context, id string, content string) (*Comment, error)
    Delete(ctx context.Context, id string) error

    // Listing
    ListByThread(ctx context.Context, threadID string, opts ListOpts, viewerID string) ([]*Comment, error)
    ListByParent(ctx context.Context, parentID string, opts ListOpts, viewerID string) ([]*Comment, error)
    ListByAuthor(ctx context.Context, authorID string, opts ListOpts, viewerID string) ([]*Comment, error)

    // Tree operations
    GetTree(ctx context.Context, threadID string, opts TreeOpts, viewerID string) ([]*Comment, error)
    GetSubtree(ctx context.Context, parentID string, depth int, viewerID string) ([]*Comment, error)

    // Moderation
    Remove(ctx context.Context, id string, reason string) error
    Approve(ctx context.Context, id string) error
}

type CreateIn struct {
    ThreadID string
    ParentID string // Empty for top-level
    Content  string
}

type ListOpts struct {
    Limit  int
    Cursor string
    SortBy CommentSort
}

type TreeOpts struct {
    Sort       CommentSort
    Limit      int
    MaxDepth   int
    CollapseAt int
}
```

### 4.5 Votes API

```go
package votes

import "context"

type API interface {
    // Voting
    Vote(ctx context.Context, accountID, targetType, targetID string, value int) error
    Unvote(ctx context.Context, accountID, targetType, targetID string) error
    GetVote(ctx context.Context, accountID, targetType, targetID string) (*Vote, error)

    // Batch operations
    GetVotes(ctx context.Context, accountID, targetType string, targetIDs []string) (map[string]int, error)

    // Stats
    GetVoteCounts(ctx context.Context, targetType, targetID string) (up, down int64, error)
}
```

### 4.6 Bookmarks API

```go
package bookmarks

import "context"

type API interface {
    Create(ctx context.Context, accountID, targetType, targetID string) error
    Delete(ctx context.Context, accountID, targetType, targetID string) error
    IsBookmarked(ctx context.Context, accountID, targetType, targetID string) (bool, error)
    List(ctx context.Context, accountID, targetType string, opts ListOpts) ([]Bookmark, error)

    // Batch check
    GetBookmarked(ctx context.Context, accountID, targetType string, targetIDs []string) (map[string]bool, error)
}
```

### 4.7 Search API

```go
package search

import "context"

type API interface {
    SearchThreads(ctx context.Context, query string, opts SearchOpts) ([]*Thread, error)
    SearchComments(ctx context.Context, query string, opts SearchOpts) ([]*Comment, error)
    SearchBoards(ctx context.Context, query string, limit int) ([]*Board, error)
    SearchUsers(ctx context.Context, query string, limit int) ([]*Account, error)

    // Suggestions
    Suggest(ctx context.Context, prefix string, limit int) ([]Suggestion, error)
}

type SearchOpts struct {
    Limit    int
    Offset   int
    BoardID  string   // Filter by board
    AuthorID string   // Filter by author
    SortBy   string   // relevance, new, top
    NSFW     bool     // Include NSFW
}

type Suggestion struct {
    Type  string // board, user, tag
    Value string
    Label string
}
```

### 4.8 Moderation API

```go
package moderation

import "context"

type API interface {
    // Actions
    RemoveThread(ctx context.Context, modID, threadID, reason string) error
    ApproveThread(ctx context.Context, modID, threadID string) error
    RemoveComment(ctx context.Context, modID, commentID, reason string) error
    ApproveComment(ctx context.Context, modID, commentID string) error

    // User moderation
    BanUser(ctx context.Context, modID, boardID, accountID string, opts BanOpts) error
    UnbanUser(ctx context.Context, modID, boardID, accountID string) error
    MuteUser(ctx context.Context, modID, boardID, accountID string, duration time.Duration) error
    WarnUser(ctx context.Context, modID, boardID, accountID, message string) error

    // Reports
    CreateReport(ctx context.Context, reporterID string, in ReportIn) (*Report, error)
    ListReports(ctx context.Context, boardID string, status ReportStatus, opts ListOpts) ([]*Report, error)
    ResolveReport(ctx context.Context, modID, reportID string, action string) error
    DismissReport(ctx context.Context, modID, reportID string) error

    // Mod log
    ListActions(ctx context.Context, boardID string, opts ListOpts) ([]*ModAction, error)

    // Bans
    ListBans(ctx context.Context, boardID string, opts ListOpts) ([]*Ban, error)
    IsBanned(ctx context.Context, boardID, accountID string) (*Ban, error)
}

type BanOpts struct {
    Reason    string
    Message   string
    Duration  time.Duration // 0 for permanent
}

type ReportIn struct {
    BoardID    string
    TargetType string
    TargetID   string
    Reason     string
    Details    string
}
```

## 5. Database Schema

```sql
-- schema.sql

-- Accounts
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR PRIMARY KEY,
    username VARCHAR UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    display_name VARCHAR,
    bio TEXT,
    avatar_url VARCHAR,
    banner_url VARCHAR,
    karma BIGINT DEFAULT 0,
    post_karma BIGINT DEFAULT 0,
    comment_karma BIGINT DEFAULT 0,
    is_admin BOOLEAN DEFAULT FALSE,
    is_suspended BOOLEAN DEFAULT FALSE,
    suspend_reason VARCHAR,
    suspend_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_username ON accounts(LOWER(username));
CREATE INDEX IF NOT EXISTS idx_accounts_karma ON accounts(karma DESC);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token VARCHAR UNIQUE NOT NULL,
    user_agent VARCHAR,
    ip VARCHAR,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_account ON sessions(account_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Boards
CREATE TABLE IF NOT EXISTS boards (
    id VARCHAR PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    title VARCHAR NOT NULL,
    description TEXT,
    sidebar TEXT,
    sidebar_html TEXT,
    icon_url VARCHAR,
    banner_url VARCHAR,
    primary_color VARCHAR,
    is_nsfw BOOLEAN DEFAULT FALSE,
    is_private BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    member_count BIGINT DEFAULT 0,
    thread_count BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR REFERENCES accounts(id),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_boards_name ON boards(LOWER(name));
CREATE INDEX IF NOT EXISTS idx_boards_members ON boards(member_count DESC);

-- Board members
CREATE TABLE IF NOT EXISTS board_members (
    board_id VARCHAR REFERENCES boards(id) ON DELETE CASCADE,
    account_id VARCHAR REFERENCES accounts(id) ON DELETE CASCADE,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (board_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_board_members_account ON board_members(account_id);

-- Board moderators
CREATE TABLE IF NOT EXISTS board_moderators (
    board_id VARCHAR REFERENCES boards(id) ON DELETE CASCADE,
    account_id VARCHAR REFERENCES accounts(id) ON DELETE CASCADE,
    permissions JSON,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    added_by VARCHAR REFERENCES accounts(id),
    PRIMARY KEY (board_id, account_id)
);

-- Threads
CREATE TABLE IF NOT EXISTS threads (
    id VARCHAR PRIMARY KEY,
    board_id VARCHAR NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    author_id VARCHAR NOT NULL REFERENCES accounts(id),
    title VARCHAR NOT NULL,
    content TEXT,
    content_html TEXT,
    url VARCHAR,
    domain VARCHAR,
    thumbnail_url VARCHAR,
    type VARCHAR DEFAULT 'text',
    score BIGINT DEFAULT 0,
    upvote_count BIGINT DEFAULT 0,
    downvote_count BIGINT DEFAULT 0,
    comment_count BIGINT DEFAULT 0,
    view_count BIGINT DEFAULT 0,
    hot_score DOUBLE DEFAULT 0,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_locked BOOLEAN DEFAULT FALSE,
    is_removed BOOLEAN DEFAULT FALSE,
    is_nsfw BOOLEAN DEFAULT FALSE,
    is_spoiler BOOLEAN DEFAULT FALSE,
    is_oc BOOLEAN DEFAULT FALSE,
    remove_reason VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_threads_board ON threads(board_id, hot_score DESC);
CREATE INDEX IF NOT EXISTS idx_threads_author ON threads(author_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_hot ON threads(hot_score DESC);
CREATE INDEX IF NOT EXISTS idx_threads_new ON threads(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_score ON threads(score DESC);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR PRIMARY KEY,
    thread_id VARCHAR NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    parent_id VARCHAR REFERENCES comments(id),
    author_id VARCHAR NOT NULL REFERENCES accounts(id),
    content TEXT NOT NULL,
    content_html TEXT,
    score BIGINT DEFAULT 0,
    upvote_count BIGINT DEFAULT 0,
    downvote_count BIGINT DEFAULT 0,
    depth INT DEFAULT 0,
    path VARCHAR NOT NULL,
    child_count BIGINT DEFAULT 0,
    is_removed BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    remove_reason VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_thread ON comments(thread_id, path);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);
CREATE INDEX IF NOT EXISTS idx_comments_author ON comments(author_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comments_path ON comments(path);

-- Votes
CREATE TABLE IF NOT EXISTS votes (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    value INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (account_id, target_type, target_id)
);

CREATE INDEX IF NOT EXISTS idx_votes_target ON votes(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_votes_account ON votes(account_id);

-- Bookmarks
CREATE TABLE IF NOT EXISTS bookmarks (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (account_id, target_type, target_id)
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_account ON bookmarks(account_id, created_at DESC);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    type VARCHAR NOT NULL,
    actor_id VARCHAR REFERENCES accounts(id),
    board_id VARCHAR REFERENCES boards(id),
    thread_id VARCHAR REFERENCES threads(id),
    comment_id VARCHAR REFERENCES comments(id),
    message TEXT,
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_account ON notifications(account_id, read, created_at DESC);

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    id VARCHAR PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    label VARCHAR NOT NULL,
    color VARCHAR,
    use_count BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS thread_tags (
    thread_id VARCHAR REFERENCES threads(id) ON DELETE CASCADE,
    tag_id VARCHAR REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (thread_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_thread_tags_tag ON thread_tags(tag_id);

-- Mod actions
CREATE TABLE IF NOT EXISTS mod_actions (
    id VARCHAR PRIMARY KEY,
    board_id VARCHAR NOT NULL REFERENCES boards(id),
    moderator_id VARCHAR NOT NULL REFERENCES accounts(id),
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    action VARCHAR NOT NULL,
    reason TEXT,
    details JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mod_actions_board ON mod_actions(board_id, created_at DESC);

-- Bans
CREATE TABLE IF NOT EXISTS bans (
    id VARCHAR PRIMARY KEY,
    board_id VARCHAR REFERENCES boards(id),
    account_id VARCHAR NOT NULL REFERENCES accounts(id),
    reason TEXT,
    message TEXT,
    mod_id VARCHAR REFERENCES accounts(id),
    is_permanent BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bans_board_account ON bans(board_id, account_id);
CREATE INDEX IF NOT EXISTS idx_bans_account ON bans(account_id);

-- Reports
CREATE TABLE IF NOT EXISTS reports (
    id VARCHAR PRIMARY KEY,
    reporter_id VARCHAR NOT NULL REFERENCES accounts(id),
    board_id VARCHAR REFERENCES boards(id),
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    reason VARCHAR NOT NULL,
    details TEXT,
    status VARCHAR DEFAULT 'pending',
    resolved_by VARCHAR REFERENCES accounts(id),
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reports_board_status ON reports(board_id, status, created_at DESC);

-- Full-text search
CREATE INDEX IF NOT EXISTS idx_threads_fts ON threads USING gin(to_tsvector('english', title || ' ' || COALESCE(content, '')));
CREATE INDEX IF NOT EXISTS idx_comments_fts ON comments USING gin(to_tsvector('english', content));
```

## 6. HTTP Handlers

### 6.1 Route Definitions

```go
// routes.go

func (s *Server) setupRoutes() {
    r := s.app.Router

    // Static files
    r.Get("/static/{path...}", s.staticHandler())

    // API routes
    api := r.Group("/api")

    // Auth
    api.Post("/auth/register", s.auth.Register)
    api.Post("/auth/login", s.auth.Login)
    api.Post("/auth/logout", s.authRequired(s.auth.Logout))
    api.Get("/auth/me", s.authRequired(s.auth.Me))

    // Boards
    api.Get("/boards", s.board.List)
    api.Post("/boards", s.authRequired(s.board.Create))
    api.Get("/boards/{name}", s.board.Get)
    api.Put("/boards/{name}", s.authRequired(s.modRequired(s.board.Update)))
    api.Delete("/boards/{name}", s.authRequired(s.adminRequired(s.board.Delete)))
    api.Post("/boards/{name}/join", s.authRequired(s.board.Join))
    api.Delete("/boards/{name}/join", s.authRequired(s.board.Leave))
    api.Get("/boards/{name}/moderators", s.board.ListModerators)

    // Threads
    api.Get("/threads", s.thread.List)
    api.Get("/threads/{id}", s.thread.Get)
    api.Put("/threads/{id}", s.authRequired(s.thread.Update))
    api.Delete("/threads/{id}", s.authRequired(s.thread.Delete))
    api.Post("/boards/{name}/threads", s.authRequired(s.thread.Create))

    // Voting
    api.Post("/threads/{id}/vote", s.authRequired(s.vote.VoteThread))
    api.Delete("/threads/{id}/vote", s.authRequired(s.vote.UnvoteThread))
    api.Post("/comments/{id}/vote", s.authRequired(s.vote.VoteComment))
    api.Delete("/comments/{id}/vote", s.authRequired(s.vote.UnvoteComment))

    // Bookmarks
    api.Post("/threads/{id}/bookmark", s.authRequired(s.bookmark.SaveThread))
    api.Delete("/threads/{id}/bookmark", s.authRequired(s.bookmark.UnsaveThread))
    api.Post("/comments/{id}/bookmark", s.authRequired(s.bookmark.SaveComment))
    api.Delete("/comments/{id}/bookmark", s.authRequired(s.bookmark.UnsaveComment))

    // Comments
    api.Get("/threads/{id}/comments", s.comment.List)
    api.Post("/threads/{id}/comments", s.authRequired(s.comment.Create))
    api.Get("/comments/{id}", s.comment.Get)
    api.Put("/comments/{id}", s.authRequired(s.comment.Update))
    api.Delete("/comments/{id}", s.authRequired(s.comment.Delete))

    // Users
    api.Get("/users/{username}", s.user.Get)
    api.Get("/users/{username}/threads", s.user.ListThreads)
    api.Get("/users/{username}/comments", s.user.ListComments)
    api.Post("/users/{username}/follow", s.authRequired(s.user.Follow))
    api.Delete("/users/{username}/follow", s.authRequired(s.user.Unfollow))

    // Search
    api.Get("/search", s.search.Search)
    api.Get("/search/suggest", s.search.Suggest)

    // Notifications
    api.Get("/notifications", s.authRequired(s.notification.List))
    api.Post("/notifications/read", s.authRequired(s.notification.MarkRead))
    api.Post("/notifications/read-all", s.authRequired(s.notification.MarkAllRead))

    // Moderation
    mod := api.Group("/mod")
    mod.Post("/threads/{id}/remove", s.authRequired(s.modRequired(s.mod.RemoveThread)))
    mod.Post("/threads/{id}/approve", s.authRequired(s.modRequired(s.mod.ApproveThread)))
    mod.Post("/threads/{id}/lock", s.authRequired(s.modRequired(s.mod.LockThread)))
    mod.Post("/threads/{id}/pin", s.authRequired(s.modRequired(s.mod.PinThread)))
    mod.Post("/comments/{id}/remove", s.authRequired(s.modRequired(s.mod.RemoveComment)))
    mod.Get("/boards/{name}/queue", s.authRequired(s.modRequired(s.mod.Queue)))
    mod.Get("/boards/{name}/log", s.authRequired(s.modRequired(s.mod.Log)))
    mod.Post("/boards/{name}/ban", s.authRequired(s.modRequired(s.mod.BanUser)))
    mod.Delete("/boards/{name}/ban/{username}", s.authRequired(s.modRequired(s.mod.UnbanUser)))

    // HTML pages
    r.Get("/", s.page.Home)
    r.Get("/all", s.page.All)
    r.Get("/b/{name}", s.page.Board)
    r.Get("/b/{name}/submit", s.authRequired(s.page.Submit))
    r.Get("/b/{name}/{id}/{slug}", s.page.Thread)
    r.Get("/u/{username}", s.page.User)
    r.Get("/search", s.page.Search)
    r.Get("/login", s.page.Login)
    r.Get("/register", s.page.Register)
    r.Get("/settings", s.authRequired(s.page.Settings))
    r.Get("/bookmarks", s.authRequired(s.page.Bookmarks))
    r.Get("/notifications", s.authRequired(s.page.Notifications))

    // Moderation pages
    r.Get("/b/{name}/mod", s.authRequired(s.modRequired(s.page.ModDashboard)))
    r.Get("/b/{name}/mod/queue", s.authRequired(s.modRequired(s.page.ModQueue)))
    r.Get("/b/{name}/mod/log", s.authRequired(s.modRequired(s.page.ModLog)))
}
```

### 6.2 Response Format

```go
// response.go

type Response struct {
    Data  any    `json:"data,omitempty"`
    Error *Error `json:"error,omitempty"`
}

type Error struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ListResponse struct {
    Data       any    `json:"data"`
    NextCursor string `json:"next_cursor,omitempty"`
    HasMore    bool   `json:"has_more"`
}

func Success(c *mizu.Ctx, data any) error {
    return c.JSON(200, Response{Data: data})
}

func Created(c *mizu.Ctx, data any) error {
    return c.JSON(201, Response{Data: data})
}

func ErrorResponse(c *mizu.Ctx, status int, code, message string) error {
    return c.JSON(status, Response{Error: &Error{Code: code, Message: message}})
}

func BadRequest(c *mizu.Ctx, message string) error {
    return ErrorResponse(c, 400, "BAD_REQUEST", message)
}

func Unauthorized(c *mizu.Ctx) error {
    return ErrorResponse(c, 401, "UNAUTHORIZED", "Authentication required")
}

func Forbidden(c *mizu.Ctx) error {
    return ErrorResponse(c, 403, "FORBIDDEN", "You don't have permission")
}

func NotFound(c *mizu.Ctx, what string) error {
    return ErrorResponse(c, 404, "NOT_FOUND", what+" not found")
}
```

## 7. UI Components

### 7.1 Template Structure

```
views/
├── layouts/
│   └── default.html      # Base layout with nav, footer
├── pages/
│   ├── home.html         # Home feed
│   ├── all.html          # All posts
│   ├── board.html        # Board page
│   ├── thread.html       # Thread with comments
│   ├── submit.html       # Create thread form
│   ├── user.html         # User profile
│   ├── search.html       # Search results
│   ├── login.html        # Login form
│   ├── register.html     # Registration form
│   ├── settings.html     # User settings
│   ├── bookmarks.html    # Saved content
│   ├── notifications.html # Notifications
│   └── mod/
│       ├── dashboard.html
│       ├── queue.html
│       └── log.html
└── components/
    ├── nav.html          # Navigation bar
    ├── sidebar.html      # Board sidebar
    ├── thread_card.html  # Thread preview card
    ├── thread_full.html  # Full thread display
    ├── comment.html      # Comment (recursive)
    ├── vote_buttons.html # Upvote/downvote
    ├── user_card.html    # User info card
    ├── board_card.html   # Board info card
    ├── pagination.html   # Pagination controls
    ├── sort_tabs.html    # Sort options
    ├── flair.html        # Post flair/tags
    └── compose.html      # Comment composer
```

### 7.2 CSS Architecture

```css
/* app.css - CSS Custom Properties */

:root {
    /* Colors */
    --color-primary: #ff4500;      /* Reddit orange */
    --color-primary-hover: #ff5722;
    --color-bg: #ffffff;
    --color-bg-secondary: #f6f7f8;
    --color-bg-tertiary: #edeff1;
    --color-text: #1c1c1c;
    --color-text-secondary: #576f76;
    --color-text-muted: #878a8c;
    --color-border: #ccc;
    --color-upvote: #ff4500;
    --color-downvote: #7193ff;
    --color-link: #0079d3;

    /* Dark mode */
    --color-bg-dark: #1a1a1b;
    --color-bg-secondary-dark: #272729;
    --color-text-dark: #d7dadc;

    /* Spacing */
    --space-xs: 4px;
    --space-sm: 8px;
    --space-md: 16px;
    --space-lg: 24px;
    --space-xl: 32px;

    /* Typography */
    --font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    --font-size-xs: 12px;
    --font-size-sm: 14px;
    --font-size-md: 16px;
    --font-size-lg: 18px;
    --font-size-xl: 24px;

    /* Border radius */
    --radius-sm: 4px;
    --radius-md: 8px;
    --radius-lg: 16px;
    --radius-full: 9999px;

    /* Shadows */
    --shadow-sm: 0 1px 2px rgba(0,0,0,0.1);
    --shadow-md: 0 2px 4px rgba(0,0,0,0.1);
    --shadow-lg: 0 4px 8px rgba(0,0,0,0.15);

    /* Layout */
    --content-width: 640px;
    --sidebar-width: 312px;
    --nav-height: 48px;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
    :root {
        --color-bg: var(--color-bg-dark);
        --color-bg-secondary: var(--color-bg-secondary-dark);
        --color-text: var(--color-text-dark);
    }
}
```

## 8. Algorithms

### 8.1 Hot Score

```go
// Reddit-style hot ranking algorithm

func HotScore(ups, downs int64, created time.Time) float64 {
    score := float64(ups - downs)
    order := math.Log10(math.Max(math.Abs(score), 1))

    sign := 0.0
    if score > 0 {
        sign = 1
    } else if score < 0 {
        sign = -1
    }

    // Reddit epoch: Dec 8, 2005
    seconds := created.Unix() - 1134028003

    return sign*order + float64(seconds)/45000
}
```

### 8.2 Wilson Score (Best Comments)

```go
// Lower bound of Wilson score confidence interval

func WilsonScore(ups, downs int64) float64 {
    n := float64(ups + downs)
    if n == 0 {
        return 0
    }

    z := 1.96 // 95% confidence
    phat := float64(ups) / n

    return (phat + z*z/(2*n) - z*math.Sqrt((phat*(1-phat)+z*z/(4*n))/n)) / (1 + z*z/n)
}
```

### 8.3 Controversial Score

```go
// Higher when votes are balanced and numerous

func ControversialScore(ups, downs int64) float64 {
    if ups <= 0 || downs <= 0 {
        return 0
    }

    magnitude := float64(ups + downs)
    balance := float64(min(ups, downs)) / float64(max(ups, downs))

    return magnitude * balance
}
```

### 8.4 Comment Tree Building

```go
// Efficient tree building using materialized paths

func BuildCommentTree(comments []*Comment) []*Comment {
    byID := make(map[string]*Comment, len(comments))
    roots := make([]*Comment, 0)

    for _, c := range comments {
        byID[c.ID] = c
        c.Children = make([]*Comment, 0)
    }

    for _, c := range comments {
        if c.ParentID == "" {
            roots = append(roots, c)
        } else if parent, ok := byID[c.ParentID]; ok {
            parent.Children = append(parent.Children, c)
        }
    }

    return roots
}
```

## 9. Security Considerations

### 9.1 Authentication

- Passwords hashed with Argon2id
- Sessions use cryptographically random tokens
- HttpOnly cookies for web sessions
- CSRF tokens for form submissions
- Rate limiting on auth endpoints

### 9.2 Authorization

- Board-level permissions (member, moderator, admin)
- Resource ownership checks
- Mod actions require explicit permissions
- Admin bypass for site-wide operations

### 9.3 Input Validation

- Username: 3-20 chars, alphanumeric + underscore
- Email: RFC 5322 format
- Password: minimum 8 chars
- Content: Markdown sanitized, HTML stripped
- URLs: Protocol whitelist (http, https)

### 9.4 Rate Limiting

```go
// Rate limits per action
const (
    AuthLoginLimit    = "5/minute"
    AuthRegisterLimit = "3/hour"
    PostThreadLimit   = "10/hour"
    PostCommentLimit  = "60/hour"
    VoteLimit         = "100/minute"
    SearchLimit       = "30/minute"
)
```

## 10. Performance Optimization

### 10.1 Database

- Materialized paths for comment trees (O(1) subtree queries)
- Pre-computed hot scores (updated periodically)
- Denormalized counts (member_count, comment_count)
- Efficient pagination with cursors

### 10.2 Caching Strategy

- Board metadata: 5 minute cache
- Thread lists: 1 minute cache
- User sessions: In-memory with DB backup
- Static assets: 1 year cache with versioning

### 10.3 Query Optimization

```sql
-- Efficient thread listing with viewer state
SELECT
    t.*,
    a.username, a.display_name, a.avatar_url,
    b.name as board_name, b.title as board_title,
    COALESCE(v.value, 0) as vote,
    bm.id IS NOT NULL as is_bookmarked
FROM threads t
JOIN accounts a ON t.author_id = a.id
JOIN boards b ON t.board_id = b.id
LEFT JOIN votes v ON v.target_type = 'thread'
    AND v.target_id = t.id
    AND v.account_id = $1
LEFT JOIN bookmarks bm ON bm.target_type = 'thread'
    AND bm.target_id = t.id
    AND bm.account_id = $1
WHERE t.board_id = $2
    AND NOT t.is_removed
ORDER BY t.hot_score DESC
LIMIT 25;
```

## 11. Testing Strategy

### 11.1 Unit Tests

- Service layer logic
- Algorithm implementations
- Input validation
- Database operations

### 11.2 Integration Tests

- API endpoint behavior
- Authentication flows
- Permission checks
- Database transactions

### 11.3 E2E Tests

- User registration and login
- Create and view threads
- Comment threads
- Voting interactions
- Moderation actions

## 12. Deployment

### 12.1 Single Binary

```bash
# Build production binary
go build -ldflags="-s -w" -o forum ./cmd/forum

# Run with configuration
./forum serve --addr :8080 --data ~/.forum
```

### 12.2 Docker

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o forum ./cmd/forum

FROM alpine:latest
COPY --from=builder /app/forum /usr/local/bin/
EXPOSE 8080
CMD ["forum", "serve"]
```

### 12.3 Configuration

```yaml
# config.yaml
server:
  addr: ":8080"
  read_timeout: 30s
  write_timeout: 30s

database:
  path: "/data/forum.db"

security:
  session_duration: 720h  # 30 days
  bcrypt_cost: 12

limits:
  max_title_length: 300
  max_content_length: 40000
  max_comment_depth: 10
```

## 13. Future Considerations

### 13.1 Phase 2

- Private messaging
- User flairs
- Board flairs
- Post awards
- Wiki pages per board

### 13.2 Phase 3

- Federation (ActivityPub)
- Real-time notifications (WebSocket)
- Mobile app API
- Analytics dashboard
- A/B testing framework

## 14. Appendix

### 14.1 Error Codes

| Code | HTTP | Description |
|------|------|-------------|
| `BAD_REQUEST` | 400 | Invalid input |
| `UNAUTHORIZED` | 401 | Not authenticated |
| `FORBIDDEN` | 403 | Not authorized |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource already exists |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL` | 500 | Server error |

### 14.2 Notification Types

| Type | Description |
|------|-------------|
| `reply` | Someone replied to your comment |
| `mention` | Someone mentioned you |
| `thread_vote` | Milestone votes on your thread |
| `comment_vote` | Milestone votes on your comment |
| `follow` | Someone followed you |
| `mod` | Moderation action on your content |

### 14.3 Report Reasons

- Spam
- Harassment
- Hate speech
- Misinformation
- NSFW (unmarked)
- Self-harm
- Illegal content
- Other

---

*End of Specification*
