package cc

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	collInfoURL = "https://index.commoncrawl.org/collinfo.json"
	userAgent   = "MizuCC/1.0 (Common Crawl client; github.com/go-mizu/mizu)"
)

// Client provides HTTP access to Common Crawl data.
type Client struct {
	baseURL     string
	apiClient   *http.Client // Short timeout for API calls
	dlClients   []*http.Client // Sharded clients for data downloads
	shardCount  int
}

// NewClient creates a new Common Crawl client.
func NewClient(baseURL string, transportShards int) *Client {
	if transportShards <= 0 {
		transportShards = 32
	}
	if baseURL == "" {
		baseURL = "https://data.commoncrawl.org"
	}

	apiClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	dlClients := make([]*http.Client, transportShards)
	for i := range transportShards {
		_ = i
		dlClients[i] = &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          500,
				MaxIdleConnsPerHost:   500,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				DisableCompression:    true, // WARC data is already compressed
				ForceAttemptHTTP2:     true, // HTTP/2 multiplexing to single host
			},
		}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiClient:  apiClient,
		dlClients:  dlClients,
		shardCount: transportShards,
	}
}

// ClientForShard returns the HTTP client for a given shard index.
func (c *Client) ClientForShard(shard int) *http.Client {
	return c.dlClients[shard%c.shardCount]
}

// collInfoEntry matches the JSON shape from collinfo.json.
type collInfoEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Timegate string `json:"timegate"`
	CDXAPI   string `json:"cdx-api"`
	From     string `json:"from"`  // 2026-01-12T16:12:39
	To       string `json:"to"`    // 2026-01-25T14:05:40
}

// ListCrawls fetches the list of available Common Crawl datasets.
func (c *Client) ListCrawls(ctx context.Context) ([]Crawl, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", collInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching crawl list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("crawl list: HTTP %d", resp.StatusCode)
	}

	var entries []collInfoEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("parsing crawl list: %w", err)
	}

	crawls := make([]Crawl, 0, len(entries))
	for _, e := range entries {
		from, _ := time.Parse("2006-01-02T15:04:05", e.From)
		to, _ := time.Parse("2006-01-02T15:04:05", e.To)
		crawls = append(crawls, Crawl{
			ID:      e.ID,
			Name:    e.Name,
			From:    from,
			To:      to,
			CDXAPI:  e.CDXAPI,
			Gateway: e.Timegate,
		})
	}
	return crawls, nil
}

// DownloadManifest downloads and parses a manifest file (e.g. cc-index-table.paths.gz).
// Returns the list of relative paths.
func (c *Client) DownloadManifest(ctx context.Context, crawlID, kind string) ([]string, error) {
	// kind is one of: warc.paths.gz, cc-index-table.paths.gz, cc-index.paths.gz, etc.
	url := fmt.Sprintf("%s/crawl-data/%s/%s", c.baseURL, crawlID, kind)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading manifest %s: %w", kind, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("manifest %s: HTTP %d", kind, resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	if strings.HasSuffix(kind, ".gz") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("decompressing manifest: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}

// DownloadFile downloads a remote file to a local path with optional progress reporting.
// Supports resume by checking existing file size.
func (c *Client) DownloadFile(ctx context.Context, remotePath, localPath string, progress func(received, total int64)) error {
	url := fmt.Sprintf("%s/%s", c.baseURL, remotePath)

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Check existing file for resume
	var existingSize int64
	if fi, err := os.Stat(localPath); err == nil {
		existingSize = fi.Size()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	if existingSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
	}

	resp, err := c.apiClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", remotePath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 416 {
		// Range not satisfiable = file is complete
		if progress != nil {
			progress(existingSize, existingSize)
		}
		return nil
	}

	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return fmt.Errorf("download %s: HTTP %d", remotePath, resp.StatusCode)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == 206 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
		existingSize = 0
	}

	f, err := os.OpenFile(localPath, flags, 0644)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	totalSize := resp.ContentLength
	if totalSize > 0 && resp.StatusCode == 206 {
		totalSize += existingSize
	}

	buf := make([]byte, 256*1024) // 256KB buffer
	received := existingSize
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return fmt.Errorf("writing file: %w", err)
			}
			received += int64(n)
			if progress != nil {
				progress(received, totalSize)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("reading response: %w", readErr)
		}
	}

	return nil
}

// FetchWARCRecord fetches a single WARC record using an HTTP Range request.
// Returns the raw gzip-compressed bytes of the record.
func (c *Client) FetchWARCRecord(ctx context.Context, shard int, p WARCPointer) ([]byte, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, p.WARCFilename)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", p.RecordOffset, p.RecordOffset+p.RecordLength-1))

	client := c.dlClients[shard%c.shardCount]
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching WARC record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 206 && resp.StatusCode != 200 {
		return nil, fmt.Errorf("WARC record: HTTP %d (expected 206)", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading WARC record: %w", err)
	}
	return data, nil
}
