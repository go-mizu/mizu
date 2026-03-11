# spec/0713 — `pkg/cloudflare/browser`: CF Browser Rendering REST API Client

## Overview

A standalone Go package (`pkg/cloudflare/browser`) that wraps every endpoint of
the [Cloudflare Browser Rendering REST API](https://developers.cloudflare.com/browser-rendering/rest-api/).

**Scope**: pure HTTP client only — no DuckDB, no polling loop, no progress display.
Those concerns remain in `pkg/scrape/cloudflare.go`, which is **not** changed.

---

## Credentials

Same file as the existing scrape integration:

```
$HOME/data/cloudflare/cloudflare.json
{"account_id": "...", "api_token": "..."}
```

Token permission: **Account > Browser Rendering > Edit**.

---

## Base URL

```
https://api.cloudflare.com/client/v4/accounts/{accountID}/browser-rendering
```

---

## File Structure

```
pkg/cloudflare/browser/
├── client.go       — Credentials, Client, shared HTTP helpers, all 8 sync endpoint methods
├── crawl.go        — CrawlRequest/Result types, StartCrawl / GetCrawl / DeleteCrawl
└── browser_test.go — Integration tests (sqlite.org), skipped without credentials
```

---

## Shared Types

Defined in `client.go`.

```go
type Credentials struct {
    AccountID string `json:"account_id"`
    APIToken  string `json:"api_token"`
}

type GotoOptions struct {
    WaitUntil string `json:"waitUntil,omitempty"` // load|domcontentloaded|networkidle0|networkidle2
    Timeout   int    `json:"timeout,omitempty"`   // ms
}

type WaitForSelector struct {
    Selector string `json:"selector"`
    Timeout  int    `json:"timeout,omitempty"` // ms
    Visible  bool   `json:"visible,omitempty"`
}

type Viewport struct {
    Width             int     `json:"width,omitempty"`
    Height            int     `json:"height,omitempty"`
    DeviceScaleFactor float64 `json:"deviceScaleFactor,omitempty"`
}

type Cookie struct {
    Name  string `json:"name"`
    Value string `json:"value"`
    URL   string `json:"url,omitempty"`
}

type Authenticate struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type ScriptTag struct {
    URL     string `json:"url,omitempty"`
    Content string `json:"content,omitempty"`
}

type StyleTag struct {
    URL     string `json:"url,omitempty"`
    Content string `json:"content,omitempty"`
}

// CommonRequest holds fields shared across most endpoints.
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
    RejectRequestPattern []string          `json:"rejectRequestPattern,omitempty"` // regex patterns
}
```

---

## Synchronous Endpoints

All return quickly (one page rendered, synchronous HTTP response).

### POST `/content` — Fetch HTML

```go
type ContentRequest struct{ CommonRequest }

func (c *Client) Content(req ContentRequest) (string, error)
```

Response: plain HTML string.

---

### POST `/markdown` — Extract Markdown

```go
type MarkdownRequest struct{ CommonRequest }

func (c *Client) Markdown(req MarkdownRequest) (string, error)
```

Response: Markdown string.

---

### POST `/links` — Extract links

```go
type LinksRequest struct {
    CommonRequest
    VisibleLinksOnly     bool `json:"visibleLinksOnly,omitempty"`
    ExcludeExternalLinks bool `json:"excludeExternalLinks,omitempty"`
}

func (c *Client) Links(req LinksRequest) ([]string, error)
```

Response: `[]string` of URLs.

---

### POST `/screenshot` — Capture screenshot

```go
type ScreenshotOptions struct {
    OmitBackground        bool   `json:"omitBackground,omitempty"`
    FullPage              bool   `json:"fullPage,omitempty"`
    Quality               int    `json:"quality,omitempty"`
    Type                  string `json:"type,omitempty"` // png (default) | jpeg
    CaptureBeyondViewport bool   `json:"captureBeyondViewport,omitempty"`
}

type ScreenshotRequest struct {
    CommonRequest
    ScreenshotOptions *ScreenshotOptions `json:"screenshotOptions,omitempty"`
    Viewport          *Viewport          `json:"viewport,omitempty"`
    Selector          string             `json:"selector,omitempty"`
    AddScriptTag      []ScriptTag        `json:"addScriptTag,omitempty"`
    AddStyleTag       []StyleTag         `json:"addStyleTag,omitempty"`
}

func (c *Client) Screenshot(req ScreenshotRequest) ([]byte, error)
```

Response: raw PNG (or JPEG) bytes.

---

### POST `/pdf` — Render PDF

```go
type PDFOptions struct {
    Format          string  `json:"format,omitempty"`   // A4|Letter|etc
    Landscape       bool    `json:"landscape,omitempty"`
    Scale           float64 `json:"scale,omitempty"`
    PrintBackground bool    `json:"printBackground,omitempty"`
}

type PDFRequest struct {
    CommonRequest
    PDFOptions          *PDFOptions `json:"pdfOptions,omitempty"`
    Viewport            *Viewport   `json:"viewport,omitempty"`
    DisplayHeaderFooter bool        `json:"displayHeaderFooter,omitempty"`
    HeaderTemplate      string      `json:"headerTemplate,omitempty"`
    FooterTemplate      string      `json:"footerTemplate,omitempty"`
    AddStyleTag         []StyleTag  `json:"addStyleTag,omitempty"`
}

func (c *Client) PDF(req PDFRequest) ([]byte, error)
```

Response: raw PDF bytes.

---

### POST `/snapshot` — Screenshot + HTML

```go
type SnapshotRequest struct {
    CommonRequest
    ScreenshotOptions    *ScreenshotOptions `json:"screenshotOptions,omitempty"`
    Viewport             *Viewport          `json:"viewport,omitempty"`
    AddScriptTag         []ScriptTag        `json:"addScriptTag,omitempty"`
    SetJavaScriptEnabled *bool              `json:"setJavaScriptEnabled,omitempty"`
}

type SnapshotResult struct {
    Screenshot string `json:"screenshot"` // base64-encoded PNG
    Content    string `json:"content"`    // HTML
}

func (c *Client) Snapshot(req SnapshotRequest) (*SnapshotResult, error)
```

---

### POST `/scrape` — Extract HTML elements

```go
type ScrapeElement struct {
    Selector string `json:"selector"`
}

type ScrapeRequest struct {
    CommonRequest
    Elements []ScrapeElement `json:"elements"`
}

type ScrapedAttribute struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

type ScrapedElement struct {
    Text       string             `json:"text"`
    HTML       string             `json:"html"`
    Attributes []ScrapedAttribute `json:"attributes"`
    Height     float64            `json:"height"`
    Width      float64            `json:"width"`
    Top        float64            `json:"top"`
    Left       float64            `json:"left"`
}

type ScrapeSelector struct {
    Selector string           `json:"selector"`
    Results  []ScrapedElement `json:"results"`
}

func (c *Client) Scrape(req ScrapeRequest) ([]ScrapeSelector, error)
```

---

### POST `/json` — AI-structured data extraction

```go
type JSONRequest struct {
    CommonRequest
    Prompt         string         `json:"prompt,omitempty"`
    ResponseFormat map[string]any `json:"response_format,omitempty"`
}

func (c *Client) JSON(req JSONRequest) (map[string]any, error)
```

Response: arbitrary JSON object per the prompt/schema.

---

## Async Crawl Endpoints

Defined in `crawl.go`. The job is async: POST starts it, GET polls for results.

### POST `/crawl` — Start crawl

```go
type CrawlOptions struct {
    IncludeSubdomains    bool     `json:"includeSubdomains,omitempty"`
    IncludeExternalLinks bool     `json:"includeExternalLinks,omitempty"`
    IncludePatterns      []string `json:"includePatterns,omitempty"`
    ExcludePatterns      []string `json:"excludePatterns,omitempty"`
}

type CrawlRequest struct {
    URL                 string            `json:"url"`
    Limit               int               `json:"limit,omitempty"`        // max pages; default 10
    Depth               int               `json:"depth,omitempty"`        // max link depth
    Source              string            `json:"source,omitempty"`       // all|sitemaps|links
    Formats             []string          `json:"formats,omitempty"`      // html|markdown|json
    Render              *bool             `json:"render,omitempty"`       // JS rendering; default true
    MaxAge              int               `json:"maxAge,omitempty"`       // cache TTL seconds
    ModifiedSince       int64             `json:"modifiedSince,omitempty"` // Unix timestamp
    UserAgent           string            `json:"userAgent,omitempty"`
    Authenticate        *Authenticate     `json:"authenticate,omitempty"`
    SetExtraHTTPHeaders map[string]string `json:"setExtraHTTPHeaders,omitempty"`
    RejectResourceTypes []string          `json:"rejectResourceTypes,omitempty"`
    GotoOptions         *GotoOptions      `json:"gotoOptions,omitempty"`
    WaitForSelector     *WaitForSelector  `json:"waitForSelector,omitempty"`
    Options             *CrawlOptions     `json:"options,omitempty"`
}

type CrawlJob struct {
    ID                 string  `json:"id"`
    Status             string  `json:"status"`             // running|completed|errored|cancelled_*
    BrowserSecondsUsed float64 `json:"browserSecondsUsed"`
    Total              int     `json:"total"`
    Finished           int     `json:"finished"`
}

func (c *Client) StartCrawl(req CrawlRequest) (*CrawlJob, error)
```

### GET `/crawl/{jobID}` — Poll job + records

```go
type GetCrawlOptions struct {
    Cursor string // pagination token from previous response
    Limit  int    // max records per response
    Status string // filter: queued|completed|disallowed|skipped|errored|cancelled
}

type CrawlRecordMetadata struct {
    Status int    `json:"status"` // HTTP status
    Title  string `json:"title"`
    URL    string `json:"url"`
}

type CrawlRecord struct {
    URL      string               `json:"url"`
    Status   string               `json:"status"` // completed|queued|disallowed|skipped|errored|cancelled
    Markdown string               `json:"markdown,omitempty"`
    HTML     string               `json:"html,omitempty"`
    JSON     map[string]any       `json:"json,omitempty"`
    Metadata *CrawlRecordMetadata `json:"metadata,omitempty"`
}

type CrawlResult struct {
    ID                 string        `json:"id"`
    Status             string        `json:"status"`
    BrowserSecondsUsed float64       `json:"browserSecondsUsed"`
    Total              int           `json:"total"`
    Finished           int           `json:"finished"`
    Records            []CrawlRecord `json:"records"`
    Cursor             string        `json:"cursor"`
}

func (c *Client) GetCrawl(jobID string, opts *GetCrawlOptions) (*CrawlResult, error)
```

### DELETE `/crawl/{jobID}` — Cancel job

```go
func (c *Client) DeleteCrawl(jobID string) error
```

---

## Client

```go
type Client struct { /* unexported */ }

func NewClient(creds Credentials) *Client
func LoadCredentials() (Credentials, error) // reads ~/data/cloudflare/cloudflare.json
```

HTTP client timeout: 60 s (synchronous endpoints). Crawl endpoints use the same client; the caller controls polling timing.

Error handling:
- HTTP 429 → `fmt.Errorf("rate limited (429); retry after %s", retryAfter)`
- `success=false` → `fmt.Errorf("API error %d: %s", code, message)`
- Non-200 on binary endpoints → include status + body excerpt in error

---

## Testing Strategy

`browser_test.go` contains integration tests. Each test calls `newTestClient(t)`:

```go
func newTestClient(t *testing.T) *Client {
    t.Helper()
    creds, err := LoadCredentials()
    if err != nil {
        t.Skipf("no CF credentials (%v); skipping integration test", err)
    }
    return NewClient(creds)
}
```

Tests will be skipped automatically in environments without `~/data/cloudflare/cloudflare.json`.

Test URL: `https://sqlite.org` (publicly available, stable content).

| Test | What it checks |
|------|---------------|
| `TestContent` | HTML contains "SQLite" |
| `TestMarkdown` | Markdown contains "SQLite" |
| `TestLinks` | At least one link returned |
| `TestScreenshot` | len > 0; PNG magic bytes `\x89PNG` |
| `TestPDF` | len > 0; PDF magic bytes `%PDF` |
| `TestSnapshot` | Screenshot non-empty; Content contains "sqlite" |
| `TestScrape` | Selector `"a"` returns ≥1 result |
| `TestJSON` | Prompt returns non-empty map |
| `TestCrawl` | Start 3-page crawl; poll until done; ≥1 record |

Run: `go test ./pkg/cloudflare/browser/ -v -timeout 3m`

---

## Non-goals

- No polling loop or progress display (stays in `pkg/scrape/cloudflare.go`)
- No DuckDB integration
- No CLI commands
- No changes to existing code
