package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/store"
)

// AI handles Workers AI requests.
type AI struct {
	store store.AIStore
}

// NewAI creates a new AI handler.
func NewAI(store store.AIStore) *AI {
	return &AI{store: store}
}

// RunModelInput is the input for running inference.
type RunModelInput struct {
	// For text generation
	Prompt   string        `json:"prompt"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`

	// For embeddings
	Text []string `json:"text"`

	// Common options
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// ChatMessage represents a chat message.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RunModel runs inference on a model.
func (h *AI) RunModel(c *mizu.Ctx) error {
	model := c.Param("model")

	var input RunModelInput
	if err := c.BindJSON(&input, 10<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	// Handle embeddings request
	if len(input.Text) > 0 {
		embeddings, err := h.store.GenerateEmbeddings(c.Request().Context(), model, input.Text)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}

		shape := []int{len(embeddings), 0}
		if len(embeddings) > 0 {
			shape[1] = len(embeddings[0])
		}

		return c.JSON(200, map[string]any{
			"success": true,
			"result": map[string]any{
				"shape": shape,
				"data":  embeddings,
			},
		})
	}

	// Handle text generation
	prompt := input.Prompt
	if len(input.Messages) > 0 {
		// Convert messages to prompt
		for _, msg := range input.Messages {
			if msg.Role == "user" {
				prompt += msg.Content + "\n"
			} else if msg.Role == "assistant" {
				prompt += msg.Content + "\n"
			} else if msg.Role == "system" {
				prompt = msg.Content + "\n" + prompt
			}
		}
	}

	opts := map[string]interface{}{
		"max_tokens":  input.MaxTokens,
		"temperature": input.Temperature,
	}

	if input.Stream {
		// Streaming response
		c.Writer().Header().Set("Content-Type", "text/event-stream")
		c.Writer().Header().Set("Cache-Control", "no-cache")
		c.Writer().Header().Set("Connection", "keep-alive")

		ch, err := h.store.StreamText(c.Request().Context(), model, prompt, opts)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}

		flusher, _ := c.Writer().(interface{ Flush() })

		for chunk := range ch {
			c.Writer().Write([]byte("data: " + chunk + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}

		c.Writer().Write([]byte("data: [DONE]\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}

	// Non-streaming response
	result, err := h.store.GenerateText(c.Request().Context(), model, prompt, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"response": result,
		},
	})
}

// ListModels lists available models.
func (h *AI) ListModels(c *mizu.Ctx) error {
	task := c.Query("task")
	models, err := h.store.ListModels(c.Request().Context(), task)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  models,
	})
}

// GetModel retrieves a model by name.
func (h *AI) GetModel(c *mizu.Ctx) error {
	name := c.Param("model")
	model, err := h.store.GetModel(c.Request().Context(), name)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Model not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  model,
	})
}
