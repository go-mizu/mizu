# Social

A general-purpose social network platform with user profiles, feeds, and relationships. Built with Mizu.

## Features

- **User Accounts**: Registration, authentication, profile management
- **Social Graph**: Follow/unfollow, block, mute relationships
- **Posts**: Create, edit, delete posts with visibility controls
- **Timelines**: Home feed, public feed, user timelines, hashtag timelines
- **Interactions**: Like, repost, bookmark, reply
- **Notifications**: Follow, like, repost, mention, reply notifications
- **Search**: Search posts, accounts, and hashtags
- **Trending**: Trending tags and posts
- **Lists**: Curated lists of accounts

## Quick Start

```bash
# Initialize the database
go run ./cmd/social init

# Seed with sample data
go run ./cmd/social seed

# Start the server
go run ./cmd/social serve
```

Open http://localhost:8080 in your browser.

## CLI Commands

```bash
# Start server
social serve [--addr :8080] [--data ./data] [--dev]

# Initialize database
social init [--data ./data]

# Seed sample data
social seed [--data ./data] [--users 10] [--posts 50]
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Create account
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/logout` - Logout

### Accounts
- `GET /api/v1/accounts/verify_credentials` - Get current user
- `PATCH /api/v1/accounts/update_credentials` - Update profile
- `GET /api/v1/accounts/:id` - Get account
- `GET /api/v1/accounts/:id/posts` - Get account posts
- `GET /api/v1/accounts/:id/followers` - Get followers
- `GET /api/v1/accounts/:id/following` - Get following

### Posts
- `POST /api/v1/posts` - Create post
- `GET /api/v1/posts/:id` - Get post
- `PUT /api/v1/posts/:id` - Update post
- `DELETE /api/v1/posts/:id` - Delete post
- `GET /api/v1/posts/:id/context` - Get thread context

### Interactions
- `POST /api/v1/posts/:id/like` - Like post
- `DELETE /api/v1/posts/:id/like` - Unlike post
- `POST /api/v1/posts/:id/repost` - Repost
- `DELETE /api/v1/posts/:id/repost` - Unrepost
- `POST /api/v1/posts/:id/bookmark` - Bookmark
- `DELETE /api/v1/posts/:id/bookmark` - Unbookmark

### Timelines
- `GET /api/v1/timelines/home` - Home timeline
- `GET /api/v1/timelines/public` - Public timeline
- `GET /api/v1/timelines/tag/:tag` - Hashtag timeline

### Notifications
- `GET /api/v1/notifications` - Get notifications
- `POST /api/v1/notifications/clear` - Clear all

### Search
- `GET /api/v1/search?q=query` - Search
- `GET /api/v1/trends/tags` - Trending tags
- `GET /api/v1/trends/posts` - Trending posts

## Project Structure

```
social/
├── cmd/social/          # Main entry point
├── app/web/             # HTTP server and handlers
├── cli/                 # CLI commands
├── feature/             # Business logic
│   ├── accounts/
│   ├── posts/
│   ├── relationships/
│   ├── timelines/
│   ├── interactions/
│   ├── notifications/
│   ├── search/
│   ├── trending/
│   └── lists/
├── store/duckdb/        # Data access layer
├── assets/              # Static files and templates
└── pkg/                 # Utility packages
```

## Tech Stack

- **Framework**: Mizu (Go web framework)
- **Database**: DuckDB
- **Authentication**: Session-based with bcrypt
- **Templates**: Go html/template
