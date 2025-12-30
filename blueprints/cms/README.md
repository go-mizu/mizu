# Mizu CMS

A modern, headless content management system built with Go. Features a RESTful API, WordPress-compatible API, and a WordPress-like admin interface.

## Features

- **Headless CMS** - RESTful API for content delivery
- **WordPress API Compatible** - Drop-in replacement for WordPress REST API
- **Admin Interface** - WordPress-like dashboard for content management
- **DuckDB Storage** - Embedded database, no separate server required
- **Single Binary** - Deploy anywhere with one executable
- **Content Types** - Posts, Pages, Categories, Tags, Media, Comments, Menus

## Quick Start

```bash
# Build the CMS binary
make build

# Initialize the database
cms init

# Seed demo data (optional)
cms seed

# Start the server
cms serve
```

Open your browser:
- **Admin UI**: http://localhost:8080/wp-admin/
- **REST API**: http://localhost:8080/api/v1/
- **WordPress API**: http://localhost:8080/wp-json/wp/v2/

**Demo credentials** (after seeding): `admin@example.com` / `password123`

## Admin User & Authentication

### Default Admin User

After running `cms seed`, an admin user is created:

| Field | Value |
|-------|-------|
| Email | `admin@example.com` |
| Password | `password123` |
| Role | `admin` |

### Creating Users Manually

If you don't want to use seeded data, you can create users via the REST API:

```bash
# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your-secure-password",
    "name": "Admin User"
  }'
```

### Admin Login Flow

1. Navigate to http://localhost:8080/wp-admin/ or http://localhost:8080/wp-login.php
2. Enter your email and password
3. Check "Remember Me" for persistent sessions (14 days)
4. Click "Log In" to access the dashboard

### Logout

- Click "Log Out" in the admin bar, or
- Navigate to http://localhost:8080/wp-logout.php

### URL Patterns

The CMS supports both WordPress-style and clean URLs:

| WordPress Style | Clean URL |
|-----------------|-----------|
| `/wp-login.php` | `/wp-admin/login` |
| `/wp-logout.php` | `/wp-admin/logout` |
| `/wp-admin/edit.php` | `/wp-admin/posts` |
| `/wp-admin/post-new.php` | `/wp-admin/posts/new` |

## Installation

### Prerequisites

- Go 1.22+ with CGO enabled
- GCC or compatible C compiler (for DuckDB)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/go-mizu/mizu
cd mizu/blueprints/cms

# Build the binary (installs to ~/bin/cms)
make build

# Or build with custom output
CGO_ENABLED=1 go build -o cms ./cmd/cms
```

### Verify Installation

```bash
cms --version
# cms version v0.1.0 (abc1234) built at 2024-01-15T10:30:00Z
```

## CLI Commands

```bash
# Initialize database schema
cms init [--data DIR]

# Seed demo data
cms seed [--data DIR]

# Start the server
cms serve [--addr :8080] [--data DIR] [--dev]

# Show help
cms --help
```

### Configuration Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--data` | `~/data/blueprint/cms` | Data directory for database and uploads |
| `--addr` | `:8080` | Server listen address |
| `--dev` | `false` | Enable development mode |

## REST API

### Authentication

```bash
# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret123", "display_name": "John"}'

# Login (returns session cookie)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret123"}' \
  -c cookies.txt

# Get current user
curl http://localhost:8080/api/v1/auth/me -b cookies.txt

# Logout
curl -X POST http://localhost:8080/api/v1/auth/logout -b cookies.txt
```

### Posts

```bash
# List posts
curl http://localhost:8080/api/v1/posts

# List with filters
curl "http://localhost:8080/api/v1/posts?page=1&per_page=10&category_id=xxx&search=golang"

# Get single post
curl http://localhost:8080/api/v1/posts/{id}

# Get by slug
curl http://localhost:8080/api/v1/posts/by-slug/my-first-post

# Create post (auth required)
curl -X POST http://localhost:8080/api/v1/posts \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "title": "My Post",
    "content": "<p>Hello World</p>",
    "excerpt": "A brief intro",
    "status": "published",
    "category_ids": ["cat-id-1"],
    "tag_ids": ["tag-id-1"]
  }'

# Update post
curl -X PUT http://localhost:8080/api/v1/posts/{id} \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"title": "Updated Title"}'

# Delete post
curl -X DELETE http://localhost:8080/api/v1/posts/{id} -b cookies.txt

# Publish/Unpublish
curl -X POST http://localhost:8080/api/v1/posts/{id}/publish -b cookies.txt
curl -X POST http://localhost:8080/api/v1/posts/{id}/unpublish -b cookies.txt
```

### Pages

```bash
# List pages
curl http://localhost:8080/api/v1/pages

# Get hierarchical tree
curl http://localhost:8080/api/v1/pages/tree

# Get by slug
curl http://localhost:8080/api/v1/pages/by-slug/about

# Create page (auth required)
curl -X POST http://localhost:8080/api/v1/pages \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "title": "About Us",
    "content": "<p>About our company</p>",
    "template": "default"
  }'
```

### Categories & Tags

```bash
# Categories
curl http://localhost:8080/api/v1/categories
curl http://localhost:8080/api/v1/categories/tree  # Hierarchical
curl -X POST http://localhost:8080/api/v1/categories \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name": "Technology", "description": "Tech posts"}'

# Tags
curl http://localhost:8080/api/v1/tags
curl -X POST http://localhost:8080/api/v1/tags \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name": "golang", "description": "Go language"}'
```

### Media

```bash
# Upload file
curl -X POST http://localhost:8080/api/v1/media \
  -b cookies.txt \
  -F "file=@image.jpg" \
  -F "title=My Image" \
  -F "alt_text=Description"

# List media
curl http://localhost:8080/api/v1/media -b cookies.txt

# Get media
curl http://localhost:8080/api/v1/media/{id}

# Update metadata
curl -X PUT http://localhost:8080/api/v1/media/{id} \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"title": "New Title", "alt_text": "New alt text"}'
```

### Comments

```bash
# Get comments for a post (public)
curl http://localhost:8080/api/v1/comments/for-post/{postID}

# Create comment on post (public)
curl -X POST http://localhost:8080/api/v1/comments/for-post/{postID} \
  -H "Content-Type: application/json" \
  -d '{
    "author_name": "John",
    "author_email": "john@example.com",
    "content": "Great post!"
  }'

# Moderate comments (auth required)
curl -X POST http://localhost:8080/api/v1/comments/approve/{id} -b cookies.txt
curl -X POST http://localhost:8080/api/v1/comments/spam/{id} -b cookies.txt
```

### Menus

```bash
# List menus
curl http://localhost:8080/api/v1/menus

# Get by location
curl http://localhost:8080/api/v1/menus/by-location/header

# Create menu (auth required)
curl -X POST http://localhost:8080/api/v1/menus \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name": "Main Menu", "location": "header"}'

# Add menu item
curl -X POST http://localhost:8080/api/v1/menus/{id}/items \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "title": "Home",
    "url": "/",
    "position": 0
  }'
```

### Settings

```bash
# Get public settings
curl http://localhost:8080/api/v1/settings/public

# Get all settings (auth required)
curl http://localhost:8080/api/v1/settings -b cookies.txt

# Update settings
curl -X PUT http://localhost:8080/api/v1/settings \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "site_title": "My Blog",
    "site_description": "A blog about things"
  }'
```

### API Response Format

**Success Response**
```json
{
  "data": { ... },
  "meta": {
    "page": 1,
    "per_page": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

**Error Response**
```json
{
  "error": {
    "code": "not_found",
    "message": "Post not found"
  }
}
```

## WordPress API

The CMS provides WordPress REST API compatibility at `/wp-json/wp/v2/`.

```bash
# API Discovery
curl http://localhost:8080/wp-json/

# Posts (WordPress format)
curl http://localhost:8080/wp-json/wp/v2/posts
curl http://localhost:8080/wp-json/wp/v2/posts?per_page=5&page=1

# Pages
curl http://localhost:8080/wp-json/wp/v2/pages

# Categories
curl http://localhost:8080/wp-json/wp/v2/categories

# Tags
curl http://localhost:8080/wp-json/wp/v2/tags

# Media
curl http://localhost:8080/wp-json/wp/v2/media

# Users
curl http://localhost:8080/wp-json/wp/v2/users
```

This allows integration with WordPress themes, plugins, and frontend frameworks designed for WordPress.

## Admin Interface

Access the WordPress-like admin interface at `/wp-admin/`.

### Available Pages

| Path | Description |
|------|-------------|
| `/wp-admin/` | Dashboard |
| `/wp-admin/posts` | Manage posts |
| `/wp-admin/posts/new` | Create new post |
| `/wp-admin/posts/{id}/edit` | Edit post |
| `/wp-admin/pages` | Manage pages |
| `/wp-admin/media` | Media library |
| `/wp-admin/comments` | Comment moderation |
| `/wp-admin/categories` | Category management |
| `/wp-admin/tags` | Tag management |
| `/wp-admin/menus` | Menu builder |
| `/wp-admin/users` | User management |
| `/wp-admin/settings` | Site settings |
| `/wp-admin/login` | Login page |

## Project Structure

```
cms/
├── cmd/cms/main.go       # Entry point
├── cli/                  # CLI commands (serve, init, seed)
├── app/web/
│   ├── server.go         # HTTP server setup
│   └── handler/
│       ├── rest/         # REST API handlers
│       ├── wpapi/        # WordPress API handlers
│       └── wpadmin/      # Admin UI handlers
├── feature/              # Business logic services
│   ├── users/            # User management
│   ├── posts/            # Post management
│   ├── pages/            # Page management
│   ├── categories/       # Category taxonomy
│   ├── tags/             # Tag taxonomy
│   ├── media/            # Media library
│   ├── comments/         # Comment system
│   ├── settings/         # Site settings
│   └── menus/            # Navigation menus
├── store/duckdb/         # Data access layer
│   ├── schema.sql        # Database schema
│   └── *_store.go        # Feature stores
├── assets/               # Embedded assets
│   ├── static/           # CSS, JS
│   └── views/            # HTML templates
├── pkg/                  # Utilities
│   ├── ulid/             # ID generation
│   ├── slug/             # URL slug generation
│   └── password/         # Password hashing
└── e2e/                  # End-to-end tests
```

## Database Schema

The CMS uses DuckDB with the following main tables:

| Table | Description |
|-------|-------------|
| `users` | User accounts with roles |
| `sessions` | Authentication sessions |
| `posts` | Blog posts |
| `pages` | Static pages (hierarchical) |
| `categories` | Hierarchical taxonomy |
| `tags` | Flat taxonomy |
| `post_categories` | Post-category relationships |
| `post_tags` | Post-tag relationships |
| `media` | Uploaded files |
| `comments` | Post comments (threaded) |
| `menus` | Navigation menus |
| `menu_items` | Menu items |
| `settings` | Key-value settings |
| `revisions` | Content version history |

## Development

### Running Locally

```bash
# Run without building
make run

# Run with arguments
make run ARGS="serve --addr :3000 --dev"

# Run tests
make test

# Verbose test output
make test-v
```

### Running E2E Tests

```bash
# Install Playwright
cd e2e && npm install && npx playwright install

# Run tests
make e2e

# Run with UI
make e2e-ui

# Run headed (visible browser)
make e2e-headed
```

### Adding a New Feature

1. **Define the API** - Create types in `feature/{name}/api.go`
2. **Implement the Service** - Create business logic in `feature/{name}/service.go`
3. **Add the Store** - Create data access in `store/duckdb/{name}_store.go`
4. **Add HTTP Handlers** - Create handlers in `app/web/handler/rest/{name}.go`
5. **Register Routes** - Update `app/web/server.go`

### Code Style

- Use ULID for entity IDs
- Use slugs for URL-friendly identifiers
- Return errors from service methods
- Use dependency injection for stores

## Makefile Reference

```bash
make build      # Build binary to ~/bin/cms
make run        # Run locally
make init       # Initialize database
make seed       # Seed demo data
make test       # Run all tests
make test-v     # Verbose tests
make clean      # Remove binary and data
make update     # Update dependencies
make e2e        # Run E2E tests
make e2e-ui     # E2E with Playwright UI
make e2e-headed # E2E with visible browser
make help       # Show all commands
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CGO_ENABLED` | Must be `1` for DuckDB support |
| `GOWORK` | Set to `off` when building CLI module |

## Troubleshooting

### CGO Errors

DuckDB requires CGO. Ensure you have a C compiler installed:

```bash
# macOS
xcode-select --install

# Ubuntu/Debian
sudo apt-get install build-essential

# Verify CGO is enabled
go env CGO_ENABLED  # Should output: 1
```

### Database Errors

```bash
# Reset database
rm -rf ~/data/blueprint/cms
cms init
cms seed
```

### Port Already in Use

```bash
# Use a different port
cms serve --addr :3001
```

## License

MIT License - see LICENSE file for details.
