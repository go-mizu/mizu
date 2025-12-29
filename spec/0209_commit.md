# 0209 - Commit List and Single Commit View

## Overview

This specification defines the implementation of GitHub-compatible commit list and single commit view APIs for githome. The goal is 100% API compatibility with GitHub's REST API v3.

## GitHub API Reference

### List Commits Endpoint

**Endpoint:** `GET /repos/{owner}/{repo}/commits`

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `sha` | string | SHA or branch to start listing from. Default: default branch |
| `path` | string | Only commits containing this file path |
| `author` | string | GitHub username or email for filtering |
| `committer` | string | GitHub username or email for filtering by committer |
| `since` | string | ISO 8601 timestamp (YYYY-MM-DDTHH:MM:SSZ) |
| `until` | string | ISO 8601 timestamp |
| `per_page` | integer | Results per page (max 100, default 30) |
| `page` | integer | Page number (default 1) |

**Response:** Array of Commit objects (without `stats` and `files`)

### Get Single Commit Endpoint

**Endpoint:** `GET /repos/{owner}/{repo}/commits/{ref}`

**Parameters:**
- `ref`: SHA, branch name, or tag name

**Response:** Full Commit object (with `stats` and `files`)

## Data Structures

### Commit Object

```json
{
  "sha": "f4cec7917cc53c8c7ef2ea456b4bf0474c41189a",
  "node_id": "C_kwDOAWBuf9oAKGY0Y2VjNzkxN2NjNTNjOGM3ZWYyZWE0NTZiNGJmMDQ3NGM0MTE4OWE",
  "commit": {
    "author": {
      "name": "Lin Lin",
      "email": "linlin152@foxmail.com",
      "date": "2025-12-18T05:05:26Z"
    },
    "committer": {
      "name": "Sean Liao",
      "email": "sean@liao.dev",
      "date": "2025-12-27T21:02:20Z"
    },
    "message": "cmd: fix unused errors reported by ineffassign\n\nUpdates golang/go#35136\n\nChange-Id: I36d26089d29933e363d9fa50f3174530b698450e",
    "tree": {
      "sha": "2472184df91376ec03db2ce82036deafb0891f9c",
      "url": "https://api.github.com/repos/golang/go/git/trees/2472184df91376ec03db2ce82036deafb0891f9c"
    },
    "url": "https://api.github.com/repos/golang/go/git/commits/f4cec7917cc53c8c7ef2ea456b4bf0474c41189a",
    "comment_count": 0,
    "verification": {
      "verified": false,
      "reason": "unsigned",
      "signature": null,
      "payload": null,
      "verified_at": null
    }
  },
  "url": "https://api.github.com/repos/{owner}/{repo}/commits/{sha}",
  "html_url": "https://github.com/{owner}/{repo}/commit/{sha}",
  "comments_url": "https://api.github.com/repos/{owner}/{repo}/commits/{sha}/comments",
  "author": {
    "login": "username",
    "id": 29177310,
    "node_id": "MDQ6VXNlcjI5MTc3MzEw",
    "avatar_url": "https://avatars.githubusercontent.com/u/29177310?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/username",
    "html_url": "https://github.com/username",
    "followers_url": "https://api.github.com/users/username/followers",
    "following_url": "https://api.github.com/users/username/following{/other_user}",
    "gists_url": "https://api.github.com/users/username/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/username/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/username/subscriptions",
    "organizations_url": "https://api.github.com/users/username/orgs",
    "repos_url": "https://api.github.com/users/username/repos",
    "events_url": "https://api.github.com/users/username/events{/privacy}",
    "received_events_url": "https://api.github.com/users/username/received_events",
    "type": "User",
    "user_view_type": "public",
    "site_admin": false
  },
  "committer": { /* Same structure as author */ },
  "parents": [
    {
      "sha": "ca13fe02c48db993a34d441d87180cf665d5b288",
      "url": "https://api.github.com/repos/{owner}/{repo}/commits/{parent_sha}",
      "html_url": "https://github.com/{owner}/{repo}/commit/{parent_sha}"
    }
  ],
  "stats": {
    "total": 6,
    "additions": 6,
    "deletions": 0
  },
  "files": [
    {
      "sha": "bee3214b67576fa6b47998e9eb57b10af4022812",
      "filename": "src/cmd/internal/bootstrap_test/overlaydir_test.go",
      "status": "modified",
      "additions": 3,
      "deletions": 0,
      "changes": 3,
      "blob_url": "https://github.com/{owner}/{repo}/blob/{sha}/{path}",
      "raw_url": "https://github.com/{owner}/{repo}/raw/{sha}/{path}",
      "contents_url": "https://api.github.com/repos/{owner}/{repo}/contents/{path}?ref={sha}",
      "patch": "@@ -43,6 +43,9 @@ func overlayDir..."
    }
  ]
}
```

### CommitData Object

```go
type CommitData struct {
    URL          string        `json:"url"`
    Author       *CommitAuthor `json:"author"`
    Committer    *CommitAuthor `json:"committer"`
    Message      string        `json:"message"`
    Tree         *TreeRef      `json:"tree"`
    CommentCount int           `json:"comment_count"`
    Verification *Verification `json:"verification,omitempty"`
}
```

### CommitAuthor Object

```go
type CommitAuthor struct {
    Name  string    `json:"name"`
    Email string    `json:"email"`
    Date  time.Time `json:"date"`
}
```

### TreeRef Object

```go
type TreeRef struct {
    SHA string `json:"sha"`
    URL string `json:"url"`
}
```

### CommitRef Object (for parents)

```go
type CommitRef struct {
    SHA     string `json:"sha"`
    URL     string `json:"url"`
    HTMLURL string `json:"html_url"`
}
```

### CommitStats Object

```go
type CommitStats struct {
    Additions int `json:"additions"`
    Deletions int `json:"deletions"`
    Total     int `json:"total"`
}
```

### CommitFile Object

```go
type CommitFile struct {
    SHA              string `json:"sha"`
    Filename         string `json:"filename"`
    Status           string `json:"status"` // added, removed, modified, renamed, copied, changed, unchanged
    Additions        int    `json:"additions"`
    Deletions        int    `json:"deletions"`
    Changes          int    `json:"changes"`
    BlobURL          string `json:"blob_url"`
    RawURL           string `json:"raw_url"`
    ContentsURL      string `json:"contents_url"`
    Patch            string `json:"patch,omitempty"`
    PreviousFilename string `json:"previous_filename,omitempty"` // Only for renamed files
}
```

### Verification Object

```go
type Verification struct {
    Verified   bool    `json:"verified"`
    Reason     string  `json:"reason"`
    Signature  *string `json:"signature"`
    Payload    *string `json:"payload"`
    VerifiedAt *string `json:"verified_at"`
}
```

## Implementation Details

### 1. Enhanced List Commits

**Current Implementation Issues:**
- Missing `committer` filter parameter
- Not returning proper pagination headers
- Missing `verification` object in response

**Required Changes:**

1. Add `committer` filter to `ListOpts`:
```go
type ListOpts struct {
    Page      int       `json:"page,omitempty"`
    PerPage   int       `json:"per_page,omitempty"`
    SHA       string    `json:"sha,omitempty"`
    Path      string    `json:"path,omitempty"`
    Author    string    `json:"author,omitempty"`
    Committer string    `json:"committer,omitempty"` // NEW
    Since     time.Time `json:"since,omitempty"`
    Until     time.Time `json:"until,omitempty"`
}
```

2. Add pagination offset calculation:
```go
offset := (opts.Page - 1) * opts.PerPage
if offset < 0 {
    offset = 0
}
```

3. Add path filtering using `git.FileLog()` when path is specified

4. Add `verification` object to response (always `verified: false, reason: "unsigned"` for non-GPG signed commits)

### 2. Enhanced Get Single Commit

**Current Implementation Issues:**
- Missing `stats` calculation
- Missing `files` with diff information
- Missing `verification` object

**Required Changes:**

1. Calculate commit stats using diff:
```go
func (s *Service) calculateStats(gitRepo *pkggit.Repository, sha string) (*CommitStats, error) {
    commit, err := gitRepo.GetCommit(sha)
    if err != nil {
        return nil, err
    }

    // For first commit (no parents), compare with empty tree
    var parentSHA string
    if len(commit.Parents) > 0 {
        parentSHA = commit.Parents[0]
    }

    // Use git diff-tree to get stats
    stats, err := gitRepo.DiffStats(parentSHA, sha)
    return stats, err
}
```

2. Get files with diff information:
```go
func (s *Service) getCommitFiles(gitRepo *pkggit.Repository, owner, repo, sha string) ([]*CommitFile, error) {
    commit, err := gitRepo.GetCommit(sha)
    if err != nil {
        return nil, err
    }

    var parentSHA string
    if len(commit.Parents) > 0 {
        parentSHA = commit.Parents[0]
    }

    files, err := gitRepo.DiffFiles(parentSHA, sha)
    // Populate URLs for each file
    for _, f := range files {
        s.populateFileURLs(f, owner, repo, sha)
    }
    return files, err
}
```

### 3. New Git Package Functions

Add to `pkg/git/repository.go`:

```go
// DiffStat represents diff statistics
type DiffStat struct {
    Additions int
    Deletions int
    Total     int
}

// DiffFile represents a file in a diff
type DiffFile struct {
    SHA              string
    Filename         string
    Status           string // added, removed, modified, renamed
    Additions        int
    Deletions        int
    Changes          int
    Patch            string
    PreviousFilename string // for renamed files
}

// DiffStats returns the diff statistics between two commits
func (r *Repository) DiffStats(fromSHA, toSHA string) (*DiffStat, error) {
    // Use native git: git diff --stat fromSHA toSHA
}

// DiffFiles returns detailed file changes between two commits
func (r *Repository) DiffFiles(fromSHA, toSHA string) ([]*DiffFile, error) {
    // Use native git: git diff --name-status fromSHA toSHA
    // Then git diff --numstat fromSHA toSHA for line counts
    // Then git diff fromSHA toSHA -- <file> for patch content
}
```

### 4. URL Templates

All URLs must follow GitHub's format:

```go
const (
    // API URLs
    apiCommitURL     = "%s/api/v3/repos/%s/%s/commits/%s"
    apiCommentsURL   = "%s/api/v3/repos/%s/%s/commits/%s/comments"
    apiTreeURL       = "%s/api/v3/repos/%s/%s/git/trees/%s"
    apiContentsURL   = "%s/api/v3/repos/%s/%s/contents/%s?ref=%s"

    // HTML URLs
    htmlCommitURL    = "%s/%s/%s/commit/%s"
    htmlBlobURL      = "%s/%s/%s/blob/%s/%s"
    htmlRawURL       = "%s/%s/%s/raw/%s/%s"
)
```

### 5. HTTP Handler Updates

Update `app/web/handler/api/commit.go`:

```go
// ListCommits handles GET /repos/{owner}/{repo}/commits
func (h *CommitHandler) ListCommits(c *mizu.Ctx) error {
    owner := c.Param("owner")
    repoName := c.Param("repo")

    pagination := GetPagination(c)

    // Parse time filters
    var since, until time.Time
    if s := c.Query("since"); s != "" {
        since, _ = time.Parse(time.RFC3339, s)
    }
    if u := c.Query("until"); u != "" {
        until, _ = time.Parse(time.RFC3339, u)
    }

    opts := &commits.ListOpts{
        Page:      pagination.Page,
        PerPage:   pagination.PerPage,
        SHA:       c.Query("sha"),
        Path:      c.Query("path"),
        Author:    c.Query("author"),
        Committer: c.Query("committer"), // NEW
        Since:     since,
        Until:     until,
    }

    commitList, err := h.commits.List(c.Context(), owner, repoName, opts)
    if err != nil {
        return WriteError(c, http.StatusInternalServerError, err.Error())
    }

    // Add Link header for pagination
    // Link: <url>; rel="next", <url>; rel="last"

    return c.JSON(http.StatusOK, commitList)
}
```

## Test Plan

### Unit Tests

1. **TestService_List_DefaultPagination** - Default 30 items, page 1
2. **TestService_List_CustomPagination** - Custom per_page and page
3. **TestService_List_MaxPerPage** - Ensure max 100 items
4. **TestService_List_BySHA** - List from specific SHA/branch
5. **TestService_List_ByPath** - Filter by file path
6. **TestService_List_ByAuthor** - Filter by author
7. **TestService_List_ByCommitter** - Filter by committer
8. **TestService_List_BySince** - Filter by since timestamp
9. **TestService_List_ByUntil** - Filter by until timestamp
10. **TestService_List_CombinedFilters** - Multiple filters together

11. **TestService_Get_BySHA** - Get commit by full SHA
12. **TestService_Get_ByShortSHA** - Get by abbreviated SHA (7+ chars)
13. **TestService_Get_ByBranch** - Get by branch name
14. **TestService_Get_ByTag** - Get by tag name
15. **TestService_Get_WithStats** - Verify stats calculation
16. **TestService_Get_WithFiles** - Verify files list
17. **TestService_Get_WithPatch** - Verify patch content
18. **TestService_Get_RenamedFile** - Verify previous_filename

### Integration Tests

Test against real repository (golang/go mirror):

1. **TestGitHubCompatibility_ListCommits** - Compare response structure
2. **TestGitHubCompatibility_GetCommit** - Compare response structure
3. **TestGitHubCompatibility_CommitStats** - Verify stats match
4. **TestGitHubCompatibility_CommitFiles** - Verify files match

### Compatibility Test Script

```go
func TestGitHubCompatibility(t *testing.T) {
    // Clone golang/go to test directory
    // Start githome server
    // Compare responses field by field

    testCases := []struct {
        name     string
        endpoint string
    }{
        {"list_commits", "/repos/golang/go/commits?per_page=5"},
        {"get_commit", "/repos/golang/go/commits/f4cec7917cc53c8c7ef2ea456b4bf0474c41189a"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            githomeResp := fetchGithome(tc.endpoint)
            githubResp := fetchGitHub(tc.endpoint)
            compareResponses(t, githomeResp, githubResp)
        })
    }
}
```

## File Changes Summary

| File | Changes |
|------|---------|
| `feature/commits/api.go` | Add `Committer` to ListOpts, add Verification type |
| `feature/commits/service.go` | Add stats/files calculation, add path filtering |
| `feature/commits/service_test.go` | Add new test cases |
| `pkg/git/repository.go` | Add DiffStats, DiffFiles functions |
| `pkg/git/types.go` | Add DiffStat, DiffFile types |
| `app/web/handler/api/commit.go` | Add committer param, add since/until parsing |

## Verification Checklist

- [ ] List commits returns correct structure
- [ ] List commits pagination works (per_page, page)
- [ ] List commits filters work (sha, path, author, committer, since, until)
- [ ] Get commit returns full structure with stats and files
- [ ] Stats (additions, deletions, total) are accurate
- [ ] Files list includes all changed files
- [ ] File status is correct (added, modified, removed, renamed)
- [ ] Patch content is correct
- [ ] URLs follow GitHub format exactly
- [ ] Verification object is present (even if unsigned)
- [ ] Parent commits include html_url
- [ ] Empty repository returns empty array (not error)
- [ ] Non-existent commit returns 404
