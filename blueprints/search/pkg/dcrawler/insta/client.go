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
	"time"
)

// Client is an HTTP client for Instagram's public web API.
type Client struct {
	http      *http.Client
	cfg       Config
	csrfToken string
}

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

	// Extract csrftoken from cookies
	u, _ := url.Parse("https://www.instagram.com/")
	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "csrftoken" {
			c.csrfToken = cookie.Value
			break
		}
	}

	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Referer", "https://www.instagram.com/")
	req.Header.Set("X-IG-App-ID", WebAppID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if c.csrfToken != "" {
		req.Header.Set("X-CSRFToken", c.csrfToken)
	}
}

// doGet performs a GET request with standard headers and retry logic.
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
