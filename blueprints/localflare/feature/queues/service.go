package queues

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Queues API.
type Service struct {
	store Store
}

// NewService creates a new Queues service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new queue.
func (s *Service) Create(ctx context.Context, in *CreateQueueIn) (*Queue, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}

	// Apply defaults
	settings := in.Settings
	if settings.MessageTTL == 0 {
		settings.MessageTTL = 86400 // 24 hours
	}
	if settings.MaxRetries == 0 {
		settings.MaxRetries = 3
	}
	if settings.MaxBatchSize == 0 {
		settings.MaxBatchSize = 10
	}
	if settings.MaxBatchTimeout == 0 {
		settings.MaxBatchTimeout = 30
	}

	queue := &Queue{
		ID:        ulid.Make().String(),
		Name:      in.Name,
		Settings:  settings,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateQueue(ctx, queue); err != nil {
		return nil, err
	}

	return queue, nil
}

// Get retrieves a queue by ID.
func (s *Service) Get(ctx context.Context, id string) (*Queue, error) {
	queue, err := s.store.GetQueue(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return queue, nil
}

// List lists all queues.
func (s *Service) List(ctx context.Context) ([]*Queue, error) {
	return s.store.ListQueues(ctx)
}

// Delete deletes a queue.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteQueue(ctx, id)
}

// SendMessage sends a message to a queue.
func (s *Service) SendMessage(ctx context.Context, queueID string, in *SendMessageIn) (*Message, error) {
	queue, err := s.store.GetQueue(ctx, queueID)
	if err != nil {
		return nil, ErrNotFound
	}

	now := time.Now()
	visibleAt := now
	if in.DelaySeconds > 0 {
		visibleAt = now.Add(time.Duration(in.DelaySeconds) * time.Second)
	} else if queue.Settings.DeliveryDelay > 0 {
		visibleAt = now.Add(time.Duration(queue.Settings.DeliveryDelay) * time.Second)
	}

	contentType := in.ContentType
	if contentType == "" {
		contentType = "json"
	}

	msg := &Message{
		ID:          ulid.Make().String(),
		QueueID:     queueID,
		Body:        in.Body,
		ContentType: contentType,
		Attempts:    0,
		CreatedAt:   now,
		VisibleAt:   visibleAt,
		ExpiresAt:   now.Add(time.Duration(queue.Settings.MessageTTL) * time.Second),
	}

	if err := s.store.SendMessage(ctx, queueID, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// PullMessages pulls messages from a queue.
func (s *Service) PullMessages(ctx context.Context, queueID string, in *PullMessagesIn) ([]*Message, error) {
	batchSize := in.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	if batchSize > 100 {
		batchSize = 100
	}

	visibilityTimeout := in.VisibilityTimeout
	if visibilityTimeout <= 0 {
		visibilityTimeout = 30
	}

	return s.store.PullMessages(ctx, queueID, batchSize, visibilityTimeout)
}

// AckMessages acknowledges messages.
func (s *Service) AckMessages(ctx context.Context, queueID string, in *AckMessagesIn) error {
	if len(in.MessageIDs) == 0 {
		return nil
	}
	return s.store.AckBatch(ctx, queueID, in.MessageIDs)
}

// GetStats gets queue statistics.
func (s *Service) GetStats(ctx context.Context, queueID string) (*Stats, error) {
	return s.store.GetQueueStats(ctx, queueID)
}
