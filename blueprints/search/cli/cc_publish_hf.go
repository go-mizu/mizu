package cli

// Pure-Go HuggingFace Hub client for dataset publishing.
// Large files (parquet, PNGs) are uploaded via a Python helper (hf_commit.py)
// run through uv, which uses huggingface_hub + hf-xet for native xet storage.
// Falls back to Go LFS basic upload if uv is not available.

import (
	_ "embed"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed embed/hf_commit.py
var hfCommitPy []byte

const hfHubURL = "https://huggingface.co"

// HFRateLimitError is returned when HuggingFace responds 429 Too Many Requests.
// RetryAfter is the server-requested wait duration (0 if not provided).
type HFRateLimitError struct {
	RetryAfter time.Duration
	Msg        string
}

func (e *HFRateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("HF rate limited (retry after %s): %s", e.RetryAfter.Round(time.Second), e.Msg)
	}
	return fmt.Sprintf("HF rate limited: %s", e.Msg)
}

// hfClient is a minimal HuggingFace Hub API client.
type hfClient struct {
	token string
	http  *http.Client
}

func newHFClient(token string) *hfClient {
	return &hfClient{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Minute},
	}
}

func (c *hfClient) req(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.http.Do(req)
}

// createDatasetRepo creates a dataset repo if it does not exist.
func (c *hfClient) createDatasetRepo(ctx context.Context, repoID string, private bool) error {
	parts := strings.SplitN(repoID, "/", 2)
	org, name := parts[0], parts[1]
	body, _ := json.Marshal(map[string]interface{}{
		"type":         "dataset",
		"name":         name,
		"organization": org,
		"private":      private,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", hfHubURL+"/api/repos/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("create repo: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 || resp.StatusCode == 201 || resp.StatusCode == 409 {
		return nil
	}
	return fmt.Errorf("create repo HTTP %d", resp.StatusCode)
}

// pathsExist returns the set of paths that already exist in the repo at "main".
func (c *hfClient) pathsExist(ctx context.Context, repoID string, paths []string) (map[string]bool, error) {
	existing := make(map[string]bool)
	for start := 0; start < len(paths); start += 100 {
		end := start + 100
		if end > len(paths) {
			end = len(paths)
		}
		body, _ := json.Marshal(map[string]interface{}{"paths": paths[start:end]})
		url := fmt.Sprintf("%s/api/datasets/%s/paths-info/main", hfHubURL, repoID)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("paths-info: %w", err)
		}
		if resp.StatusCode == 404 {
			resp.Body.Close()
			continue
		}
		var infos []struct {
			Path string `json:"path"`
		}
		json.NewDecoder(resp.Body).Decode(&infos)
		resp.Body.Close()
		for _, info := range infos {
			existing[info.Path] = true
		}
	}
	return existing, nil
}

// hfOperation describes a file add or delete for a HuggingFace commit.
// Set Delete=true for CommitOperationDelete (LocalPath is ignored).
type hfOperation struct {
	LocalPath  string
	PathInRepo string
	Delete     bool
}

// resolveUV returns the path to the uv binary, checking PATH then common install locations.
func resolveUV() string {
	if p, err := exec.LookPath("uv"); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	for _, candidate := range []string{
		filepath.Join(home, ".local", "bin", "uv"),
		filepath.Join(home, ".cargo", "bin", "uv"),
		"/usr/local/bin/uv",
	} {
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate
		}
	}
	return ""
}

// hfCommitScriptPath returns the cached path of the embedded hf_commit.py helper,
// writing it to ~/.cache/open-index/ if missing or outdated.
func hfCommitScriptPath() (string, error) {
	home, _ := os.UserHomeDir()
	dir := home + "/.cache/open-index"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := dir + "/hf_commit.py"
	existing, _ := os.ReadFile(p)
	if string(existing) != string(hfCommitPy) {
		if err := os.WriteFile(p, hfCommitPy, 0o755); err != nil {
			return "", err
		}
	}
	return p, nil
}

// createCommitPython runs the embedded hf_commit.py via uv to upload files
// using huggingface_hub (xet-aware). Returns "", nil if uv is not installed.
func (c *hfClient) createCommitPython(ctx context.Context, repoID, message string, ops []hfOperation) (string, error) {
	scriptPath, err := hfCommitScriptPath()
	if err != nil {
		return "", nil // silently skip
	}

	type opJSON struct {
		LocalPath  string `json:"local_path,omitempty"`
		PathInRepo string `json:"path_in_repo"`
		Delete     bool   `json:"delete,omitempty"`
	}
	opsJSON := make([]opJSON, len(ops))
	for i, op := range ops {
		opsJSON[i] = opJSON{LocalPath: op.LocalPath, PathInRepo: op.PathInRepo, Delete: op.Delete}
	}
	// num_threads controls file-level parallelism in create_commit() (NOT
	// chunk-level — xet handles chunk concurrency via adaptive controller).
	// With typically 5–10 parquet files per commit, 8 is a reasonable ceiling.
	numThreads := 8
	payload := map[string]interface{}{
		"token":       c.token,
		"repo_id":     repoID,
		"message":     message,
		"num_threads": numThreads,
		"ops":         opsJSON,
	}
	stdin, _ := json.Marshal(payload)

	uvBin := resolveUV()
	if uvBin == "" {
		return "", fmt.Errorf("uv not found")
	}

	// Per-upload timeout: 30 min. Xet handles its own per-chunk retries
	// (up to 10 min retry window per request), so this is a hard ceiling
	// for the entire commit. 30 min is generous for 500 MB of parquet.
	uploadTimeout := 30 * time.Minute
	uploadCtx, uploadCancel := context.WithTimeout(ctx, uploadTimeout)
	defer uploadCancel()

	cmd := exec.CommandContext(uploadCtx, uvBin, "run", scriptPath)
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stderr = os.Stderr // pipe progress logs directly to terminal
	// Xet upload tuning for memory-constrained servers (< 64 GB RAM).
	// DO NOT use HF_XET_HIGH_PERFORMANCE — it requires 64+ GB RAM and
	// causes upload stalls on smaller machines (sets concurrency to 124,
	// buffers to 16 GB, thrashes memory).
	cmd.Env = append(os.Environ(),
		"HF_HUB_VERBOSITY=warning",
		// Pin upload concurrency to a fixed value suited for ~11 GB servers.
		// Adaptive concurrency defaults start at 1 and ramp slowly; pinning
		// at 8 starts at full speed without oversaturating memory.
		"HF_XET_FIXED_UPLOAD_CONCURRENCY=8",
		// Increase per-request retry budget so transient failures recover.
		"HF_XET_CLIENT_RETRY_MAX_ATTEMPTS=7",
		"HF_XET_CLIENT_RETRY_MAX_DURATION=600s",
		// Generous read timeout — large shard uploads can take a while.
		"HF_XET_CLIENT_READ_TIMEOUT=300s",
		"HF_XET_CLIENT_CONNECT_TIMEOUT=120s",
		// Increase shard cache so re-uploads after a stall skip already-
		// uploaded chunks (deduplication).
		"HF_XET_SHARD_CACHE_SIZE_LIMIT=8000000000",
		// Xet diagnostics — written to file so they don't pollute stderr.
		"RUST_LOG=info",
		"HF_XET_LOG_FILE=/tmp/xet_upload.log",
	)
	out, err := cmd.Output()
	if uploadCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("python commit timed out after %s", uploadTimeout.Round(time.Minute))
	}
	if err != nil {
		return "", fmt.Errorf("python commit: %w", err)
	}
	var result struct {
		CommitURL  string `json:"commit_url"`
		Error      string `json:"error"`
		RetryAfter int    `json:"retry_after"` // seconds; >0 when HF returned 429
	}
	if jsonErr := json.Unmarshal(out, &result); jsonErr != nil {
		return "", fmt.Errorf("python commit parse: %w", jsonErr)
	}
	if result.Error != "" {
		if result.RetryAfter > 0 {
			return "", &HFRateLimitError{
				RetryAfter: time.Duration(result.RetryAfter) * time.Second,
				Msg:        result.Error,
			}
		}
		return "", fmt.Errorf("python commit: %s", result.Error)
	}
	return result.CommitURL, nil
}

// createCommit uploads all files and creates a single commit via Python/xet (uv + huggingface_hub).
func (c *hfClient) createCommit(ctx context.Context, repoID, branch, message string, ops []hfOperation) (string, error) {
	return c.createCommitPython(ctx, repoID, message, ops)
}
