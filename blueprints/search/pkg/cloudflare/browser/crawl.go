package browser

import (
	"encoding/json"
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
	Limit               int               `json:"limit,omitempty"`
	Depth               int               `json:"depth,omitempty"`
	Source              string            `json:"source,omitempty"`        // all|sitemaps|links (CF only)
	Formats             []string          `json:"formats,omitempty"`       // html|markdown|json
	Render              *bool             `json:"render,omitempty"`        // JS rendering; CF default true
	MaxAge              int               `json:"maxAge,omitempty"`        // cache TTL seconds (CF only)
	ModifiedSince       int64             `json:"modifiedSince,omitempty"` // Unix timestamp (CF only)
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

// FlexCursor unmarshals from both JSON string (CF API) and JSON number (proxy).
type FlexCursor string

func (f *FlexCursor) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexCursor(s)
		return nil
	}
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexCursor(strconv.FormatInt(int64(n), 10))
		return nil
	}
	return nil // ignore unknown cursor types — pagination simply won't advance
}

func (f FlexCursor) String() string { return string(f) }

// GetCrawlOptions are optional query params for GetCrawl.
type GetCrawlOptions struct {
	Cursor FlexCursor // pagination token from previous CrawlResult.Cursor
	Limit  int        // max records per response
	Status string     // filter: queued|completed|disallowed|skipped|errored|cancelled
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
	Cursor             FlexCursor    `json:"cursor"`
}

// StartCrawl submits an async crawl job and returns the job metadata including ID.
//
// The proxy worker returns just a job ID string; the CF API returns a full
// CrawlJob object. Both are handled transparently.
func (c *Client) StartCrawl(req CrawlRequest) (*CrawlJob, error) {
	var raw json.RawMessage
	if err := c.postJSON("/crawl", req, &raw); err != nil {
		return nil, err
	}

	// CF API: full CrawlJob object.
	var job CrawlJob
	if err := json.Unmarshal(raw, &job); err == nil && job.ID != "" {
		return &job, nil
	}

	// Proxy: plain job ID string.
	var id string
	if err := json.Unmarshal(raw, &id); err == nil && id != "" {
		return &CrawlJob{ID: id, Status: "running"}, nil
	}

	return nil, fmt.Errorf("unexpected StartCrawl response: %s", raw)
}

// GetCrawl retrieves crawl job status and records (paginated).
func (c *Client) GetCrawl(jobID string, opts *GetCrawlOptions) (*CrawlResult, error) {
	base, _ := c.crawlBase()
	rawURL := fmt.Sprintf("%s/crawl/%s", base, jobID)

	if opts != nil {
		q := url.Values{}
		if opts.Cursor != "" {
			q.Set("cursor", opts.Cursor.String())
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
	base, _ := c.crawlBase()
	return c.doDelete(fmt.Sprintf("%s/crawl/%s", base, jobID))
}
