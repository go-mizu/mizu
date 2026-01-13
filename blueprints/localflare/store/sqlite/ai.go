package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

// AIStoreImpl implements store.AIStore using Ollama as the backend.
type AIStoreImpl struct {
	db         *sql.DB
	ollamaURL  string
	httpClient *http.Client
}

// NewAIStore creates a new AI store.
func NewAIStore(db *sql.DB) *AIStoreImpl {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	return &AIStoreImpl{
		db:         db,
		ollamaURL:  ollamaURL,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Default models that map to Ollama models
var defaultModels = []*store.AIModel{
	{
		ID:          "@cf/meta/llama-3.2-3b-instruct",
		Name:        "llama-3.2-3b-instruct",
		Description: "Meta's Llama 3.2 3B instruction-tuned model",
		Task:        "text-generation",
		Properties:  map[string]interface{}{"max_tokens": 4096},
	},
	{
		ID:          "@cf/meta/llama-3.3-70b-instruct-fp8-fast",
		Name:        "llama-3.3-70b-instruct",
		Description: "Meta's Llama 3.3 70B with 2-4x speed boost",
		Task:        "text-generation",
		Properties:  map[string]interface{}{"max_tokens": 8192},
	},
	{
		ID:          "@cf/mistral/mistral-7b-instruct-v0.2",
		Name:        "mistral-7b-instruct",
		Description: "Mistral 7B instruction-tuned model",
		Task:        "text-generation",
		Properties:  map[string]interface{}{"max_tokens": 4096},
	},
	{
		ID:          "@cf/baai/bge-base-en-v1.5",
		Name:        "bge-base-en-v1.5",
		Description: "BGE Base English embedding model",
		Task:        "text-embeddings",
		Properties:  map[string]interface{}{"dimensions": 768},
	},
	{
		ID:          "@cf/baai/bge-large-en-v1.5",
		Name:        "bge-large-en-v1.5",
		Description: "BGE Large English embedding model",
		Task:        "text-embeddings",
		Properties:  map[string]interface{}{"dimensions": 1024},
	},
}

// modelMapping maps Cloudflare model names to Ollama models
var modelMapping = map[string]string{
	"@cf/meta/llama-3.2-3b-instruct":          "llama3.2",
	"@cf/meta/llama-3.3-70b-instruct-fp8-fast": "llama3.3:70b",
	"@cf/mistral/mistral-7b-instruct-v0.2":     "mistral",
	"@cf/baai/bge-base-en-v1.5":                "nomic-embed-text",
	"@cf/baai/bge-large-en-v1.5":               "nomic-embed-text",
}

func (s *AIStoreImpl) ListModels(ctx context.Context, task string) ([]*store.AIModel, error) {
	if task == "" {
		return defaultModels, nil
	}
	var filtered []*store.AIModel
	for _, m := range defaultModels {
		if m.Task == task {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

func (s *AIStoreImpl) GetModel(ctx context.Context, name string) (*store.AIModel, error) {
	for _, m := range defaultModels {
		if m.ID == name || m.Name == name {
			return m, nil
		}
	}
	return nil, fmt.Errorf("model not found: %s", name)
}

func (s *AIStoreImpl) Run(ctx context.Context, req *store.AIInferenceRequest) (*store.AIInferenceResponse, error) {
	// Determine the task type from the model
	model, err := s.GetModel(ctx, req.Model)
	if err != nil {
		// Try to run with Ollama directly
		return s.runWithOllama(ctx, req)
	}

	switch model.Task {
	case "text-generation":
		return s.runTextGeneration(ctx, req)
	case "text-embeddings":
		return s.runTextEmbeddings(ctx, req)
	default:
		return s.runWithOllama(ctx, req)
	}
}

func (s *AIStoreImpl) runTextGeneration(ctx context.Context, req *store.AIInferenceRequest) (*store.AIInferenceResponse, error) {
	// Get the Ollama model name
	ollamaModel := s.getOllamaModel(req.Model)

	// Build the prompt
	var prompt string
	if p, ok := req.Inputs["prompt"].(string); ok {
		prompt = p
	} else if messages, ok := req.Inputs["messages"].([]interface{}); ok {
		// Convert messages to prompt format
		for _, m := range messages {
			if msg, ok := m.(map[string]interface{}); ok {
				role := msg["role"].(string)
				content := msg["content"].(string)
				prompt += fmt.Sprintf("%s: %s\n", role, content)
			}
		}
	}

	// Call Ollama
	reqBody := map[string]interface{}{
		"model":  ollamaModel,
		"prompt": prompt,
		"stream": false,
	}

	if opts := req.Options; opts != nil {
		if maxTokens, ok := opts["max_tokens"].(float64); ok {
			reqBody["options"] = map[string]interface{}{"num_predict": int(maxTokens)}
		}
	}

	body, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.ollamaURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var ollamaResp struct {
		Response string `json:"response"`
		Context  []int  `json:"context"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	return &store.AIInferenceResponse{
		Result: map[string]interface{}{
			"response": ollamaResp.Response,
		},
	}, nil
}

func (s *AIStoreImpl) runTextEmbeddings(ctx context.Context, req *store.AIInferenceRequest) (*store.AIInferenceResponse, error) {
	ollamaModel := s.getOllamaModel(req.Model)

	// Get input text
	var texts []string
	if text, ok := req.Inputs["text"].(string); ok {
		texts = []string{text}
	} else if textArr, ok := req.Inputs["text"].([]interface{}); ok {
		for _, t := range textArr {
			if ts, ok := t.(string); ok {
				texts = append(texts, ts)
			}
		}
	}

	var embeddings [][]float32
	for _, text := range texts {
		reqBody := map[string]interface{}{
			"model":  ollamaModel,
			"prompt": text,
		}
		body, _ := json.Marshal(reqBody)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", s.ollamaURL+"/api/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("ollama embeddings request failed: %w", err)
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var ollamaResp struct {
			Embedding []float32 `json:"embedding"`
		}
		if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
			return nil, fmt.Errorf("failed to parse ollama response: %w", err)
		}
		embeddings = append(embeddings, ollamaResp.Embedding)
	}

	return &store.AIInferenceResponse{
		Result: map[string]interface{}{
			"data": embeddings,
		},
	}, nil
}

func (s *AIStoreImpl) runWithOllama(ctx context.Context, req *store.AIInferenceRequest) (*store.AIInferenceResponse, error) {
	return s.runTextGeneration(ctx, req)
}

func (s *AIStoreImpl) getOllamaModel(cfModel string) string {
	if mapped, ok := modelMapping[cfModel]; ok {
		return mapped
	}
	// Try to use the model name directly
	return cfModel
}

func (s *AIStoreImpl) GenerateEmbeddings(ctx context.Context, model string, texts []string) ([][]float32, error) {
	resp, err := s.Run(ctx, &store.AIInferenceRequest{
		Model:  model,
		Inputs: map[string]interface{}{"text": texts},
	})
	if err != nil {
		return nil, err
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	data, ok := result["data"].([][]float32)
	if !ok {
		// Try to convert from interface slice
		if dataIface, ok := result["data"].([]interface{}); ok {
			data = make([][]float32, len(dataIface))
			for i, d := range dataIface {
				if floats, ok := d.([]float32); ok {
					data[i] = floats
				}
			}
		}
	}
	return data, nil
}

func (s *AIStoreImpl) GenerateText(ctx context.Context, model string, prompt string, opts map[string]interface{}) (string, error) {
	resp, err := s.Run(ctx, &store.AIInferenceRequest{
		Model:   model,
		Inputs:  map[string]interface{}{"prompt": prompt},
		Options: opts,
	})
	if err != nil {
		return "", err
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	if text, ok := result["response"].(string); ok {
		return text, nil
	}
	return "", fmt.Errorf("no text in response")
}

func (s *AIStoreImpl) StreamText(ctx context.Context, model string, prompt string, opts map[string]interface{}) (<-chan string, error) {
	ollamaModel := s.getOllamaModel(model)

	reqBody := map[string]interface{}{
		"model":  ollamaModel,
		"prompt": prompt,
		"stream": true,
	}
	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.ollamaURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk struct {
				Response string `json:"response"`
				Done     bool   `json:"done"`
			}
			if err := decoder.Decode(&chunk); err != nil {
				return
			}
			if chunk.Response != "" {
				select {
				case ch <- chunk.Response:
				case <-ctx.Done():
					return
				}
			}
			if chunk.Done {
				return
			}
		}
	}()

	return ch, nil
}
