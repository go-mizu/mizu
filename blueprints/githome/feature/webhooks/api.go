package webhooks

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound         = errors.New("webhook not found")
	ErrInvalidInput     = errors.New("invalid input")
	ErrMissingURL       = errors.New("webhook URL is required")
	ErrDeliveryNotFound = errors.New("delivery not found")
	ErrAccessDenied     = errors.New("access denied")
)

// Event types
const (
	EventPush             = "push"
	EventCreate           = "create"
	EventDelete           = "delete"
	EventFork             = "fork"
	EventIssues           = "issues"
	EventIssueComment     = "issue_comment"
	EventPullRequest      = "pull_request"
	EventPullRequestReview = "pull_request_review"
	EventRelease          = "release"
	EventStar             = "star"
	EventWatch            = "watch"
	EventMember           = "member"
	EventPublic           = "public"
	EventPing             = "ping"
	EventWildcard         = "*"
)

// Content types
const (
	ContentTypeJSON = "json"
	ContentTypeForm = "form"
)

// Webhook represents a webhook configuration
type Webhook struct {
	ID               string     `json:"id"`
	RepoID           string     `json:"repo_id,omitempty"`
	OrgID            string     `json:"org_id,omitempty"`
	URL              string     `json:"url"`
	Secret           string     `json:"-"`
	ContentType      string     `json:"content_type"`
	Events           []string   `json:"events"`
	Active           bool       `json:"active"`
	InsecureSSL      bool       `json:"insecure_ssl"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastResponseCode int        `json:"last_response_code,omitempty"`
	LastResponseAt   *time.Time `json:"last_response_at,omitempty"`
}

// Delivery represents a webhook delivery attempt
type Delivery struct {
	ID              string    `json:"id"`
	WebhookID       string    `json:"webhook_id"`
	Event           string    `json:"event"`
	GUID            string    `json:"guid"`
	Payload         string    `json:"payload"`
	RequestHeaders  string    `json:"request_headers"`
	ResponseHeaders string    `json:"response_headers"`
	ResponseBody    string    `json:"response_body"`
	StatusCode      int       `json:"status_code"`
	Delivered       bool      `json:"delivered"`
	DurationMS      int       `json:"duration_ms"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateIn is the input for creating a webhook
type CreateIn struct {
	RepoID      string   `json:"repo_id,omitempty"`
	OrgID       string   `json:"org_id,omitempty"`
	URL         string   `json:"url"`
	Secret      string   `json:"secret,omitempty"`
	ContentType string   `json:"content_type"`
	Events      []string `json:"events"`
	Active      bool     `json:"active"`
	InsecureSSL bool     `json:"insecure_ssl"`
}

// UpdateIn is the input for updating a webhook
type UpdateIn struct {
	URL         *string   `json:"url,omitempty"`
	Secret      *string   `json:"secret,omitempty"`
	ContentType *string   `json:"content_type,omitempty"`
	Events      *[]string `json:"events,omitempty"`
	Active      *bool     `json:"active,omitempty"`
	InsecureSSL *bool     `json:"insecure_ssl,omitempty"`
}

// ListOpts are options for listing
type ListOpts struct {
	Limit  int
	Offset int
}

// API is the webhooks service interface
type API interface {
	// Webhook CRUD
	Create(ctx context.Context, in *CreateIn) (*Webhook, error)
	GetByID(ctx context.Context, id string) (*Webhook, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Webhook, error)
	Delete(ctx context.Context, id string) error
	ListByRepo(ctx context.Context, repoID string, opts *ListOpts) ([]*Webhook, error)
	ListByOrg(ctx context.Context, orgID string, opts *ListOpts) ([]*Webhook, error)

	// Operations
	Ping(ctx context.Context, id string) (*Delivery, error)
	Test(ctx context.Context, id string, event string) (*Delivery, error)

	// Deliveries
	RecordDelivery(ctx context.Context, d *Delivery) error
	GetDelivery(ctx context.Context, id string) (*Delivery, error)
	ListDeliveries(ctx context.Context, webhookID string, opts *ListOpts) ([]*Delivery, error)
	Redeliver(ctx context.Context, deliveryID string) (*Delivery, error)
}

// Store is the webhooks data store interface
type Store interface {
	// Webhooks
	Create(ctx context.Context, w *Webhook) error
	GetByID(ctx context.Context, id string) (*Webhook, error)
	Update(ctx context.Context, w *Webhook) error
	Delete(ctx context.Context, id string) error
	ListByRepo(ctx context.Context, repoID string, limit, offset int) ([]*Webhook, error)
	ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]*Webhook, error)
	ListByEvent(ctx context.Context, repoID, event string) ([]*Webhook, error)

	// Deliveries
	CreateDelivery(ctx context.Context, d *Delivery) error
	GetDelivery(ctx context.Context, id string) (*Delivery, error)
	UpdateDelivery(ctx context.Context, d *Delivery) error
	ListDeliveries(ctx context.Context, webhookID string, limit, offset int) ([]*Delivery, error)
}
