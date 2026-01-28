package fineweb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// HuggingFace API base URL
	hfAPIBase = "https://huggingface.co/api/datasets"
	// Dataset repository
	datasetRepo = "HuggingFaceFW/fineweb-2"
	// Download base URL
	hfDownloadBase = "https://huggingface.co/datasets"
)

// Client interfaces with HuggingFace Hub API.
type Client struct {
	httpClient         *http.Client
	downloadHTTPClient *http.Client
	userAgent          string
}

// NewClient creates a new HuggingFace client.
func NewClient() *Client {
	// Transport for downloads - no timeout, relies on context
	downloadTransport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		MaxIdleConnsPerHost: 5,
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // For API calls
		},
		downloadHTTPClient: &http.Client{
			Transport: downloadTransport,
			// No Timeout set - rely on context deadline
		},
		userAgent: "mizu-search/1.0",
	}
}

// hfFileEntry represents a file entry from HuggingFace API.
type hfFileEntry struct {
	Type string `json:"type"` // "file" or "directory"
	Path string `json:"path"`
	OID  string `json:"oid,omitempty"`
	Size int64  `json:"size,omitempty"`
	LFS  *struct {
		OID         string `json:"oid"`
		Size        int64  `json:"size"`
		PointerSize int64  `json:"pointerSize"`
	} `json:"lfs,omitempty"`
}

// ListFiles returns available parquet files for a language.
func (c *Client) ListFiles(ctx context.Context, lang string) ([]FileInfo, error) {
	// Try multiple path patterns as the dataset structure may vary
	pathPatterns := []string{
		fmt.Sprintf("data/%s/train", lang),      // data/{lang}/train
		fmt.Sprintf("data/%s", lang),            // data/{lang}
		fmt.Sprintf("%s/train", lang),           // {lang}/train
		lang,                                     // {lang}
	}

	var lastErr error
	for _, pathPrefix := range pathPatterns {
		files, err := c.listFilesAtPath(ctx, pathPrefix)
		if err == nil && len(files) > 0 {
			return files, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("language %q not found in dataset", lang)
}

func (c *Client) listFilesAtPath(ctx context.Context, pathPrefix string) ([]FileInfo, error) {
	apiURL := fmt.Sprintf("%s/%s/tree/main/%s", hfAPIBase, datasetRepo, pathPrefix)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching file list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("path %q not found", pathPrefix)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var entries []hfFileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		if entry.Type != "file" {
			continue
		}
		if !strings.HasSuffix(entry.Path, ".parquet") {
			continue
		}

		file := FileInfo{
			Name: filepath.Base(entry.Path),
			Path: entry.Path,
			URL:  fmt.Sprintf("%s/%s/resolve/main/%s", hfDownloadBase, datasetRepo, entry.Path),
		}

		// Use LFS size if available
		if entry.LFS != nil {
			file.Size = entry.LFS.Size
			file.OID = entry.LFS.OID
			file.LFS = true
		} else {
			file.Size = entry.Size
			file.OID = entry.OID
		}

		files = append(files, file)
	}

	return files, nil
}

// DownloadFile downloads a single file to the destination path.
func (c *Client) DownloadFile(ctx context.Context, file FileInfo, destPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Check if file already exists with correct size
	if info, err := os.Stat(destPath); err == nil {
		if info.Size() == file.Size {
			return nil // Already downloaded
		}
	}

	// Create temp file for download
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer func() {
		out.Close()
		os.Remove(tmpPath) // Clean up on failure
	}()

	// Download file (use download client without fixed timeout)
	req, err := http.NewRequestWithContext(ctx, "GET", file.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.downloadHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download error %d", resp.StatusCode)
	}

	// Copy to file
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	// Verify size if known
	if file.Size > 0 && written != file.Size {
		return fmt.Errorf("size mismatch: expected %d, got %d", file.Size, written)
	}

	// Close before rename
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing file: %w", err)
	}

	// Rename to final destination
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("renaming file: %w", err)
	}

	return nil
}
