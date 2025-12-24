# Social Blueprint UI Testing

This spec covers the implementation of UI testing for the social blueprint, including:
1. Makefile for development/build commands
2. Comprehensive UI tests for all HTML pages

## Overview

The social blueprint is a Twitter/Mastodon-style social network application with the following pages:
- Home page (`/`)
- Login page (`/login`)
- Register page (`/register`)
- Explore page (`/explore`)
- Search page (`/search`)
- Profile page (`/u/{username}`)
- Post page (`/u/{username}/post/{id}`)
- Follow list page (`/u/{username}/followers`, `/u/{username}/following`)
- Tag page (`/tags/{tag}`)
- Notifications page (`/notifications`)
- Bookmarks page (`/bookmarks`)
- Lists page (`/lists`)
- List view page (`/lists/{id}`)
- Settings page (`/settings`, `/settings/{page}`)
- 404 page

## Makefile

Based on the forum blueprint's Makefile, create a similar structure with:

### Targets:
- `build` - Build binary to `$HOME/bin/social`
- `run` - Run CLI with arguments
- `serve` - Run `social serve`
- `dev` - Run `social serve --dev`
- `init` - Run `social init`
- `seed` - Run `social seed`
- `test` - Run all tests
- `test-v` - Run tests with verbose output
- `test-race` - Run tests with race detector
- `test-store` - Run store tests only
- `test-cli` - Run CLI tests only
- `test-e2e` - Run e2e tests (E2E_TEST=1)
- `test-cover` - Run tests with coverage
- `update` - Refresh dependencies
- `clean` - Remove binary
- `clean-data` - Remove data directory
- `fmt` - Format code
- `vet` - Run go vet
- `lint` - Run fmt and vet
- `help` - Show available commands

### Variables:
- `DATA_DIR` - `$HOME/data/blueprint/social`
- `BINARY` - `$HOME/bin/social`
- `CGO_ENABLED=1` - Required for DuckDB

## UI Tests (`app/web/server_ui_test.go`)

### Test Setup Functions

1. `setupUITestServer(t *testing.T) *httptest.Server`
   - Create temp directory
   - Open store with duckdb.Open()
   - Create server with web.New()
   - Return httptest.Server

2. `setupUITestServerWithData(t *testing.T) (*httptest.Server, *duckdb.Store)`
   - Same as above but also creates:
     - Test user account
     - Test post
   - Returns both server and store for data access

### Helper Functions

1. `getUIPage(t, url string) (status int, body string)` - GET page and return status/body
2. `assertUIValidHTML(t, body string)` - Check DOCTYPE and closing tags
3. `assertUIPageContains(t, body string, substrings ...string)` - Check content
4. `assertNoTemplateErrors(t, body, path string)` - Check for template errors
5. `truncateForError(s string) string` - Truncate body for error messages

### Test Cases

#### Basic Page Tests
- `TestUI_HomePage` - Home page renders with title, nav, compose box
- `TestUI_LoginPage` - Login form renders correctly
- `TestUI_RegisterPage` - Register form renders correctly
- `TestUI_ExplorePage` - Explore page with trending content
- `TestUI_SearchPage` - Search page with search input
- `TestUI_SearchPage_WithQuery` - Search preserves query param
- `TestUI_ProfilePage` - Profile page with user data
- `TestUI_ProfilePage_NotFound` - 404 for missing user
- `TestUI_PostPage` - Single post view
- `TestUI_PostPage_NotFound` - 404 for missing post
- `TestUI_TagPage` - Tag timeline page
- `TestUI_NotificationsPage` - Notifications page
- `TestUI_BookmarksPage` - Bookmarks page
- `TestUI_ListsPage` - Lists page
- `TestUI_ListViewPage` - Single list view
- `TestUI_SettingsPage` - Settings page
- `TestUI_404Page` - 404 error page

#### Component Tests
- `TestUI_Navigation` - Nav component with links
- `TestUI_StaticAssets` - CSS and JS served correctly
- `TestUI_ResponsiveMetaTag` - Viewport meta tag present

#### Template Isolation Tests
- `TestUI_TemplateIsolation` - Each page renders its own content block
- `TestUI_AllPagesRenderWithoutErrors` - All pages render without template errors

#### Form Tests
- `TestUI_FormValidation` - Required fields and validation attributes

## Implementation Steps

1. Create Makefile
   - Copy structure from forum blueprint
   - Update binary name, paths, module references

2. Create server_ui_test.go
   - Import required packages
   - Implement setup functions
   - Implement helper functions
   - Implement all test cases

3. Run tests
   - `make test` - All tests pass
   - `make test-v` - Verbose output
   - Verify each command works

## Dependencies

The UI tests depend on:
- `github.com/go-mizu/blueprints/social/app/web`
- `github.com/go-mizu/blueprints/social/feature/accounts`
- `github.com/go-mizu/blueprints/social/feature/posts`
- `github.com/go-mizu/blueprints/social/store/duckdb`
- `github.com/duckdb/duckdb-go/v2`
