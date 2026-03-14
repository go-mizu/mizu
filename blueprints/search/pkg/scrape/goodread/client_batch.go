package goodread

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

// BatchHTMLFetcher extends HTMLFetcher with a bulk-fetch method that retrieves
// multiple URLs in a single HTTP call to the crawler worker.
// FetchTask detects this interface and uses batch mode for higher throughput.
type BatchHTMLFetcher interface {
	HTMLFetcher
	FetchHTMLBatch(ctx context.Context, urls []string) ([]BatchHTMLResult, error)
}

// BatchHTMLResult is one item returned from a batch fetch.
type BatchHTMLResult struct {
	URL        string
	Doc        *goquery.Document // nil on error or non-200
	StatusCode int
	Err        error
}

// crawlerRequest is the JSON body sent to POST /crawl on crawler.go-mizu.workers.dev.
type crawlerRequest struct {
	URLs    []string `json:"urls"`
	Timeout int      `json:"timeout,omitempty"`
}

// crawlerResult is one element of the JSON array response.
type crawlerResult struct {
	URL    string  `json:"url"`
	Status int     `json:"status"`
	HTML   *string `json:"html"`
	Error  *string `json:"error"`
}

const defaultCrawlerURL = "https://crawler.go-mizu.workers.dev"

// BatchClient sends batch requests to crawler.go-mizu.workers.dev.
// Each FetchHTMLBatch call fetches all requested URLs in parallel from CF edge.
//
// With batchWorkers=20 and batchSize=50:
//
//	20 concurrent batches × 50 parallel CF fetches / ~9s = ~111 rps
type BatchClient struct {
	url     string
	token   string
	timeout time.Duration
	http    *http.Client
	limiter *rate.Limiter
}

// NewBatchClient creates a BatchClient.
// Token is read from MIZU_TOKEN env var or cfg.WorkerToken.
func NewBatchClient(cfg Config) (*BatchClient, error) {
	token := cfg.WorkerToken
	if token == "" {
		token = os.Getenv("MIZU_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("batch worker token required: set --worker-token or MIZU_TOKEN env var")
	}

	workerURL := cfg.WorkerURL
	if workerURL == "" {
		workerURL = defaultCrawlerURL
	}

	perFetchTimeout := cfg.Timeout
	if perFetchTimeout <= 0 {
		perFetchTimeout = 15 * time.Second
	}

	// HTTP client timeout covers the whole batch roundtrip.
	// A batch of 50 URLs all fetch in parallel; allow 60s for the worst case.
	httpTimeout := 90 * time.Second

	return &BatchClient{
		url:     workerURL,
		token:   token,
		timeout: perFetchTimeout,
		http:    &http.Client{Timeout: httpTimeout},
		limiter: newLimiter(cfg),
	}, nil
}

// FetchHTML satisfies HTMLFetcher by sending a single-URL batch request.
func (c *BatchClient) FetchHTML(ctx context.Context, url string) (*goquery.Document, int, error) {
	results, err := c.FetchHTMLBatch(ctx, []string{url})
	if err != nil {
		return nil, 0, err
	}
	if len(results) == 0 {
		return nil, 0, fmt.Errorf("empty batch response")
	}
	r := results[0]
	return r.Doc, r.StatusCode, r.Err
}

// FetchHTMLBatch sends all URLs to the crawler worker in one POST request.
// The worker fetches them all in parallel via Promise.allSettled, so the call
// completes in roughly the time of the slowest single URL (~2–10s).
func (c *BatchClient) FetchHTMLBatch(ctx context.Context, urls []string) ([]BatchHTMLResult, error) {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	reqBody := crawlerRequest{
		URLs:    urls,
		Timeout: int(c.timeout.Milliseconds()),
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal batch request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/crawl", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create batch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("batch request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read batch response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("batch worker HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var items []crawlerResult
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("unmarshal batch response: %w", err)
	}

	results := make([]BatchHTMLResult, len(items))
	for i, item := range items {
		results[i].URL = item.URL
		results[i].StatusCode = item.Status

		if item.Error != nil && *item.Error != "" {
			results[i].Err = fmt.Errorf("worker: %s", *item.Error)
			continue
		}
		if item.HTML == nil || item.Status == 404 {
			results[i].StatusCode = item.Status
			continue // 404 or empty — caller marks done
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(*item.HTML))
		if err != nil {
			results[i].Err = fmt.Errorf("parse HTML: %w", err)
			continue
		}
		results[i].Doc = doc
	}
	return results, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
