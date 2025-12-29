# Code Browser Enhancement Specification

## Overview

This specification outlines enhancements to GitHome's code browser to achieve feature parity with GitHub. The code browser consists of three main views: repository home, tree (directory) view, and blob (file) view.

## Current State Analysis

### Working Features
- Basic file/directory listing in repo home and tree views
- File content display with syntax highlighting (highlight.js)
- Markdown preview with code/preview toggle
- Line numbers with click-to-select
- URL hash-based line linking (#L1, #L1-L10)
- Raw file download
- Copy content button
- Binary file detection and download prompt

### Issues Identified
1. **Missing last commit info** - File listings don't show last commit message/author/date for each entry
2. **No commit header** - Repo home doesn't show the latest commit info at the top
3. **Missing branch/tag selector** - Dropdown exists but is not functional
4. **No file search** - "Go to file" button links but fuzzy search not implemented
5. **No blame view** - Link exists but page not implemented
6. **No file history** - No way to see commit history for a specific file
7. **No edit/delete** - Authenticated users can't edit files through the UI
8. **Image preview** - Binary images show download prompt instead of inline preview
9. **No contributors list** - Repo home sidebar is incomplete
10. **No commit count** - Total commits not shown

## Enhancement Plan

### Phase 1: Core Data Enhancements (pkg/git)

#### 1.1 Add Last Commit Per File
Add method to get the last commit that modified each file in a tree:

```go
// pkg/git/repository.go
type TreeEntryWithCommit struct {
    TreeEntry
    LastCommit *Commit
}

func (r *Repository) GetTreeWithLastCommits(sha string) ([]*TreeEntryWithCommit, error)
```

This requires walking the commit log to find when each file was last modified. For performance, consider:
- Caching results
- Limiting history depth (e.g., last 1000 commits)
- Async loading in UI

#### 1.2 Add File History
```go
// pkg/git/repository.go
func (r *Repository) FileLog(ref, path string, limit int) ([]*Commit, error)
```

Returns commits that modified a specific file path.

#### 1.3 Add Blame Information
```go
// pkg/git/types.go
type BlameLine struct {
    LineNumber int
    Content    string
    CommitSHA  string
    Author     Signature
    Date       time.Time
}

type BlameResult struct {
    Path  string
    Lines []BlameLine
}

// pkg/git/repository.go
func (r *Repository) Blame(ref, path string) (*BlameResult, error)
```

#### 1.4 Add Commit Count
```go
// pkg/git/repository.go
func (r *Repository) CommitCount(ref string) (int, error)
```

### Phase 2: Service Layer Updates (feature/repos)

#### 2.1 Enhanced Tree Entry Response
Update `TreeEntry` in `feature/repos/api.go`:

```go
type TreeEntry struct {
    Name          string    `json:"name"`
    Path          string    `json:"path"`
    Type          string    `json:"type"`
    Size          int64     `json:"size,omitempty"`
    SHA           string    `json:"sha"`
    URL           string    `json:"url"`
    HTMLURL       string    `json:"html_url"`
    GitURL        string    `json:"git_url"`
    DownloadURL   string    `json:"download_url,omitempty"`
    // New fields
    LastCommit    *CommitInfo `json:"last_commit,omitempty"`
}

type CommitInfo struct {
    SHA       string    `json:"sha"`
    Message   string    `json:"message"`
    Author    string    `json:"author"`
    Date      time.Time `json:"date"`
}
```

#### 2.2 Add Blame API
```go
// feature/repos/api.go
GetBlame(ctx context.Context, owner, repo, ref, path string) (*BlameResult, error)
```

#### 2.3 Add File History API
```go
// feature/repos/api.go
ListFileCommits(ctx context.Context, owner, repo, ref, path string, opts *ListOpts) ([]*Commit, error)
```

### Phase 3: API Handler Updates (app/web/handler/api)

#### 3.1 Add Blame Endpoint
```
GET /repos/{owner}/{repo}/blame/{ref}/{path}
```

#### 3.2 Add File Commits Endpoint
```
GET /repos/{owner}/{repo}/commits?path={path}&sha={ref}
```

Already exists per GitHub API, but ensure it filters by path.

### Phase 4: Web Page Updates (app/web/handler)

#### 4.1 Update RepoHomeData
```go
type RepoHomeData struct {
    // ... existing fields
    LatestCommit    *CommitView
    CommitCount     int
    Contributors    []*Contributor
}
```

#### 4.2 Update TreeEntry Handler Data
```go
type TreeEntry struct {
    // ... existing fields
    LastCommitMessage string
    LastCommitAuthor  string
    LastCommitDate    string // formatted as "2 days ago"
}
```

#### 4.3 Add Blame Page
New handler `RepoBlame` and template `repo_blame.html`.

#### 4.4 Add File History Page
New handler `RepoFileHistory` and template `repo_file_history.html`.

### Phase 5: Template Updates

#### 5.1 Repository Home (repo_home.html)
- Add commit header showing latest commit message, author, date, SHA
- Add commit count next to branch count
- File listing: add last commit message column (truncated)
- File listing: add relative date column

#### 5.2 Tree View (repo_code.html)
- Same file listing enhancements as repo_home
- Working branch/tag dropdown selector

#### 5.3 Blob View (repo_blob.html)
- Add "History" button linking to file history
- Add "Edit" button for authenticated users
- Add "Delete" button for authenticated users with permission
- Image files: Show inline preview instead of binary warning
- PDF files: Consider embedded PDF viewer or at minimum a preview

#### 5.4 New: Blame View (repo_blame.html)
- Show file with blame annotations
- Each line shows: commit SHA (abbreviated), author, date, line content
- Click commit SHA to go to commit
- Same header as blob view (breadcrumbs, branch selector, etc.)

#### 5.5 New: File History (repo_file_history.html)
- List of commits that modified this file
- Similar to main commits page but filtered

### Phase 6: Interactive Features

#### 6.1 Branch/Tag Selector
- Implement dropdown menu with JavaScript
- Load branches via AJAX: `GET /api/v3/repos/{owner}/{repo}/branches`
- Load tags via AJAX: `GET /api/v3/repos/{owner}/{repo}/tags`
- Switch branches: redirect to same path with new ref

#### 6.2 File Finder ("Go to file")
- New page `/repos/{owner}/{repo}/find/{ref}`
- Input field with fuzzy search
- Load file tree recursively via AJAX
- Client-side filtering as user types
- Arrow key navigation, Enter to select

#### 6.3 File Editor
- New page `/repos/{owner}/{repo}/edit/{ref}/{path}`
- CodeMirror or Monaco editor
- Commit message input
- Branch selection (commit to current or new branch)
- Create PR option when committing to new branch

### Phase 7: Image and Media Support

#### 7.1 Image Preview
Detect image files by extension and content-type:
- `.png`, `.jpg`, `.jpeg`, `.gif`, `.svg`, `.webp`, `.ico`

In blob view, instead of binary warning, show:
```html
<div class="Box-body text-center p-4">
    <img src="/{{.Repo.FullName}}/raw/{{.CurrentBranch}}/{{.FilePath}}"
         alt="{{.FileName}}"
         style="max-width: 100%; max-height: 80vh;">
</div>
```

#### 7.2 SVG Rendering
- Option to render SVG inline or show as code
- Security consideration: sanitize SVG or use sandbox

#### 7.3 PDF Support (Optional)
- Embed PDF.js viewer for PDF files
- Fallback to download link

## Implementation Priority

### Must Have (P0)
1. Last commit info per file in tree listing
2. Latest commit header on repo home
3. Working branch selector dropdown
4. Image preview for common formats
5. Blame view

### Should Have (P1)
1. File history view
2. File finder/search
3. Commit count on repo home
4. Contributors list

### Nice to Have (P2)
1. File editor
2. File delete
3. SVG inline rendering
4. PDF viewer

## File Changes Summary

### New Files
- `assets/views/default/pages/repo_blame.html`
- `assets/views/default/pages/repo_file_history.html`
- `assets/views/default/pages/repo_find.html`
- `assets/views/default/pages/repo_edit.html` (P2)

### Modified Files
- `pkg/git/repository.go` - Add GetTreeWithLastCommits, FileLog, Blame, CommitCount
- `pkg/git/types.go` - Add BlameResult, TreeEntryWithCommit types
- `feature/repos/api.go` - Add GetBlame, ListFileCommits, update TreeEntry
- `feature/repos/service.go` - Implement new methods
- `app/web/handler/page.go` - Add RepoBlame, RepoFileHistory, RepoFind handlers, update data structs
- `app/web/server.go` - Add new routes
- `assets/views/default/pages/repo_home.html` - Add commit header, enhance file listing
- `assets/views/default/pages/repo_code.html` - Enhance file listing, branch selector
- `assets/views/default/pages/repo_blob.html` - Add image preview, history/edit buttons

## API Compatibility

All new endpoints should follow GitHub API conventions for compatibility:
- `GET /repos/{owner}/{repo}/git/blobs/{sha}` - Blob by SHA (existing)
- `GET /repos/{owner}/{repo}/contents/{path}?ref={ref}` - Contents (existing)
- `GET /repos/{owner}/{repo}/commits?path={path}&sha={ref}` - File history (existing, needs filter)
- `GET /repos/{owner}/{repo}/blame/{ref}/{path}` - Blame (new, non-standard but useful)

## Testing

### Unit Tests
- `pkg/git/repository_test.go` - Test new methods with real git repos
- `feature/repos/service_test.go` - Test service layer

### Integration Tests
- Test blame view rendering
- Test file history pagination
- Test branch switching
- Test image preview for various formats

### Manual Testing
- Compare side-by-side with GitHub on golang/go repo
- Test with repos of various sizes
- Test edge cases: empty repo, single commit, binary files, large files

## Performance Considerations

### Last Commit Info
Getting last commit for each file requires walking commit history. Strategies:
1. **Lazy loading**: Load tree first, fetch commit info via AJAX
2. **Caching**: Cache results in database or memory
3. **Depth limit**: Only search recent N commits
4. **Batch API**: Single request for all entries' commit info

Recommended: Lazy loading for initial implementation, add caching later.

### Large Files
- Set max file size for inline display (e.g., 1MB)
- Large files show truncated preview with download link
- Very large repos: paginate tree entries

### Blame
- Blame can be slow for files with long history
- Consider streaming response or progress indicator
- Cache blame results

## Security Considerations

1. **Path traversal**: Validate paths don't escape repo
2. **XSS in file content**: Escape HTML in code display (already done via templates)
3. **SVG security**: Sanitize or sandbox SVG rendering
4. **File uploads**: When editing, validate content and paths
5. **Rate limiting**: Blame and history can be expensive

## Rollout Plan

1. Implement Phase 1-2 (data layer)
2. Deploy and test with internal repos
3. Implement Phase 3-4 (API and handlers)
4. Deploy and test web views
5. Implement Phase 5-6 (templates and interactivity)
6. Full testing against GitHub
7. Performance optimization based on real usage
