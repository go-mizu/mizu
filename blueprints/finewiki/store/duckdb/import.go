package duckdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ImportOptions struct {
	Dir string

	Timeout time.Duration

	Token string

	Client *http.Client
}

func ImportParquet(ctx context.Context, src string, opt ImportOptions) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", errors.New("duckdb: empty src")
	}
	dir := strings.TrimSpace(opt.Dir)
	if dir == "" {
		return "", errors.New("duckdb: empty dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	if opt.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}

	if isHTTP(src) {
		dst := filepath.Join(dir, fileNameFromURL(src))
		if dst == dir || strings.HasSuffix(dst, string(filepath.Separator)) {
			dst = filepath.Join(dir, "data.parquet")
		}
		if err := download(ctx, httpClient(opt), src, dst, opt.Token); err != nil {
			return "", err
		}
		return dst, nil
	}

	fi, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return "", errors.New("duckdb: src is a directory")
	}

	dst := filepath.Join(dir, filepath.Base(src))
	if err := copyFile(src, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func ListParquet(ctx context.Context, dataset string, token string) ([]string, error) {
	dataset = strings.TrimSpace(dataset)
	if dataset == "" {
		return nil, errors.New("duckdb: empty dataset")
	}

	u := fmt.Sprintf("https://datasets-server.huggingface.co/parquet?dataset=%s", dataset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	c := &http.Client{Timeout: 60 * time.Second}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("duckdb: list parquet: http %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}

	var payload struct {
		ParquetFiles []struct {
			URL string `json:"url,omitempty"`
		} `json:"parquet_files,omitempty"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(payload.ParquetFiles))
	for _, f := range payload.ParquetFiles {
		if f.URL != "" {
			out = append(out, f.URL)
		}
	}
	return out, nil
}

func httpClient(opt ImportOptions) *http.Client {
	if opt.Client != nil {
		return opt.Client
	}
	return &http.Client{Timeout: 0}
}

func download(ctx context.Context, c *http.Client, url, dst, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("duckdb: download: http %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}

	tmp := dst + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(f, res.Body)
	closeErr := f.Close()

	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}

	return os.Rename(tmp, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".partial"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()

	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}

	return os.Rename(tmp, dst)
}

func isHTTP(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func fileNameFromURL(u string) string {
	if i := strings.IndexByte(u, '?'); i >= 0 {
		u = u[:i]
	}
	u = strings.TrimRight(u, "/")
	if u == "" {
		return ""
	}
	if j := strings.LastIndexByte(u, '/'); j >= 0 && j < len(u)-1 {
		return u[j+1:]
	}
	return ""
}
