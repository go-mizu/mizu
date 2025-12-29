# PR Commits and Files Changed Tabs Implementation

## Overview

Implement GitHub-style Commits and Files Changed tabs for Pull Request pages to match the exact look and feel of GitHub's PR interface.

## Current State

The current `pull_view.html` template has:
- All content rendered inline on a single page
- Basic anchor-based tab navigation (`#conversation`, `#commits`, `#files-changed`)
- Simple commits list without date grouping
- Basic diff table without file tree, collapsible sections, or viewed tracking

## Target State

Match GitHub's PR interface exactly as shown in the reference screenshots:

### 1. Tab Navigation Enhancement
- Real page routes for each tab (no JavaScript):
  - `/{owner}/{repo}/pull/{number}` - Conversation (default)
  - `/{owner}/{repo}/pull/{number}/commits` - Commits tab
  - `/{owner}/{repo}/pull/{number}/files` - Files changed tab
- Active tab styling with border highlight
- Diff stats positioned on the right side of tabs

### 2. Commits Tab (GitHub-style)

**Layout:**
- Commits grouped by date (e.g., "Commits on Sep 27, 2025")
- Each date group has a commit icon marker on left timeline

**Each Commit Row:**
```
[branch-prefix] commit message [...]          sha [copy] [browse]
  [avatar] author committed on date · [check] n/n
```

**Elements:**
- Branch prefix in brackets (e.g., `[klauspost:deflate-improve-comp]`)
- Commit message with expand icon if truncated
- Short SHA (7 chars) on right side
- Copy SHA button (clipboard icon)
- Browse files button (code icon `<>`)
- Author avatar (20x20)
- Author name as link
- Relative date
- Check status with icon (✓ n/n for passed, etc.)

### 3. Files Changed Tab (GitHub-style)

**Header Bar:**
```
[sidebar toggle] [All commits ▼]    X / Y viewed [filter input] [settings] [review icons] [Submit review ▼] [gear]
```

**Elements:**
- "All commits" dropdown to filter by commit
- View progress counter (e.g., "0 / 45 viewed")
- File filter input with placeholder "Filter files..."
- Settings/display options button
- Review conversation count
- Submit review button (dropdown with Approve/Request changes/Comment)

**File List:**
Each file section:
```
[collapse icon] path/to/file.go [copy] [expand]        +X -Y [status] [Viewed checkbox] [menu ...]
```

**Diff Display:**
- Split or unified view toggle
- Line numbers on both sides (old/new)
- Expand collapsed lines button (↕)
- Hunk header row with `@@ -115,9 +115,6 @@` style
- Addition lines (green background)
- Deletion lines (red background)
- Context lines (neutral)
- Line-level comment button on hover

## Implementation Plan

### Phase 1: Template Structure Changes

#### 1.1 Add New Routes

Add to router:
```go
// PR tabs - each tab is a separate page
r.GET("/:owner/:repo/pull/:number", h.PullDetail)           // Conversation
r.GET("/:owner/:repo/pull/:number/commits", h.PullCommits)  // Commits tab
r.GET("/:owner/:repo/pull/:number/files", h.PullFiles)      // Files changed tab
```

#### 1.2 Update Tab Navigation Template

Create shared partial `_pr_tabs.html`:
```html
<div class="tabnav-pr">
    <nav class="tabnav-tabs">
        <a href="/{{.Repo.FullName}}/pull/{{.Pull.Number}}" class="tabnav-tab{{if eq .ActiveTab "conversation"}} selected{{end}}">
            <svg>...</svg> Conversation
            {{if gt .Pull.Comments 0}}<span class="Counter">{{.Pull.Comments}}</span>{{end}}
        </a>
        <a href="/{{.Repo.FullName}}/pull/{{.Pull.Number}}/commits" class="tabnav-tab{{if eq .ActiveTab "commits"}} selected{{end}}">
            <svg>...</svg> Commits
            {{if gt .Pull.Commits 0}}<span class="Counter">{{.Pull.Commits}}</span>{{end}}
        </a>
        <a href="/{{.Repo.FullName}}/pull/{{.Pull.Number}}/files" class="tabnav-tab{{if eq .ActiveTab "files"}} selected{{end}}">
            <svg>...</svg> Files changed
            {{if gt .Pull.ChangedFiles 0}}<span class="Counter">{{.Pull.ChangedFiles}}</span>{{end}}
        </a>
    </nav>
    <div class="diffstat">
        <span class="color-fg-success">+{{formatNumber .Pull.Additions}}</span>
        <span class="color-fg-danger">-{{formatNumber .Pull.Deletions}}</span>
        <span class="diffstat-bar">...</span>
    </div>
</div>
```

#### 1.3 Commits Tab Template (`pull_commits.html`)

Standalone page for commits tab:
```html
{{define "content"}}
<!-- Repo header, nav, PR header - same as pull_view.html -->
{{template "_pr_header" .}}
{{template "_pr_tabs" .}}

<div class="container-xl px-3 px-md-4 px-lg-5 mt-4">
    <div class="commits-timeline">
        {{range .CommitGroups}}
        <div class="TimelineItem">
            <div class="TimelineItem-badge">
                <svg class="octicon octicon-git-commit" viewBox="0 0 16 16" width="16" height="16">
                    <path d="M11.93 8.5a4.002 4.002 0 0 1-7.86 0H.75a.75.75 0 0 1 0-1.5h3.32a4.002 4.002 0 0 1 7.86 0h3.32a.75.75 0 0 1 0 1.5Zm-1.43-.75a2.5 2.5 0 1 0-5 0 2.5 2.5 0 0 0 5 0Z"/>
                </svg>
            </div>
            <div class="TimelineItem-body">
                <span class="commit-group-title">Commits on {{.Date}}</span>
            </div>
        </div>
        {{range .Commits}}
        <div class="Box commit-item ml-7 mb-2">
            <div class="Box-row d-flex flex-items-start">
                <div class="flex-auto min-width-0">
                    <div class="d-flex flex-items-center">
                        <span class="commit-ref">[{{$.Pull.Head.Label}}]</span>
                        <a href="/{{$.Repo.FullName}}/commit/{{.SHA}}" class="Link--primary text-bold ml-1">
                            {{.MessageTitle}}
                        </a>
                        {{if .IsTruncated}}
                        <button class="ellipsis-expander ml-1">…</button>
                        {{end}}
                    </div>
                    <div class="d-flex flex-items-center mt-1 text-small color-fg-muted">
                        {{if .Author}}
                        <img src="{{.Author.AvatarURL}}" class="avatar avatar-user mr-1" width="20" height="20">
                        <a href="/{{.Author.Login}}" class="Link--secondary">{{.Author.Login}}</a>
                        {{else if .Commit.Author}}
                        <span>{{.Commit.Author.Name}}</span>
                        {{end}}
                        <span class="ml-1">committed on {{.DateShort}}</span>
                        <span class="commit-status ml-2">
                            <svg class="octicon color-fg-success" viewBox="0 0 16 16" width="16" height="16">
                                <path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"/>
                            </svg>
                            <span>1 / 1</span>
                        </span>
                    </div>
                </div>
                <div class="flex-shrink-0 d-flex flex-items-center ml-3">
                    <a href="/{{$.Repo.FullName}}/commit/{{.SHA}}" class="sha Link--secondary text-mono f6">{{truncate .SHA 7}}</a>
                    <button class="btn-octicon ml-2 tooltipped tooltipped-s" aria-label="Copy full SHA" data-clipboard-text="{{.SHA}}">
                        <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                            <path d="M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 0 1 0 1.5h-1.5a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-1.5a.75.75 0 0 1 1.5 0v1.5A1.75 1.75 0 0 1 9.25 16h-7.5A1.75 1.75 0 0 1 0 14.25Z"/>
                            <path d="M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0 1 14.25 11h-7.5A1.75 1.75 0 0 1 5 9.25Zm1.75-.25a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-7.5a.25.25 0 0 0-.25-.25Z"/>
                        </svg>
                    </button>
                    <a href="/{{$.Repo.FullName}}/tree/{{.SHA}}" class="btn-octicon ml-1 tooltipped tooltipped-s" aria-label="Browse files">
                        <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                            <path d="m11.28 3.22 4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.749.749 0 0 1-1.275-.326.749.749 0 0 1 .215-.734L13.94 8l-3.72-3.72a.749.749 0 0 1 .326-1.275.749.749 0 0 1 .734.215Zm-6.56 0a.751.751 0 0 1 1.042.018.751.751 0 0 1 .018 1.042L2.06 8l3.72 3.72a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L.47 8.53a.75.75 0 0 1 0-1.06Z"/>
                        </svg>
                    </a>
                </div>
            </div>
        </div>
        {{end}}
        {{end}}
    </div>
</div>
{{end}}
```

#### 1.4 Files Changed Tab Template (`pull_files.html`)

Standalone page for files changed tab:
```html
{{define "content"}}
{{template "_pr_header" .}}
{{template "_pr_tabs" .}}

<div class="container-xl px-3 px-md-4 px-lg-5 mt-4">
    <!-- Files Toolbar -->
    <div class="pr-review-toolbar d-flex flex-items-center mb-3 p-2 rounded-2" style="background: var(--bgColor-muted);">
        <div class="d-flex flex-items-center flex-auto">
            <button class="btn-octicon mr-2" type="button" aria-label="Toggle file tree">
                <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                    <path d="M0 1.75C0 .784.784 0 1.75 0h3.5C6.216 0 7 .784 7 1.75v3.5A1.75 1.75 0 0 1 5.25 7H4v4a1 1 0 0 0 1 1h4v-1.25C9 9.784 9.784 9 10.75 9h3.5c.966 0 1.75.784 1.75 1.75v3.5A1.75 1.75 0 0 1 14.25 16h-3.5A1.75 1.75 0 0 1 9 14.25v-.75H5A2.5 2.5 0 0 1 2.5 11V7h-.75A1.75 1.75 0 0 1 0 5.25Zm1.75-.25a.25.25 0 0 0-.25.25v3.5c0 .138.112.25.25.25h3.5a.25.25 0 0 0 .25-.25v-3.5a.25.25 0 0 0-.25-.25Zm9 9a.25.25 0 0 0-.25.25v3.5c0 .138.112.25.25.25h3.5a.25.25 0 0 0 .25-.25v-3.5a.25.25 0 0 0-.25-.25Z"/>
                </svg>
            </button>
            <details class="details-reset details-overlay">
                <summary class="btn btn-sm">
                    All commits
                    <span class="dropdown-caret"></span>
                </summary>
                <div class="SelectMenu SelectMenu--hasFilter">
                    <div class="SelectMenu-modal">
                        <div class="SelectMenu-list">
                            <a class="SelectMenu-item" href="#">All commits</a>
                            {{range .Commits}}
                            <a class="SelectMenu-item" href="#">{{truncate .SHA 7}} - {{truncate .Commit.Message 50}}</a>
                            {{end}}
                        </div>
                    </div>
                </div>
            </details>
        </div>
        <div class="d-flex flex-items-center">
            <span class="text-small color-fg-muted mr-3">
                <span id="files-viewed">0</span> / {{len .FileViews}} viewed
            </span>
            <input type="text" class="form-control form-control-sm mr-2" placeholder="Filter files..." style="width: 180px;">
            <div class="d-flex flex-items-center">
                <button class="btn-octicon mr-1" type="button">
                    <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                        <path d="M8 2c1.981 0 3.671.992 4.933 2.078 1.27 1.091 2.187 2.345 2.637 3.023a1.62 1.62 0 0 1 0 1.798c-.45.678-1.367 1.932-2.637 3.023C11.67 13.008 9.981 14 8 14c-1.981 0-3.671-.992-4.933-2.078C1.797 10.83.88 9.576.43 8.898a1.62 1.62 0 0 1 0-1.798c.45-.677 1.367-1.931 2.637-3.022C4.33 2.992 6.019 2 8 2ZM1.679 7.932a.12.12 0 0 0 0 .136c.411.622 1.241 1.75 2.366 2.717C5.176 11.758 6.527 12.5 8 12.5c1.473 0 2.825-.742 3.955-1.715 1.124-.967 1.954-2.096 2.366-2.717a.12.12 0 0 0 0-.136c-.412-.621-1.242-1.75-2.366-2.717C10.824 4.242 9.473 3.5 8 3.5c-1.473 0-2.824.742-3.955 1.715-1.124.967-1.954 2.096-2.366 2.717ZM8 10a2 2 0 1 1-.001-3.999A2 2 0 0 1 8 10Z"/>
                    </svg>
                </button>
                <span class="Counter mr-2">0</span>
            </div>
            <button class="btn btn-sm btn-primary">Submit review</button>
            <button class="btn-octicon ml-2" type="button">
                <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                    <path d="M8 0a8.2 8.2 0 0 1 .701.031C9.444.095 9.99.645 10.16 1.29l.288 1.107c.018.066.079.158.212.224.231.114.454.243.668.386.123.082.233.09.299.071l1.103-.303c.644-.176 1.392.021 1.82.63.27.385.506.792.704 1.218.315.675.111 1.422-.364 1.891l-.814.806c-.049.048-.098.147-.088.294.016.257.016.515 0 .772-.01.147.04.246.088.294l.814.806c.475.469.679 1.216.364 1.891a7.977 7.977 0 0 1-.704 1.217c-.428.61-1.176.807-1.82.63l-1.102-.302c-.067-.019-.177-.011-.3.071a5.909 5.909 0 0 1-.668.386c-.133.066-.194.158-.211.224l-.29 1.106c-.168.646-.715 1.196-1.458 1.26a8.006 8.006 0 0 1-1.402 0c-.743-.064-1.289-.614-1.458-1.26l-.289-1.106c-.018-.066-.079-.158-.212-.224a5.738 5.738 0 0 1-.668-.386c-.123-.082-.233-.09-.299-.071l-1.103.303c-.644.176-1.392-.021-1.82-.63a8.12 8.12 0 0 1-.704-1.218c-.315-.675-.111-1.422.363-1.891l.815-.806c.05-.048.098-.147.088-.294a6.214 6.214 0 0 1 0-.772c.01-.147-.04-.246-.088-.294l-.815-.806C.635 6.045.431 5.298.746 4.623a7.92 7.92 0 0 1 .704-1.217c.428-.61 1.176-.807 1.82-.63l1.102.302c.067.019.177.011.3-.071.214-.143.437-.272.668-.386.133-.066.194-.158.211-.224l.29-1.106C6.009.645 6.556.095 7.299.03 7.53.01 7.764 0 8 0Z"/>
                </svg>
            </button>
        </div>
    </div>

    <!-- Files List -->
    <div class="diff-view">
        {{range .FileViews}}
        <div class="file mb-4" id="diff-{{.SHA}}">
            <div class="file-header d-flex flex-items-center px-3 py-2" style="background: var(--bgColor-muted); border: 1px solid var(--borderColor-muted); border-bottom: 0; border-radius: 6px 6px 0 0;">
                <button class="btn-octicon mr-2" type="button" aria-expanded="true">
                    <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                        <path d="M12.78 5.22a.749.749 0 0 1 0 1.06l-4.25 4.25a.749.749 0 0 1-1.06 0L3.22 6.28a.749.749 0 1 1 1.06-1.06L8 8.939l3.72-3.719a.749.749 0 0 1 1.06 0Z"/>
                    </svg>
                </button>
                <div class="file-info flex-auto d-flex flex-items-center min-width-0">
                    <a href="#diff-{{.SHA}}" class="Link--primary text-mono f6 text-bold">{{.Filename}}</a>
                    <button class="btn-octicon ml-2" type="button" aria-label="Copy path">
                        <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                            <path d="M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 0 1 0 1.5h-1.5a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-1.5a.75.75 0 0 1 1.5 0v1.5A1.75 1.75 0 0 1 9.25 16h-7.5A1.75 1.75 0 0 1 0 14.25Z"/>
                            <path d="M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0 1 14.25 11h-7.5A1.75 1.75 0 0 1 5 9.25Zm1.75-.25a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-7.5a.25.25 0 0 0-.25-.25Z"/>
                        </svg>
                    </button>
                </div>
                <div class="file-actions d-flex flex-items-center">
                    <span class="diffstat text-small mr-3">
                        <span class="color-fg-success text-bold">+{{.Additions}}</span>
                        <span class="color-fg-danger text-bold">-{{.Deletions}}</span>
                        {{diffstatBlocks .Additions .Deletions}}
                    </span>
                    <label class="d-flex flex-items-center mr-2">
                        <input type="checkbox" class="mr-1 file-viewed-cb">
                        <span class="text-small color-fg-muted">Viewed</span>
                    </label>
                    <button class="btn-octicon" type="button">
                        <svg class="octicon" viewBox="0 0 16 16" width="16" height="16">
                            <path d="M8 9a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3ZM1.5 9a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Zm13 0a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z"/>
                        </svg>
                    </button>
                </div>
            </div>
            <div class="file-content" style="border: 1px solid var(--borderColor-muted); border-top: 0; border-radius: 0 0 6px 6px; overflow: hidden;">
                {{if .IsBinary}}
                <div class="text-center py-4 color-fg-muted">Binary file not shown</div>
                {{else if .TooLarge}}
                <div class="text-center py-4 color-fg-muted">Large diffs are not rendered by default.</div>
                {{else}}
                <table class="diff-table" style="width: 100%; border-collapse: collapse; font-family: monospace; font-size: 12px;">
                    <tbody>
                        {{range .DiffLines}}
                        {{if eq .Type "hunk"}}
                        <tr class="diff-hunk-row">
                            <td class="blob-num blob-num-hunk" style="width: 50px; min-width: 50px; padding: 0 10px; text-align: right; background: var(--diffBlob-hunk-bgColor-num); color: var(--fgColor-muted); user-select: none;">…</td>
                            <td class="blob-num blob-num-hunk" style="width: 50px; min-width: 50px; padding: 0 10px; text-align: right; background: var(--diffBlob-hunk-bgColor-num); color: var(--fgColor-muted); user-select: none;">…</td>
                            <td class="blob-code blob-code-hunk" style="padding: 0 10px; background: var(--diffBlob-hunk-bgColor-num); color: var(--fgColor-muted);">{{.Content}}</td>
                        </tr>
                        {{else}}
                        <tr class="diff-line-{{.Type}}">
                            <td class="blob-num" style="width: 50px; min-width: 50px; padding: 0 10px; text-align: right; user-select: none; {{if eq .Type "addition"}}background: var(--diffBlob-addition-bgColor-num);{{else if eq .Type "deletion"}}background: var(--diffBlob-deletion-bgColor-num);{{else}}background: var(--bgColor-muted);{{end}} color: var(--fgColor-muted);">{{if .OldLineNum}}{{.OldLineNum}}{{end}}</td>
                            <td class="blob-num" style="width: 50px; min-width: 50px; padding: 0 10px; text-align: right; user-select: none; {{if eq .Type "addition"}}background: var(--diffBlob-addition-bgColor-num);{{else if eq .Type "deletion"}}background: var(--diffBlob-deletion-bgColor-num);{{else}}background: var(--bgColor-muted);{{end}} color: var(--fgColor-muted);">{{if .NewLineNum}}{{.NewLineNum}}{{end}}</td>
                            <td class="blob-code" style="padding: 0 10px; line-height: 20px; white-space: pre-wrap; {{if eq .Type "addition"}}background: var(--diffBlob-addition-bgColor-line);{{else if eq .Type "deletion"}}background: var(--diffBlob-deletion-bgColor-line);{{end}}"><span class="blob-code-marker" style="display: inline-block; width: 16px; user-select: none;">{{if eq .Type "addition"}}+{{else if eq .Type "deletion"}}-{{else}} {{end}}</span><span class="blob-code-inner">{{.Content}}</span></td>
                        </tr>
                        {{end}}
                        {{end}}
                    </tbody>
                </table>
                {{end}}
            </div>
        </div>
        {{end}}
    </div>
</div>
{{end}}
```

### Phase 2: Backend Changes

#### 2.1 New View Models

Add to `handler/page.go`:

```go
// CommitGroup represents commits grouped by date
type CommitGroup struct {
    Date    string           // "Sep 27, 2025"
    DateKey string           // "2025-09-27" for sorting
    Commits []*CommitView
}

// CommitView extends pull commit with display fields
type CommitView struct {
    *pulls.Commit
    MessageTitle     string    // First line of commit message
    MessageBody      string    // Rest of commit message
    IsTruncated      bool      // Message > 72 chars
    DateShort        string    // "Sep 27"
    ChecksPassed     bool
    ChecksPassedCount int
    ChecksTotalCount  int
}
```

#### 2.2 Group Commits by Date

```go
func groupCommitsByDate(commits []*pulls.Commit) []*CommitGroup {
    groups := make(map[string]*CommitGroup)
    var order []string

    for _, c := range commits {
        date := c.Commit.Author.Date.Format("Jan 2, 2006")
        key := c.Commit.Author.Date.Format("2006-01-02")

        if _, ok := groups[key]; !ok {
            groups[key] = &CommitGroup{Date: date, DateKey: key}
            order = append(order, key)
        }
        groups[key].Commits = append(groups[key].Commits, &CommitView{
            Commit:       c,
            MessageTitle: firstLine(c.Commit.Message),
            IsTruncated:  len(firstLine(c.Commit.Message)) > 72,
            DateShort:    c.Commit.Author.Date.Format("Jan 2"),
        })
    }

    // Sort by date descending
    sort.Sort(sort.Reverse(sort.StringSlice(order)))

    result := make([]*CommitGroup, len(order))
    for i, k := range order {
        result[i] = groups[k]
    }
    return result
}
```

#### 2.3 Update PullDetailData

```go
type PullDetailData struct {
    // ... existing fields ...
    CommitGroups []*CommitGroup  // Commits grouped by date
    ActiveTab    string          // "conversation", "commits", "checks", "files"
}
```

### Phase 3: CSS Styling

#### 3.1 Tab Navigation Styles

```css
/* PR Tab Navigation */
.tabnav-pr {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    margin-bottom: -1px;
    border-bottom: 1px solid var(--borderColor-muted);
}

.tabnav-tabs {
    display: flex;
    gap: 0;
}

.tabnav-tab {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    font-size: 14px;
    font-weight: 400;
    color: var(--fgColor-muted);
    background: transparent;
    border: none;
    border-bottom: 2px solid transparent;
    cursor: pointer;
}

.tabnav-tab:hover {
    color: var(--fgColor-default);
}

.tabnav-tab[aria-selected="true"] {
    font-weight: 600;
    color: var(--fgColor-default);
    border-bottom-color: var(--underlineNav-borderColor-active, #fd8c73);
}
```

#### 3.2 Commits Tab Styles

```css
/* Commits Timeline */
.TimelineItem {
    position: relative;
    display: flex;
    padding: 16px 0 0;
}

.TimelineItem::before {
    content: "";
    position: absolute;
    top: 0;
    bottom: 0;
    left: 15px;
    width: 2px;
    background-color: var(--borderColor-muted);
}

.TimelineItem-badge {
    position: relative;
    z-index: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    margin-right: 16px;
    color: var(--fgColor-muted);
    background-color: var(--bgColor-muted);
    border: 2px solid var(--bgColor-default);
    border-radius: 50%;
}

.commit-group-title {
    font-size: 14px;
    font-weight: 400;
    color: var(--fgColor-muted);
    line-height: 32px;
}

/* Commit Item */
.commit-item {
    margin-left: 48px;
    margin-bottom: 8px;
}

.commit-item .Box-row {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-start;
    padding: 8px 16px;
}

.commit-message {
    flex: 1 1 auto;
    min-width: 0;
    font-size: 14px;
}

.commit-ref-prefix {
    display: inline;
    padding: 2px 6px;
    margin-right: 4px;
    font-size: 12px;
    font-family: monospace;
    color: var(--fgColor-accent);
    background-color: var(--bgColor-accent-muted);
    border-radius: 6px;
}

.commit-meta {
    width: 100%;
    margin-top: 4px;
    font-size: 12px;
    color: var(--fgColor-muted);
}

.commit-meta .avatar {
    vertical-align: middle;
    margin-right: 4px;
}

.commit-sha {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
    margin-left: auto;
}

.commit-sha .sha {
    font-family: monospace;
    font-size: 12px;
    color: var(--fgColor-muted);
}

.commit-status {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: 8px;
}
```

#### 3.3 Files Changed Tab Styles

```css
/* PR Toolbar */
.pr-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 16px;
    background-color: var(--bgColor-muted);
    border: 1px solid var(--borderColor-muted);
    border-radius: 6px 6px 0 0;
}

.pr-toolbar-left,
.pr-toolbar-right {
    display: flex;
    align-items: center;
    gap: 8px;
}

.files-viewed-count {
    font-weight: 600;
}

/* File Tree */
.file-tree-container {
    position: sticky;
    top: 60px;
    width: 260px;
    max-height: calc(100vh - 120px);
    overflow-y: auto;
    padding: 8px;
    border-right: 1px solid var(--borderColor-muted);
}

.file-tree-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 8px;
    font-size: 12px;
    color: var(--fgColor-default);
    text-decoration: none;
    border-radius: 6px;
}

.file-tree-item:hover {
    background-color: var(--bgColor-muted);
}

/* Diff File */
.file {
    margin-bottom: 16px;
    border: 1px solid var(--borderColor-muted);
    border-radius: 6px;
    overflow: hidden;
}

.file-header {
    display: flex;
    align-items: center;
    padding: 8px 16px;
    background-color: var(--bgColor-muted);
    border-bottom: 1px solid var(--borderColor-muted);
}

.file-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
}

.file-path {
    font-family: monospace;
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.file-actions {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-shrink: 0;
}

.diffstat-blocks {
    display: inline-flex;
    gap: 1px;
}

.diffstat-block {
    width: 8px;
    height: 8px;
    border-radius: 1px;
}

.diffstat-block-added { background-color: var(--diffBlob-addition-fgColor-num); }
.diffstat-block-deleted { background-color: var(--diffBlob-deletion-fgColor-num); }
.diffstat-block-neutral { background-color: var(--bgColor-neutral-muted); }

.viewed-checkbox {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--fgColor-muted);
    cursor: pointer;
}

/* Diff Table */
.diff-table {
    width: 100%;
    border-collapse: collapse;
    font-family: monospace;
    font-size: 12px;
    table-layout: fixed;
}

.diff-table col.blob-num {
    width: 50px;
}

.blob-num {
    width: 50px;
    min-width: 50px;
    padding: 0 10px;
    text-align: right;
    color: var(--fgColor-muted);
    background-color: var(--bgColor-default);
    vertical-align: top;
    cursor: pointer;
    user-select: none;
}

.blob-num::before {
    content: attr(data-line-number);
}

.blob-num:hover {
    color: var(--fgColor-accent);
}

.blob-code {
    padding: 0 10px;
    line-height: 20px;
    white-space: pre-wrap;
    word-wrap: break-word;
}

.blob-code-marker {
    display: inline-block;
    width: 16px;
    text-align: center;
    user-select: none;
}

/* Diff Colors */
.blob-num-addition { background-color: var(--diffBlob-addition-bgColor-num); }
.blob-code-addition { background-color: var(--diffBlob-addition-bgColor-line); }
.blob-code-addition .blob-code-marker { color: var(--diffBlob-addition-fgColor-text); }

.blob-num-deletion { background-color: var(--diffBlob-deletion-bgColor-num); }
.blob-code-deletion { background-color: var(--diffBlob-deletion-bgColor-line); }
.blob-code-deletion .blob-code-marker { color: var(--diffBlob-deletion-fgColor-text); }

.blob-num-hunk,
.blob-code-hunk {
    background-color: var(--diffBlob-hunk-bgColor-num);
    color: var(--fgColor-muted);
}

/* Expand Lines */
.blob-expand {
    cursor: pointer;
}

.blob-expand:hover {
    background-color: var(--bgColor-accent-muted);
}

.blob-expand td {
    text-align: center;
    padding: 4px;
}
```

### Phase 4: Template Functions

Add to `handler/page.go` template functions:

```go
// diffstatBlocks generates the 5-block diffstat visualization
func diffstatBlocks(additions, deletions int) template.HTML {
    total := additions + deletions
    if total == 0 {
        return template.HTML(`<span class="diffstat-block diffstat-block-neutral"></span>`.Repeat(5))
    }

    addBlocks := int(math.Round(float64(additions) / float64(total) * 5))
    delBlocks := 5 - addBlocks

    var html strings.Builder
    for i := 0; i < addBlocks; i++ {
        html.WriteString(`<span class="diffstat-block diffstat-block-added"></span>`)
    }
    for i := 0; i < delBlocks; i++ {
        html.WriteString(`<span class="diffstat-block diffstat-block-deleted"></span>`)
    }
    return template.HTML(html.String())
}

// diffLineClass returns CSS class for diff line type
func diffLineClass(lineType string) string {
    switch lineType {
    case "addition":
        return "blob-code-addition"
    case "deletion":
        return "blob-code-deletion"
    case "hunk":
        return "blob-code-hunk"
    default:
        return ""
    }
}

// diffMarker returns the +/- marker for diff lines
func diffMarker(lineType string) string {
    switch lineType {
    case "addition":
        return "+"
    case "deletion":
        return "-"
    default:
        return " "
    }
}
```

## File Changes Summary

| File | Changes |
|------|---------|
| `assets/views/default/pages/pull_view.html` | Update to be conversation-only page with shared header |
| `assets/views/default/pages/pull_commits.html` | New file - Commits tab page |
| `assets/views/default/pages/pull_files.html` | New file - Files changed tab page |
| `app/web/handler/page.go` | Add CommitGroup, CommitView structs; groupCommitsByDate function; add PullCommits and PullFiles handlers |
| `app/web/routes.go` | Add routes for /pull/:number/commits and /pull/:number/files |

## Testing Checklist

- [ ] Conversation tab displays PR description and comments
- [ ] Commits tab navigates to separate page
- [ ] Commits are grouped by date with timeline markers
- [ ] Commit messages display with branch prefix (e.g., [user:branch])
- [ ] Short SHA (7 chars) with copy and browse buttons
- [ ] Author avatar, name, and date displayed
- [ ] Check status icons displayed (✓ n/n)
- [ ] Files changed tab navigates to separate page
- [ ] Files display with proper diff coloring (green/red)
- [ ] File header shows filename, copy button, diff stats
- [ ] Diff stats show +X -Y with color blocks
- [ ] Viewed checkbox for each file
- [ ] Hunk headers (@@ -X,Y +X,Y @@) styled correctly
- [ ] Line numbers displayed on both sides
- [ ] Addition/deletion markers (+/-) displayed
- [ ] Responsive layout on mobile
- [ ] Dark mode colors are correct
- [ ] Tab navigation highlights current tab
