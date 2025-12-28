# GitHome DuckDB Schema v2 Migration Spec

## Overview

This spec describes the comprehensive migration from GitHome's DuckDB schema v1 to v2. The v2 schema introduces several architectural improvements:

1. **Actors abstraction**: Unified owner model for users and orgs via `actors` table
2. **Composite primary keys**: Link tables use `(entity_id, related_id)` PKs instead of surrogate IDs
3. **Normalized token scopes**: `api_token_scopes` table replaces CSV `scopes` column
4. **Normalized repo topics**: `repo_topics` table replaces CSV `topics` column
5. **New repo_storage**: Track storage backend and path for each repository
6. **Enhanced PR model**: Added `lock_reason`, `merge_method`, `merge_message` fields
7. **Proper constraints**: CHECK constraints for valid states, UNIQUE constraints for idempotency

## Schema Comparison

### Tables Changed

| Table | v1 | v2 | Changes |
|-------|----|----|---------|
| `users` | exists | exists | No structural changes |
| `sessions` | exists | exists | Added FK to users |
| `ssh_keys` | exists | exists | Added `UNIQUE(user_id, fingerprint)` |
| `api_tokens` | `scopes VARCHAR` | removed `scopes` | Normalized to `api_token_scopes` |
| `organizations` | exists | exists | No structural changes |
| `org_members` | `id VARCHAR PK` | `PRIMARY KEY(org_id, user_id)` | Composite PK, no ID column |
| `teams` | exists | exists | Added `UNIQUE(org_id, slug)` |
| `team_members` | `id VARCHAR PK` | `PRIMARY KEY(team_id, user_id)` | Composite PK, no ID column |
| `team_repos` | `id VARCHAR PK` | `PRIMARY KEY(team_id, repo_id)` | Composite PK, no ID column |
| `repositories` | `owner_id + owner_type` | `owner_actor_id` | Uses actors, `forked_from_repo_id` |
| `collaborators` | `id VARCHAR PK` | `PRIMARY KEY(repo_id, user_id)` | Composite PK, no ID column |
| `stars` | `id VARCHAR PK` | `PRIMARY KEY(user_id, repo_id)` | Composite PK, no ID column |
| `watches` | `id VARCHAR PK` | `PRIMARY KEY(user_id, repo_id)` | Composite PK, no ID column |
| `labels` | exists | exists | Added `UNIQUE(repo_id, name)` |
| `milestones` | exists | exists | Added `UNIQUE(repo_id, number)` |
| `issues` | `assignee_id` single | removed single | Multi-assignee via `issue_assignees` |
| `issue_labels` | `id VARCHAR PK` | `PRIMARY KEY(issue_id, label_id)` | Composite PK |
| `issue_assignees` | `id VARCHAR PK` | `PRIMARY KEY(issue_id, user_id)` | Composite PK |
| `pull_requests` | exists | exists | Added `lock_reason`, `merge_method`, `merge_message` |
| `pr_labels` | `id VARCHAR PK` | `PRIMARY KEY(pr_id, label_id)` | Composite PK |
| `pr_assignees` | `id VARCHAR PK` | `PRIMARY KEY(pr_id, user_id)` | Composite PK |
| `pr_reviewers` | `id VARCHAR PK` | `PRIMARY KEY(pr_id, user_id)` | Composite PK |

### New Tables

| Table | Purpose |
|-------|---------|
| `actors` | Unifies user/org ownership via polymorphic relation |
| `api_token_scopes` | Normalized token scopes with `PRIMARY KEY(token_id, scope)` |
| `repo_topics` | Normalized topics with `PRIMARY KEY(repo_id, topic)` |
| `repo_storage` | Maps repo to storage backend (fs/s3/r2) |

### Removed Tables

None - all v1 tables are preserved or enhanced.

## Implementation Plan

### Phase 1: Schema Migration

1. Replace `store/duckdb/schema.sql` with v2 schema
2. The `Store.Ensure()` method will create new tables on startup

### Phase 2: Feature Model Updates

#### 2.1 Repos Feature (`feature/repos/api.go`)

```go
// Add to Repository struct
type Repository struct {
    // ... existing fields ...
    OwnerActorID string `json:"owner_actor_id"` // NEW: replaces OwnerID+OwnerType
    // OwnerID and OwnerType kept for compatibility, populated from actor join
}
```

#### 2.2 Users Feature (`feature/users/api.go`)

```go
// APIToken: Scopes changed from string to []string
type APIToken struct {
    // ...
    Scopes     []string   `json:"scopes"` // Changed from string
    // ...
}
```

### Phase 3: Store Updates

#### 3.1 `actors_store.go` (NEW)

```go
package duckdb

type ActorsStore struct {
    db *sql.DB
}

// Actor represents a unified owner (user or org)
type Actor struct {
    ID        string
    ActorType string // "user" or "org"
    UserID    string
    OrgID     string
    CreatedAt time.Time
}

func (s *ActorsStore) Create(ctx context.Context, a *Actor) error
func (s *ActorsStore) GetByID(ctx context.Context, id string) (*Actor, error)
func (s *ActorsStore) GetByUserID(ctx context.Context, userID string) (*Actor, error)
func (s *ActorsStore) GetByOrgID(ctx context.Context, orgID string) (*Actor, error)
func (s *ActorsStore) Delete(ctx context.Context, id string) error
```

#### 3.2 `repos_store.go` Updates

**Key Changes:**
- Replace `owner_id, owner_type` with `owner_actor_id`
- Replace `topics` string column with `repo_topics` table
- Replace `forked_from_id` with `forked_from_repo_id`
- Star operations: Remove ID field, use composite PK `(user_id, repo_id)`
- Collaborator operations: Remove ID field, use composite PK `(repo_id, user_id)`

```go
// Create - updated columns
func (s *ReposStore) Create(ctx context.Context, r *repos.Repository) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO repositories (id, owner_actor_id, name, slug, ...)
        VALUES ($1, $2, $3, ...)
    `, r.ID, r.OwnerActorID, r.Name, r.Slug, ...)
    if err != nil {
        return err
    }
    // Insert topics to repo_topics table
    for _, topic := range r.Topics {
        s.db.ExecContext(ctx, `
            INSERT INTO repo_topics (repo_id, topic) VALUES ($1, $2)
            ON CONFLICT DO NOTHING
        `, r.ID, topic)
    }
    return nil
}

// Star - no more ID field
func (s *ReposStore) Star(ctx context.Context, star *repos.Star) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO stars (user_id, repo_id, created_at)
        VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING
    `, star.UserID, star.RepoID, star.CreatedAt)
    return err
}

// AddCollaborator - no more ID field
func (s *ReposStore) AddCollaborator(ctx context.Context, collab *repos.Collaborator) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO collaborators (repo_id, user_id, permission, created_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (repo_id, user_id) DO UPDATE SET permission = $3
    `, collab.RepoID, collab.UserID, collab.Permission, collab.CreatedAt)
    return err
}
```

#### 3.3 `orgs_store.go` Updates

**Key Changes:**
- `org_members`: Remove ID field, use composite PK `(org_id, user_id)`
- AddMember should upsert (ON CONFLICT DO UPDATE for role changes)

```go
// AddMember - no ID field, upsert pattern
func (s *OrgsStore) AddMember(ctx context.Context, m *orgs.Member) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO org_members (org_id, user_id, role, created_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (org_id, user_id) DO UPDATE SET role = $3
    `, m.OrgID, m.UserID, m.Role, m.CreatedAt)
    return err
}

// GetMember - remove ID from scan
func (s *OrgsStore) GetMember(ctx context.Context, orgID, userID string) (*orgs.Member, error) {
    m := &orgs.Member{}
    err := s.db.QueryRowContext(ctx, `
        SELECT org_id, user_id, role, created_at
        FROM org_members WHERE org_id = $1 AND user_id = $2
    `, orgID, userID).Scan(&m.OrgID, &m.UserID, &m.Role, &m.CreatedAt)
    // ...
}
```

#### 3.4 `teams_store.go` Updates

**Key Changes:**
- `team_members`: Remove ID field, composite PK `(team_id, user_id)`
- `team_repos`: Remove ID field, composite PK `(team_id, repo_id)`

#### 3.5 `issues_store.go` Updates

**Key Changes:**
- `issue_labels`: Remove ID field, composite PK `(issue_id, label_id)`
- `issue_assignees`: Remove ID field, composite PK `(issue_id, user_id)`
- Remove single `assignee_id` from issues table (use `issue_assignees` exclusively)

```go
// AddLabel - no ID field
func (s *IssuesStore) AddLabel(ctx context.Context, issueLabel *issues.IssueLabel) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO issue_labels (issue_id, label_id, created_at)
        VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING
    `, issueLabel.IssueID, issueLabel.LabelID, issueLabel.CreatedAt)
    return err
}

// AddAssignee - no ID field
func (s *IssuesStore) AddAssignee(ctx context.Context, ia *issues.IssueAssignee) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO issue_assignees (issue_id, user_id, created_at)
        VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING
    `, ia.IssueID, ia.UserID, ia.CreatedAt)
    return err
}
```

#### 3.6 `pulls_store.go` Updates

**Key Changes:**
- Add `lock_reason`, `merge_method`, `merge_message` to insert/update
- `pr_labels`: Remove ID field, composite PK `(pr_id, label_id)`
- `pr_assignees`: Remove ID field, composite PK `(pr_id, user_id)`
- `pr_reviewers`: Remove ID field, composite PK `(pr_id, user_id)`

#### 3.7 `stars_store.go` Updates

**Key Changes:**
- Remove ID field from Star struct usage
- Use composite PK `(user_id, repo_id)`

#### 3.8 `watches_store.go` Updates

**Key Changes:**
- Remove ID field from Watch struct usage
- Use composite PK `(user_id, repo_id)`

#### 3.9 `collaborators_store.go` Updates

**Key Changes:**
- Remove ID field from Collaborator struct usage
- Use composite PK `(repo_id, user_id)`

#### 3.10 `users_store.go` Updates

**Key Changes:**
- Add SSH key and API token methods with new schema
- `ssh_keys`: Add `UNIQUE(user_id, fingerprint)` constraint handling
- `api_token_scopes`: New table for normalized scopes

```go
// CreateAPIToken - insert token then scopes
func (s *UsersStore) CreateAPIToken(ctx context.Context, token *users.APIToken) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO api_tokens (id, user_id, name, token_hash, expires_at, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, token.ID, token.UserID, token.Name, token.TokenHash, token.ExpiresAt, token.CreatedAt)
    if err != nil {
        return err
    }
    // Insert scopes
    for _, scope := range token.Scopes {
        s.db.ExecContext(ctx, `
            INSERT INTO api_token_scopes (token_id, scope)
            VALUES ($1, $2) ON CONFLICT DO NOTHING
        `, token.ID, scope)
    }
    return nil
}
```

### Phase 4: Feature Model Updates

#### 4.1 Update `feature/repos/api.go`

```go
// Star - remove ID field requirement
type Star struct {
    UserID    string    `json:"user_id"`
    RepoID    string    `json:"repo_id"`
    CreatedAt time.Time `json:"created_at"`
    // ID removed - composite PK
}

// Collaborator - remove ID field requirement
type Collaborator struct {
    RepoID     string     `json:"repo_id"`
    UserID     string     `json:"user_id"`
    Permission Permission `json:"permission"`
    CreatedAt  time.Time  `json:"created_at"`
    // ID removed - composite PK
}
```

#### 4.2 Update `feature/orgs/api.go`

```go
// Member - remove ID field requirement
type Member struct {
    OrgID     string    `json:"org_id"`
    UserID    string    `json:"user_id"`
    Role      string    `json:"role"`
    CreatedAt time.Time `json:"created_at"`
    // ID removed - composite PK
}
```

#### 4.3 Update `feature/issues/api.go`

```go
// IssueLabel - remove ID field requirement
type IssueLabel struct {
    IssueID   string    `json:"issue_id"`
    LabelID   string    `json:"label_id"`
    CreatedAt time.Time `json:"created_at"`
    // ID removed - composite PK
}

// IssueAssignee - remove ID field requirement
type IssueAssignee struct {
    IssueID   string    `json:"issue_id"`
    UserID    string    `json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
    // ID removed - composite PK
}
```

#### 4.4 Update `feature/pulls/api.go`

Add new fields to PullRequest struct:
```go
type PullRequest struct {
    // ... existing fields ...
    LockReason   string `json:"lock_reason,omitempty"`   // NEW
    MergeMethod  string `json:"merge_method,omitempty"`  // NEW
    MergeMessage string `json:"merge_message,omitempty"` // NEW
}

// PRLabel - remove ID field
type PRLabel struct {
    PRID      string    `json:"pr_id"`
    LabelID   string    `json:"label_id"`
    CreatedAt time.Time `json:"created_at"`
}

// PRAssignee - remove ID field
type PRAssignee struct {
    PRID      string    `json:"pr_id"`
    UserID    string    `json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
}

// PRReviewer - remove ID field
type PRReviewer struct {
    PRID      string    `json:"pr_id"`
    UserID    string    `json:"user_id"`
    State     string    `json:"state"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Phase 5: Test Updates

All tests need updating to:
1. Remove ID generation for link tables
2. Use upsert patterns for idempotent operations
3. Test new constraint behaviors (duplicates should conflict or be ignored)

Example test update:
```go
// Old
star := &repos.Star{
    ID:        ulid.New(),  // REMOVE
    UserID:    user.ID,
    RepoID:    repo.ID,
    CreatedAt: time.Now(),
}

// New
star := &repos.Star{
    UserID:    user.ID,
    RepoID:    repo.ID,
    CreatedAt: time.Now(),
}
```

### Phase 6: Migration Safety

The v2 schema uses `CREATE TABLE IF NOT EXISTS` so it's safe to run on existing databases. However:

1. **Data migration** is NOT automatic - existing data in old schema will not be migrated
2. For production: Create a migration script to:
   - Create `actors` entries for existing users/orgs
   - Update `repositories.owner_actor_id` from `owner_id/owner_type`
   - Migrate `api_tokens.scopes` CSV to `api_token_scopes` rows
   - Migrate `repositories.topics` CSV to `repo_topics` rows

## Files to Modify

### Store Layer
1. `store/duckdb/schema.sql` - Replace with v2 schema
2. `store/duckdb/actors_store.go` - NEW file
3. `store/duckdb/repos_store.go` - Major updates for actors and topics
4. `store/duckdb/orgs_store.go` - Update org_members
5. `store/duckdb/teams_store.go` - Update team_members, team_repos
6. `store/duckdb/issues_store.go` - Update issue_labels, issue_assignees
7. `store/duckdb/pulls_store.go` - Add new fields, update link tables
8. `store/duckdb/stars_store.go` - Remove ID from stars
9. `store/duckdb/watches_store.go` - Remove ID from watches
10. `store/duckdb/collaborators_store.go` - Remove ID from collaborators
11. `store/duckdb/users_store.go` - Add SSH keys, API tokens with scopes

### Feature Layer
1. `feature/repos/api.go` - Update Star, Collaborator structs
2. `feature/orgs/api.go` - Update Member struct
3. `feature/issues/api.go` - Update IssueLabel, IssueAssignee structs
4. `feature/pulls/api.go` - Add PullRequest fields, update link structs
5. `feature/teams/api.go` - Update TeamMember struct
6. `feature/users/api.go` - Update APIToken.Scopes to []string

### Test Files
1. `store/duckdb/repos_store_test.go`
2. `store/duckdb/orgs_store_test.go`
3. `store/duckdb/teams_store_test.go`
4. `store/duckdb/issues_store_test.go`
5. `store/duckdb/pulls_store_test.go`
6. `store/duckdb/stars_store_test.go`
7. `store/duckdb/watches_store_test.go`
8. `store/duckdb/collaborators_store_test.go`
9. `store/duckdb/users_store_test.go`

## Success Criteria

1. All existing tests pass with updated schema
2. New tests for:
   - Actor creation/lookup
   - Composite PK idempotency (duplicate inserts don't error)
   - New PR fields
   - API token scopes normalization
   - Repo topics normalization
3. No regressions in existing functionality
