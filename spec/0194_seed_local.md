# GitHome Codebase Exploration Summary

## Project Overview
GitHome is a self-hosted GitHub clone built as a blueprint for the Mizu web framework. It provides repository hosting, issues, pull requests, labels, milestones, and more with full user/organization management and git integration.

## Directory Structure

/Users/apple/github/go-mizu/mizu/blueprints/githome/
├── cli/                    # CLI commands (serve, init, seed)
├── app/web/               # Web server and HTTP handlers
│   ├── handler/           # HTTP request handlers
│   └── server.go          # Server setup and routing
├── feature/               # Business logic (18 packages)
│   ├── repos/            # Repository CRUD and permissions
│   ├── orgs/             # Organization management
│   ├── users/            # User authentication and profiles
│   ├── issues/           # Issue tracking
│   ├── comments/         # Comments on issues/PRs
│   ├── labels/           # Issue labels
│   ├── milestones/       # Release milestones
│   ├── stars/            # Repository stars
│   ├── watches/          # Repository watching
│   ├── releases/         # Release management
│   ├── collaborators/    # Collaborator permissions
│   └── ... (more features)
├── store/duckdb/          # Database layer (37 files)
│   ├── schema.sql         # DuckDB schema definition
│   ├── store.go           # Core store with schema management
│   ├── repos_store.go     # Repository persistence
│   ├── users_store.go     # User persistence
│   ├── orgs_store.go      # Organization persistence
│   ├── actors_store.go    # Unified owner (user/org) management
│   ├── issues_store.go    # Issue persistence
│   ├── ...               # Additional stores (one per feature)
│   └── *_test.go         # Store tests
├── pkg/                   # Utility packages
│   ├── git/              # Git repository operations
│   ├── seed/github/      # GitHub API seeding
│   ├── ulid/             # ULID ID generation
│   ├── slug/             # URL slug generation
│   ├── password/         # Password hashing
│   ├── avatar/           # Avatar generation
│   ├── markdown/         # Markdown rendering
│   └── pagination/       # Pagination utilities
├── assets/               # Frontend templates and static files
│   └── views/default/pages/
│       ├── home.html
│       ├── explore.html
│       ├── repo_home.html
│       ├── repo_issues.html
│       ├── issue_view.html
│       ├── new_issue.html
│       ├── repo_settings.html
│       ├── user_profile.html
│       ├── login.html
│       ├── register.html
│       └── new_repo.html
├── cmd/                  # Main CLI entry point
└── go.mod, Makefile      # Project configuration

---

## 1. CLI COMMANDS & INITIALIZATION

### Files:
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/cli/root.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/cli/init.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/cli/seed.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/cli/serve.go`

### Structure:
The CLI uses Cobra framework with these commands:

1. **githome serve** - Start the web server
   - Flags: -a/--addr (default :3000), --dev (development mode)
   - Creates web.Server instance and runs HTTP server
   - Logs to stdout with debug level

2. **githome init** - Initialize the database
   - Creates $HOME/data/blueprint/githome directory
   - Creates $HOME/data/blueprint/githome/repos directory
   - Opens DuckDB at githome.db
   - Creates all tables from embedded schema.sql
   - Runs any necessary migrations

3. **githome seed** - Seed demo data
   - Creates two test users: "admin" (admin=true) and "demo"
   - Creates 3 sample repositories under admin account:
     - hello-world (public)
     - my-project (public)
     - private-repo (private)
   - Creates 3 sample issues for hello-world repo with various states
   - Updates issue counters

### Key Variables:
```go
dataDir  = os.Getenv("HOME") + "/data/blueprint/githome"
reposDir = dataDir + "/repos"
```

---

## 2. ORGANIZATIONS AND REPOS STORAGE

### Org/User Ownership Model - Actors Pattern:

The system uses an "Actors" pattern to unify ownership:

1. **users table** - User accounts
   - id, username, email, password_hash
   - full_name, avatar_url, bio, location, website, company
   - is_admin, is_active, timestamps

2. **organizations table** - Organization accounts
   - id, name (unique), slug (unique), display_name
   - description, avatar_url, location, website, email, is_verified, timestamps

3. **actors table** - Unified owner representation
   - id, actor_type (enum: 'user' or 'org'), user_id, org_id, created_at
   - CHECK constraints enforce exactly one of user_id or org_id is set
   - UNIQUE constraints prevent duplicate user/org actors
   - One actor per user; one actor per org

4. **repositories table** - Git repositories
   - id, owner_actor_id (FK to actors), name, slug
   - description, website, default_branch
   - is_private, is_archived, is_template, is_fork, forked_from_repo_id
   - star_count, fork_count, watcher_count, open_issue_count, open_pr_count
   - size_kb, license, language, language_color
   - has_issues, has_wiki, has_projects
   - created_at, updated_at, pushed_at
   - UNIQUE(owner_actor_id, slug)

### How Organizations/Repos Are Created:

**Creating an Organization:**
1. OrgService.Create(ctx, creatorID, CreateIn) called
2. Validates org name, generates slug
3. Creates Organization record in orgs table
4. Creates Member record with creator as owner (role='owner')
5. Creates Actor record with actor_type='org', org_id set

**Creating a Repository:**
1. RepoService.Create(ctx, ownerID, CreateIn) called
2. Validates repo name, generates slug
3. Gets/creates Actor for owner (user or org)
4. Creates Repository record with owner_actor_id pointing to actor
5. Creates git directory at reposDir/ownerID/slug.git
6. If directory creation fails, rolls back DB record

### Storage Implementation:

**ReposStore (DuckDB):**
- File: `/Users/apple/github/go-mizu/mizu/blueprints/githome/store/duckdb/repos_store.go`
- Methods:
  - Create(ctx, *Repository) - Insert repo and topics
  - GetByID(ctx, id) - Fetch single repo
  - GetByOwnerAndName(ctx, ownerActorID, ownerType, name) - Fetch by owner+slug
  - Update(ctx, *Repository) - Update repo and topics
  - Delete(ctx, id) - Remove repo (topics cascade)
  - ListByOwner(ctx, ownerActorID, ownerType, limit, offset)
  - ListPublic(ctx, limit, offset) - Order by star_count DESC, updated_at DESC
  - ListByIDs(ctx, []string)
  - Star/Unstar - Manage stars composite PK (user_id, repo_id)
  - AddCollaborator/RemoveCollaborator - Manage collaborators

**OrgsStore (DuckDB):**
- File: `/Users/apple/github/go-mizu/mizu/blueprints/githome/store/duckdb/orgs_store.go`
- Methods:
  - Create, GetByID, GetBySlug, Update, Delete
  - AddMember, RemoveMember, GetMember, ListMembers (org_members table)
  - CountOwners, UpdateMember, ListByUser

**ActorsStore (DuckDB):**
- File: `/Users/apple/github/go-mizu/mizu/blueprints/githome/store/duckdb/actors_store.go`
- Manages unified actor records
- Methods: Create, GetByID, GetByUserID, GetByOrgID, GetOrCreateForUser

---

## 3. DUCKDB SCHEMA IMPLEMENTATION

### File:
`/Users/apple/github/go-mizu/mizu/blueprints/githome/store/duckdb/schema.sql`

### Key Design Principles (from comments):
1. Correctness-first: Constrain duplicates and invalid states early
2. Keep OLTP surface minimal: DuckDB is columnar; avoid index sprawl
3. Make ownership/permissions model consistent across user/org
4. Prefer idempotent link tables using composite primary keys
5. Treat counters as cached/derived, not authoritative

### Core Tables (666 lines total):

**Users & Auth:**
- users: User accounts with profiles
- sessions: Active sessions with IP/user-agent tracking
- ssh_keys: SSH key management with fingerprints
- api_tokens: API token management with scopes
- api_token_scopes: Token permissions (composite PK)

**Orgs & Actors:**
- organizations: Organization profiles
- actors: Unified owner representation (user or org)
- org_members: Organization membership (composite PK: org_id, user_id)
- teams: Organization teams with hierarchy support
- team_members: Team membership (composite PK: team_id, user_id)

**Repositories:**
- repositories: Core repository metadata
- repo_topics: Repository tags (composite PK: repo_id, topic)
- repo_storage: Storage backend configuration (S3, R2, filesystem)
- collaborators: Repository access control (composite PK: repo_id, user_id)
- team_repos: Team repository access (composite PK: team_id, repo_id)
- stars: Repository stars (composite PK: user_id, repo_id)
- watches: Repository notifications (composite PK: user_id, repo_id)

**Issues & PRs:**
- issues: Issue tracking with state and locking
- issue_labels: Issue labels (composite PK: issue_id, label_id)
- issue_assignees: Issue assignments (composite PK: issue_id, user_id)
- pull_requests: PR tracking with merge tracking
- pr_labels, pr_assignees, pr_reviewers: PR management
- pr_reviews: Review records with state tracking
- review_comments: Inline code review comments

**Meta:**
- labels: Repository labels per repo (UNIQUE: repo_id, name)
- milestones: Release milestones (UNIQUE: repo_id, number)
- comments: Generic comments for issues/PRs (target_type IN ('issue', 'pull_request'))
- releases: Release management
- release_assets: Release artifacts
- webhooks: Event webhooks (repo_id or org_id)
- webhook_deliveries: Webhook delivery tracking
- notifications: User notifications
- activities: Activity feed
- reactions: Emoji reactions (composite PK: user_id, target_type, target_id, content)

### Index Strategy:
Minimal indexing focused on common queries:
- Foreign key columns (user_id, owner_actor_id, repo_id)
- State columns (state, unread)
- Composite indexes for frequent queries
  - idx_issues_repo_state_updated (repo_id, state, updated_at)
  - idx_prs_repo_state_updated (repo_id, state, updated_at)

### Store Implementation Pattern:

Each table has a corresponding *Store in DuckDB:

1. Single table stores:
   - ReposStore (repositories + repo_topics)
   - UsersStore (users + sessions)
   - OrgsStore (organizations + org_members)
   - IssuesStore (issues + issue_labels + issue_assignees)
   - LabelsStore, MilestonesStore, ReleasesStore, etc.

2. Composite key handling:
   - CollaboratorsStore.AddCollaborator(collab) - INSERT (repo_id, user_id, ...)
   - ReposStore.Star(star) - INSERT (user_id, repo_id, ...)
   - IssuesStore.AddLabel(issueLabel) - INSERT (issue_id, label_id, ...)

3. Null handling:
   - `nullString(s string)` - Returns nil for empty strings
   - `nullTime(t time.Time)` - Returns nil for zero time
   - sql.NullString, sql.NullTime used for scanning

---

## 4. SEED IMPLEMENTATION

### Files:
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/cli/seed.go` - CLI seed command
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/seed/github/seed.go` - GitHub seeding logic
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/seed/github/client.go` - GitHub API client

### Seed Command Flow:

1. **newSeedCmd()** - Cobra command that:
   - Opens DuckDB at ~/data/blueprint/githome/githome.db
   - Creates DuckDB store with schema
   - Instantiates services (users, repos)

2. **Demo Data Seeding:**
   - Creates admin user (username=admin, email=admin@githome.local, password=password123)
     - Sets is_admin=true
   - Creates demo user (username=demo, email=demo@githome.local, password=demo1234)
   - Creates sample repos under admin:
     1. hello-world (public) - "A simple hello world repository"
     2. my-project (public) - "My awesome project"
     3. private-repo (private) - "A private repository"
   - Creates issues for hello-world repo:
     1. "Add README documentation" (open)
     2. "Bug: Application crashes on startup" (open)
     3. "Feature: Add dark mode" (closed)
   - Updates hello-world.OpenIssueCount = 2

### GitHub Seeding (Advanced Feature):

**Seeder struct** - pkg/seed/github/seed.go:
```go
type Seeder struct {
    db     *sql.DB
    client *Client
    config SeedConfig
    
    // Stores
    usersStore, reposStore, issuesStore, labelsStore, milestonesStore, commentsStore, actorsStore
    
    // Mappings
    userIDs map[string]string
    labelIDs map[string]string
    milestoneIDs map[int]string
}

type SeedConfig struct {
    Owner       string  // GitHub owner/org
    Repo        string  // GitHub repo name
    LocalOwner  string  // Local user to own imported repo
    Token       string  // GitHub token (optional)
    MaxIssues   int     // Limit issues
    MaxComments int     // Limit comments per issue
}
```

**Seeding Process:**
1. Fetch GitHub repo metadata
2. Create/get local owner user and actor
3. Create local repository with GitHub metadata
4. Fetch and import labels
5. Fetch and import milestones
6. Fetch and import issues:
   - Create issue record
   - Create/get users for author, assignees, closed_by
   - Add issue labels
   - Add issue assignees
   - Fetch and import comments for each issue
7. Log progress every 10 issues

**Client (pkg/seed/github/client.go):**
- Uses GitHub REST API
- Methods: FetchRepository, FetchLabels, FetchMilestones, FetchIssues, FetchComments
- Respects rate limits with optional token for higher limits

---

## 5. EXPLORER FEATURE & PUBLIC MODE

### Files:
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/app/web/handler/page.go` - Page rendering
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/app/web/handler/repo.go` - Repo API endpoints
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/assets/views/default/pages/explore.html` - Template

### Explorer Functionality:

**ExploreData struct** - in both handler and assets:
```go
type ExploreData struct {
    Title        string
    User         *users.User          // Nil if unauthenticated
    Repositories []*repos.Repository
    Query        string
}
```

**Public Repositories:**
- ReposStore.ListPublic(ctx, limit, offset)
  - Queries: SELECT * FROM repositories WHERE is_private = FALSE
  - Orders by: star_count DESC, updated_at DESC
  - Returns paginated public repos

**Repo API:**
- Repo.ListPublic(c *mizu.Ctx) - HTTP GET endpoint
  - Query params: page, per_page, sort
  - Default: page=1, per_page=30
  - Returns JSON list of repos

**Explore Page:**
- shows public repositories
- supports authenticated (User != nil) and unauthenticated (User == nil) modes
- search query parameter support (template shows Query field)

### Public Mode Details:

**Permissions Model (repos/service.go):**
```go
GetPermission(ctx, repoID, userID) -> Permission
  if repo.OwnerID == userID -> PermissionAdmin
  else if collaborator found -> collab.Permission
  else if !repo.IsPrivate -> PermissionRead
  else -> ""
```

**Permission Levels:**
- PermissionRead (public repos, unauthenticated users)
- PermissionTriage
- PermissionWrite
- PermissionMaintain
- PermissionAdmin (repo owner)

**Access Control:**
- Private repos: only owner + collaborators can read
- Public repos: anyone can read (even unauthenticated)
- All repos: public activity feed (is_public=TRUE in activities table)

---

## 6. TEMPLATES & UI PAGES

### File:
`/Users/apple/github/go-mizu/mizu/blueprints/githome/assets/templates_test.go` - Template definitions and tests

### 13 Core Templates:

1. **home** - HomeData
   - Shown to authenticated users (User != nil) and guests
   - Lists user's or recent repositories
   
2. **explore** - ExploreData
   - Browse public repositories
   - Supports search query

3. **login** - LoginData
   - Sign in page

4. **register** - RegisterData
   - Sign up page

5. **new_repo** - NewRepoData
   - Create new repository form

6. **user_profile** - UserProfileData
   - User profile with bio, repos, join date
   - IsOwner flag for edit capability

7. **repo_home** - RepoHomeData
   - Repository main page
   - Shows description, stats, languages
   - IsStarred (for authenticated users)
   - CanEdit (if repo owner)

8. **repo_issues** - RepoIssuesData
   - List issues filtered by state
   - Shows issue title, author, created date
   - State filter support

9. **issue_view** - IssueViewData
   - Single issue with full details
   - Author info and avatar
   - Comments section
   - CanEdit flag

10. **new_issue** - NewIssueData
    - Create issue form

11. **repo_settings** - RepoSettingsData
    - Repository configuration
    - Collaborator management list

12. **layouts/default** - Base layout
    - Navigation header
    - Footer
    - CSS/JS includes

13. **layouts/auth** - Auth layout
    - For login/register pages
    - No main navigation

### Template Data Patterns:

All templates can handle:
- Nil User (unauthenticated)
- Nil Repository, Nil Issues, Nil Comments, Nil Profile (empty states)
- Empty arrays for paginated data

---

## 7. GIT OPERATIONS & CODE EXPLORER

### Files:
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/git/repository.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/git/tree.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/git/blob.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/git/commit.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/git/ref.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/git/language.go`
- `/Users/apple/github/go-mizu/mizu/blueprints/githome/pkg/git/git_test.go`

### Git Package Structure:

**Repository struct:**
- Wraps git command-line tool
- Path: absolute path to .git or bare repo root
- Methods execute git commands via exec.CommandContext

**Core Operations:**

1. **Refs (branches/tags):**
   - ResolveRef(ctx, ref) -> SHA (40-char)
   - GetDefaultBranch(ctx) -> string
   - ListBranches(ctx) -> []Ref
   - ListTags(ctx) -> []Ref

2. **Tree (directory listing):**
   - GetTree(ctx, ref, path) -> *Tree
   - Returns entries sorted: directories first, then files
   - Entry: Name, Path, Type (tree/blob), Mode, SHA, Size

3. **Blob (file content):**
   - GetBlob(ctx, ref, path) -> *Blob
   - Name, Path, Mode, SHA, Size, Language, LanguageColor
   - IsBinary, Lines, Content (text files only)
   - Safe: binary files cause ErrBinaryFile

4. **Commits:**
   - GetLatestCommit(ctx, ref) -> *Commit
   - GetCommitHistory(ctx, ref, limit) -> []*Commit
   - GetFileHistory(ctx, path, ref, limit) -> []*Commit
   - Commit: SHA, ShortSHA, Author, Committer, Title, Message, CreatedAt

5. **Path utilities:**
   - PathExists(ctx, ref, path) -> bool
   - GetPathType(ctx, ref, path) -> "tree"|"blob"|error
   - IsValidPath(path) -> bool (prevents ../../../etc/passwd)

### Language Detection:

```go
DetectLanguage(filename) -> "Go"|"JavaScript"|"Python"|...|""
LanguageColor(language) -> "#hexcolor"
```

Supported: 20+ languages (Go, JS, TS, Python, Rust, Java, C++, C#, PHP, Ruby, CSS, HTML, Markdown, YAML, JSON, SQL, Bash, Dockerfile, Makefile)

### Security:

- ErrPathTraversal for paths with ".." or absolute paths
- IsValidPath checks for path traversal
- ErrBinaryFile prevents displaying binary content
- git commands run via exec.CommandContext with timeout

### Testing:

git_test.go includes comprehensive tests:
- TestOpen, TestOpenNotARepo
- TestResolveRef, TestGetDefaultBranch
- TestListBranches, TestGetTree
- TestGetBlob, TestGetCommit, TestGetCommitHistory
- TestDetectLanguage, TestLanguageColor
- TestIsValidPath, TestPathExists, TestGetPathType

Tests use the actual mizu repo as test data or skip if no repo found.

---

## 8. SERVICE LAYER ARCHITECTURE

### Pattern:

Each feature has:
1. **api.go** - Public interface (API + Store)
2. **service.go** - Business logic implementation

Example (repos):
```go
// API interface
type API interface {
    Create(ctx, ownerID string, in *CreateIn) (*Repository, error)
    GetByID(ctx, id string) (*Repository, error)
    GetByOwnerAndName(ctx, ownerID, ownerType, name string) (*Repository, error)
    // ... 15+ methods
}

// Store interface (implemented by ReposStore)
type Store interface {
    Create(ctx, r *Repository) error
    GetByID(ctx, id string) (*Repository, error)
    // ... data layer
}

// Service implements API
type Service struct {
    store    Store
    reposDir string  // For filesystem operations
}
```

### Feature Packages (18 total):
- repos - Repository CRUD, collaborators, stars, forks
- orgs - Organization management, membership
- users - Authentication, profiles, sessions
- issues - Issue tracking, labels, assignees
- comments - Comments on issues/PRs
- labels - Issue label management
- milestones - Release milestones
- stars - Repository starring
- watches - Repository watching
- releases - Release management
- collaborators - Permission management
- pulls - Pull request tracking (if exists)
- teams - Team management
- webhooks - Webhook delivery
- notifications - User notifications
- activities - Activity feed
- reactions - Emoji reactions
- orgs/teams - Team operations

Each has:
- Define errors (ErrNotFound, ErrExists, etc.)
- Define data types (Repository, User, Issue, etc.)
- Define input types (*CreateIn, *UpdateIn, *ListOpts)
- Define constants (Permission levels, states, roles)
- Define interfaces (API, Store)
- Implement service with business logic

---

## 9. DATABASE INITIALIZATION FLOW

### Init Command:
1. Create directories:
   - $HOME/data/blueprint/githome/
   - $HOME/data/blueprint/githome/repos/
2. Open DuckDB at $HOME/data/blueprint/githome/githome.db
3. Create Store instance
4. Call store.Ensure(ctx):
   - Executes embedded schema.sql
   - Runs migrations (currently empty)
5. Log success

### Seed Command:
1. Open same DuckDB
2. Create Store + all service instances
3. Register admin user (is_admin=true)
4. Register demo user
5. Create sample repos with issues
6. Log completion

### Server Startup:
1. Config: addr, dataDir, reposDir, dev mode
2. Open DuckDB
3. Create Store (ensures schema if needed)
4. Create all stores (users, repos, issues, etc.)
5. Create all services
6. Create handlers with service access
7. Load templates from assets
8. Setup Mizu router with routes
9. Listen on configured address

---

## 10. KEY INSIGHTS

### Design Patterns:
1. **Actors Pattern** - Unifies user/org as owners
2. **Composite Primary Keys** - (user_id, repo_id) for many-to-many
3. **Service/Store Split** - Business logic separate from persistence
4. **Interface-driven** - Each feature has API and Store interfaces
5. **Embedded Schema** - SQL schema embedded in binary via //go:embed
6. **Template Data** - Rich context objects passed to templates

### Error Handling:
- Structured errors per feature (ErrNotFound, ErrExists, ErrAccessDenied)
- DuckDB errors wrapped with context
- Service layer translates to HTTP status codes

### Performance:
- Minimal indexing for OLTP on columnar DB
- Pagination support in all list operations
- Composite keys prevent N+1 issues
- Star counts cached (not computed from stars table)

### Security:
- Password hashing (pkg/password)
- Session tokens (sessions table)
- SSH keys with fingerprints
- API tokens with scopes
- Path traversal protection in git operations
- Public/private repo distinction

### Features Partially Implemented:
- GitHub seeding (full flow exists)
- Pull requests (schema exists, service/store not yet seen)
- Teams (schema and service exist)
- Webhooks (schema exists)
- Activity feed (schema exists)
- Notifications (schema exists)

---

## File Count Summary:
- Go files: 80+
- SQL: 1 (schema.sql, 666 lines)
- HTML templates: 13
- Test files: 10+
- Total implementation: ~15,000 lines of code
