package api

import (
	"encoding/json"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/feature/queues"
)

// Queues handles Queue requests.
type Queues struct {
	svc queues.API
}

// NewQueues creates a new Queues handler.
func NewQueues(svc queues.API) *Queues {
	return &Queues{svc: svc}
}

// List lists all queues.
func (h *Queues) List(c *mizu.Ctx) error {
	result, err := h.svc.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"queues": result,
		},
	})
}

// Create creates a new queue.
func (h *Queues) Create(c *mizu.Ctx) error {
	var input struct {
		Name     string                `json:"queue_name"`
		Settings queues.QueueSettings `json:"settings"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	in := &queues.CreateQueueIn{
		Name:     input.Name,
		Settings: input.Settings,
	}

	queue, err := h.svc.Create(c.Request().Context(), in)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  queue,
	})
}

// Get retrieves a queue by ID.
func (h *Queues) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	queue, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Queue not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  queue,
	})
}

// Delete deletes a queue.
func (h *Queues) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"queue_id": id},
	})
}

// SendMessage sends a message to a queue.
func (h *Queues) SendMessage(c *mizu.Ctx) error {
	queueID := c.Param("id")

	var input struct {
		Body         any    `json:"body"`
		ContentType  string `json:"content_type"`
		DelaySeconds int    `json:"delay_seconds"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	// Serialize body
	bodyBytes := serializeBody(input.Body)

	in := &queues.SendMessageIn{
		Body:         bodyBytes,
		ContentType:  input.ContentType,
		DelaySeconds: input.DelaySeconds,
	}

	msg, err := h.svc.SendMessage(c.Request().Context(), queueID, in)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]string{
			"message_id": msg.ID,
		},
	})
}

// PullMessages pulls messages from a queue.
func (h *Queues) PullMessages(c *mizu.Ctx) error {
	queueID := c.Param("id")

	var input struct {
		BatchSize         int `json:"batch_size"`
		VisibilityTimeout int `json:"visibility_timeout_ms"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		input.BatchSize = 10
		input.VisibilityTimeout = 30000
	}

	// Convert ms to seconds
	visTimeout := input.VisibilityTimeout / 1000
	if visTimeout < 1 {
		visTimeout = 30
	}

	in := &queues.PullMessagesIn{
		BatchSize:         input.BatchSize,
		VisibilityTimeout: visTimeout,
	}

	msgs, err := h.svc.PullMessages(c.Request().Context(), queueID, in)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"messages": msgs,
		},
	})
}

// AckMessages acknowledges messages.
func (h *Queues) AckMessages(c *mizu.Ctx) error {
	queueID := c.Param("id")

	var input struct {
		Acks []struct {
			Lease struct {
				ID string `json:"id"`
			} `json:"lease"`
		} `json:"acks"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	var msgIDs []string
	for _, ack := range input.Acks {
		msgIDs = append(msgIDs, ack.Lease.ID)
	}

	in := &queues.AckMessagesIn{MessageIDs: msgIDs}
	if err := h.svc.AckMessages(c.Request().Context(), queueID, in); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success":   true,
		"ack_count": len(msgIDs),
	})
}

// GetStats returns queue statistics.
func (h *Queues) GetStats(c *mizu.Ctx) error {
	queueID := c.Param("id")
	stats, err := h.svc.GetStats(c.Request().Context(), queueID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  stats,
	})
}

func serializeBody(body any) []byte {
	switch v := body.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	default:
		b, _ := json.Marshal(v)
		return b
	}
}
