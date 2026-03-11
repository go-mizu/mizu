package scrape

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DataDog/zstd"
	"github.com/cespare/xxhash/v2"
)

// CloudflareCredentials holds CF account credentials, read from
// $HOME/data/cloudflare/cloudflare.json.
type CloudflareCredentials struct {
	AccountID string `json:"account_id"`
	APIToken  string `json:"api_token"`
}

// LoadCloudflareCredentials reads credentials from $HOME/data/cloudflare/cloudflare.json.
func LoadCloudflareCredentials() (CloudflareCredentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return CloudflareCredentials{}, fmt.Errorf("get home dir: %w", err)
	}
	path := filepath.Join(home, "data", "cloudflare", "cloudflare.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return CloudflareCredentials{}, fmt.Errorf("read %s: %w\n\nCreate it with:\n  mkdir -p ~/data/cloudflare\n  echo '{\"account_id\":\"...\",\"api_token\":\"...\"}' > ~/data/cloudflare/cloudflare.json", path, err)
	}
	var creds CloudflareCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return CloudflareCredentials{}, fmt.Errorf("parse cloudflare.json: %w", err)
	}
	if creds.AccountID == "" {
		return CloudflareCredentials{}, fmt.Errorf("cloudflare.json missing \"account_id\"")
	}
	if creds.APIToken == "" {
		return CloudflareCredentials{}, fmt.Errorf("cloudflare.json missing \"api_token\"")
	}
	return creds, nil
}

func (c CloudflareCredentials) crawlBaseURL() string {
	return fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering/crawl", c.AccountID)
}

// --- CF crawl options (maps directly to CF API request body) ---

// CFOptions controls Cloudflare Browser Rendering crawl behaviour.
// Zero values use CF defaults.
type CFOptions struct {
	// Source controls which links to follow: "all" (default), "sitemaps", "links".
	Source string
	// Render: nil = static HTML fetch (our default), true = full JS rendering.
	// Set to true explicitly with --cf-render.
	Render *bool
	// Formats: "markdown" (default), "html", "json".
	Formats []string
	// RejectResourceTypes blocks resource types for faster crawls.
	// Common values: "image", "media", "font", "stylesheet".
	RejectResourceTypes []string
	// IncludeSubdomains follows links to subdomains of the seed domain.
	IncludeSubdomains bool
	// IncludePatterns restricts crawl to URLs matching these wildcard patterns.
	IncludePatterns []string
	// ExcludePatterns skips URLs matching these wildcard patterns.
	ExcludePatterns []string
	// WaitForSelector waits for a CSS selector before extracting content.
	WaitForSelector string
	// WaitForSelectorTimeout is the max wait time for WaitForSelector (default 5000ms).
	WaitForSelectorTimeout int
	// GotoWaitUntil controls when navigation is considered complete:
	// "load", "domcontentloaded", "networkidle0", "networkidle2".
	GotoWaitUntil string
	// GotoTimeout overrides the per-page navigation timeout (ms).
	GotoTimeout int
	// UserAgent sets a custom User-Agent header.
	UserAgent string
	// ExtraHeaders sets additional HTTP headers for every request.
	ExtraHeaders map[string]string
	// MaxAge is the maximum cache age in seconds CF can reuse (default 86400).
	MaxAge int
	// ModifiedSince is a Unix timestamp; only crawl pages modified after this.
	ModifiedSince int64
}

// cfStartRequest is the POST /crawl body sent to the CF API.
type cfStartRequest struct {
	URL                 string            `json:"url"`
	Limit               int               `json:"limit,omitempty"`
	Depth               int               `json:"depth,omitempty"`
	Source              string            `json:"source,omitempty"`
	Formats             []string          `json:"formats,omitempty"`
	Render              *bool             `json:"render,omitempty"`
	RejectResourceTypes []string          `json:"rejectResourceTypes,omitempty"`
	Options             *cfCrawlOptions   `json:"options,omitempty"`
	WaitForSelector     *cfSelectorOpts   `json:"waitForSelector,omitempty"`
	GotoOptions         *cfGotoOptions    `json:"gotoOptions,omitempty"`
	UserAgent           string            `json:"userAgent,omitempty"`
	SetExtraHTTPHeaders map[string]string `json:"setExtraHTTPHeaders,omitempty"`
	MaxAge              int               `json:"maxAge,omitempty"`
	ModifiedSince       int64             `json:"modifiedSince,omitempty"`
}

type cfCrawlOptions struct {
	IncludeSubdomains bool     `json:"includeSubdomains,omitempty"`
	IncludePatterns   []string `json:"includePatterns,omitempty"`
	ExcludePatterns   []string `json:"excludePatterns,omitempty"`
}

type cfSelectorOpts struct {
	Selector string `json:"selector"`
	Timeout  int    `json:"timeout,omitempty"` // ms
	Visible  bool   `json:"visible,omitempty"`
}

type cfGotoOptions struct {
	WaitUntil string `json:"waitUntil,omitempty"`
	Timeout   int    `json:"timeout,omitempty"` // ms
}

// boolPtr returns a pointer to a bool value.
func boolPtr(v bool) *bool { return &v }

// cfRateLimitError builds a clear error message for HTTP 429 responses,
// including the reset time from the Retry-After header when present.
func cfRateLimitError(resp *http.Response, body []byte) error {
	msg := "CF Browser Rendering rate limit exceeded"

	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			resetAt := time.Now().Add(time.Duration(secs) * time.Second)
			dur := time.Duration(secs) * time.Second
			h := int(dur.Hours())
			m := int(dur.Minutes()) % 60
			s := int(dur.Seconds()) % 60
			var durStr string
			if h > 0 {
				durStr = fmt.Sprintf("%dh %dm", h, m)
			} else if m > 0 {
				durStr = fmt.Sprintf("%dm %ds", m, s)
			} else {
				durStr = fmt.Sprintf("%ds", s)
			}
			msg += fmt.Sprintf("\n  Retry-After: %s (resets at %s)",
				durStr, resetAt.Format("15:04:05 MST"))
		}
	}

	// This is typically the free-plan 10 min/day browser time quota.
	msg += "\n  Free plan: 10 min/day browser time. Upgrade at dash.cloudflare.com"
	return fmt.Errorf("%s", msg)
}

// buildCFRequest assembles a cfStartRequest from seed URL, limit/depth, and options.
// Default: render=false (fast static HTML fetch). Pass Render=boolPtr(true) for JS rendering.
func buildCFRequest(seedURL string, limit, depth int, opts CFOptions) cfStartRequest {
	render := opts.Render
	if render == nil {
		render = boolPtr(false) // our default: static fetch is faster and sufficient for most sites
	}
	req := cfStartRequest{
		URL:    seedURL,
		Limit:  limit,
		Depth:  depth,
		Source: opts.Source,
		Render: render,
	}

	// Formats — default to markdown if not specified
	if len(opts.Formats) > 0 {
		req.Formats = opts.Formats
	} else {
		req.Formats = []string{"markdown"}
	}

	if len(opts.RejectResourceTypes) > 0 {
		req.RejectResourceTypes = opts.RejectResourceTypes
	}
	if opts.IncludeSubdomains || len(opts.IncludePatterns) > 0 || len(opts.ExcludePatterns) > 0 {
		req.Options = &cfCrawlOptions{
			IncludeSubdomains: opts.IncludeSubdomains,
			IncludePatterns:   opts.IncludePatterns,
			ExcludePatterns:   opts.ExcludePatterns,
		}
	}
	if opts.WaitForSelector != "" {
		timeout := opts.WaitForSelectorTimeout
		if timeout == 0 {
			timeout = 5000
		}
		req.WaitForSelector = &cfSelectorOpts{
			Selector: opts.WaitForSelector,
			Timeout:  timeout,
		}
	}
	if opts.GotoWaitUntil != "" || opts.GotoTimeout > 0 {
		req.GotoOptions = &cfGotoOptions{
			WaitUntil: opts.GotoWaitUntil,
			Timeout:   opts.GotoTimeout,
		}
	}
	if opts.UserAgent != "" {
		req.UserAgent = opts.UserAgent
	}
	if len(opts.ExtraHeaders) > 0 {
		req.SetExtraHTTPHeaders = opts.ExtraHeaders
	}
	if opts.MaxAge > 0 {
		req.MaxAge = opts.MaxAge
	}
	if opts.ModifiedSince > 0 {
		req.ModifiedSince = opts.ModifiedSince
	}
	return req
}

// --- API response types ---

type cfStartResponse struct {
	Success bool         `json:"success"`
	Result  string       `json:"result"` // job ID
	Errors  []cfAPIError `json:"errors"`
}

type cfAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfStatusResponse struct {
	Success bool         `json:"success"`
	Result  cfJobResult  `json:"result"`
	Errors  []cfAPIError `json:"errors"`
}

type cfJobResult struct {
	ID                 string         `json:"id"`
	Status             string         `json:"status"` // running|completed|errored|cancelled_*
	BrowserSecondsUsed float64        `json:"browserSecondsUsed"`
	Total              int            `json:"total"`
	Finished           int            `json:"finished"`
	Records            []cfPageRecord `json:"records"`
	// Cursor is the next offset to use for pagination.
	// CF may return it as a number or string; we accept both via json.Number.
	Cursor json.Number `json:"cursor"`
}

func (r *cfJobResult) cursorInt() int {
	if r.Cursor == "" {
		return 0
	}
	n, _ := r.Cursor.Int64()
	return int(n)
}

// cfPageRecord is a single crawled-page result from the CF API.
type cfPageRecord struct {
	URL      string     `json:"url"`
	Status   string     `json:"status"` // completed|queued|disallowed|skipped|errored|cancelled
	Markdown *string    `json:"markdown"`
	HTML     *string    `json:"html"`
	Metadata cfPageMeta `json:"metadata"`
}

type cfPageMeta struct {
	Status int    `json:"status"` // HTTP status code
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// --- Client ---

// CloudflareClient interacts with the Cloudflare Browser Rendering REST API.
type CloudflareClient struct {
	creds  CloudflareCredentials
	client *http.Client
}

// NewCloudflareClient creates a client using the loaded credentials.
func NewCloudflareClient(creds CloudflareCredentials) *CloudflareClient {
	return &CloudflareClient{
		creds:  creds,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// StartCrawl submits a new crawl job and returns the job ID.
func (c *CloudflareClient) StartCrawl(ctx context.Context, seedURL string, limit, depth int, opts CFOptions) (string, error) {
	reqBody := buildCFRequest(seedURL, limit, depth, opts)
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.creds.crawlBaseURL(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.creds.APIToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("start crawl: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode == 429 {
		return "", cfRateLimitError(resp, respBytes)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("CF API %d: %s", resp.StatusCode, string(respBytes))
	}

	var startResp cfStartResponse
	if err := json.Unmarshal(respBytes, &startResp); err != nil {
		return "", fmt.Errorf("decode start response: %w", err)
	}
	if !startResp.Success || startResp.Result == "" {
		if len(startResp.Errors) > 0 {
			return "", fmt.Errorf("CF API error: %s", startResp.Errors[0].Message)
		}
		return "", fmt.Errorf("CF API returned no job ID")
	}
	return startResp.Result, nil
}

// PollCrawl fetches the current job status and a page of records from cursor.
func (c *CloudflareClient) PollCrawl(ctx context.Context, jobID string, cursor, pageLimit int) (*cfJobResult, error) {
	u := fmt.Sprintf("%s/%s?cursor=%d&limit=%d", c.creds.crawlBaseURL(), url.PathEscape(jobID), cursor, pageLimit)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.creds.APIToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll crawl: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("CF API %d: %s", resp.StatusCode, string(respBytes))
	}

	var statusResp cfStatusResponse
	if err := json.Unmarshal(respBytes, &statusResp); err != nil {
		return nil, fmt.Errorf("decode poll response: %w", err)
	}
	if !statusResp.Success {
		if len(statusResp.Errors) > 0 {
			return nil, fmt.Errorf("CF API error: %s", statusResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("CF API returned failure")
	}
	return &statusResp.Result, nil
}

// --- Live job status (shared between poll loop and display loop) ---

type cfJobStatus struct {
	mu       sync.Mutex
	status   string
	total    int
	finished int
	browser  float64
	pollErr  string
}

func (s *cfJobStatus) update(status string, total, finished int, browser float64) {
	s.mu.Lock()
	s.status = status
	s.total = total
	s.finished = finished
	s.browser = browser
	s.mu.Unlock()
}

func (s *cfJobStatus) get() (status string, total, finished int, browser float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, s.total, s.finished, s.browser
}

// --- Poll loop ---

const (
	cfPollInterval = 3 * time.Second
	cfPageSize     = 100
)

// cfPollLoop polls the CF API and sends records to ch as they arrive.
// Advances cursor with each page; closes ch when the job is done and all
// records have been delivered ("capture and pool for status updates along the way").
func cfPollLoop(ctx context.Context, cf *CloudflareClient, jobID string,
	ch chan<- cfPageRecord, jobStatus *cfJobStatus) {

	defer close(ch)

	cursor := 0
	pollTicker := time.NewTicker(cfPollInterval)
	defer pollTicker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}

		result, err := cf.PollCrawl(ctx, jobID, cursor, cfPageSize)
		if err != nil {
			jobStatus.mu.Lock()
			jobStatus.pollErr = err.Error()
			jobStatus.mu.Unlock()
			// Back off on transient errors
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
			}
			continue
		}

		jobStatus.update(result.Status, result.Total, result.Finished, result.BrowserSecondsUsed)

		// Deliver records to the consumer
		for _, rec := range result.Records {
			select {
			case ch <- rec:
			case <-ctx.Done():
				return
			}
		}

		if len(result.Records) > 0 {
			cursor = result.cursorInt()
		}

		if result.Status != "running" {
			// Job finished — drain remaining pages without sleeping
			if len(result.Records) < cfPageSize {
				return // no more pages
			}
			continue // immediately fetch next page
		}

		// Job still running — wait before next poll
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
		}
	}
}

// --- RunCloudflareCrawl ---

// RunCloudflareCrawl submits a Cloudflare Browser Rendering crawl job and
// streams results into the domain's ResultDB with live progress display.
//
// Records are captured and stored as they arrive during polling — not only
// when the job completes ("capture and pool for status updates along the way").
func RunCloudflareCrawl(ctx context.Context, cfg Config, seedURL string, limit, depth int, opts CFOptions) error {
	creds, err := LoadCloudflareCredentials()
	if err != nil {
		return fmt.Errorf("load cloudflare credentials: %w", err)
	}
	cf := NewCloudflareClient(creds)

	fmt.Printf("  Submitting CF crawl job for %s\n", seedURL)
	if limit > 0 {
		fmt.Printf("  Limit: %d pages  Depth: %d\n", limit, depth)
	}

	jobID, err := cf.StartCrawl(ctx, seedURL, limit, depth, opts)
	if err != nil {
		return fmt.Errorf("start CF crawl: %w", err)
	}
	fmt.Printf("  Job ID: %s\n\n", jobID)

	// Open result DB
	if err := os.MkdirAll(cfg.ResultDir(), 0o755); err != nil {
		return fmt.Errorf("create result dir: %w", err)
	}
	rdb, err := NewResultDB(cfg.ResultDir(), cfg.ShardCount, cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("open result DB: %w", err)
	}
	defer rdb.Close()

	jobStatus := &cfJobStatus{status: "running"}

	// Launch poll loop — streams records into ch
	ch := make(chan cfPageRecord, 200)
	go cfPollLoop(ctx, cf, jobID, ch, jobStatus)

	// Display goroutine — prints a rolling progress line every second.
	var ok, fail atomic.Int64
	stopDisplay := make(chan struct{})
	displayStopped := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(displayStopped)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				status, total, finished, browser := jobStatus.get()
				elapsed := time.Since(start).Truncate(time.Second)
				received := ok.Load() + fail.Load()
				fmt.Printf("\r\033[K\033[1;36m%s\033[0m  \033[1;32m%d ok\033[0m",
					elapsed, ok.Load())
				if f := fail.Load(); f > 0 {
					fmt.Printf("  \033[1;31m%d err\033[0m", f)
				}
				fmt.Printf("  recv:%d  CF:%d/%d  browser:%.1fs  [%s]",
					received, finished, total, browser, status)
			case <-stopDisplay:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Main loop: consume records from poll loop and store to DB as they arrive.
	var errLines []string
	for rec := range ch {
		if errLine := storeCFRecord(rdb, &ok, &fail, rec); errLine != "" {
			errLines = append(errLines, errLine)
		}
	}

	// Stop display and wait for it to exit before printing final summary.
	close(stopDisplay)
	<-displayStopped
	fmt.Println() // newline after rolling progress line

	finalStatus, total, finished, browser := jobStatus.get()
	totalStored := ok.Load() + fail.Load()
	elapsed := time.Since(start).Truncate(time.Second)

	switch finalStatus {
	case "completed":
		fmt.Printf("\033[1;32m  Completed\033[0m  %d/%d pages  stored:%d  browser:%.1fs  elapsed:%s\n",
			finished, total, totalStored, browser, elapsed)
	case "errored":
		return fmt.Errorf("CF crawl job errored (stored %d pages)", totalStored)
	case "cancelled_due_to_timeout":
		fmt.Printf("\033[1;33m  Cancelled (timeout)\033[0m  stored:%d  browser:%.1fs\n",
			totalStored, browser)
	case "cancelled_due_to_limits":
		fmt.Printf("\033[1;33m  Cancelled (limit)\033[0m  stored:%d  browser:%.1fs\n",
			totalStored, browser)
	case "cancelled_by_user":
		fmt.Printf("  Cancelled by user  stored:%d\n", totalStored)
	default:
		fmt.Printf("  Status: %s  stored:%d\n", finalStatus, totalStored)
	}
	fmt.Printf("  Results: %s\n", rdb.Dir())
	if len(errLines) > 0 {
		fmt.Printf("  \033[1;31mErrors (%d):\033[0m\n", len(errLines))
		for _, l := range errLines {
			fmt.Printf("    %s\n", l)
		}
	}
	return nil
}

// storeCFRecord stores a CF page record to the ResultDB and updates counters.
// Returns a non-empty error description string if the record was a failure worth logging.
// CF-side skips (disallowed, queued, skipped, cancelled) are silently dropped.
func storeCFRecord(rdb *ResultDB, ok, fail *atomic.Int64, rec cfPageRecord) string {
	switch rec.Status {
	case "disallowed", "queued", "skipped", "cancelled":
		// CF-side decision — not a crawl error on our end; don't store or count.
		return ""
	}

	urlHash := xxhash.Sum64String(rec.URL)
	result := Result{
		URL:        rec.URL,
		URLHash:    urlHash,
		StatusCode: rec.Metadata.Status,
		Title:      rec.Metadata.Title,
		CrawledAt:  time.Now(),
	}
	if rec.Markdown != nil {
		result.Markdown = *rec.Markdown
	}
	if rec.HTML != nil {
		if compressed, err := zstd.Compress(nil, []byte(*rec.HTML)); err == nil {
			result.HTML = compressed
			result.BodyHash = xxhash.Sum64String(*rec.HTML)
			if result.ContentLength == 0 {
				result.ContentLength = int64(len(*rec.HTML))
			}
		}
	}

	isOK := rec.Status == "completed" && rec.Metadata.Status >= 200 && rec.Metadata.Status < 400
	var errLine string

	switch {
	case rec.Status == "errored":
		result.Error = "cf-crawl-errored"
		rdb.AddPage(result)
		fail.Add(1)
		errLine = fmt.Sprintf("\033[1;31merrored  \033[0m%s", rec.URL)
	case isOK:
		rdb.AddPage(result)
		ok.Add(1)
	default:
		// completed but non-2xx (redirect chain end, 4xx, 5xx, etc.)
		rdb.AddPage(result)
		fail.Add(1)
		errLine = fmt.Sprintf("\033[1;33mHTTP %-3d \033[0m%s", rec.Metadata.Status, rec.URL)
	}

	return errLine
}
