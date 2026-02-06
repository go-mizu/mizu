package fw1

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
	hfAPIBase      = "https://huggingface.co/api/datasets"
	hfViewerAPI    = "https://datasets-server.huggingface.co"
	datasetRepo    = "HuggingFaceFW/fineweb"
	hfDownloadBase = "https://huggingface.co/datasets"
)

// Client interfaces with HuggingFace Hub API for FineWeb (v1).
type Client struct {
	httpClient         *http.Client
	downloadHTTPClient *http.Client
	userAgent          string
}

// NewClient creates a new HuggingFace client.
func NewClient() *Client {
	downloadTransport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		MaxIdleConnsPerHost: 5,
	}
	return &Client{
		httpClient:         &http.Client{Timeout: 60 * time.Second},
		downloadHTTPClient: &http.Client{Transport: downloadTransport},
		userAgent:          "mizu-search/1.0",
	}
}

// hfFileEntry represents a file entry from HuggingFace tree API.
type hfFileEntry struct {
	Type string `json:"type"`
	Path string `json:"path"`
	OID  string `json:"oid,omitempty"`
	Size int64  `json:"size,omitempty"`
	LFS  *struct {
		OID  string `json:"oid"`
		Size int64  `json:"size"`
	} `json:"lfs,omitempty"`
}

// ListDumps returns all available CC dump configs from the dataset.
func (c *Client) ListDumps(ctx context.Context) ([]DatasetConfig, error) {
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

	var configs []DatasetConfig
	for _, s := range result.Splits {
		configs = append(configs, DatasetConfig{Config: s.Config, Split: s.Split})
	}
	return configs, nil
}

// ListFiles returns parquet files for a specific CC dump.
// Uses limit=1000 to fetch all files in a single request (most dumps have <500 files).
// Falls back to skip-based pagination for extremely large dumps.
func (c *Client) ListFiles(ctx context.Context, dump string) ([]FileInfo, error) {
	var allFiles []FileInfo
	pathPrefix := fmt.Sprintf("data/%s", dump)
	skip := 0
	const pageLimit = 1000

	for {
		apiURL := fmt.Sprintf("%s/%s/tree/main/%s?limit=%d", hfAPIBase, datasetRepo, pathPrefix, pageLimit)
		if skip > 0 {
			apiURL += fmt.Sprintf("&skip=%d", skip)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)
		if token := os.Getenv("HF_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching file list: %w", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil, fmt.Errorf("dump %q not found in dataset", dump)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var entries []hfFileEntry
		if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		resp.Body.Close()

		if len(entries) == 0 {
			break
		}

		for _, entry := range entries {
			if entry.Type != "file" || !strings.HasSuffix(entry.Path, ".parquet") {
				continue
			}
			file := FileInfo{
				Name: filepath.Base(entry.Path),
				Path: entry.Path,
				URL:  fmt.Sprintf("%s/%s/resolve/main/%s", hfDownloadBase, datasetRepo, entry.Path),
			}
			if entry.LFS != nil {
				file.Size = entry.LFS.Size
				file.OID = entry.LFS.OID
				file.LFS = true
			} else {
				file.Size = entry.Size
				file.OID = entry.OID
			}
			allFiles = append(allFiles, file)
		}

		skip += len(entries)

		if len(entries) < pageLimit {
			break
		}
	}

	return allFiles, nil
}

// GetDatasetSize returns size info for the dataset (optionally filtered by dump config).
func (c *Client) GetDatasetSize(ctx context.Context, dump string) (*DatasetSizeInfo, error) {
	apiURL := fmt.Sprintf("%s/size?dataset=%s", hfViewerAPI, datasetRepo)
	if dump != "" {
		apiURL += "&config=" + dump
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var wrapper struct {
		Size json.RawMessage `json:"size"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var probe map[string]json.RawMessage
	if err := json.Unmarshal(wrapper.Size, &probe); err != nil {
		return nil, fmt.Errorf("decoding size object: %w", err)
	}

	info := &DatasetSizeInfo{}

	if _, hasConfig := probe["config"]; hasConfig {
		var result struct {
			Config struct {
				Config          string `json:"config"`
				NumRows         int64  `json:"num_rows"`
				NumBytesParquet int64  `json:"num_bytes_parquet_files"`
				NumBytesMemory  int64  `json:"num_bytes_memory"`
				NumColumns      int    `json:"num_columns"`
			} `json:"config"`
		}
		if err := json.Unmarshal(wrapper.Size, &result); err != nil {
			return nil, fmt.Errorf("decoding config response: %w", err)
		}
		info.TotalRows = result.Config.NumRows
		info.TotalBytes = result.Config.NumBytesParquet
		info.TotalBytesMemory = result.Config.NumBytesMemory
		info.Configs = []DumpSize{{
			Config:         result.Config.Config,
			NumRows:        result.Config.NumRows,
			NumBytes:       result.Config.NumBytesParquet,
			NumBytesMemory: result.Config.NumBytesMemory,
			NumColumns:     result.Config.NumColumns,
		}}
	} else {
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
		}
		if err := json.Unmarshal(wrapper.Size, &result); err != nil {
			return nil, fmt.Errorf("decoding dataset response: %w", err)
		}
		info.TotalRows = result.Dataset.NumRows
		info.TotalBytes = result.Dataset.NumBytesParquet
		info.TotalBytesMemory = result.Dataset.NumBytesMemory
		for _, cfg := range result.Configs {
			info.Configs = append(info.Configs, DumpSize{
				Config:         cfg.Config,
				NumRows:        cfg.NumRows,
				NumBytes:       cfg.NumBytesParquet,
				NumBytesMemory: cfg.NumBytesMemory,
				NumColumns:     cfg.NumColumns,
			})
		}
	}

	return info, nil
}

// DownloadFile downloads a single file to the destination path.
func (c *Client) DownloadFile(ctx context.Context, file FileInfo, destPath string) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if info, err := os.Stat(destPath); err == nil && info.Size() == file.Size {
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

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if file.Size > 0 && written != file.Size {
		return fmt.Errorf("size mismatch: expected %d, got %d", file.Size, written)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("closing file: %w", err)
	}

	return os.Rename(tmpPath, destPath)
}

// DownloadFileWithProgress downloads a file with byte-level progress reporting.
func (c *Client) DownloadFileWithProgress(ctx context.Context, file FileInfo, destPath string, progress ByteProgressFn) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

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

	buf := make([]byte, 64*1024)
	var written int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
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
