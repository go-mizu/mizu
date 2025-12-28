# Spec 0203: Test Services - Implementation Plan

## Overview

This document outlines the comprehensive plan to:
1. Identify and implement all placeholder methods in feature services
2. Review and fix any abstraction leaks (services using other features' Stores directly instead of Services)
3. Ensure all tests pass via `gotestsum ./...`

## Analysis Summary

### Services Reviewed

21 services were reviewed in `feature/`:
- activities, branches, collaborators, comments, commits, git, issues, labels, milestones, notifications, orgs, pulls, reactions, releases, repos, search, stars, teams, users, watches, webhooks

### Abstraction Leak Analysis

**Result: No abstraction leaks found in production code.**

All services correctly use dependency injection pattern:
- Services receive Store interfaces from other features through constructor parameters
- No service directly instantiates or accesses another feature's Store implementation

Example of correct pattern used throughout:
```go
type Service struct {
    store     Store       // Own domain store
    repoStore repos.Store // Injected dependency (interface)
    userStore users.Store // Injected dependency (interface)
}
```

## Placeholder Methods to Implement

### 1. repos/service.go

| Method | Line | Issue | Implementation |
|--------|------|-------|----------------|
| `ListContributors()` | 502-503 | Returns empty slice | Query commits from git, aggregate by author, return sorted by contribution count |
| `GetReadme()` | 515-516 | Returns nil | Use git service to find README.md in tree, return content |
| `GetContents()` | 528-529 | Returns nil | Use git service to traverse tree and return file/directory content |
| `CreateOrUpdateFile()` | 541-542 | Returns nil | Create blob, tree, commit via git service |
| `DeleteFile()` | 554-555 | Returns nil | Create tree without file, commit via git service |

**Dependencies needed:**
- Add `git.Service` or equivalent git access (via pkg/git)
- Add `reposDir` configuration for git repository paths

### 2. branches/service.go

| Method | Line | Issue | Implementation |
|--------|------|-------|----------------|
| `List()` | 36-47 | Returns only default branch | Use git.ListRefs to get all refs/heads/* branches |
| `Get()` | 60-70 | Returns placeholder SHA | Use git.GetRef to get actual branch ref |
| `Rename()` | 83-90 | Returns placeholder | Create new ref, delete old ref via git |

**Dependencies needed:**
- Add `gitService git.Service` or `reposDir string` to Service struct

### 3. commits/service.go

| Method | Line | Issue | Implementation |
|--------|------|-------|----------------|
| `List()` | 51-53 | Returns empty slice | Use git.Log to list commits from ref |
| `Get()` | 66-88 | Returns placeholder | Use git.GetCommit to get actual commit |
| `Compare()` | 101-117 | Returns empty comparison | Use git.Diff and log to compare commits |
| `ListBranchesForHead()` | 130-131 | Returns empty slice | Use git.ListRefs and check if branch contains commit |
| `ListPullsForCommit()` | 144-145 | Returns empty slice | Query pulls store for PRs matching head SHA |

**Dependencies needed:**
- Add git access to Service struct
- Add pulls.Store for ListPullsForCommit

### 4. pulls/service.go

| Method | Line | Issue | Implementation |
|--------|------|-------|----------------|
| `ListCommits()` | 190-191 | Returns empty slice | Use git.Log from base..head range |
| `ListFiles()` | 212-213 | Returns empty slice | Use git.Diff to get changed files |
| `UpdateBranch()` | 301-302 | Returns nil silently | Merge base into head via git |

**Dependencies needed:**
- Add git access to Service struct

### 5. issues/service.go

| Method | Line | Issue | Implementation |
|--------|------|-------|----------------|
| `ListForOrg()` | 279-280 | Returns empty slice | Query org by login, then list issues across org's repos |
| `ListAssignees()` | 308-309 | Returns empty slice | Return collaborators + owner for the repo |
| `getAuthorAssociation()` | 446-447 | Only checks ownership | Check collaborators, org membership, contributors |

**Dependencies needed:**
- Add `orgs.Store` to query org repos
- Add `collaborators.Store` for assignee/author association checks

### 6. labels/service.go

| Method | Line | Issue | Implementation |
|--------|------|-------|----------------|
| `ListForMilestone()` | 343-345 | Returns empty slice | Query issues with milestone, aggregate labels |

**Dependencies needed:**
- Add `milestones.Store` to look up milestone by number

## Implementation Order

### Phase 1: Add Git Integration Dependencies

1. Update service constructors to accept git access:
   - `repos.Service` - add reposDir and git package access
   - `branches.Service` - add reposDir and git package access
   - `commits.Service` - add reposDir and git package access
   - `pulls.Service` - add reposDir and git package access

### Phase 2: Implement Git-Based Methods

2. Implement in order of dependency:
   a. `branches.List()`, `branches.Get()`, `branches.Rename()`
   b. `commits.List()`, `commits.Get()`, `commits.Compare()`
   c. `commits.ListBranchesForHead()`
   d. `repos.GetContents()`, `repos.GetReadme()`
   e. `repos.CreateOrUpdateFile()`, `repos.DeleteFile()`
   f. `repos.ListContributors()`
   g. `pulls.ListCommits()`, `pulls.ListFiles()`, `pulls.UpdateBranch()`

### Phase 3: Add Store Dependencies for Cross-Feature Logic

3. Update service constructors:
   - `issues.Service` - add orgs.Store, collaborators.Store
   - `labels.Service` - add milestones.Store
   - `commits.Service` - add pulls.Store

4. Implement:
   a. `issues.ListForOrg()`
   b. `issues.ListAssignees()`
   c. `issues.getAuthorAssociation()`
   d. `labels.ListForMilestone()`
   e. `commits.ListPullsForCommit()`

### Phase 4: Test and Validate

5. Run `gotestsum ./...` on feature folder
6. Fix any failing tests
7. Add new tests for implemented methods

## Service Dependency Graph (Updated)

```
repos
├─ userStore
├─ orgStore
└─ reposDir (git access)

pulls
├─ repoStore
├─ userStore
└─ reposDir (git access)

issues
├─ repoStore
├─ userStore
├─ orgStore      (NEW: for ListForOrg)
└─ collaboratorsStore (NEW: for ListAssignees, getAuthorAssociation)

commits
├─ repoStore
├─ userStore
├─ pullsStore    (NEW: for ListPullsForCommit)
└─ reposDir (git access)

branches
├─ repoStore
└─ reposDir (git access)

labels
├─ repoStore
├─ issueStore
└─ milestonesStore (NEW: for ListForMilestone)
```

## Implementation Notes

### Git Integration Pattern

All git operations should follow this pattern:
```go
func (s *Service) getRepoPath(owner, repo string) string {
    return filepath.Join(s.reposDir, owner, repo+".git")
}

func (s *Service) openRepo(owner, repo string) (*pkggit.Repository, error) {
    path := s.getRepoPath(owner, repo)
    return pkggit.Open(path)
}
```

### Error Handling

- Return `ErrNotFound` for missing git objects
- Handle `pkggit.ErrNotARepository` for missing repos
- Handle `pkggit.ErrEmptyRepository` for repos with no commits

### URL Population

All returned objects should have URLs populated using the existing `populateURLs` helper pattern.

## Testing Strategy

1. Unit tests with mock stores
2. Integration tests with actual git repositories (created in temp directories)
3. Verify existing tests continue to pass

## Success Criteria

- All placeholder methods implemented with real business logic
- All tests pass via `gotestsum ./...`
- No abstraction leaks introduced
- Code follows existing patterns and conventions
