package cc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading CDX response: %w", err)
	}

	var entries []CDXJEntry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
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

	return entries, nil
}
