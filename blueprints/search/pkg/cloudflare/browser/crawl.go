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
