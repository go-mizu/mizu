# pkg/cloudflare/browser Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `pkg/cloudflare/browser` — a standalone Go package covering all 9 CF Browser Rendering REST API endpoints with integration tests using sqlite.org.

**Architecture:** Two files: `client.go` holds the Client, shared types, HTTP helpers, and all 8 synchronous endpoint methods; `crawl.go` holds the async crawl types and methods. `browser_test.go` has integration tests that skip automatically without credentials.

**Tech Stack:** Go stdlib only (`net/http`, `encoding/json`, `os`). No external dependencies.

**Spec:** `spec/0713_pkg_cloudflare_browser.md`

---

## Chunk 1: Scaffold + Client Infrastructure

### Task 1: Create package scaffold and Client

**Files:**
- Create: `pkg/cloudflare/browser/client.go`

- [ ] **Step 1: Create `pkg/cloudflare/browser/client.go`** with Credentials, LoadCredentials, Client, NewClient, and private HTTP helpers.

```go
package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const baseURLTemplate = "https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering"

// Credentials holds CF account credentials.
type Credentials struct {
	AccountID string `json:"account_id"`
	APIToken  string `json:"api_token"`
}

// LoadCredentials reads credentials from $HOME/data/cloudflare/cloudflare.json.
func LoadCredentials() (Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Credentials{}, fmt.Errorf("get home dir: %w", err)
	}
	path := filepath.Join(home, "data", "cloudflare", "cloudflare.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, fmt.Errorf("read %s: %w", path, err)
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return Credentials{}, fmt.Errorf("parse cloudflare.json: %w", err)
	}
	if creds.AccountID == "" {
		return Credentials{}, fmt.Errorf("cloudflare.json missing account_id")
	}
	if creds.APIToken == "" {
		return Credentials{}, fmt.Errorf("cloudflare.json missing api_token")
	}
	return creds, nil
}

// Client sends requests to the CF Browser Rendering REST API.
type Client struct {
	creds   Credentials
	http    *http.Client
	baseURL string
}

// NewClient creates a new Client.
func NewClient(creds Credentials) *Client {
	return &Client{
		creds:   creds,
		http:    &http.Client{Timeout: 60 * time.Second},
		baseURL: fmt.Sprintf(baseURLTemplate, creds.AccountID),
	}
}

// apiResponse is the CF envelope for JSON endpoints.
type apiResponse struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []apiError      `json:"errors"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// postJSON POSTs JSON and decodes the envelope, placing result into out.
func (c *Client) postJSON(path string, req, out any) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.creds.APIToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}

	var env apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !env.Success {
		if len(env.Errors) > 0 {
			return fmt.Errorf("API error %d: %s", env.Errors[0].Code, env.Errors[0].Message)
		}
		return fmt.Errorf("API returned success=false")
	}
	if out != nil {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}

// postBinary POSTs JSON and returns the raw binary response body.
func (c *Client) postBinary(path string, req any) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.creds.APIToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("POST %s: HTTP %d: %s", path, resp.StatusCode, b)
	}
	return io.ReadAll(resp.Body)
}

// getJSON sends GET to rawURL and decodes the envelope into out.
func (c *Client) getJSON(rawURL string, out any) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.creds.APIToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}

	var env apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !env.Success {
		if len(env.Errors) > 0 {
			return fmt.Errorf("API error %d: %s", env.Errors[0].Code, env.Errors[0].Message)
		}
		return fmt.Errorf("API returned success=false")
	}
	if out != nil {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}

// doDelete sends DELETE to rawURL.
func (c *Client) doDelete(rawURL string) error {
	req, err := http.NewRequest(http.MethodDelete, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.creds.APIToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("DELETE: HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}
```

Shared request/response types (append to `client.go`):

```go
// --- Shared option types ---

// GotoOptions controls page navigation behaviour.
type GotoOptions struct {
	WaitUntil string `json:"waitUntil,omitempty"` // load|domcontentloaded|networkidle0|networkidle2
	Timeout   int    `json:"timeout,omitempty"`   // ms
}

// WaitForSelector waits for a CSS selector before returning.
type WaitForSelector struct {
	Selector string `json:"selector"`
	Timeout  int    `json:"timeout,omitempty"` // ms
	Visible  bool   `json:"visible,omitempty"`
}

// Viewport sets the browser viewport size.
type Viewport struct {
	Width             int     `json:"width,omitempty"`
	Height            int     `json:"height,omitempty"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor,omitempty"`
}

// Cookie sets a browser cookie.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	URL   string `json:"url,omitempty"`
}

// Authenticate provides HTTP Basic Auth credentials.
type Authenticate struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ScriptTag injects a JavaScript tag.
type ScriptTag struct {
	URL     string `json:"url,omitempty"`
	Content string `json:"content,omitempty"`
}

// StyleTag injects a CSS tag.
type StyleTag struct {
	URL     string `json:"url,omitempty"`
	Content string `json:"content,omitempty"`
}

// CommonRequest holds fields shared across most Browser Rendering endpoints.
// At least one of URL or HTML must be set.
type CommonRequest struct {
	URL                  string            `json:"url,omitempty"`
	HTML                 string            `json:"html,omitempty"`
	UserAgent            string            `json:"userAgent,omitempty"`
	Authenticate         *Authenticate     `json:"authenticate,omitempty"`
	Cookies              []Cookie          `json:"cookies,omitempty"`
	GotoOptions          *GotoOptions      `json:"gotoOptions,omitempty"`
	WaitForSelector      *WaitForSelector  `json:"waitForSelector,omitempty"`
	SetExtraHTTPHeaders  map[string]string `json:"setExtraHTTPHeaders,omitempty"`
	RejectResourceTypes  []string          `json:"rejectResourceTypes,omitempty"`  // image|media|font|stylesheet
	RejectRequestPattern []string          `json:"rejectRequestPattern,omitempty"` // regex
}

// --- Endpoint request/response types ---

// ContentRequest fetches rendered HTML.
type ContentRequest struct{ CommonRequest }

// MarkdownRequest extracts Markdown from a page.
type MarkdownRequest struct{ CommonRequest }

// LinksRequest extracts hyperlinks from a page.
type LinksRequest struct {
	CommonRequest
	VisibleLinksOnly     bool `json:"visibleLinksOnly,omitempty"`
	ExcludeExternalLinks bool `json:"excludeExternalLinks,omitempty"`
}

// ScreenshotOptions controls screenshot capture.
type ScreenshotOptions struct {
	OmitBackground        bool   `json:"omitBackground,omitempty"`
	FullPage              bool   `json:"fullPage,omitempty"`
	Quality               int    `json:"quality,omitempty"`
	Type                  string `json:"type,omitempty"` // png (default) | jpeg
	CaptureBeyondViewport bool   `json:"captureBeyondViewport,omitempty"`
}

// ScreenshotRequest captures a screenshot.
type ScreenshotRequest struct {
	CommonRequest
	ScreenshotOptions *ScreenshotOptions `json:"screenshotOptions,omitempty"`
	Viewport          *Viewport          `json:"viewport,omitempty"`
	Selector          string             `json:"selector,omitempty"`
	AddScriptTag      []ScriptTag        `json:"addScriptTag,omitempty"`
	AddStyleTag       []StyleTag         `json:"addStyleTag,omitempty"`
}

// PDFOptions controls PDF rendering.
type PDFOptions struct {
	Format          string  `json:"format,omitempty"` // A4|Letter|etc
	Landscape       bool    `json:"landscape,omitempty"`
	Scale           float64 `json:"scale,omitempty"`
	PrintBackground bool    `json:"printBackground,omitempty"`
}

// PDFRequest renders a page as PDF.
type PDFRequest struct {
	CommonRequest
	PDFOptions          *PDFOptions `json:"pdfOptions,omitempty"`
	Viewport            *Viewport   `json:"viewport,omitempty"`
	DisplayHeaderFooter bool        `json:"displayHeaderFooter,omitempty"`
	HeaderTemplate      string      `json:"headerTemplate,omitempty"`
	FooterTemplate      string      `json:"footerTemplate,omitempty"`
	AddStyleTag         []StyleTag  `json:"addStyleTag,omitempty"`
}

// SnapshotRequest captures both screenshot and HTML in one call.
type SnapshotRequest struct {
	CommonRequest
	ScreenshotOptions    *ScreenshotOptions `json:"screenshotOptions,omitempty"`
	Viewport             *Viewport          `json:"viewport,omitempty"`
	AddScriptTag         []ScriptTag        `json:"addScriptTag,omitempty"`
	SetJavaScriptEnabled *bool              `json:"setJavaScriptEnabled,omitempty"`
}

// SnapshotResult contains a base64 screenshot and the page HTML.
type SnapshotResult struct {
	Screenshot string `json:"screenshot"` // base64-encoded PNG
	Content    string `json:"content"`    // HTML
}

// ScrapeElement specifies a CSS selector to scrape.
type ScrapeElement struct {
	Selector string `json:"selector"`
}

// ScrapeRequest extracts HTML elements matching CSS selectors.
type ScrapeRequest struct {
	CommonRequest
	Elements []ScrapeElement `json:"elements"`
}

// ScrapedAttribute is a single HTML attribute key/value pair.
type ScrapedAttribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ScrapedElement holds the extracted data for one DOM element.
type ScrapedElement struct {
	Text       string             `json:"text"`
	HTML       string             `json:"html"`
	Attributes []ScrapedAttribute `json:"attributes"`
	Height     float64            `json:"height"`
	Width      float64            `json:"width"`
	Top        float64            `json:"top"`
	Left       float64            `json:"left"`
}

// ScrapeSelector groups results for one CSS selector.
type ScrapeSelector struct {
	Selector string           `json:"selector"`
	Results  []ScrapedElement `json:"results"`
}

// JSONRequest extracts structured data from a page using AI.
// At least one of Prompt or ResponseFormat must be set.
type JSONRequest struct {
	CommonRequest
	Prompt         string         `json:"prompt,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

// --- Synchronous endpoint methods ---

// Content fetches the rendered HTML of a page.
func (c *Client) Content(req ContentRequest) (string, error) {
	var result string
	return result, c.postJSON("/content", req, &result)
}

// Markdown extracts Markdown from a page.
func (c *Client) Markdown(req MarkdownRequest) (string, error) {
	var result string
	return result, c.postJSON("/markdown", req, &result)
}

// Links extracts hyperlinks from a page.
func (c *Client) Links(req LinksRequest) ([]string, error) {
	var result []string
	return result, c.postJSON("/links", req, &result)
}

// Screenshot captures a screenshot of a page and returns raw PNG/JPEG bytes.
func (c *Client) Screenshot(req ScreenshotRequest) ([]byte, error) {
	return c.postBinary("/screenshot", req)
}

// PDF renders a page as PDF and returns raw PDF bytes.
func (c *Client) PDF(req PDFRequest) ([]byte, error) {
	return c.postBinary("/pdf", req)
}

// Snapshot captures both a screenshot and the HTML in one request.
func (c *Client) Snapshot(req SnapshotRequest) (*SnapshotResult, error) {
	var result SnapshotResult
	return &result, c.postJSON("/snapshot", req, &result)
}

// Scrape extracts HTML elements matching CSS selectors.
func (c *Client) Scrape(req ScrapeRequest) ([]ScrapeSelector, error) {
	var result []ScrapeSelector
	return result, c.postJSON("/scrape", req, &result)
}

// JSON extracts structured data from a page using AI.
func (c *Client) JSON(req JSONRequest) (map[string]any, error) {
	var result map[string]any
	return result, c.postJSON("/json", req, &result)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/cloudflare/browser/
```

Expected: no errors.

- [ ] **Step 3: Commit scaffold**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
git add pkg/cloudflare/browser/client.go spec/0713_pkg_cloudflare_browser.md
git commit -m "feat(cloudflare/browser): scaffold Client + all sync endpoint types and methods"
```

---

### Task 2: Crawl endpoints

**Files:**
- Create: `pkg/cloudflare/browser/crawl.go`

- [ ] **Step 1: Create `pkg/cloudflare/browser/crawl.go`**

```go
package browser

import (
	"fmt"
	"net/url"
	"strconv"
)

// CrawlOptions controls which links are followed during a crawl.
type CrawlOptions struct {
	IncludeSubdomains    bool     `json:"includeSubdomains,omitempty"`
	IncludeExternalLinks bool     `json:"includeExternalLinks,omitempty"`
	IncludePatterns      []string `json:"includePatterns,omitempty"`
	ExcludePatterns      []string `json:"excludePatterns,omitempty"`
}

// CrawlRequest starts an async crawl job.
type CrawlRequest struct {
	URL                 string            `json:"url"`
	Limit               int               `json:"limit,omitempty"`         // max pages; CF default 10
	Depth               int               `json:"depth,omitempty"`         // max link depth
	Source              string            `json:"source,omitempty"`        // all|sitemaps|links
	Formats             []string          `json:"formats,omitempty"`       // html|markdown|json
	Render              *bool             `json:"render,omitempty"`        // JS rendering; CF default true
	MaxAge              int               `json:"maxAge,omitempty"`        // cache TTL seconds
	ModifiedSince       int64             `json:"modifiedSince,omitempty"` // Unix timestamp
	UserAgent           string            `json:"userAgent,omitempty"`
	Authenticate        *Authenticate     `json:"authenticate,omitempty"`
	SetExtraHTTPHeaders map[string]string `json:"setExtraHTTPHeaders,omitempty"`
	RejectResourceTypes []string          `json:"rejectResourceTypes,omitempty"`
	GotoOptions         *GotoOptions      `json:"gotoOptions,omitempty"`
	WaitForSelector     *WaitForSelector  `json:"waitForSelector,omitempty"`
	Options             *CrawlOptions     `json:"options,omitempty"`
}

// CrawlJob is returned by StartCrawl.
type CrawlJob struct {
	ID                 string  `json:"id"`
	Status             string  `json:"status"` // running|completed|errored|cancelled_*
	BrowserSecondsUsed float64 `json:"browserSecondsUsed"`
	Total              int     `json:"total"`
	Finished           int     `json:"finished"`
}

// GetCrawlOptions are optional query params for GetCrawl.
type GetCrawlOptions struct {
	Cursor string // pagination token
	Limit  int    // max records per response
	Status string // filter: queued|completed|disallowed|skipped|errored|cancelled
}

// CrawlRecordMetadata holds HTTP-level metadata for a crawled page.
type CrawlRecordMetadata struct {
	Status int    `json:"status"` // HTTP status code
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// CrawlRecord holds the result for one crawled URL.
type CrawlRecord struct {
	URL      string               `json:"url"`
	Status   string               `json:"status"` // completed|queued|disallowed|skipped|errored|cancelled
	Markdown string               `json:"markdown,omitempty"`
	HTML     string               `json:"html,omitempty"`
	JSON     map[string]any       `json:"json,omitempty"`
	Metadata *CrawlRecordMetadata `json:"metadata,omitempty"`
}

// CrawlResult is returned by GetCrawl.
type CrawlResult struct {
	ID                 string        `json:"id"`
	Status             string        `json:"status"`
	BrowserSecondsUsed float64       `json:"browserSecondsUsed"`
	Total              int           `json:"total"`
	Finished           int           `json:"finished"`
	Records            []CrawlRecord `json:"records"`
	Cursor             string        `json:"cursor"`
}

// StartCrawl submits an async crawl job and returns the job metadata including ID.
func (c *Client) StartCrawl(req CrawlRequest) (*CrawlJob, error) {
	var job CrawlJob
	return &job, c.postJSON("/crawl", req, &job)
}

// GetCrawl retrieves crawl job status and records (paginated).
func (c *Client) GetCrawl(jobID string, opts *GetCrawlOptions) (*CrawlResult, error) {
	rawURL := fmt.Sprintf("%s/crawl/%s", c.baseURL, jobID)
	if opts != nil {
		q := url.Values{}
		if opts.Cursor != "" {
			q.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Status != "" {
			q.Set("status", opts.Status)
		}
		if len(q) > 0 {
			rawURL += "?" + q.Encode()
		}
	}
	var result CrawlResult
	return &result, c.getJSON(rawURL, &result)
}

// DeleteCrawl cancels a running crawl job.
func (c *Client) DeleteCrawl(jobID string) error {
	return c.doDelete(fmt.Sprintf("%s/crawl/%s", c.baseURL, jobID))
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/cloudflare/browser/
```

Expected: no errors.

- [ ] **Step 3: Commit crawl endpoints**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
git add pkg/cloudflare/browser/crawl.go
git commit -m "feat(cloudflare/browser): add async crawl endpoints (StartCrawl/GetCrawl/DeleteCrawl)"
```

---

## Chunk 2: Integration Tests

### Task 3: Integration tests (sqlite.org)

**Files:**
- Create: `pkg/cloudflare/browser/browser_test.go`

- [ ] **Step 1: Create `pkg/cloudflare/browser/browser_test.go`**

```go
package browser_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cloudflare/browser"
)

const testURL = "https://sqlite.org"

func newTestClient(t *testing.T) *browser.Client {
	t.Helper()
	creds, err := browser.LoadCredentials()
	if err != nil {
		t.Skipf("no CF credentials (%v); skipping integration test", err)
	}
	return browser.NewClient(creds)
}

func TestContent(t *testing.T) {
	c := newTestClient(t)
	html, err := c.Content(browser.ContentRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(html), "sqlite") {
		t.Errorf("expected 'sqlite' in content, got first 200 chars: %.200s", html)
	}
}

func TestMarkdown(t *testing.T) {
	c := newTestClient(t)
	md, err := c.Markdown(browser.MarkdownRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(md), "sqlite") {
		t.Errorf("expected 'sqlite' in markdown, got first 200 chars: %.200s", md)
	}
}

func TestLinks(t *testing.T) {
	c := newTestClient(t)
	links, err := c.Links(browser.LinksRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(links) == 0 {
		t.Error("expected at least one link")
	}
	t.Logf("found %d links; first: %s", len(links), links[0])
}

func TestScreenshot(t *testing.T) {
	c := newTestClient(t)
	img, err := c.Screenshot(browser.ScreenshotRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(img) == 0 {
		t.Fatal("expected non-empty screenshot")
	}
	// PNG magic: 89 50 4E 47
	if len(img) < 4 || img[0] != 0x89 || img[1] != 0x50 || img[2] != 0x4E || img[3] != 0x47 {
		t.Errorf("expected PNG magic bytes, got: %x", img[:4])
	}
	t.Logf("screenshot size: %d bytes", len(img))
}

func TestPDF(t *testing.T) {
	c := newTestClient(t)
	pdf, err := c.PDF(browser.PDFRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) == 0 {
		t.Fatal("expected non-empty PDF")
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Errorf("expected PDF magic bytes %%PDF, got: %q", pdf[:min(8, len(pdf))])
	}
	t.Logf("PDF size: %d bytes", len(pdf))
}

func TestSnapshot(t *testing.T) {
	c := newTestClient(t)
	snap, err := c.Snapshot(browser.SnapshotRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Screenshot == "" {
		t.Error("expected non-empty screenshot in snapshot")
	}
	if !strings.Contains(strings.ToLower(snap.Content), "sqlite") {
		t.Error("expected 'sqlite' in snapshot content")
	}
}

func TestScrape(t *testing.T) {
	c := newTestClient(t)
	result, err := c.Scrape(browser.ScrapeRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
		Elements:      []browser.ScrapeElement{{Selector: "a"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one selector result")
	}
	if len(result[0].Results) == 0 {
		t.Error("expected at least one scraped element for selector 'a'")
	}
	t.Logf("scraped %d <a> elements", len(result[0].Results))
}

func TestJSON(t *testing.T) {
	c := newTestClient(t)
	result, err := c.JSON(browser.JSONRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
		Prompt:        "Extract the page title and list the main navigation links",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty JSON result")
	}
	t.Logf("JSON keys: %v", func() []string {
		keys := make([]string, 0, len(result))
		for k := range result {
			keys = append(keys, k)
		}
		return keys
	}())
}

func TestCrawl(t *testing.T) {
	c := newTestClient(t)

	renderFalse := false
	job, err := c.StartCrawl(browser.CrawlRequest{
		URL:    testURL,
		Limit:  3,
		Render: &renderFalse,
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" {
		t.Fatal("expected non-empty job ID")
	}
	t.Logf("started crawl job %s (status=%s)", job.ID, job.Status)

	// Poll until complete (max 3 min).
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)
		result, err := c.GetCrawl(job.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("job %s: status=%s finished=%d/%d", result.ID, result.Status, result.Finished, result.Total)
		if result.Status != "running" {
			if len(result.Records) == 0 {
				t.Error("expected at least one crawl record")
			}
			t.Logf("crawl done: %d records, status=%s", len(result.Records), result.Status)
			return
		}
	}
	// Attempt cleanup.
	_ = c.DeleteCrawl(job.ID)
	t.Error("crawl did not complete within 3 minutes")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/cloudflare/browser/
go vet ./pkg/cloudflare/browser/
```

Expected: no errors.

- [ ] **Step 3: Run tests (expect skip without credentials, or pass with credentials)**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go test ./pkg/cloudflare/browser/ -v -timeout 3m 2>&1 | head -30
```

Expected without credentials: all tests output `SKIP` lines.
Expected with credentials: all tests `PASS` (TestCrawl may take 1-2 min).

- [ ] **Step 4: Commit tests**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
git add pkg/cloudflare/browser/browser_test.go
git commit -m "test(cloudflare/browser): integration tests for all 9 endpoints using sqlite.org"
```

---

## Final Verification

- [ ] **Full build and vet**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./...
go vet ./pkg/cloudflare/browser/
```

- [ ] **Run all tests (with credentials)**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go test ./pkg/cloudflare/browser/ -v -timeout 3m
```

All 9 tests should pass: TestContent, TestMarkdown, TestLinks, TestScreenshot, TestPDF, TestSnapshot, TestScrape, TestJSON, TestCrawl.
