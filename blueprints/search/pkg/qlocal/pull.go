package qlocal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultEmbedModelURI  = "hf:ggml-org/embeddinggemma-300M-GGUF/embeddinggemma-300M-Q8_0.gguf"
	DefaultRerankModelURI = "hf:ggml-org/Qwen3-Reranker-0.6B-Q8_0-GGUF/qwen3-reranker-0.6b-q8_0.gguf"
	DefaultExpandModelURI = "hf:tobil/qmd-query-expansion-1.7B-gguf/qmd-query-expansion-1.7B-q4_k_m.gguf"
)

type PullOptions struct {
	Models   []string
	Refresh  bool
	CacheDir string
	Client   *http.Client
}

type PullResult struct {
	Model      string `json:"model"`
	Path       string `json:"path"`
	SizeBytes  int64  `json:"sizeBytes"`
	Refreshed  bool   `json:"refreshed"`
	Downloaded bool   `json:"downloaded"`
}

func defaultModelCacheDir() string {
	if d := strings.TrimSpace(os.Getenv("QLOCAL_MODEL_CACHE_DIR")); d != "" {
		return d
	}
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		home, _ := os.UserHomeDir()
		cache = filepath.Join(home, ".cache")
	}
	return filepath.Join(cache, "mizu", "qlocal", "models")
}

func qlocalHFBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("QLOCAL_HF_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://huggingface.co"
}

type hfRef struct {
	Repo string
	File string
}

func parseHFURI(model string) (*hfRef, bool) {
	if !strings.HasPrefix(model, "hf:") {
		return nil, false
	}
	parts := strings.Split(strings.TrimPrefix(model, "hf:"), "/")
	if len(parts) < 3 {
		return nil, false
	}
	return &hfRef{
		Repo: strings.Join(parts[:2], "/"),
		File: strings.Join(parts[2:], "/"),
	}, true
}

func (r *hfRef) resolveURL() string {
	return qlocalHFBaseURL() + "/" + r.Repo + "/resolve/main/" + r.File
}

func (a *App) Pull(ctx context.Context, opts PullOptions) ([]PullResult, error) {
	models := opts.Models
	if len(models) == 0 {
		models = []string{DefaultEmbedModelURI, DefaultExpandModelURI, DefaultRerankModelURI}
	}
	cacheDir := opts.CacheDir
	if cacheDir == "" {
		cacheDir = defaultModelCacheDir()
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create model cache dir: %w", err)
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	var out []PullResult
	for _, model := range models {
		res, err := pullOneModel(ctx, client, cacheDir, model, opts.Refresh)
		if err != nil {
			return out, err
		}
		out = append(out, res)
	}
	return out, nil
}

func pullOneModel(ctx context.Context, client *http.Client, cacheDir, model string, refresh bool) (PullResult, error) {
	ref, ok := parseHFURI(model)
	if !ok {
		return PullResult{}, fmt.Errorf("unsupported model URI (expected hf:...): %s", model)
	}
	filename := filepath.Base(ref.File)
	targetPath := filepath.Join(cacheDir, filename)
	etagPath := targetPath + ".etag"
	url := ref.resolveURL()

	var remoteETag string
	headReq, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if resp, err := client.Do(headReq); err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			remoteETag = strings.Trim(resp.Header.Get("ETag"), `"`)
		}
	}
	localETagBytes, _ := os.ReadFile(etagPath)
	localETag := strings.Trim(strings.TrimSpace(string(localETagBytes)), `"`)
	needDownload := refresh
	if !needDownload {
		if st, err := os.Stat(targetPath); err != nil || st.Size() == 0 {
			needDownload = true
		} else if remoteETag != "" && localETag != "" && remoteETag != localETag {
			needDownload = true
		}
	}

	result := PullResult{Model: model, Path: targetPath}
	if needDownload {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err != nil {
			return result, fmt.Errorf("download %s: %w", model, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return result, fmt.Errorf("download %s: status %d: %s", model, resp.StatusCode, string(body))
		}
		tmp := targetPath + ".part"
		f, err := os.Create(tmp)
		if err != nil {
			return result, fmt.Errorf("create temp file: %w", err)
		}
		n, copyErr := io.Copy(f, resp.Body)
		closeErr := f.Close()
		if copyErr != nil {
			_ = os.Remove(tmp)
			return result, fmt.Errorf("download copy: %w", copyErr)
		}
		if closeErr != nil {
			_ = os.Remove(tmp)
			return result, fmt.Errorf("close temp file: %w", closeErr)
		}
		if err := os.Rename(tmp, targetPath); err != nil {
			return result, fmt.Errorf("rename model file: %w", err)
		}
		if remoteETag == "" {
			remoteETag = strings.Trim(resp.Header.Get("ETag"), `"`)
		}
		if remoteETag != "" {
			_ = os.WriteFile(etagPath, []byte(remoteETag+"\n"), 0o644)
		}
		result.Downloaded = true
		result.Refreshed = true
		result.SizeBytes = n
	} else {
		if st, err := os.Stat(targetPath); err == nil {
			result.SizeBytes = st.Size()
		}
		result.Refreshed = false
	}
	if result.SizeBytes == 0 {
		if st, err := os.Stat(targetPath); err == nil {
			result.SizeBytes = st.Size()
		}
	}
	return result, nil
}

func (a *App) PullModels(refresh bool) string {
	results, err := a.Pull(context.Background(), PullOptions{Refresh: refresh})
	if err != nil {
		return "qlocal pull error: " + err.Error()
	}
	if len(results) == 0 {
		return "qlocal pull: no models configured"
	}
	var lines []string
	for _, r := range results {
		note := "cached/checked"
		if r.Refreshed {
			note = "downloaded"
		}
		lines = append(lines, fmt.Sprintf("%s -> %s (%d bytes, %s)", r.Model, r.Path, r.SizeBytes, note))
	}
	return strings.Join(lines, "\n")
}
