# News Blueprint Tests

This document outlines the test implementation plan for the news blueprint, following the same patterns as the forum blueprint.

## Test Files to Create

### 1. E2E Tests: `app/web/server_e2e_test.go`

End-to-end tests for the web server covering all API endpoints and HTML pages.

**Build Tag**: `//go:build e2e`
**Run Command**: `E2E_TEST=1 go test -v -tags=e2e ./app/web/...`

#### Test Helpers
- `setupTestServer(t)` - Creates httptest server with temp DuckDB store
- `createTestUser(t, store, username)` - Creates test user via service layer
- `loginUser(t, ts, username, password)` - Logs in and returns session token
- `authRequest(t, method, url, token, body)` - Makes authenticated HTTP request
- `get(t, url)` - Makes GET request
- `assertStatus(t, resp, want)` - Asserts HTTP status code
- `assertContains(t, body, substr)` - Asserts body contains substring

#### Test Cases

**Authentication (`TestE2E_Auth`)**
- Register new user
- Login with valid credentials
- Login with invalid password
- Logout

**Stories (`TestE2E_Stories`)**
- Create link story
- Create text story
- Get story by ID
- List stories

**Comments (`TestE2E_Comments`)**
- Create comment on story
- List comments for story

**Voting (`TestE2E_Voting`)**
- Upvote story
- Downvote story
- Unvote story
- Vote on comment
- Unvote comment

**Users (`TestE2E_UserProfile`)**
- Get user profile
- Get user's stories
- Get user's comments

**HTML Pages (`TestE2E_HTMLPages`)**
- Home page (/)
- Newest page (/newest)
- Top page (/top)
- Story page (/story/{id})
- User page (/user/{username})
- Tag page (/tag/{name})
- Login page (/login)
- Register page (/register)

**User Journey (`TestE2E_Scenario_UserJourney`)**
Complete user flow:
1. Register
2. Login
3. Create story
4. Add comment
5. Vote on story
6. Check profile
7. Logout

**Authorization (`TestE2E_Unauthorized`)**
- Create story without auth
- Create comment without auth
- Vote without auth

### 2. CLI Tests

#### `cli/root_test.go`
- `TestVersionString` - Test with dev version
- `TestVersionString_WithVersion` - Test with specific version
- `TestVersionString_Empty` - Test empty version fallback
- `TestVersionString_Whitespace` - Test whitespace version fallback
- `TestVersionVariables` - Verify default values

#### `cli/init_test.go`
- `TestNewInit` - Verify command structure (Use, Short, RunE)
- `TestRunInit` - Test database creation in temp dir
- `TestRunInit_InvalidPath` - Test error for invalid path

#### `cli/serve_test.go`
- `TestNewServe` - Verify command structure
- `TestRunServe_InvalidPath` - Test error for invalid path
- `TestRunServe_StartsServer` - Test server starts and stops with context
- `TestModeString` - Test dev/production mode strings

#### `cli/seed_test.go`
- `TestNewSeed` - Verify command structure
- `TestNewSeedHN` - Verify HN subcommand structure
- `TestNewSeedSample` - Verify sample subcommand structure
- `TestRunSeedSample` - Test sample data seeding

#### `cli/ui_test.go`
- `TestNewUI` - Test UI creation
- `TestUI_Header` - Test header output
- `TestUI_Info` - Test info output
- `TestUI_Spinner` - Test spinner start/update/stop
- `TestUI_Success` - Test success message
- `TestUI_Error` - Test error message
- `TestUI_Warn` - Test warning message
- `TestUI_Summary` - Test summary section
- `TestUI_StoryRow` - Test story row formatting
- `TestUI_StoryRow_LongTitle` - Test title truncation
- `TestUI_UserRow` - Test user row formatting
- `TestIsTerminal` - Test terminal detection

## Running Tests

```bash
# Run CLI tests
go test -v ./cli/...

# Run E2E tests
E2E_TEST=1 go test -v -tags=e2e ./app/web/...

# Run all tests
E2E_TEST=1 go test -v -tags=e2e ./...
```

## Implementation Notes

1. E2E tests use `//go:build e2e` build tag and `E2E_TEST=1` env var gate
2. Each test function checks for E2E_TEST env var and skips if not set
3. Tests use temp directories for database to ensure isolation
4. Tests restore global state (dataDir, addr) after modification
5. HTTP tests use httptest.Server for efficient testing
6. UI tests capture stdout using os.Pipe for verification
