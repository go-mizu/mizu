# spec/0695 — Domains Tab for FTS Dashboard

## Goal

Add a dedicated **Domains** tab to the `search cc fts dashboard` that:
1. Lists all domains across all indexed shards with their URL count
2. Clicking a domain navigates to a domain detail page listing all URLs in that domain
3. Both the domain list and domain URL list are sortable and paginated
4. Uses a DuckDB cache (`domains.duckdb`) for fast queries; cache is rebuilt idempotently

---

## Context

The dashboard already has per-shard metadata in `{warcMdBase}/{shard}.meta.duckdb` files (table `doc_records` with columns: `doc_id`, `url`, `host`, `title`, `crawl_date`, `size_bytes`, `word_count`).

The existing Browse tab's Stats view shows top 20 domains per shard via `GET /api/browse/stats?shard=`. The new Domains tab aggregates **all shards** cross-shard into one domain-centric view.

---

## Data Architecture

### Cache File

`{warcMdBase}/domains.duckdb` — a single DuckDB file with two tables:

```sql
-- Tracks which shard was imported and when
CREATE TABLE IF NOT EXISTS domain_shard_versions (
    shard           TEXT PRIMARY KEY,
    last_scanned_at TEXT NOT NULL DEFAULT ''
);

-- Denormalized domain→URL records from all shards
CREATE TABLE IF NOT EXISTS domain_docs (
    host        TEXT NOT NULL DEFAULT '',
    shard       TEXT NOT NULL DEFAULT '',
    doc_id      TEXT NOT NULL DEFAULT '',
    url         TEXT NOT NULL DEFAULT '',
    title       TEXT NOT NULL DEFAULT '',
    crawl_date  TEXT NOT NULL DEFAULT '',
    size_bytes  BIGINT DEFAULT 0,
    word_count  INTEGER DEFAULT 0,
    PRIMARY KEY (shard, doc_id)
);

CREATE INDEX IF NOT EXISTS idx_domain_docs_host ON domain_docs(host);
```

### Idempotent Sync (EnsureFresh)

Called before every API response. For each available shard meta DB:
1. Read `last_scanned_at` from `doc_scan_meta` in the shard's `.meta.duckdb`
2. Compare with `domain_shard_versions.last_scanned_at`
3. If different (or missing): `DELETE FROM domain_docs WHERE shard = ?` then bulk-INSERT from shard DB
4. Update `domain_shard_versions` row

Also removes shards from `domain_docs` / `domain_shard_versions` that no longer have a `.meta.duckdb` file (deleted shards).

If the shard meta DB doesn't exist yet (not yet scanned), skip it silently.

---

## API Endpoints

Both are dashboard-only (registered when `Hub != nil`).

### GET /api/domains

Query params:
- `sort` — `count` (default, descending) | `alpha` (domain A→Z)
- `page` — integer ≥ 1 (default 1)
- `page_size` — integer (default 100, max 500)
- `q` — domain prefix/substring filter

Response:
```json
{
  "domains": [
    {"domain": "example.com", "count": 42}
  ],
  "total": 1234,
  "page": 1,
  "page_size": 100
}
```

### GET /api/domains/{domain}

Query params:
- `sort` — `date` (default) | `size` | `words` | `url` | `title`
- `page` — integer ≥ 1 (default 1)
- `page_size` — integer (default 100, max 500)

Response:
```json
{
  "domain": "example.com",
  "total": 42,
  "page": 1,
  "page_size": 100,
  "docs": [
    {
      "doc_id": "...",
      "shard": "0001",
      "url": "https://...",
      "title": "...",
      "crawl_date": "...",
      "size_bytes": 12345,
      "word_count": 500
    }
  ]
}
```

---

## Frontend

### Nav Tab

Add "Domains" tab in `index.html` nav (dashboard-only, after Browse, before WARC):
```html
<a href="#/domains" data-tab="domains" class="text-sm pb-1 tab-inactive transition-colors cursor-pointer">Domains</a>
```

### Routes

In `router.js`:
- `/domains` → `renderDomains()`
- `/domains/{domain}` → `renderDomainDetail(domain)`

`updateActiveTab`: add `domains` case for `/domains` and `/domains/` paths.

### New File: `domains.js`

**`renderDomains()`** — domain list page:
- Searchable input (filter by domain name, debounced 300ms)
- Sort controls: "Count ↓" / "A→Z"
- Table: Domain | URL Count | bar chart (relative to max)
- Each row clickable → `#/domains/{domain}`
- Pagination

**`renderDomainDetail(domain)`** — single-domain URL list:
- Breadcrumb: Domains → {domain}
- Sort controls: Date ↓ / Size ↓ / Words ↓ / Title A→Z / URL A→Z
- Table: Title | URL | Date | Size | Words (reuses same style as Browse docs table)
- Each title links to `#/doc/{shard}/{docid}`
- Pagination

### New API helpers in `api.js`

```js
async function apiDomains(opts = {}) {
  const p = new URLSearchParams();
  if (opts.sort) p.set('sort', opts.sort);
  if (opts.page) p.set('page', String(opts.page));
  if (opts.q) p.set('q', opts.q);
  return apiFetch('/api/domains?' + p.toString());
}

async function apiDomainDetail(domain, opts = {}) {
  const p = new URLSearchParams();
  if (opts.sort) p.set('sort', opts.sort);
  if (opts.page) p.set('page', String(opts.page));
  return apiFetch('/api/domains/' + encodeURIComponent(domain) + '?' + p.toString());
}
```

---

## New Backend File: `domain_store.go`

```go
type DomainStore struct {
    warcMdBase string
    mu         sync.Mutex
    db         *sql.DB  // lazily opened
}

func NewDomainStore(warcMdBase string) *DomainStore

// EnsureFresh syncs stale/new shards into the cache DB.
// Acquires a mutex so concurrent requests don't double-sync.
func (ds *DomainStore) EnsureFresh(ctx context.Context) error

// ListDomains returns paginated domain rows.
func (ds *DomainStore) ListDomains(ctx context.Context, sort, q string, page, pageSize int) (DomainsResponse, error)

// ListDomainURLs returns paginated docs for one domain.
func (ds *DomainStore) ListDomainURLs(ctx context.Context, domain, sort string, page, pageSize int) (DomainDetailResponse, error)
```

Response types:
```go
type DomainRow struct {
    Domain string `json:"domain"`
    Count  int64  `json:"count"`
}
type DomainsResponse struct {
    Domains  []DomainRow `json:"domains"`
    Total    int64       `json:"total"`
    Page     int         `json:"page"`
    PageSize int         `json:"page_size"`
}
type DomainDocRow struct {
    DocID     string `json:"doc_id"`
    Shard     string `json:"shard"`
    URL       string `json:"url"`
    Title     string `json:"title"`
    CrawlDate string `json:"crawl_date"`
    SizeBytes int64  `json:"size_bytes"`
    WordCount int    `json:"word_count"`
}
type DomainDetailResponse struct {
    Domain   string         `json:"domain"`
    Total    int64          `json:"total"`
    Page     int            `json:"page"`
    PageSize int            `json:"page_size"`
    Docs     []DomainDocRow `json:"docs"`
}
```

---

## Server Changes (`server.go`)

Add `DomainStore *DomainStore` field to `Server`.

In `NewDashboard()`:
```go
s.DomainStore = NewDomainStore(s.WARCMdBase)
```

New handler registrations (dashboard-only):
```go
router.Get("/api/domains", s.handleDomainList)
router.Get("/api/domains/{domain}", s.handleDomainDetail)
```

Handlers call `s.DomainStore.EnsureFresh(ctx)` then delegate to `DomainStore`.

---

## Deploy

Local:
```bash
make install
search cc fts dashboard
```

Server 2 (inside screen):
```bash
make deploy-linux-noble SERVER=2
ssh server2 "screen -S fts -X stuff 'search cc fts dashboard\n'"
# or: ssh server2 "screen -r fts" and restart manually
```
