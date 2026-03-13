# 0716 — Pinterest Scraper

## Overview

Scrape all public Pinterest data (pins, boards, users, and search results) into a local DuckDB database. Designed for visual content research, board archiving, and creator analytics.

Architecture follows the established pattern (`pkg/scrape/goodread`, `pkg/scrape/amazon`): entity-specific Task structs implementing `pkg/core.Task[State, Metric]`, a shared HTTP client with session/CSRF handling, and CLI subcommands wired into `cli/pinterest.go`.

Pinterest's live public surface is split: search still works through the internal Resource API, while user and board pages expose the needed public data through SSR bootstrap JSON in `__PWS_DATA__` / `__PWS_INITIAL_PROPS__`. Direct unauthenticated replay of `UserResource`, `BoardsResource`, and `BoardFeedResource` now returns `403 Invalid Resource Request`. The implementation follows the live-compatible mix: Resource API for search, SSR bootstrap extraction for users and boards. The existing `pkg/scrape/pinterest.go` (used by `search scrape`) is unchanged; this package is a standalone dedicated scraper that stores results in its own `pinterest.duckdb`.

Two DuckDB files: `pinterest.duckdb` for all scraped data, `state.duckdb` for the job queue and crawl state.

Anti-bot strategy: session warm-up on startup (GET pinterest.com → collect cookies + CSRF token), rotating browser User-Agents, 200ms default delay between requests. No rod fallback needed.

## Package Layout

```
blueprints/search/
├── pkg/scrape/pinterest/
│   ├── types.go          # Pin, Board, User, QueueItem, SearchResult, entity constants
│   ├── config.go         # Config + DefaultConfig()
│   ├── client.go         # HTTP client: session warmup, CSRF, UA rotation, all API methods
│   ├── db.go             # pinterest.duckdb schema + upsert methods
│   ├── state.go          # state.duckdb: queue, jobs, visited (same pattern as goodread)
│   ├── display.go        # PrintStats, PrintCrawlProgress
│   ├── task_search.go    # SearchTask — paginated pin search → DB
│   ├── task_board.go     # BoardTask — all pins from one board → DB
│   ├── task_user.go      # UserTask — user profile + board list → DB, enqueues boards
│   └── task_crawl.go     # CrawlTask — queue-driven dispatcher
│
└── cli/pinterest.go      # All CLI subcommands under "pinterest"
```

## Data Model

### pinterest.duckdb

Driver: `github.com/duckdb/duckdb-go/v2`. Array columns stored as JSON text (VARCHAR). Upserts via `INSERT OR REPLACE INTO`.

```sql
CREATE TABLE IF NOT EXISTS pins (
  pin_id         VARCHAR PRIMARY KEY,
  title          VARCHAR,
  description    VARCHAR,
  alt_text       VARCHAR,
  image_url      VARCHAR,
  image_width    INTEGER,
  image_height   INTEGER,
  pin_url        VARCHAR,
  source_url     VARCHAR,
  board_id       VARCHAR,
  board_name     VARCHAR,
  user_id        VARCHAR,
  username       VARCHAR,
  saved_count    INTEGER,
  comment_count  INTEGER,
  created_at     TIMESTAMP,
  fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS boards (
  board_id       VARCHAR PRIMARY KEY,
  name           VARCHAR,
  slug           VARCHAR,
  description    VARCHAR,
  user_id        VARCHAR,
  username       VARCHAR,
  pin_count      INTEGER,
  follower_count INTEGER,
  cover_url      VARCHAR,
  category       VARCHAR,
  is_secret      BOOLEAN DEFAULT FALSE,
  url            VARCHAR,
  fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
  user_id          VARCHAR PRIMARY KEY,
  username         VARCHAR,
  full_name        VARCHAR,
  bio              VARCHAR,
  website          VARCHAR,
  follower_count   INTEGER,
  following_count  INTEGER,
  board_count      INTEGER,
  pin_count        INTEGER,
  monthly_views    BIGINT,
  avatar_url       VARCHAR,
  url              VARCHAR,
  fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### state.duckdb

Same schema as `pkg/scrape/goodread/state.go`:

```sql
CREATE SEQUENCE IF NOT EXISTS queue_id_seq;

CREATE TABLE IF NOT EXISTS queue (
  id           BIGINT DEFAULT nextval('queue_id_seq') PRIMARY KEY,
  url          VARCHAR UNIQUE NOT NULL,
  entity_type  VARCHAR NOT NULL,  -- search|board|user
  priority     INTEGER DEFAULT 0,
  status       VARCHAR DEFAULT 'pending',
  attempts     INTEGER DEFAULT 0,
  last_attempt TIMESTAMP,
  error        VARCHAR,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_queue_status_priority ON queue(status, priority DESC, created_at);

CREATE TABLE IF NOT EXISTS jobs (
  job_id       VARCHAR PRIMARY KEY,
  name         VARCHAR,
  type         VARCHAR,
  status       VARCHAR,
  started_at   TIMESTAMP,
  completed_at TIMESTAMP,
  config       VARCHAR,
  stats        VARCHAR
);

CREATE TABLE IF NOT EXISTS visited (
  url         VARCHAR PRIMARY KEY,
  fetched_at  TIMESTAMP,
  status_code INTEGER,
  entity_type VARCHAR
);
```

**Queue priorities:**

| Priority | Entity types |
|----------|-------------|
| 10 | board |
| 5  | user |
| 3  | search |

## Pinterest Public JSON Surfaces

Search API requests use the same host (`https://www.pinterest.com`) and require:
- `User-Agent`: rotating browser UA string
- `X-Requested-With: XMLHttpRequest`
- `X-Pinterest-Appstate: active`
- `X-Pinterest-Pws-Handler: www/search/[scope].js`
- `X-CSRFToken: <value from csrftoken cookie>`
- `Referer: https://www.pinterest.com<source_url>`

### Session Warm-Up

```go
GET https://www.pinterest.com/
// Collects: csrftoken, _pinterest_sess, _auth (cookies)
// CSRF token extracted from jar and sent in X-CSRFToken header
```

### Search Pins

```
GET /resource/BaseSearchResource/get/
  ?source_url=/search/pins/?q={query}&rs=typed
  &data={"options":{"query":"...","scope":"pins","rs":"typed","page_size":25,"bookmarks":["..."]}}
  &_={timestamp_ms}

Response: resource_response.data.results[] (array of pins)
          resource_response.bookmark (next page token; "-end-" or base64 "Y2JOb25l..." = end)
```

### Board Pins

Direct unauthenticated replay of `BoardFeedResource` now returns `403 Invalid Resource Request`.

Implementation path:
1. `GET https://www.pinterest.com/{username}/{board-slug}/`
2. Parse `<script id="__PWS_INITIAL_PROPS__" type="application/json">`
3. Extract:
   - `initialReduxState.boards` for board metadata
   - `BoardFeedResource.*.data[]` for the SSR pin batch
   - fallback to `initialReduxState.pins` when needed

### User Boards

Direct unauthenticated replay of `BoardsResource` now returns `403 Invalid Resource Request`.

Implementation path:
1. `GET https://www.pinterest.com/{username}/`
2. Parse `<script id="__PWS_INITIAL_PROPS__" type="application/json">`
3. Extract the visible board list from `initialReduxState.boards`

### User Profile

User profile fields are extracted from SSR bootstrap JSON on the public profile page:
- `initialReduxState.users`
- fallback `resources.UserResource`

### API Response Shapes

**Pin object** (from both search and board feed):
```json
{
  "id": "123456",
  "type": "pin",
  "title": "...",
  "grid_title": "...",
  "description": "...",
  "auto_alt_text": "...",
  "link": "https://source.example.com/...",
  "save_count": 42,
  "comment_count": 3,
  "created_at": "Thu, 01 Jan 2026 00:00:00 +0000",
  "images": {
    "orig": {"url": "...", "width": 1920, "height": 1080},
    "736x": {"url": "...", "width": 736, "height": 414}
  },
  "board": {"id": "111", "name": "My Board"},
  "pinner": {"id": "222", "username": "someuser"}
}
```

**Board object**:
```json
{
  "id": "111",
  "name": "My Board",
  "url": "/username/my-board/",
  "description": "...",
  "pin_count": 100,
  "follower_count": 50,
  "cover_pin": {"images": {"736x": {"url": "..."}}},
  "owner": {"id": "222", "username": "someuser"},
  "privacy": "public",
  "category": "art"
}
```

**User object**:
```json
{
  "id": "222",
  "username": "someuser",
  "full_name": "Some User",
  "about": "Bio text...",
  "website_url": "https://example.com",
  "follower_count": 1000,
  "following_count": 200,
  "board_count": 15,
  "pin_count": 800,
  "monthly_views": 50000,
  "image_medium_url": "https://i.pinimg.com/..."
}
```

## core.Task Wiring

```go
// pkg/core.Task[State, Metric] interface:
//   Run(ctx context.Context, emit func(*State)) (Metric, error)
```

Compile-time interface assertion required in every task file.

### Task Table

| Task | State fields | Metric fields |
|------|-------------|---------------|
| SearchTask | `Query, Status, Error string; Page, PinsFound int` | `Fetched, Skipped, Failed, Pages int` |
| BoardTask | `URL, BoardID, Status, Error string; PinsFound int` | `Fetched, Skipped, Failed, Pages int` |
| UserTask | `URL, Username, Status, Error string; BoardsFound int` | `Fetched, Skipped, Failed int` |
| CrawlTask | `Done, Pending, Failed int64; InFlight []string; RPS float64` | `Done, Failed int64; Duration time.Duration` |

### SearchTask (canonical example)

```go
type SearchState  struct { Query, Status, Error string; Page, PinsFound int }
type SearchMetric struct { Fetched, Skipped, Failed, Pages int }

type SearchTask struct {
    Query   string
    MaxPins int       // 0 = use config default (500)
    Client  *Client
    DB      *DB
    StateDB *State    // optional; marks visited
    Config  Config
}

var _ core.Task[SearchState, SearchMetric] = (*SearchTask)(nil)
```

**SearchTask.Run flow:**
1. Call `Client.SearchPins(ctx, query, maxPins)` — returns `[]Pin`
2. For each pin: call `DB.UpsertPin(pin)`
3. If `StateDB != nil`: call `StateDB.Done(searchURL, 200, EntitySearch)`
4. Emit progress every 50 pins

### BoardTask.Run flow:
1. `Client.FetchBoardBootstrap(ctx, url)` — fetches HTML and extracts SSR bootstrap JSON
2. `DB.UpsertBoard(board)`
3. `DB.UpsertPin(pin)` for each bootstrap pin

### UserTask.Run flow:
1. `Client.FetchUserBootstrap(ctx, username)` → `DB.UpsertUser(user)`
2. Store visible SSR boards from `initialReduxState.boards` via `DB.UpsertBoard(board)`
3. For each board: `StateDB.Enqueue(boardURL, EntityBoard, 10)` if StateDB != nil

### CrawlTask dispatch:
```go
switch item.EntityType {
case EntitySearch: task = &SearchTask{...}
case EntityBoard:  task = &BoardTask{...}
case EntityUser:   task = &UserTask{...}
}
```

## File-by-File Notes

### types.go

```go
const (
    BaseURL      = "https://www.pinterest.com"
    EntitySearch = "search"
    EntityBoard  = "board"
    EntityUser   = "user"
)
```

`Pin`, `Board`, `User` structs matching DB schema with `json` tags.

`ExtractUsername(url string) string` — extracts username from pinterest URLs.
`ExtractBoardSlug(url string) (username, slug string)` — extracts `/{username}/{slug}/`.
`NormalizeBoardURL(s string) string` — accepts `username/board` or full URL.
`NormalizeUserURL(s string) string` — accepts username or full URL.

### client.go

```go
type Client struct {
    http       *http.Client
    cookies    []*http.Cookie
    userAgents []string
    delay      time.Duration
    lastReq    time.Time
    mu         sync.Mutex
}

func NewClient(cfg Config) (*Client, error)        // warms up session
func (c *Client) SearchPins(ctx, query string, maxPins int) ([]Pin, error)
func (c *Client) FetchBoardPage(ctx, boardURL string) (boardID string, err error)
func (c *Client) FetchBoardBootstrap(ctx, boardURL string) (*Board, []Pin, error)
func (c *Client) FetchBoardPins(ctx, boardID, bookmark string) ([]Pin, string, error)
func (c *Client) FetchUser(ctx, username string) (*User, error)
func (c *Client) FetchUserBootstrap(ctx, username string) (*User, []Board, error)
func (c *Client) FetchUserBoards(ctx, username, bookmark string) ([]Board, string, error)
```

`SearchPins` is a pagination loop over `searchPage()` (mirrors existing `pkg/scrape/pinterest.go`) and includes the live-required `X-Pinterest-Pws-Handler` header.

`FetchBoardPage` fetches the board HTML and extracts the board_id using `extractBoardID(html []byte) string` which searches for `"board_id":"(\d+)"` and `"id":"(\d+)"` patterns in the embedded JSON state.

All network calls: `rateLimit()` before request. Search API calls also require `X-CSRFToken` from the cookie jar.

### db.go

- `DB` struct wrapping `*sql.DB`
- `OpenDB(path string) (*DB, error)`
- `Close() error`, `Path() string`
- `UpsertPin(p Pin) error`
- `UpsertBoard(b Board) error`
- `UpsertUser(u User) error`
- `GetStats() (DBStats, error)` — counts per table + file size
- `RecentPins(limit int) ([]Pin, error)`
- Helpers: `nullStr`, `nullInt`, `nullTime`, `nullBool`

### state.go

Identical to `pkg/scrape/goodread/state.go`. Same struct, same methods, same schema. Copy verbatim and change package name to `pinterest`.

### display.go

```go
func PrintStats(db *DB, stateDB *State) error
func PrintCrawlProgress(state *CrawlState)
```

`PrintCrawlProgress` prints one-line progress: `done=N pending=N failed=N in-flight=N rps=X.X`.

## CLI Commands

Register with `root.AddCommand(NewPinterest())` in `cli/root.go`.

```
search pinterest search  <query>          [--max-pins 500]
search pinterest board   <url|user/board> [--max-pins 0]
search pinterest user    <username>       [--boards]
search pinterest seed    --file urls.txt  [--entity board|user|search] [--priority 10]
search pinterest crawl                   [--workers 2] [--delay 200]
search pinterest info
search pinterest jobs                    [--limit 20]
search pinterest queue                   [--status pending] [--limit 20]

# Global flags (all subcommands)
  --db      path to pinterest.duckdb   (default $HOME/data/pinterest/pinterest.duckdb)
  --state   path to state.duckdb       (default $HOME/data/pinterest/state.duckdb)
  --delay   ms between API requests    (default 200)
```

### cli/pinterest.go structure (mirrors cli/goodread.go)

```go
func NewPinterest() *cobra.Command
func newPinterestSearch() *cobra.Command
func newPinterestBoard() *cobra.Command
func newPinterestUser() *cobra.Command
func newPinterestSeed() *cobra.Command
func newPinterestCrawl() *cobra.Command
func newPinterestInfo() *cobra.Command
func newPinterestJobs() *cobra.Command
func newPinterestQueue() *cobra.Command

func addPinterestFlags(cmd *cobra.Command, dbPath, statePath *string, delay *int)
func openPinterestDBs(dbPath, statePath string, delay int) (*pinterest.DB, *pinterest.State, *pinterest.Client, error)
```

## Key Lessons (inherited + Pinterest-specific)

- Session warm-up is mandatory: `GET pinterest.com` first to collect `csrftoken` cookie.
- Search API requests require `X-CSRFToken` and `X-Pinterest-Pws-Handler`.
- Board and user public data are now more reliable in SSR bootstrap JSON than by replaying internal resources unauthenticated.
- Board ID is not in the URL and must be extracted from embedded HTML JSON state.
- Search bookmark ending: `""`, `"-end-"`, or base64 prefix `"Y2JOb25l"`.
- `INSERT OR REPLACE INTO` for upserts; arrays as JSON VARCHAR.
- DuckDB single-connection per file — do not open same file from two goroutines.
- 200ms delay is safe for Pinterest's internal API (much lower than Goodreads 2s).
- `queue.Pop` must use `UPDATE ... RETURNING` for atomic claim.
- `state.duckdb` uses `INSERT OR IGNORE` to silently skip duplicate URLs.
- Image priority: `orig > 736x > 474x > 236x` by resolution.
