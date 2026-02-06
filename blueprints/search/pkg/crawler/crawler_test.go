package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestCrawlerBasic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><head><title>Home</title></head><body>
				<a href="/about">About</a>
				<a href="/contact">Contact</a>
			</body></html>`)
		case "/about":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><head><title>About</title></head><body>
				<p>About page content</p>
			</body></html>`)
		case "/contact":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><head><title>Contact</title></head><body>
				<p>Contact page content</p>
			</body></html>`)
		case "/robots.txt":
			fmt.Fprintf(w, "User-agent: *\nAllow: /\n")
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Workers = 2
	cfg.MaxDepth = 1
	cfg.MaxPages = 10
	cfg.Delay = 0
	cfg.RespectRobots = false

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var mu sync.Mutex
	var results []CrawlResult
	c.OnResult(func(r CrawlResult) {
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.Crawl(ctx, srv.URL+"/")
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}

	if stats.PagesSuccess < 1 {
		t.Errorf("PagesSuccess = %d, want >= 1", stats.PagesSuccess)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(results) < 1 {
		t.Fatalf("got %d results, want >= 1", len(results))
	}

	// Check that we got the home page
	found := false
	for _, r := range results {
		if r.Title == "Home" {
			found = true
			break
		}
	}
	if !found {
		t.Error("should have crawled the home page")
	}
}

func TestCrawlerSitemap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>%s/page1</loc></url>
  <url><loc>%s/page2</loc></url>
</urlset>`, "http://"+r.Host, "http://"+r.Host)
		case "/page1":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><head><title>Page 1</title></head><body>Content 1</body></html>`)
		case "/page2":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><head><title>Page 2</title></head><body>Content 2</body></html>`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Workers = 2
	cfg.MaxPages = 10
	cfg.Delay = 0
	cfg.RespectRobots = false

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var mu sync.Mutex
	var results []CrawlResult
	c.OnResult(func(r CrawlResult) {
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.CrawlSitemap(ctx, srv.URL+"/sitemap.xml")
	if err != nil {
		t.Fatalf("CrawlSitemap error: %v", err)
	}

	if stats.PagesSuccess != 2 {
		t.Errorf("PagesSuccess = %d, want 2", stats.PagesSuccess)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestCrawlerMaxPages(t *testing.T) {
	pageCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		pageCount++
		fmt.Fprintf(w, `<html><head><title>Page %d</title></head><body>
			<a href="/page%d">Next</a>
		</body></html>`, pageCount, pageCount+1)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Workers = 1
	cfg.MaxPages = 3
	cfg.MaxDepth = 10
	cfg.Delay = 0
	cfg.RespectRobots = false

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.Crawl(ctx, srv.URL+"/")
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}

	if stats.PagesSuccess > 3 {
		t.Errorf("PagesSuccess = %d, should be <= 3", stats.PagesSuccess)
	}
}

func TestCrawlerContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><head><title>Page</title></head><body>
			<a href="/page2">Next</a>
		</body></html>`)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Workers = 1
	cfg.MaxPages = 1000
	cfg.MaxDepth = 100
	cfg.Delay = 0
	cfg.RespectRobots = false

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = c.Crawl(ctx, srv.URL+"/")
	if err == nil {
		// It's OK if it finishes quickly without error
		return
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestCrawlerInvalidConfig(t *testing.T) {
	cfg := Config{Workers: -1}
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}
