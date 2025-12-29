# Code Browser and File View Enhancement

## Overview

This document outlines the implementation plan to enhance the GitHome code browser and file view to achieve 100% feature parity with GitHub.

## Current Issues

### 1. Repository Home Page - Empty Tree
- **Location**: `app/web/handler/page.go:588-626` (RepoHome)
- **Issue**: The `RepoHomeData` struct passed to template doesn't include populated `Tree` field
- **Root cause**: Handler doesn't call any method to get directory entries for root path
- **Template expects**: `.Tree` slice with entries (lines 98-121 in repo_home.html)

### 2. Tree View - Empty Directory
- **Location**: `app/web/handler/page.go:920-989` (RepoTree)
- **Issue**: `tree` variable is initialized empty and never populated (line 946)
- **Root cause**: Comment says "For now, we'll just show the directory structure" but no code follows
- **GetContents limitation**: Returns only Content metadata for directories, not entry list

### 3. Blob View - Base64 Instead of Decoded Content
- **Location**: `app/web/handler/page.go:992-1066` (RepoBlob)
- **Issue**: `content.Content` is base64-encoded (from GetContents line 722)
- **Handler passes**: `FileContent: content.Content` directly (line 1054)
- **Template displays**: Base64 string `IyBUaGUgR28gUHJvZ3JhbW1pbmcgTGFuZ3VhZ2U=`

### 4. README Display - Base64 Instead of Rendered Markdown
- **Location**: `app/web/handler/page.go:611-614` (RepoHome)
- **Issue**: `readme.Content` is base64, set directly as `template.HTML`
- **Expected**: Decode base64, then render Markdown to HTML

### 5. Missing GitHub Features
- No syntax highlighting (highlight.js not integrated)
- No Markdown preview for .md files
- No image rendering for image files
- No CSV/JSON preview
- Line linking (e.g., #L10-L20)
- File size display in human-readable format

## Architecture Changes Required

### Service Layer (`feature/repos/service.go`)

#### New Method: `ListTreeEntries`

```go
// TreeEntry represents a file or directory entry
type TreeEntry struct {
    Name        string `json:"name"`
    Path        string `json:"path"`
    SHA         string `json:"sha"`
    Size        int64  `json:"size"`
    Type        string `json:"type"` // "file", "dir", "symlink", "submodule"
    Mode        string `json:"mode"` // "100644", "100755", "040000", etc.
    URL         string `json:"url"`
    HTMLURL     string `json:"html_url"`
    DownloadURL string `json:"download_url,omitempty"`
}

// ListTreeEntries returns all entries in a directory
func (s *Service) ListTreeEntries(ctx context.Context, owner, repoName, path, ref string) ([]*TreeEntry, error)
```

**Implementation Steps:**
1. Open git repository
2. Resolve ref to commit SHA
3. Get commit and root tree
4. Navigate to target path (if not root)
5. For each entry in tree:
   - Build TreeEntry with name, path, type, size, mode
   - Generate URLs (content, HTML, download)
6. Sort: directories first, then files alphabetically
7. Return slice of TreeEntry

#### Enhanced Method: `GetContents`

Add a `DecodedContent` field or create a new method `GetDecodedContents`:

```go
// Content with decoded content for UI rendering
type ContentWithDecoded struct {
    *Content
    DecodedContent string `json:"-"` // Not in JSON, only for templates
}
```

Or modify GetContents to optionally decode:

```go
func (s *Service) GetContents(ctx context.Context, owner, repoName, path, ref string, decode bool) (*Content, error)
```

### Handler Layer (`app/web/handler/page.go`)

#### TreeEntry Type for Templates

```go
type TreeEntry struct {
    Name       string
    Path       string
    Type       string // "dir" or "file"
    Size       int64
    SizeHuman  string // "1.2 KB"
    Mode       string
    SHA        string
    LastCommit string // Optional: commit message preview
}
```

#### Fix RepoHome Handler

```go
func (h *Page) RepoHome(c *mizu.Ctx) error {
    // ... existing code ...

    // Get root tree entries
    entries, err := h.repos.ListTreeEntries(ctx, owner, repoName, "", repo.DefaultBranch)
    if err != nil {
        // Log error but continue - empty tree is acceptable
    }

    tree := make([]*TreeEntry, len(entries))
    for i, e := range entries {
        tree[i] = &TreeEntry{
            Name: e.Name,
            Path: e.Path,
            Type: e.Type,
            Size: e.Size,
            SizeHuman: humanizeBytes(e.Size),
        }
    }

    // Decode README if exists
    var readmeHTML template.HTML
    if readme != nil && readme.Content != "" {
        decoded, err := base64.StdEncoding.DecodeString(readme.Content)
        if err == nil {
            // Render Markdown to HTML
            rendered := renderMarkdown(decoded)
            readmeHTML = template.HTML(rendered)
        }
    }

    return render(h, c, "repo_home", RepoHomeData{
        // ...
        Tree:   tree,  // ADD THIS
        Readme: readmeHTML,
        // ...
    })
}
```

#### Fix RepoTree Handler

```go
func (h *Page) RepoTree(c *mizu.Ctx) error {
    // ... existing code ...

    // Get directory entries
    entries, err := h.repos.ListTreeEntries(ctx, owner, repoName, path, ref)
    if err != nil {
        return c.Text(http.StatusNotFound, "Path not found")
    }

    tree := make([]*TreeEntry, len(entries))
    for i, e := range entries {
        tree[i] = &TreeEntry{
            Name: e.Name,
            Path: e.Path,
            Type: e.Type,
            Size: e.Size,
            SizeHuman: humanizeBytes(e.Size),
        }
    }

    return render(h, c, "repo_code", RepoCodeData{
        // ...
        Tree: tree,  // NOW POPULATED
        // ...
    })
}
```

#### Fix RepoBlob Handler

```go
func (h *Page) RepoBlob(c *mizu.Ctx) error {
    // ... existing code ...

    // Decode base64 content
    var decodedContent string
    if content.Encoding == "base64" && content.Content != "" {
        decoded, err := base64.StdEncoding.DecodeString(content.Content)
        if err == nil {
            decodedContent = string(decoded)
        }
    } else {
        decodedContent = content.Content
    }

    // Check for Markdown preview
    var markdownHTML template.HTML
    isMarkdown := detectLanguage(path) == "markdown"
    if isMarkdown {
        rendered := renderMarkdown([]byte(decodedContent))
        markdownHTML = template.HTML(rendered)
    }

    return render(h, c, "repo_blob", RepoCodeData{
        // ...
        FileContent:  decodedContent,  // DECODED
        MarkdownHTML: markdownHTML,    // NEW: for .md files
        IsMarkdown:   isMarkdown,      // NEW
        // ...
    })
}
```

### Template Updates

#### repo_blob.html - Add Markdown Preview

```html
{{if .IsMarkdown}}
<div class="Box-body markdown-body p-4">
    {{.MarkdownHTML}}
</div>
{{else if .IsBinary}}
<!-- binary handling -->
{{else}}
<div class="Box-body p-0">
    <div class="code-view">
        <!-- code with line numbers -->
    </div>
</div>
{{end}}
```

#### Add highlight.js Integration

```html
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css">
<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
<script>hljs.highlightAll();</script>
```

#### Add Line Linking

```javascript
// Highlight lines from URL hash
function highlightLines() {
    const hash = window.location.hash;
    const match = hash.match(/^#L(\d+)(-L(\d+))?$/);
    if (match) {
        const start = parseInt(match[1]);
        const end = match[3] ? parseInt(match[3]) : start;
        for (let i = start; i <= end; i++) {
            const line = document.querySelector(`[data-line-number="${i}"]`);
            if (line) line.parentElement.classList.add('highlighted');
        }
    }
}
```

### New Helper Functions

```go
// humanizeBytes converts bytes to human readable format
func humanizeBytes(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// renderMarkdown converts markdown to HTML
func renderMarkdown(content []byte) string {
    // Use goldmark or blackfriday library
    md := goldmark.New(
        goldmark.WithExtensions(extension.GFM),
        goldmark.WithRendererOptions(html.WithHardWraps()),
    )
    var buf bytes.Buffer
    if err := md.Convert(content, &buf); err != nil {
        return string(content)
    }
    return buf.String()
}
```

## pkg/git Enhancements

### Current State

The `pkg/git` package provides:
- `GetTree(sha)` - Returns tree with entries
- `GetBlob(sha)` - Returns blob content
- `ResolveRef(name)` - Resolves ref to SHA

### Required Enhancements

#### 1. Add Recursive Tree Option

```go
// GetTreeRecursive returns tree with all nested entries flattened
func (r *Repository) GetTreeRecursive(sha string, maxDepth int) (*Tree, error)
```

#### 2. Add File Mode Constants

```go
const (
    FileModeFile       = "100644"
    FileModeExecutable = "100755"
    FileModeSymlink    = "120000"
    FileModeSubmodule  = "160000"
    FileModeDirectory  = "040000"
)
```

#### 3. Enhance TreeEntry

```go
type TreeEntry struct {
    Name string
    Mode FileMode
    Type ObjectType
    SHA  string
    Size int64  // Add size for blobs
    // Add method to determine if file/dir/symlink
}

func (e *TreeEntry) IsFile() bool
func (e *TreeEntry) IsDirectory() bool
func (e *TreeEntry) IsSymlink() bool
func (e *TreeEntry) IsSubmodule() bool
```

## Implementation Steps

### Phase 1: Service Layer (Core Functionality)

1. **Add `ListTreeEntries` method to repos service**
   - File: `feature/repos/service.go`
   - Returns sorted tree entries for a path
   - Directories first, then files alphabetically

2. **Add decoded content support**
   - Modify Content struct or add helper function
   - Decode base64 when needed for UI

3. **Add Markdown rendering**
   - Add goldmark dependency
   - Create renderMarkdown helper function

### Phase 2: Handler Updates

4. **Fix RepoHome handler**
   - Call ListTreeEntries for root
   - Decode README and render Markdown
   - Pass Tree to template

5. **Fix RepoTree handler**
   - Call ListTreeEntries for path
   - Pass populated Tree to template

6. **Fix RepoBlob handler**
   - Decode base64 content
   - Detect Markdown files
   - Render Markdown preview for .md files

### Phase 3: Template Enhancements

7. **Integrate highlight.js**
   - Add CDN links or bundle locally
   - Apply highlighting on page load

8. **Add Markdown preview**
   - Conditional rendering for .md files
   - Toggle between source and preview

9. **Add line linking**
   - JavaScript to parse URL hash
   - Highlight selected lines
   - Click to update URL

### Phase 4: Additional GitHub Features

10. **Image rendering**
    - Detect image file types
    - Display inline or download link

11. **CSV/JSON preview**
    - Parse and render as tables
    - Syntax highlight JSON

12. **File size display**
    - Show human-readable size
    - Add to file header

13. **Raw file endpoint**
    - Route: `/{owner}/{repo}/raw/{ref}/{path}`
    - Return decoded content with proper Content-Type

14. **Blame view**
    - Route: `/{owner}/{repo}/blame/{ref}/{path}`
    - Show line-by-line commit attribution

## File Changes Summary

| File | Changes |
|------|---------|
| `feature/repos/service.go` | Add `ListTreeEntries`, decode helpers |
| `feature/repos/api.go` | Add `TreeEntry` type if needed |
| `app/web/handler/page.go` | Fix RepoHome, RepoTree, RepoBlob |
| `assets/views/default/pages/repo_home.html` | Verify Tree iteration works |
| `assets/views/default/pages/repo_code.html` | Verify Tree iteration works |
| `assets/views/default/pages/repo_blob.html` | Add Markdown preview, highlight.js |
| `assets/views/default/layouts/default.html` | Add highlight.js CDN |
| `pkg/git/repository.go` | Optional: Add helper methods |
| `go.mod` | Add goldmark for Markdown rendering |

## Dependencies to Add

```
github.com/yuin/goldmark v1.6.0
github.com/yuin/goldmark-highlighting/v2 v2.0.0-20230729083705-37449abec8cc
```

## Testing

### Manual Testing URLs

After implementation, verify:

1. **Repo Home**: `http://localhost:8080/golang/go`
   - Should show file listing
   - Should render README.md

2. **Tree View**: `http://localhost:8080/golang/go/tree/master`
   - Should list all root entries
   - Directories first, files second

3. **Subdirectory**: `http://localhost:8080/golang/go/tree/master/src`
   - Should show subdirectory contents
   - Breadcrumbs should work

4. **Blob View**: `http://localhost:8080/golang/go/blob/master/README.md`
   - Should show decoded content
   - Should render Markdown preview

5. **Code File**: `http://localhost:8080/golang/go/blob/master/src/main.go`
   - Should show syntax highlighted code
   - Line numbers should be clickable

6. **Line Linking**: `http://localhost:8080/golang/go/blob/master/README.md#L5-L10`
   - Should scroll to and highlight lines 5-10

### Unit Tests

Add tests for:
- `ListTreeEntries` - verify sorting, path navigation
- `base64` decoding
- `renderMarkdown` - verify GFM support
- `humanizeBytes` - verify formatting
- `detectLanguage` - verify all extensions

## Success Criteria

- [ ] Repo home shows file/directory listing
- [ ] Tree view shows directory contents
- [ ] Blob view shows decoded file content
- [ ] README.md renders as Markdown
- [ ] Syntax highlighting works for code files
- [ ] Line numbers are clickable and linkable
- [ ] Directories sorted before files
- [ ] File sizes displayed in human-readable format
- [ ] Binary files show download option
- [ ] Image files render inline
