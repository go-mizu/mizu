# 0206: Code Browser

## Overview

Implement a GitHub-like code browser for GitHome that allows users to navigate repository file trees, view file contents with syntax highlighting, and browse through directory structures seamlessly.

## Goals

1. **100% GitHub UI parity** - Match GitHub's code browser layout, styling, and interactions
2. **Full navigation** - Support tree browsing, file viewing, branch switching, and breadcrumb navigation
3. **Syntax highlighting** - Display code with proper syntax coloring for all major languages
4. **Performance** - Handle large repositories like golang/go efficiently

## Routes

### Page Routes (Web UI)

| Route | Handler | Description |
|-------|---------|-------------|
| `GET /` | `Home` | Dashboard (authenticated) or landing page |
| `GET /login` | `Login` | Login page |
| `GET /register` | `Register` | Registration page |
| `GET /explore` | `Explore` | Explore repositories |
| `GET /new` | `NewRepo` | Create new repository |
| `GET /notifications` | `Notifications` | Notifications page |
| `GET /{owner}` | `UserProfile` | User profile page |
| `GET /{owner}/{repo}` | `RepoHome` | Repository home (default branch, root) |
| `GET /{owner}/{repo}/tree/{ref}` | `RepoTree` | Directory listing at ref (root) |
| `GET /{owner}/{repo}/tree/{ref}/{path...}` | `RepoTree` | Directory listing at path |
| `GET /{owner}/{repo}/blob/{ref}/{path...}` | `RepoBlob` | File content view |
| `GET /{owner}/{repo}/raw/{ref}/{path...}` | `RepoRaw` | Raw file download |
| `GET /{owner}/{repo}/blame/{ref}/{path...}` | `RepoBlame` | Blame view |
| `GET /{owner}/{repo}/commits/{ref}` | `RepoCommits` | Commit history |
| `GET /{owner}/{repo}/commit/{sha}` | `RepoCommit` | Single commit view |
| `GET /{owner}/{repo}/issues` | `RepoIssues` | Issues list |
| `GET /{owner}/{repo}/issues/{number}` | `IssueDetail` | Issue detail |
| `GET /{owner}/{repo}/issues/new` | `NewIssue` | Create issue |
| `GET /{owner}/{repo}/pulls` | `RepoPulls` | Pull requests list |
| `GET /{owner}/{repo}/settings` | `RepoSettings` | Repository settings |

## UI Components

### 1. Repository Header

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  📁 owner / repo-name   [Public]          [Watch ▼] 100  [★ Star] 500  [⑂ Fork] 50│
└──────────────────────────────────────────────────────────────────────────────┘
```

- Repository icon (book for public, lock for private)
- Owner link → user profile
- Repository name link → repo home
- Visibility badge (Public/Private)
- Action buttons: Watch, Star, Fork with counts

### 2. Repository Navigation Tabs

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ < > Code   ○ Issues 42   ⊘ Pull requests 5   ▷ Actions   ⚙ Settings        │
└──────────────────────────────────────────────────────────────────────────────┘
```

Tabs with icons and optional counts:
- **Code** (active when viewing files)
- **Issues** (count of open issues)
- **Pull requests** (count of open PRs)
- **Actions** (optional - if CI/CD enabled)
- **Wiki** (if enabled)
- **Security** (if enabled)
- **Settings** (if user has admin access)

### 3. Branch Selector & File Navigation Bar

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ [⑂ main ▼] [Branches: 5] [Tags: 12]           [Go to file] [Add file ▼] [<> Code ▼]│
└──────────────────────────────────────────────────────────────────────────────┘
```

Components:
- Branch/tag selector dropdown
- Branch count link
- Tag count link
- "Go to file" fuzzy finder button
- "Add file" dropdown (Create/Upload)
- "Code" clone dropdown with HTTPS/SSH/CLI

### 4. Breadcrumb Navigation

```
owner / repo / src / pkg / utils / helpers.go
```

- Each path segment is clickable
- Links to appropriate tree or blob view
- Current file/folder highlighted

### 5. Commit Bar (above file list)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ [👤] Author Name    commit message here...                    abc1234  2 days ago │
│                                                              [123 commits]   │
└──────────────────────────────────────────────────────────────────────────────┘
```

Shows:
- Last commit author avatar
- Author name (linked to profile)
- Commit message (linked to commit)
- Short SHA (linked to commit)
- Relative time
- Total commit count (linked to history)

### 6. File Tree / Directory Listing

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ 📁 .github              Initial commit                           3 days ago │
│ 📁 cmd                  Add CLI implementation                   2 days ago │
│ 📁 pkg                  Refactor utils package                   1 day ago  │
│ 📄 .gitignore           Add gitignore                            5 days ago │
│ 📄 README.md            Update documentation                     1 day ago  │
│ 📄 go.mod               Bump dependencies                        2 days ago │
└──────────────────────────────────────────────────────────────────────────────┘
```

Columns:
1. Icon (folder/file type icon)
2. Name (clickable link)
3. Last commit message (truncated, clickable)
4. Relative time

Sort order:
- Directories first, then files
- Alphabetical within each group

### 7. File Viewer (blob view)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ 📄 helpers.go   12.5 KB   318 lines (283 loc)   [Raw] [Blame] [History] [...] │
├──────────────────────────────────────────────────────────────────────────────┤
│   1 │ package utils                                                          │
│   2 │                                                                         │
│   3 │ import (                                                                │
│   4 │     "fmt"                                                               │
│   5 │     "strings"                                                           │
│   6 │ )                                                                       │
│ ... │ ...                                                                     │
└──────────────────────────────────────────────────────────────────────────────┘
```

Header:
- File icon
- File name
- File size (e.g., "12.5 KB")
- Line count with loc count
- Actions: Raw, Blame, History, Copy, Edit

Code display:
- Line numbers (fixed width, right-aligned)
- Syntax-highlighted code
- Clickable line numbers for permalinks
- Horizontal scroll for long lines

### 8. Right Sidebar (Repository Home)

```
┌─────────────────────┐
│ About               │
│ Description text... │
│ 🔗 website.com      │
│ 📦 release  📝 MIT  │
│ [tag1] [tag2] [go]  │
├─────────────────────┤
│ ★ 500 stars         │
│ 👁 100 watching     │
│ ⑂ 50 forks          │
├─────────────────────┤
│ Releases            │
│ v1.2.0  Latest      │
│ + 5 releases        │
├─────────────────────┤
│ Languages           │
│ ████████░░ Go 89.7% │
│ █░░░░░░░░░ JS 5.4%  │
├─────────────────────┤
│ Contributors        │
│ [👤][👤][👤] + 42   │
└─────────────────────┘
```

Sections:
- **About**: Description, website, topics
- **Stats**: Stars, watchers, forks
- **Releases**: Latest release, total count
- **Languages**: Progress bar visualization
- **Contributors**: Avatar list with overflow count

## Data Structures

### TreeEntry (Directory Item)

```go
type TreeEntry struct {
    Name       string    // File or directory name
    Path       string    // Full path from repo root
    Type       string    // "file", "dir", "submodule", "symlink"
    Size       int64     // File size in bytes (0 for dirs)
    SHA        string    // Git object SHA
    Mode       string    // File mode (e.g., "100644")
    Target     string    // Symlink target (if applicable)
    LastCommit *CommitInfo // Last commit that modified this entry
}

type CommitInfo struct {
    SHA       string
    Message   string    // First line only
    Author    string
    AuthorID  int64
    Timestamp time.Time
}
```

### FileContent (Blob View)

```go
type FileContent struct {
    Path       string
    Name       string
    Size       int64
    Content    string    // File content (text files)
    IsBinary   bool      // True if binary file
    Encoding   string    // Base64 for binary, utf-8 for text
    SHA        string
    LineCount  int
    LOC        int       // Lines of code (excluding blanks/comments)
    Language   string    // Detected language for highlighting
    Truncated  bool      // True if file too large
}
```

### RepoCodeData (Template Data)

```go
type RepoCodeData struct {
    Title         string
    User          *users.User
    Repo          *RepoView

    // Navigation context
    CurrentRef    string        // Branch/tag/commit SHA
    CurrentPath   string        // Current path in repo
    Branches      []*branches.Branch
    Tags          []*releases.Tag
    DefaultBranch string

    // Content (one of these populated)
    Tree          []*TreeEntry  // For directory view
    File          *FileContent  // For file view

    // Breadcrumb
    Breadcrumbs   []Breadcrumb

    // Commit info
    LastCommit    *CommitInfo
    CommitCount   int

    // Standard fields
    UnreadCount   int
    ActiveNav     string
}
```

## API Endpoints Required

The code browser relies on these Git API endpoints:

| Endpoint | Purpose |
|----------|---------|
| `GET /repos/{owner}/{repo}/contents/{path}` | Get file/dir contents |
| `GET /repos/{owner}/{repo}/git/trees/{sha}` | Get tree object |
| `GET /repos/{owner}/{repo}/git/blobs/{sha}` | Get blob (file) object |
| `GET /repos/{owner}/{repo}/branches` | List branches |
| `GET /repos/{owner}/{repo}/tags` | List tags |
| `GET /repos/{owner}/{repo}/commits` | List commits |
| `GET /repos/{owner}/{repo}/git/refs` | Get refs |

## Syntax Highlighting

### Client-Side Approach

Use [highlight.js](https://highlightjs.org/) or [Prism.js](https://prismjs.com/) for client-side highlighting:

```html
<link rel="stylesheet" href="/static/css/highlight.css">
<script src="/static/js/highlight.min.js"></script>
<script>hljs.highlightAll();</script>
```

Language detection by file extension:
- `.go` → Go
- `.js`, `.jsx` → JavaScript
- `.ts`, `.tsx` → TypeScript
- `.py` → Python
- `.rs` → Rust
- `.md` → Markdown
- etc.

### Styling

GitHub-like color scheme for code:
```css
/* Keywords */ .hljs-keyword { color: #cf222e; }
/* Strings */ .hljs-string { color: #0a3069; }
/* Comments */ .hljs-comment { color: #6e7781; }
/* Functions */ .hljs-function { color: #8250df; }
/* Numbers */ .hljs-number { color: #0550ae; }
/* Types */ .hljs-type { color: #953800; }
```

## Implementation Steps

### Phase 1: Page Router Setup

1. Add page handler to server initialization:
```go
// In server.go New()
pageHandler := handler.NewPage(templates, services...)
```

2. Register page routes:
```go
// Page routes (HTML)
app.Get("/", pageHandler.Home)
app.Get("/login", pageHandler.Login)
app.Get("/register", pageHandler.Register)
app.Get("/{owner}", pageHandler.UserProfile)
app.Get("/{owner}/{repo}", pageHandler.RepoHome)
app.Get("/{owner}/{repo}/tree/{ref}", pageHandler.RepoTree)
app.Get("/{owner}/{repo}/tree/{ref}/{path...}", pageHandler.RepoTree)
app.Get("/{owner}/{repo}/blob/{ref}/{path...}", pageHandler.RepoBlob)
// ... etc
```

### Phase 2: Handler Implementation

1. Implement `RepoTree` handler:
```go
func (h *Page) RepoTree(c *mizu.Ctx) error {
    owner := c.Param("owner")
    repo := c.Param("repo")
    ref := c.Param("ref")
    path := c.Param("path") // May be empty for root

    // Get repo
    repository, err := h.repos.Get(ctx, owner, repo)
    if err != nil {
        return c.Text(404, "Repository not found")
    }

    // Get tree contents
    tree, err := h.git.GetTree(ctx, owner, repo, ref, path)
    if err != nil {
        return c.Text(404, "Path not found")
    }

    // Populate last commit for each entry
    for _, entry := range tree {
        commit, _ := h.commits.GetForPath(ctx, owner, repo, ref, entry.Path)
        entry.LastCommit = commit
    }

    return render(h, c, "repo_code", RepoCodeData{
        Repo:        h.buildRepoView(ctx, repository, userID, "code"),
        CurrentRef:  ref,
        CurrentPath: path,
        Tree:        tree,
        Breadcrumbs: buildBreadcrumbs(owner, repo, ref, path),
    })
}
```

2. Implement `RepoBlob` handler:
```go
func (h *Page) RepoBlob(c *mizu.Ctx) error {
    // Similar to RepoTree but fetches file content
    content, err := h.git.GetBlob(ctx, owner, repo, ref, path)
    if err != nil {
        return c.Text(404, "File not found")
    }

    return render(h, c, "repo_code", RepoCodeData{
        File: content,
        // ...
    })
}
```

### Phase 3: Templates

1. Create `repo_code.html` template:
   - Shared header with repo tabs
   - Conditional rendering for tree vs blob view
   - Breadcrumb navigation
   - Branch/tag selector

2. Create partial templates:
   - `_file_tree.html` - Directory listing
   - `_file_viewer.html` - File content with line numbers
   - `_breadcrumb.html` - Path navigation
   - `_branch_selector.html` - Branch/tag dropdown

### Phase 4: Styling

1. Add code browser CSS to `main.css`:
   - File tree styles
   - Line number gutter
   - Syntax highlighting theme
   - Breadcrumb navigation
   - Responsive layout

### Phase 5: JavaScript

1. Add to `app.js`:
   - Branch selector dropdown behavior
   - Line number click → URL update
   - Keyboard navigation
   - Copy button functionality
   - Syntax highlighting initialization

## File Structure

```
assets/
├── static/
│   ├── css/
│   │   ├── main.css           # Main styles
│   │   └── highlight.css      # Syntax highlighting
│   └── js/
│       ├── app.js             # Main app JS
│       └── highlight.min.js   # Highlight.js library
└── views/
    └── default/
        ├── layouts/
        │   └── default.html   # Main layout
        └── pages/
            ├── repo_home.html   # Repository home (tree + readme)
            ├── repo_tree.html   # Directory listing (no readme)
            └── repo_blob.html   # File viewer
```

## Testing

### Test Repository

Use `$HOME/github/golang/go` as the test repository:

1. Large codebase (good for performance testing)
2. Variety of file types (.go, .s, .html, .md, etc.)
3. Deep directory structure
4. Many branches and tags
5. Well-known reference for UI comparison

### Test Cases

1. **Root tree view** - `/golang/go` should show root directory
2. **Subdirectory** - `/golang/go/tree/master/src/runtime` should show runtime files
3. **File view** - `/golang/go/blob/master/src/runtime/runtime.go` should show file with highlighting
4. **Binary file** - Should show "Binary file not shown" message
5. **Large file** - Should show truncation warning
6. **Branch switch** - Changing branch should update content
7. **Non-existent path** - Should show 404

## CSS Classes Reference

Key GitHub-like CSS classes to implement:

```css
/* File tree */
.file-wrap { }
.js-navigation-container { }
.react-directory-row { }
.react-directory-row-name-cell-large-screen { }

/* File viewer */
.react-code-file-contents { }
.react-blob-header-edit-and-raw-actions { }
.react-file-line { }
.react-line-number { }

/* Breadcrumb */
.react-tree-pane-header { }
.react-tree-show-tree-button { }

/* Branch selector */
.ref-selector-button { }
.ref-selector-button-text-container { }
```

## Performance Considerations

1. **Lazy loading** - Don't fetch last commit for all files upfront
2. **Caching** - Cache tree structures in memory
3. **Pagination** - Limit directory listings to 1000 entries
4. **Binary detection** - Check file headers, not content
5. **Large files** - Truncate at 1MB, show warning

## Security

1. **Path traversal** - Validate paths don't escape repo root
2. **Symlinks** - Handle symlinks carefully (optional: disallow)
3. **Private repos** - Enforce access control
4. **Rate limiting** - Limit requests per user

## Future Enhancements

1. **Blame view** - Show line-by-line authorship
2. **Commit history** - Per-file commit history
3. **File search** - Fuzzy file finder (Cmd+K)
4. **Code search** - Search within files
5. **Diff view** - Compare branches/commits
6. **Edit inline** - Edit files in browser
7. **Download** - Download as ZIP

## References

- [GitHub UI](https://github.com/golang/go)
- [GitHub Primer CSS](https://primer.style/css)
- [Highlight.js](https://highlightjs.org/)
- [Octicons](https://primer.style/foundations/icons)
