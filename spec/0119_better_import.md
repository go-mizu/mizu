# Spec 0119: Better Import Workflow & UX Enhancements

## Summary

Improve the finewiki import/serve workflow and overall user experience by:
1. Moving database seeding from `serve` to `import`
2. Adding beautiful CLI output with colors and animations (Bubble Tea style)
3. Enhancing article metadata display (word count, read time)
4. Fixing duplicate search box on home page
5. Improving all user-facing messages

## Current Issues

### 1. Seeding in Wrong Command
**Problem**: `finewiki serve zh` shows "Seeding database: tables are empty" during startup.

**Current flow**:
```
finewiki import zh  → Downloads parquet files only
finewiki serve zh   → Seeds DuckDB + builds indexes + starts server
```

**Expected flow**:
```
finewiki import zh  → Downloads parquet + seeds DuckDB + builds indexes
finewiki serve zh   → Just starts the server (fast)
```

### 2. Plain CLI Output
**Problem**: Current output is plain text with no visual hierarchy:
```
Downloading ZH Wikipedia
Source: huggingface.co/datasets/HuggingFaceFW/finewiki
Target: /Users/apple/data/blueprint/finewiki/zh/
Files: 1 (total: 256.3 MB)
Using: curl
[1/1] data.parquet (256.3 MB)
```

### 3. Duplicate Search Box on Home Page
**Problem**: Home page shows two search boxes:
- One in the topbar (always visible)
- One in the home content area

### 4. Missing Article Metadata
**Problem**: Article view only shows: wiki badge, language, last modified date.
Missing: word count, estimated reading time.

## Proposed Changes

### 1. Move Seeding to Import Command

**cli/import.go** - Add seeding after download:

```go
func runImport(ctx context.Context, dataDir, lang string) error {
    // ... existing download logic ...

    // NEW: Seed database after successful download
    if err := seedDatabase(ctx, dataDir, lang, ui); err != nil {
        return err
    }

    return nil
}

func seedDatabase(ctx context.Context, dataDir, lang string, ui *UI) error {
    ui.StartPhase("Preparing database")

    dbPath := DuckDBPath(dataDir, lang)
    parquetGlob := ParquetGlob(dataDir, lang)

    db, err := sql.Open("duckdb", dbPath)
    if err != nil {
        return err
    }
    defer db.Close()

    store, err := duckdb.New(db)
    if err != nil {
        return err
    }

    start := time.Now()

    if err := store.Ensure(ctx, duckdb.Config{
        ParquetGlob: parquetGlob,
        EnableFTS:   true,
    }, duckdb.EnsureOptions{
        SeedIfEmpty: true,
        BuildIndex:  true,
        BuildFTS:    true,
    }); err != nil {
        return err
    }

    ui.CompletePhase("Database ready", time.Since(start))
    return nil
}
```

**cli/serve.go** - Skip seeding, only verify:

```go
func runServe(ctx context.Context, addr, dataDir, lang string) error {
    // ... existing setup ...

    if err := store.Ensure(ctx, duckdb.Config{
        ParquetGlob: parquetGlob,
        EnableFTS:   true,
    }, duckdb.EnsureOptions{
        SeedIfEmpty: false,  // Don't seed during serve
        BuildIndex:  false,  // Already built during import
        BuildFTS:    false,
    }); err != nil {
        // If database is empty, prompt user to run import
        return fmt.Errorf("database not initialized\nrun 'finewiki import %s' first", lang)
    }

    // ... rest of serve logic ...
}
```

### 2. Beautiful CLI Output with Lipgloss

**cli/ui.go** - New UI package:

```go
package cli

import (
    "fmt"
    "time"

    "github.com/charmbracelet/lipgloss"
)

// Color palette
var (
    primaryColor   = lipgloss.Color("#10B981") // Emerald green
    secondaryColor = lipgloss.Color("#6B7280") // Gray
    accentColor    = lipgloss.Color("#3B82F6") // Blue
    warnColor      = lipgloss.Color("#F59E0B") // Amber
    errorColor     = lipgloss.Color("#EF4444") // Red
    successColor   = lipgloss.Color("#10B981") // Green
)

// Styles
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(primaryColor)

    subtitleStyle = lipgloss.NewStyle().
        Foreground(secondaryColor)

    labelStyle = lipgloss.NewStyle().
        Foreground(secondaryColor).
        Width(10)

    valueStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FFFFFF"))

    progressStyle = lipgloss.NewStyle().
        Foreground(accentColor)

    successStyle = lipgloss.NewStyle().
        Foreground(successColor).
        Bold(true)

    errorStyle = lipgloss.NewStyle().
        Foreground(errorColor).
        Bold(true)

    boxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(secondaryColor).
        Padding(0, 1)
)

// Icons
const (
    iconDownload = "↓"
    iconCheck    = "✓"
    iconCross    = "✗"
    iconArrow    = "→"
    iconDot      = "•"
    iconSpinner  = "◐"
    iconDatabase = "◉"
    iconFile     = "◎"
    iconTime     = "◷"
)

// UI handles formatted output
type UI struct {
    spinner *Spinner
}

func NewUI() *UI {
    return &UI{spinner: NewSpinner()}
}

// Header prints a styled header
func (u *UI) Header(title, subtitle string) {
    fmt.Println()
    fmt.Println(titleStyle.Render(title))
    if subtitle != "" {
        fmt.Println(subtitleStyle.Render(subtitle))
    }
    fmt.Println()
}

// Info prints a key-value pair
func (u *UI) Info(label, value string) {
    fmt.Printf("%s %s\n",
        labelStyle.Render(label+":"),
        valueStyle.Render(value))
}

// StartPhase starts a new operation phase with spinner
func (u *UI) StartPhase(message string) {
    u.spinner.Start(message)
}

// UpdatePhase updates the current spinner message
func (u *UI) UpdatePhase(message string) {
    u.spinner.Update(message)
}

// CompletePhase stops spinner and shows completion
func (u *UI) CompletePhase(message string, duration time.Duration) {
    u.spinner.Stop()
    fmt.Printf("%s %s %s\n",
        successStyle.Render(iconCheck),
        message,
        subtitleStyle.Render(fmt.Sprintf("(%s)", duration.Round(time.Millisecond))))
}

// Progress prints file download progress
func (u *UI) Progress(current, total int, filename, size string) {
    progress := fmt.Sprintf("[%d/%d]", current, total)
    fmt.Printf("%s %s %s\n",
        progressStyle.Render(progress),
        filename,
        subtitleStyle.Render(size))
}

// Success prints a success message
func (u *UI) Success(message string) {
    fmt.Println()
    fmt.Printf("%s %s\n", successStyle.Render(iconCheck), message)
}

// Error prints an error message
func (u *UI) Error(message string) {
    fmt.Printf("%s %s\n", errorStyle.Render(iconCross), message)
}

// Summary prints a completion summary box
func (u *UI) Summary(items map[string]string) {
    var lines []string
    for k, v := range items {
        lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render(k+":"), v))
    }
    // Print as simple list for now
    fmt.Println()
    for _, line := range lines {
        fmt.Println(line)
    }
}
```

**cli/spinner.go** - Animated spinner:

```go
package cli

import (
    "fmt"
    "sync"
    "time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Spinner struct {
    mu      sync.Mutex
    active  bool
    message string
    done    chan struct{}
}

func NewSpinner() *Spinner {
    return &Spinner{}
}

func (s *Spinner) Start(message string) {
    s.mu.Lock()
    if s.active {
        s.mu.Unlock()
        return
    }
    s.active = true
    s.message = message
    s.done = make(chan struct{})
    s.mu.Unlock()

    go func() {
        i := 0
        for {
            select {
            case <-s.done:
                fmt.Print("\r\033[K") // Clear line
                return
            default:
                s.mu.Lock()
                msg := s.message
                s.mu.Unlock()
                fmt.Printf("\r%s %s", progressStyle.Render(spinnerFrames[i]), msg)
                i = (i + 1) % len(spinnerFrames)
                time.Sleep(80 * time.Millisecond)
            }
        }
    }()
}

func (s *Spinner) Update(message string) {
    s.mu.Lock()
    s.message = message
    s.mu.Unlock()
}

func (s *Spinner) Stop() {
    s.mu.Lock()
    defer s.mu.Unlock()
    if !s.active {
        return
    }
    close(s.done)
    s.active = false
}
```

### 3. Updated Import Command Output

**Expected output**:
```
↓ Downloading Vietnamese Wikipedia
  Source: huggingface.co/datasets/HuggingFaceFW/finewiki
  Target: ~/data/blueprint/finewiki/vi/

  Files: 3 (256.3 MB total)
  Using: curl

⠋ Downloading data-000.parquet...
✓ data-000.parquet (85.4 MB) 12.3s
✓ data-001.parquet (85.4 MB) 11.8s
✓ data-002.parquet (85.5 MB) 12.1s

⠋ Preparing database...
✓ Database ready (142,857 articles indexed) 4.2s

────────────────────────────────────
✓ Import complete!

  Articles:  142,857
  Database:  ~/data/blueprint/finewiki/vi/wiki.duckdb
  Duration:  40.4s

  Run: finewiki serve vi
────────────────────────────────────
```

### 4. Fix Duplicate Search Box on Home Page

**Option A: Hide topbar search on home page** (Recommended)

Modify `views/layout/app.html`:
```html
{{if not .IsHome}}
<div class="topbar-search">
    <form action="/search" method="get" class="search-box">
        ...
    </form>
</div>
{{end}}
```

Pass `IsHome: true` from handler when rendering home page.

**Option B: Hide home page search, keep topbar only**

Simplify home.html to just show title and hint, no search form.

### 5. Add Article Metadata

**feature/view/api.go** - Add computed fields:

```go
type Page struct {
    // ... existing fields ...

    // Computed fields (not stored)
    WordCount      int    `json:"-"`
    ReadTimeMin    int    `json:"-"`
    ReadTimeStr    string `json:"-"`
}

// ComputeReadStats calculates word count and read time
func (p *Page) ComputeReadStats() {
    // Strip HTML and count words
    text := stripHTML(p.Text)
    words := strings.Fields(text)
    p.WordCount = len(words)

    // Average reading speed: 200-250 words per minute
    // Use 225 wpm as middle ground
    minutes := float64(p.WordCount) / 225.0
    p.ReadTimeMin = int(math.Ceil(minutes))

    if p.ReadTimeMin < 1 {
        p.ReadTimeStr = "< 1 min read"
    } else if p.ReadTimeMin == 1 {
        p.ReadTimeStr = "1 min read"
    } else {
        p.ReadTimeStr = fmt.Sprintf("%d min read", p.ReadTimeMin)
    }
}

func stripHTML(s string) string {
    // Simple HTML tag removal
    re := regexp.MustCompile(`<[^>]*>`)
    return re.ReplaceAllString(s, "")
}
```

**views/page/view.html** - Display metadata:

```html
<div class="article-meta">
    <span class="wiki-badge">{{.Page.WikiName}}</span>
    <span class="meta-dot">&middot;</span>
    <span>{{.Page.InLanguage}}</span>
    {{if .Page.WordCount}}
    <span class="meta-dot">&middot;</span>
    <span>{{.Page.WordCount | formatNumber}} words</span>
    {{end}}
    {{if .Page.ReadTimeStr}}
    <span class="meta-dot">&middot;</span>
    <span>{{.Page.ReadTimeStr}}</span>
    {{end}}
    {{if .Page.DateModifiedRel}}
    <span class="meta-dot">&middot;</span>
    <span title="{{.Page.DateModifiedFmt}}">Updated {{.Page.DateModifiedRel}}</span>
    {{end}}
    {{if .Page.URL}}
    <span class="meta-dot">&middot;</span>
    <a href="{{.Page.URL}}" target="_blank" rel="noopener">View on Wikipedia</a>
    {{end}}
</div>
```

### 6. Better Serve Command Output

**Expected output**:
```
◉ FineWiki Server

  Language:  Vietnamese (vi)
  Articles:  142,857
  Database:  ~/data/blueprint/finewiki/vi/wiki.duckdb

  Listening on http://localhost:8080

  Press Ctrl+C to stop
```

### 7. Friendly Error Messages

**Import errors**:
```
✗ Language 'xyz' not found

  Available languages: en, zh, vi, ja, ko, de, fr, es, ...

  Browse all: https://huggingface.co/datasets/HuggingFaceFW/finewiki
```

**Serve errors**:
```
✗ Database not initialized for 'vi'

  The parquet data hasn't been imported yet.

  Run: finewiki import vi
```

```
✗ Port 8080 already in use

  Another process is using this port.

  Try: finewiki serve vi --addr :8081
```

## Implementation Phases

### Phase 1: Core Workflow Fix
1. Move seeding from `serve` to `import`
2. Update `serve` to verify database exists
3. Add proper error messages

### Phase 2: CLI UI Enhancement
1. Add Lipgloss dependency
2. Create `cli/ui.go` with styles and helpers
3. Create `cli/spinner.go` for animated spinners
4. Update `import.go` to use new UI
5. Update `serve.go` to use new UI

### Phase 3: Web UX Enhancement
1. Fix duplicate search box (add `IsHome` flag)
2. Add word count and read time to Page struct
3. Update view template to show new metadata
4. Add `formatNumber` template function for thousands separator

### Phase 4: Polish
1. Add timing for all operations
2. Add summary box at end of import
3. Improve all error messages
4. Test on various terminal emulators

## Dependencies

- `github.com/charmbracelet/lipgloss` - Terminal styling
- No Bubble Tea needed (overkill for this use case, Lipgloss is sufficient)

## Files Changed

| File | Change |
|------|--------|
| `cli/import.go` | Add seeding, use new UI |
| `cli/serve.go` | Remove seeding, add verification |
| `cli/ui.go` | New file - UI helpers |
| `cli/spinner.go` | New file - Animated spinner |
| `store/duckdb/store.go` | Add callback for progress reporting |
| `feature/view/api.go` | Add word count, read time |
| `app/web/handlers.go` | Add `IsHome` flag, compute read stats |
| `app/web/templates.go` | Add `formatNumber` function |
| `app/web/views/layout/app.html` | Conditional topbar search |
| `app/web/views/page/view.html` | Show word count, read time |

## Testing

1. Fresh import: `rm -rf ~/data/blueprint/finewiki/vi && finewiki import vi`
2. Re-import (should skip): `finewiki import vi`
3. Serve after import: `finewiki serve vi`
4. Serve without import: `rm ~/data/blueprint/finewiki/vi/wiki.duckdb && finewiki serve vi`
5. Visual check: All messages are colored and formatted
6. Article view: Word count and read time display correctly

## Success Criteria

- [ ] `finewiki import` downloads + seeds database
- [ ] `finewiki serve` starts quickly without seeding
- [ ] All CLI output uses colors and icons
- [ ] Import shows animated spinner during operations
- [ ] Import shows total time at completion
- [ ] Home page has single search box
- [ ] Article view shows word count
- [ ] Article view shows estimated read time
- [ ] Error messages are helpful and suggest fixes
