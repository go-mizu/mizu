package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// RemoteParquetInfo describes the primary parquet snapshot source.
type RemoteParquetInfo struct {
	URL          string    `json:"url"`
	FinalURL     string    `json:"final_url,omitempty"`
	Size         int64     `json:"size"`
	ETag         string    `json:"etag,omitempty"`
	AcceptRanges bool      `json:"accept_ranges"`
	LastModified string    `json:"last_modified,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

func (c Config) HeadParquet(ctx context.Context) (*RemoteParquetInfo, error) {
	cfg := c.WithDefaults()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, cfg.ParquetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create HEAD request: %w", err)
	}
	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("HEAD parquet source: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HEAD parquet source returned %d", resp.StatusCode)
	}
	info := &RemoteParquetInfo{
		URL:          cfg.ParquetURL,
		FinalURL:     resp.Request.URL.String(),
		Size:         resp.ContentLength,
		ETag:         strings.TrimSpace(resp.Header.Get("ETag")),
		AcceptRanges: strings.Contains(strings.ToLower(resp.Header.Get("Accept-Ranges")), "bytes"),
		LastModified: strings.TrimSpace(resp.Header.Get("Last-Modified")),
		CheckedAt:    time.Now().UTC(),
	}
	_ = writeJSONFile(cfg.ParquetHeadCachePath(), info)
	return info, nil
}

func (c Config) ReadCachedParquetHead() (*RemoteParquetInfo, error) {
	var info RemoteParquetInfo
	f, err := os.Open(c.ParquetHeadCachePath())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (c Config) GetMaxItem(ctx context.Context) (int64, error) {
	cfg := c.WithDefaults()
	url := strings.TrimRight(cfg.APIBaseURL, "/") + "/maxitem.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("create maxitem request: %w", err)
	}
	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return 0, fmt.Errorf("GET maxitem: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GET maxitem returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return 0, fmt.Errorf("read maxitem: %w", err)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(body)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse maxitem %q: %w", strings.TrimSpace(string(body)), err)
	}
	return n, nil
}
