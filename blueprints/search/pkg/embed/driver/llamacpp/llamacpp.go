// Package llamacpp implements an embed.Driver that calls a llama.cpp server's
// OpenAI-compatible /v1/embeddings endpoint.
package llamacpp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

const (
	defaultAddr      = "http://localhost:8086"
	defaultBatchSize = 64
)

func init() {
	embed.Register("llamacpp", func() embed.Driver { return &Driver{} })
}

// Driver calls a llama.cpp server for embedding generation.
type Driver struct {
	addr      string
	model     string
	dim       int
	batchSize int
	client    *http.Client
}

func (d *Driver) Name() string {
	if d.model != "" {
		return "llamacpp/" + d.model
	}
	return "llamacpp"
}

func (d *Driver) Dimension() int { return d.dim }

// Open connects to the llama.cpp server and probes for the embedding dimension.
func (d *Driver) Open(ctx context.Context, cfg embed.Config) error {
	d.addr = cfg.Addr
	if d.addr == "" {
		d.addr = defaultAddr
	}
	d.model = cfg.Model
	d.batchSize = cfg.BatchSize
	if d.batchSize <= 0 {
		d.batchSize = defaultBatchSize
	}
	d.client = &http.Client{Timeout: 120 * time.Second}

	// Health check first — gives a clearer error than a probe failure.
	if err := d.healthCheck(ctx); err != nil {
		return fmt.Errorf("llamacpp: server not reachable at %s\n\n"+
			"  Start the server:\n"+
			"    cd docker/llamacpp && docker compose up llamacpp-embed -d\n\n"+
			"  Or manually:\n"+
			"    llama-server --model <model.gguf> --embedding --pooling mean --port 8080\n\n"+
			"  Error: %w", d.addr, err)
	}

	// Probe to discover dimension.
	vecs, err := d.embed(ctx, []string{"hello"})
	if err != nil {
		return fmt.Errorf("llamacpp: probe failed at %s: %w\n"+
			"  Ensure the server was started with --embedding flag", d.addr, err)
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return fmt.Errorf("llamacpp: server returned empty embedding — is --embedding flag enabled?")
	}
	d.dim = len(vecs[0])
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
		batch, err := d.embed(ctx, texts[i:end])
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

// healthCheck pings /health to verify the server is running.
func (d *Driver) healthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.addr+"/health", nil)
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

func (d *Driver) embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := embeddingRequest{
		Input: texts,
		Model: d.model,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("llamacpp: marshal: %w", err)
	}

	url := d.addr + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llamacpp: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llamacpp: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llamacpp: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
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
