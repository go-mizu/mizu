package insta

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is an HTTP client for Instagram's public web API.
type Client struct {
	http      *http.Client
	cfg       Config
	csrfToken string
	username  string
	userID    string
	loggedIn  bool
	rate      rateController
}

// rateController tracks request timestamps for rate limiting.
type rateController struct {
	mu         sync.Mutex
	timestamps []time.Time // sliding window of request times
}

const (
	rateWindow = 11 * time.Minute // 660 seconds
	rateLimit  = 200              // max requests per window
)

// NewClient creates a new Instagram client.
func NewClient(cfg Config) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	c := &Client{
		http: &http.Client{
			Timeout: cfg.Timeout,
			Jar:     jar,
		},
		cfg: cfg,
	}

	return c, nil
}

// Init loads the Instagram homepage to acquire cookies and CSRF token.
func (c *Client) Init(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.instagram.com/", nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	c.extractCSRF()
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.instagram.com/")
	req.Header.Set("Origin", "https://www.instagram.com")
	req.Header.Set("X-IG-App-ID", WebAppID)
	req.Header.Set("X-Instagram-AJAX", "1")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if c.csrfToken != "" {
		req.Header.Set("X-CSRFToken", c.csrfToken)
	}
}

// doGet performs a GET request with standard headers, rate limiting, and retry logic.
func (c *Client) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := range c.cfg.MaxRetry {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		if err := c.waitRate(ctx); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		c.setHeaders(req)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := readBody(resp)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == 429 {
			lastErr = fmt.Errorf("rate limited (HTTP 429)")
			continue
		}

		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("not found (HTTP 404)")
		}

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
			continue
		}

		return body, nil
	}
	return nil, fmt.Errorf("after %d retries: %w", c.cfg.MaxRetry, lastErr)
}

// graphQL performs a GraphQL query with the given query_hash and variables.
func (c *Client) graphQL(ctx context.Context, queryHash string, variables map[string]any) ([]byte, error) {
	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("query_hash", queryHash)
	params.Set("variables", string(varsJSON))

	fullURL := GraphQLURL + "?" + params.Encode()
	return c.doGet(ctx, fullURL)
}

// delay waits the configured delay with jitter between requests.
func (c *Client) delay(ctx context.Context) error {
	if c.cfg.Delay <= 0 {
		return nil
	}
	// Add 0-50% jitter
	jitter := time.Duration(rand.Int64N(int64(c.cfg.Delay / 2)))
	d := c.cfg.Delay + jitter

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func readBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	}
	return io.ReadAll(reader)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// waitRate blocks until the rate limiter allows a new request.
func (c *Client) waitRate(ctx context.Context) error {
	c.rate.mu.Lock()
	now := time.Now()

	// Remove timestamps outside the window
	cutoff := now.Add(-rateWindow)
	valid := c.rate.timestamps[:0]
	for _, ts := range c.rate.timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	c.rate.timestamps = valid

	if len(c.rate.timestamps) < rateLimit {
		c.rate.timestamps = append(c.rate.timestamps, now)
		c.rate.mu.Unlock()
		return nil
	}

	// Calculate wait time: oldest timestamp + window duration
	waitUntil := c.rate.timestamps[0].Add(rateWindow).Add(time.Second)
	c.rate.mu.Unlock()

	wait := time.Until(waitUntil)
	if wait <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
		return c.waitRate(ctx) // re-check after waiting
	}
}

// DoGetRaw exposes doGet for debugging/testing.
func (c *Client) DoGetRaw(ctx context.Context, rawURL string) ([]byte, error) {
	return c.doGet(ctx, rawURL)
}

// DocIDQueryRaw exposes docIDQuery for debugging/testing.
func (c *Client) DocIDQueryRaw(ctx context.Context, docID string, variables map[string]any) ([]byte, error) {
	return c.docIDQuery(ctx, docID, variables)
}

// recordRequest adds a timestamp to the rate limiter.
func (c *Client) recordRequest() {
	c.rate.mu.Lock()
	c.rate.timestamps = append(c.rate.timestamps, time.Now())
	c.rate.mu.Unlock()
}
