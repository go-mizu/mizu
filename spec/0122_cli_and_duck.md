# Spec 0122: CLI Modernization & DuckDB Store Tests

## Overview

This spec defines the implementation plan for:
1. Modernizing the microblog CLI with Fang + Lipgloss for best-in-class DX
2. Implementing comprehensive DuckDB store tests with real database operations

## CLI Modernization

### Current State

The microblog CLI uses plain Cobra without:
- Fang for enhanced execution (styled help, auto version, completions)
- Lipgloss for styled terminal output
- Bubbletea for interactive elements

### Target State

```
microblog CLI
├── Fang wrapper for enhanced DX
│   ├── Styled help output
│   ├── Auto version/commit display
│   ├── Shell completion generation
│   └── Man page generation
├── Lipgloss styling
│   ├── Color palette matching mizu theme
│   ├── Success/error/warning/info styles
│   ├── Spinner animations
│   ├── Progress indicators
│   └── Table formatting
└── Commands with rich output
    ├── init - Database initialization with spinner
    ├── serve - Server startup with status info
    └── user - User management with styled output
```

### Implementation

#### 1. Add Dependencies

```bash
# In blueprints/microblog/go.mod
github.com/charmbracelet/fang
github.com/charmbracelet/lipgloss
```

#### 2. Create UI Package (cli/ui.go)

```go
package cli

import (
    "github.com/charmbracelet/lipgloss"
)

// Color palette (matching mizu CLI theme)
var (
    primaryColor   = lipgloss.Color("#10B981") // Emerald green
    secondaryColor = lipgloss.Color("#6B7280") // Gray
    accentColor    = lipgloss.Color("#3B82F6") // Blue
    successColor   = lipgloss.Color("#10B981") // Green
    errorColor     = lipgloss.Color("#EF4444") // Red
    warnColor      = lipgloss.Color("#F59E0B") // Amber
    dimColor       = lipgloss.Color("#9CA3AF") // Dim gray
)

// Styles
var (
    titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
    labelStyle   = lipgloss.NewStyle().Foreground(dimColor)
    valueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
    successStyle = lipgloss.NewStyle().Foreground(successColor).Bold(true)
    errorStyle   = lipgloss.NewStyle().Foreground(errorColor).Bold(true)
    warnStyle    = lipgloss.NewStyle().Foreground(warnColor)
    hintStyle    = lipgloss.NewStyle().Foreground(dimColor).Italic(true)
)

// Icons
const (
    iconCheck    = "✓"
    iconCross    = "✗"
    iconDatabase = "◉"
    iconServer   = "◎"
    iconUser     = "◇"
    iconInfo     = "●"
)

// Spinner frames
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
```

#### 3. Update Root Command (cli/root.go)

```go
package cli

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "runtime/debug"
    "strings"

    "github.com/charmbracelet/fang"
    "github.com/spf13/cobra"
)

var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
)

var dataDir string

func Execute(ctx context.Context) error {
    homeDir, _ := os.UserHomeDir()
    defaultDataDir := filepath.Join(homeDir, "data", "blueprint", "microblog")

    root := &cobra.Command{
        Use:   "microblog",
        Short: "A modern microblogging platform",
        Long: `Microblog is a self-hosted microblogging platform combining the best
features from X/Twitter, Threads, and Mastodon.

Features include:
  - Short-form posts with mentions and hashtags
  - Reply threads and conversations
  - Likes, reposts, and bookmarks
  - Following/followers social graph
  - Content warnings and visibility controls
  - Full-text search and trending topics`,
        SilenceUsage:  true,
        SilenceErrors: true,
    }

    root.Version = versionString()
    root.PersistentFlags().StringVar(&dataDir, "data", defaultDataDir, "Data directory")

    root.AddCommand(
        NewServe(),
        NewInit(),
        NewUser(),
    )

    if err := fang.Execute(ctx, root,
        fang.WithVersion(Version),
        fang.WithCommit(Commit),
    ); err != nil {
        fmt.Fprintln(os.Stderr, errorStyle.Render(iconCross+" "+err.Error()))
        return err
    }
    return nil
}

func versionString() string {
    if strings.TrimSpace(Version) != "" && Version != "dev" {
        return Version
    }
    if bi, ok := debug.ReadBuildInfo(); ok {
        if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
            return bi.Main.Version
        }
    }
    return "dev"
}
```

#### 4. Update Init Command (cli/init.go)

```go
func NewInit() *cobra.Command {
    return &cobra.Command{
        Use:   "init",
        Short: "Initialize the database",
        Long:  `Initialize the database and create all required tables.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            ui := NewUI()

            ui.Header(iconDatabase, "Initializing Database")
            ui.Info("Data directory", dataDir)

            // Create directory
            ui.StartSpinner("Creating data directory...")
            if err := os.MkdirAll(dataDir, 0755); err != nil {
                ui.StopSpinnerError("Failed to create directory")
                return fmt.Errorf("create data dir: %w", err)
            }
            ui.StopSpinner("Directory created", time.Since(start))

            // Initialize database
            dbPath := filepath.Join(dataDir, "microblog.duckdb")
            ui.StartSpinner("Opening database...")
            db, err := sql.Open("duckdb", dbPath)
            if err != nil {
                ui.StopSpinnerError("Failed to open database")
                return fmt.Errorf("open database: %w", err)
            }
            defer db.Close()

            store, err := duckdb.New(db)
            if err != nil {
                ui.StopSpinnerError("Failed to create store")
                return fmt.Errorf("create store: %w", err)
            }

            ui.StartSpinner("Creating tables...")
            start := time.Now()
            if err := store.Ensure(context.Background()); err != nil {
                ui.StopSpinnerError("Failed to initialize schema")
                return fmt.Errorf("initialize schema: %w", err)
            }
            ui.StopSpinner("Tables created", time.Since(start))

            ui.Success("Database initialized at:")
            ui.Hint(dbPath)
            return nil
        },
    }
}
```

#### 5. Update User Commands (cli/user.go)

Enhanced with styled output for list, create, verify, suspend operations.

#### 6. Entry Point (cmd/microblog/main.go)

```go
package main

import (
    "context"
    "os"

    "github.com/go-mizu/blueprints/microblog/cli"
)

var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
)

func main() {
    cli.Version = Version
    cli.Commit = Commit
    cli.BuildTime = BuildTime

    if err := cli.Execute(context.Background()); err != nil {
        os.Exit(1)
    }
}
```

## DuckDB Store Tests

### Test Strategy

Real database tests (not mocks) using in-memory DuckDB:

```go
// Helper to create test database
func setupTestDB(t *testing.T) (*sql.DB, func()) {
    t.Helper()
    db, err := sql.Open("duckdb", ":memory:")
    require.NoError(t, err)

    store, err := New(db)
    require.NoError(t, err)

    err = store.Ensure(context.Background())
    require.NoError(t, err)

    return db, func() { db.Close() }
}
```

### Test Files

#### store/duckdb/store_test.go

Core store tests:
- `TestNew` - Store creation
- `TestEnsure` - Schema initialization
- `TestStats` - Statistics gathering

#### store/duckdb/accounts_store_test.go

```go
func TestAccountsStore_Insert(t *testing.T)
func TestAccountsStore_GetByID(t *testing.T)
func TestAccountsStore_GetByUsername(t *testing.T)
func TestAccountsStore_GetByEmail(t *testing.T)
func TestAccountsStore_Update(t *testing.T)
func TestAccountsStore_ExistsUsername(t *testing.T)
func TestAccountsStore_ExistsEmail(t *testing.T)
func TestAccountsStore_GetPasswordHash(t *testing.T)
func TestAccountsStore_List(t *testing.T)
func TestAccountsStore_Search(t *testing.T)
func TestAccountsStore_SetVerified(t *testing.T)
func TestAccountsStore_SetSuspended(t *testing.T)
func TestAccountsStore_SetAdmin(t *testing.T)
func TestAccountsStore_Sessions(t *testing.T)
```

#### store/duckdb/posts_store_test.go

```go
func TestPostsStore_Insert(t *testing.T)
func TestPostsStore_GetByID(t *testing.T)
func TestPostsStore_Update(t *testing.T)
func TestPostsStore_Delete(t *testing.T)
func TestPostsStore_GetByAccountID(t *testing.T)
func TestPostsStore_Threading(t *testing.T)
```

#### store/duckdb/interactions_store_test.go

```go
func TestInteractionsStore_Like(t *testing.T)
func TestInteractionsStore_Unlike(t *testing.T)
func TestInteractionsStore_Repost(t *testing.T)
func TestInteractionsStore_Unrepost(t *testing.T)
func TestInteractionsStore_Bookmark(t *testing.T)
func TestInteractionsStore_Unbookmark(t *testing.T)
```

#### store/duckdb/relationships_store_test.go

```go
func TestRelationshipsStore_Follow(t *testing.T)
func TestRelationshipsStore_Unfollow(t *testing.T)
func TestRelationshipsStore_Block(t *testing.T)
func TestRelationshipsStore_Mute(t *testing.T)
func TestRelationshipsStore_GetFollowers(t *testing.T)
func TestRelationshipsStore_GetFollowing(t *testing.T)
```

#### store/duckdb/timelines_store_test.go

```go
func TestTimelinesStore_Home(t *testing.T)
func TestTimelinesStore_Local(t *testing.T)
func TestTimelinesStore_Hashtag(t *testing.T)
func TestTimelinesStore_Bookmarks(t *testing.T)
```

### Test Patterns

#### Table-Driven Tests

```go
func TestAccountsStore_GetByUsername(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    store := NewAccountsStore(db)
    ctx := context.Background()

    // Setup test account
    acct := &accounts.Account{
        ID:        "01ABCDEF",
        Username:  "testuser",
        Email:     "test@example.com",
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    require.NoError(t, store.Insert(ctx, acct, "hash"))

    tests := []struct {
        name     string
        username string
        wantID   string
        wantErr  bool
    }{
        {"exact match", "testuser", "01ABCDEF", false},
        {"case insensitive", "TestUser", "01ABCDEF", false},
        {"not found", "nobody", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := store.GetByUsername(ctx, tt.username)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            require.Equal(t, tt.wantID, got.ID)
        })
    }
}
```

#### Integration Tests

```go
func TestAccountsStore_FullLifecycle(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    store := NewAccountsStore(db)
    ctx := context.Background()

    // Create
    acct := &accounts.Account{...}
    require.NoError(t, store.Insert(ctx, acct, "hash"))

    // Read
    got, err := store.GetByID(ctx, acct.ID)
    require.NoError(t, err)
    require.Equal(t, acct.Username, got.Username)

    // Update
    bio := "Updated bio"
    require.NoError(t, store.Update(ctx, acct.ID, &accounts.UpdateIn{Bio: &bio}))

    got, _ = store.GetByID(ctx, acct.ID)
    require.Equal(t, bio, got.Bio)

    // List
    list, total, err := store.List(ctx, 10, 0)
    require.NoError(t, err)
    require.Equal(t, 1, total)
    require.Len(t, list, 1)

    // Sessions
    sess := &accounts.Session{...}
    require.NoError(t, store.CreateSession(ctx, sess))
    gotSess, err := store.GetSession(ctx, sess.Token)
    require.NoError(t, err)
    require.Equal(t, sess.AccountID, gotSess.AccountID)
}
```

## Implementation Steps

### Phase 1: CLI Modernization

1. Add Fang and Lipgloss dependencies to go.mod
2. Create `cli/ui.go` with styles and UI helper type
3. Update `cli/root.go` to use Fang execution
4. Update `cli/init.go` with styled output and spinners
5. Update `cli/serve.go` with startup banner
6. Update `cli/user.go` with table output for lists
7. Update `cmd/microblog/main.go` for new entry point

### Phase 2: Store Tests

1. Create `store/duckdb/store_test.go` with test helpers
2. Create `store/duckdb/accounts_store_test.go`
3. Create `store/duckdb/posts_store_test.go`
4. Create `store/duckdb/interactions_store_test.go`
5. Create `store/duckdb/relationships_store_test.go`
6. Create `store/duckdb/timelines_store_test.go`

### Phase 3: Verification

1. Build and run all commands
2. Run tests with race detector
3. Verify styled output in terminal

## Files Changed

```
blueprints/microblog/
├── go.mod                           # Add fang, lipgloss
├── cmd/microblog/main.go            # Updated entry point
├── cli/
│   ├── root.go                      # Fang wrapper
│   ├── ui.go                        # NEW: UI styles and helpers
│   ├── init.go                      # Styled output
│   ├── serve.go                     # Styled output
│   └── user.go                      # Styled output
└── store/duckdb/
    ├── store_test.go                # NEW: Core tests
    ├── accounts_store_test.go       # NEW: Accounts tests
    ├── posts_store_test.go          # NEW: Posts tests
    ├── interactions_store_test.go   # NEW: Interactions tests
    ├── relationships_store_test.go  # NEW: Relationships tests
    └── timelines_store_test.go      # NEW: Timelines tests
```

## Success Criteria

1. All CLI commands display styled output with colors and spinners
2. Help text is properly styled via Fang
3. All store operations have test coverage
4. Tests use real DuckDB (in-memory) not mocks
5. `go test ./blueprints/microblog/...` passes with no failures
6. Binary builds and runs correctly with `make build && microblog --help`
