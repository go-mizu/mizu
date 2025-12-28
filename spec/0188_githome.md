# GitHome - Full-Featured GitHub Clone

## Overview

GitHome is a self-hosted GitHub clone built on the Mizu framework. It provides complete Git repository hosting with web-based collaboration features including issues, pull requests, code review, and team management.

## Goals

1. **Feature Parity**: Implement core GitHub functionality for self-hosted use
2. **Simplicity**: Single binary, embedded database (DuckDB), minimal dependencies
3. **Performance**: Fast Git operations, efficient database queries
4. **Modern UI**: Clean, responsive interface with real-time updates

---

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         GitHome Server                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Web UI    │  │  REST API   │  │  Git Smart HTTP/SSH    │  │
│  │  (HTML/JS)  │  │  /api/v1/*  │  │  (git clone/push/pull) │  │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │
│         │                │                      │                │
│  ┌──────┴────────────────┴──────────────────────┴─────────────┐ │
│  │                    Mizu Router Layer                        │ │
│  │              (HTTP handlers, middleware, auth)              │ │
│  └─────────────────────────┬───────────────────────────────────┘ │
│                            │                                     │
│  ┌─────────────────────────┴───────────────────────────────────┐ │
│  │                     Feature Services                         │ │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐ │ │
│  │  │  Users  │ │  Repos  │ │ Issues  │ │  Pulls  │ │ Teams  │ │ │
│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └───┬────┘ │ │
│  └───────┼──────────┼──────────┼──────────┼─────────────┼──────┘ │
│          │          │          │          │             │        │
│  ┌───────┴──────────┴──────────┴──────────┴─────────────┴──────┐ │
│  │                      Data Layer                              │ │
│  │  ┌─────────────────────┐  ┌───────────────────────────────┐ │ │
│  │  │    DuckDB Store     │  │       Git Filesystem          │ │ │
│  │  │  (metadata, users,  │  │   (bare repositories,         │ │ │
│  │  │   issues, PRs...)   │  │    commits, branches)         │ │ │
│  │  └─────────────────────┘  └───────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
githome/
├── cmd/
│   └── githome/
│       └── main.go                 # CLI entry point
├── cli/
│   ├── root.go                     # Root command
│   ├── serve.go                    # HTTP server command
│   ├── init.go                     # Initialize database
│   └── seed.go                     # Seed demo data
├── app/
│   └── web/
│       ├── server.go               # Server setup & routes
│       ├── handler/
│       │   ├── response.go         # Standard response helpers
│       │   ├── auth.go             # Auth endpoints
│       │   ├── user.go             # User endpoints
│       │   ├── repo.go             # Repository endpoints
│       │   ├── issue.go            # Issue endpoints
│       │   ├── pull.go             # Pull request endpoints
│       │   ├── commit.go           # Commit/branch endpoints
│       │   ├── file.go             # File browser endpoints
│       │   ├── label.go            # Label management
│       │   ├── milestone.go        # Milestone management
│       │   ├── org.go              # Organization endpoints
│       │   ├── team.go             # Team endpoints
│       │   ├── webhook.go          # Webhook endpoints
│       │   ├── notification.go     # Notification endpoints
│       │   └── page.go             # HTML page handlers
│       └── ws/
│           └── hub.go              # WebSocket hub for real-time
├── feature/
│   ├── users/
│   │   ├── api.go                  # User models & interfaces
│   │   └── service.go              # User business logic
│   ├── repos/
│   │   ├── api.go                  # Repository models
│   │   └── service.go              # Repository logic
│   ├── issues/
│   │   ├── api.go                  # Issue models
│   │   └── service.go              # Issue logic
│   ├── pulls/
│   │   ├── api.go                  # Pull request models
│   │   └── service.go              # PR logic
│   ├── comments/
│   │   ├── api.go                  # Comment models
│   │   └── service.go              # Comment logic
│   ├── labels/
│   │   ├── api.go                  # Label models
│   │   └── service.go              # Label logic
│   ├── milestones/
│   │   ├── api.go                  # Milestone models
│   │   └── service.go              # Milestone logic
│   ├── orgs/
│   │   ├── api.go                  # Organization models
│   │   └── service.go              # Organization logic
│   ├── teams/
│   │   ├── api.go                  # Team models
│   │   └── service.go              # Team logic
│   ├── collaborators/
│   │   ├── api.go                  # Collaborator models
│   │   └── service.go              # Collaborator logic
│   ├── notifications/
│   │   ├── api.go                  # Notification models
│   │   └── service.go              # Notification logic
│   ├── webhooks/
│   │   ├── api.go                  # Webhook models
│   │   └── service.go              # Webhook logic
│   ├── stars/
│   │   ├── api.go                  # Star models
│   │   └── service.go              # Star logic
│   ├── watches/
│   │   ├── api.go                  # Watch models
│   │   └── service.go              # Watch logic
│   ├── forks/
│   │   ├── api.go                  # Fork models
│   │   └── service.go              # Fork logic
│   ├── releases/
│   │   ├── api.go                  # Release models
│   │   └── service.go              # Release logic
│   └── activities/
│       ├── api.go                  # Activity feed models
│       └── service.go              # Activity logic
├── git/
│   ├── repository.go               # Git repository operations
│   ├── commit.go                   # Commit operations
│   ├── branch.go                   # Branch operations
│   ├── tree.go                     # Tree/file operations
│   ├── diff.go                     # Diff generation
│   ├── blame.go                    # Blame information
│   └── hooks.go                    # Git hooks management
├── store/
│   └── duckdb/
│       ├── store.go                # Core store with schema
│       ├── schema.sql              # Database schema
│       ├── users_store.go          # Users data access
│       ├── repos_store.go          # Repos data access
│       ├── issues_store.go         # Issues data access
│       ├── pulls_store.go          # Pulls data access
│       ├── comments_store.go       # Comments data access
│       ├── labels_store.go         # Labels data access
│       ├── milestones_store.go     # Milestones data access
│       ├── orgs_store.go           # Organizations data access
│       ├── teams_store.go          # Teams data access
│       ├── collaborators_store.go  # Collaborators data access
│       ├── notifications_store.go  # Notifications data access
│       ├── webhooks_store.go       # Webhooks data access
│       ├── stars_store.go          # Stars data access
│       ├── watches_store.go        # Watches data access
│       ├── releases_store.go       # Releases data access
│       └── activities_store.go     # Activities data access
├── assets/
│   ├── static/
│   │   ├── css/
│   │   │   ├── main.css            # Main styles
│   │   │   ├── syntax.css          # Syntax highlighting
│   │   │   └── markdown.css        # Markdown rendering
│   │   ├── js/
│   │   │   ├── main.js             # Main JavaScript
│   │   │   ├── editor.js           # Code editor
│   │   │   ├── diff.js             # Diff viewer
│   │   │   └── websocket.js        # Real-time updates
│   │   └── img/
│   │       └── logo.svg            # GitHome logo
│   ├── views/
│   │   ├── layouts/
│   │   │   ├── base.html           # Base layout
│   │   │   ├── app.html            # Authenticated layout
│   │   │   └── repo.html           # Repository layout
│   │   ├── auth/
│   │   │   ├── login.html          # Login page
│   │   │   ├── register.html       # Registration page
│   │   │   └── forgot.html         # Password reset
│   │   ├── user/
│   │   │   ├── profile.html        # User profile
│   │   │   ├── settings.html       # User settings
│   │   │   ├── repos.html          # User repositories
│   │   │   └── stars.html          # Starred repos
│   │   ├── repo/
│   │   │   ├── home.html           # Repository home (README)
│   │   │   ├── tree.html           # File browser
│   │   │   ├── blob.html           # File viewer
│   │   │   ├── commits.html        # Commit history
│   │   │   ├── commit.html         # Single commit view
│   │   │   ├── branches.html       # Branch list
│   │   │   ├── tags.html           # Tag list
│   │   │   ├── releases.html       # Release list
│   │   │   ├── settings.html       # Repository settings
│   │   │   └── new.html            # Create repository
│   │   ├── issues/
│   │   │   ├── list.html           # Issue list
│   │   │   ├── view.html           # Single issue
│   │   │   ├── new.html            # Create issue
│   │   │   └── edit.html           # Edit issue
│   │   ├── pulls/
│   │   │   ├── list.html           # PR list
│   │   │   ├── view.html           # Single PR
│   │   │   ├── new.html            # Create PR
│   │   │   ├── files.html          # PR file changes
│   │   │   └── commits.html        # PR commits
│   │   ├── org/
│   │   │   ├── home.html           # Organization home
│   │   │   ├── teams.html          # Team list
│   │   │   ├── members.html        # Member list
│   │   │   └── settings.html       # Org settings
│   │   ├── explore/
│   │   │   ├── repos.html          # Explore repositories
│   │   │   └── users.html          # Explore users
│   │   └── components/
│   │       ├── header.html         # Header partial
│   │       ├── footer.html         # Footer partial
│   │       ├── sidebar.html        # Sidebar partial
│   │       ├── pagination.html     # Pagination partial
│   │       ├── issue_item.html     # Issue list item
│   │       ├── pr_item.html        # PR list item
│   │       ├── commit_item.html    # Commit list item
│   │       ├── file_tree.html      # File tree partial
│   │       └── diff.html           # Diff partial
│   └── emails/
│       ├── welcome.html            # Welcome email
│       ├── reset.html              # Password reset
│       ├── mention.html            # Mention notification
│       └── pr_merged.html          # PR merged notification
├── pkg/
│   ├── ulid/
│   │   └── ulid.go                 # ID generation
│   ├── password/
│   │   └── password.go             # Password hashing
│   ├── markdown/
│   │   └── markdown.go             # Markdown rendering
│   ├── highlight/
│   │   └── highlight.go            # Syntax highlighting
│   ├── avatar/
│   │   └── avatar.go               # Avatar generation
│   ├── slug/
│   │   └── slug.go                 # URL slug generation
│   └── pagination/
│       └── pagination.go           # Pagination helpers
├── go.mod
├── go.sum
└── Makefile
```

---

## Database Schema

### Entity-Relationship Diagram

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│     users       │       │  organizations  │       │     teams       │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id              │───┐   │ id              │───┐   │ id              │
│ username        │   │   │ name            │   │   │ org_id          │──┐
│ email           │   │   │ slug            │   │   │ name            │  │
│ password_hash   │   │   │ description     │   │   │ slug            │  │
│ full_name       │   │   │ avatar_url      │   │   │ description     │  │
│ avatar_url      │   │   │ location        │   │   │ permission      │  │
│ bio             │   │   │ website         │   │   │ created_at      │  │
│ location        │   │   │ created_at      │   │   └────────┬────────┘  │
│ website         │   │   └────────┬────────┘              │           │
│ is_admin        │   │            │                       │           │
│ created_at      │   │   ┌────────┴────────┐     ┌────────┴───────┐   │
│ updated_at      │   │   │   org_members   │     │  team_members  │   │
└────────┬────────┘   │   ├─────────────────┤     ├────────────────┤   │
         │            │   │ org_id          │     │ team_id        │───┘
         │            │   │ user_id         │─────│ user_id        │
         │            └───│ role            │     │ created_at     │
         │                │ created_at      │     └────────────────┘
         │                └─────────────────┘
         │
┌────────┴────────┐       ┌─────────────────┐       ┌─────────────────┐
│  repositories   │       │  collaborators  │       │     stars       │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id              │───┬───│ repo_id         │       │ user_id         │
│ owner_id        │   │   │ user_id         │───────│ repo_id         │
│ owner_type      │   │   │ permission      │       │ created_at      │
│ name            │   │   │ created_at      │       └─────────────────┘
│ slug            │   │   └─────────────────┘
│ description     │   │                             ┌─────────────────┐
│ default_branch  │   │   ┌─────────────────┐       │    watches      │
│ is_private      │   │   │     issues      │       ├─────────────────┤
│ is_fork         │   │   ├─────────────────┤       │ user_id         │
│ forked_from_id  │   ├───│ id              │       │ repo_id         │
│ star_count      │   │   │ repo_id         │       │ level           │
│ fork_count      │   │   │ number          │       │ created_at      │
│ issue_count     │   │   │ title           │       └─────────────────┘
│ pr_count        │   │   │ body            │
│ created_at      │   │   │ author_id       │───────┐
│ updated_at      │   │   │ assignee_id     │       │
│ pushed_at       │   │   │ state           │       │
└────────┬────────┘   │   │ is_locked       │       │
         │            │   │ milestone_id    │───┐   │
         │            │   │ created_at      │   │   │
         │            │   │ updated_at      │   │   │
         │            │   │ closed_at       │   │   │
         │            │   └────────┬────────┘   │   │
         │            │            │            │   │
         │            │   ┌────────┴────────┐   │   │
         │            │   │  issue_labels   │   │   │
         │            │   ├─────────────────┤   │   │
         │            │   │ issue_id        │   │   │
         │            │   │ label_id        │───┼───┼───┐
         │            │   └─────────────────┘   │   │   │
         │            │                         │   │   │
         │            │   ┌─────────────────┐   │   │   │
         │            │   │   milestones    │───┘   │   │
         │            │   ├─────────────────┤       │   │
         │            └───│ id              │       │   │
         │                │ repo_id         │       │   │
         │                │ number          │       │   │
         │                │ title           │       │   │
         │                │ description     │       │   │
         │                │ state           │       │   │
         │                │ due_date        │       │   │
         │                │ created_at      │       │   │
         │                │ closed_at       │       │   │
         │                └─────────────────┘       │   │
         │                                          │   │
         │            ┌─────────────────┐           │   │
         │            │     labels      │───────────┼───┘
         │            ├─────────────────┤           │
         └────────────│ id              │           │
                      │ repo_id         │           │
                      │ name            │           │
                      │ color           │           │
                      │ description     │           │
                      │ created_at      │           │
                      └─────────────────┘           │
                                                    │
┌─────────────────┐       ┌─────────────────┐       │
│  pull_requests  │       │   pr_reviews    │       │
├─────────────────┤       ├─────────────────┤       │
│ id              │───┬───│ id              │       │
│ repo_id         │   │   │ pr_id           │       │
│ number          │   │   │ user_id         │───────┤
│ title           │   │   │ body            │       │
│ body            │   │   │ state           │       │
│ author_id       │───┼───│ commit_id       │       │
│ head_branch     │   │   │ created_at      │       │
│ head_sha        │   │   └─────────────────┘       │
│ base_branch     │   │                             │
│ base_sha        │   │   ┌─────────────────┐       │
│ state           │   │   │ review_comments │       │
│ is_draft        │   │   ├─────────────────┤       │
│ merged_at       │   │   │ id              │       │
│ merged_by_id    │   └───│ review_id       │       │
│ merge_commit    │       │ user_id         │───────┤
│ created_at      │       │ path            │       │
│ updated_at      │       │ position        │       │
│ closed_at       │       │ line            │       │
└────────┬────────┘       │ body            │       │
         │                │ created_at      │       │
         │                └─────────────────┘       │
         │                                          │
         │            ┌─────────────────┐           │
         └────────────│    comments     │           │
                      ├─────────────────┤           │
                      │ id              │           │
                      │ target_type     │ (issue/pr)│
                      │ target_id       │           │
                      │ user_id         │───────────┘
                      │ body            │
                      │ created_at      │
                      │ updated_at      │
                      └─────────────────┘

┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│    releases     │       │  release_assets │       │   activities    │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id              │───────│ id              │       │ id              │
│ repo_id         │       │ release_id      │       │ actor_id        │
│ tag_name        │       │ name            │       │ event_type      │
│ target_commit   │       │ size            │       │ repo_id         │
│ name            │       │ content_type    │       │ target_type     │
│ body            │       │ download_count  │       │ target_id       │
│ is_draft        │       │ created_at      │       │ payload         │
│ is_prerelease   │       └─────────────────┘       │ created_at      │
│ author_id       │                                 │ is_public       │
│ created_at      │       ┌─────────────────┐       └─────────────────┘
│ published_at    │       │  notifications  │
└─────────────────┘       ├─────────────────┤       ┌─────────────────┐
                          │ id              │       │    webhooks     │
                          │ user_id         │       ├─────────────────┤
                          │ repo_id         │       │ id              │
                          │ type            │       │ repo_id         │
                          │ actor_id        │       │ url             │
                          │ target_type     │       │ secret          │
                          │ target_id       │       │ events          │
                          │ read            │       │ active          │
                          │ created_at      │       │ created_at      │
                          └─────────────────┘       │ last_response   │
                                                    └─────────────────┘

┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│    sessions     │       │   ssh_keys      │       │   api_tokens    │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id              │       │ id              │       │ id              │
│ user_id         │       │ user_id         │       │ user_id         │
│ expires_at      │       │ name            │       │ name            │
│ user_agent      │       │ key             │       │ token_hash      │
│ ip_address      │       │ fingerprint     │       │ scopes          │
│ created_at      │       │ created_at      │       │ expires_at      │
│ last_active     │       │ last_used_at    │       │ created_at      │
└─────────────────┘       └─────────────────┘       │ last_used_at    │
                                                    └─────────────────┘
```

### SQL Schema

```sql
-- Users and Authentication
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR PRIMARY KEY,
    username VARCHAR UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    full_name VARCHAR DEFAULT '',
    avatar_url VARCHAR DEFAULT '',
    bio TEXT DEFAULT '',
    location VARCHAR DEFAULT '',
    website VARCHAR DEFAULT '',
    company VARCHAR DEFAULT '',
    is_admin BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    user_agent VARCHAR DEFAULT '',
    ip_address VARCHAR DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ssh_keys (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    public_key TEXT NOT NULL,
    fingerprint VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_tokens (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    token_hash VARCHAR NOT NULL,
    scopes VARCHAR DEFAULT '',
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP
);

-- Organizations and Teams
CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    slug VARCHAR UNIQUE NOT NULL,
    display_name VARCHAR DEFAULT '',
    description TEXT DEFAULT '',
    avatar_url VARCHAR DEFAULT '',
    location VARCHAR DEFAULT '',
    website VARCHAR DEFAULT '',
    email VARCHAR DEFAULT '',
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS org_members (
    id VARCHAR PRIMARY KEY,
    org_id VARCHAR NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR NOT NULL DEFAULT 'member', -- owner, admin, member
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(org_id, user_id)
);

CREATE TABLE IF NOT EXISTS teams (
    id VARCHAR PRIMARY KEY,
    org_id VARCHAR NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    slug VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    permission VARCHAR NOT NULL DEFAULT 'read', -- read, write, admin
    parent_id VARCHAR REFERENCES teams(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(org_id, slug)
);

CREATE TABLE IF NOT EXISTS team_members (
    id VARCHAR PRIMARY KEY,
    team_id VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR NOT NULL DEFAULT 'member', -- maintainer, member
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(team_id, user_id)
);

CREATE TABLE IF NOT EXISTS team_repos (
    id VARCHAR PRIMARY KEY,
    team_id VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    repo_id VARCHAR NOT NULL,
    permission VARCHAR NOT NULL DEFAULT 'read', -- read, write, admin
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(team_id, repo_id)
);

-- Repositories
CREATE TABLE IF NOT EXISTS repositories (
    id VARCHAR PRIMARY KEY,
    owner_id VARCHAR NOT NULL,
    owner_type VARCHAR NOT NULL DEFAULT 'user', -- user, org
    name VARCHAR NOT NULL,
    slug VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    website VARCHAR DEFAULT '',
    default_branch VARCHAR DEFAULT 'main',
    is_private BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_template BOOLEAN DEFAULT FALSE,
    is_fork BOOLEAN DEFAULT FALSE,
    forked_from_id VARCHAR REFERENCES repositories(id) ON DELETE SET NULL,
    star_count INTEGER DEFAULT 0,
    fork_count INTEGER DEFAULT 0,
    watcher_count INTEGER DEFAULT 0,
    open_issue_count INTEGER DEFAULT 0,
    open_pr_count INTEGER DEFAULT 0,
    size_kb INTEGER DEFAULT 0,
    topics VARCHAR DEFAULT '',
    license VARCHAR DEFAULT '',
    has_issues BOOLEAN DEFAULT TRUE,
    has_wiki BOOLEAN DEFAULT FALSE,
    has_projects BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    pushed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS collaborators (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR NOT NULL DEFAULT 'read', -- read, triage, write, maintain, admin
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repo_id, user_id)
);

CREATE TABLE IF NOT EXISTS stars (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    repo_id VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, repo_id)
);

CREATE TABLE IF NOT EXISTS watches (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    repo_id VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    level VARCHAR NOT NULL DEFAULT 'watching', -- participating, watching, ignoring, custom
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, repo_id)
);

-- Labels and Milestones
CREATE TABLE IF NOT EXISTS labels (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    color VARCHAR NOT NULL DEFAULT '0366d6',
    description TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repo_id, name)
);

CREATE TABLE IF NOT EXISTS milestones (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    state VARCHAR NOT NULL DEFAULT 'open', -- open, closed
    due_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    UNIQUE(repo_id, number)
);

-- Issues
CREATE TABLE IF NOT EXISTS issues (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    body TEXT DEFAULT '',
    author_id VARCHAR NOT NULL REFERENCES users(id),
    assignee_id VARCHAR REFERENCES users(id),
    state VARCHAR NOT NULL DEFAULT 'open', -- open, closed
    state_reason VARCHAR DEFAULT '', -- completed, not_planned, reopened
    is_locked BOOLEAN DEFAULT FALSE,
    lock_reason VARCHAR DEFAULT '',
    milestone_id VARCHAR REFERENCES milestones(id) ON DELETE SET NULL,
    comment_count INTEGER DEFAULT 0,
    reactions_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    closed_by_id VARCHAR REFERENCES users(id),
    UNIQUE(repo_id, number)
);

CREATE TABLE IF NOT EXISTS issue_labels (
    id VARCHAR PRIMARY KEY,
    issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id VARCHAR NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(issue_id, label_id)
);

CREATE TABLE IF NOT EXISTS issue_assignees (
    id VARCHAR PRIMARY KEY,
    issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(issue_id, user_id)
);

-- Pull Requests
CREATE TABLE IF NOT EXISTS pull_requests (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    body TEXT DEFAULT '',
    author_id VARCHAR NOT NULL REFERENCES users(id),
    head_repo_id VARCHAR REFERENCES repositories(id) ON DELETE SET NULL,
    head_branch VARCHAR NOT NULL,
    head_sha VARCHAR NOT NULL,
    base_branch VARCHAR NOT NULL,
    base_sha VARCHAR NOT NULL,
    state VARCHAR NOT NULL DEFAULT 'open', -- open, closed, merged
    is_draft BOOLEAN DEFAULT FALSE,
    is_locked BOOLEAN DEFAULT FALSE,
    mergeable BOOLEAN DEFAULT TRUE,
    mergeable_state VARCHAR DEFAULT '', -- clean, dirty, unknown, blocked, unstable
    merge_commit_sha VARCHAR DEFAULT '',
    merged_at TIMESTAMP,
    merged_by_id VARCHAR REFERENCES users(id),
    additions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    changed_files INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    review_comments INTEGER DEFAULT 0,
    commits INTEGER DEFAULT 0,
    milestone_id VARCHAR REFERENCES milestones(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    UNIQUE(repo_id, number)
);

CREATE TABLE IF NOT EXISTS pr_labels (
    id VARCHAR PRIMARY KEY,
    pr_id VARCHAR NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    label_id VARCHAR NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pr_id, label_id)
);

CREATE TABLE IF NOT EXISTS pr_assignees (
    id VARCHAR PRIMARY KEY,
    pr_id VARCHAR NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pr_id, user_id)
);

CREATE TABLE IF NOT EXISTS pr_reviewers (
    id VARCHAR PRIMARY KEY,
    pr_id VARCHAR NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state VARCHAR DEFAULT 'pending', -- pending, approved, changes_requested, commented, dismissed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pr_id, user_id)
);

CREATE TABLE IF NOT EXISTS pr_reviews (
    id VARCHAR PRIMARY KEY,
    pr_id VARCHAR NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id),
    body TEXT DEFAULT '',
    state VARCHAR NOT NULL, -- pending, approved, changes_requested, commented, dismissed
    commit_sha VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    submitted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS review_comments (
    id VARCHAR PRIMARY KEY,
    review_id VARCHAR NOT NULL REFERENCES pr_reviews(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id),
    path VARCHAR NOT NULL,
    position INTEGER,
    original_position INTEGER,
    diff_hunk TEXT,
    line INTEGER,
    original_line INTEGER,
    side VARCHAR DEFAULT 'RIGHT', -- LEFT, RIGHT
    body TEXT NOT NULL,
    in_reply_to_id VARCHAR REFERENCES review_comments(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Comments (for issues and PRs)
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR PRIMARY KEY,
    target_type VARCHAR NOT NULL, -- issue, pull_request
    target_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL REFERENCES users(id),
    body TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Releases
CREATE TABLE IF NOT EXISTS releases (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    tag_name VARCHAR NOT NULL,
    target_commitish VARCHAR NOT NULL DEFAULT 'main',
    name VARCHAR DEFAULT '',
    body TEXT DEFAULT '',
    is_draft BOOLEAN DEFAULT FALSE,
    is_prerelease BOOLEAN DEFAULT FALSE,
    author_id VARCHAR NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    UNIQUE(repo_id, tag_name)
);

CREATE TABLE IF NOT EXISTS release_assets (
    id VARCHAR PRIMARY KEY,
    release_id VARCHAR NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    label VARCHAR DEFAULT '',
    content_type VARCHAR NOT NULL,
    size_bytes INTEGER NOT NULL,
    download_count INTEGER DEFAULT 0,
    uploader_id VARCHAR NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR REFERENCES repositories(id) ON DELETE CASCADE,
    org_id VARCHAR REFERENCES organizations(id) ON DELETE CASCADE,
    url VARCHAR NOT NULL,
    secret VARCHAR DEFAULT '',
    content_type VARCHAR DEFAULT 'json', -- json, form
    events VARCHAR NOT NULL DEFAULT 'push', -- comma-separated: push,pull_request,issues,...
    active BOOLEAN DEFAULT TRUE,
    insecure_ssl BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_response_code INTEGER,
    last_response_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR PRIMARY KEY,
    webhook_id VARCHAR NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event VARCHAR NOT NULL,
    guid VARCHAR NOT NULL,
    payload TEXT NOT NULL,
    request_headers TEXT,
    response_headers TEXT,
    response_body TEXT,
    status_code INTEGER,
    delivered BOOLEAN DEFAULT FALSE,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    repo_id VARCHAR REFERENCES repositories(id) ON DELETE CASCADE,
    type VARCHAR NOT NULL, -- mention, review_requested, state_change, comment, etc.
    actor_id VARCHAR REFERENCES users(id),
    target_type VARCHAR NOT NULL, -- issue, pull_request, commit, release
    target_id VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    reason VARCHAR NOT NULL, -- assign, author, comment, mention, review_requested, subscribed
    unread BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_read_at TIMESTAMP
);

-- Activity Feed
CREATE TABLE IF NOT EXISTS activities (
    id VARCHAR PRIMARY KEY,
    actor_id VARCHAR NOT NULL REFERENCES users(id),
    event_type VARCHAR NOT NULL, -- create, delete, push, pull_request, issues, comment, star, fork, etc.
    repo_id VARCHAR REFERENCES repositories(id) ON DELETE CASCADE,
    target_type VARCHAR, -- repository, issue, pull_request, comment, etc.
    target_id VARCHAR,
    ref VARCHAR DEFAULT '',
    ref_type VARCHAR DEFAULT '',
    payload TEXT DEFAULT '{}',
    is_public BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Reactions (for issues, comments, PRs)
CREATE TABLE IF NOT EXISTS reactions (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR NOT NULL, -- issue, pull_request, comment, review_comment
    target_id VARCHAR NOT NULL,
    content VARCHAR NOT NULL, -- +1, -1, laugh, hooray, confused, heart, rocket, eyes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, target_type, target_id, content)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_repos_owner ON repositories(owner_id, owner_type);
CREATE INDEX IF NOT EXISTS idx_repos_name ON repositories(slug);
CREATE INDEX IF NOT EXISTS idx_issues_repo ON issues(repo_id);
CREATE INDEX IF NOT EXISTS idx_issues_state ON issues(repo_id, state);
CREATE INDEX IF NOT EXISTS idx_issues_author ON issues(author_id);
CREATE INDEX IF NOT EXISTS idx_prs_repo ON pull_requests(repo_id);
CREATE INDEX IF NOT EXISTS idx_prs_state ON pull_requests(repo_id, state);
CREATE INDEX IF NOT EXISTS idx_prs_author ON pull_requests(author_id);
CREATE INDEX IF NOT EXISTS idx_comments_target ON comments(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_activities_actor ON activities(actor_id);
CREATE INDEX IF NOT EXISTS idx_activities_repo ON activities(repo_id);
CREATE INDEX IF NOT EXISTS idx_activities_created ON activities(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, unread);
CREATE INDEX IF NOT EXISTS idx_stars_user ON stars(user_id);
CREATE INDEX IF NOT EXISTS idx_stars_repo ON stars(repo_id);
```

---

## API Design

### Authentication Endpoints

```
POST   /api/v1/auth/register          Register new user
POST   /api/v1/auth/login             Login
POST   /api/v1/auth/logout            Logout
POST   /api/v1/auth/refresh           Refresh session
POST   /api/v1/auth/forgot-password   Request password reset
POST   /api/v1/auth/reset-password    Reset password with token
GET    /api/v1/auth/me                Get current user
```

### User Endpoints

```
GET    /api/v1/users                  List users (admin)
GET    /api/v1/users/:username        Get user profile
PATCH  /api/v1/users/:username        Update user (self or admin)
DELETE /api/v1/users/:username        Delete user (self or admin)

GET    /api/v1/user                   Get authenticated user
PATCH  /api/v1/user                   Update authenticated user
GET    /api/v1/user/repos             List user's repos
GET    /api/v1/user/orgs              List user's organizations
GET    /api/v1/user/starred           List starred repos
GET    /api/v1/user/notifications     List notifications
PATCH  /api/v1/user/notifications     Mark notifications as read

GET    /api/v1/user/keys              List SSH keys
POST   /api/v1/user/keys              Add SSH key
GET    /api/v1/user/keys/:id          Get SSH key
DELETE /api/v1/user/keys/:id          Delete SSH key

GET    /api/v1/user/tokens            List API tokens
POST   /api/v1/user/tokens            Create API token
DELETE /api/v1/user/tokens/:id        Delete API token
```

### Repository Endpoints

```
GET    /api/v1/repos                            List all accessible repos
POST   /api/v1/repos                            Create repository
GET    /api/v1/repos/:owner/:repo               Get repository
PATCH  /api/v1/repos/:owner/:repo               Update repository
DELETE /api/v1/repos/:owner/:repo               Delete repository
POST   /api/v1/repos/:owner/:repo/transfer      Transfer ownership

GET    /api/v1/repos/:owner/:repo/contributors  List contributors
GET    /api/v1/repos/:owner/:repo/languages     List languages
GET    /api/v1/repos/:owner/:repo/topics        Get topics
PUT    /api/v1/repos/:owner/:repo/topics        Replace topics

GET    /api/v1/repos/:owner/:repo/collaborators           List collaborators
PUT    /api/v1/repos/:owner/:repo/collaborators/:user     Add collaborator
DELETE /api/v1/repos/:owner/:repo/collaborators/:user     Remove collaborator
GET    /api/v1/repos/:owner/:repo/collaborators/:user/permission  Check permission

GET    /api/v1/repos/:owner/:repo/forks         List forks
POST   /api/v1/repos/:owner/:repo/forks         Fork repository

GET    /api/v1/repos/:owner/:repo/stargazers    List stargazers
GET    /api/v1/repos/:owner/:repo/watchers      List watchers
PUT    /api/v1/repos/:owner/:repo/subscription  Watch repo
DELETE /api/v1/repos/:owner/:repo/subscription  Unwatch repo
PUT    /api/v1/user/starred/:owner/:repo        Star repo
DELETE /api/v1/user/starred/:owner/:repo        Unstar repo
```

### Git Data Endpoints

```
GET    /api/v1/repos/:owner/:repo/branches              List branches
GET    /api/v1/repos/:owner/:repo/branches/:branch      Get branch
POST   /api/v1/repos/:owner/:repo/branches              Create branch
DELETE /api/v1/repos/:owner/:repo/branches/:branch      Delete branch
POST   /api/v1/repos/:owner/:repo/branches/:branch/protection  Set protection

GET    /api/v1/repos/:owner/:repo/commits               List commits
GET    /api/v1/repos/:owner/:repo/commits/:sha          Get commit
GET    /api/v1/repos/:owner/:repo/compare/:base...:head Compare commits

GET    /api/v1/repos/:owner/:repo/contents/:path        Get contents (file/dir)
PUT    /api/v1/repos/:owner/:repo/contents/:path        Create/update file
DELETE /api/v1/repos/:owner/:repo/contents/:path        Delete file

GET    /api/v1/repos/:owner/:repo/git/trees/:sha        Get tree
POST   /api/v1/repos/:owner/:repo/git/trees             Create tree
GET    /api/v1/repos/:owner/:repo/git/blobs/:sha        Get blob
POST   /api/v1/repos/:owner/:repo/git/blobs             Create blob
GET    /api/v1/repos/:owner/:repo/git/refs              List refs
GET    /api/v1/repos/:owner/:repo/git/refs/:ref         Get ref
POST   /api/v1/repos/:owner/:repo/git/refs              Create ref
PATCH  /api/v1/repos/:owner/:repo/git/refs/:ref         Update ref
DELETE /api/v1/repos/:owner/:repo/git/refs/:ref         Delete ref

GET    /api/v1/repos/:owner/:repo/tags                  List tags
GET    /api/v1/repos/:owner/:repo/tags/:tag             Get tag
POST   /api/v1/repos/:owner/:repo/git/tags              Create tag
```

### Issues Endpoints

```
GET    /api/v1/repos/:owner/:repo/issues                List issues
POST   /api/v1/repos/:owner/:repo/issues                Create issue
GET    /api/v1/repos/:owner/:repo/issues/:number        Get issue
PATCH  /api/v1/repos/:owner/:repo/issues/:number        Update issue
DELETE /api/v1/repos/:owner/:repo/issues/:number        Delete issue (admin)
PUT    /api/v1/repos/:owner/:repo/issues/:number/lock   Lock issue
DELETE /api/v1/repos/:owner/:repo/issues/:number/lock   Unlock issue

GET    /api/v1/repos/:owner/:repo/issues/:number/comments      List comments
POST   /api/v1/repos/:owner/:repo/issues/:number/comments      Add comment
GET    /api/v1/repos/:owner/:repo/issues/comments/:id          Get comment
PATCH  /api/v1/repos/:owner/:repo/issues/comments/:id          Update comment
DELETE /api/v1/repos/:owner/:repo/issues/comments/:id          Delete comment

GET    /api/v1/repos/:owner/:repo/issues/:number/labels        List issue labels
POST   /api/v1/repos/:owner/:repo/issues/:number/labels        Add labels
PUT    /api/v1/repos/:owner/:repo/issues/:number/labels        Replace labels
DELETE /api/v1/repos/:owner/:repo/issues/:number/labels/:name  Remove label

GET    /api/v1/repos/:owner/:repo/issues/:number/assignees     List assignees
POST   /api/v1/repos/:owner/:repo/issues/:number/assignees     Add assignees
DELETE /api/v1/repos/:owner/:repo/issues/:number/assignees     Remove assignees

GET    /api/v1/repos/:owner/:repo/issues/:number/timeline      Get timeline
GET    /api/v1/repos/:owner/:repo/issues/:number/reactions     List reactions
POST   /api/v1/repos/:owner/:repo/issues/:number/reactions     Add reaction
DELETE /api/v1/repos/:owner/:repo/issues/:number/reactions/:id Delete reaction
```

### Pull Request Endpoints

```
GET    /api/v1/repos/:owner/:repo/pulls                 List PRs
POST   /api/v1/repos/:owner/:repo/pulls                 Create PR
GET    /api/v1/repos/:owner/:repo/pulls/:number         Get PR
PATCH  /api/v1/repos/:owner/:repo/pulls/:number         Update PR
PUT    /api/v1/repos/:owner/:repo/pulls/:number/merge   Merge PR
GET    /api/v1/repos/:owner/:repo/pulls/:number/merge   Check merge status

GET    /api/v1/repos/:owner/:repo/pulls/:number/commits List PR commits
GET    /api/v1/repos/:owner/:repo/pulls/:number/files   List changed files
GET    /api/v1/repos/:owner/:repo/pulls/:number/diff    Get diff

GET    /api/v1/repos/:owner/:repo/pulls/:number/reviews          List reviews
POST   /api/v1/repos/:owner/:repo/pulls/:number/reviews          Create review
GET    /api/v1/repos/:owner/:repo/pulls/:number/reviews/:id      Get review
PUT    /api/v1/repos/:owner/:repo/pulls/:number/reviews/:id      Update review
DELETE /api/v1/repos/:owner/:repo/pulls/:number/reviews/:id      Delete review
POST   /api/v1/repos/:owner/:repo/pulls/:number/reviews/:id/submit Submit review
POST   /api/v1/repos/:owner/:repo/pulls/:number/reviews/:id/dismiss Dismiss review

GET    /api/v1/repos/:owner/:repo/pulls/:number/comments         List review comments
POST   /api/v1/repos/:owner/:repo/pulls/:number/comments         Create comment
GET    /api/v1/repos/:owner/:repo/pulls/comments/:id             Get comment
PATCH  /api/v1/repos/:owner/:repo/pulls/comments/:id             Update comment
DELETE /api/v1/repos/:owner/:repo/pulls/comments/:id             Delete comment

GET    /api/v1/repos/:owner/:repo/pulls/:number/requested_reviewers  Get reviewers
POST   /api/v1/repos/:owner/:repo/pulls/:number/requested_reviewers  Request reviewers
DELETE /api/v1/repos/:owner/:repo/pulls/:number/requested_reviewers  Remove reviewers
```

### Labels Endpoints

```
GET    /api/v1/repos/:owner/:repo/labels         List labels
POST   /api/v1/repos/:owner/:repo/labels         Create label
GET    /api/v1/repos/:owner/:repo/labels/:name   Get label
PATCH  /api/v1/repos/:owner/:repo/labels/:name   Update label
DELETE /api/v1/repos/:owner/:repo/labels/:name   Delete label
```

### Milestones Endpoints

```
GET    /api/v1/repos/:owner/:repo/milestones         List milestones
POST   /api/v1/repos/:owner/:repo/milestones         Create milestone
GET    /api/v1/repos/:owner/:repo/milestones/:number Get milestone
PATCH  /api/v1/repos/:owner/:repo/milestones/:number Update milestone
DELETE /api/v1/repos/:owner/:repo/milestones/:number Delete milestone
```

### Releases Endpoints

```
GET    /api/v1/repos/:owner/:repo/releases           List releases
POST   /api/v1/repos/:owner/:repo/releases           Create release
GET    /api/v1/repos/:owner/:repo/releases/:id       Get release
GET    /api/v1/repos/:owner/:repo/releases/latest    Get latest release
GET    /api/v1/repos/:owner/:repo/releases/tags/:tag Get by tag
PATCH  /api/v1/repos/:owner/:repo/releases/:id       Update release
DELETE /api/v1/repos/:owner/:repo/releases/:id       Delete release

GET    /api/v1/repos/:owner/:repo/releases/:id/assets        List assets
POST   /api/v1/repos/:owner/:repo/releases/:id/assets        Upload asset
GET    /api/v1/repos/:owner/:repo/releases/assets/:id        Get asset
PATCH  /api/v1/repos/:owner/:repo/releases/assets/:id        Update asset
DELETE /api/v1/repos/:owner/:repo/releases/assets/:id        Delete asset
```

### Organization Endpoints

```
GET    /api/v1/orgs                           List organizations
POST   /api/v1/orgs                           Create organization
GET    /api/v1/orgs/:org                      Get organization
PATCH  /api/v1/orgs/:org                      Update organization
DELETE /api/v1/orgs/:org                      Delete organization

GET    /api/v1/orgs/:org/members              List members
GET    /api/v1/orgs/:org/members/:user        Check membership
PUT    /api/v1/orgs/:org/members/:user        Add member
DELETE /api/v1/orgs/:org/members/:user        Remove member
PATCH  /api/v1/orgs/:org/memberships/:user    Update membership

GET    /api/v1/orgs/:org/repos                List org repos
POST   /api/v1/orgs/:org/repos                Create org repo

GET    /api/v1/orgs/:org/teams                List teams
POST   /api/v1/orgs/:org/teams                Create team
GET    /api/v1/orgs/:org/teams/:team          Get team
PATCH  /api/v1/orgs/:org/teams/:team          Update team
DELETE /api/v1/orgs/:org/teams/:team          Delete team

GET    /api/v1/orgs/:org/teams/:team/members  List team members
PUT    /api/v1/orgs/:org/teams/:team/members/:user    Add team member
DELETE /api/v1/orgs/:org/teams/:team/members/:user    Remove team member

GET    /api/v1/orgs/:org/teams/:team/repos    List team repos
PUT    /api/v1/orgs/:org/teams/:team/repos/:owner/:repo    Add repo to team
DELETE /api/v1/orgs/:org/teams/:team/repos/:owner/:repo    Remove repo from team
```

### Webhooks Endpoints

```
GET    /api/v1/repos/:owner/:repo/hooks       List webhooks
POST   /api/v1/repos/:owner/:repo/hooks       Create webhook
GET    /api/v1/repos/:owner/:repo/hooks/:id   Get webhook
PATCH  /api/v1/repos/:owner/:repo/hooks/:id   Update webhook
DELETE /api/v1/repos/:owner/:repo/hooks/:id   Delete webhook
POST   /api/v1/repos/:owner/:repo/hooks/:id/pings    Ping webhook
GET    /api/v1/repos/:owner/:repo/hooks/:id/deliveries  List deliveries
```

### Search Endpoints

```
GET    /api/v1/search/repositories   Search repositories
GET    /api/v1/search/commits        Search commits
GET    /api/v1/search/code           Search code
GET    /api/v1/search/issues         Search issues/PRs
GET    /api/v1/search/users          Search users
GET    /api/v1/search/topics         Search topics
```

### Activity Endpoints

```
GET    /api/v1/feeds                         Get activity feeds URLs
GET    /api/v1/users/:username/events        List user events
GET    /api/v1/users/:username/events/public List public user events
GET    /api/v1/repos/:owner/:repo/events     List repo events
GET    /api/v1/orgs/:org/events              List org events
```

### Git Smart HTTP Protocol

```
GET    /:owner/:repo.git/info/refs           Git info refs
POST   /:owner/:repo.git/git-upload-pack     Git upload pack (clone/fetch)
POST   /:owner/:repo.git/git-receive-pack    Git receive pack (push)
GET    /:owner/:repo.git/HEAD                Get HEAD reference
GET    /:owner/:repo.git/objects/:sha        Get git object
```

---

## Feature Modules

### 1. Users Module

**Models:**
```go
type User struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`
    FullName     string    `json:"full_name"`
    AvatarURL    string    `json:"avatar_url"`
    Bio          string    `json:"bio"`
    Location     string    `json:"location"`
    Website      string    `json:"website"`
    Company      string    `json:"company"`
    IsAdmin      bool      `json:"is_admin"`
    IsActive     bool      `json:"is_active"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
    ID           string    `json:"id"`
    UserID       string    `json:"user_id"`
    ExpiresAt    time.Time `json:"expires_at"`
    UserAgent    string    `json:"user_agent"`
    IPAddress    string    `json:"ip_address"`
    CreatedAt    time.Time `json:"created_at"`
    LastActiveAt time.Time `json:"last_active_at"`
}
```

**API Interface:**
```go
type API interface {
    // Registration/Auth
    Register(ctx context.Context, in *RegisterIn) (*User, *Session, error)
    Login(ctx context.Context, in *LoginIn) (*User, *Session, error)
    Logout(ctx context.Context, sessionID string) error
    ValidateSession(ctx context.Context, sessionID string) (*User, error)

    // User CRUD
    GetByID(ctx context.Context, id string) (*User, error)
    GetByUsername(ctx context.Context, username string) (*User, error)
    Update(ctx context.Context, id string, in *UpdateIn) (*User, error)
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, opts *ListOpts) ([]*User, error)

    // SSH Keys
    ListSSHKeys(ctx context.Context, userID string) ([]*SSHKey, error)
    AddSSHKey(ctx context.Context, userID string, in *AddSSHKeyIn) (*SSHKey, error)
    DeleteSSHKey(ctx context.Context, userID, keyID string) error

    // API Tokens
    ListTokens(ctx context.Context, userID string) ([]*APIToken, error)
    CreateToken(ctx context.Context, userID string, in *CreateTokenIn) (*APIToken, string, error)
    DeleteToken(ctx context.Context, userID, tokenID string) error
    ValidateToken(ctx context.Context, token string) (*User, error)
}
```

### 2. Repositories Module

**Models:**
```go
type Repository struct {
    ID             string    `json:"id"`
    OwnerID        string    `json:"owner_id"`
    OwnerType      string    `json:"owner_type"` // user, org
    OwnerName      string    `json:"owner_name"` // populated from join
    Name           string    `json:"name"`
    Slug           string    `json:"slug"`
    Description    string    `json:"description"`
    Website        string    `json:"website"`
    DefaultBranch  string    `json:"default_branch"`
    IsPrivate      bool      `json:"is_private"`
    IsArchived     bool      `json:"is_archived"`
    IsTemplate     bool      `json:"is_template"`
    IsFork         bool      `json:"is_fork"`
    ForkedFromID   string    `json:"forked_from_id,omitempty"`
    StarCount      int       `json:"star_count"`
    ForkCount      int       `json:"fork_count"`
    WatcherCount   int       `json:"watcher_count"`
    OpenIssueCount int       `json:"open_issue_count"`
    OpenPRCount    int       `json:"open_pr_count"`
    SizeKB         int       `json:"size_kb"`
    Topics         []string  `json:"topics"`
    License        string    `json:"license"`
    HasIssues      bool      `json:"has_issues"`
    HasWiki        bool      `json:"has_wiki"`
    HasProjects    bool      `json:"has_projects"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
    PushedAt       time.Time `json:"pushed_at,omitempty"`
}

type Permission string

const (
    PermissionRead     Permission = "read"
    PermissionTriage   Permission = "triage"
    PermissionWrite    Permission = "write"
    PermissionMaintain Permission = "maintain"
    PermissionAdmin    Permission = "admin"
)
```

**API Interface:**
```go
type API interface {
    // CRUD
    Create(ctx context.Context, userID string, in *CreateIn) (*Repository, error)
    GetByID(ctx context.Context, id string) (*Repository, error)
    GetByOwnerAndName(ctx context.Context, owner, name string) (*Repository, error)
    Update(ctx context.Context, id string, in *UpdateIn) (*Repository, error)
    Delete(ctx context.Context, id string) error

    // Listing
    List(ctx context.Context, opts *ListOpts) ([]*Repository, error)
    ListByUser(ctx context.Context, userID string, opts *ListOpts) ([]*Repository, error)
    ListByOrg(ctx context.Context, orgID string, opts *ListOpts) ([]*Repository, error)

    // Collaborators
    ListCollaborators(ctx context.Context, repoID string) ([]*Collaborator, error)
    AddCollaborator(ctx context.Context, repoID, userID string, perm Permission) error
    RemoveCollaborator(ctx context.Context, repoID, userID string) error
    GetPermission(ctx context.Context, repoID, userID string) (Permission, error)
    CanAccess(ctx context.Context, repoID, userID string, required Permission) bool

    // Stars
    Star(ctx context.Context, userID, repoID string) error
    Unstar(ctx context.Context, userID, repoID string) error
    IsStarred(ctx context.Context, userID, repoID string) (bool, error)
    ListStargazers(ctx context.Context, repoID string) ([]*User, error)
    ListStarred(ctx context.Context, userID string) ([]*Repository, error)

    // Watching
    Watch(ctx context.Context, userID, repoID, level string) error
    Unwatch(ctx context.Context, userID, repoID string) error
    GetWatchLevel(ctx context.Context, userID, repoID string) (string, error)
    ListWatchers(ctx context.Context, repoID string) ([]*User, error)

    // Forking
    Fork(ctx context.Context, userID, repoID string, opts *ForkOpts) (*Repository, error)
    ListForks(ctx context.Context, repoID string) ([]*Repository, error)

    // Transfer
    Transfer(ctx context.Context, repoID, newOwnerID string) error
}
```

### 3. Issues Module

**Models:**
```go
type Issue struct {
    ID             string     `json:"id"`
    RepoID         string     `json:"repo_id"`
    Number         int        `json:"number"`
    Title          string     `json:"title"`
    Body           string     `json:"body"`
    AuthorID       string     `json:"author_id"`
    Author         *User      `json:"author,omitempty"`
    AssigneeID     string     `json:"assignee_id,omitempty"`
    Assignee       *User      `json:"assignee,omitempty"`
    Assignees      []*User    `json:"assignees,omitempty"`
    State          string     `json:"state"` // open, closed
    StateReason    string     `json:"state_reason,omitempty"`
    IsLocked       bool       `json:"is_locked"`
    LockReason     string     `json:"lock_reason,omitempty"`
    Labels         []*Label   `json:"labels,omitempty"`
    MilestoneID    string     `json:"milestone_id,omitempty"`
    Milestone      *Milestone `json:"milestone,omitempty"`
    CommentCount   int        `json:"comment_count"`
    ReactionsCount int        `json:"reactions_count"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    ClosedAt       *time.Time `json:"closed_at,omitempty"`
    ClosedByID     string     `json:"closed_by_id,omitempty"`
}
```

**API Interface:**
```go
type API interface {
    Create(ctx context.Context, repoID, authorID string, in *CreateIn) (*Issue, error)
    GetByID(ctx context.Context, id string) (*Issue, error)
    GetByNumber(ctx context.Context, repoID string, number int) (*Issue, error)
    Update(ctx context.Context, id string, in *UpdateIn) (*Issue, error)
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, repoID string, opts *ListOpts) ([]*Issue, int, error)

    // State
    Close(ctx context.Context, id, userID, reason string) error
    Reopen(ctx context.Context, id string) error
    Lock(ctx context.Context, id, reason string) error
    Unlock(ctx context.Context, id string) error

    // Assignees
    AddAssignees(ctx context.Context, id string, userIDs []string) error
    RemoveAssignees(ctx context.Context, id string, userIDs []string) error

    // Labels
    AddLabels(ctx context.Context, id string, labelIDs []string) error
    RemoveLabel(ctx context.Context, id, labelID string) error
    SetLabels(ctx context.Context, id string, labelIDs []string) error

    // Comments
    AddComment(ctx context.Context, id, userID string, in *AddCommentIn) (*Comment, error)
    UpdateComment(ctx context.Context, commentID string, in *UpdateCommentIn) (*Comment, error)
    DeleteComment(ctx context.Context, commentID string) error
    ListComments(ctx context.Context, id string) ([]*Comment, error)

    // Timeline
    GetTimeline(ctx context.Context, id string) ([]*TimelineEvent, error)

    // Reactions
    AddReaction(ctx context.Context, id, userID, content string) (*Reaction, error)
    RemoveReaction(ctx context.Context, reactionID string) error
    ListReactions(ctx context.Context, id string) ([]*Reaction, error)
}
```

### 4. Pull Requests Module

**Models:**
```go
type PullRequest struct {
    ID             string     `json:"id"`
    RepoID         string     `json:"repo_id"`
    Number         int        `json:"number"`
    Title          string     `json:"title"`
    Body           string     `json:"body"`
    AuthorID       string     `json:"author_id"`
    Author         *User      `json:"author,omitempty"`
    HeadRepoID     string     `json:"head_repo_id"`
    HeadBranch     string     `json:"head_branch"`
    HeadSHA        string     `json:"head_sha"`
    BaseBranch     string     `json:"base_branch"`
    BaseSHA        string     `json:"base_sha"`
    State          string     `json:"state"` // open, closed, merged
    IsDraft        bool       `json:"is_draft"`
    IsLocked       bool       `json:"is_locked"`
    Mergeable      bool       `json:"mergeable"`
    MergeableState string     `json:"mergeable_state"`
    MergeCommitSHA string     `json:"merge_commit_sha,omitempty"`
    MergedAt       *time.Time `json:"merged_at,omitempty"`
    MergedByID     string     `json:"merged_by_id,omitempty"`
    MergedBy       *User      `json:"merged_by,omitempty"`
    Additions      int        `json:"additions"`
    Deletions      int        `json:"deletions"`
    ChangedFiles   int        `json:"changed_files"`
    CommentCount   int        `json:"comment_count"`
    ReviewComments int        `json:"review_comments"`
    Commits        int        `json:"commits"`
    Labels         []*Label   `json:"labels,omitempty"`
    Assignees      []*User    `json:"assignees,omitempty"`
    Reviewers      []*User    `json:"reviewers,omitempty"`
    MilestoneID    string     `json:"milestone_id,omitempty"`
    Milestone      *Milestone `json:"milestone,omitempty"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    ClosedAt       *time.Time `json:"closed_at,omitempty"`
}

type Review struct {
    ID          string    `json:"id"`
    PRID        string    `json:"pr_id"`
    UserID      string    `json:"user_id"`
    User        *User     `json:"user,omitempty"`
    Body        string    `json:"body"`
    State       string    `json:"state"` // pending, approved, changes_requested, commented, dismissed
    CommitSHA   string    `json:"commit_sha"`
    CreatedAt   time.Time `json:"created_at"`
    SubmittedAt time.Time `json:"submitted_at,omitempty"`
}

type ReviewComment struct {
    ID               string    `json:"id"`
    ReviewID         string    `json:"review_id"`
    UserID           string    `json:"user_id"`
    User             *User     `json:"user,omitempty"`
    Path             string    `json:"path"`
    Position         int       `json:"position,omitempty"`
    OriginalPosition int       `json:"original_position,omitempty"`
    DiffHunk         string    `json:"diff_hunk,omitempty"`
    Line             int       `json:"line,omitempty"`
    OriginalLine     int       `json:"original_line,omitempty"`
    Side             string    `json:"side"` // LEFT, RIGHT
    Body             string    `json:"body"`
    InReplyToID      string    `json:"in_reply_to_id,omitempty"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

**API Interface:**
```go
type API interface {
    Create(ctx context.Context, repoID, authorID string, in *CreateIn) (*PullRequest, error)
    GetByID(ctx context.Context, id string) (*PullRequest, error)
    GetByNumber(ctx context.Context, repoID string, number int) (*PullRequest, error)
    Update(ctx context.Context, id string, in *UpdateIn) (*PullRequest, error)
    Close(ctx context.Context, id string) error
    Reopen(ctx context.Context, id string) error
    List(ctx context.Context, repoID string, opts *ListOpts) ([]*PullRequest, int, error)

    // Merge
    Merge(ctx context.Context, id, userID string, opts *MergeOpts) error
    GetMergeStatus(ctx context.Context, id string) (*MergeStatus, error)

    // Commits/Files
    ListCommits(ctx context.Context, id string) ([]*Commit, error)
    ListFiles(ctx context.Context, id string) ([]*DiffFile, error)
    GetDiff(ctx context.Context, id string) (string, error)

    // Reviews
    CreateReview(ctx context.Context, prID, userID string, in *CreateReviewIn) (*Review, error)
    GetReview(ctx context.Context, reviewID string) (*Review, error)
    UpdateReview(ctx context.Context, reviewID string, in *UpdateReviewIn) (*Review, error)
    SubmitReview(ctx context.Context, reviewID string, in *SubmitReviewIn) (*Review, error)
    DismissReview(ctx context.Context, reviewID, message string) error
    ListReviews(ctx context.Context, prID string) ([]*Review, error)

    // Review Comments
    CreateReviewComment(ctx context.Context, reviewID, userID string, in *CreateCommentIn) (*ReviewComment, error)
    UpdateReviewComment(ctx context.Context, commentID string, in *UpdateCommentIn) (*ReviewComment, error)
    DeleteReviewComment(ctx context.Context, commentID string) error
    ListReviewComments(ctx context.Context, prID string) ([]*ReviewComment, error)

    // Reviewers
    RequestReviewers(ctx context.Context, prID string, userIDs []string) error
    RemoveReviewers(ctx context.Context, prID string, userIDs []string) error
    ListRequestedReviewers(ctx context.Context, prID string) ([]*User, error)
}
```

### 5. Git Integration Module

**Repository Operations:**
```go
type GitRepository interface {
    // Repository management
    Init(ctx context.Context, bare bool) error
    Clone(ctx context.Context, url string) error

    // References
    GetRef(ctx context.Context, name string) (*Ref, error)
    ListRefs(ctx context.Context, prefix string) ([]*Ref, error)
    CreateRef(ctx context.Context, name, sha string) error
    UpdateRef(ctx context.Context, name, sha string) error
    DeleteRef(ctx context.Context, name string) error

    // Branches
    GetBranch(ctx context.Context, name string) (*Branch, error)
    ListBranches(ctx context.Context) ([]*Branch, error)
    CreateBranch(ctx context.Context, name, sha string) error
    DeleteBranch(ctx context.Context, name string) error
    GetDefaultBranch(ctx context.Context) (string, error)
    SetDefaultBranch(ctx context.Context, name string) error

    // Tags
    GetTag(ctx context.Context, name string) (*Tag, error)
    ListTags(ctx context.Context) ([]*Tag, error)
    CreateTag(ctx context.Context, name, sha, message string, annotated bool) error
    DeleteTag(ctx context.Context, name string) error

    // Commits
    GetCommit(ctx context.Context, sha string) (*Commit, error)
    ListCommits(ctx context.Context, opts *CommitListOpts) ([]*Commit, error)
    GetCommitsBetween(ctx context.Context, base, head string) ([]*Commit, error)

    // Trees
    GetTree(ctx context.Context, sha string, recursive bool) (*Tree, error)
    GetTreeEntry(ctx context.Context, sha, path string) (*TreeEntry, error)

    // Blobs
    GetBlob(ctx context.Context, sha string) (*Blob, error)
    GetBlobContent(ctx context.Context, sha string) (io.ReadCloser, error)

    // File operations
    GetFileContent(ctx context.Context, ref, path string) ([]byte, error)
    CreateFile(ctx context.Context, opts *CreateFileOpts) (*Commit, error)
    UpdateFile(ctx context.Context, opts *UpdateFileOpts) (*Commit, error)
    DeleteFile(ctx context.Context, opts *DeleteFileOpts) (*Commit, error)

    // Diff
    GetDiff(ctx context.Context, base, head string) (*Diff, error)
    GetDiffFiles(ctx context.Context, base, head string) ([]*DiffFile, error)

    // Blame
    GetBlame(ctx context.Context, ref, path string) ([]*BlameLine, error)

    // Compare
    Compare(ctx context.Context, base, head string) (*Comparison, error)

    // Merge
    CanMerge(ctx context.Context, base, head string) (bool, error)
    Merge(ctx context.Context, base, head, message string) (string, error)
}
```

**Git Models:**
```go
type Commit struct {
    SHA       string     `json:"sha"`
    Message   string     `json:"message"`
    Author    Signature  `json:"author"`
    Committer Signature  `json:"committer"`
    Parents   []string   `json:"parents"`
    TreeSHA   string     `json:"tree_sha"`
    Verified  bool       `json:"verified"`
    Stats     *CommitStats `json:"stats,omitempty"`
}

type Signature struct {
    Name  string    `json:"name"`
    Email string    `json:"email"`
    Date  time.Time `json:"date"`
}

type Branch struct {
    Name      string  `json:"name"`
    SHA       string  `json:"sha"`
    Commit    *Commit `json:"commit,omitempty"`
    Protected bool    `json:"protected"`
}

type Tag struct {
    Name       string    `json:"name"`
    SHA        string    `json:"sha"`
    Commit     *Commit   `json:"commit,omitempty"`
    Message    string    `json:"message,omitempty"`
    Tagger     *Signature `json:"tagger,omitempty"`
    Annotated  bool      `json:"annotated"`
}

type Tree struct {
    SHA     string       `json:"sha"`
    Entries []*TreeEntry `json:"entries"`
}

type TreeEntry struct {
    Path string `json:"path"`
    Mode string `json:"mode"`
    Type string `json:"type"` // blob, tree, commit
    Size int64  `json:"size,omitempty"`
    SHA  string `json:"sha"`
}

type Diff struct {
    Files     []*DiffFile `json:"files"`
    Additions int         `json:"additions"`
    Deletions int         `json:"deletions"`
    Changes   int         `json:"changes"`
}

type DiffFile struct {
    Path         string     `json:"path"`
    PreviousPath string     `json:"previous_path,omitempty"`
    Status       string     `json:"status"` // added, modified, deleted, renamed, copied
    Additions    int        `json:"additions"`
    Deletions    int        `json:"deletions"`
    Changes      int        `json:"changes"`
    Patch        string     `json:"patch,omitempty"`
    Hunks        []*DiffHunk `json:"hunks,omitempty"`
}

type DiffHunk struct {
    Header    string      `json:"header"`
    OldStart  int         `json:"old_start"`
    OldLines  int         `json:"old_lines"`
    NewStart  int         `json:"new_start"`
    NewLines  int         `json:"new_lines"`
    Lines     []*DiffLine `json:"lines"`
}

type DiffLine struct {
    Type      string `json:"type"` // context, addition, deletion
    Content   string `json:"content"`
    OldNumber int    `json:"old_number,omitempty"`
    NewNumber int    `json:"new_number,omitempty"`
}
```

### 6. Organizations & Teams Module

```go
type Organization struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Slug        string    `json:"slug"`
    DisplayName string    `json:"display_name"`
    Description string    `json:"description"`
    AvatarURL   string    `json:"avatar_url"`
    Location    string    `json:"location"`
    Website     string    `json:"website"`
    Email       string    `json:"email"`
    IsVerified  bool      `json:"is_verified"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type OrgMember struct {
    ID        string    `json:"id"`
    OrgID     string    `json:"org_id"`
    UserID    string    `json:"user_id"`
    User      *User     `json:"user,omitempty"`
    Role      string    `json:"role"` // owner, admin, member
    CreatedAt time.Time `json:"created_at"`
}

type Team struct {
    ID          string    `json:"id"`
    OrgID       string    `json:"org_id"`
    Name        string    `json:"name"`
    Slug        string    `json:"slug"`
    Description string    `json:"description"`
    Permission  string    `json:"permission"` // read, write, admin
    ParentID    string    `json:"parent_id,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type OrgsAPI interface {
    Create(ctx context.Context, userID string, in *CreateOrgIn) (*Organization, error)
    GetByID(ctx context.Context, id string) (*Organization, error)
    GetBySlug(ctx context.Context, slug string) (*Organization, error)
    Update(ctx context.Context, id string, in *UpdateOrgIn) (*Organization, error)
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, opts *ListOpts) ([]*Organization, error)
    ListByUser(ctx context.Context, userID string) ([]*Organization, error)

    // Members
    AddMember(ctx context.Context, orgID, userID, role string) error
    RemoveMember(ctx context.Context, orgID, userID string) error
    UpdateMemberRole(ctx context.Context, orgID, userID, role string) error
    ListMembers(ctx context.Context, orgID string) ([]*OrgMember, error)
    IsMember(ctx context.Context, orgID, userID string) (bool, error)
    GetMemberRole(ctx context.Context, orgID, userID string) (string, error)
}

type TeamsAPI interface {
    Create(ctx context.Context, orgID string, in *CreateTeamIn) (*Team, error)
    GetByID(ctx context.Context, id string) (*Team, error)
    GetBySlug(ctx context.Context, orgID, slug string) (*Team, error)
    Update(ctx context.Context, id string, in *UpdateTeamIn) (*Team, error)
    Delete(ctx context.Context, id string) error
    ListByOrg(ctx context.Context, orgID string) ([]*Team, error)

    // Members
    AddMember(ctx context.Context, teamID, userID string) error
    RemoveMember(ctx context.Context, teamID, userID string) error
    ListMembers(ctx context.Context, teamID string) ([]*TeamMember, error)

    // Repos
    AddRepo(ctx context.Context, teamID, repoID, permission string) error
    RemoveRepo(ctx context.Context, teamID, repoID string) error
    ListRepos(ctx context.Context, teamID string) ([]*Repository, error)
}
```

### 7. Notifications Module

```go
type Notification struct {
    ID         string    `json:"id"`
    UserID     string    `json:"user_id"`
    RepoID     string    `json:"repo_id,omitempty"`
    Repo       *Repository `json:"repository,omitempty"`
    Type       string    `json:"type"`
    ActorID    string    `json:"actor_id,omitempty"`
    Actor      *User     `json:"actor,omitempty"`
    TargetType string    `json:"target_type"` // issue, pull_request, commit, release
    TargetID   string    `json:"target_id"`
    Title      string    `json:"title"`
    Reason     string    `json:"reason"` // assign, author, comment, mention, review_requested, subscribed
    Unread     bool      `json:"unread"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
    LastReadAt *time.Time `json:"last_read_at,omitempty"`
}

type NotificationsAPI interface {
    List(ctx context.Context, userID string, opts *ListOpts) ([]*Notification, error)
    GetByID(ctx context.Context, id string) (*Notification, error)
    MarkAsRead(ctx context.Context, id string) error
    MarkAllAsRead(ctx context.Context, userID string) error
    MarkRepoAsRead(ctx context.Context, userID, repoID string) error

    // Internal - create notifications
    Create(ctx context.Context, in *CreateNotificationIn) (*Notification, error)
    CreateForMention(ctx context.Context, userID, actorID, repoID, targetType, targetID, title string) error
    CreateForReviewRequest(ctx context.Context, userID, actorID, repoID, prID, title string) error
    CreateForAssignment(ctx context.Context, userID, actorID, repoID, targetType, targetID, title string) error
}
```

### 8. Webhooks Module

```go
type Webhook struct {
    ID               string    `json:"id"`
    RepoID           string    `json:"repo_id,omitempty"`
    OrgID            string    `json:"org_id,omitempty"`
    URL              string    `json:"url"`
    Secret           string    `json:"-"`
    ContentType      string    `json:"content_type"` // json, form
    Events           []string  `json:"events"`
    Active           bool      `json:"active"`
    InsecureSSL      bool      `json:"insecure_ssl"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
    LastResponseCode int       `json:"last_response_code,omitempty"`
    LastResponseAt   *time.Time `json:"last_response_at,omitempty"`
}

type WebhookDelivery struct {
    ID              string    `json:"id"`
    WebhookID       string    `json:"webhook_id"`
    Event           string    `json:"event"`
    GUID            string    `json:"guid"`
    Payload         string    `json:"payload"`
    RequestHeaders  string    `json:"request_headers"`
    ResponseHeaders string    `json:"response_headers"`
    ResponseBody    string    `json:"response_body"`
    StatusCode      int       `json:"status_code"`
    Delivered       bool      `json:"delivered"`
    DurationMs      int       `json:"duration_ms"`
    CreatedAt       time.Time `json:"created_at"`
}

type WebhooksAPI interface {
    Create(ctx context.Context, in *CreateWebhookIn) (*Webhook, error)
    GetByID(ctx context.Context, id string) (*Webhook, error)
    Update(ctx context.Context, id string, in *UpdateWebhookIn) (*Webhook, error)
    Delete(ctx context.Context, id string) error
    ListByRepo(ctx context.Context, repoID string) ([]*Webhook, error)
    ListByOrg(ctx context.Context, orgID string) ([]*Webhook, error)

    Ping(ctx context.Context, id string) error
    ListDeliveries(ctx context.Context, webhookID string) ([]*WebhookDelivery, error)
    RedeliverDelivery(ctx context.Context, deliveryID string) error

    // Dispatch events
    DispatchPush(ctx context.Context, repoID string, payload *PushPayload) error
    DispatchPullRequest(ctx context.Context, repoID string, payload *PRPayload) error
    DispatchIssues(ctx context.Context, repoID string, payload *IssuePayload) error
    DispatchRelease(ctx context.Context, repoID string, payload *ReleasePayload) error
}
```

### 9. Activities Module

```go
type Activity struct {
    ID         string    `json:"id"`
    ActorID    string    `json:"actor_id"`
    Actor      *User     `json:"actor,omitempty"`
    EventType  string    `json:"event_type"`
    RepoID     string    `json:"repo_id,omitempty"`
    Repo       *Repository `json:"repo,omitempty"`
    TargetType string    `json:"target_type,omitempty"`
    TargetID   string    `json:"target_id,omitempty"`
    Ref        string    `json:"ref,omitempty"`
    RefType    string    `json:"ref_type,omitempty"`
    Payload    string    `json:"payload,omitempty"` // JSON
    IsPublic   bool      `json:"is_public"`
    CreatedAt  time.Time `json:"created_at"`
}

// Event types
const (
    EventPush             = "push"
    EventCreate           = "create"           // branch, tag
    EventDelete           = "delete"           // branch, tag
    EventPullRequest      = "pull_request"
    EventPullRequestReview = "pull_request_review"
    EventIssues           = "issues"
    EventIssueComment     = "issue_comment"
    EventCommitComment    = "commit_comment"
    EventRelease          = "release"
    EventFork             = "fork"
    EventWatch            = "watch"            // star
    EventMember           = "member"
    EventPublic           = "public"
)

type ActivitiesAPI interface {
    Create(ctx context.Context, in *CreateActivityIn) (*Activity, error)
    ListByUser(ctx context.Context, userID string, opts *ListOpts) ([]*Activity, error)
    ListByRepo(ctx context.Context, repoID string, opts *ListOpts) ([]*Activity, error)
    ListByOrg(ctx context.Context, orgID string, opts *ListOpts) ([]*Activity, error)
    ListPublic(ctx context.Context, opts *ListOpts) ([]*Activity, error)
    ListReceivedByUser(ctx context.Context, userID string, opts *ListOpts) ([]*Activity, error)
}
```

---

## UI Pages & Components

### Page Layout

```
┌──────────────────────────────────────────────────────────────────┐
│                          Header                                   │
│  ┌────────┐  ┌──────────────────┐              ┌──────┐ ┌──────┐ │
│  │ Logo   │  │ Search           │              │ Bell │ │Avatar│ │
│  └────────┘  └──────────────────┘              └──────┘ └──────┘ │
├──────────────────────────────────────────────────────────────────┤
│                       Repository Header                           │
│  owner/repo  ⭐ Star (123)  Fork (45)  👁 Watch                  │
│  ┌────────┬────────┬────────┬────────┬────────┬────────┐        │
│  │ Code   │ Issues │ PRs    │ Actions│ Wiki   │Settings│        │
│  └────────┴────────┴────────┴────────┴────────┴────────┘        │
├──────────────────────────────────────────────────────────────────┤
│                         Main Content                              │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐   │
│  │                                                           │   │
│  │                                                           │   │
│  │                                                           │   │
│  │                                                           │   │
│  │                                                           │   │
│  │                                                           │   │
│  │                                                           │   │
│  └───────────────────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────────────────────┤
│                          Footer                                   │
│  © 2024 GitHome  •  Terms  •  Privacy  •  Status  •  Docs        │
└──────────────────────────────────────────────────────────────────┘
```

### Key Pages

#### Repository Home
```
┌──────────────────────────────────────────────────────────────────┐
│ Branch: main ▾   │ Go to file │ Add file ▾ │ Code ▾            │
├──────────────────────────────────────────────────────────────────┤
│ 📁 src/              commit message                    2 days ago│
│ 📁 tests/            fix: resolve test failures        3 days ago│
│ 📄 README.md         docs: update installation guide   1 week ago│
│ 📄 go.mod            feat: add new dependency          2 weeks   │
├──────────────────────────────────────────────────────────────────┤
│                        README.md                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ # Project Name                                              │  │
│  │                                                             │  │
│  │ Project description goes here...                           │  │
│  │                                                             │  │
│  │ ## Installation                                             │  │
│  │ ```bash                                                     │  │
│  │ go install github.com/example/project                      │  │
│  │ ```                                                         │  │
│  └────────────────────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────────────────────┤
│  About                                        Releases           │
│  Description text here                        v1.2.0  Latest     │
│                                              v1.1.0              │
│  🔗 example.com                               v1.0.0              │
│  📦 12 Releases                                                   │
│  ⭐ 1.2k stars                               Languages            │
│  🍴 234 forks                                ████████░░ Go 80%   │
│  👁 45 watching                              ██░░░░░░░░ JS 20%   │
└──────────────────────────────────────────────────────────────────┘
```

#### Issues List
```
┌──────────────────────────────────────────────────────────────────┐
│ Filters ▾  │ Labels ▾ │ Milestones ▾ │ Assignee ▾ │ New Issue  │
├──────────────────────────────────────────────────────────────────┤
│ ○ 42 Open    ● 156 Closed                                        │
├──────────────────────────────────────────────────────────────────┤
│ □ │ 🔴 Bug - Cannot login with OAuth                    #247     │
│   │ [bug] [auth]           opened 2 hours ago by @user1          │
├──────────────────────────────────────────────────────────────────┤
│ □ │ 🟢 Feature request - Dark mode                      #245     │
│   │ [enhancement]          opened yesterday by @user2            │
├──────────────────────────────────────────────────────────────────┤
│ □ │ 🔵 Docs - Update API documentation                  #244     │
│   │ [documentation]        opened 3 days ago by @user3           │
└──────────────────────────────────────────────────────────────────┘
```

#### Pull Request View
```
┌──────────────────────────────────────────────────────────────────┐
│ ← #248  Add user authentication feature                          │
│ Open  @author wants to merge 5 commits into main from feature/auth│
├────────────┬────────────┬────────────┬──────────────────────────┤
│ Conversation│ Commits(5) │ Files(12) │ Checks(3)               │
├──────────────────────────────────────────────────────────────────┤
│                                              │ Reviewers          │
│ ## Description                               │ ┌───────────────┐  │
│                                              │ │ 👤 @reviewer1 │  │
│ This PR adds complete user authentication   │ │ ✅ Approved    │  │
│ with OAuth support...                        │ └───────────────┘  │
│                                              │                    │
│ - [x] Login/Register                         │ Assignees          │
│ - [x] OAuth providers                        │ @author            │
│ - [x] Session management                     │                    │
│                                              │ Labels             │
│ ──────────────────────────────────          │ [feature] [auth]   │
│                                              │                    │
│ 💬 @reviewer1 commented 2 hours ago          │ Milestone          │
│ ┌─────────────────────────────────────────┐ │ v2.0.0             │
│ │ LGTM! Nice implementation.              │ │                    │
│ └─────────────────────────────────────────┘ │                    │
│                                              │                    │
│ ✅ All checks passed                         │                    │
│ ┌─────────────────────────────────────────┐ │                    │
│ │ [Merge pull request ▾]                  │ │                    │
│ └─────────────────────────────────────────┘ │                    │
└──────────────────────────────────────────────────────────────────┘
```

#### Diff View
```
┌──────────────────────────────────────────────────────────────────┐
│ ◀ Previous │ Next ▶    Viewing 3 of 12 files    Split │ Unified │
├──────────────────────────────────────────────────────────────────┤
│ 📄 src/auth/login.go  +45 -12                         Viewed ☐  │
├──────────────────────────────────────────────────────────────────┤
│    │ @@ -10,12 +10,45 @@ func Login(...)                          │
│  8 │   func Login(ctx context.Context, creds *Credentials) {      │
│  9 │       // Validate credentials                                 │
│ 10 │-      if creds.Username == "" {                              │
│ 11 │-          return ErrInvalidCredentials                       │
│ 12 │-      }                                                       │
│    │+      if err := validateCredentials(creds); err != nil {     │
│    │+          return fmt.Errorf("validation: %w", err)           │
│    │+      }                                                       │
│    │+                                                              │
│    │+      // Check rate limit                                     │
│    │+      if !checkRateLimit(ctx, creds.Username) {              │
│    │+          return ErrRateLimited                               │
│    │+      }                                                       │
│────┼───────────────────────────────────────────────────────────────│
│    │ 💬 + Add comment                                              │
└──────────────────────────────────────────────────────────────────┘
```

---

## Security Considerations

### Authentication

1. **Password Security**
   - Argon2id hashing with memory-hard parameters
   - Minimum password length: 8 characters
   - Check against common password lists
   - Rate limiting on login attempts

2. **Session Management**
   - Secure, HTTP-only cookies
   - Session timeout: 7 days (configurable)
   - Session invalidation on password change
   - Concurrent session limiting

3. **API Tokens**
   - SHA-256 hashed storage
   - Scoped permissions
   - Expiration dates
   - Token rotation support

### Authorization

1. **Repository Access**
   - Owner has full control
   - Collaborators with role-based permissions
   - Team-based access for organizations
   - Visibility: public/private

2. **Permission Levels**
   - `read`: View code, issues, PRs
   - `triage`: Manage issues (no code changes)
   - `write`: Push to non-protected branches
   - `maintain`: Manage repo settings (not destructive)
   - `admin`: Full access including deletion

3. **Protected Branches**
   - Required reviews before merge
   - Required status checks
   - Restrict who can push
   - Restrict force pushes

### Input Validation

1. **Request Validation**
   - Size limits on request bodies
   - Disallow unknown JSON fields
   - Sanitize user-generated content
   - Validate file uploads

2. **Git Operations**
   - Validate ref names
   - Size limits on pushes
   - Reject dangerous file patterns

### CORS & CSRF

1. **CORS Configuration**
   - Restrict allowed origins
   - Credential handling
   - Preflight caching

2. **CSRF Protection**
   - Token-based for web UI
   - SameSite cookie attribute

---

## Configuration

```go
type Config struct {
    // Server
    Addr         string `env:"GITHOME_ADDR" default:":3000"`
    BaseURL      string `env:"GITHOME_BASE_URL" default:"http://localhost:3000"`

    // Database
    DataDir      string `env:"GITHOME_DATA_DIR" default:"./data"`

    // Git
    ReposDir     string `env:"GITHOME_REPOS_DIR" default:"./data/repos"`
    GitBinPath   string `env:"GITHOME_GIT_BIN" default:"git"`

    // Session
    SessionSecret   string        `env:"GITHOME_SESSION_SECRET" required:"true"`
    SessionDuration time.Duration `env:"GITHOME_SESSION_DURATION" default:"168h"`

    // Security
    RateLimitRPS    int  `env:"GITHOME_RATE_LIMIT_RPS" default:"100"`
    MaxUploadSizeMB int  `env:"GITHOME_MAX_UPLOAD_MB" default:"100"`

    // Features
    EnableSignup    bool `env:"GITHOME_ENABLE_SIGNUP" default:"true"`
    EnableWiki      bool `env:"GITHOME_ENABLE_WIKI" default:"false"`
    EnableActions   bool `env:"GITHOME_ENABLE_ACTIONS" default:"false"`

    // Email (optional)
    SMTPHost     string `env:"GITHOME_SMTP_HOST"`
    SMTPPort     int    `env:"GITHOME_SMTP_PORT" default:"587"`
    SMTPUser     string `env:"GITHOME_SMTP_USER"`
    SMTPPassword string `env:"GITHOME_SMTP_PASSWORD"`
    SMTPFrom     string `env:"GITHOME_SMTP_FROM"`

    // OAuth providers (optional)
    GithubClientID     string `env:"GITHOME_GITHUB_CLIENT_ID"`
    GithubClientSecret string `env:"GITHOME_GITHUB_CLIENT_SECRET"`
    GitlabClientID     string `env:"GITHOME_GITLAB_CLIENT_ID"`
    GitlabClientSecret string `env:"GITHOME_GITLAB_CLIENT_SECRET"`
}
```

---

## Implementation Phases

### Phase 1: Foundation
- [ ] Project structure setup
- [ ] Database schema and migrations
- [ ] User authentication (register/login/logout)
- [ ] Session management
- [ ] Basic repository CRUD

### Phase 2: Core Git Features
- [ ] Git repository initialization
- [ ] Branch/tag operations
- [ ] File browsing
- [ ] Commit history
- [ ] Git Smart HTTP protocol

### Phase 3: Collaboration
- [ ] Issues (CRUD, comments, labels)
- [ ] Milestones
- [ ] Pull requests (CRUD, merge)
- [ ] Code review (reviews, comments)
- [ ] Collaborators and permissions

### Phase 4: Organizations
- [ ] Organization CRUD
- [ ] Org membership
- [ ] Teams
- [ ] Team-based repository access

### Phase 5: Notifications & Activity
- [ ] Activity feed
- [ ] Notifications
- [ ] Watch/star repositories
- [ ] Email notifications

### Phase 6: Advanced Features
- [ ] Webhooks
- [ ] Releases
- [ ] Fork repositories
- [ ] Search
- [ ] SSH key authentication
- [ ] API tokens

### Phase 7: Polish
- [ ] Real-time updates (WebSocket)
- [ ] Markdown preview
- [ ] Syntax highlighting
- [ ] Reactions
- [ ] Protected branches
- [ ] Blame view

---

## Technology Stack

| Component | Technology |
|-----------|------------|
| Framework | Mizu (Go) |
| Database | DuckDB |
| Git | go-git / native git |
| Frontend | HTML templates + HTMX + Alpine.js |
| CSS | Tailwind CSS |
| Syntax Highlighting | Prism.js / Chroma |
| Markdown | goldmark |
| WebSocket | gorilla/websocket |

---

## External Dependencies

```go
require (
    github.com/go-mizu/mizu v0.0.0
    github.com/go-git/go-git/v5 v5.12.0
    github.com/yuin/goldmark v1.7.0
    github.com/alecthomas/chroma/v2 v2.14.0
    github.com/marcboeker/go-duckdb v1.8.0
    github.com/gorilla/websocket v1.5.0
    golang.org/x/crypto v0.28.0
    github.com/oklog/ulid/v2 v2.1.0
)
```

---

## Testing Strategy

1. **Unit Tests**
   - Service layer business logic
   - Store layer with in-memory DuckDB
   - Git operations with temporary directories

2. **Integration Tests**
   - API endpoint testing
   - Git protocol testing
   - Authentication flows

3. **E2E Tests**
   - Full user workflows
   - Git clone/push/pull operations

---

## Performance Considerations

1. **Database**
   - Proper indexing on frequently queried columns
   - Pagination for all list operations
   - Caching for frequently accessed data

2. **Git Operations**
   - Lazy loading of repository data
   - Streaming for large file downloads
   - Background processing for diff calculations

3. **Frontend**
   - Progressive loading
   - Virtual scrolling for long lists
   - Lazy loading of images and content

---

## Monitoring & Observability

1. **Logging**
   - Structured logging with slog
   - Request/response logging
   - Error tracking with stack traces

2. **Metrics**
   - Request latency
   - Error rates
   - Active sessions
   - Repository counts

3. **Health Checks**
   - Liveness endpoint
   - Readiness endpoint (database connectivity)

---

## Future Enhancements

1. **GitHub Actions Alternative**
   - CI/CD pipelines
   - Workflow definitions
   - Self-hosted runners

2. **Wiki**
   - Git-backed wiki pages
   - Markdown editor

3. **Projects**
   - Kanban boards
   - Project milestones

4. **Security**
   - Dependency scanning
   - Secret scanning
   - Code scanning

5. **LFS Support**
   - Large file storage
   - Bandwidth management
