// Package llamacpp implements an embed.Driver that calls a llama.cpp server's
// OpenAI-compatible /v1/embeddings endpoint.
//
// Supports multiple server addresses for round-robin load balancing:
//
//	cfg.Addr = "http://host1:8087,http://host2:8088"
package llamacpp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

const (
	defaultAddr      = "http://localhost:8086"
	defaultBatchSize = 16
)

func init() {
	embed.Register("llamacpp", func() embed.Driver { return &Driver{} })
}

// Driver calls a llama.cpp server for embedding generation.
// Supports multiple server addresses for round-robin load balancing.
type Driver struct {
	addrs     []string
	model     string
	dim       int
	batchSize int
	client    *http.Client
	nextAddr  atomic.Uint64
}

func (d *Driver) Name() string {
	if d.model != "" {
		return "llamacpp/" + d.model
	}
	return "llamacpp"
}

func (d *Driver) Dimension() int { return d.dim }

// pickAddr returns the next server address in round-robin order.
func (d *Driver) pickAddr() string {
	if len(d.addrs) == 1 {
		return d.addrs[0]
	}
	idx := d.nextAddr.Add(1) - 1
	return d.addrs[idx%uint64(len(d.addrs))]
}

// Open connects to the llama.cpp server(s) and probes for the embedding dimension.
// cfg.Addr supports comma-separated addresses for round-robin: "http://h1:8087,http://h2:8088"
func (d *Driver) Open(ctx context.Context, cfg embed.Config) error {
	addr := cfg.Addr
	if addr == "" {
		addr = defaultAddr
	}
	for _, a := range strings.Split(addr, ",") {
		a = strings.TrimSpace(a)
		if a != "" {
			d.addrs = append(d.addrs, a)
		}
	}
	if len(d.addrs) == 0 {
		return fmt.Errorf("llamacpp: no server addresses configured")
	}

	d.model = cfg.Model
	d.batchSize = cfg.BatchSize
	if d.batchSize <= 0 {
		d.batchSize = defaultBatchSize
	}
	d.client = &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        64,
			MaxIdleConnsPerHost: 64,
			MaxConnsPerHost:     64,
			IdleConnTimeout:     120 * time.Second,
		},
	}

	// Health check all servers.
	for _, a := range d.addrs {
		if err := d.healthCheckAddr(ctx, a); err != nil {
			return fmt.Errorf("llamacpp: server not reachable at %s\n\n"+
				"  Start the server:\n"+
				"    cd docker/llamacpp && docker compose up llamacpp-embed -d\n\n"+
				"  Or manually:\n"+
				"    llama-server --model <model.gguf> --embedding --pooling mean --port 8080\n\n"+
				"  Error: %w", a, err)
		}
	}

	// Probe to discover dimension (use first server).
	vecs, err := d.embedAddr(ctx, d.addrs[0], []string{"hello"})
	if err != nil {
		return fmt.Errorf("llamacpp: probe failed at %s: %w\n"+
			"  Ensure the server was started with --embedding flag", d.addrs[0], err)
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return fmt.Errorf("llamacpp: server returned empty embedding — is --embedding flag enabled?")
	}
	d.dim = len(vecs[0])

	if len(d.addrs) > 1 {
		fmt.Fprintf(io.Discard, "llamacpp: %d servers configured\n", len(d.addrs))
	}
	return nil
}

func (d *Driver) Close() error {
	d.client.CloseIdleConnections()
	return nil
}

// Embed generates embeddings for a batch of inputs.
func (d *Driver) Embed(ctx context.Context, inputs []embed.Input) ([]embed.Vector, error) {
	texts := make([]string, len(inputs))
	for i, inp := range inputs {
		texts[i] = inp.Text
	}

	var allVecs [][]float32
	for i := 0; i < len(texts); i += d.batchSize {
		end := i + d.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch, err := d.embedAddr(ctx, d.pickAddr(), texts[i:end])
		if err != nil {
			return nil, err
		}
		allVecs = append(allVecs, batch...)
	}

	result := make([]embed.Vector, len(allVecs))
	for i, v := range allVecs {
		result[i] = embed.Vector{Values: v}
	}
	return result, nil
}

// healthCheckAddr pings /health on the given server address.
func (d *Driver) healthCheckAddr(ctx context.Context, addr string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check HTTP %d", resp.StatusCode)
	}
	return nil
}

// --- OpenAI-compatible embeddings API ---

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model,omitempty"`
}

type embeddingResponse struct {
	Data []embeddingData `json:"data"`
}

type embeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

func (d *Driver) embedAddr(ctx context.Context, addr string, texts []string) ([][]float32, error) {
	reqBody := embeddingRequest{
		Input: texts,
		Model: d.model,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("llamacpp: marshal: %w", err)
	}

	url := addr + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llamacpp: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llamacpp: request to %s: %w", addr, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llamacpp: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llamacpp: HTTP %d from %s: %s", resp.StatusCode, addr, string(respBody))
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("llamacpp: decode: %w", err)
	}

	// Sort by index to maintain input order.
	vecs := make([][]float32, len(texts))
	for _, d := range embResp.Data {
		if d.Index < len(vecs) {
			vecs[d.Index] = d.Embedding
		}
	}

	for i, v := range vecs {
		if v == nil {
			return nil, fmt.Errorf("llamacpp: missing embedding for input %d", i)
		}
	}
	return vecs, nil
}
