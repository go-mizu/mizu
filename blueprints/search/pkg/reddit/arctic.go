package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ArcticClient is an HTTP client for the Arctic Shift API.
type ArcticClient struct {
	http    *http.Client
	baseURL string
}

// NewArcticClient creates a new Arctic Shift API client.
func NewArcticClient() *ArcticClient {
	return &ArcticClient{
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: ArcticBaseURL,
	}
}

// GetMinDate returns the earliest post date for a target.
func (c *ArcticClient) GetMinDate(ctx context.Context, target ArcticTarget) (time.Time, error) {
	param := "subreddit"
	if target.Kind == "user" {
		param = "author"
	}
	url := fmt.Sprintf("%s/api/utils/min?%s=%s", c.baseURL, param, target.Name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return time.Time{}, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("fetch min date: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return time.Time{}, fmt.Errorf("min date API returned %d", resp.StatusCode)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return time.Time{}, fmt.Errorf("decode min date: %w", err)
	}

	if result.Data == "" {
		return time.Time{}, fmt.Errorf("no data found for %s %s", target.Kind, target.Name)
	}

	t, err := time.Parse(time.RFC3339, result.Data)
	if err != nil {
		// Try other formats
		t, err = time.Parse("2006-01-02T15:04:05Z", result.Data)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05.000Z", result.Data)
			if err != nil {
				return time.Time{}, fmt.Errorf("parse min date %q: %w", result.Data, err)
			}
		}
	}
	return t, nil
}

// ArcticProgress reports download progress.
type ArcticProgress struct {
	Kind      FileKind
	Items     int64         // total items downloaded so far
	Bytes     int64         // total bytes written
	Oldest    time.Time     // oldest item timestamp
	Newest    time.Time     // newest item timestamp
	BatchSize int           // last batch size
	Done      bool
	Elapsed   time.Duration
}

// ArcticProgressCallback is called with download progress updates.
type ArcticProgressCallback func(ArcticProgress)

// Download fetches all items for a target and writes JSONL to disk.
// Supports resume via afterEpoch (0 = start from beginning).
func (c *ArcticClient) Download(ctx context.Context, target ArcticTarget, kind FileKind,
	afterEpoch int64, beforeEpoch int64, cb ArcticProgressCallback) error {

	// Create output directory
	dir := target.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	jsonlPath := target.JSONLPath(kind)

	// Open file in append mode for resume support
	var flags int
	if afterEpoch > 0 {
		flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	} else {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(jsonlPath, flags, 0o644)
	if err != nil {
		return fmt.Errorf("open output: %w", err)
	}
	defer f.Close()

	// Build API endpoint
	endpoint := "posts"
	if kind == Comments {
		endpoint = "comments"
	}

	param := "subreddit"
	if target.Kind == "user" {
		param = "author"
	}

	// Select fields to reduce response size
	fields := commentFields
	if kind == Submissions {
		fields = submissionFields
	}

	start := time.Now()
	var totalItems int64
	var totalBytes int64
	var oldest, newest time.Time
	currentAfter := afterEpoch
	// Arctic Shift API requires epoch >= 1000000000 (2001-09-09).
	// Use a date before Reddit existed (2005-01-01) as the minimum.
	if currentAfter < 1104537600 {
		currentAfter = 1104537600 // 2005-01-01 00:00:00 UTC
	}
	retries := 0
	maxRetries := 10

	for {
		select {
		case <-ctx.Done():
			// Save progress before exiting
			saveProgress(target, kind, currentAfter)
			return ctx.Err()
		default:
		}

		// Build URL
		url := fmt.Sprintf("%s/api/%s/search?%s=%s&limit=auto&sort=asc&fields=%s&after=%d",
			c.baseURL, endpoint, param, target.Name, fields, currentAfter)
		if beforeEpoch > 0 {
			url += fmt.Sprintf("&before=%d", beforeEpoch)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			retries++
			if retries > maxRetries {
				saveProgress(target, kind, currentAfter)
				return fmt.Errorf("max retries exceeded: %w", err)
			}
			backoff := time.Duration(1<<uint(retries-1)) * time.Second
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
			time.Sleep(backoff)
			continue
		}

		if resp.StatusCode == 429 {
			// Rate limited
			resp.Body.Close()
			resetStr := resp.Header.Get("X-RateLimit-Reset")
			wait := 30 * time.Second
			if resetStr != "" {
				if resetEpoch, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
					wait = time.Until(time.Unix(resetEpoch, 0))
					if wait < time.Second {
						wait = time.Second
					}
				}
			}
			time.Sleep(wait)
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			retries++
			if retries > maxRetries {
				saveProgress(target, kind, currentAfter)
				return fmt.Errorf("API returned %d after %d retries", resp.StatusCode, retries)
			}
			backoff := time.Duration(1<<uint(retries-1)) * time.Second
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
			time.Sleep(backoff)
			continue
		}

		// Parse response
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			retries++
			if retries > maxRetries {
				saveProgress(target, kind, currentAfter)
				return fmt.Errorf("read body: %w", err)
			}
			time.Sleep(time.Duration(1<<uint(retries-1)) * time.Second)
			continue
		}

		retries = 0 // Reset on success

		var result struct {
			Data []json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		if len(result.Data) == 0 {
			// Done
			break
		}

		// Write each item as a JSONL line
		for _, item := range result.Data {
			line := append(item, '\n')
			n, err := f.Write(line)
			if err != nil {
				saveProgress(target, kind, currentAfter)
				return fmt.Errorf("write jsonl: %w", err)
			}
			totalBytes += int64(n)
		}
		totalItems += int64(len(result.Data))

		// Extract created_utc from last item for pagination
		var lastItem struct {
			CreatedUTC json.Number `json:"created_utc"`
		}
		if err := json.Unmarshal(result.Data[len(result.Data)-1], &lastItem); err != nil {
			return fmt.Errorf("decode last item: %w", err)
		}

		lastEpoch, err := lastItem.CreatedUTC.Int64()
		if err != nil {
			// Try float
			f64, err2 := lastItem.CreatedUTC.Float64()
			if err2 != nil {
				return fmt.Errorf("parse created_utc %q: %w", lastItem.CreatedUTC, err)
			}
			lastEpoch = int64(f64)
		}

		// Avoid infinite loop on duplicate timestamps
		if lastEpoch == currentAfter {
			lastEpoch++
		}
		currentAfter = lastEpoch

		// Track timestamps for display
		newest = time.Unix(lastEpoch, 0)
		if oldest.IsZero() {
			// Extract from first item of first batch
			var firstItem struct {
				CreatedUTC json.Number `json:"created_utc"`
			}
			if json.Unmarshal(result.Data[0], &firstItem) == nil {
				if e, err := firstItem.CreatedUTC.Int64(); err == nil {
					oldest = time.Unix(e, 0)
				}
			}
		}

		// Progress callback
		if cb != nil {
			cb(ArcticProgress{
				Kind:      kind,
				Items:     totalItems,
				Bytes:     totalBytes,
				Oldest:    oldest,
				Newest:    newest,
				BatchSize: len(result.Data),
				Elapsed:   time.Since(start),
			})
		}

		// Save progress periodically
		saveProgress(target, kind, currentAfter)
	}

	// Final callback
	if cb != nil {
		cb(ArcticProgress{
			Kind:    kind,
			Items:   totalItems,
			Bytes:   totalBytes,
			Oldest:  oldest,
			Newest:  newest,
			Done:    true,
			Elapsed: time.Since(start),
		})
	}

	// Clean up progress file on completion
	os.Remove(target.ProgressPath())

	return nil
}

// Comment fields to request (reduces response size).
// Only fields validated against the Arctic Shift API (invalid fields return 400).
const commentFields = "id,author,body,created_utc,score,subreddit,link_id,parent_id,distinguished,author_flair_text"

// Submission fields to request.
const submissionFields = "id,title,selftext,author,created_utc,score,num_comments,subreddit,url,over_18,link_flair_text,author_flair_text"

// Progress file for resume support.
type progressData struct {
	CommentsAfter    int64 `json:"comments_after,omitempty"`
	SubmissionsAfter int64 `json:"submissions_after,omitempty"`
}

func saveProgress(target ArcticTarget, kind FileKind, afterEpoch int64) {
	path := target.ProgressPath()

	// Load existing
	var prog progressData
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &prog)
	}

	if kind == Comments {
		prog.CommentsAfter = afterEpoch
	} else {
		prog.SubmissionsAfter = afterEpoch
	}

	data, _ := json.Marshal(prog)
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, data, 0o644)
}

// LoadProgress loads the progress file for resume support.
func LoadProgress(target ArcticTarget) (commentsAfter, submissionsAfter int64) {
	data, err := os.ReadFile(target.ProgressPath())
	if err != nil {
		return 0, 0
	}
	var prog progressData
	if json.Unmarshal(data, &prog) != nil {
		return 0, 0
	}
	return prog.CommentsAfter, prog.SubmissionsAfter
}
