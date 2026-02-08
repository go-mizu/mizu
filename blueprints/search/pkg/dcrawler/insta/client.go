package insta

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// queryType classifies requests for per-type rate limiting.
type queryType int

const (
	queryTypeOther   queryType = iota // non-GraphQL web requests
	queryTypeGraphQL                  // GraphQL query_hash or doc_id requests
	queryTypeIPhone                   // i.instagram.com API requests
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

	// iPhone API state: response headers mirrored back on next request
	iphoneMu      sync.Mutex
	iphoneHeaders map[string]string

	// Adaptive page length: halved on 400 errors (from 50 -> 25 -> 12)
	graphqlPageLength int
}

// rateController tracks per-query-type timestamps for fine-grained rate limiting.
// Mirrors instaloader's RateController with separate windows for:
// - Per query_hash GraphQL (200 req / 11 min)
// - Non-GraphQL "other" requests (75 req / 11 min)
// - Accumulated all-GraphQL (275 req / 10 min)
// - iPhone API (199 req / 30 min)
type rateController struct {
	mu sync.Mutex

	// Per-type timestamps keyed by query identifier
	// For GraphQL: key is the query_hash or doc_id
	// For other: key is "other"
	// For iPhone: key is "iphone"
	perType map[string][]time.Time

	// All GraphQL timestamps combined (for accumulated limit)
	gqlAccum []time.Time

	// iPhone timestamps
	iphone []time.Time

	// Earliest next request time (set by 429 handler)
	earliestNext time.Time
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
			// Don't follow redirects automatically — we detect login redirects manually
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cfg:               cfg,
		iphoneHeaders:     make(map[string]string),
		graphqlPageLength: PostsPerPage,
	}

	c.rate.perType = make(map[string][]time.Time)

	return c, nil
}

// Init loads the Instagram homepage to acquire cookies and CSRF token.
func (c *Client) Init(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.instagram.com/", nil)
	if err != nil {
		return err
	}
	c.setWebHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	c.extractCSRF()
	return nil
}

// setWebHeaders sets standard Instagram web headers on a request.
func (c *Client) setWebHeaders(req *http.Request) {
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

// setIPhoneHeaders sets iPhone/iPad app headers for i.instagram.com requests.
// Mirrors instaloader's default_iphone_headers().
func (c *Client) setIPhoneHeaders(req *http.Request) {
	req.Header.Set("User-Agent", IPhoneUserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("x-ads-opt-out", "1")
	req.Header.Set("x-bloks-is-panorama-enabled", "true")
	req.Header.Set("x-bloks-version-id", "16b7bd25c6c06886d57c4d455265669345a2d96625385b8ee30026ac2dc5ed97")
	req.Header.Set("x-fb-client-ip", "True")
	req.Header.Set("x-fb-connection-type", "wifi")
	req.Header.Set("x-fb-http-engine", "Liger")
	req.Header.Set("x-fb-server-cluster", "True")
	req.Header.Set("x-fb", "1")
	req.Header.Set("x-ig-abr-connection-speed-kbps", "2")
	req.Header.Set("x-ig-app-id", IPhoneAppID)
	req.Header.Set("x-ig-app-locale", "en-US")
	req.Header.Set("x-ig-app-startup-country", "US")
	req.Header.Set("x-ig-bandwidth-speed-kbps", "0.000")
	req.Header.Set("x-ig-capabilities", "36r/F/8=")
	req.Header.Set("x-ig-connection-speed", fmt.Sprintf("%dkbps", 1000+rand.IntN(19000)))
	req.Header.Set("x-ig-connection-type", "WiFi")
	req.Header.Set("x-ig-device-locale", "en-US")
	req.Header.Set("x-ig-mapped-locale", "en-US")
	req.Header.Set("x-ig-www-claim", "0")
	req.Header.Set("x-tigon-is-retry", "False")
	req.Header.Set("x-whatsapp", "0")

	// Set user-specific headers
	if c.userID != "" {
		req.Header.Set("ig-intended-user-id", c.userID)
	}
	req.Header.Set("x-pigeon-rawclienttime", fmt.Sprintf("%d", time.Now().Unix()))

	// Mirror stored iPhone response headers
	c.iphoneMu.Lock()
	for k, v := range c.iphoneHeaders {
		req.Header.Set(k, v)
	}
	c.iphoneMu.Unlock()

	// Map cookies to headers (like instaloader does)
	u, _ := url.Parse("https://www.instagram.com/")
	for _, cookie := range c.http.Jar.Cookies(u) {
		switch cookie.Name {
		case "mid":
			req.Header.Set("x-mid", cookie.Value)
		case "ds_user_id":
			req.Header.Set("ig-u-ds-user-id", cookie.Value)
		case "ig_did":
			req.Header.Set("x-ig-device-id", cookie.Value)
			req.Header.Set("x-ig-family-device-id", cookie.Value)
		}
	}

	if c.csrfToken != "" {
		req.Header.Set("X-CSRFToken", c.csrfToken)
	}
}

// storeIPhoneResponseHeaders extracts ig-set-* headers from iPhone API responses
// and stores them for subsequent requests.
func (c *Client) storeIPhoneResponseHeaders(resp *http.Response) {
	c.iphoneMu.Lock()
	defer c.iphoneMu.Unlock()

	for key, values := range resp.Header {
		lk := strings.ToLower(key)
		if strings.HasPrefix(lk, "ig-set-") || strings.HasPrefix(lk, "x-ig-set-") {
			// Convert ig-set-X to X for next request
			headerName := strings.TrimPrefix(lk, "ig-set-")
			headerName = strings.TrimPrefix(headerName, "x-ig-set-")
			if len(values) > 0 {
				c.iphoneHeaders[headerName] = values[0]
			}
		}
	}
}

// doSleep performs an exponential-distribution sleep before requests (like instaloader).
// Mean sleep = 1/lambda = ~1.67s, capped at 15s.
func (c *Client) doSleep(ctx context.Context) error {
	if c.cfg.Delay <= 0 {
		return nil
	}

	// Exponential distribution: lambda=0.6, mean=1.67s, capped at 15s
	// This provides natural-looking random delays
	lambda := 0.6
	sleepSec := math.Min(-math.Log(1-rand.Float64())/lambda, 15.0)

	// Scale by configured delay ratio (3s default → multiplier ~1.8)
	multiplier := float64(c.cfg.Delay) / float64(3*time.Second)
	if multiplier < 0.1 {
		multiplier = 0.1
	}
	d := time.Duration(sleepSec * multiplier * float64(time.Second))

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// delay is an alias for doSleep, kept for backward compatibility.
func (c *Client) delay(ctx context.Context) error {
	return c.doSleep(ctx)
}

// handleRedirects follows same-host redirects and detects login page redirects.
// Returns the final response body, or an error if redirected to login.
func (c *Client) handleRedirects(ctx context.Context, resp *http.Response, host string) (*http.Response, error) {
	for resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location == "" {
			break
		}

		// Detect login page redirects
		if strings.Contains(location, "/accounts/login") {
			resp.Body.Close()
			if !c.loggedIn {
				return nil, fmt.Errorf("login required: redirected to login page")
			}
			return nil, fmt.Errorf("session expired: redirected to login page (re-login required)")
		}

		// Only follow same-host redirects
		if !strings.HasPrefix(location, "https://"+host+"/") {
			break
		}

		resp.Body.Close()
		req, err := http.NewRequestWithContext(ctx, "GET", location, nil)
		if err != nil {
			return nil, err
		}
		c.setWebHeaders(req)

		resp, err = c.http.Do(req)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// doGet performs a GET request with standard web headers, rate limiting, and retry logic.
func (c *Client) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	return c.doGetTyped(ctx, rawURL, queryTypeOther, "other")
}

// doGetTyped performs a GET request with query-type-aware rate limiting.
func (c *Client) doGetTyped(ctx context.Context, rawURL string, qt queryType, queryID string) ([]byte, error) {
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

		if err := c.waitRate(ctx, qt, queryID); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		c.setWebHeaders(req)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Handle redirects (detect login page)
		resp, err = c.handleRedirects(ctx, resp, "www.instagram.com")
		if err != nil {
			resp = nil
			// Login redirects are not retriable
			return nil, err
		}

		body, err := readBody(resp)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == 429 {
			c.handle429(qt, queryID)
			lastErr = fmt.Errorf("rate limited (HTTP 429)")
			continue
		}

		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("not found (HTTP 404)")
		}

		if resp.StatusCode == 400 {
			msg := truncate(string(body), 200)
			// Check for fatal messages that should not be retried
			if containsAny(string(body), "feedback_required", "checkpoint_required", "challenge_required") {
				return nil, fmt.Errorf("blocked: %s (HTTP 400)", msg)
			}
			lastErr = fmt.Errorf("bad request (HTTP 400): %s", msg)
			continue
		}

		if resp.StatusCode != 200 {
			msg := truncate(string(body), 200)
			if strings.Contains(string(body), "checkpoint_required") {
				return nil, fmt.Errorf("checkpoint required (HTTP %d)", resp.StatusCode)
			}
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
			continue
		}

		return body, nil
	}
	return nil, fmt.Errorf("after %d retries: %w", c.cfg.MaxRetry, lastErr)
}

// doGetIPhone performs a GET request against the iPhone API (i.instagram.com).
func (c *Client) doGetIPhone(ctx context.Context, path string, params url.Values) ([]byte, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("iPhone API requires authentication")
	}

	rawURL := IPhoneAPIBase + path
	if len(params) > 0 {
		rawURL += "?" + params.Encode()
	}

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

		if err := c.waitRate(ctx, queryTypeIPhone, "iphone"); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		c.setIPhoneHeaders(req)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Store response headers for future requests
		c.storeIPhoneResponseHeaders(resp)

		// Handle redirects
		resp, err = c.handleRedirects(ctx, resp, "i.instagram.com")
		if err != nil {
			return nil, err
		}

		body, err := readBody(resp)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == 429 {
			c.handle429(queryTypeIPhone, "iphone")
			lastErr = fmt.Errorf("rate limited (HTTP 429)")
			continue
		}

		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("not found (HTTP 404)")
		}

		if resp.StatusCode != 200 {
			msg := truncate(string(body), 200)
			if containsAny(string(body), "checkpoint_required", "challenge_required", "feedback_required") {
				return nil, fmt.Errorf("blocked: %s (HTTP %d)", msg, resp.StatusCode)
			}
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
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
	return c.doGetTyped(ctx, fullURL, queryTypeGraphQL, queryHash)
}

// graphQLWithAutoReduce performs a GraphQL query and automatically reduces page length on 400 errors.
func (c *Client) graphQLWithAutoReduce(ctx context.Context, queryHash string, variables map[string]any) ([]byte, error) {
	data, err := c.graphQL(ctx, queryHash, variables)
	if err != nil && strings.Contains(err.Error(), "bad request (HTTP 400)") {
		// Try with reduced page length
		newLen := c.graphqlPageLength / 2
		if newLen >= 6 {
			c.graphqlPageLength = newLen
			// Update the "first" variable if present
			if _, ok := variables["first"]; ok {
				variables["first"] = c.graphqlPageLength
			}
			return c.graphQL(ctx, queryHash, variables)
		}
	}
	return data, err
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

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ── Rate Controller ─────────────────────────────────────────

// waitRate blocks until the rate limiter allows a new request, respecting
// per-query-type, accumulated, and iPhone windows.
func (c *Client) waitRate(ctx context.Context, qt queryType, queryID string) error {
	for {
		wait := c.queryWaitTime(qt, queryID)
		if wait <= 0 {
			// Record this request
			c.recordTimestamp(qt, queryID)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
			// Re-check after waiting
		}
	}
}

// queryWaitTime calculates how long to wait before the next request.
// Returns 0 if the request can proceed immediately.
func (c *Client) queryWaitTime(qt queryType, queryID string) time.Duration {
	c.rate.mu.Lock()
	defer c.rate.mu.Unlock()

	now := time.Now()
	var maxWait time.Duration

	// Check 429-imposed earliest next request
	if now.Before(c.rate.earliestNext) {
		wait := c.rate.earliestNext.Sub(now)
		if wait > maxWait {
			maxWait = wait
		}
	}

	switch qt {
	case queryTypeGraphQL:
		// Per query type limit: 200 requests per 11 minutes
		w := perTypeWait(c.rate.perType[queryID], now, RateWindowGQL, RateLimitGQL, RateBufferRegular)
		if w > maxWait {
			maxWait = w
		}
		// Accumulated GQL limit: 275 requests per 10 minutes
		w = perTypeWait(c.rate.gqlAccum, now, RateWindowGQLAccum, RateLimitGQLAccum, RateBufferRegular)
		if w > maxWait {
			maxWait = w
		}

	case queryTypeIPhone:
		// iPhone limit: 199 requests per 30 minutes
		w := perTypeWait(c.rate.iphone, now, RateWindowIPhone, RateLimitIPhone, RateBufferIPhone)
		if w > maxWait {
			maxWait = w
		}

	case queryTypeOther:
		// Other limit: 75 requests per 11 minutes
		w := perTypeWait(c.rate.perType["other"], now, RateWindowOther, RateLimitOther, RateBufferRegular)
		if w > maxWait {
			maxWait = w
		}
	}

	return maxWait
}

// perTypeWait calculates wait time for a single sliding window.
func perTypeWait(timestamps []time.Time, now time.Time, windowSec, limit, bufferSec int) time.Duration {
	window := time.Duration(windowSec) * time.Second
	buffer := time.Duration(bufferSec) * time.Second
	cutoff := now.Add(-window)

	count := 0
	var oldest time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			count++
			if oldest.IsZero() || ts.Before(oldest) {
				oldest = ts
			}
		}
	}

	if count < limit {
		return 0
	}

	// Wait until oldest timestamp exits the window + buffer
	waitUntil := oldest.Add(window + buffer)
	if waitUntil.After(now) {
		return waitUntil.Sub(now)
	}
	return 0
}

// recordTimestamp records a request timestamp for rate limiting.
func (c *Client) recordTimestamp(qt queryType, queryID string) {
	c.rate.mu.Lock()
	defer c.rate.mu.Unlock()

	now := time.Now()
	retention := time.Hour // keep timestamps for 1 hour

	switch qt {
	case queryTypeGraphQL:
		c.rate.perType[queryID] = pruneAndAppend(c.rate.perType[queryID], now, retention)
		c.rate.gqlAccum = pruneAndAppend(c.rate.gqlAccum, now, retention)

	case queryTypeIPhone:
		c.rate.iphone = pruneAndAppend(c.rate.iphone, now, retention)

	case queryTypeOther:
		c.rate.perType["other"] = pruneAndAppend(c.rate.perType["other"], now, retention)
	}
}

// pruneAndAppend removes timestamps older than retention and appends now.
func pruneAndAppend(timestamps []time.Time, now time.Time, retention time.Duration) []time.Time {
	cutoff := now.Add(-retention)
	valid := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	return append(valid, now)
}

// handle429 is called when a 429 response is received.
// Sets an aggressive earliest-next-request time based on the current window state.
func (c *Client) handle429(qt queryType, queryID string) {
	c.rate.mu.Lock()
	defer c.rate.mu.Unlock()

	now := time.Now()

	// Set earliest next request based on query type
	var waitDuration time.Duration
	switch qt {
	case queryTypeGraphQL:
		// Wait the full per-type window + buffer
		waitDuration = time.Duration(RateWindowGQL+RateBufferRegular) * time.Second
	case queryTypeIPhone:
		waitDuration = time.Duration(RateWindowIPhone+RateBufferIPhone) * time.Second
	default:
		waitDuration = time.Duration(RateWindowOther+RateBufferRegular) * time.Second
	}

	earliest := now.Add(waitDuration / 4) // Wait 1/4 of the full window on 429
	if earliest.After(c.rate.earliestNext) {
		c.rate.earliestNext = earliest
	}
}

// GraphQLPageLength returns the current adaptive page length.
func (c *Client) GraphQLPageLength() int {
	return c.graphqlPageLength
}
