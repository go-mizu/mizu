package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Change is a single durable state change from the server.
type Change struct {
	Cursor uint64    `json:"cursor"`
	Scope  string    `json:"scope"`
	Entity string    `json:"entity"`
	ID     string    `json:"id"`
	Op     Op        `json:"op"`
	Data   []byte    `json:"data,omitempty"`
	Time   time.Time `json:"time"`
}

// Result describes the outcome of applying a mutation.
type Result struct {
	OK      bool     `json:"ok"`
	Cursor  uint64   `json:"cursor,omitempty"`
	Code    string   `json:"code,omitempty"`
	Error   string   `json:"error,omitempty"`
	Changes []Change `json:"changes,omitempty"`
}

// PushRequest is the payload for POST /_sync/push
type PushRequest struct {
	Mutations []Mutation `json:"mutations"`
}

// PushResponse is returned from POST /_sync/push
type PushResponse struct {
	Results []Result `json:"results"`
}

// PullRequest is the payload for POST /_sync/pull
type PullRequest struct {
	Scope  string `json:"scope,omitempty"`
	Cursor uint64 `json:"cursor"`
	Limit  int    `json:"limit,omitempty"`
}

// PullResponse is returned from POST /_sync/pull
type PullResponse struct {
	Changes []Change `json:"changes"`
	HasMore bool     `json:"has_more"`
}

// SnapshotRequest is the payload for POST /_sync/snapshot
type SnapshotRequest struct {
	Scope string `json:"scope,omitempty"`
}

// SnapshotResponse is returned from POST /_sync/snapshot
type SnapshotResponse struct {
	Data   map[string]map[string][]byte `json:"data"`
	Cursor uint64                       `json:"cursor"`
}

// Transport handles HTTP communication with the sync server.
type Transport struct {
	baseURL string
	http    *http.Client
}

// NewTransport creates a new HTTP transport.
func NewTransport(baseURL string, httpClient *http.Client) *Transport {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Transport{
		baseURL: baseURL,
		http:    httpClient,
	}
}

// Push sends mutations to the server.
func (t *Transport) Push(ctx context.Context, mutations []Mutation) (*PushResponse, error) {
	req := PushRequest{Mutations: mutations}
	var resp PushResponse
	if err := t.post(ctx, "/push", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Pull fetches changes since a cursor.
func (t *Transport) Pull(ctx context.Context, scope string, cursor uint64, limit int) (*PullResponse, error) {
	req := PullRequest{Scope: scope, Cursor: cursor, Limit: limit}
	var resp PullResponse
	if err := t.post(ctx, "/pull", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Snapshot fetches the full state for a scope.
func (t *Transport) Snapshot(ctx context.Context, scope string) (*SnapshotResponse, error) {
	req := SnapshotRequest{Scope: scope}
	var resp SnapshotResponse
	if err := t.post(ctx, "/snapshot", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (t *Transport) post(ctx context.Context, path string, body, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusGone {
		return ErrCursorTooOld
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Code  string `json:"code"`
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			if errResp.Code == "conflict" {
				return ErrConflict
			}
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	return nil
}
