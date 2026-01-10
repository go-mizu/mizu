// Package webhooks provides webhook management functionality.
package webhooks

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound = errors.New("webhook not found")
)

// Event types
const (
	EventRecordCreated = "record.created"
	EventRecordUpdated = "record.updated"
	EventRecordDeleted = "record.deleted"
	EventFieldCreated  = "field.created"
	EventFieldUpdated  = "field.updated"
	EventFieldDeleted  = "field.deleted"
)

// Webhook represents a webhook subscription.
type Webhook struct {
	ID        string    `json:"id"`
	BaseID    string    `json:"base_id"`
	TableID   string    `json:"table_id,omitempty"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Delivery represents a webhook delivery attempt.
type Delivery struct {
	ID         string    `json:"id"`
	WebhookID  string    `json:"webhook_id"`
	Event      string    `json:"event"`
	Payload    string    `json:"payload"`
	StatusCode int       `json:"status_code"`
	Response   string    `json:"response,omitempty"`
	DurationMs int       `json:"duration_ms"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateIn contains input for creating a webhook.
type CreateIn struct {
	BaseID  string   `json:"base_id"`
	TableID string   `json:"table_id,omitempty"`
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Secret  string   `json:"secret,omitempty"`
}

// UpdateIn contains input for updating a webhook.
type UpdateIn struct {
	URL      *string   `json:"url,omitempty"`
	Events   *[]string `json:"events,omitempty"`
	Secret   *string   `json:"secret,omitempty"`
	IsActive *bool     `json:"is_active,omitempty"`
}

// ListOpts contains options for listing deliveries.
type ListOpts struct {
	Limit  int
	Cursor string
}

// API defines the webhooks service interface.
type API interface {
	Create(ctx context.Context, userID string, in CreateIn) (*Webhook, error)
	GetByID(ctx context.Context, id string) (*Webhook, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Webhook, error)
	Delete(ctx context.Context, id string) error
	ListByBase(ctx context.Context, baseID string) ([]*Webhook, error)

	// Delivery
	Trigger(ctx context.Context, tableID string, event string, payload interface{}) error
	ListDeliveries(ctx context.Context, webhookID string, opts ListOpts) ([]*Delivery, error)
	RetryDelivery(ctx context.Context, deliveryID string) error
}

// Store defines the webhooks data access interface.
type Store interface {
	Create(ctx context.Context, webhook *Webhook) error
	GetByID(ctx context.Context, id string) (*Webhook, error)
	Update(ctx context.Context, webhook *Webhook) error
	Delete(ctx context.Context, id string) error
	ListByBase(ctx context.Context, baseID string) ([]*Webhook, error)
	ListByTable(ctx context.Context, tableID string) ([]*Webhook, error)

	// Deliveries
	CreateDelivery(ctx context.Context, delivery *Delivery) error
	GetDelivery(ctx context.Context, id string) (*Delivery, error)
	ListDeliveries(ctx context.Context, webhookID string, opts ListOpts) ([]*Delivery, error)
}
