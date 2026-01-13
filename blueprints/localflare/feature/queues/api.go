// Package queues provides message queue management.
package queues

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("queue not found")
	ErrNameRequired = errors.New("name is required")
)

// Queue represents a message queue.
type Queue struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Settings  QueueSettings `json:"settings"`
	CreatedAt time.Time     `json:"created_at"`
}

// QueueSettings configures queue behavior.
type QueueSettings struct {
	DeliveryDelay   int `json:"delivery_delay"`
	MessageTTL      int `json:"message_ttl"`
	MaxRetries      int `json:"max_retries"`
	MaxBatchSize    int `json:"max_batch_size"`
	MaxBatchTimeout int `json:"max_batch_timeout"`
}

// Message represents a queue message.
type Message struct {
	ID          string    `json:"id"`
	QueueID     string    `json:"queue_id"`
	Body        []byte    `json:"body"`
	ContentType string    `json:"content_type"`
	Attempts    int       `json:"attempts"`
	CreatedAt   time.Time `json:"created_at"`
	VisibleAt   time.Time `json:"visible_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Stats contains queue statistics.
type Stats struct {
	Messages        int64 `json:"messages"`
	MessagesReady   int64 `json:"messages_ready"`
	MessagesDelayed int64 `json:"messages_delayed"`
}

// CreateQueueIn contains input for creating a queue.
type CreateQueueIn struct {
	Name     string        `json:"name"`
	Settings QueueSettings `json:"settings"`
}

// SendMessageIn contains input for sending a message.
type SendMessageIn struct {
	Body        []byte `json:"body"`
	ContentType string `json:"content_type"`
	DelaySeconds int   `json:"delay_seconds,omitempty"`
}

// PullMessagesIn contains input for pulling messages.
type PullMessagesIn struct {
	BatchSize         int `json:"batch_size"`
	VisibilityTimeout int `json:"visibility_timeout"`
}

// AckMessagesIn contains input for acknowledging messages.
type AckMessagesIn struct {
	MessageIDs []string `json:"acks"`
}

// API defines the Queues service contract.
type API interface {
	Create(ctx context.Context, in *CreateQueueIn) (*Queue, error)
	Get(ctx context.Context, id string) (*Queue, error)
	List(ctx context.Context) ([]*Queue, error)
	Delete(ctx context.Context, id string) error
	SendMessage(ctx context.Context, queueID string, in *SendMessageIn) (*Message, error)
	PullMessages(ctx context.Context, queueID string, in *PullMessagesIn) ([]*Message, error)
	AckMessages(ctx context.Context, queueID string, in *AckMessagesIn) error
	GetStats(ctx context.Context, queueID string) (*Stats, error)
}

// Store defines the data access contract.
type Store interface {
	CreateQueue(ctx context.Context, queue *Queue) error
	GetQueue(ctx context.Context, id string) (*Queue, error)
	ListQueues(ctx context.Context) ([]*Queue, error)
	DeleteQueue(ctx context.Context, id string) error
	SendMessage(ctx context.Context, queueID string, msg *Message) error
	PullMessages(ctx context.Context, queueID string, batchSize, visibilityTimeout int) ([]*Message, error)
	AckMessage(ctx context.Context, queueID, msgID string) error
	AckBatch(ctx context.Context, queueID string, msgIDs []string) error
	GetQueueStats(ctx context.Context, queueID string) (*Stats, error)
}
