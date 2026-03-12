// Package gemini implements an embed.Driver that calls the Google Gemini
// Embedding API (generativelanguage.googleapis.com).
package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

const (
	// gemini-embedding-2-preview is the current best Gemini embedding model (3072-dim,
	// Matryoshka-capable). Free-tier limits: 5 RPM and 100 RPD.
	defaultModel     = "gemini-embedding-2-preview"
	defaultBatchSize = 100 // batchEmbedContents maximum; use max to minimise RPM consumption
	apiBase          = "https://generativelanguage.googleapis.com/v1beta"
	localEnvRelPath  = ".local.env" // relative to $HOME/data/
)

func init() {
	embed.Register("gemini", func() embed.Driver { return &Driver{} })
}

type Driver struct {
	apiKey    string
	model     string
	outputDim int  // Matryoshka override (0 = model default)
	dim       int  // actual dim from probe
	batchSize int
	client    *http.Client
	baseURL   string // overridable for tests
}

func (d *Driver) Name() string {
	if d.model != "" {
		return "gemini/" + d.model
	}
	return "gemini"
}

func (d *Driver) Dimension() int { return d.dim }

func (d *Driver) Open(ctx context.Context, cfg embed.Config) error {
	// API key: cfg.Addr → GEMINI_API_KEY env → $HOME/data/.local.env
	apiKey := cfg.Addr
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		home, _ := os.UserHomeDir()
		apiKey = loadKeyFromFile(filepath.Join(home, "data", localEnvRelPath), "GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("gemini: GEMINI_API_KEY not set\n" +
			"  Set it in the environment:  export GEMINI_API_KEY=<key>\n" +
			"  Or add it to $HOME/data/.local.env")
	}
	d.apiKey = apiKey

	modelInput := cfg.Model
	if modelInput == "" {
		modelInput = defaultModel
	}
	d.model, d.outputDim = parseModelName(modelInput)

	d.batchSize = cfg.BatchSize
	if d.batchSize <= 0 || d.batchSize > defaultBatchSize {
		// Cap at API maximum: batchEmbedContents accepts at most 100 requests per call.
		// Use max batch size (100) to minimise the number of API calls and stay within
		// free-tier RPM limits (5 RPM / 100 RPD for gemini-embedding-2-preview).
		d.batchSize = defaultBatchSize
	}

	d.client = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        16,
			MaxIdleConnsPerHost: 16,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	if d.baseURL == "" {
		d.baseURL = apiBase
	}

	// Probe: confirm key works, discover actual dimension.
	vecs, err := d.callBatch(ctx, []string{"hello"})
	if err != nil {
		return fmt.Errorf("gemini: probe failed: %w", err)
	}
	if len(vecs) == 0 || len(vecs[0].Values) == 0 {
		return fmt.Errorf("gemini: probe returned empty embedding")
	}
	d.dim = len(vecs[0].Values)
	return nil
}

func (d *Driver) Close() error {
	if d.client != nil {
		d.client.CloseIdleConnections()
	}
	return nil
}

func (d *Driver) Embed(ctx context.Context, inputs []embed.Input) ([]embed.Vector, error) {
	texts := make([]string, len(inputs))
	for i, inp := range inputs {
		texts[i] = inp.Text
	}

	var all []embed.Vector
	for i := 0; i < len(texts); i += d.batchSize {
		end := i + d.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := d.callBatch(ctx, texts[i:end])
		if err != nil {
			return nil, err
		}
		all = append(all, vecs...)
	}
	return all, nil
}

// --- API types ---

type embedContentRequest struct {
	Model                string       `json:"model"`
	Content              embedContent `json:"content"`
	TaskType             string       `json:"taskType,omitempty"`
	OutputDimensionality int          `json:"outputDimensionality,omitempty"`
}

type embedContent struct {
	Parts []embedPart `json:"parts"`
}

type embedPart struct {
	Text string `json:"text"`
}

type batchEmbedRequest struct {
	Requests []embedContentRequest `json:"requests"`
}

type embeddingValue struct {
	Values []float32 `json:"values"`
}

type batchEmbedResponse struct {
	Embeddings []embeddingValue `json:"embeddings"`
}

func (d *Driver) callBatch(ctx context.Context, texts []string) ([]embed.Vector, error) {
	reqs := make([]embedContentRequest, len(texts))
	for i, t := range texts {
		r := embedContentRequest{
			Model:   "models/" + d.model,
			Content: embedContent{Parts: []embedPart{{Text: t}}},
			// RETRIEVAL_DOCUMENT is the correct task type for offline indexing.
			// For query-time use, callers should use a separate driver instance configured
			// with taskType RETRIEVAL_QUERY via a future Config extension.
			TaskType: "RETRIEVAL_DOCUMENT",
		}
		if d.outputDim > 0 {
			r.OutputDimensionality = d.outputDim
		}
		reqs[i] = r
	}

	body, err := json.Marshal(batchEmbedRequest{Requests: reqs})
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal: %w", err)
	}

	// Pass the API key as a header rather than a URL query parameter so that
	// the key is never exposed in HTTP error messages or server access logs.
	url := fmt.Sprintf("%s/models/%s:batchEmbedContents", d.baseURL, d.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", d.apiKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var embResp batchEmbedResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("gemini: decode: %w", err)
	}
	if len(embResp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("gemini: expected %d embeddings, got %d", len(texts), len(embResp.Embeddings))
	}

	result := make([]embed.Vector, len(embResp.Embeddings))
	for i, e := range embResp.Embeddings {
		result[i] = embed.Vector{Values: e.Values}
	}
	return result, nil
}

// parseModelName splits "model:dim" into (model, dim). dim=0 if no suffix.
func parseModelName(s string) (model string, dim int) {
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		if n, err := strconv.Atoi(s[idx+1:]); err == nil {
			return s[:idx], n
		}
	}
	return s, 0
}

// loadKeyFromFile reads KEY=VALUE or export KEY="VALUE" lines from a shell env file.
func loadKeyFromFile(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eqIdx])
		if k != key {
			continue
		}
		v := strings.TrimSpace(line[eqIdx+1:])
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		return v
	}
	return ""
}
