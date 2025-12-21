# 0121 Microblog Architecture Refactoring

## Overview

This spec defines the architecture refactoring for the microblog blueprint to achieve:
1. Clean separation between feature APIs and storage implementations
2. Dependency injection via interfaces instead of concrete types
3. Embedded assets for static files and templates
4. Testable handlers with proper interface boundaries

## Current State

```
blueprints/microblog/
├── app/web/           # HTTP handlers (depend on concrete *Service)
├── assets/            # Static files + views (not embedded)
├── cli/               # CLI commands
├── cmd/microblog/     # Entry point
├── feature/           # 8 features with services depending on *duckdb.Store
│   ├── accounts/      # types.go + service.go
│   ├── posts/         # types.go + service.go
│   ├── interactions/  # service.go only
│   ├── relationships/ # service.go only
│   ├── timelines/     # service.go only
│   ├── notifications/ # service.go only
│   ├── search/        # service.go only
│   └── trending/      # service.go only
├── pkg/               # Utilities (ulid, text, password)
└── store/duckdb/      # Single store.go with raw SQL helpers
```

### Current Problems

1. **Tight coupling**: All services import `*duckdb.Store` directly
2. **No API contracts**: Services expose implementation, not interfaces
3. **Untestable**: Cannot mock store for unit testing
4. **No embedded assets**: Templates not bundled in binary

## Target Architecture

```
blueprints/microblog/
├── app/web/
│   ├── server.go        # Server setup with dependency injection
│   ├── handlers.go      # API handlers (JSON)
│   ├── pages.go         # Web page handlers (HTML)
│   ├── middleware.go    # Auth middleware
│   └── handlers_test.go # Handler tests with mocked services
├── assets/
│   ├── assets.go        # //go:embed for static + views
│   ├── static/css/
│   ├── static/js/
│   └── views/
├── feature/
│   ├── accounts/
│   │   ├── api.go       # Types + API interface + Store interface
│   │   └── service.go   # Service implements API, takes Store
│   ├── posts/
│   │   ├── api.go
│   │   └── service.go
│   └── ... (all 8 features)
└── store/duckdb/
    ├── store.go           # Core DB connection + schema init
    ├── accounts_store.go  # Implements accounts.Store
    ├── posts_store.go     # Implements posts.Store
    └── ... (all feature stores)
```

## Implementation Details

### 1. Feature API Files (api.go)

Each feature gets an `api.go` file defining:

```go
// feature/accounts/api.go
package accounts

import (
    "context"
    "database/sql"
    "time"
)

// === Types ===

type Account struct { ... }     // Domain types (unchanged)
type CreateIn struct { ... }
type Session struct { ... }

// === API Interface ===

// API defines the accounts service contract.
// Service must implement this interface.
type API interface {
    Create(ctx context.Context, in *CreateIn) (*Account, error)
    GetByID(ctx context.Context, id string) (*Account, error)
    GetByUsername(ctx context.Context, username string) (*Account, error)
    Update(ctx context.Context, id string, in *UpdateIn) (*Account, error)
    Login(ctx context.Context, in *LoginIn) (*Session, error)
    GetSession(ctx context.Context, token string) (*Session, error)
    DeleteSession(ctx context.Context, token string) error
    List(ctx context.Context, limit, offset int) (*AccountList, error)
    Search(ctx context.Context, query string, limit int) ([]*Account, error)
}

// === Store Interface ===

// Store defines the data access contract for accounts.
// Implemented by store/duckdb/accounts_store.go
type Store interface {
    // Account operations
    Insert(ctx context.Context, a *Account, passwordHash string) error
    GetByID(ctx context.Context, id string) (*Account, error)
    GetByUsername(ctx context.Context, username string) (*Account, error)
    GetByEmail(ctx context.Context, email string) (*Account, error)
    Update(ctx context.Context, id string, in *UpdateIn) error
    ExistsUsername(ctx context.Context, username string) (bool, error)
    ExistsEmail(ctx context.Context, email string) (bool, error)
    GetPasswordHash(ctx context.Context, usernameOrEmail string) (id, hash string, suspended bool, err error)
    List(ctx context.Context, limit, offset int) ([]*Account, int, error)
    Search(ctx context.Context, query string, limit int) ([]*Account, error)

    // Session operations
    CreateSession(ctx context.Context, s *Session) error
    GetSession(ctx context.Context, token string) (*Session, error)
    DeleteSession(ctx context.Context, token string) error
}
```

### 2. Service Refactoring

Services change from depending on `*duckdb.Store` to their feature's `Store` interface:

```go
// feature/accounts/service.go
package accounts

import (
    "context"
    "github.com/go-mizu/blueprints/microblog/pkg/password"
    "github.com/go-mizu/blueprints/microblog/pkg/ulid"
)

// Service handles account operations.
// Implements API interface.
type Service struct {
    store Store  // Interface, not *duckdb.Store
}

// NewService creates a new accounts service.
func NewService(store Store) *Service {
    return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in *CreateIn) (*Account, error) {
    // Validate
    if !usernameRegex.MatchString(in.Username) {
        return nil, ErrInvalidUsername
    }

    exists, err := s.store.ExistsUsername(ctx, in.Username)
    if err != nil { return nil, err }
    if exists { return nil, ErrUsernameTaken }

    hash, err := password.Hash(in.Password)
    if err != nil { return nil, err }

    account := &Account{
        ID:          ulid.New(),
        Username:    strings.ToLower(in.Username),
        DisplayName: in.DisplayName,
        // ...
    }

    if err := s.store.Insert(ctx, account, hash); err != nil {
        return nil, err
    }

    return account, nil
}
```

### 3. DuckDB Store Implementation

Each feature gets a dedicated store file:

```go
// store/duckdb/accounts_store.go
package duckdb

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"

    "github.com/go-mizu/blueprints/microblog/feature/accounts"
)

// AccountsStore implements accounts.Store using DuckDB.
type AccountsStore struct {
    db *sql.DB
}

// NewAccountsStore creates a new accounts store.
func NewAccountsStore(db *sql.DB) *AccountsStore {
    return &AccountsStore{db: db}
}

func (s *AccountsStore) Insert(ctx context.Context, a *accounts.Account, passwordHash string) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO accounts (id, username, display_name, email, password_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, a.ID, a.Username, a.DisplayName, a.Email, passwordHash, a.CreatedAt, a.UpdatedAt)
    return err
}

func (s *AccountsStore) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
    row := s.db.QueryRowContext(ctx, `
        SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
               verified, admin, suspended, created_at, updated_at
        FROM accounts WHERE id = $1
    `, id)
    return s.scanAccount(row)
}

// ... implement all Store interface methods
```

### 4. Assets Embedding

```go
// assets/assets.go
package assets

import (
    "embed"
    "html/template"
    "io/fs"
)

//go:embed static views
var FS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
    sub, _ := fs.Sub(FS, "static")
    return sub
}

// Views returns the views filesystem.
func Views() fs.FS {
    sub, _ := fs.Sub(FS, "views")
    return sub
}

// Templates parses all view templates.
func Templates() (*template.Template, error) {
    return template.ParseFS(Views(), "layouts/*.html", "pages/*.html", "components/*.html")
}
```

### 5. Web Server Updates

```go
// app/web/server.go
package web

import (
    "github.com/go-mizu/mizu"
    "github.com/go-mizu/blueprints/microblog/assets"
    "github.com/go-mizu/blueprints/microblog/feature/accounts"
    // ...
)

type Server struct {
    app       *mizu.App
    cfg       Config
    templates *template.Template

    // Services as interfaces
    accounts      accounts.API
    posts         posts.API
    timelines     timelines.API
    interactions  interactions.API
    relationships relationships.API
    notifications notifications.API
    search        search.API
    trending      trending.API
}

func New(cfg Config) (*Server, error) {
    // ... database setup ...

    // Create stores
    accountsStore := duckdb.NewAccountsStore(db)
    postsStore := duckdb.NewPostsStore(db)
    // ...

    // Create services with stores
    accountsSvc := accounts.NewService(accountsStore)
    postsSvc := posts.NewService(postsStore, accountsSvc)
    // ...

    // Parse templates
    tmpl, err := assets.Templates()
    if err != nil {
        return nil, err
    }

    s := &Server{
        app:       mizu.New(),
        templates: tmpl,
        accounts:  accountsSvc,
        posts:     postsSvc,
        // ...
    }

    // Serve static files
    s.app.StaticFS("/static", assets.Static())

    s.setupRoutes()
    return s, nil
}
```

### 6. Handler Tests

```go
// app/web/handlers_test.go
package web

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-mizu/blueprints/microblog/feature/accounts"
)

// Mock accounts service
type mockAccountsAPI struct {
    accounts map[string]*accounts.Account
}

func (m *mockAccountsAPI) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
    if a, ok := m.accounts[id]; ok {
        return a, nil
    }
    return nil, accounts.ErrNotFound
}
// ... implement other API methods

func TestGetAccount(t *testing.T) {
    mock := &mockAccountsAPI{
        accounts: map[string]*accounts.Account{
            "123": {ID: "123", Username: "testuser"},
        },
    }

    s := &Server{accounts: mock}

    req := httptest.NewRequest("GET", "/api/v1/accounts/123", nil)
    rec := httptest.NewRecorder()

    // ... execute handler and assert response
}
```

## Feature-by-Feature Interface Definitions

### accounts.Store
```go
Insert(ctx, *Account, passwordHash string) error
GetByID(ctx, id string) (*Account, error)
GetByUsername(ctx, username string) (*Account, error)
GetByEmail(ctx, email string) (*Account, error)
Update(ctx, id string, in *UpdateIn) error
ExistsUsername(ctx, username string) (bool, error)
ExistsEmail(ctx, email string) (bool, error)
GetPasswordHash(ctx, usernameOrEmail string) (id, hash string, suspended bool, err error)
List(ctx, limit, offset int) ([]*Account, total int, error)
Search(ctx, query string, limit int) ([]*Account, error)
CreateSession(ctx, *Session) error
GetSession(ctx, token string) (*Session, error)
DeleteSession(ctx, token string) error
SetVerified(ctx, id string, verified bool) error
SetSuspended(ctx, id string, suspended bool) error
SetAdmin(ctx, id string, admin bool) error
```

### posts.Store
```go
Insert(ctx, *Post) error
GetByID(ctx, id string) (*Post, error)
Update(ctx, id string, in *UpdateIn) error
Delete(ctx, id string) error
GetThreadID(ctx, replyToID string) (string, error)
IncrementReplies(ctx, postID string) error
DecrementReplies(ctx, postID string) error
IncrementReposts(ctx, postID string) error
DecrementReposts(ctx, postID string) error
GetMedia(ctx, postID string) ([]*Media, error)
GetPoll(ctx, postID string) (*Poll, error)
GetVoterChoices(ctx, pollID, accountID string) ([]int, error)
SaveEditHistory(ctx, postID, content, cw string, sensitive bool) error
GetDescendants(ctx, postID string, limit int) ([]*Post, error)
CheckLiked(ctx, accountID, postID string) (bool, error)
CheckReposted(ctx, accountID, postID string) (bool, error)
CheckBookmarked(ctx, accountID, postID string) (bool, error)
```

### interactions.Store
```go
Like(ctx, accountID, postID string) (created bool, err error)
Unlike(ctx, accountID, postID string) (deleted bool, err error)
Repost(ctx, accountID, postID string) (created bool, err error)
Unrepost(ctx, accountID, postID string) (deleted bool, err error)
Bookmark(ctx, accountID, postID string) error
Unbookmark(ctx, accountID, postID string) error
GetLikedBy(ctx, postID string, limit, offset int) ([]string, error)
GetRepostedBy(ctx, postID string, limit, offset int) ([]string, error)
VotePoll(ctx, accountID, pollID string, choices []int) error
GetPostOwner(ctx, postID string) (string, error)
CreateNotification(ctx, accountID, actorID, notifType, postID string) error
```

### relationships.Store
```go
Follow(ctx, followerID, followingID string) error
Unfollow(ctx, followerID, followingID string) error
Block(ctx, accountID, targetID string) error
Unblock(ctx, accountID, targetID string) error
Mute(ctx, accountID, targetID string, hideNotifs bool, expiresAt *time.Time) error
Unmute(ctx, accountID, targetID string) error
IsFollowing(ctx, followerID, followingID string) (bool, error)
IsBlocking(ctx, accountID, targetID string) (bool, error)
IsMuting(ctx, accountID, targetID string) (bool, hideNotifs bool, err error)
GetFollowers(ctx, targetID string, limit, offset int) ([]string, error)
GetFollowing(ctx, targetID string, limit, offset int) ([]string, error)
GetBlocked(ctx, accountID string, limit, offset int) ([]string, error)
GetMuted(ctx, accountID string, limit, offset int) ([]string, error)
CountFollowers(ctx, accountID string) (int, error)
CountFollowing(ctx, accountID string) (int, error)
```

### timelines.Store
```go
Home(ctx, accountID string, limit int, maxID, sinceID string) ([]*posts.Post, error)
Local(ctx, limit int, maxID, sinceID string) ([]*posts.Post, error)
Hashtag(ctx, tag string, limit int, maxID, sinceID string) ([]*posts.Post, error)
Account(ctx, accountID string, limit int, maxID string, onlyMedia, excludeReplies bool) ([]*posts.Post, error)
Bookmarks(ctx, accountID string, limit int, maxID string) ([]*posts.Post, error)
List(ctx, listID string, limit int, maxID string) ([]*posts.Post, error)
```

### notifications.Store
```go
List(ctx, accountID string, types, excludeTypes []NotificationType, limit int, maxID, sinceID string) ([]*Notification, error)
Get(ctx, id, accountID string) (*Notification, error)
MarkAsRead(ctx, id, accountID string) error
MarkAllAsRead(ctx, accountID string) error
Dismiss(ctx, id, accountID string) error
DismissAll(ctx, accountID string) error
CountUnread(ctx, accountID string) (int, error)
CleanOld(ctx, olderThan time.Duration) (int64, error)
```

### search.Store
```go
SearchAccounts(ctx, query string, limit int) ([]*search.Result, error)
SearchHashtags(ctx, query string, limit int) ([]*search.Result, error)
SearchPosts(ctx, query string, limit int, viewerID string) ([]*search.Result, error)
SearchPostIDs(ctx, query string, limit int, maxID, sinceID string) ([]string, error)
SearchAccountIDs(ctx, query string, limit int) ([]string, error)
```

### trending.Store
```go
Tags(ctx, limit int) ([]*TrendingTag, error)
Posts(ctx, limit int) ([]string, error)
SuggestedAccounts(ctx, accountID string, limit int) ([]string, error)
```

## Migration Steps

1. Create `api.go` files for each feature (move types + add interfaces)
2. Update services to use Store interface
3. Create `store/duckdb/*_store.go` files implementing interfaces
4. Add `assets/assets.go` with embedding
5. Update `app/web/server.go` with new architecture
6. Split handlers into `handlers.go` (API) and `pages.go` (web)
7. Add `handlers_test.go` with mocks

## Verification

- All tests pass: `go test ./blueprints/microblog/...`
- Binary size remains reasonable with embedded assets
- No import cycles between packages
- Services only depend on interfaces, not concrete types
