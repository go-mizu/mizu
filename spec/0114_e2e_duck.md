# 0114: E2E Tests for DuckDB Store and Web Server

**Status: COMPLETED**

## Overview

Add end-to-end tests for the finewiki blueprint that test the full stack with real Vietnamese (vi) wiki data:
1. `store/duckdb/store_e2e_test.go` - Test DuckDB store functions directly ✅
2. `app/web/server_e2e_test.go` - Test HTML rendering endpoints ✅
3. Move `/search` to `/` root (combine home and search) ✅
4. Fix template architecture (unique content templates) ✅

## Prerequisites

- Vietnamese wiki data imported: `finewiki import vi`
- Parquet files at `~/data/blueprint/finewiki/vi/data*.parquet`
- Run with: `E2E_TEST=1 go test -tags=e2e ./... -v`

---

## Part 1: store/duckdb/store_e2e_test.go

### File Location
`blueprints/finewiki/store/duckdb/store_e2e_test.go`

### Build Tag
```go
//go:build e2e
```

### Test Functions

#### 1. TestStore_Search_E2E
Test the Search function with real vi wiki data.

```go
func TestStore_Search_E2E(t *testing.T) {
    // Setup: create store with vi parquet data
    // Test cases:
    // - Search for "Vietnam" returns results
    // - Search with Vietnamese characters (e.g., "Việt Nam")
    // - Search with empty query returns empty results
    // - Search with WikiName filter ("viwiki")
    // - Search with InLanguage filter ("vi")
    // - Search limit works correctly
}
```

**Test Cases:**
| Test | Query | Expected |
|------|-------|----------|
| Basic search | "Vietnam" | >0 results with viwiki/vi |
| Vietnamese chars | "Việt Nam" | >0 results |
| Empty query | "" | 0 results |
| WikiName filter | q="vietnam", wiki="viwiki" | All results have WikiName="viwiki" |
| InLanguage filter | q="vietnam", lang="vi" | All results have InLanguage="vi" |
| Limit | q="a", limit=5 | <=5 results |

#### 2. TestStore_GetByID_E2E
Test the GetByID function.

```go
func TestStore_GetByID_E2E(t *testing.T) {
    // 1. Search for a known title to get an ID
    // 2. Call GetByID with that ID
    // 3. Verify all fields are populated:
    //    - ID, WikiName, PageID, Title, URL
    //    - InLanguage, Text (non-empty)
    //    - DateModified, WikidataID (may be empty)
    // 4. Test with non-existent ID returns error
}
```

**Assertions:**
- `page.ID` matches requested ID
- `page.Title` is non-empty
- `page.Text` is non-empty (article body)
- `page.InLanguage` == "vi"
- `page.WikiName` == "viwiki"
- Non-existent ID returns "page not found" error

#### 3. TestStore_GetByTitle_E2E
Test the GetByTitle function.

```go
func TestStore_GetByTitle_E2E(t *testing.T) {
    // 1. Search for a known title
    // 2. Call GetByTitle(wikiname, title)
    // 3. Verify the page matches
    // 4. Test with non-existent title returns error
}
```

#### 4. TestStore_Stats_E2E
Test the Stats function.

```go
func TestStore_Stats_E2E(t *testing.T) {
    // Verify stats contains:
    // - "titles": int64 > 0
    // - "wikis": map with "viwiki" key
    // - "seeded_at": non-empty timestamp string
}
```

#### 5. TestStore_Ensure_E2E
Test the Ensure function with different options.

```go
func TestStore_Ensure_E2E(t *testing.T) {
    // Test scenarios:
    // - SeedIfEmpty=true seeds titles table
    // - BuildIndex=true creates indexes
    // - Calling Ensure twice is idempotent
}
```

### Implementation Template

```go
//go:build e2e

package duckdb_test

import (
    "context"
    "database/sql"
    "testing"

    "github.com/go-mizu/blueprints/finewiki/cli"
    "github.com/go-mizu/blueprints/finewiki/feature/search"
    "github.com/go-mizu/blueprints/finewiki/store/duckdb"

    _ "github.com/duckdb/duckdb-go/v2"
)

func setupStore(t *testing.T) *duckdb.Store {
    t.Helper()

    dataDir := cli.DefaultDataDir()
    lang := "vi"

    if !parquetExists(dataDir, lang) {
        t.Skipf("Parquet not found; run 'finewiki import vi' first")
    }

    db, err := sql.Open("duckdb", "")
    if err != nil {
        t.Fatalf("open duckdb: %v", err)
    }
    t.Cleanup(func() { db.Close() })

    store, err := duckdb.New(db)
    if err != nil {
        t.Fatalf("new store: %v", err)
    }

    err = store.Ensure(context.Background(), duckdb.Config{
        ParquetGlob: cli.ParquetGlob(dataDir, lang),
        EnableFTS:   false,
    }, duckdb.EnsureOptions{
        SeedIfEmpty: true,
        BuildIndex:  true,
    })
    if err != nil {
        t.Fatalf("ensure: %v", err)
    }

    return store
}

func parquetExists(dataDir, lang string) bool {
    // Check for data.parquet or data-*.parquet
}
```

---

## Part 2: Move /search to / Root

### Current Routes
```
GET /         -> home (static landing page)
GET /search   -> searchPage (search results)
GET /page     -> page (view article)
```

### New Routes
```
GET /         -> searchPage (combined home + search)
GET /page     -> page (view article)
```

### Changes Required

#### 1. handlers.go

Modify `searchPage` to handle both cases:
- Empty query (`?q=`) → Show home UI with hero, search box, features
- Query present (`?q=vietnam`) → Show search results

```go
func (s *Server) searchPage(c *mizu.Ctx) error {
    ctx := c.Request().Context()
    text := strings.TrimSpace(c.Query("q"))

    // If no query, render home view
    if text == "" {
        s.render(c, "page/home.html", map[string]any{
            "Query": "",
            "Theme": "",
        })
        return nil
    }

    // Otherwise, perform search
    wikiname := strings.TrimSpace(c.Query("wiki"))
    lang := strings.TrimSpace(c.Query("lang"))

    results, err := s.search.Search(ctx, search.Query{
        Text:       text,
        WikiName:   wikiname,
        InLanguage: lang,
        Limit:      20,
    })
    if err != nil {
        return c.Text(500, err.Error())
    }

    s.render(c, "page/search.html", map[string]any{
        "Query":      text,
        "WikiName":   wikiname,
        "InLanguage": lang,
        "Results":    results,
        "Theme":      "",
    })
    return nil
}
```

#### 2. server.go

Update routes:
```go
func (s *Server) routes() {
    r := s.app
    r.Use(logging())

    // Combined home + search at root
    r.Get("/", s.searchPage)
    r.Get("/page", s.page)

    // Keep /search as alias for backwards compatibility (optional)
    // r.Get("/search", s.searchPage)

    // ... rest unchanged
}
```

Remove `home` handler (no longer needed).

#### 3. home.html Template Update

Update form action from `/search` to `/`:
```html
<form class="home-search" action="/" method="get">
```

Update hint links:
```html
<a href="/?q=Alan+Turing">Alan Turing</a>
```

---

## Part 3: app/web/server_e2e_test.go

### File Location
`blueprints/finewiki/app/web/server_e2e_test.go`

### Build Tag
```go
//go:build e2e
```

### Test Functions

#### 1. TestHTMLRoutes_E2E
Test that HTML pages render correctly.

```go
func TestHTMLRoutes_E2E(t *testing.T) {
    // Setup server with real store

    t.Run("Home page", func(t *testing.T) {
        resp := GET(ts.URL + "/")
        // Status: 200
        // Content-Type: text/html; charset=utf-8
        // Body contains: "FineWiki", "<form", "search"
    })

    t.Run("Search page with query", func(t *testing.T) {
        resp := GET(ts.URL + "/?q=vietnam")
        // Status: 200
        // Content-Type: text/html; charset=utf-8
        // Body contains: "Search results", "vietnam"
        // Body contains: <li class="search-result">
        // Body contains: href="/page?id=
    })

    t.Run("Search page empty query", func(t *testing.T) {
        resp := GET(ts.URL + "/?q=")
        // Status: 200
        // Content-Type: text/html
        // Body contains: "FineWiki" (home page)
    })

    t.Run("Page by ID", func(t *testing.T) {
        // First get a valid ID from search
        // Then request /page?id=...
        // Status: 200
        // Content-Type: text/html
        // Body contains: article content, title
    })

    t.Run("Page by wiki and title", func(t *testing.T) {
        // First search to get a title
        // Then request /page?wiki=viwiki&title=...
        // Status: 200
    })

    t.Run("Page not found", func(t *testing.T) {
        resp := GET(ts.URL + "/page?id=nonexistent/999999")
        // Status: 404
    })

    t.Run("Page missing params", func(t *testing.T) {
        resp := GET(ts.URL + "/page")
        // Status: 400
        // Body contains: "missing id or (wiki,title)"
    })
}
```

### Test Assertions

| Route | Status | Content-Type | Body Contains |
|-------|--------|--------------|---------------|
| `GET /` | 200 | text/html | "FineWiki", `<form`, `action="/"` |
| `GET /?q=vietnam` | 200 | text/html | "Search results", `<li class="search-result">` |
| `GET /?q=` | 200 | text/html | "FineWiki" (home) |
| `GET /page?id={valid}` | 200 | text/html | Page title, article text |
| `GET /page?wiki=viwiki&title={valid}` | 200 | text/html | Same as above |
| `GET /page?id=bad` | 404 | text/plain | "page not found" |
| `GET /page` | 400 | text/plain | "missing id or (wiki,title)" |

### Implementation Template

```go
//go:build e2e

package web_test

import (
    "context"
    "database/sql"
    "io"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "testing"

    "github.com/go-mizu/blueprints/finewiki/app/web"
    "github.com/go-mizu/blueprints/finewiki/cli"
    "github.com/go-mizu/blueprints/finewiki/feature/search"
    "github.com/go-mizu/blueprints/finewiki/feature/view"
    "github.com/go-mizu/blueprints/finewiki/store/duckdb"

    _ "github.com/duckdb/duckdb-go/v2"
)

func TestHTMLRoutes_E2E(t *testing.T) {
    if os.Getenv("E2E_TEST") != "1" {
        t.Skip("set E2E_TEST=1 to run")
    }

    ts := setupServer(t)
    defer ts.Close()

    t.Run("Home page", func(t *testing.T) {
        resp, body := get(t, ts.URL+"/")

        assertStatus(t, resp, 200)
        assertContentType(t, resp, "text/html")
        assertContains(t, body, "FineWiki")
        assertContains(t, body, "<form")
        assertContains(t, body, `action="/"`)
    })

    t.Run("Search with results", func(t *testing.T) {
        resp, body := get(t, ts.URL+"/?q=vietnam")

        assertStatus(t, resp, 200)
        assertContentType(t, resp, "text/html")
        assertContains(t, body, "Search results")
        assertContains(t, body, `class="search-result"`)
        assertContains(t, body, `href="/page?id=`)
    })

    t.Run("Page by ID", func(t *testing.T) {
        // Extract an ID from search results
        _, searchBody := get(t, ts.URL+"/?q=vietnam")
        id := extractPageID(t, searchBody)

        resp, body := get(t, ts.URL+"/page?id="+id)

        assertStatus(t, resp, 200)
        assertContentType(t, resp, "text/html")
        // Page should have content
        if len(body) < 1000 {
            t.Error("page body too short")
        }
    })
}

func setupServer(t *testing.T) *httptest.Server {
    // ... similar to api_e2e_test.go setup
}

func get(t *testing.T, url string) (*http.Response, string) {
    t.Helper()
    resp, err := http.Get(url)
    if err != nil {
        t.Fatalf("GET %s: %v", url, err)
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    return resp, string(body)
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
    t.Helper()
    if resp.StatusCode != want {
        t.Errorf("status: got %d, want %d", resp.StatusCode, want)
    }
}

func assertContentType(t *testing.T, resp *http.Response, want string) {
    t.Helper()
    ct := resp.Header.Get("Content-Type")
    if !strings.HasPrefix(ct, want) {
        t.Errorf("content-type: got %s, want prefix %s", ct, want)
    }
}

func assertContains(t *testing.T, body, substr string) {
    t.Helper()
    if !strings.Contains(body, substr) {
        t.Errorf("body missing %q", substr)
    }
}

func extractPageID(t *testing.T, body string) string {
    t.Helper()
    re := regexp.MustCompile(`href="/page\?id=([^"]+)"`)
    m := re.FindStringSubmatch(body)
    if len(m) < 2 {
        t.Fatal("no page ID found in search results")
    }
    return m[1]
}
```

---

## Implementation Order

1. **Write store/duckdb/store_e2e_test.go**
   - Implement all store-level tests
   - Verify with: `E2E_TEST=1 go test -tags=e2e ./store/duckdb -v`

2. **Modify routes: move /search to /**
   - Update `handlers.go`: combine home + search logic
   - Update `server.go`: change routes
   - Update `home.html`: fix form action and links
   - Verify: `go build ./...`

3. **Write app/web/server_e2e_test.go**
   - Implement HTML route tests
   - Verify with: `E2E_TEST=1 go test -tags=e2e ./app/web -run HTML -v`

4. **Run full test suite**
   ```bash
   E2E_TEST=1 go test -tags=e2e ./... -v
   ```

---

## Files Changed

| File | Change |
|------|--------|
| `store/duckdb/store_e2e_test.go` | New file |
| `app/web/server_e2e_test.go` | New file |
| `app/web/handlers.go` | Modify `searchPage` to handle empty query |
| `app/web/server.go` | Remove `/search`, update `/` to `searchPage` |
| `cli/views/page/home.html` | Update form action to `/` |

---

## Run Commands

```bash
# Run all E2E tests
E2E_TEST=1 go test -tags=e2e ./... -v

# Run only store tests
E2E_TEST=1 go test -tags=e2e ./store/duckdb -v

# Run only web tests
E2E_TEST=1 go test -tags=e2e ./app/web -v

# Run specific test
E2E_TEST=1 go test -tags=e2e ./store/duckdb -run TestStore_Search_E2E -v
```
