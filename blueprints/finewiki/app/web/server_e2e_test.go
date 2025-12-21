//go:build e2e

package web_test

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/finewiki/app/web"
	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
	"github.com/go-mizu/blueprints/finewiki/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestHTMLRoutes_E2E tests the HTML rendering endpoints with real data.
// Run with: E2E_TEST=1 go test -tags=e2e ./app/web -run HTMLRoutes -v
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

	t.Run("Home page with empty query", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/?q=")

		assertStatus(t, resp, 200)
		assertContentType(t, resp, "text/html")
		assertContains(t, body, "FineWiki")
	})

	t.Run("Search with results", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/?q=vietnam")

		assertStatus(t, resp, 200)
		assertContentType(t, resp, "text/html")
		assertContains(t, body, "Search results")
		assertContains(t, body, `class="search-result"`)
		assertContains(t, body, `href="/page?id=`)
	})

	t.Run("Search with Vietnamese characters", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/?q=Việt")

		assertStatus(t, resp, 200)
		assertContentType(t, resp, "text/html")
		// Should have results or at least the search page
		assertContains(t, body, "Search results")
	})

	t.Run("Search no results", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/?q=xyznonexistent12345")

		assertStatus(t, resp, 200)
		assertContentType(t, resp, "text/html")
		assertContains(t, body, "No articles found")
	})

	t.Run("Page by ID", func(t *testing.T) {
		// First get a valid ID from search results
		_, searchBody := get(t, ts.URL+"/?q=vietnam")
		id := extractPageID(t, searchBody)
		if id == "" {
			t.Skip("no page ID found in search results")
		}

		resp, body := get(t, ts.URL+"/page?id="+id)

		assertStatus(t, resp, 200)
		assertContentType(t, resp, "text/html")
		// Page should have substantial content
		if len(body) < 500 {
			t.Errorf("page body too short: %d bytes", len(body))
		}
	})

	t.Run("Page by wiki and title", func(t *testing.T) {
		// First search to get a title
		_, searchBody := get(t, ts.URL+"/?q=vietnam")
		id := extractPageID(t, searchBody)
		if id == "" {
			t.Skip("no page ID found in search results")
		}

		// Extract wikiname and get page to find title
		parts := strings.SplitN(id, "/", 2)
		if len(parts) != 2 {
			t.Skipf("unexpected ID format: %s", id)
		}
		wikiname := parts[0]

		// Get page by ID first to get the title
		_, pageBody := get(t, ts.URL+"/page?id="+id)
		title := extractPageTitle(pageBody)
		if title == "" {
			t.Skip("could not extract page title")
		}

		// Now test getting by wiki and title
		resp, body := get(t, ts.URL+"/page?wiki="+wikiname+"&title="+title)

		assertStatus(t, resp, 200)
		assertContentType(t, resp, "text/html")
		if len(body) < 500 {
			t.Errorf("page body too short: %d bytes", len(body))
		}
	})

	t.Run("Page not found", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/page?id=nonexistent/99999999")

		assertStatus(t, resp, 404)
		assertContains(t, body, "not found")
	})

	t.Run("Page missing params", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/page")

		assertStatus(t, resp, 400)
		assertContains(t, body, "missing id or (wiki,title)")
	})

	t.Run("Page missing title", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/page?wiki=viwiki")

		assertStatus(t, resp, 400)
		assertContains(t, body, "missing id or (wiki,title)")
	})
}

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()

	ctx := context.Background()
	dataDir := cli.DefaultDataDir()
	lang := "vi"

	if !parquetExists(dataDir, lang) {
		t.Skipf("Parquet not found at %s/%s; run 'finewiki import vi' first", dataDir, lang)
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

	if err := store.Ensure(ctx, duckdb.Config{
		ParquetGlob: cli.ParquetGlob(dataDir, lang),
		EnableFTS:   false,
	}, duckdb.EnsureOptions{
		SeedIfEmpty: true,
		BuildIndex:  true,
	}); err != nil {
		t.Fatalf("ensure store: %v", err)
	}

	searchSvc := search.New(store)
	viewSvc := view.New(store)

	tmpl, err := web.NewTemplates()
	if err != nil {
		t.Fatalf("new templates: %v", err)
	}

	srv := web.New(viewSvc, searchSvc, tmpl)
	return httptest.NewServer(srv.Handler())
}

func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
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
		t.Errorf("content-type: got %q, want prefix %q", ct, want)
	}
}

func assertContains(t *testing.T, body, substr string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Errorf("body missing %q (len=%d)", substr, len(body))
	}
}

func extractPageID(t *testing.T, body string) string {
	t.Helper()
	re := regexp.MustCompile(`href="/page\?id=([^"]+)"`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	// URL-decode the ID
	id, err := url.QueryUnescape(m[1])
	if err != nil {
		return m[1]
	}
	return id
}

func extractPageTitle(body string) string {
	// Try to find title in common patterns
	// Look for <h1>Title</h1> or similar
	re := regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	m := re.FindStringSubmatch(body)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}
