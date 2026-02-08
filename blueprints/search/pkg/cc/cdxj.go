package cc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// cdxClient is a shared HTTP client for CDX API requests.
// Uses transport-level timeouts so large response bodies don't trigger deadline exceeded.
var cdxClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:  15 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConnsPerHost:   10,
	},
}

// LookupURL queries the CDX API for a specific URL in a crawl.
func LookupURL(ctx context.Context, crawlID, targetURL string) ([]CDXJEntry, error) {
	apiURL := fmt.Sprintf("https://index.commoncrawl.org/%s-index?url=%s&output=json",
		crawlID, url.QueryEscape(targetURL))
	return fetchCDXJ(ctx, apiURL)
}

// LookupDomain queries the CDX API for all URLs under a domain.
func LookupDomain(ctx context.Context, crawlID, domain string, limit int) ([]CDXJEntry, error) {
	apiURL := fmt.Sprintf("https://index.commoncrawl.org/%s-index?url=%s/*&output=json&limit=%d",
		crawlID, url.QueryEscape(domain), limit)
	return fetchCDXJ(ctx, apiURL)
}

func fetchCDXJ(ctx context.Context, apiURL string) ([]CDXJEntry, error) {
	const maxRetries = 5
	var lastErr error

	for attempt := range maxRetries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		entries, err := fetchCDXJOnce(ctx, apiURL)
		if err == nil {
			return entries, nil
		}
		lastErr = err
		if attempt < maxRetries-1 {
			backoff := time.Duration(attempt+1) * 10 * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	return nil, lastErr
}

func fetchCDXJOnce(ctx context.Context, apiURL string) ([]CDXJEntry, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := cdxClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CDX API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil // No results
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("CDX API: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Stream-parse JSON lines instead of io.ReadAll to handle large responses
	var entries []CDXJEntry
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 256*1024), 1024*1024) // 1MB max line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]string
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		entries = append(entries, CDXJEntry{
			URL:       raw["url"],
			Mime:      raw["mime"],
			Status:    raw["status"],
			Digest:    raw["digest"],
			Length:    raw["length"],
			Offset:    raw["offset"],
			Filename:  raw["filename"],
			Languages: raw["languages"],
			Encoding:  raw["charset"],
			Timestamp: raw["timestamp"],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading CDX response: %w", err)
	}

	return entries, nil
}

// CDXJPageCount returns the number of CDX API result pages for a domain.
func CDXJPageCount(ctx context.Context, crawlID, domain string) (int, error) {
	apiURL := fmt.Sprintf("https://index.commoncrawl.org/%s-index?url=%s&matchType=domain&output=json&showNumPages=true",
		crawlID, url.QueryEscape(domain))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := cdxClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("CDX page count: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return 0, nil
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return 0, fmt.Errorf("CDX page count: HTTP %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading page count: %w", err)
	}

	text := strings.TrimSpace(string(data))

	// Try plain integer first
	if n, err := strconv.Atoi(text); err == nil {
		return n, nil
	}

	// Try JSON: {"pages": N, ...}
	var result struct {
		Pages int `json:"pages"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("parsing page count %q: %w", text, err)
	}
	return result.Pages, nil
}

// LookupDomainPage fetches a single page of CDX API results for a domain.
func LookupDomainPage(ctx context.Context, crawlID, domain string, page int) ([]CDXJEntry, error) {
	apiURL := fmt.Sprintf("https://index.commoncrawl.org/%s-index?url=%s&matchType=domain&output=json&page=%d",
		crawlID, url.QueryEscape(domain), page)
	return fetchCDXJ(ctx, apiURL)
}

// LookupDomainAll fetches all CDX API results for a domain using pagination.
// Progress callback receives (pagesCompleted, totalPages).
func LookupDomainAll(ctx context.Context, crawlID, domain string, concurrency int, progress func(done, total int)) ([]CDXJEntry, error) {
	if concurrency <= 0 {
		concurrency = 10
	}

	numPages, err := CDXJPageCount(ctx, crawlID, domain)
	if err != nil {
		return nil, fmt.Errorf("getting page count: %w", err)
	}
	if numPages == 0 {
		return nil, nil
	}

	type pageResult struct {
		page    int
		entries []CDXJEntry
		err     error
	}

	results := make([]pageResult, numPages)
	var mu sync.Mutex
	var done int

	pageCh := make(chan int, numPages)
	for i := range numPages {
		pageCh <- i
	}
	close(pageCh)

	var wg sync.WaitGroup
	for range min(concurrency, numPages) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range pageCh {
				if ctx.Err() != nil {
					return
				}
				entries, fetchErr := LookupDomainPage(ctx, crawlID, domain, page)
				results[page] = pageResult{page: page, entries: entries, err: fetchErr}

				mu.Lock()
				done++
				if progress != nil {
					progress(done, numPages)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Collect all entries in order
	var all []CDXJEntry
	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("fetching CDX page %d: %w", r.page, r.err)
		}
		all = append(all, r.entries...)
	}
	return all, nil
}

// CDXJToWARCPointer converts a CDXJEntry to a WARCPointer.
func CDXJToWARCPointer(e CDXJEntry, domain string) (WARCPointer, error) {
	offset, err := strconv.ParseInt(e.Offset, 10, 64)
	if err != nil {
		return WARCPointer{}, fmt.Errorf("parsing offset %q: %w", e.Offset, err)
	}
	length, err := strconv.ParseInt(e.Length, 10, 64)
	if err != nil {
		return WARCPointer{}, fmt.Errorf("parsing length %q: %w", e.Length, err)
	}
	status, _ := strconv.Atoi(e.Status)

	return WARCPointer{
		URL:          e.URL,
		WARCFilename: e.Filename,
		RecordOffset: offset,
		RecordLength: length,
		ContentType:  e.Mime,
		Language:     e.Languages,
		FetchStatus:  status,
		Domain:       domain,
	}, nil
}
