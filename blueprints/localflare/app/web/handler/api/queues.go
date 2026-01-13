package api

import (
	"encoding/json"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Queues handles Queue requests.
type Queues struct {
	store store.QueueStore
}

// NewQueues creates a new Queues handler.
func NewQueues(store store.QueueStore) *Queues {
	return &Queues{store: store}
}

// List lists all queues.
func (h *Queues) List(c *mizu.Ctx) error {
	queues, err := h.store.ListQueues(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  queues,
	})
}

// CreateQueueInput is the input for creating a queue.
type CreateQueueInput struct {
	Name     string              `json:"queue_name"`
	Settings store.QueueSettings `json:"settings"`
}

// Create creates a new queue.
func (h *Queues) Create(c *mizu.Ctx) error {
	var input CreateQueueInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "queue_name is required"})
	}

	// Set defaults
	if input.Settings.MaxRetries == 0 {
		input.Settings.MaxRetries = 3
	}
	if input.Settings.MaxBatchSize == 0 {
		input.Settings.MaxBatchSize = 10
	}
	if input.Settings.MaxBatchTimeout == 0 {
		input.Settings.MaxBatchTimeout = 30
	}
	if input.Settings.MessageTTL == 0 {
		input.Settings.MessageTTL = 86400 * 4 // 4 days default
	}

	queue := &store.Queue{
		ID:        ulid.Make().String(),
		Name:      input.Name,
		Settings:  input.Settings,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateQueue(c.Request().Context(), queue); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  queue,
	})
}

// Get retrieves a queue by ID.
func (h *Queues) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	queue, err := h.store.GetQueue(c.Request().Context(), id)
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
	if err := h.store.DeleteQueue(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"queue_id": id},
	})
}

// SendMessageInput is the input for sending a message.
type SendMessageInput struct {
	Body         any    `json:"body"`
	ContentType  string `json:"content_type"`
	DelaySeconds int    `json:"delay_seconds"`
}

// SendMessage sends a message to a queue.
func (h *Queues) SendMessage(c *mizu.Ctx) error {
	queueID := c.Param("id")

	var input SendMessageInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	// Get queue for settings
	queue, err := h.store.GetQueue(c.Request().Context(), queueID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Queue not found"})
	}

	now := time.Now()
	visibleAt := now
	if input.DelaySeconds > 0 {
		visibleAt = now.Add(time.Duration(input.DelaySeconds) * time.Second)
	}

	contentType := input.ContentType
	if contentType == "" {
		contentType = "json"
	}

	// Serialize body
	bodyBytes, _ := serializeBody(input.Body)

	msg := &store.QueueMessage{
		ID:          ulid.Make().String(),
		QueueID:     queueID,
		Body:        bodyBytes,
		ContentType: contentType,
		Attempts:    0,
		CreatedAt:   now,
		VisibleAt:   visibleAt,
		ExpiresAt:   now.Add(time.Duration(queue.Settings.MessageTTL) * time.Second),
	}

	if err := h.store.SendMessage(c.Request().Context(), queueID, msg); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]string{
			"message_id": msg.ID,
		},
	})
}

// PullMessages pulls messages from a queue (HTTP pull consumer).
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

	if input.BatchSize == 0 {
		input.BatchSize = 10
	}
	if input.VisibilityTimeout == 0 {
		input.VisibilityTimeout = 30000
	}

	// Convert ms to seconds
	visTimeout := input.VisibilityTimeout / 1000
	if visTimeout < 1 {
		visTimeout = 30
	}

	msgs, err := h.store.PullMessages(c.Request().Context(), queueID, input.BatchSize, visTimeout)
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

// AckMessagesInput is the input for acknowledging messages.
type AckMessagesInput struct {
	Acks []struct {
		Lease struct {
			ID string `json:"id"`
		} `json:"lease"`
	} `json:"acks"`
}

// AckMessages acknowledges messages.
func (h *Queues) AckMessages(c *mizu.Ctx) error {
	queueID := c.Param("id")

	var input AckMessagesInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	var msgIDs []string
	for _, ack := range input.Acks {
		msgIDs = append(msgIDs, ack.Lease.ID)
	}

	if err := h.store.AckBatch(c.Request().Context(), queueID, msgIDs); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success":  true,
		"ack_count": len(msgIDs),
	})
}

// GetStats returns queue statistics.
func (h *Queues) GetStats(c *mizu.Ctx) error {
	queueID := c.Param("id")
	stats, err := h.store.GetQueueStats(c.Request().Context(), queueID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  stats,
	})
}

func serializeBody(body any) ([]byte, error) {
	switch v := body.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		// JSON encode
		return jsonMarshal(v), nil
	}
}

func jsonMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
