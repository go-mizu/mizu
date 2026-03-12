# Domains Tab Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Domains tab to the FTS dashboard that lists all domains with URL counts, and a domain detail page showing paginated/sortable URLs for each domain.

**Architecture:** A new `DomainStore` backed by `domains.duckdb` aggregates `doc_records` from all per-shard `.meta.duckdb` files. On each API call it syncs stale shards idempotently (delete+reinsert per changed shard). Two new dashboard routes serve domain list and domain detail. A new `domains.js` frontend file handles both views.

**Tech Stack:** Go, DuckDB (duckdb-go/v2), Vanilla JS (no framework), Tailwind CSS (existing classes)

---

### Task 1: Backend — DomainStore

**Files:**
- Create: `pkg/index/web/domain_store.go`

This is the core data layer. No tests needed (DuckDB integration is already tested by doc_store patterns).

**Step 1: Create `domain_store.go`**

```go
package web

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DomainStore maintains a cross-shard DuckDB cache of domain→doc mappings.
// Cache file: {warcMdBase}/domains.duckdb
// Idempotent: syncs only shards whose last_scanned_at changed.
type DomainStore struct {
	warcMdBase string
	mu         sync.Mutex
	db         *sql.DB
}

// NewDomainStore creates a DomainStore. Call EnsureFresh before querying.
func NewDomainStore(warcMdBase string) *DomainStore {
	return &DomainStore{warcMdBase: warcMdBase}
}

func (ds *DomainStore) dbPath() string {
	return filepath.Join(ds.warcMdBase, "domains.duckdb")
}

func (ds *DomainStore) openDB() (*sql.DB, error) {
	db, err := sql.Open("duckdb", ds.dbPath())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func (ds *DomainStore) initSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS domain_shard_versions (
			shard           TEXT PRIMARY KEY,
			last_scanned_at TEXT NOT NULL DEFAULT ''
		);
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
	`)
	return err
}

// EnsureFresh syncs any stale or new shards into the cache.
// Shards whose .meta.duckdb no longer exists are removed from the cache.
func (ds *DomainStore) EnsureFresh(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.db == nil {
		db, err := ds.openDB()
		if err != nil {
			return fmt.Errorf("domain_store: open: %w", err)
		}
		if err := ds.initSchema(ctx, db); err != nil {
			db.Close()
			return fmt.Errorf("domain_store: schema: %w", err)
		}
		ds.db = db
	}

	// Find all shard meta DBs.
	entries, err := os.ReadDir(ds.warcMdBase)
	if err != nil {
		return nil // directory may not exist yet
	}
	presentShards := make(map[string]string) // shard → meta db path
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.duckdb") {
			continue
		}
		shard := strings.TrimSuffix(e.Name(), ".meta.duckdb")
		presentShards[shard] = filepath.Join(ds.warcMdBase, e.Name())
	}

	// Load known versions from cache.
	rows, err := ds.db.QueryContext(ctx, `SELECT shard, last_scanned_at FROM domain_shard_versions`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	knownVersions := make(map[string]string)
	for rows.Next() {
		var shard, ts string
		rows.Scan(&shard, &ts)
		knownVersions[shard] = ts
	}
	rows.Close()

	// Remove deleted shards.
	for shard := range knownVersions {
		if _, ok := presentShards[shard]; !ok {
			ds.db.ExecContext(ctx, `DELETE FROM domain_docs WHERE shard = ?`, shard)
			ds.db.ExecContext(ctx, `DELETE FROM domain_shard_versions WHERE shard = ?`, shard)
		}
	}

	// Sync new/changed shards.
	for shard, metaPath := range presentShards {
		lastScanned := ds.readShardLastScannedAt(ctx, metaPath)
		if lastScanned == "" {
			continue // not yet scanned
		}
		if knownVersions[shard] == lastScanned {
			continue // up to date
		}
		if err := ds.syncShard(ctx, shard, metaPath, lastScanned); err != nil {
			// Non-fatal: log and continue with other shards.
			_ = err
		}
	}
	return nil
}

func (ds *DomainStore) readShardLastScannedAt(ctx context.Context, metaPath string) string {
	db, err := sql.Open("duckdb", metaPath+"?access_mode=read_only")
	if err != nil {
		return ""
	}
	defer db.Close()
	var ts string
	db.QueryRowContext(ctx, `SELECT last_scanned_at FROM doc_scan_meta LIMIT 1`).Scan(&ts)
	return ts
}

func (ds *DomainStore) syncShard(ctx context.Context, shard, metaPath, lastScanned string) error {
	srcDB, err := sql.Open("duckdb", metaPath+"?access_mode=read_only")
	if err != nil {
		return err
	}
	defer srcDB.Close()

	rows, err := srcDB.QueryContext(ctx, `
		SELECT doc_id, url, host, title, crawl_date, size_bytes, word_count
		FROM doc_records
		WHERE host != ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type rec struct {
		docID, url, host, title, crawlDate string
		size                               int64
		words                              int
	}
	var records []rec
	for rows.Next() {
		var r rec
		rows.Scan(&r.docID, &r.url, &r.host, &r.title, &r.crawlDate, &r.size, &r.words)
		records = append(records, r)
	}
	rows.Close()

	// Replace shard data atomically (best-effort; no transaction across duckdb files).
	ds.db.ExecContext(ctx, `DELETE FROM domain_docs WHERE shard = ?`, shard)

	const batchSize = 500
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)*8)
		for j, r := range batch {
			placeholders[j] = "(?,?,?,?,?,?,?,?)"
			args = append(args, r.host, shard, r.docID, r.url, r.title, r.crawlDate, r.size, r.words)
		}
		query := `INSERT OR REPLACE INTO domain_docs (host,shard,doc_id,url,title,crawl_date,size_bytes,word_count) VALUES ` +
			strings.Join(placeholders, ",")
		ds.db.ExecContext(ctx, query, args...)
	}

	ds.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO domain_shard_versions (shard, last_scanned_at) VALUES (?, ?)`,
		shard, lastScanned,
	)
	return nil
}

// ── Query methods ─────────────────────────────────────────────────────────────

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

func (ds *DomainStore) ListDomains(ctx context.Context, sortBy, q string, page, pageSize int) (DomainsResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	orderClause := "ORDER BY cnt DESC, host ASC"
	if sortBy == "alpha" {
		orderClause = "ORDER BY host ASC"
	}

	whereClause := ""
	args := []any{}
	if q != "" {
		whereClause = "WHERE host ILIKE ?"
		args = append(args, "%"+q+"%")
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT host) FROM domain_docs %s
	`, whereClause)
	var total int64
	if err := ds.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return DomainsResponse{}, err
	}

	listQuery := fmt.Sprintf(`
		SELECT host, COUNT(*) AS cnt
		FROM domain_docs
		%s
		GROUP BY host
		%s
		LIMIT ? OFFSET ?
	`, whereClause, orderClause)
	listArgs := append(args, pageSize, offset)

	rows, err := ds.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return DomainsResponse{}, err
	}
	defer rows.Close()

	var domains []DomainRow
	for rows.Next() {
		var d DomainRow
		rows.Scan(&d.Domain, &d.Count)
		domains = append(domains, d)
	}
	if domains == nil {
		domains = []DomainRow{}
	}
	return DomainsResponse{Domains: domains, Total: total, Page: page, PageSize: pageSize}, nil
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

func (ds *DomainStore) ListDomainURLs(ctx context.Context, domain, sortBy string, page, pageSize int) (DomainDetailResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	orderClause := "ORDER BY crawl_date DESC"
	switch sortBy {
	case "size":
		orderClause = "ORDER BY size_bytes DESC"
	case "words":
		orderClause = "ORDER BY word_count DESC"
	case "title":
		orderClause = "ORDER BY title ASC"
	case "url":
		orderClause = "ORDER BY url ASC"
	}

	var total int64
	ds.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM domain_docs WHERE host = ?`, domain).Scan(&total)

	rows, err := ds.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT doc_id, shard, url, title, crawl_date, size_bytes, word_count
		FROM domain_docs
		WHERE host = ?
		%s
		LIMIT ? OFFSET ?
	`, orderClause), domain, pageSize, offset)
	if err != nil {
		return DomainDetailResponse{}, err
	}
	defer rows.Close()

	var docs []DomainDocRow
	for rows.Next() {
		var d DomainDocRow
		rows.Scan(&d.DocID, &d.Shard, &d.URL, &d.Title, &d.CrawlDate, &d.SizeBytes, &d.WordCount)
		docs = append(docs, d)
	}
	if docs == nil {
		docs = []DomainDocRow{}
	}
	return DomainDetailResponse{Domain: domain, Total: total, Page: page, PageSize: pageSize, Docs: docs}, nil
}

// Close releases the underlying DB connection.
func (ds *DomainStore) Close() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.db != nil {
		err := ds.db.Close()
		ds.db = nil
		return err
	}
	return nil
}
```

**Step 2: Verify it compiles**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/index/web/...
```
Expected: no errors.

**Step 3: Commit**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
git add pkg/index/web/domain_store.go
git commit -m "feat(fts/domains): add DomainStore with idempotent shard sync"
```

---

### Task 2: Backend — Server Handlers

**Files:**
- Modify: `pkg/index/web/server.go`

**Step 1: Add `DomainStore` field to Server struct**

Find the `Server` struct (around line 200). After the `Docs *DocStore` field, add:
```go
DomainStore *DomainStore // cross-shard domain cache (dashboard only)
```

**Step 2: Initialize DomainStore in NewDashboard**

Find where `s.Docs` is initialized in `NewDashboard` (around line 306). Add after it:
```go
s.DomainStore = NewDomainStore(s.WARCMdBase)
```

**Step 3: Register routes (dashboard-only block)**

After the `/api/browse/stats` route registration, add:
```go
router.Get("/api/domains", s.handleDomainList)
router.Get("/api/domains/{domain}", s.handleDomainDetail)
```

**Step 4: Add handler methods (append to server.go)**

```go
// handleDomainList returns paginated domain list across all shards.
func (s *Server) handleDomainList(c *mizu.Ctx) error {
	if s.DomainStore == nil {
		return c.JSON(503, errResp{"domain store not available"})
	}
	if err := s.DomainStore.EnsureFresh(c.Context()); err != nil {
		return c.JSON(500, errResp{err.Error()})
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 100)
	sort := c.Query("sort")
	q := c.Query("q")
	resp, err := s.DomainStore.ListDomains(c.Context(), sort, q, page, pageSize)
	if err != nil {
		return c.JSON(500, errResp{err.Error()})
	}
	return c.JSON(200, resp)
}

// handleDomainDetail returns paginated URL list for a single domain.
func (s *Server) handleDomainDetail(c *mizu.Ctx) error {
	if s.DomainStore == nil {
		return c.JSON(503, errResp{"domain store not available"})
	}
	domain := c.Param("domain")
	if domain == "" {
		return c.JSON(400, errResp{"domain required"})
	}
	if err := s.DomainStore.EnsureFresh(c.Context()); err != nil {
		return c.JSON(500, errResp{err.Error()})
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 100)
	sort := c.Query("sort")
	resp, err := s.DomainStore.ListDomainURLs(c.Context(), domain, sort, page, pageSize)
	if err != nil {
		return c.JSON(500, errResp{err.Error()})
	}
	return c.JSON(200, resp)
}
```

**Step 5: Check if `parseIntQuery` already exists**

```bash
grep -n "parseIntQuery" /Users/apple/github/go-mizu/mizu/blueprints/search/pkg/index/web/server.go | head -5
```

If not found, add to server.go:
```go
func parseIntQuery(c *mizu.Ctx, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return def
	}
	return n
}
```
Check if `strconv` is already imported; add it if missing.

**Step 6: Build to verify**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/index/web/...
```
Expected: no errors.

**Step 7: Commit**

```bash
git add pkg/index/web/server.go
git commit -m "feat(fts/domains): add /api/domains and /api/domains/{domain} handlers"
```

---

### Task 3: Frontend — API helpers

**Files:**
- Modify: `pkg/index/web/static/js/api.js`

**Step 1: Append to `api.js`**

At the end of `api.js`, add:
```js
async function apiDomains(opts = {}) {
  const p = new URLSearchParams();
  if (opts.sort) p.set('sort', opts.sort);
  if (opts.page) p.set('page', String(opts.page));
  if (opts.pageSize) p.set('page_size', String(opts.pageSize));
  if (opts.q) p.set('q', opts.q);
  const qs = p.toString();
  return apiFetch('/api/domains' + (qs ? '?' + qs : ''));
}

async function apiDomainDetail(domain, opts = {}) {
  const p = new URLSearchParams();
  if (opts.sort) p.set('sort', opts.sort);
  if (opts.page) p.set('page', String(opts.page));
  if (opts.pageSize) p.set('page_size', String(opts.pageSize));
  const qs = p.toString();
  return apiFetch('/api/domains/' + encodeURIComponent(domain) + (qs ? '?' + qs : ''));
}
```

**Step 2: Commit**

```bash
git add pkg/index/web/static/js/api.js
git commit -m "feat(fts/domains): add apiDomains and apiDomainDetail helpers"
```

---

### Task 4: Frontend — Nav Tab + Router

**Files:**
- Modify: `pkg/index/web/static/index.html`
- Modify: `pkg/index/web/static/js/router.js`

**Step 1: Add nav tab to `index.html`**

Find the nav section in `index.html` (around line 280-286). Add the Domains tab after Browse and before WARC:
```html
<a href="#/domains" data-tab="domains" class="text-sm pb-1 tab-inactive transition-colors cursor-pointer">Domains</a>
```
Insert it so the order is: Overview | Search | Browse | Domains | WARC | Parquet | Jobs

Also add `domains.js` script tag. Find where other JS files are included (near end of `<body>`):
```html
<script src="/static/js/domains.js"></script>
```
Add it after `browse.js`.

**Step 2: Update `router.js` — add route handling**

In the `route()` function, after the `/browse` block and before the `/doc/` block, add:
```js
} else if (path === '/domains' && isDashboard) {
  showHeaderSearch(false);
  renderDomains();
} else if (path.startsWith('/domains/') && isDashboard) {
  showHeaderSearch(false);
  const domain = decodeURIComponent(path.slice('/domains/'.length));
  renderDomainDetail(domain);
```

In `updateActiveTab()`, add the domains case:
```js
else if (path === '/domains' || path.startsWith('/domains/')) activeTab = 'domains';
```

**Step 3: Commit**

```bash
git add pkg/index/web/static/index.html pkg/index/web/static/js/router.js
git commit -m "feat(fts/domains): add Domains nav tab and router entries"
```

---

### Task 5: Frontend — domains.js

**Files:**
- Create: `pkg/index/web/static/js/domains.js`

**Step 1: Create `domains.js`**

```js
// ===================================================================
// Tab: Domains
// ===================================================================

let domainsFilterTimer = null;

async function renderDomains() {
  state.currentPage = 'domains';
  state.domainPage = state.domainPage || 1;
  state.domainSort = state.domainSort || 'count';
  state.domainQ = state.domainQ || '';

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <h1 class="page-title">Domains</h1>
      </div>
      <div class="surface p-4">
        <div id="domains-content">${renderTableSkeleton(6)}</div>
      </div>
    </div>`;

  await loadDomains();
}

async function loadDomains(page) {
  if (page !== undefined) state.domainPage = page;
  const el = $('domains-content');
  if (!el) return;

  try {
    const data = await apiDomains({
      sort: state.domainSort,
      page: state.domainPage,
      q: state.domainQ,
    });
    if (state.currentPage !== 'domains') return;
    renderDomainsTable(data);
  } catch(e) {
    if (el) el.innerHTML = `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

function renderDomainsTable(data) {
  const el = $('domains-content');
  if (!el) return;

  const domains = data.domains || [];
  const total = data.total || 0;
  const page = data.page || 1;
  const pageSize = data.page_size || 100;
  const totalPages = Math.ceil(total / pageSize);
  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);

  const maxCount = domains.reduce((m, d) => Math.max(m, d.count || 0), 0) || 1;

  el.innerHTML = `
    <div class="flex items-center gap-3 mb-4 flex-wrap">
      <span class="meta-line">${total.toLocaleString()} domain${total !== 1 ? 's' : ''}</span>
      <input id="domains-filter" type="search" placeholder="Filter domains\u2026" value="${esc(state.domainQ || '')}"
        class="ui-input text-xs px-2 py-1 w-40 sm:w-56" oninput="debounceDomainFilter(this.value)">
      <select class="ui-input text-xs px-2 py-1 ml-auto" onchange="state.domainSort=this.value;loadDomains(1)">
        <option value="count" ${(state.domainSort||'count')==='count'?'selected':''}>Count \u2193</option>
        <option value="alpha" ${state.domainSort==='alpha'?'selected':''}>Domain A\u2013Z</option>
      </select>
    </div>
    ${domains.length === 0 ? `<div class="ui-empty">${state.domainQ ? 'No domains match filter.' : 'No domain data yet. Scan documents first.'}</div>` : `
    <div class="overflow-x-auto">
    <table class="w-full text-xs ui-table">
      <thead>
        <tr class="text-left">
          <th class="pb-2 pr-3 font-medium">Domain</th>
          <th class="pb-2 font-medium text-right">URLs</th>
        </tr>
      </thead>
      <tbody>
        ${domains.map((d, i) => `
          <tr class="file-row anim-fade-up" style="animation-delay:${Math.min(i,20)*10}ms">
            <td class="py-2 pr-3">
              <a href="#/domains/${encodeURIComponent(d.domain)}" class="ui-link font-mono font-medium">${esc(d.domain)}</a>
              <div class="mt-1 progress-track" style="height:3px">
                <div class="progress-fill" style="width:${Math.max(2,(d.count/maxCount)*100).toFixed(1)}%"></div>
              </div>
            </td>
            <td class="py-2 text-right font-mono ui-subtle whitespace-nowrap">${(d.count||0).toLocaleString()}</td>
          </tr>`).join('')}
      </tbody>
    </table>
    </div>
    ${totalPages > 1 ? `
    <div class="flex items-center justify-between mt-4 text-xs">
      <button onclick="loadDomains(${page - 1})" ${page <= 1 ? 'disabled' : ''} class="ui-btn px-3 py-1.5">&larr; Prev</button>
      <span class="ui-subtle">Page ${page} of ${totalPages}</span>
      <button onclick="loadDomains(${page + 1})" ${page >= totalPages ? 'disabled' : ''} class="ui-btn px-3 py-1.5">Next &rarr;</button>
    </div>` : ''}
    `}`;
}

function debounceDomainFilter(val) {
  state.domainQ = val;
  clearTimeout(domainsFilterTimer);
  domainsFilterTimer = setTimeout(() => loadDomains(1), 300);
}

// ── Domain Detail ─────────────────────────────────────────────────────────────

async function renderDomainDetail(domain) {
  state.currentPage = 'domain-detail';
  state.domainDetailDomain = domain;
  state.domainDetailPage = 1;
  state.domainDetailSort = state.domainDetailSort || 'date';

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <div class="flex items-center gap-2 text-xs font-mono ui-subtle mb-1">
          <a href="#/domains" class="ui-link" onclick="state.domainDetailPage=1">Domains</a>
          <span>/</span>
          <span class="font-medium" style="color:var(--text)">${esc(domain)}</span>
        </div>
        <h1 class="page-title">${esc(domain)}</h1>
      </div>
      <div class="surface p-4">
        <div id="domain-detail-content">${renderTableSkeleton(6)}</div>
      </div>
    </div>`;

  await loadDomainDetail();
}

async function loadDomainDetail(page) {
  if (page !== undefined) state.domainDetailPage = page;
  const el = $('domain-detail-content');
  if (!el) return;
  const domain = state.domainDetailDomain;
  if (!domain) return;

  try {
    const data = await apiDomainDetail(domain, {
      sort: state.domainDetailSort,
      page: state.domainDetailPage,
    });
    if (state.currentPage !== 'domain-detail') return;
    renderDomainDetailTable(data);
  } catch(e) {
    if (el) el.innerHTML = `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

function renderDomainDetailTable(data) {
  const el = $('domain-detail-content');
  if (!el) return;

  const docs = data.docs || [];
  const total = data.total || 0;
  const page = data.page || 1;
  const pageSize = data.page_size || 100;
  const totalPages = Math.ceil(total / pageSize);
  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);

  el.innerHTML = `
    <div class="flex items-center gap-3 mb-4 flex-wrap">
      <span class="meta-line">${start}\u2013${end} of ${total.toLocaleString()}</span>
      <select class="ui-input text-xs px-2 py-1 ml-auto" onchange="state.domainDetailSort=this.value;loadDomainDetail(1)">
        <option value="date" ${(state.domainDetailSort||'date')==='date'?'selected':''}>Date \u2193</option>
        <option value="size" ${state.domainDetailSort==='size'?'selected':''}>Size \u2193</option>
        <option value="words" ${state.domainDetailSort==='words'?'selected':''}>Words \u2193</option>
        <option value="title" ${state.domainDetailSort==='title'?'selected':''}>Title A\u2013Z</option>
        <option value="url" ${state.domainDetailSort==='url'?'selected':''}>URL A\u2013Z</option>
      </select>
    </div>
    ${docs.length === 0 ? `<div class="ui-empty">No documents found.</div>` : `
    <div class="overflow-x-auto">
    <table class="w-full text-xs ui-table">
      <thead>
        <tr class="text-left">
          <th class="pb-2 pr-3 font-medium">Title</th>
          <th class="pb-2 pr-3 font-medium hidden sm:table-cell">URL</th>
          <th class="pb-2 pr-3 font-medium text-right whitespace-nowrap">Date</th>
          <th class="pb-2 pr-3 font-medium text-right hidden sm:table-cell">Size</th>
          <th class="pb-2 font-medium text-right hidden md:table-cell">Words</th>
        </tr>
      </thead>
      <tbody>
        ${docs.map((d, i) => `
          <tr class="file-row anim-fade-up" style="animation-delay:${Math.min(i,20)*10}ms">
            <td class="py-2 pr-3">
              <a href="#/doc/${esc(d.shard)}/${encodeURIComponent(d.doc_id)}" class="ui-link font-medium truncate block max-w-[200px] sm:max-w-xs" title="${esc(d.title||d.doc_id)}">
                ${esc(d.title || d.doc_id)}
              </a>
            </td>
            <td class="py-2 pr-3 hidden sm:table-cell max-w-[240px]">
              ${d.url ? `<a href="${esc(d.url)}" target="_blank" rel="noopener noreferrer" class="ui-subtle hover:text-[var(--accent)] font-mono truncate block" title="${esc(d.url)}">${truncateURL(d.url, 40)}</a>` : ''}
            </td>
            <td class="py-2 pr-3 ui-subtle text-right whitespace-nowrap">${d.crawl_date ? fmtDate(d.crawl_date) : ''}</td>
            <td class="py-2 pr-3 ui-subtle text-right whitespace-nowrap hidden sm:table-cell">${d.size_bytes ? fmtBytes(d.size_bytes) : ''}</td>
            <td class="py-2 ui-subtle text-right whitespace-nowrap hidden md:table-cell">${d.word_count ? d.word_count.toLocaleString() : ''}</td>
          </tr>`).join('')}
      </tbody>
    </table>
    </div>
    ${totalPages > 1 ? `
    <div class="flex items-center justify-between mt-4 text-xs">
      <button onclick="loadDomainDetail(${page - 1})" ${page <= 1 ? 'disabled' : ''} class="ui-btn px-3 py-1.5">&larr; Prev</button>
      <span class="ui-subtle">Page ${page} of ${totalPages}</span>
      <button onclick="loadDomainDetail(${page + 1})" ${page >= totalPages ? 'disabled' : ''} class="ui-btn px-3 py-1.5">Next &rarr;</button>
    </div>` : ''}
    `}`;
}

function renderTableSkeleton(rows) {
  return `<div class="space-y-2">` +
    Array.from({length: rows}, () => `
      <div class="flex gap-3 py-2">
        <div class="h-3 w-40 ui-skeleton"></div>
        <div class="h-3 w-12 ui-skeleton ml-auto"></div>
      </div>`).join('') +
    `</div>`;
}
```

**Step 2: Commit**

```bash
git add pkg/index/web/static/js/domains.js
git commit -m "feat(fts/domains): add domains.js with list and detail views"
```

---

### Task 6: Build & Test Locally

**Step 1: Build and install**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
make install
```
Expected: binary installed to `$HOME/bin/search`.

**Step 2: Run dashboard locally**

```bash
search cc fts dashboard
```
Expected: dashboard starts on port 3456.

**Step 3: Verify in browser**

Open `http://localhost:3456`. Check:
- "Domains" tab appears in nav (dashboard mode only)
- Clicking it shows the domain list (may be empty if no docs scanned)
- Clicking a domain shows its URL list
- Sort dropdowns work
- Pagination appears when >100 domains or URLs

**Step 4: If nav tab is missing** — confirm `isDashboard` is true (dashboard mode started with hub). The tab in index.html uses `isDashboard` check in JS (router.js), not in HTML. The `<a>` tag is always in HTML but only routes if `isDashboard`.

Actually re-check: looking at the existing nav — WARC and Parquet tabs are in HTML unconditionally but their routes only work if `isDashboard`. The Domains tab should follow the same pattern.

---

### Task 7: Deploy to Server 2

**Step 1: Build noble binary**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
make build-linux-noble
```
Note: this takes 15-20 min under QEMU. Run with `run_in_background=true` and `timeout=1200000`.

**Step 2: Deploy**

```bash
make deploy-linux-noble SERVER=2
```

**Step 3: Restart on server 2 (inside screen)**

```bash
ssh server2 "screen -ls"
```
Find the FTS screen session name. Then:
```bash
ssh server2 "screen -S <session-name> -X stuff $'\003'"
# Wait 2s, then restart:
ssh server2 "screen -S <session-name> -X stuff 'search cc fts dashboard\n'"
```

Or if the session name is unknown, check with `screen -ls` and restart manually.

**Step 4: Verify on server 2**

SSH tunnel or direct access to the dashboard port and confirm Domains tab works.

---

### Task 8: Fingerprint Update for JS Asset

The server uses SHA-256 fingerprinting for JS files. After any JS change, the fingerprints must be regenerated so browsers pick up the new file.

**Step 1: Check if fingerprints are auto-generated at startup**

```bash
grep -n "fingerprint\|sha256\|hash" /Users/apple/github/go-mizu/mizu/blueprints/search/pkg/index/web/server.go | head -20
```

If fingerprints are computed at server startup from the embedded FS (likely), no action is needed — restarting the server regenerates them.

If they're hardcoded in a file, update that file after adding `domains.js`.

**Step 2: Verify in browser**

Open DevTools → Network tab → confirm `domains.js` is loaded with a `?v=` cache-busting param.
