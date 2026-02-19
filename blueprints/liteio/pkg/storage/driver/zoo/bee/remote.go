package bee

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

type remoteNode struct {
	id      int
	baseURL string
	client  *http.Client
}

type listResponseItem struct {
	Key         string `json:"key"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Created     int64  `json:"created"`
	Updated     int64  `json:"updated"`
}

func newRemoteNode(id int, endpoint string) (*remoteNode, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("bee: empty peer endpoint")
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}
	endpoint = strings.TrimRight(endpoint, "/")

	r := &remoteNode{
		id:      id,
		baseURL: endpoint,
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				MaxIdleConns:        4096,
				MaxIdleConnsPerHost: 1024,
				MaxConnsPerHost:     0, // Unlimited; bounded by caller concurrency and OS limits.
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
			},
		},
	}

	if err := r.ping(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *remoteNode) ping() error {
	resp, err := r.client.Get(r.baseURL + "/v1/ping")
	if err != nil {
		return fmt.Errorf("bee: peer %s unreachable: %w", r.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bee: peer %s ping failed: %s", r.baseURL, strings.TrimSpace(string(body)))
	}
	return nil
}

func (r *remoteNode) write(bucket, key, contentType string, data []byte, ts int64) (*nodeEntry, error) {
	u := r.baseURL + "/v1/object?bucket=" + url.QueryEscape(bucket) +
		"&key=" + url.QueryEscape(key) +
		"&content_type=" + url.QueryEscape(contentType) +
		"&timestamp=" + strconv.FormatInt(ts, 10)

	req, err := http.NewRequest(http.MethodPut, u, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bee: peer write %s: %w", r.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("bee: peer write status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	e, err := parseEntryHeaders(resp.Header)
	if err != nil {
		return nil, err
	}
	if e.size == 0 {
		e.size = int64(len(data))
	}
	if e.contentType == "" {
		e.contentType = contentType
	}
	return e, nil
}

func (r *remoteNode) read(bucket, key string) ([]byte, *nodeEntry, error) {
	u := r.baseURL + "/v1/object?bucket=" + url.QueryEscape(bucket) + "&key=" + url.QueryEscape(key)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("bee: peer read %s: %w", r.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, storage.ErrNotExist
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("bee: peer read status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	e, err := parseEntryHeaders(resp.Header)
	if err != nil {
		return nil, nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if e.size == 0 {
		e.size = int64(len(data))
	}
	return data, e, nil
}

func (r *remoteNode) stat(bucket, key string) (*nodeEntry, error) {
	u := r.baseURL + "/v1/object?bucket=" + url.QueryEscape(bucket) + "&key=" + url.QueryEscape(key)
	req, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bee: peer stat %s: %w", r.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, storage.ErrNotExist
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bee: peer stat status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return parseEntryHeaders(resp.Header)
}

func (r *remoteNode) delete(bucket, key string, ts int64) (bool, error) {
	u := r.baseURL + "/v1/object?bucket=" + url.QueryEscape(bucket) +
		"&key=" + url.QueryEscape(key) +
		"&timestamp=" + strconv.FormatInt(ts, 10)

	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return false, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("bee: peer delete %s: %w", r.baseURL, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, storage.ErrNotExist
	default:
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("bee: peer delete status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func (r *remoteNode) hasBucket(bucket string) bool {
	u := r.baseURL + "/v1/has-bucket?bucket=" + url.QueryEscape(bucket)
	resp, err := r.client.Get(u)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (r *remoteNode) bucketNames() []string {
	resp, err := r.client.Get(r.baseURL + "/v1/buckets")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var out []string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	return out
}

func (r *remoteNode) list(bucket, prefix string, recursive bool) []nodeListItem {
	u := r.baseURL + "/v1/list?bucket=" + url.QueryEscape(bucket) +
		"&prefix=" + url.QueryEscape(prefix)
	if recursive {
		u += "&recursive=1"
	} else {
		u += "&recursive=0"
	}

	resp, err := r.client.Get(u)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var payload []listResponseItem
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}

	out := make([]nodeListItem, 0, len(payload))
	for _, it := range payload {
		out = append(out, nodeListItem{
			key: it.Key,
			entry: &nodeEntry{
				size:        it.Size,
				contentType: it.ContentType,
				created:     it.Created,
				updated:     it.Updated,
			},
		})
	}
	return out
}

func (r *remoteNode) deleteBucket(bucket string, ts int64) {
	u := r.baseURL + "/v1/bucket?bucket=" + url.QueryEscape(bucket) +
		"&timestamp=" + strconv.FormatInt(ts, 10)
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func (r *remoteNode) close() error { return nil }

func parseEntryHeaders(h http.Header) (*nodeEntry, error) {
	parseInt := func(k string) (int64, error) {
		v := strings.TrimSpace(h.Get(k))
		if v == "" {
			return 0, nil
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	}

	size, err := parseInt("X-Bee-Size")
	if err != nil {
		return nil, err
	}
	created, err := parseInt("X-Bee-Created")
	if err != nil {
		return nil, err
	}
	updated, err := parseInt("X-Bee-Updated")
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		updated = created
	}
	if created == 0 {
		created = updated
	}

	entry := &nodeEntry{
		size:        size,
		contentType: h.Get("X-Bee-Content-Type"),
		created:     created,
		updated:     updated,
	}
	if entry.updated == 0 {
		return nil, errors.New("bee: missing entry timestamp headers")
	}
	return entry, nil
}
