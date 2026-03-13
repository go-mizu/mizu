# 0715 — Amazon Scraper

## Overview

Scrape all public Amazon data (products, brands, authors, categories, search results, bestseller lists, reviews, Q&A, sellers) into a local DuckDB database. Designed for product research, pricing analytics, and content aggregation.

Architecture follows the project's established scraper pattern (`pkg/scrape/goodread`, `pkg/scrape/insta`, `pkg/scrape/x`): entity-specific Task structs implementing `pkg/core.Task[State, Metric]`, a shared HTTP client with rod fallback, and CLI subcommands wired into `cli/amazon.go`.

Two DuckDB files: `amazon.duckdb` for all scraped data, `state.duckdb` for the job queue and crawl state.

Anti-bot strategy: plain `net/http` first with rotating browser User-Agents and realistic headers; rod headless Chrome fallback on 503, 429, or CAPTCHA detection. Default delay: 3s (more aggressive than Goodreads).

## Implementation Status

`pkg/scrape/amazon` and `cli/amazon.go` are implemented in-tree. As of March 13, 2026, the implementation covers products, brands, authors, categories, sellers, search result pages, bestseller lists, review pagination, and Q&A pagination.

Recent correctness fixes that must remain true:

- Crawl dispatch extracts ASINs from product, review, and Q&A URLs, not only `/dp/<ASIN>`.
- Category frontier expansion enqueues canonical browse-node URLs (`/b?node=<id>`), not search URLs.
- `search amazon search --max-results N` limits queued product URLs while still storing the full search page snapshot.
- Q&A row IDs are content-stable across pagination so later pages do not overwrite earlier records.
- Product parsing keeps richer graph fields when present: brand/store ID, seller ID/name, breadcrumb node IDs, fulfillment text, deduplicated image and relation arrays.

## Package Layout

```
blueprints/search/
├── pkg/scrape/amazon/
│   ├── types.go             # All Go structs: Product, Brand, Author, Category,
│   │                        #   BestsellerList, BestsellerEntry, Review, QA, Seller,
│   │                        #   SearchResult, QueueItem
│   ├── client.go            # HTTP client: UA rotation, delay, retry, rod fallback
│   │                        #   NEW DEPENDENCY: github.com/go-rod/rod (headless Chrome)
│   │                        #   Run: go get github.com/go-rod/rod in cmd/
│   ├── config.go            # Config struct + DefaultConfig()
│   ├── db.go                # amazon.duckdb schema + upsert methods
│   ├── state.go             # state.duckdb: queue, jobs, visited + Pop/Enqueue/Done/Fail
│   ├── display.go           # Lipgloss progress display
│   │                        #   (same pattern as pkg/scrape/goodread/display.go)
│   │
│   ├── parse_product.go     # /dp/<ASIN> — full product page parser
│   ├── parse_brand.go       # /stores/<brand> — brand store parser
│   ├── parse_author.go      # /author/<name> — Author Central parser
│   ├── parse_category.go    # /b?node=<id> — category/browse node parser
│   ├── parse_search.go      # /s?k=... — search results parser
│   ├── parse_bestseller.go  # /bestsellers, /new-releases, /most-wished-for parser
│   ├── parse_review.go      # /product-reviews/<ASIN> — paginated reviews parser
│   ├── parse_qa.go          # /ask/<ASIN> — Q&A pages parser
│   ├── parse_seller.go      # /sp?seller=<id> — seller profile parser
│   │
│   ├── task_product.go      # ProductTask — fetches product, enqueues reviews+Q&A at priority 1
│   ├── task_brand.go        # BrandTask
│   ├── task_author.go       # AuthorTask
│   ├── task_category.go     # CategoryTask — enqueues child browse nodes + products
│   ├── task_search.go       # SearchTask — stores search snapshot, enqueues capped product set
│   ├── task_bestseller.go   # BestsellerTask — enqueues products at priority 10
│   ├── task_review.go       # ReviewTask — exhausts all review pages for an ASIN
│   ├── task_qa.go           # QATask — exhausts all Q&A pages for an ASIN
│   ├── task_seller.go       # SellerTask
│   └── task_crawl.go        # CrawlTask — queue-driven dispatcher, priority tiers
│
└── cli/amazon.go            # All CLI subcommands under "amazon"
```

## Data Model

### amazon.duckdb

Driver: `github.com/duckdb/duckdb-go/v2` (matches `pkg/scrape/goodread/db.go`).

**Array columns**: store as JSON text (VARCHAR), same approach as goodread. Helper functions `encodeStringSlice([]string) string` and `decodeStringSlice(string) []string` in `db.go`.

**Upserts**: `INSERT OR REPLACE INTO` (DuckDB SQLite-compatible syntax).

**Migrations**: `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` block in `runMigrations()`.

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version     INTEGER PRIMARY KEY,
  applied_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products (core entity)
CREATE TABLE IF NOT EXISTS products (
  asin              VARCHAR PRIMARY KEY,
  title             VARCHAR,
  brand             VARCHAR,
  brand_id          VARCHAR,
  price             DOUBLE,
  currency          VARCHAR,
  list_price        DOUBLE,
  rating            DOUBLE,
  ratings_count     BIGINT,
  reviews_count     BIGINT,
  answered_qs       INTEGER,
  availability      VARCHAR,
  description       VARCHAR,
  bullet_points     VARCHAR,   -- JSON: ["point1","point2",...]
  specs             VARCHAR,   -- JSON: {"Key":"Value",...}
  images            VARCHAR,   -- JSON: ["url1","url2",...]
  category_path     VARCHAR,   -- JSON: ["Electronics","Cameras","..."]
  browse_node_ids   VARCHAR,   -- JSON: ["123456","789012"]
  seller_id         VARCHAR,
  seller_name       VARCHAR,
  sold_by           VARCHAR,
  fulfilled_by      VARCHAR,
  variant_asins     VARCHAR,   -- JSON
  parent_asin       VARCHAR,
  similar_asins     VARCHAR,   -- JSON
  rank              INTEGER,
  rank_category     VARCHAR,
  url               VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS brands (
  brand_id          VARCHAR PRIMARY KEY,
  name              VARCHAR,
  description       VARCHAR,
  logo_url          VARCHAR,
  banner_url        VARCHAR,
  follower_count    INTEGER,
  url               VARCHAR,
  featured_asins    VARCHAR,   -- JSON
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS authors (
  author_id         VARCHAR PRIMARY KEY,  -- slug from URL
  name              VARCHAR,
  bio               VARCHAR,
  photo_url         VARCHAR,
  website           VARCHAR,
  twitter           VARCHAR,
  book_asins        VARCHAR,   -- JSON
  follower_count    INTEGER,
  url               VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
  node_id           VARCHAR PRIMARY KEY,
  name              VARCHAR,
  parent_node_id    VARCHAR,
  breadcrumb        VARCHAR,   -- JSON: ["Electronics","Cameras"]
  child_node_ids    VARCHAR,   -- JSON
  top_asins         VARCHAR,   -- JSON
  url               VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bestseller_lists (
  list_id           VARCHAR PRIMARY KEY,  -- e.g. "bestsellers/electronics/2026-03-12"
  list_type         VARCHAR,   -- bestsellers|new-releases|most-wished-for|movers-and-shakers
  category          VARCHAR,
  node_id           VARCHAR,
  snapshot_date     DATE,
  url               VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bestseller_entries (
  list_id           VARCHAR NOT NULL,
  asin              VARCHAR NOT NULL,
  rank              INTEGER,
  title             VARCHAR,
  price             DOUBLE,
  rating            DOUBLE,
  ratings_count     BIGINT,
  PRIMARY KEY (list_id, asin)
);

CREATE TABLE IF NOT EXISTS reviews (
  review_id         VARCHAR PRIMARY KEY,
  asin              VARCHAR,
  reviewer_id       VARCHAR,
  reviewer_name     VARCHAR,
  rating            INTEGER,
  title             VARCHAR,
  text              VARCHAR,
  date_posted       TIMESTAMP,
  verified_purchase BOOLEAN,
  helpful_votes     INTEGER,
  total_votes       INTEGER,
  images            VARCHAR,   -- JSON: review image URLs
  variant_attrs     VARCHAR,   -- JSON: {"Color":"Black","Size":"XL"}
  url               VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS qa (
  qa_id             VARCHAR PRIMARY KEY,
  asin              VARCHAR,
  question          VARCHAR,
  question_by       VARCHAR,
  question_date     TIMESTAMP,
  answer            VARCHAR,
  answer_by         VARCHAR,
  answer_date       TIMESTAMP,
  helpful_votes     INTEGER,
  is_seller_answer  BOOLEAN,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sellers (
  seller_id         VARCHAR PRIMARY KEY,
  name              VARCHAR,
  rating            DOUBLE,
  rating_count      INTEGER,
  positive_pct      DOUBLE,
  neutral_pct       DOUBLE,
  negative_pct      DOUBLE,
  url               VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS search_results (
  search_id         VARCHAR PRIMARY KEY,  -- hash of query+page
  query             VARCHAR,
  page              INTEGER,
  result_asins      VARCHAR,   -- JSON: ordered list
  total_results     VARCHAR,   -- display string, e.g. "over 2,000 results" — not parsed as int
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### state.duckdb

Same schema as `pkg/scrape/goodread/state.go`:

```sql
CREATE SEQUENCE IF NOT EXISTS queue_id_seq;

CREATE TABLE IF NOT EXISTS queue (
  id           BIGINT DEFAULT nextval('queue_id_seq') PRIMARY KEY,
  url          VARCHAR UNIQUE NOT NULL,
  entity_type  VARCHAR NOT NULL,  -- product|brand|author|category|search|bestseller|review|qa|seller
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
| 10 | product, seller |
| 8  | bestseller |
| 5  | category, search, brand, author |
| 1  | review, qa |

`Pop` uses atomic claim:

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

## HTTP Client Strategy

1. **Plain HTTP first**: `net/http` with rotating browser User-Agents, realistic `Accept`/`Accept-Language`/`Referer` headers (previous Amazon page as referer), default delay 3s.
2. **Rod fallback**: on 503, 429, or CAPTCHA detected (`/errors/validateCaptcha` in response body or redirect). Rod browser is lazy-initialized.
3. **Rate limiting**: token bucket (1 req/delay). On 429: parse `Retry-After`, back off, surface wait in display.
4. **Retry**: up to 3 attempts per URL. 404 → mark `failed`, no retry.
5. **User-Agents**: 6–8 real Chrome/Firefox strings (desktop + mobile mix; Amazon sometimes serves different HTML for mobile).

```go
type Client struct {
    http       *http.Client
    rodBrowser *rod.Browser  // lazy-init
    delay      time.Duration
    userAgents []string
    mu         sync.Mutex
}

func (c *Client) Fetch(ctx context.Context, url string) ([]byte, int, error)
func (c *Client) FetchHTML(ctx context.Context, url string) (*goquery.Document, error)
```

CAPTCHA detection: check if response URL contains `/errors/validateCaptcha` or response body contains `<title>Robot Check</title>` or `<form action="/errors/validateCaptcha"`.

## core.Task Wiring

```go
// pkg/core.Task[State, Metric] interface:
//   Run(ctx context.Context, emit func(*State)) (Metric, error)
```

Compile-time interface assertion required in every task file.

### Entity Tasks

| Task | State fields | Metric fields |
|------|-------------|---------------|
| ProductTask | `URL, Status, Error string` | `Fetched, Skipped, Failed int` |
| BrandTask | `URL, Status, Error string` | `Fetched, Skipped, Failed int` |
| AuthorTask | `URL, Status, Error string` | `Fetched, Skipped, Failed int` |
| CategoryTask | `URL, Status, Error string; ProductsFound int` | `Fetched, Skipped, Failed int` |
| SearchTask | `URL, Query, Status, Error string; ResultsFound int` | `Fetched, Skipped, Failed int` |
| BestsellerTask | `URL, Status, Error string; EntriesFound int` | `Fetched, Skipped, Failed int` |
| ReviewTask | `URL, ASIN, Status, Error string; Pages, ReviewsFound int` | `Fetched, Skipped, Failed, Pages int` |
| QATask | `URL, ASIN, Status, Error string; Pages, QAsFound int` | `Fetched, Skipped, Failed, Pages int` |
| SellerTask | `URL, Status, Error string` | `Fetched, Skipped, Failed int` |

### ProductTask (canonical example)

```go
type ProductState  struct { URL string; Status string; Error string }
type ProductMetric struct { Fetched, Skipped, Failed int }

type ProductTask struct {
    URL      string
    Client   *Client
    DB       *DB
    StateDB  *State
    MaxPages int  // for review/QA pagination; 0 = unlimited
}

var _ core.Task[ProductState, ProductMetric] = (*ProductTask)(nil)

func (t *ProductTask) Run(ctx context.Context, emit func(*ProductState)) (ProductMetric, error)
```

**ProductTask.Run flow:**
1. Call `StateDB.IsVisited(url)` — if true, return with `Skipped++`
2. Fetch + parse product page
3. Call `DB.UpsertProduct`
4. Call `StateDB.Done(url, EntityProduct, statusCode)`
5. Call `StateDB.EnqueueBatch` with all discovered links:
   - `seller_id` → entity_type `seller`, priority 10
   - `brand_id` → entity_type `brand`, priority 5
   - each `similar_asin` → entity_type `product`, priority 10
   - each `variant_asin` → entity_type `product`, priority 10
   - `/product-reviews/<ASIN>` → entity_type `review`, priority 1
   - `/ask/<ASIN>` → entity_type `qa`, priority 1

All other entity tasks follow the same `IsVisited` → fetch → upsert → `Done` → enqueue pattern.

### ReviewTask

```go
type ReviewState  struct { URL, ASIN, Status, Error string; Pages, ReviewsFound int }
type ReviewMetric struct { Fetched, Skipped, Failed, Pages int }

type ReviewTask struct {
    URL      string
    ASIN     string
    Client   *Client
    DB       *DB
    StateDB  *State
    MaxPages int  // 0 = unlimited
}

var _ core.Task[ReviewState, ReviewMetric] = (*ReviewTask)(nil)
```

Pagination: fetch page, parse reviews, check for `li.a-last:not(.a-disabled)` next-page link, loop until none or MaxPages hit. Each page: `sortBy=recent&pageNumber=N`.

### QATask

Same pagination pattern as ReviewTask, URL: `/ask/<ASIN>?pageNumber=N`.

### CrawlTask

```go
type CrawlState struct {
    Done, Pending, Failed int64
    InFlight              []string
    RPS                   float64
}
type CrawlMetric struct { Done, Failed int64; Duration time.Duration }

type CrawlTask struct {
    Config  Config
    Client  *Client
    DB      *DB
    StateDB *State
}

var _ core.Task[CrawlState, CrawlMetric] = (*CrawlTask)(nil)
```

Dispatch table:

```go
switch item.EntityType {
case "product":   task = &ProductTask{...}
case "brand":     task = &BrandTask{...}
case "author":    task = &AuthorTask{...}
case "category":  task = &CategoryTask{...}
case "search":    task = &SearchTask{...}
case "bestseller": task = &BestsellerTask{...}
case "review":    task = &ReviewTask{...}
case "qa":        task = &QATask{...}
case "seller":    task = &SellerTask{...}
}
```

## Parse File Notes

### parse_product.go

Amazon product pages embed structured data in multiple places (priority order):
1. `<script type="application/ld+json">` — title, description, image, brand, rating, price (most reliable)
2. `#productTitle` — title fallback
3. `#priceblock_ourprice`, `.a-price .a-offscreen` — price
4. `#feature-bullets ul li` — bullet points
5. `#productDescription`, `#aplus` — description fallback
6. `#detailBullets_feature_div`, `#productDetails_techSpec_section_1` — specs table
7. `#imageBlock` data attributes — image gallery JSON
8. `#variation_color_name`, `#variation_size_name` — variant selectors → extract ASINs from `data-defaultasin`
9. `#SalesRank` or `#detailBulletsWrapper_feature_div` — BSR rank + category
10. `#bylineInfo` — brand/author link
11. `#sellerProfileTriggerId` — seller name + ID from link href
12. `#acrCustomerReviewText` — review count
13. `#askATFLink` — answered questions count
14. `#similarity-widget`, `#sp_detail` — similar items

Export: `func ParseProduct(doc *goquery.Document, asin string) (*Product, error)`

### parse_review.go

Review page URL: `/product-reviews/<ASIN>?sortBy=recent&pageNumber=N`

- `#cm_cr-review_list div[data-hook="review"]` — each review
  - `[data-hook="review-title"] span` — title
  - `[data-hook="review-body"] span` — text
  - `[data-hook="rating-out-of-five"]` — rating from class `a-star-N`
  - `[data-hook="avp-badge"]` — verified purchase
  - `[data-hook="helpful-vote-statement"]` — helpful votes
  - `[data-hook="review-date"]` — date
  - `[data-hook="review-author"] a` — reviewer name + profile link (extract ID)
  - `.review-image-tile` — review images
- Next page: `li.a-last:not(.a-disabled) a` href

Export: `func ParseReviews(doc *goquery.Document) ([]Review, string, error)` — returns reviews + next page URL

### parse_qa.go

URL: `/ask/<ASIN>?pageNumber=N`

Amazon's Q&A selectors are internal rendering artifacts that change with redesigns. Use these selectors in priority order; log raw HTML on parse failure to aid debugging selector drift:

Primary selectors (current as of 2026):
- Question container: `[id^="question-"]` (more stable than `cel_widget_id`)
- Question text: `[id^="question-"] .a-size-base`
- Answer container: `[id^="answer-"]`
- Answer text: `[id^="answer-"] .a-size-base`
- Author name: `.a-profile-name`
- Date: `.a-size-base.a-color-tertiary`
- Seller answer badge: `[data-hook="askSeller"]`

Fallback selectors (use if primary returns 0 results):
- `div[cel_widget_id^="QA"]` — question container
- `.a-declarative[data-action="askATFLink"]` — question text
- `span[cel_widget_id^="answer"]` — answer text

On parse failure: `log.Printf("parse_qa: 0 items parsed from %s, dumping HTML snippet", url)` then log first 2KB.

Export: `func ParseQA(doc *goquery.Document) ([]QA, string, error)` — returns QAs + next page URL

### parse_bestseller.go

URL patterns: `/bestsellers/<category>`, `/new-releases/<category>`, `/most-wished-for/<category>`, `/movers-and-shakers/<category>`

Entity constant: `EntityBestseller = "bestseller"`. CLI `--type` flag values: `bestsellers`, `new-releases`, `most-wished-for`, `movers-and-shakers` (use hyphens throughout; `movers-and-shakers` is the actual Amazon URL path).

- `#zg-ordered-list li` — each entry
  - `.zg-badge-text` — rank
  - `.a-link-normal[href*="/dp/"]` — ASIN from href
  - `.p13n-sc-truncate-desktop-type2` — title
  - `.a-price .a-offscreen` — price
  - `.a-icon-star-small` — rating
- list_id: `"{list_type}/{category}/{YYYY-MM-DD}"`
- snapshot_date: today

Export: `func ParseBestseller(doc *goquery.Document, listType, category, nodeID string) (*BestsellerList, []BestsellerEntry, error)`

### parse_search.go

URL: `/s?k=<query>&page=N`

- `[data-component-type="s-search-result"]` — each result
  - `data-asin` attribute — ASIN
  - `h2 a.a-link-normal` — title + URL
  - `.a-price .a-offscreen` — price
  - `.a-icon-star-small` — rating
- `span[data-component-type="s-result-info-bar"]` — total results text
- Next page: `.s-pagination-next:not(.s-pagination-disabled)` href

Export: `func ParseSearch(doc *goquery.Document) ([]SearchResult, string, string, error)` — results, next page URL, total results string

### parse_category.go

URL: `/b?node=<node_id>`

- `#nav-subnav .nav-a` — subcategory links (extract node_id from href `?node=`)
- `#zg_listTitle` or breadcrumb `#wayfinding-breadcrumbs_feature_div li` — name + breadcrumb
- Featured/top products: `[data-component-type="s-search-result"]` same as search

Export: `func ParseCategory(doc *goquery.Document, url string) (*Category, error)`

### parse_brand.go

URL: `/stores/<brand-slug>/page/...`

- `meta[name="title"]` or `h1` — brand name
- `.store-description` or JSON-LD — description
- Product cards: `.a-link-normal[href*="/dp/"]` — featured ASINs
- Follower count: not always publicly visible; attempt `.followers-count`. If selector is absent, set `follower_count = 0` — do not treat as parse error.

Export: `func ParseBrand(doc *goquery.Document, url string) (*Brand, error)`

### parse_author.go

URL: `/author/<slug>` (Amazon Author Central)

- `#ap_author_name` — name
- `#ap_author_bio` — bio
- `#ap_author_image` — photo
- `#author-book-list-template-1 .a-link-normal[href*="/dp/"]` — book ASINs
- `#ap_author_website` — website
- `#ap_author_twitter` — twitter handle

Export: `func ParseAuthor(doc *goquery.Document, url string) (*Author, error)`

### parse_seller.go

URL: `/sp?seller=<seller_id>` or `/gp/aag/main?seller=<seller_id>`

- `#sellerName` — name
- `#effective-timeframe-12month .a-text-bold` — rating
- `.feedback-detail-list` — positive/neutral/negative percentages
- `.total-ratings-count` — total rating count

Export: `func ParseSeller(doc *goquery.Document, url string) (*Seller, error)`

## File-by-File Notes

### types.go
All domain structs matching the DB schema. Arrays as `[]string` in Go. `time.Time` for timestamps. All fields exported with `json` tags.

Additional helper types:
```go
const BaseURL = "https://www.amazon.com"

// ASIN extraction helper used across parse files
func ExtractASIN(url string) string  // extracts from /dp/<ASIN>/ patterns

// entity_type constants
const (
    EntityProduct    = "product"
    EntityBrand      = "brand"
    EntityAuthor     = "author"
    EntityCategory   = "category"
    EntitySearch     = "search"
    EntityBestseller = "bestseller"
    EntityReview     = "review"
    EntityQA         = "qa"
    EntitySeller     = "seller"
)
```

### config.go

```go
type Config struct {
    DataDir   string        // base dir; DBPath + StatePath derived from it if not overridden
    DBPath    string
    StatePath string
    Delay     time.Duration
    Timeout   time.Duration // per-request HTTP timeout; 0 = no timeout (not recommended)
    Workers   int
    MaxPages  int           // 0 = unlimited; applies to review + QA pagination
    ForceRod  bool
}

func DefaultConfig() Config {
    dataDir := filepath.Join(os.Getenv("HOME"), "data/amazon")
    return Config{
        DataDir:   dataDir,
        DBPath:    filepath.Join(dataDir, "amazon.duckdb"),
        StatePath: filepath.Join(dataDir, "state.duckdb"),
        Delay:     3 * time.Second,
        Timeout:   30 * time.Second,
        Workers:   2,
        MaxPages:  0,
    }
}
```

### db.go
- `DB` struct wrapping `*sql.DB`
- `OpenDB(path string) (*DB, error)` — creates dir, opens DuckDB, runs `initSchema()` + `runMigrations()`
- `Close() error`
- Upsert methods: `UpsertProduct`, `UpsertBrand`, `UpsertAuthor`, `UpsertCategory`, `UpsertBestsellerList`, `UpsertReview`, `UpsertQA`, `UpsertSeller`, `UpsertSearchResult`
- `InsertBestsellerEntries(entries []BestsellerEntry) error` — wrapped in a transaction (`BEGIN`/`COMMIT`, `defer tx.Rollback()`); uses `INSERT OR REPLACE INTO bestseller_entries`
- `encodeStringSlice([]string) string` / `decodeStringSlice(string) []string`
- `encodeMap(map[string]string) string` for specs field

### state.go
Identical structure to `pkg/scrape/goodread/state.go`:
- `State` wrapping `*sql.DB`
- `OpenState(path string) (*State, error)`
- `Enqueue(url, entityType string, priority int) error` — `INSERT OR IGNORE`
- `EnqueueBatch(items []QueueItem) error` — bulk insert in a single transaction; used when a task discovers many links at once (e.g. similar ASINs, variants)
- `IsVisited(url string) bool` — check `visited` table before fetching; used by all entity tasks to skip re-fetching on restart
- `Pop(n int) ([]QueueItem, error)` — atomic claim via `UPDATE ... RETURNING`
- `Done(url, entityType string, statusCode int) error` — marks `done` in queue, upserts into `visited` with entity_type
- `Fail(url, errMsg string) error` — mark failed if attempts >= 3, else reset to pending
- `QueueStats() (pending, inProgress, done, failed int64)` — no error return (matches goodread pattern)
- `CreateJob(id, name, jobType string) error`
- `UpdateJob(id, status string, stats any) error`
- `ListJobs(limit int) ([]Job, error)`
- `ListQueue(status string, limit int) ([]QueueItem, error)`
- `Close() error`

### display.go
Lipgloss-based progress: queue depth, done/failed counts, in-flight URLs (one per worker), requests/sec. Pattern from `pkg/scrape/goodread/display.go` and `pkg/scrape/display.go`.

```go
func PrintCrawlProgress(s *CrawlState)
func PrintStats(db *DB, stateDB *State) error
```

## CLI Commands

Register with `root.AddCommand(NewAmazon())` in `cli/root.go`.

```
# Single entity fetch
search amazon product    <ASIN|url>
search amazon brand      <slug|url>
search amazon author     <slug|url>
search amazon category   <node_id|url>
search amazon seller     <seller_id|url>

# Paginated / multi-result
search amazon search     <query>        [--max-results 100] [--page 1]
search amazon bestsellers               [--category electronics]
                                        [--type bestsellers|new-releases|most-wished-for|movers-and-shakers]
search amazon reviews    <ASIN>         [--max-pages 0]
search amazon qa         <ASIN>         [--max-pages 0]

# Seed queue from file (one ASIN or URL per line)
search amazon seed --file asins.txt     [--entity product] [--priority 10]

# Bulk crawl
search amazon crawl                     [--workers 2] [--delay 3000] [--max-pages 0]

# Inspection
search amazon info
search amazon jobs
search amazon queue                     [--status pending|failed|done] [--limit 20]

# Global flags (all subcommands)
  --db        path to amazon.duckdb   (default $HOME/data/amazon/amazon.duckdb)
  --state     path to state.duckdb    (default $HOME/data/amazon/state.duckdb)
  --delay     ms between requests     (default 3000)
  --rod       force rod for all fetches
  --max-pages max review/QA pages     (default 0 = unlimited)
```

### cli/amazon.go structure (mirrors cli/goodread.go)

```go
func NewAmazon() *cobra.Command  // parent command
func newAmazonProduct() *cobra.Command
func newAmazonBrand() *cobra.Command
func newAmazonAuthor() *cobra.Command
func newAmazonCategory() *cobra.Command
func newAmazonSeller() *cobra.Command
func newAmazonSearch() *cobra.Command
func newAmazonBestsellers() *cobra.Command
func newAmazonReviews() *cobra.Command
func newAmazonQA() *cobra.Command
func newAmazonSeed() *cobra.Command      // reads --file, enqueues each line as given entity_type
func newAmazonCrawl() *cobra.Command
func newAmazonInfo() *cobra.Command
func newAmazonJobs() *cobra.Command
func newAmazonQueue() *cobra.Command

// Shared helpers
func addAmazonFlags(cmd *cobra.Command, dbPath, statePath *string, delay, maxPages *int)
func openAmazonDBs(dbPath, statePath string, delay int) (*amazon.DB, *amazon.State, *amazon.Client, error)
func normalizeProductURL(s string) string  // ASIN → full URL
func normalizeSellerURL(s string) string
```

## ASIN URL Normalization

```go
// normalizeProductURL: accepts bare ASIN (10 chars) or full URL
func normalizeProductURL(s string) string {
    if strings.HasPrefix(s, "http") {
        return s
    }
    return BaseURL + "/dp/" + s
}

// normalizeSellerURL: accepts seller ID or full URL (/sp?seller=<id> or /gp/aag/main?seller=<id>)
func normalizeSellerURL(s string) string {
    if strings.HasPrefix(s, "http") {
        return s
    }
    return BaseURL + "/sp?seller=" + s
}
```

## Link-Following Rules (CrawlTask)

| Source entity | Auto-enqueued | Priority |
|---|---|---|
| product | similar_asins (product), variant_asins (product) | 10 |
| product | seller | 10 |
| product | brand | 5 |
| product | /product-reviews/<ASIN> | 1 |
| product | /ask/<ASIN> | 1 |
| category | subcategory nodes | 5 |
| category | top products on page | 10 |
| bestseller | each listed product | 10 |
| search | each result product | 10 |
| brand | featured ASINs | 10 |
| author | book ASINs | 10 |

## Deployment

- Build: `make build-linux-noble` → deploy to server
- Data dir: `$HOME/data/amazon/`
- Local test: `search amazon product B08N5WRWNW` (Echo Dot)
- Verify: `duckdb $HOME/data/amazon/amazon.duckdb "SELECT asin, title, rating FROM products LIMIT 5"`

## Key Lessons (inherited from project patterns)

- Store arrays as JSON text (not `VARCHAR[]`) — Go driver impedance.
- `INSERT OR REPLACE INTO` for upserts; DuckDB supports SQLite-compatible syntax.
- DuckDB single-connection per file — do not open same file from two goroutines concurrently.
- Rod: lazy-init — don't launch Chrome unless a fetch actually fails.
- Amazon CAPTCHA check: `/errors/validateCaptcha` in response URL OR `<title>Robot Check</title>` in body.
- Default delay 3s — Amazon bans faster scrapers more aggressively than Goodreads.
- `queue.Pop` must use `UPDATE ... RETURNING` for atomic claim under concurrent workers.
- Schema migrations: always `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` rather than dropping DB.
- `state.duckdb` queue uses `INSERT OR IGNORE` for `Enqueue` to silently skip duplicate URLs.
- ASIN is 10 characters, always alphanumeric — use as primary key across all product-related tables.
- list_id for bestseller snapshots: `"{list_type}/{category}/{YYYY-MM-DD}"` — allows daily snapshot diffs.
