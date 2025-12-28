package webhooks

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("webhook not found")
	ErrWebhookExists  = errors.New("webhook already exists")
	ErrDeliveryNotFound = errors.New("delivery not found")
)

// Webhook represents a webhook
type Webhook struct {
	ID        int64          `json:"id"`
	NodeID    string         `json:"node_id"`
	URL       string         `json:"url"`
	TestURL   string         `json:"test_url"`
	PingURL   string         `json:"ping_url"`
	Name      string         `json:"name"`
	Events    []string       `json:"events"`
	Config    *Config        `json:"config"`
	Active    bool           `json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	// Internal
	OwnerID   int64  `json:"-"`
	OwnerType string `json:"-"` // repo, org
}

// Config represents webhook configuration
type Config struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"` // json, form
	Secret      string `json:"secret,omitempty"`
	InsecureSSL string `json:"insecure_ssl"` // 0, 1
}

// Delivery represents a webhook delivery
type Delivery struct {
	ID             int64     `json:"id"`
	GUID           string    `json:"guid"`
	DeliveredAt    time.Time `json:"delivered_at"`
	Redelivery     bool      `json:"redelivery"`
	Duration       float64   `json:"duration"`
	Status         string    `json:"status"`
	StatusCode     int       `json:"status_code"`
	Event          string    `json:"event"`
	Action         string    `json:"action,omitempty"`
	InstallationID int64     `json:"installation_id,omitempty"`
	RepositoryID   int64     `json:"repository_id,omitempty"`
	URL            string    `json:"url"`
	// Full delivery details
	Request  *DeliveryRequest  `json:"request,omitempty"`
	Response *DeliveryResponse `json:"response,omitempty"`
}

// DeliveryRequest represents the request of a delivery
type DeliveryRequest struct {
	Headers map[string]string `json:"headers"`
	Payload interface{}       `json:"payload"`
}

// DeliveryResponse represents the response of a delivery
type DeliveryResponse struct {
	Headers map[string]string `json:"headers"`
	Payload string            `json:"payload"`
}

// CreateIn represents input for creating a webhook
type CreateIn struct {
	Name   string   `json:"name,omitempty"` // should be "web"
	Config *Config  `json:"config"`
	Events []string `json:"events,omitempty"`
	Active *bool    `json:"active,omitempty"`
}

// UpdateIn represents input for updating a webhook
type UpdateIn struct {
	Config        *Config  `json:"config,omitempty"`
	Events        []string `json:"events,omitempty"`
	AddEvents     []string `json:"add_events,omitempty"`
	RemoveEvents  []string `json:"remove_events,omitempty"`
	Active        *bool    `json:"active,omitempty"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
}

// API defines the webhooks service interface
type API interface {
	// Repository webhooks
	ListForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Webhook, error)
	GetForRepo(ctx context.Context, owner, repo string, hookID int64) (*Webhook, error)
	CreateForRepo(ctx context.Context, owner, repo string, in *CreateIn) (*Webhook, error)
	UpdateForRepo(ctx context.Context, owner, repo string, hookID int64, in *UpdateIn) (*Webhook, error)
	DeleteForRepo(ctx context.Context, owner, repo string, hookID int64) error
	PingRepo(ctx context.Context, owner, repo string, hookID int64) error
	TestRepo(ctx context.Context, owner, repo string, hookID int64) error

	// Repository webhook deliveries
	ListDeliveriesForRepo(ctx context.Context, owner, repo string, hookID int64, opts *ListOpts) ([]*Delivery, error)
	GetDeliveryForRepo(ctx context.Context, owner, repo string, hookID int64, deliveryID int64) (*Delivery, error)
	RedeliverForRepo(ctx context.Context, owner, repo string, hookID int64, deliveryID int64) (*Delivery, error)

	// Organization webhooks
	ListForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Webhook, error)
	GetForOrg(ctx context.Context, org string, hookID int64) (*Webhook, error)
	CreateForOrg(ctx context.Context, org string, in *CreateIn) (*Webhook, error)
	UpdateForOrg(ctx context.Context, org string, hookID int64, in *UpdateIn) (*Webhook, error)
	DeleteForOrg(ctx context.Context, org string, hookID int64) error
	PingOrg(ctx context.Context, org string, hookID int64) error

	// Organization webhook deliveries
	ListDeliveriesForOrg(ctx context.Context, org string, hookID int64, opts *ListOpts) ([]*Delivery, error)
	GetDeliveryForOrg(ctx context.Context, org string, hookID int64, deliveryID int64) (*Delivery, error)
	RedeliverForOrg(ctx context.Context, org string, hookID int64, deliveryID int64) (*Delivery, error)

	// Dispatch a webhook event (internal use)
	Dispatch(ctx context.Context, hookID int64, event string, payload interface{}) (*Delivery, error)
}

// Store defines the data access interface for webhooks
type Store interface {
	Create(ctx context.Context, w *Webhook) error
	GetByID(ctx context.Context, id int64) (*Webhook, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	ListByOwner(ctx context.Context, ownerID int64, ownerType string, opts *ListOpts) ([]*Webhook, error)

	// Deliveries
	CreateDelivery(ctx context.Context, d *Delivery) error
	GetDeliveryByID(ctx context.Context, id int64) (*Delivery, error)
	ListDeliveries(ctx context.Context, hookID int64, opts *ListOpts) ([]*Delivery, error)
}
