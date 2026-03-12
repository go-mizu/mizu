# 0714 — Goodreads Scraper

## Overview

Scrape all public Goodreads data (books, authors, reviews, series, lists, quotes, users, genres, shelves, search results) into a local DuckDB database. Designed for research, analytics, and content aggregation use cases.

Architecture follows the project's established scraper pattern (`pkg/scrape/insta`, `pkg/scrape/x`, `pkg/scrape/qq`): entity-specific Task structs implementing `pkg/core.Task[State, Metric]`, a shared HTTP client with rod fallback, and CLI subcommands wired into `cli/goodread.go`.

## Package Layout

```
blueprints/search/
├── pkg/scrape/goodread/
│   ├── types.go          # all Go structs: Book, Author, Review, Series, List, Quote, User, Genre, Shelf
│   ├── client.go         # HTTP client: retry, rate limiting, User-Agent rotation, rod fallback
│   ├── config.go         # Config struct
│   ├── parse_book.go     # goquery parser for /book/show/<id>
│   ├── parse_author.go   # goquery parser for /author/show/<id>
│   ├── parse_list.go     # goquery parser for /list/show/<id> and /list/tag/<tag>
│   ├── parse_series.go   # goquery parser for /series/<id>
│   ├── parse_search.go   # goquery parser for /search?q=...
│   ├── parse_quote.go    # goquery parser for /quotes/<id> and /author/quotes/<id>
│   ├── parse_genre.go    # goquery parser for /genres/<name>
│   ├── parse_user.go     # goquery parser for /user/show/<id> and /<username>
│   ├── parse_shelf.go    # goquery parser for /review/list/<user_id>?shelf=...
│   ├── parse_review.go   # extracts embedded reviews from book page HTML/JSON-LD
│   ├── task_book.go      # BookTask implements core.Task[BookState, BookMetric]
│   ├── task_author.go    # AuthorTask implements core.Task[AuthorState, AuthorMetric]
│   ├── task_series.go    # SeriesTask implements core.Task[SeriesState, SeriesMetric]
│   ├── task_list.go      # ListTask implements core.Task[ListState, ListMetric]
│   ├── task_quote.go     # QuoteTask implements core.Task[QuoteState, QuoteMetric]
│   ├── task_user.go      # UserTask implements core.Task[UserState, UserMetric]
│   ├── task_genre.go     # GenreTask implements core.Task[GenreState, GenreMetric]
│   ├── task_shelf.go     # ShelfTask implements core.Task[ShelfState, ShelfMetric]
│   ├── task_crawl.go     # CrawlTask: frontier loop, dispatches to entity tasks
│   ├── db.go             # goodread.duckdb schema + upsert methods
│   ├── state.go          # state.duckdb: queue, jobs, visited tables + frontier ops
│   └── display.go        # progress/stats display helpers (lipgloss)
└── cli/goodread.go       # all CLI subcommands under "goodread"
```

Reference: `blueprints/book/pkg/goodreads/` contains an existing parser — read it for selector/regex patterns but implement from scratch here. Several fields in the reference (`WorkID`, `OriginalTitle`, `Characters`, `Settings`, `LiteraryAwards`, `RatingDist`, `CurrentlyReading`, `WantToRead`) are intentionally excluded from the initial schema to keep it focused; they can be added in a migration later.

## Data Model

### goodread.duckdb

Driver: `github.com/duckdb/duckdb-go/v2` (matches `pkg/scrape/x/db.go` and `pkg/scrape/qq/db.go`).

**Array columns** (`VARCHAR[]`): DuckDB arrays require special handling in Go. Store arrays as JSON text (e.g., `["a","b","c"]`) and cast on read — the same approach used in `pkg/scrape/x/db.go`. Declare the column as `VARCHAR` in DDL (storing JSON), not `VARCHAR[]`, to avoid driver impedance. Add helper functions `encodeStringSlice([]string) string` and `decodeStringSlice(string) []string` in `db.go`.

**Upserts**: use `INSERT OR REPLACE INTO` (DuckDB supports SQLite-compatible syntax with `duckdb-go/v2`). Include a migrations block (pattern from `pkg/scrape/x/db.go`) using `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` so future columns can be added without deleting the database.

```sql
-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS books (
  book_id              VARCHAR PRIMARY KEY,
  title                VARCHAR,
  title_without_series VARCHAR,
  description          VARCHAR,
  author_id            VARCHAR,
  author_name          VARCHAR,
  isbn                 VARCHAR,
  isbn13               VARCHAR,
  asin                 VARCHAR,
  avg_rating           DOUBLE,
  ratings_count        BIGINT,
  reviews_count        BIGINT,
  published_year       INTEGER,
  publisher            VARCHAR,
  language             VARCHAR,
  pages                INTEGER,
  format               VARCHAR,
  series_id            VARCHAR,
  series_name          VARCHAR,
  series_position      VARCHAR,
  genres               VARCHAR,   -- JSON: ["Fantasy","Young Adult"]
  cover_url            VARCHAR,
  url                  VARCHAR,
  similar_book_ids     VARCHAR,   -- JSON: ["123","456"]
  fetched_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS authors (
  author_id       VARCHAR PRIMARY KEY,
  name            VARCHAR,
  bio             VARCHAR,
  photo_url       VARCHAR,
  website         VARCHAR,
  born_date       VARCHAR,
  died_date       VARCHAR,
  hometown        VARCHAR,
  influences      VARCHAR,   -- JSON
  genres          VARCHAR,   -- JSON
  avg_rating      DOUBLE,
  ratings_count   BIGINT,
  books_count     INTEGER,
  followers_count INTEGER,
  url             VARCHAR,
  fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS series (
  series_id          VARCHAR PRIMARY KEY,
  name               VARCHAR,
  description        VARCHAR,
  total_books        INTEGER,
  primary_work_count INTEGER,
  url                VARCHAR,
  fetched_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS series_books (
  series_id VARCHAR NOT NULL,
  book_id   VARCHAR NOT NULL,
  position  INTEGER,
  PRIMARY KEY (series_id, book_id)
);

CREATE TABLE IF NOT EXISTS lists (
  list_id         VARCHAR PRIMARY KEY,
  name            VARCHAR,
  description     VARCHAR,
  books_count     INTEGER,
  voters_count    INTEGER,
  tags            VARCHAR,   -- JSON
  created_by_user VARCHAR,
  url             VARCHAR,
  fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS list_books (
  list_id VARCHAR NOT NULL,
  book_id VARCHAR NOT NULL,
  rank    INTEGER,
  votes   INTEGER,
  PRIMARY KEY (list_id, book_id)
);

CREATE TABLE IF NOT EXISTS reviews (
  review_id   VARCHAR PRIMARY KEY,
  book_id     VARCHAR,
  user_id     VARCHAR,
  user_name   VARCHAR,
  rating      INTEGER,
  text        VARCHAR,
  date_added  TIMESTAMP,
  likes_count INTEGER,
  is_spoiler  BOOLEAN DEFAULT FALSE,
  url         VARCHAR,
  fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS quotes (
  quote_id    VARCHAR PRIMARY KEY,
  text        VARCHAR,
  author_id   VARCHAR,
  author_name VARCHAR,
  book_id     VARCHAR,
  book_title  VARCHAR,
  likes_count INTEGER,
  tags        VARCHAR,   -- JSON
  url         VARCHAR,
  fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
  user_id           VARCHAR PRIMARY KEY,
  name              VARCHAR,
  username          VARCHAR,
  location          VARCHAR,
  joined_date       TIMESTAMP,
  friends_count     INTEGER,
  books_read_count  INTEGER,
  ratings_count     INTEGER,
  reviews_count     INTEGER,
  avg_rating        DOUBLE,
  bio               VARCHAR,
  website           VARCHAR,
  avatar_url        VARCHAR,
  favorite_book_ids VARCHAR,  -- JSON
  url               VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS genres (
  slug        VARCHAR PRIMARY KEY,
  name        VARCHAR,
  description VARCHAR,
  books_count INTEGER,
  url         VARCHAR,
  fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- shelf_id convention: "{user_id}/{shelf_name}" (e.g. "12345678/to-read")
-- Goodreads has no numeric shelf ID; the composite user_id+name is the natural key.
CREATE TABLE IF NOT EXISTS shelves (
  shelf_id    VARCHAR PRIMARY KEY,
  user_id     VARCHAR,
  name        VARCHAR,
  books_count INTEGER,
  url         VARCHAR,
  fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shelf_books (
  shelf_id   VARCHAR NOT NULL,
  user_id    VARCHAR NOT NULL,
  book_id    VARCHAR NOT NULL,
  date_added TIMESTAMP,
  rating     INTEGER,
  date_read  TIMESTAMP,
  PRIMARY KEY (shelf_id, book_id)
);
```

### state.duckdb

```sql
CREATE SEQUENCE IF NOT EXISTS queue_id_seq;

CREATE TABLE IF NOT EXISTS queue (
  id           BIGINT DEFAULT nextval('queue_id_seq') PRIMARY KEY,
  url          VARCHAR UNIQUE NOT NULL,
  entity_type  VARCHAR NOT NULL,  -- book|author|list|series|quote|user|genre|shelf|search
  priority     INTEGER DEFAULT 0,
  status       VARCHAR DEFAULT 'pending',  -- pending|in_progress|done|failed
  attempts     INTEGER DEFAULT 0,
  last_attempt TIMESTAMP,
  error        VARCHAR,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_queue_status_priority ON queue(status, priority DESC, created_at);

CREATE TABLE IF NOT EXISTS jobs (
  job_id       VARCHAR PRIMARY KEY,
  name         VARCHAR,
  type         VARCHAR,   -- crawl|search|sitemap|single
  status       VARCHAR,   -- running|done|failed|paused
  started_at   TIMESTAMP,
  completed_at TIMESTAMP,
  config       VARCHAR,   -- JSON text
  stats        VARCHAR    -- JSON text
);

CREATE TABLE IF NOT EXISTS visited (
  url          VARCHAR PRIMARY KEY,
  fetched_at   TIMESTAMP,
  status_code  INTEGER,
  entity_type  VARCHAR
);
```

## HTTP Client Strategy

1. **Plain HTTP first**: `net/http` with rotating browser User-Agents, realistic `Accept`/`Accept-Language` headers, configurable delay (default 2s).
2. **Rod fallback**: on 403, 429, or captcha pattern detected in response body, retry with `go-rod` headless Chrome using the stealth proxy pattern from `pkg/scrape/rod.go`. Rod browser is lazy-initialized — Chrome is not launched unless actually needed.
3. **Rate limiting**: token bucket (1 req/delay). On 429: parse `Retry-After` header, back off, surface wait time in display.
4. **Retry**: up to 3 attempts per URL; permanent failures (404) are marked `failed` in queue and not retried.

## core.Task Wiring

```go
// pkg/core.Task[State, Metric] interface:
//   Run(ctx context.Context, emit func(*State)) (Metric, error)

// Each entity task has its own State and Metric pair.
// Compile-time interface assertions are required in every task file:
//   var _ core.Task[BookState, BookMetric] = (*BookTask)(nil)
```

### Entity task State and Metric types

| Task | State fields | Metric fields |
|------|-------------|---------------|
| BookTask | `URL, Status, Error string` | `Fetched, Skipped, Failed int` |
| AuthorTask | `URL, Status, Error string` | `Fetched, Skipped, Failed int` |
| SeriesTask | `URL, Status, Error string` | `Fetched, Skipped, Failed int` |
| ListTask | `URL, Status, Error string; BooksFound int` | `Fetched, Skipped, Failed int` |
| QuoteTask | `URL, Status, Error string; QuotesFound int` | `Fetched, Skipped, Failed int` |
| UserTask | `URL, Status, Error string` | `Fetched, Skipped, Failed int` |
| GenreTask | `URL, Status, Error string; BooksFound int` | `Fetched, Skipped, Failed int` |
| ShelfTask | `URL, Status, Error string; BooksFound int` | `Fetched, Skipped, Failed, Pages int` |

### BookTask (canonical example)

```go
type BookState  struct { URL string; Status string; Error string }
type BookMetric struct { Fetched, Skipped, Failed int }

type BookTask struct {
    URL     string
    Client  *Client
    DB      *DB
    StateDB *State  // optional; if set, marks visited + enqueues discovered links
}

var _ core.Task[BookState, BookMetric] = (*BookTask)(nil)

func (t *BookTask) Run(ctx context.Context, emit func(*BookState)) (BookMetric, error)
```

Note: field is `StateDB *State` (not `State *State`) to avoid shadowing the `State` type name.

### SeriesTask (second canonical example)

```go
type SeriesState  struct { URL string; Status string; Error string }
type SeriesMetric struct { Fetched, Skipped, Failed int }

type SeriesTask struct {
    URL     string
    Client  *Client
    DB      *DB
    StateDB *State
}

var _ core.Task[SeriesState, SeriesMetric] = (*SeriesTask)(nil)

func (t *SeriesTask) Run(ctx context.Context, emit func(*SeriesState)) (SeriesMetric, error)
```

All other entity tasks follow the same pattern.

### CrawlTask

```go
type CrawlState  struct {
    Done, Pending, Failed int
    InFlight              []string  // URLs currently being fetched (one per worker)
    RPS                   float64
}
type CrawlMetric struct { Done, Failed int; Duration time.Duration }

type CrawlTask struct {
    Config  Config
    Client  *Client
    DB      *DB
    StateDB *State
}

var _ core.Task[CrawlState, CrawlMetric] = (*CrawlTask)(nil)

func (t *CrawlTask) Run(ctx context.Context, emit func(*CrawlState)) (CrawlMetric, error)
```

**CrawlTask dispatch pattern**: each of `Config.Workers` goroutines calls `StateDB.Pop(1)`, instantiates the appropriate entity task based on `entity_type`, calls `task.Run(ctx, nil)` (entity-level emit discarded; CrawlState tracks aggregate), then calls `StateDB.Done` or `StateDB.Fail`. `CrawlState.InFlight` is a `[]string` snapshot for display purposes — the live in-flight URL tracking is done via a `sync.Mutex` on `CrawlTask` itself (not on `CrawlState`); a copy is made into `CrawlState` before calling `emit`.

## state.go — Pop SQL

`Pop` must atomically claim N rows to prevent concurrent-worker double-claim:

```sql
UPDATE queue
SET status = 'in_progress', last_attempt = NOW(), attempts = attempts + 1
WHERE id IN (
    SELECT id FROM queue
    WHERE status = 'pending'
    ORDER BY priority DESC, created_at
    LIMIT ?
)
RETURNING id, url, entity_type, priority
```

DuckDB supports `UPDATE ... RETURNING`. `Enqueue` uses `INSERT OR IGNORE INTO queue(url, entity_type, priority) VALUES (?, ?, ?)` to silently skip duplicates.

## Reviews

Reviews are **not** fetched via a separate URL. Goodreads embeds the first 30 reviews in the book page HTML (inside a `<script type="application/ld+json">` block and/or `data-react-props` attributes). `parse_review.go` exports:

```go
func ParseReviews(doc *goquery.Document) ([]Review, error)
```

Called from within `parse_book.go` — `BookTask.Run` stores both the `Book` and its embedded `[]Review` in a single fetch. No separate `ReviewTask` is needed.

## Sitemap

Goodreads sitemap index is at `https://www.goodreads.com/sitemap.xml`. Individual sitemaps contain URLs of the form:
- `/book/show/<id>` → entity_type `book`
- `/author/show/<id>` → entity_type `author`
- `/list/show/<id>` → entity_type `list`

Infer `entity_type` from URL path prefix. The `sitemap` CLI command fetches the index, iterates each child sitemap, and calls `StateDB.Enqueue` for each URL found (skipping already-visited ones).

## Link Following Rules

CrawlTask auto-enqueues these discovered links during crawl:

| Source page | Auto-enqueued entity types |
|---|---|
| Book | author, series, similar books |
| Author | all books listed on author page |
| List | all books in list |
| Series | all books in series |
| Search results | all result books and authors |
| Genre page | top books shown on page |

Review user profiles are **not** auto-enqueued (too many, high risk of rate limiting).

## CLI Commands

Register with `root.AddCommand(NewGoodread())` in `cli/root.go`.

All subcommands registered under `search goodread` in `cli/goodread.go`.

```
# Single entity fetch
search goodread book   <id|url>
search goodread author <id|url>
search goodread series <id|url>
search goodread list   <id|url>
search goodread quote  <id|url>
search goodread user   <id|username>
search goodread genre  <slug>
search goodread shelf  <user_id> [--shelf read|to-read|currently-reading|<name>]

# Search (seeds queue + fetches result pages)
search goodread search <query>  [--type book|author|user|list] [--max-results 50]

# Sitemap seed (enqueues URLs from https://www.goodreads.com/sitemap.xml)
search goodread sitemap          [--limit N]

# Bulk crawl (CrawlTask; --resume continues existing job from state.duckdb)
search goodread crawl            [--workers 2]
                                 [--delay 2s]
                                 [--max-pages N]
                                 [--entity book,author,list,series]
                                 [--resume]

# Inspection
search goodread info             # row counts per table + queue depth + last job stats
search goodread jobs             # list recent jobs with status
search goodread queue            [--status pending|failed|done] [--limit 20]

# Global flags on all subcommands
  --db     path to goodread.duckdb   (default $HOME/data/goodread/goodread.duckdb)
  --state  path to state.duckdb      (default $HOME/data/goodread/state.duckdb)
  --delay  delay between requests    (default 2s)
  --rod    force rod for all fetches (default: only on HTTP failure)
```

## File-by-File Implementation Notes

### types.go
All domain structs matching the DB schema above. Arrays use `[]string` in Go. `time.Time` for timestamps. All fields exported with `json` tags.

### client.go
- `Client` struct: `http.Client`, `rodBrowser` (lazy-init), `delay time.Duration`, `userAgents []string`
- `Fetch(ctx, url) ([]byte, int, error)` — plain HTTP first, rod fallback on 403/429/captcha
- `FetchHTML(ctx, url) (*goquery.Document, error)` — wraps Fetch, parses HTML with goquery
- User-Agent list: 5–8 real Chrome/Firefox UA strings
- Rod: reuse `pkg/scrape/rod.go` stealth patterns; call `browser.MustConnect()` on first rod use

### parse_*.go
Each file exports `Parse<Entity>(doc *goquery.Document) (*<Entity>, error)`.
- Primary source: `<script type="application/ld+json">` (Goodreads embeds rich JSON-LD on book and author pages)
- Fallback: goquery HTML selectors for fields missing from JSON-LD
- Reference `blueprints/book/pkg/goodreads/parser.go` for selector patterns; rewrite cleanly

### db.go
- `DB` struct wrapping `*sql.DB`
- `OpenDB(path string) (*DB, error)` — creates dir, opens DuckDB, runs `initSchema()` then `runMigrations()`
- `initSchema()` runs all `CREATE TABLE IF NOT EXISTS` statements
- `runMigrations()` runs `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` for any new columns added after v1
- Upsert methods: `UpsertBook`, `UpsertAuthor`, `UpsertReview`, etc.
- Batch insert helpers: `InsertListBooks`, `InsertSeriesBooks`, `InsertShelfBooks`
- Helper functions: `encodeStringSlice([]string) string` (→ JSON), `decodeStringSlice(string) []string`

### state.go
- `State` struct wrapping `*sql.DB`
- `OpenState(path string) (*State, error)`
- `Enqueue(url, entityType string, priority int) error` — `INSERT OR IGNORE` (DuckDB `duckdb-go/v2` supports this SQLite-compatible syntax)
- `Pop(n int) ([]QueueItem, error)` — atomic claim via `UPDATE ... WHERE id IN (SELECT ... LIMIT ?) RETURNING`
- `Done(url string, statusCode int) error` — marks `done` in queue, upserts into `visited`
- `Fail(url, errMsg string) error` — increments attempts, marks `failed` if attempts >= 3
- `CreateJob(id, name, jobType string, config any) error`
- `UpdateJob(id, status string, stats any) error`

### display.go
Lipgloss-based progress display: queue depth, done/failed counts, in-flight URLs (one per worker), requests/sec. Pattern from `pkg/scrape/display.go`.

## Deployment

- Build: `make build-linux-noble` → deploy to server2
- Data dir on server: `$HOME/data/goodread/`
- Local test: `search goodread book 2767052` (The Hunger Games)
- Verify: `duckdb $HOME/data/goodread/goodread.duckdb "SELECT book_id, title, avg_rating FROM books LIMIT 5"`

## Key Lessons

- Use `INSERT OR REPLACE INTO` for upserts; DuckDB `duckdb-go/v2` supports SQLite-compatible syntax.
- Store arrays as JSON text (not `VARCHAR[]`) to avoid Go driver impedance — same approach as `pkg/scrape/x/db.go`.
- DuckDB single-connection: close `state.duckdb` before re-opening in a different context if needed (pattern from `pkg/crawl/`).
- Rod: lazy-init — don't launch Chrome unless a fetch actually fails.
- Goodreads JSON-LD on book pages is the richest data source; parse it first.
- Rate limit: 1 req/2s default. Faster = IP block.
- `shelf_id` convention: `"{user_id}/{shelf_name}"` — Goodreads has no numeric shelf ID.
- `queue.Pop` must use `UPDATE ... RETURNING` for atomic claim under concurrent workers.
- Schema migrations: always add `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` rather than dropping DB.
