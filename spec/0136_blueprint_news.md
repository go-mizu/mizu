# News Blueprint - Link Aggregation System

**Version:** 1.0
**Status:** Implementation
**Author:** Mizu Team
**Created:** 2025-12-23

## 1. Overview

News is a lightweight link aggregation and discussion platform inspired by Hacker News and Lobsters. It provides a minimal, fast, content-focused interface for sharing and discussing links with nested comments.

### 1.1 Goals

1. **Simplicity First** - Minimal UI, fast loading, content-focused
2. **Link Aggregation** - Primary focus on URL submissions with discussion
3. **Quality Ranking** - HN-style time-decay algorithm for content discovery
4. **Nested Comments** - Threaded discussions with proper indentation
5. **Tag-based Organization** - Lobsters-style tags instead of subreddits
6. **Readonly Emphasis** - Optimized for browsing, minimal write friction

### 1.2 Non-Goals

1. User profiles with avatars/banners
2. Private messaging
3. Rich media embeds
4. Real-time notifications
5. Complex moderation (keep simple)

## 2. System Architecture

### 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Client                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                    Browser (SSR)                      │   │
│  └────────────────────────┬─────────────────────────────┘   │
└───────────────────────────┼─────────────────────────────────┘
                            │
┌───────────────────────────┼─────────────────────────────────┐
│                           ▼                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   HTTP Server (Mizu)                   │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │  Middleware │  │   Handlers   │  │   Templates  │  │  │
│  │  │  (Auth)     │  │  (SSR+API)   │  │   (HTML)     │  │  │
│  │  └─────────────┘  └──────────────┘  └──────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                              │                               │
│  ┌───────────────────────────┼───────────────────────────┐  │
│  │                   Feature Layer                        │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐      │  │
│  │  │ Users   │ │ Stories │ │Comments │ │  Votes  │      │  │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘      │  │
│  │  ┌─────────┐ ┌─────────┐                              │  │
│  │  │  Tags   │ │ Ranking │                              │  │
│  │  └─────────┘ └─────────┘                              │  │
│  └───────────────────────────────────────────────────────┘  │
│                              │                               │
│  ┌───────────────────────────┼───────────────────────────┐  │
│  │                   Data Layer                           │  │
│  │  ┌─────────────────────────────────────────────────┐  │  │
│  │  │                   DuckDB                         │  │  │
│  │  │  (Users, Stories, Comments, Votes, Tags)        │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│                        News Server                           │
└──────────────────────────────────────────────────────────────┘
```

### 2.2 Directory Structure

```
news/
├── cmd/news/
│   └── main.go              # Entry point
├── cli/
│   ├── root.go              # Root command (Fang)
│   ├── serve.go             # Serve command
│   ├── init.go              # Init command
│   ├── seed.go              # Seed command (HN import)
│   ├── user.go              # User management
│   └── ui.go                # CLI UI helpers
├── app/web/
│   ├── server.go            # Server orchestration
│   ├── routes.go            # Route definitions
│   ├── middleware.go        # Auth middleware
│   ├── context.go           # Request context helpers
│   └── handler/
│       ├── auth.go          # Login/Register/Logout
│       ├── story.go         # Story handlers
│       ├── comment.go       # Comment handlers
│       ├── vote.go          # Vote handlers
│       ├── user.go          # User profile handlers
│       ├── page.go          # HTML page handlers
│       └── response.go      # Response helpers
├── feature/
│   ├── users/
│   │   ├── api.go           # Types and interface
│   │   └── service.go       # Business logic
│   ├── stories/
│   │   ├── api.go
│   │   └── service.go
│   ├── comments/
│   │   ├── api.go
│   │   └── service.go
│   ├── votes/
│   │   ├── api.go
│   │   └── service.go
│   └── tags/
│       ├── api.go
│       └── service.go
├── store/duckdb/
│   ├── store.go             # Core store
│   ├── schema.sql           # Database schema
│   ├── users_store.go
│   ├── stories_store.go
│   ├── comments_store.go
│   ├── votes_store.go
│   └── tags_store.go
├── assets/
│   ├── embed.go             # Embed directive
│   ├── static/
│   │   └── css/
│   │       └── news.css     # Minimal HN-style CSS
│   └── views/
│       ├── layout.html      # Base layout
│       ├── home.html        # Story list
│       ├── newest.html      # New stories
│       ├── story.html       # Story + comments
│       ├── submit.html      # Submit form
│       ├── user.html        # User profile
│       ├── login.html       # Login form
│       └── components/
│           ├── story_row.html
│           ├── comment.html
│           └── nav.html
├── pkg/
│   ├── ulid/
│   │   └── ulid.go          # ULID generation
│   ├── password/
│   │   └── password.go      # Password hashing
│   └── ranking/
│       └── ranking.go       # HN ranking algorithm
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 3. Data Models

### 3.1 User

```go
package users

import "time"

type User struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    Email        string    `json:"-"`
    PasswordHash string    `json:"-"`
    About        string    `json:"about,omitempty"`
    Karma        int64     `json:"karma"`
    IsAdmin      bool      `json:"is_admin,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
}

type CreateUserInput struct {
    Username string
    Email    string
    Password string
}

type UpdateUserInput struct {
    About *string
}
```

### 3.2 Story

```go
package stories

import "time"

type Story struct {
    ID           string    `json:"id"`
    AuthorID     string    `json:"author_id"`
    Title        string    `json:"title"`
    URL          string    `json:"url,omitempty"`
    Domain       string    `json:"domain,omitempty"`
    Text         string    `json:"text,omitempty"`
    TextHTML     string    `json:"text_html,omitempty"`
    Score        int64     `json:"score"`
    CommentCount int64     `json:"comment_count"`
    HotScore     float64   `json:"-"`
    IsRemoved    bool      `json:"-"`
    CreatedAt    time.Time `json:"created_at"`

    // Joined fields
    Author       *users.User `json:"author,omitempty"`
    Tags         []string    `json:"tags,omitempty"`
    UserVote     int         `json:"user_vote,omitempty"`
}

type StoryType string

const (
    StoryTypeLink StoryType = "link"
    StoryTypeText StoryType = "text"  // Ask HN style
)

type CreateStoryInput struct {
    Title string
    URL   string // Optional - if empty, it's a text post
    Text  string // Optional - body for text posts
    Tags  []string
}

type ListStoriesInput struct {
    Sort   string // "hot", "new", "top"
    Tag    string // Filter by tag
    Limit  int
    Offset int
}
```

### 3.3 Comment

```go
package comments

import "time"

type Comment struct {
    ID         string    `json:"id"`
    StoryID    string    `json:"story_id"`
    ParentID   string    `json:"parent_id,omitempty"`
    AuthorID   string    `json:"author_id"`
    Text       string    `json:"text"`
    TextHTML   string    `json:"text_html"`
    Score      int64     `json:"score"`
    Depth      int       `json:"depth"`
    Path       string    `json:"-"`
    ChildCount int64     `json:"child_count"`
    IsRemoved  bool      `json:"-"`
    CreatedAt  time.Time `json:"created_at"`

    // Joined fields
    Author    *users.User `json:"author,omitempty"`
    UserVote  int         `json:"user_vote,omitempty"`
    Children  []*Comment  `json:"children,omitempty"`
}

type CreateCommentInput struct {
    StoryID  string
    ParentID string // Optional
    Text     string
}
```

### 3.4 Vote

```go
package votes

import "time"

type Vote struct {
    ID         string    `json:"id"`
    UserID     string    `json:"user_id"`
    TargetType string    `json:"target_type"` // "story" or "comment"
    TargetID   string    `json:"target_id"`
    Value      int       `json:"value"` // 1 for upvote, -1 for downvote (future)
    CreatedAt  time.Time `json:"created_at"`
}

type VoteInput struct {
    TargetType string
    TargetID   string
    Value      int
}
```

### 3.5 Tag

```go
package tags

type Tag struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    Color       string `json:"color,omitempty"`
    StoryCount  int64  `json:"story_count"`
}
```

## 4. Database Schema

```sql
-- News Schema

-- Users
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR PRIMARY KEY,
    username VARCHAR UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    about TEXT,
    karma BIGINT DEFAULT 0,
    is_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(LOWER(username));
CREATE INDEX IF NOT EXISTS idx_users_karma ON users(karma DESC);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    token VARCHAR UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    id VARCHAR PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    description TEXT,
    color VARCHAR DEFAULT '#666666',
    story_count BIGINT DEFAULT 0
);

-- Stories
CREATE TABLE IF NOT EXISTS stories (
    id VARCHAR PRIMARY KEY,
    author_id VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    url VARCHAR,
    domain VARCHAR,
    text TEXT,
    text_html TEXT,
    score BIGINT DEFAULT 1,
    comment_count BIGINT DEFAULT 0,
    hot_score DOUBLE DEFAULT 0,
    is_removed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_stories_hot ON stories(hot_score DESC) WHERE is_removed = FALSE;
CREATE INDEX IF NOT EXISTS idx_stories_new ON stories(created_at DESC) WHERE is_removed = FALSE;
CREATE INDEX IF NOT EXISTS idx_stories_top ON stories(score DESC) WHERE is_removed = FALSE;
CREATE INDEX IF NOT EXISTS idx_stories_author ON stories(author_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_stories_domain ON stories(domain);

-- Story tags (many-to-many)
CREATE TABLE IF NOT EXISTS story_tags (
    story_id VARCHAR NOT NULL,
    tag_id VARCHAR NOT NULL,
    PRIMARY KEY (story_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_story_tags_tag ON story_tags(tag_id);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR PRIMARY KEY,
    story_id VARCHAR NOT NULL,
    parent_id VARCHAR,
    author_id VARCHAR NOT NULL,
    text TEXT NOT NULL,
    text_html TEXT,
    score BIGINT DEFAULT 1,
    depth INT DEFAULT 0,
    path VARCHAR NOT NULL,
    child_count BIGINT DEFAULT 0,
    is_removed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_story ON comments(story_id, path);
CREATE INDEX IF NOT EXISTS idx_comments_author ON comments(author_id, created_at DESC);

-- Votes
CREATE TABLE IF NOT EXISTS votes (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    value INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_unique ON votes(user_id, target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_votes_target ON votes(target_type, target_id);

-- Seed mappings (for HN import)
CREATE TABLE IF NOT EXISTS seed_mappings (
    source VARCHAR NOT NULL,
    entity_type VARCHAR NOT NULL,
    external_id VARCHAR NOT NULL,
    local_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source, entity_type, external_id)
);
```

## 5. Ranking Algorithm

### 5.1 HN-Style Hot Score

```go
package ranking

import (
    "math"
    "time"
)

const (
    gravity   = 1.8
    hourDecay = 2.0
)

// HotScore calculates HN-style ranking score
// Formula: score / (age + 2) ^ gravity
func HotScore(points int64, createdAt time.Time) float64 {
    age := time.Since(createdAt).Hours()
    return float64(points) / math.Pow(age+hourDecay, gravity)
}

// ShouldRecalculate determines if hot score needs update
// Recalculate if:
// - Score changed
// - More than 10 minutes since last calculation (for new items)
// - More than 1 hour since last calculation (for older items)
func ShouldRecalculate(score int64, lastScore int64, createdAt time.Time, lastCalc time.Time) bool {
    if score != lastScore {
        return true
    }
    age := time.Since(createdAt)
    if age < 24*time.Hour {
        return time.Since(lastCalc) > 10*time.Minute
    }
    return time.Since(lastCalc) > time.Hour
}
```

### 5.2 Background Recalculation

```go
// RecalculateHotScores updates hot scores for active stories
// Run every 5 minutes for recent stories, hourly for older
func (s *Store) RecalculateHotScores(ctx context.Context) error {
    // Update stories from last 48 hours
    query := `
        UPDATE stories
        SET hot_score = score / POWER(
            (EXTRACT(EPOCH FROM NOW() - created_at) / 3600) + 2,
            1.8
        )
        WHERE created_at > NOW() - INTERVAL '48 hours'
        AND is_removed = FALSE
    `
    _, err := s.db.ExecContext(ctx, query)
    return err
}
```

## 6. API Routes

### 6.1 HTML Pages (SSR)

| Route | Handler | Description |
|-------|---------|-------------|
| `GET /` | `PageHome` | Front page (hot stories) |
| `GET /newest` | `PageNewest` | New stories |
| `GET /top` | `PageTop` | Top stories |
| `GET /story/{id}` | `PageStory` | Story with comments |
| `GET /submit` | `PageSubmit` | Submit form (auth required) |
| `GET /user/{username}` | `PageUser` | User profile |
| `GET /login` | `PageLogin` | Login form |
| `GET /register` | `PageRegister` | Register form |
| `GET /tag/{name}` | `PageTag` | Stories by tag |

### 6.2 API Endpoints

| Route | Handler | Description |
|-------|---------|-------------|
| `POST /api/auth/register` | `APIRegister` | Create account |
| `POST /api/auth/login` | `APILogin` | Login |
| `POST /api/auth/logout` | `APILogout` | Logout |
| `POST /api/stories` | `APICreateStory` | Submit story |
| `POST /api/stories/{id}/vote` | `APIVoteStory` | Upvote story |
| `DELETE /api/stories/{id}/vote` | `APIUnvoteStory` | Remove vote |
| `POST /api/comments` | `APICreateComment` | Add comment |
| `POST /api/comments/{id}/vote` | `APIVoteComment` | Upvote comment |
| `DELETE /api/comments/{id}/vote` | `APIUnvoteComment` | Remove vote |

### 6.3 Route Implementation

```go
package web

func (s *Server) routes() {
    // Static assets
    s.app.Static("/static", s.assets.Static)

    // Public pages
    s.app.GET("/", s.handleHome)
    s.app.GET("/newest", s.handleNewest)
    s.app.GET("/top", s.handleTop)
    s.app.GET("/story/{id}", s.handleStory)
    s.app.GET("/user/{username}", s.handleUser)
    s.app.GET("/tag/{name}", s.handleTag)
    s.app.GET("/login", s.handleLoginPage)
    s.app.GET("/register", s.handleRegisterPage)

    // Auth required pages
    s.app.GET("/submit", s.requireAuth(s.handleSubmitPage))

    // Auth API
    s.app.POST("/api/auth/register", s.handleRegister)
    s.app.POST("/api/auth/login", s.handleLogin)
    s.app.POST("/api/auth/logout", s.handleLogout)

    // Protected API
    s.app.POST("/api/stories", s.requireAuth(s.handleCreateStory))
    s.app.POST("/api/stories/{id}/vote", s.requireAuth(s.handleVoteStory))
    s.app.DELETE("/api/stories/{id}/vote", s.requireAuth(s.handleUnvoteStory))
    s.app.POST("/api/comments", s.requireAuth(s.handleCreateComment))
    s.app.POST("/api/comments/{id}/vote", s.requireAuth(s.handleVoteComment))
    s.app.DELETE("/api/comments/{id}/vote", s.requireAuth(s.handleUnvoteComment))
}
```

## 7. HTML Templates

### 7.1 Layout (HN-Style)

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} | News</title>
    <link rel="stylesheet" href="/static/css/news.css">
</head>
<body>
    <header>
        <nav>
            <a href="/" class="logo"><b>News</b></a>
            <a href="/newest">new</a>
            <a href="/top">top</a>
            <a href="/submit">submit</a>
            {{if .User}}
                <span class="right">
                    <a href="/user/{{.User.Username}}">{{.User.Username}}</a> ({{.User.Karma}}) |
                    <a href="/logout">logout</a>
                </span>
            {{else}}
                <span class="right">
                    <a href="/login">login</a>
                </span>
            {{end}}
        </nav>
    </header>
    <main>
        {{template "content" .}}
    </main>
    <footer>
        <hr>
        <p>
            <a href="/guidelines">Guidelines</a> |
            <a href="/about">About</a>
        </p>
    </footer>
</body>
</html>
```

### 7.2 Story Row Component

```html
{{define "story_row"}}
<tr class="story" id="story-{{.Story.ID}}">
    <td class="rank">{{.Rank}}.</td>
    <td class="vote">
        {{if .User}}
            <a href="#" onclick="vote('story', '{{.Story.ID}}'); return false;"
               class="vote-btn {{if eq .Story.UserVote 1}}voted{{end}}">▲</a>
        {{end}}
    </td>
    <td class="title">
        {{if .Story.URL}}
            <a href="{{.Story.URL}}" rel="nofollow">{{.Story.Title}}</a>
            <span class="domain">({{.Story.Domain}})</span>
        {{else}}
            <a href="/story/{{.Story.ID}}">{{.Story.Title}}</a>
        {{end}}
    </td>
</tr>
<tr class="meta">
    <td colspan="2"></td>
    <td>
        <span class="score">{{.Story.Score}} points</span> by
        <a href="/user/{{.Story.Author.Username}}">{{.Story.Author.Username}}</a>
        <span class="age">{{.Story.CreatedAt | timeago}}</span> |
        <a href="/story/{{.Story.ID}}">{{.Story.CommentCount}} comments</a>
        {{range .Story.Tags}}
            <a href="/tag/{{.}}" class="tag">{{.}}</a>
        {{end}}
    </td>
</tr>
{{end}}
```

### 7.3 Comment Component

```html
{{define "comment"}}
<div class="comment" id="comment-{{.Comment.ID}}" style="margin-left: {{mul .Comment.Depth 20}}px">
    <div class="comment-head">
        {{if $.User}}
            <a href="#" onclick="vote('comment', '{{.Comment.ID}}'); return false;"
               class="vote-btn {{if eq .Comment.UserVote 1}}voted{{end}}">▲</a>
        {{end}}
        <a href="/user/{{.Comment.Author.Username}}">{{.Comment.Author.Username}}</a>
        <span class="age">{{.Comment.CreatedAt | timeago}}</span>
    </div>
    <div class="comment-body">
        {{.Comment.TextHTML | safe}}
    </div>
    <div class="comment-links">
        <a href="/story/{{.Comment.StoryID}}#comment-{{.Comment.ID}}">link</a>
        {{if $.User}}
            | <a href="#" onclick="reply('{{.Comment.ID}}'); return false;">reply</a>
        {{end}}
    </div>
    {{range .Comment.Children}}
        {{template "comment" dict "Comment" . "User" $.User}}
    {{end}}
</div>
{{end}}
```

### 7.4 Minimal CSS

```css
/* news.css - HN-inspired minimal styles */
:root {
    --bg: #f6f6ef;
    --fg: #000;
    --link: #000;
    --meta: #828282;
    --accent: #ff6600;
}

* { box-sizing: border-box; margin: 0; padding: 0; }

body {
    font-family: Verdana, Geneva, sans-serif;
    font-size: 10pt;
    background: var(--bg);
    color: var(--fg);
    max-width: 85%;
    margin: 0 auto;
    padding: 8px;
}

header {
    background: var(--accent);
    padding: 2px 4px;
    margin-bottom: 10px;
}

header nav {
    display: flex;
    gap: 8px;
    align-items: center;
}

header nav a {
    color: var(--fg);
    text-decoration: none;
}

header nav .logo {
    margin-right: 8px;
}

header nav .right {
    margin-left: auto;
}

main {
    padding: 10px 0;
}

table.stories {
    border-collapse: collapse;
    width: 100%;
}

table.stories td {
    padding: 2px 4px;
    vertical-align: top;
}

.rank {
    color: var(--meta);
    width: 20px;
    text-align: right;
}

.vote {
    width: 10px;
    text-align: center;
}

.vote-btn {
    color: var(--meta);
    text-decoration: none;
    cursor: pointer;
}

.vote-btn.voted {
    color: var(--accent);
}

.title a {
    color: var(--link);
    text-decoration: none;
}

.domain {
    color: var(--meta);
    font-size: 8pt;
}

.meta {
    color: var(--meta);
    font-size: 8pt;
}

.meta td {
    padding-bottom: 8px;
}

.tag {
    background: #eee;
    padding: 1px 4px;
    border-radius: 2px;
    font-size: 8pt;
    margin-left: 4px;
}

.comment {
    margin-bottom: 15px;
}

.comment-head {
    color: var(--meta);
    font-size: 8pt;
    margin-bottom: 4px;
}

.comment-body {
    margin: 8px 0;
}

.comment-body p {
    margin: 8px 0;
}

.comment-links {
    color: var(--meta);
    font-size: 8pt;
}

.comment-links a {
    color: var(--meta);
    text-decoration: underline;
}

footer {
    color: var(--meta);
    font-size: 8pt;
    text-align: center;
    padding: 20px 0;
}

footer a {
    color: var(--meta);
}

/* Forms */
input[type="text"],
input[type="email"],
input[type="password"],
input[type="url"],
textarea {
    font-family: inherit;
    font-size: inherit;
    padding: 4px;
    width: 400px;
    max-width: 100%;
}

textarea {
    height: 100px;
}

button, input[type="submit"] {
    font-family: inherit;
    font-size: inherit;
    padding: 4px 12px;
    cursor: pointer;
}

.form-row {
    margin-bottom: 10px;
}

.form-row label {
    display: inline-block;
    width: 80px;
}

.error {
    color: #c00;
    margin: 10px 0;
}
```

## 8. CLI Commands

### 8.1 Command Structure

```
news
├── serve              # Start web server
│   ├── --addr         # Listen address (default: :8080)
│   ├── --data         # Data directory (default: ./data)
│   └── --dev          # Development mode
├── init               # Initialize database
│   └── --data         # Data directory
├── seed               # Seed data
│   └── hn             # Import from Hacker News
│       ├── --feed     # Feed type (top, new, best, ask, show)
│       ├── --limit    # Number of stories
│       └── --with-comments
├── user               # User management
│   ├── create         # Create user
│   ├── list           # List users
│   └── admin          # Toggle admin
└── stats              # Show statistics
```

### 8.2 Seed from Hacker News

```go
package cli

func NewSeedHN() *cobra.Command {
    var (
        feed         string
        limit        int
        withComments bool
    )

    cmd := &cobra.Command{
        Use:   "hn",
        Short: "Seed from Hacker News",
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. Fetch story IDs from HN API
            // 2. Fetch each story
            // 3. Map HN user to local user (create if needed)
            // 4. Create story with seed_mapping
            // 5. Optionally fetch and import comments
            return nil
        },
    }

    cmd.Flags().StringVar(&feed, "feed", "top", "Feed: top, new, best, ask, show")
    cmd.Flags().IntVar(&limit, "limit", 30, "Number of stories")
    cmd.Flags().BoolVar(&withComments, "with-comments", false, "Also fetch comments")

    return cmd
}
```

## 9. Implementation Plan

### Phase 1: Foundation
1. Create `blueprints/news/` directory structure
2. Initialize Go module
3. Create Makefile
4. Implement pkg/ulid, pkg/password, pkg/ranking

### Phase 2: Data Layer
1. Create schema.sql
2. Implement store/duckdb/store.go
3. Implement users_store.go
4. Implement stories_store.go
5. Implement comments_store.go
6. Implement votes_store.go
7. Implement tags_store.go

### Phase 3: Feature Layer
1. Implement feature/users (api.go, service.go)
2. Implement feature/stories (api.go, service.go)
3. Implement feature/comments (api.go, service.go)
4. Implement feature/votes (api.go, service.go)
5. Implement feature/tags (api.go, service.go)

### Phase 4: CLI
1. Implement cli/root.go with Fang
2. Implement cli/ui.go
3. Implement cli/serve.go
4. Implement cli/init.go
5. Implement cli/seed.go (with HN support)
6. Implement cli/user.go
7. Implement cli/stats.go

### Phase 5: Web Layer
1. Create assets/embed.go
2. Create CSS (news.css)
3. Create HTML templates
4. Implement app/web/server.go
5. Implement app/web/routes.go
6. Implement app/web/middleware.go
7. Implement handlers (auth, story, comment, vote, user, page)

### Phase 6: Polish
1. Add hot score recalculation
2. Add pagination
3. Add tag filtering
4. Test full workflow

## 10. File Summary

| File | Description |
|------|-------------|
| `cmd/news/main.go` | Entry point |
| `cli/root.go` | Root command with Fang |
| `cli/serve.go` | Serve command |
| `cli/init.go` | Init command |
| `cli/seed.go` | Seed command |
| `cli/ui.go` | CLI UI helpers |
| `store/duckdb/store.go` | Store initialization |
| `store/duckdb/schema.sql` | Database schema |
| `store/duckdb/users_store.go` | User CRUD |
| `store/duckdb/stories_store.go` | Story CRUD + listing |
| `store/duckdb/comments_store.go` | Comment CRUD + tree |
| `store/duckdb/votes_store.go` | Vote CRUD |
| `store/duckdb/tags_store.go` | Tag CRUD |
| `feature/users/api.go` | User types |
| `feature/users/service.go` | User business logic |
| `feature/stories/api.go` | Story types |
| `feature/stories/service.go` | Story business logic |
| `feature/comments/api.go` | Comment types |
| `feature/comments/service.go` | Comment business logic |
| `feature/votes/api.go` | Vote types |
| `feature/votes/service.go` | Vote business logic |
| `feature/tags/api.go` | Tag types |
| `feature/tags/service.go` | Tag business logic |
| `pkg/ranking/ranking.go` | HN ranking algorithm |
| `app/web/server.go` | Web server |
| `app/web/routes.go` | Route setup |
| `app/web/middleware.go` | Auth middleware |
| `app/web/handler/auth.go` | Auth handlers |
| `app/web/handler/story.go` | Story handlers |
| `app/web/handler/comment.go` | Comment handlers |
| `app/web/handler/vote.go` | Vote handlers |
| `app/web/handler/page.go` | HTML page handlers |
| `assets/static/css/news.css` | Minimal CSS |
| `assets/views/*.html` | HTML templates |

## 11. Success Criteria

- [ ] Can run `news init` to create database
- [ ] Can run `news seed hn --limit 30` to import HN stories
- [ ] Can run `news serve` to start server
- [ ] Front page shows stories ranked by hot score
- [ ] Can browse newest and top stories
- [ ] Can view story with nested comments
- [ ] Can register and login
- [ ] Logged-in users can submit stories
- [ ] Logged-in users can comment
- [ ] Logged-in users can upvote
- [ ] User profiles show karma and submissions
- [ ] Tags filter stories
- [ ] Hot score updates over time
