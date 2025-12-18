// contract/transport/jsonrpc/client.go
package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// Client calls a JSON-RPC 2.0 endpoint described by a contract.Service.
type Client struct {
	Svc *contract.Service

	// Endpoint is the full URL to the JSON-RPC endpoint.
	// Example: "https://example.com/rpc"
	Endpoint string

	// Token is an optional bearer token.
	Token string

	// Headers are optional default headers.
	Headers map[string]string

	HTTP *http.Client

	nextID uint64
}

// NewClient creates a JSON-RPC client. Endpoint must be set by the caller.
func NewClient(svc *contract.Service) (*Client, error) {
	if svc == nil {
		return nil, errors.New("jsonrpc: nil service")
	}
	h := map[string]string{"content-type": "application/json"}
	if svc.Defaults != nil {
		for k, v := range svc.Defaults.Headers {
			h[k] = v
		}
	}
	return &Client{
		Svc:     svc,
		Headers: h,
		HTTP:    http.DefaultClient,
	}, nil
}

// Call invokes a JSON-RPC method "<resource>.<method>".
// If out is nil, the result is discarded.
func (c *Client) Call(ctx context.Context, resource string, method string, in any, out any) error {
	if strings.TrimSpace(c.Endpoint) == "" {
		return errors.New("jsonrpc: empty Endpoint")
	}

	id := atomic.AddUint64(&c.nextID, 1)
	full := resource + "." + method

	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  full,
		ID:      id,
	}
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		req.Params = b
	}

	var resp rpcResponse
	if err := c.do(ctx, req, &resp); err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("jsonrpc: %s: (%d) %s", full, resp.Error.Code, resp.Error.Message)
	}
	if out == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil
	}
	return json.Unmarshal(resp.Result, out)
}

// BatchCall issues a batch of calls in a single HTTP request.
// Each item uses a generated id and returns results in the same order as requests,
// not in the server response order.
func (c *Client) BatchCall(ctx context.Context, calls []BatchItem) ([]BatchResult, error) {
	if strings.TrimSpace(c.Endpoint) == "" {
		return nil, errors.New("jsonrpc: empty Endpoint")
	}
	if len(calls) == 0 {
		return nil, nil
	}

	reqs := make([]rpcRequest, 0, len(calls))
	idToIndex := make(map[uint64]int, len(calls))

	for i := range calls {
		id := atomic.AddUint64(&c.nextID, 1)
		idToIndex[id] = i

		full := calls[i].Resource + "." + calls[i].Method
		rq := rpcRequest{JSONRPC: "2.0", Method: full, ID: id}
		if calls[i].In != nil {
			b, err := json.Marshal(calls[i].In)
			if err != nil {
				return nil, err
			}
			rq.Params = b
		}
		reqs = append(reqs, rq)
	}

	var rawResp []rpcResponse
	if err := c.do(ctx, reqs, &rawResp); err != nil {
		return nil, err
	}

	results := make([]BatchResult, len(calls))
	for i := range results {
		results[i].Resource = calls[i].Resource
		results[i].Method = calls[i].Method
	}

	for _, rr := range rawResp {
		u, ok := rr.ID.(float64) // JSON numbers decode as float64
		if !ok {
			continue
		}
		id := uint64(u)
		idx, ok := idToIndex[id]
		if !ok {
			continue
		}

		if rr.Error != nil {
			results[idx].Err = fmt.Errorf("jsonrpc: %s.%s: (%d) %s",
				results[idx].Resource, results[idx].Method, rr.Error.Code, rr.Error.Message)
			continue
		}
		results[idx].Result = rr.Result
	}

	return results, nil
}

type BatchItem struct {
	Resource string
	Method   string
	In       any
}

type BatchResult struct {
	Resource string
	Method   string
	Result   json.RawMessage
	Err      error
}

func (c *Client) do(ctx context.Context, req any, out any) error {
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}

	for k, v := range c.Headers {
		httpReq.Header.Set(k, v)
	}
	if c.Token != "" {
		httpReq.Header.Set("authorization", "Bearer "+c.Token)
	}

	// Practical default timeout if caller did not provide one via client.
	// This avoids accidental hangs in simple usage.
	if hc.Timeout == 0 {
		hc = &http.Client{Transport: hc.Transport, Timeout: 60 * time.Second}
	}

	resp, err := hc.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jsonrpc: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}
