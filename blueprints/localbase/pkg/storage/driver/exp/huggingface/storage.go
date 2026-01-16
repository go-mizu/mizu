// File: lib/storage/driver/huggingface/storage.go
package huggingface

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
)

// DSN examples:
//
//   - Models (public):
//     huggingface://HuggingFaceH4/zephyr-7b-beta?repo_type=model
//
//   - Datasets (public):
//     huggingface://datasets/glue?repo_type=dataset
//
//   - Private repo with token in query:
//     huggingface://HuggingFaceH4/zephyr-7b-beta?repo_type=model&token=hf_xxx
//
//   - Private repo with token in userinfo:
//     huggingface://hf_xxx@HuggingFaceH4/zephyr-7b-beta?repo_type=model
//
// Supported repo_type values:
//   - "model" (default)
//   - "dataset"
//   - "space"
//
// This driver is read only. All mutating operations return storage.ErrUnsupported.
func init() {
	d := &driver{}
	storage.Register("huggingface", d)
	storage.Register("hf", d)
}

type driver struct{}

// Open parses the DSN and returns a Storage bound to a single Hugging Face repo.
func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("storage/huggingface: invalid dsn: %w", err)
	}
	if u.Scheme == "" {
		return nil, errors.New("storage/huggingface: missing scheme in dsn")
	}

	repoID := strings.Trim(u.Host+u.Path, "/")
	if repoID == "" {
		return nil, errors.New("storage/huggingface: missing repo id in dsn")
	}

	q := u.Query()

	repoType := q.Get("repo_type")
	if repoType == "" {
		repoType = "model"
	}

	revision := q.Get("revision")
	if revision == "" {
		revision = "main"
	}

	baseURL := q.Get("base_url")
	if baseURL == "" {
		baseURL = "https://huggingface.co"
	}

	token := q.Get("token")
	if token == "" && u.User != nil {
		// Allow token in userinfo: token@org/repo or user:token@org/repo
		if pw, ok := u.User.Password(); ok && pw != "" {
			token = pw
		} else {
			token = u.User.Username()
		}
	}

	timeout := 60 * time.Second
	if ts := q.Get("timeout"); ts != "" {
		if secs, err := strconv.Atoi(ts); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	st := &hfStorage{
		client:   client,
		baseURL:  strings.TrimRight(baseURL, "/"),
		repoID:   repoID,
		repoType: strings.ToLower(repoType),
		revision: revision,
		token:    token,
	}

	return st, nil
}

// hfStorage implements storage.Storage for a single Hugging Face repo.
type hfStorage struct {
	client   *http.Client
	baseURL  string
	repoID   string
	repoType string // "model", "dataset", "space"
	revision string
	token    string
}

func (s *hfStorage) Bucket(name string) storage.Bucket {
	// This storage is bound to a single repo.
	// If caller passes a different name, treat it as another repo under same account.
	repoID := s.repoID
	if name != "" && name != s.repoID {
		repoID = name
	}

	clone := *s
	clone.repoID = repoID
	return &hfBucket{st: &clone}
}

func (s *hfStorage) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	// Listing all repos for an account is not implemented.
	return nil, storage.ErrUnsupported
}

func (s *hfStorage) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	return nil, storage.ErrUnsupported
}

func (s *hfStorage) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	return storage.ErrUnsupported
}

func (s *hfStorage) Features() storage.Features {
	// These are generic and conservative.
	return storage.Features{
		"directories": true,
		"public_url":  true,
	}
}

func (s *hfStorage) Close() error {
	// Nothing to close for now.
	return nil
}

// hfBucket implements storage.Bucket for a Hugging Face repo.
type hfBucket struct {
	st *hfStorage
}

func (b *hfBucket) Name() string {
	return b.st.repoID
}

func (b *hfBucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	// We keep this minimal to avoid extra API calls.
	return &storage.BucketInfo{
		Name:     b.Name(),
		Metadata: map[string]string{"repo_type": b.st.repoType, "revision": b.st.revision},
	}, nil
}

func (b *hfBucket) Features() storage.Features {
	return b.st.Features()
}

func (b *hfBucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	// Write is not implemented. Hugging Face recommends git or huggingface_hub for push.
	return nil, storage.ErrUnsupported
}

func (b *hfBucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	urlStr := b.st.fileURL(key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, nil, err
	}
	if b.st.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.st.token)
	}

	if offset > 0 || length != 0 {
		var rangeVal string
		switch {
		case length > 0:
			rangeVal = fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)
		case length < 0:
			rangeVal = fmt.Sprintf("bytes=%d-", offset)
		default:
			rangeVal = fmt.Sprintf("bytes=%d-", offset)
		}
		req.Header.Set("Range", rangeVal)
	}

	resp, err := b.st.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close() // ignore close error in error path
		return nil, nil, storage.ErrNotExist
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = resp.Body.Close() // ignore close error in error path
		return nil, nil, storage.ErrPermission
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		defer func() {
			_ = resp.Body.Close()
		}()
		return nil, nil, fmt.Errorf("storage/huggingface: open %q: unexpected status %s", key, resp.Status)
	}

	obj := b.objectFromHeaders(key, resp.Header)
	return resp.Body, obj, nil
}

func (b *hfBucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	urlStr := b.st.fileURL(key)

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, urlStr, nil)
	if err != nil {
		return nil, err
	}
	if b.st.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.st.token)
	}

	resp, err := b.st.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, storage.ErrNotExist
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, storage.ErrPermission
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("storage/huggingface: stat %q: unexpected status %s", key, resp.Status)
	}

	return b.objectFromHeaders(key, resp.Header), nil
}

func (b *hfBucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	return storage.ErrUnsupported
}

func (b *hfBucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	return nil, storage.ErrUnsupported
}

func (b *hfBucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	return nil, storage.ErrUnsupported
}

func (b *hfBucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	urlStr := b.st.treeURL()

	values := url.Values{}
	// Default to recursive listing.
	recursive := true
	if v, ok := opts["recursive"].(bool); ok {
		recursive = v
	}
	if recursive {
		values.Set("recursive", "1")
	}
	// Ask for extra information (size etc.) when available.
	values.Set("expand", "1")

	if enc := values.Encode(); enc != "" {
		urlStr = urlStr + "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	if b.st.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.st.token)
	}

	resp, err := b.st.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, storage.ErrNotExist
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, storage.ErrPermission
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("storage/huggingface: list: unexpected status %s", resp.Status)
	}

	var raw []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("storage/huggingface: decode tree: %w", err)
	}

	var objs []*storage.Object
	for _, entry := range raw {
		p := entryString(entry, "path")
		if p == "" {
			p = entryString(entry, "rfilename")
		}
		if p == "" {
			continue
		}
		if prefix != "" && !strings.HasPrefix(p, prefix) {
			continue
		}

		typ := entryString(entry, "type")
		if typ == "directory" || typ == "dir" {
			continue
		}

		size := entryInt64(entry, "size")
		if size < 0 {
			size = entryInt64(entry, "filesize")
		}
		if size < 0 {
			if lfs, ok := entry["LFS"].(map[string]any); ok {
				size = entryInt64(lfs, "size")
			}
		}

		objs = append(objs, &storage.Object{
			Bucket: b.Name(),
			Key:    p,
			Size:   size,
			Hash:   storage.Hashes{},
			// Other metadata is not exposed here.
			Metadata: map[string]string{},
		})
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	end := len(objs)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	slice := objs[offset:end]

	return &objectSliceIter{objs: slice}, nil
}

func (b *hfBucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	if strings.ToUpper(method) != http.MethodGet {
		return "", storage.ErrUnsupported
	}
	// For public repos this works directly.
	// For private repos this URL requires Authorization header and may not be usable as is.
	return b.st.fileURL(key), nil
}

// objectFromHeaders builds a minimal Object from HTTP response headers.
func (b *hfBucket) objectFromHeaders(key string, h http.Header) *storage.Object {
	size := int64(-1)
	if cl := h.Get("Content-Length"); cl != "" {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
			size = n
		}
	}

	etag := strings.Trim(h.Get("ETag"), "\"")

	return &storage.Object{
		Bucket:      b.Name(),
		Key:         key,
		Size:        size,
		ContentType: h.Get("Content-Type"),
		ETag:        etag,
		Hash:        storage.Hashes{},
		Metadata:    map[string]string{},
	}
}

// objectSliceIter implements storage.ObjectIter over an in memory slice.
type objectSliceIter struct {
	objs []*storage.Object
	idx  int
}

func (it *objectSliceIter) Next() (*storage.Object, error) {
	if it.idx >= len(it.objs) {
		return nil, nil
	}
	o := it.objs[it.idx]
	it.idx++
	return o, nil
}

func (it *objectSliceIter) Close() error {
	it.objs = nil
	return nil
}

// Helper methods on hfStorage.

func (s *hfStorage) fileURL(key string) string {
	key = strings.TrimLeft(key, "/")

	var prefix string
	switch s.repoType {
	case "dataset", "datasets":
		prefix = "datasets"
	case "space", "spaces":
		prefix = "spaces"
	default:
		prefix = ""
	}

	if prefix != "" {
		// https://huggingface.co/datasets/{repo_id}/resolve/{revision}/{file}
		return s.baseURL + "/" + path.Join(prefix, s.repoID, "resolve", s.revision, key)
	}

	// Models: https://huggingface.co/{repo_id}/resolve/{revision}/{file}
	return s.baseURL + "/" + path.Join(s.repoID, "resolve", s.revision, key)
}

func (s *hfStorage) treeURL() string {
	var segment string
	switch s.repoType {
	case "dataset", "datasets":
		segment = "datasets"
	case "space", "spaces":
		segment = "spaces"
	default:
		segment = "models"
	}

	// https://huggingface.co/api/models/{repo_id}/tree/{revision}
	return s.baseURL + "/" + path.Join("api", segment, s.repoID, "tree", s.revision)
}

// Small helpers for JSON maps.

func entryString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func entryInt64(m map[string]any, key string) int64 {
	v, ok := m[key]
	if !ok {
		return -1
	}
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return i
		}
	}
	return -1
}
