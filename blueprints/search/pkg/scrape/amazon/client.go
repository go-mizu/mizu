package amazon

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Client handles HTTP requests to amazon.com with rate limiting,
// User-Agent rotation, CAPTCHA detection, and automatic retry.
type Client struct {
	http       *http.Client
	userAgents []string
	delay      time.Duration
	lastReq    time.Time
	mu         sync.Mutex
}

// NewClient creates a new Amazon HTTP client.
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

// Fetch fetches raw bytes of a URL. Returns (body, statusCode, error).
// Retries up to 3 times on transient errors. Backs off on 429 and 503.
// 404 returns (nil, 404, nil) — permanent failure, caller decides.
// CAPTCHA returns an error "captcha detected".
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

		if code == 404 {
			return nil, code, nil // permanent failure
		}

		if code == 429 {
			backoff := time.Duration(attempt*attempt) * 10 * time.Second
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("rate limited (HTTP 429) after %d attempts", maxAttempts)
			}
			time.Sleep(backoff)
			continue
		}

		if code == 503 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("service unavailable (HTTP 503) after %d attempts", maxAttempts)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		if code >= 500 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("server error HTTP %d", code)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		if code == 200 && isCAPTCHA(body) {
			return nil, 503, fmt.Errorf("captcha detected")
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
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.amazon.com/")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}

// isCAPTCHA returns true if the response body is an Amazon CAPTCHA page.
func isCAPTCHA(body []byte) bool {
	s := string(body)
	return strings.Contains(s, "/errors/validateCaptcha") ||
		strings.Contains(s, "Robot Check") ||
		strings.Contains(s, `action="/errors/validateCaptcha"`)
}

// rateLimit enforces the minimum delay between requests (thread-safe).
func (c *Client) rateLimit() {
	if c.delay <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	since := time.Since(c.lastReq)
	if since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}
