package goodread

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Client handles HTTP requests to goodreads.com with rate limiting,
// User-Agent rotation, and automatic retry.
type Client struct {
	http       *http.Client
	userAgents []string
	delay      time.Duration
	lastReq    time.Time
}

// NewClient creates a new Goodreads HTTP client.
func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     cfg.Workers + 2,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}

	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		userAgents: userAgents,
		delay:      cfg.Delay,
	}
}

// Fetch fetches the raw bytes of a URL. Returns (body, statusCode, error).
// Retries up to 3 times on transient errors. Backs off on 429.
func (c *Client) Fetch(ctx context.Context, url string) ([]byte, int, error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c.rateLimit()

		body, code, err := c.doGet(ctx, url)
		if err != nil {
			if attempt == maxAttempts {
				return nil, 0, err
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if code == 429 {
			backoff := time.Duration(attempt*attempt) * 10 * time.Second
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("rate limited (HTTP 429) after %d attempts", maxAttempts)
			}
			time.Sleep(backoff)
			continue
		}

		if code == 404 {
			return nil, code, nil // permanent failure, caller decides
		}

		if code >= 500 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("server error HTTP %d", code)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		return body, code, nil
	}

	return nil, 0, fmt.Errorf("all %d attempts failed", maxAttempts)
}

// FetchHTML fetches a URL and returns a parsed goquery document.
func (c *Client) FetchHTML(ctx context.Context, url string) (*goquery.Document, int, error) {
	body, code, err := c.Fetch(ctx, url)
	if err != nil {
		return nil, code, err
	}
	if code == 404 {
		return nil, code, nil
	}
	if code != 200 {
		return nil, code, fmt.Errorf("unexpected HTTP %d for %s", code, url)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, code, fmt.Errorf("parse HTML: %w", err)
	}

	return doc, code, nil
}

func (c *Client) doGet(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}

	ua := c.userAgents[rand.Intn(len(c.userAgents))]
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Do NOT set Accept-Encoding — let Go's transport handle transparent gzip decompression.

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	// Follow Goodreads HTML soft-redirect: "You are being redirected"
	if resp.StatusCode == 200 && strings.Contains(string(body), "You are being") {
		if redirectURL := extractHTMLRedirect(string(body)); redirectURL != "" && redirectURL != url {
			time.Sleep(500 * time.Millisecond)
			return c.doGet(ctx, redirectURL)
		}
	}

	return body, resp.StatusCode, nil
}

// extractHTMLRedirect extracts the redirect URL from a Goodreads HTML redirect page.
func extractHTMLRedirect(body string) string {
	// Pattern: <a href="https://www.goodreads.com/...">redirected</a>
	start := strings.Index(body, `href="`)
	if start < 0 {
		return ""
	}
	start += len(`href="`)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}

// rateLimit enforces the minimum delay between requests.
func (c *Client) rateLimit() {
	if c.delay <= 0 {
		return
	}
	since := time.Since(c.lastReq)
	if since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}
