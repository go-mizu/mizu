package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmbedProvider generates vector embeddings for text.
type EmbedProvider interface {
	// Embed returns embedding vectors for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float64, error)

	// Model returns the model identifier.
	Model() string

	// Dims returns the dimensionality of the embedding vectors.
	Dims() int
}

// OpenAIEmbedder uses the OpenAI embeddings API (or any compatible endpoint)
// with the text-embedding-3-small model. This matches OpenClaw's embedding
// provider.
type OpenAIEmbedder struct {
	apiKey   string
	model    string
	dims     int
	endpoint string
	client   *http.Client
}

// NewOpenAIEmbedder creates an embedder using the given API key.
// It targets the text-embedding-3-small model (1536 dimensions).
func NewOpenAIEmbedder(apiKey string) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		apiKey:   apiKey,
		model:    "text-embedding-3-small",
		dims:     1536,
		endpoint: "https://api.openai.com/v1/embeddings",
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// NewOpenAIEmbedderWithEndpoint creates an embedder with a custom endpoint
// for use with OpenAI-compatible APIs (e.g. local models, Azure, etc.).
func NewOpenAIEmbedderWithEndpoint(apiKey, endpoint, model string, dims int) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		apiKey:   apiKey,
		model:    model,
		dims:     dims,
		endpoint: endpoint,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Model returns the model identifier.
func (e *OpenAIEmbedder) Model() string { return e.model }

// Dims returns the embedding dimensionality.
func (e *OpenAIEmbedder) Dims() int { return e.dims }

// Embed calls the OpenAI-compatible embeddings API and returns one vector
// per input text. Texts are batched in a single API call.
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	payload := embeddingRequest{
		Input: texts,
		Model: e.model,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API %d: %s", resp.StatusCode, respBody)
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}

	// The API returns embeddings in the same order as the input.
	// Ensure we have the right count.
	if len(result.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(result.Data))
	}

	embeddings := make([][]float64, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}

	return embeddings, nil
}

// embeddingRequest is the payload for the OpenAI embeddings API.
type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// embeddingResponse is the response from the OpenAI embeddings API.
type embeddingResponse struct {
	Data  []embeddingData `json:"data"`
	Model string          `json:"model"`
	Usage embeddingUsage  `json:"usage"`
}

type embeddingData struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type embeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
