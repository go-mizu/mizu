package fw2

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
	// Dataset Viewer API base URL
	hfViewerAPI = "https://datasets-server.huggingface.co"
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

// ListConfigs returns all language configs available in the dataset.
func (c *Client) ListConfigs(ctx context.Context) ([]DatasetConfig, error) {
	apiURL := fmt.Sprintf("%s/splits?dataset=%s", hfViewerAPI, datasetRepo)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching configs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Splits []struct {
			Config string `json:"config"`
			Split  string `json:"split"`
		} `json:"splits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	configs := make([]DatasetConfig, len(result.Splits))
	for i, s := range result.Splits {
		configs[i] = DatasetConfig{Config: s.Config, Split: s.Split}
	}
	return configs, nil
}

// GetDatasetSize returns size info for the dataset (optionally filtered by lang).
func (c *Client) GetDatasetSize(ctx context.Context, lang string) (*DatasetSizeInfo, error) {
	apiURL := fmt.Sprintf("%s/size?dataset=%s", hfViewerAPI, datasetRepo)
	if lang != "" {
		apiURL += "&config=" + lang
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching size info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// The HF API returns different shapes depending on whether config= is specified:
	//   Without config: { size: { dataset: {...}, configs: [...], splits: [...] } }
	//   With config:    { size: { config: {...}, splits: [...] } }
	var raw json.RawMessage
	var wrapper struct {
		Size json.RawMessage `json:"size"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	raw = wrapper.Size

	// Try to detect which shape we got
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("decoding size object: %w", err)
	}

	info := &DatasetSizeInfo{}

	if _, hasConfig := probe["config"]; hasConfig {
		// Single-config response: { config: {...}, splits: [...] }
		var result struct {
			Config struct {
				Config          string `json:"config"`
				NumRows         int64  `json:"num_rows"`
				NumBytesParquet int64  `json:"num_bytes_parquet_files"`
				NumBytesMemory  int64  `json:"num_bytes_memory"`
				NumColumns      int    `json:"num_columns"`
			} `json:"config"`
			Splits []struct {
				Config          string `json:"config"`
				Split           string `json:"split"`
				NumRows         int64  `json:"num_rows"`
				NumBytesParquet int64  `json:"num_bytes_parquet_files"`
				NumBytesMemory  int64  `json:"num_bytes_memory"`
				NumColumns      int    `json:"num_columns"`
			} `json:"splits"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("decoding config response: %w", err)
		}

		info.TotalRows = result.Config.NumRows
		info.TotalBytes = result.Config.NumBytesParquet
		info.TotalBytesMemory = result.Config.NumBytesMemory

		var splits []SplitSize
		for _, s := range result.Splits {
			splits = append(splits, SplitSize{
				Config:         s.Config,
				Split:          s.Split,
				NumRows:        s.NumRows,
				NumBytes:       s.NumBytesParquet,
				NumBytesMemory: s.NumBytesMemory,
				NumColumns:     s.NumColumns,
			})
		}

		info.Configs = []ConfigSize{{
			Config:         result.Config.Config,
			NumRows:        result.Config.NumRows,
			NumBytes:       result.Config.NumBytesParquet,
			NumBytesMemory: result.Config.NumBytesMemory,
			NumColumns:     result.Config.NumColumns,
			Splits:         splits,
		}}
	} else {
		// Full-dataset response: { dataset: {...}, configs: [...], splits: [...] }
		var result struct {
			Dataset struct {
				NumRows         int64 `json:"num_rows"`
				NumBytesParquet int64 `json:"num_bytes_parquet_files"`
				NumBytesMemory  int64 `json:"num_bytes_memory"`
			} `json:"dataset"`
			Configs []struct {
				Config          string `json:"config"`
				NumRows         int64  `json:"num_rows"`
				NumBytesParquet int64  `json:"num_bytes_parquet_files"`
				NumBytesMemory  int64  `json:"num_bytes_memory"`
				NumColumns      int    `json:"num_columns"`
			} `json:"configs"`
			Splits []struct {
				Config          string `json:"config"`
				Split           string `json:"split"`
				NumRows         int64  `json:"num_rows"`
				NumBytesParquet int64  `json:"num_bytes_parquet_files"`
				NumBytesMemory  int64  `json:"num_bytes_memory"`
				NumColumns      int    `json:"num_columns"`
			} `json:"splits"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("decoding dataset response: %w", err)
		}

		info.TotalRows = result.Dataset.NumRows
		info.TotalBytes = result.Dataset.NumBytesParquet
		info.TotalBytesMemory = result.Dataset.NumBytesMemory

		splitMap := make(map[string][]SplitSize)
		for _, s := range result.Splits {
			splitMap[s.Config] = append(splitMap[s.Config], SplitSize{
				Config:         s.Config,
				Split:          s.Split,
				NumRows:        s.NumRows,
				NumBytes:       s.NumBytesParquet,
				NumBytesMemory: s.NumBytesMemory,
				NumColumns:     s.NumColumns,
			})
		}

		for _, cfg := range result.Configs {
			info.Configs = append(info.Configs, ConfigSize{
				Config:         cfg.Config,
				NumRows:        cfg.NumRows,
				NumBytes:       cfg.NumBytesParquet,
				NumBytesMemory: cfg.NumBytesMemory,
				NumColumns:     cfg.NumColumns,
				Splits:         splitMap[cfg.Config],
			})
		}
	}

	return info, nil
}

// ListSplitFiles returns parquet files for a specific language and split.
func (c *Client) ListSplitFiles(ctx context.Context, lang, split string) ([]FileInfo, error) {
	pathPrefix := fmt.Sprintf("data/%s/%s", lang, split)
	return c.listFilesAtPath(ctx, pathPrefix)
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

// ByteProgressFn is called periodically during download with bytes downloaded so far.
type ByteProgressFn func(bytesDownloaded, totalBytes int64)

// DownloadFileWithProgress downloads a file with byte-level progress reporting.
func (c *Client) DownloadFileWithProgress(ctx context.Context, file FileInfo, destPath string, progress ByteProgressFn) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Check if already downloaded
	if info, err := os.Stat(destPath); err == nil && info.Size() == file.Size {
		if progress != nil {
			progress(file.Size, file.Size)
		}
		return nil
	}

	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer func() {
		out.Close()
		os.Remove(tmpPath)
	}()

	req, err := http.NewRequestWithContext(ctx, "GET", file.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.downloadHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download error %d", resp.StatusCode)
	}

	totalBytes := file.Size
	if totalBytes == 0 && resp.ContentLength > 0 {
		totalBytes = resp.ContentLength
	}

	buf := make([]byte, 64*1024) // 64KB buffer
	var written int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("writing file: %w", writeErr)
			}
			written += int64(n)
			if progress != nil {
				progress(written, totalBytes)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("reading response: %w", readErr)
		}
	}

	if file.Size > 0 && written != file.Size {
		return fmt.Errorf("size mismatch: expected %d, got %d", file.Size, written)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("closing file: %w", err)
	}

	return os.Rename(tmpPath, destPath)
}
