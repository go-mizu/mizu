package goodread

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-mizu/mizu/blueprints/search/pkg/cloudflare/browser"
	"golang.org/x/time/rate"
)

// WorkerClient fetches Goodreads pages through the browser.go-mizu.workers.dev
// proxy, which issues requests from Cloudflare's IP space rather than the
// server's IP — bypassing per-IP throttling by Goodreads.
type WorkerClient struct {
	cf      *browser.Client
	limiter *rate.Limiter
	timeout time.Duration
}

// NewWorkerClient creates a client that proxies through the CF browser worker.
// Token is read from BROWSER_API_TOKEN env var or cfg.WorkerToken.
func NewWorkerClient(cfg Config) (*WorkerClient, error) {
	token := cfg.WorkerToken
	if token == "" {
		token = os.Getenv("BROWSER_API_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("worker token required: set --worker-token or BROWSER_API_TOKEN env var")
	}

	workerURL := cfg.WorkerURL
	if workerURL == "" {
		workerURL = browser.DefaultProxyURL
	}

	proxy := browser.ProxyConfig{URL: workerURL, Token: token}
	// Empty credentials — we only use the proxy, not direct CF API.
	cf := browser.NewClientWithProxy(browser.Credentials{}, proxy)

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &WorkerClient{
		cf:      cf,
		limiter: newLimiter(cfg),
		timeout: timeout,
	}, nil
}

// FetchHTML fetches a URL via the CF worker and returns a parsed goquery document.
func (c *WorkerClient) FetchHTML(ctx context.Context, url string) (*goquery.Document, int, error) {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, 0, err
		}
	}

	html, err := c.cf.Content(browser.ContentRequest{
		CommonRequest: browser.CommonRequest{
			URL: url,
			// Reject heavy resources we don't need for HTML parsing.
			RejectResourceTypes: []string{"image", "media", "font", "stylesheet"},
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("worker fetch %s: %w", url, err)
	}

	// Detect login redirect.
	if strings.Contains(html, "/sign_in") && strings.Contains(html, "You must sign in") {
		return nil, 401, fmt.Errorf("login required")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, 0, fmt.Errorf("parse html: %w", err)
	}

	return doc, 200, nil
}

// FetchHTMLTimed is an alias for FetchHTML for use in benchmarks.
func (c *WorkerClient) FetchHTMLTimed(ctx context.Context, url string) (*goquery.Document, int, error) {
	return c.FetchHTML(ctx, url)
}
