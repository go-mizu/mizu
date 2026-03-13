package cli

// Pure-Go HuggingFace Hub client for dataset publishing.
// Implements LFS basic/multipart upload and the ndjson commit API —
// no Python dependency required.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const hfHubURL = "https://huggingface.co"

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

// hfLFSInfo holds a file's SHA-256 and byte size for LFS operations.
type hfLFSInfo struct {
	SHA256 string
	Size   int64
}

func hfComputeLFS(path string) (hfLFSInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return hfLFSInfo{}, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return hfLFSInfo{}, err
	}
	return hfLFSInfo{SHA256: hex.EncodeToString(h.Sum(nil)), Size: n}, nil
}

// uploadLFS uploads a local file to HF Hub via the LFS protocol.
// Supports both "basic" (single PUT) and HF's "multipart" chunked upload.
// If the server already has this exact file (same SHA-256), the upload is skipped.
func (c *hfClient) uploadLFS(ctx context.Context, repoID, localPath string) (hfLFSInfo, error) {
	info, err := hfComputeLFS(localPath)
	if err != nil {
		return hfLFSInfo{}, err
	}

	// LFS batch — discover whether and how to upload
	batchBody, _ := json.Marshal(map[string]interface{}{
		"operation": "upload",
		"transfers": []string{"basic"},
		"objects":   []map[string]interface{}{{"oid": info.SHA256, "size": info.Size}},
		"ref":       map[string]string{"name": "main"},
	})
	batchURL := fmt.Sprintf("%s/datasets/%s.git/info/lfs/objects/batch", hfHubURL, repoID)
	req, _ := http.NewRequestWithContext(ctx, "POST", batchURL, bytes.NewReader(batchBody))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	resp, err := c.http.Do(req)
	if err != nil {
		return hfLFSInfo{}, fmt.Errorf("lfs batch: %w", err)
	}

	var batchResp struct {
		Transfer string `json:"transfer"`
		Objects  []struct {
			Actions struct {
				Upload *struct {
					HRef   string            `json:"href"`
					Header map[string]string `json:"header"`
				} `json:"upload"`
				Verify *struct {
					HRef   string            `json:"href"`
					Header map[string]string `json:"header"`
				} `json:"verify"`
			} `json:"actions"`
		} `json:"objects"`
	}
	json.NewDecoder(resp.Body).Decode(&batchResp)
	resp.Body.Close()

	if len(batchResp.Objects) == 0 || batchResp.Objects[0].Actions.Upload == nil {
		return info, nil // file already on server
	}
	obj := batchResp.Objects[0]

	// Upload
	if batchResp.Transfer == "multipart" {
		if err := c.lfsMultipart(ctx, localPath, obj.Actions.Upload.HRef, obj.Actions.Upload.Header, info.Size); err != nil {
			return hfLFSInfo{}, fmt.Errorf("lfs multipart %s: %w", localPath, err)
		}
	} else {
		if err := c.lfsBasic(ctx, localPath, obj.Actions.Upload.HRef, obj.Actions.Upload.Header, info.Size); err != nil {
			return hfLFSInfo{}, fmt.Errorf("lfs basic %s: %w", localPath, err)
		}
	}

	// Verify
	if obj.Actions.Verify != nil {
		vb, _ := json.Marshal(map[string]interface{}{"oid": info.SHA256, "size": info.Size})
		vreq, _ := http.NewRequestWithContext(ctx, "POST", obj.Actions.Verify.HRef, bytes.NewReader(vb))
		vreq.Header.Set("Content-Type", "application/vnd.git-lfs+json")
		for k, v := range obj.Actions.Verify.Header {
			vreq.Header.Set(k, v)
		}
		vresp, err := c.http.Do(vreq)
		if err != nil {
			return hfLFSInfo{}, fmt.Errorf("lfs verify: %w", err)
		}
		b, _ := io.ReadAll(vresp.Body)
		vresp.Body.Close()
		if vresp.StatusCode >= 300 {
			return hfLFSInfo{}, fmt.Errorf("lfs verify HTTP %d: %s", vresp.StatusCode, string(b))
		}
	}
	return info, nil
}

func (c *hfClient) lfsBasic(ctx context.Context, localPath, href string, headers map[string]string, size int64) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	req, _ := http.NewRequestWithContext(ctx, "PUT", href, f)
	req.ContentLength = size
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *hfClient) lfsMultipart(ctx context.Context, localPath, href string, headers map[string]string, size int64) error {
	chunkSize := size // default: one chunk
	if cs, ok := headers["X-Chunk-Size"]; ok {
		fmt.Sscanf(cs, "%d", &chunkSize)
	}
	if chunkSize <= 0 {
		chunkSize = size
	}

	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, chunkSize)
	for part := 1; ; part++ {
		n, err := io.ReadFull(f, buf)
		if n == 0 {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return err
		}
		preq, _ := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s?part=%d", href, part), bytes.NewReader(buf[:n]))
		preq.ContentLength = int64(n)
		for k, v := range headers {
			preq.Header.Set(k, v)
		}
		presp, perr := c.http.Do(preq)
		if perr != nil {
			return fmt.Errorf("part %d: %w", part, perr)
		}
		b, _ := io.ReadAll(presp.Body)
		presp.Body.Close()
		if presp.StatusCode >= 300 {
			return fmt.Errorf("part %d HTTP %d: %s", part, presp.StatusCode, string(b))
		}
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			break
		}
	}

	// Complete multipart upload
	creq, _ := http.NewRequestWithContext(ctx, "POST", href+"?commit=1", nil)
	creq.ContentLength = 0
	for k, v := range headers {
		creq.Header.Set(k, v)
	}
	cresp, err := c.http.Do(creq)
	if err != nil {
		return fmt.Errorf("multipart commit: %w", err)
	}
	b, _ := io.ReadAll(cresp.Body)
	cresp.Body.Close()
	if cresp.StatusCode >= 300 {
		return fmt.Errorf("multipart commit HTTP %d: %s", cresp.StatusCode, string(b))
	}
	return nil
}

// hfOperation is a file to include in a commit.
type hfOperation struct {
	LocalPath  string
	PathInRepo string
}

// createCommit uploads all files (large via LFS, small inline) and creates a single commit.
func (c *hfClient) createCommit(ctx context.Context, repoID, branch, message string, ops []hfOperation) (string, error) {
	const smallThreshold = 5 * 1024 * 1024

	type lfsEntry struct {
		path string
		info hfLFSInfo
	}
	var lfsEntries []lfsEntry
	var inlineOps []hfOperation

	for _, op := range ops {
		fi, err := os.Stat(op.LocalPath)
		if err != nil {
			return "", fmt.Errorf("stat %s: %w", op.LocalPath, err)
		}
		if fi.Size() >= smallThreshold {
			fmt.Printf("    uploading %-50s  %s\n", op.PathInRepo, ccFmtBytes(fi.Size()))
			info, err := c.uploadLFS(ctx, repoID, op.LocalPath)
			if err != nil {
				return "", fmt.Errorf("lfs %s: %w", op.PathInRepo, err)
			}
			lfsEntries = append(lfsEntries, lfsEntry{path: op.PathInRepo, info: info})
		} else {
			inlineOps = append(inlineOps, op)
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	enc.Encode(map[string]interface{}{
		"key":   "header",
		"value": map[string]string{"summary": message, "description": ""},
	})
	for _, e := range lfsEntries {
		enc.Encode(map[string]interface{}{
			"key": "lfsFile",
			"value": map[string]interface{}{
				"path": e.path,
				"algo": "sha256",
				"oid":  e.info.SHA256,
				"size": e.info.Size,
			},
		})
	}
	for _, op := range inlineOps {
		data, err := os.ReadFile(op.LocalPath)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", op.LocalPath, err)
		}
		enc.Encode(map[string]interface{}{
			"key": "file",
			"value": map[string]string{
				"path":     op.PathInRepo,
				"encoding": "base64",
				"content":  base64.StdEncoding.EncodeToString(data),
			},
		})
	}

	url := fmt.Sprintf("%s/api/datasets/%s/commit/%s", hfHubURL, repoID, branch)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, &buf)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("commit HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		CommitURL string `json:"commitUrl"`
	}
	json.Unmarshal(body, &result)
	return result.CommitURL, nil
}
