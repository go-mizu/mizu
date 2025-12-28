package webhooks

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the webhooks API
type Service struct {
	store Store
}

// NewService creates a new webhooks service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new webhook
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Webhook, error) {
	if in.URL == "" {
		return nil, ErrMissingURL
	}

	contentType := in.ContentType
	if contentType == "" {
		contentType = ContentTypeJSON
	}

	events := in.Events
	if len(events) == 0 {
		events = []string{EventPush}
	}

	now := time.Now()
	webhook := &Webhook{
		ID:          ulid.New(),
		RepoID:      in.RepoID,
		OrgID:       in.OrgID,
		URL:         strings.TrimSpace(in.URL),
		Secret:      in.Secret,
		ContentType: contentType,
		Events:      events,
		Active:      in.Active,
		InsecureSSL: in.InsecureSSL,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// GetByID retrieves a webhook by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Webhook, error) {
	webhook, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if webhook == nil {
		return nil, ErrNotFound
	}
	return webhook, nil
}

// Update updates a webhook
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Webhook, error) {
	webhook, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if webhook == nil {
		return nil, ErrNotFound
	}

	if in.URL != nil {
		webhook.URL = strings.TrimSpace(*in.URL)
	}
	if in.Secret != nil {
		webhook.Secret = *in.Secret
	}
	if in.ContentType != nil {
		webhook.ContentType = *in.ContentType
	}
	if in.Events != nil {
		webhook.Events = *in.Events
	}
	if in.Active != nil {
		webhook.Active = *in.Active
	}
	if in.InsecureSSL != nil {
		webhook.InsecureSSL = *in.InsecureSSL
	}

	webhook.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// Delete deletes a webhook
func (s *Service) Delete(ctx context.Context, id string) error {
	webhook, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if webhook == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// ListByRepo lists webhooks for a repository
func (s *Service) ListByRepo(ctx context.Context, repoID string, opts *ListOpts) ([]*Webhook, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByRepo(ctx, repoID, limit, offset)
}

// ListByOrg lists webhooks for an organization
func (s *Service) ListByOrg(ctx context.Context, orgID string, opts *ListOpts) ([]*Webhook, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByOrg(ctx, orgID, limit, offset)
}

// Ping sends a ping event to a webhook
func (s *Service) Ping(ctx context.Context, id string) (*Delivery, error) {
	webhook, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if webhook == nil {
		return nil, ErrNotFound
	}

	// Create a delivery record (actual HTTP call would happen in a separate service)
	now := time.Now()
	delivery := &Delivery{
		ID:        ulid.New(),
		WebhookID: id,
		Event:     EventPing,
		GUID:      ulid.New(),
		Payload:   `{"zen": "Design for failure."}`,
		CreatedAt: now,
	}

	if err := s.store.CreateDelivery(ctx, delivery); err != nil {
		return nil, err
	}

	return delivery, nil
}

// Test sends a test event to a webhook
func (s *Service) Test(ctx context.Context, id string, event string) (*Delivery, error) {
	webhook, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if webhook == nil {
		return nil, ErrNotFound
	}

	if event == "" {
		event = EventPush
	}

	// Create a delivery record
	now := time.Now()
	delivery := &Delivery{
		ID:        ulid.New(),
		WebhookID: id,
		Event:     event,
		GUID:      ulid.New(),
		Payload:   `{"test": true}`,
		CreatedAt: now,
	}

	if err := s.store.CreateDelivery(ctx, delivery); err != nil {
		return nil, err
	}

	return delivery, nil
}

// RecordDelivery records a delivery attempt
func (s *Service) RecordDelivery(ctx context.Context, d *Delivery) error {
	if d.ID == "" {
		d.ID = ulid.New()
	}
	if d.GUID == "" {
		d.GUID = ulid.New()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	return s.store.CreateDelivery(ctx, d)
}

// GetDelivery retrieves a delivery by ID
func (s *Service) GetDelivery(ctx context.Context, id string) (*Delivery, error) {
	delivery, err := s.store.GetDelivery(ctx, id)
	if err != nil {
		return nil, err
	}
	if delivery == nil {
		return nil, ErrDeliveryNotFound
	}
	return delivery, nil
}

// ListDeliveries lists deliveries for a webhook
func (s *Service) ListDeliveries(ctx context.Context, webhookID string, opts *ListOpts) ([]*Delivery, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListDeliveries(ctx, webhookID, limit, offset)
}

// Redeliver redelivers a webhook delivery
func (s *Service) Redeliver(ctx context.Context, deliveryID string) (*Delivery, error) {
	original, err := s.store.GetDelivery(ctx, deliveryID)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, ErrDeliveryNotFound
	}

	// Create a new delivery with the same payload
	now := time.Now()
	delivery := &Delivery{
		ID:        ulid.New(),
		WebhookID: original.WebhookID,
		Event:     original.Event,
		GUID:      ulid.New(),
		Payload:   original.Payload,
		CreatedAt: now,
	}

	if err := s.store.CreateDelivery(ctx, delivery); err != nil {
		return nil, err
	}

	return delivery, nil
}

func (s *Service) getPageParams(opts *ListOpts) (int, int) {
	limit := 30
	offset := 0
	if opts != nil {
		if opts.Limit > 0 && opts.Limit <= 100 {
			limit = opts.Limit
		}
		if opts.Offset >= 0 {
			offset = opts.Offset
		}
	}
	return limit, offset
}
