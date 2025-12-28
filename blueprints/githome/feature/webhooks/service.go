package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// Service implements the webhooks API
type Service struct {
	store     Store
	repoStore repos.Store
	orgStore  orgs.Store
	baseURL   string
	client    *http.Client
}

// NewService creates a new webhooks service
func NewService(store Store, repoStore repos.Store, orgStore orgs.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		orgStore:  orgStore,
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// ListForRepo returns webhooks for a repository
func (s *Service) ListForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Webhook, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	hooks, err := s.store.ListByOwner(ctx, r.ID, "repo", opts)
	if err != nil {
		return nil, err
	}

	for _, h := range hooks {
		s.populateRepoURLs(h, owner, repo)
	}
	return hooks, nil
}

// GetForRepo retrieves a webhook by ID for a repo
func (s *Service) GetForRepo(ctx context.Context, owner, repo string, hookID int64) (*Webhook, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil || h.OwnerID != r.ID || h.OwnerType != "repo" {
		return nil, ErrNotFound
	}

	s.populateRepoURLs(h, owner, repo)
	return h, nil
}

// CreateForRepo creates a webhook for a repo
func (s *Service) CreateForRepo(ctx context.Context, owner, repo string, in *CreateIn) (*Webhook, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	events := in.Events
	if len(events) == 0 {
		events = []string{"push"}
	}

	active := true
	if in.Active != nil {
		active = *in.Active
	}

	now := time.Now()
	h := &Webhook{
		Name:      "web",
		Events:    events,
		Config:    in.Config,
		Active:    active,
		CreatedAt: now,
		UpdatedAt: now,
		OwnerID:   r.ID,
		OwnerType: "repo",
	}

	if err := s.store.Create(ctx, h); err != nil {
		return nil, err
	}

	s.populateRepoURLs(h, owner, repo)
	return h, nil
}

// UpdateForRepo updates a webhook for a repo
func (s *Service) UpdateForRepo(ctx context.Context, owner, repo string, hookID int64, in *UpdateIn) (*Webhook, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil || h.OwnerID != r.ID || h.OwnerType != "repo" {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, hookID, in); err != nil {
		return nil, err
	}

	return s.GetForRepo(ctx, owner, repo, hookID)
}

// DeleteForRepo deletes a webhook for a repo
func (s *Service) DeleteForRepo(ctx context.Context, owner, repo string, hookID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return err
	}
	if h == nil || h.OwnerID != r.ID || h.OwnerType != "repo" {
		return ErrNotFound
	}

	return s.store.Delete(ctx, hookID)
}

// PingRepo sends a ping event to a repo webhook
func (s *Service) PingRepo(ctx context.Context, owner, repo string, hookID int64) error {
	h, err := s.GetForRepo(ctx, owner, repo, hookID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"zen":     "Keep it logically awesome.",
		"hook_id": h.ID,
		"hook":    h,
	}

	_, err = s.Dispatch(ctx, hookID, "ping", payload)
	return err
}

// TestRepo sends a test event to a repo webhook
func (s *Service) TestRepo(ctx context.Context, owner, repo string, hookID int64) error {
	return s.PingRepo(ctx, owner, repo, hookID)
}

// ListDeliveriesForRepo returns deliveries for a repo webhook
func (s *Service) ListDeliveriesForRepo(ctx context.Context, owner, repo string, hookID int64, opts *ListOpts) ([]*Delivery, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil || h.OwnerID != r.ID || h.OwnerType != "repo" {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListDeliveries(ctx, hookID, opts)
}

// GetDeliveryForRepo retrieves a delivery for a repo webhook
func (s *Service) GetDeliveryForRepo(ctx context.Context, owner, repo string, hookID int64, deliveryID int64) (*Delivery, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil || h.OwnerID != r.ID || h.OwnerType != "repo" {
		return nil, ErrNotFound
	}

	d, err := s.store.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, ErrDeliveryNotFound
	}

	return d, nil
}

// RedeliverForRepo redelivers a webhook for a repo
func (s *Service) RedeliverForRepo(ctx context.Context, owner, repo string, hookID int64, deliveryID int64) (*Delivery, error) {
	d, err := s.GetDeliveryForRepo(ctx, owner, repo, hookID, deliveryID)
	if err != nil {
		return nil, err
	}

	// Re-dispatch with same payload
	if d.Request != nil && d.Request.Payload != nil {
		return s.Dispatch(ctx, hookID, d.Event, d.Request.Payload)
	}

	return nil, ErrDeliveryNotFound
}

// ListForOrg returns webhooks for an organization
func (s *Service) ListForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Webhook, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	hooks, err := s.store.ListByOwner(ctx, o.ID, "org", opts)
	if err != nil {
		return nil, err
	}

	for _, h := range hooks {
		s.populateOrgURLs(h, org)
	}
	return hooks, nil
}

// GetForOrg retrieves a webhook by ID for an org
func (s *Service) GetForOrg(ctx context.Context, org string, hookID int64) (*Webhook, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil || h.OwnerID != o.ID || h.OwnerType != "org" {
		return nil, ErrNotFound
	}

	s.populateOrgURLs(h, org)
	return h, nil
}

// CreateForOrg creates a webhook for an org
func (s *Service) CreateForOrg(ctx context.Context, org string, in *CreateIn) (*Webhook, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	events := in.Events
	if len(events) == 0 {
		events = []string{"push"}
	}

	active := true
	if in.Active != nil {
		active = *in.Active
	}

	now := time.Now()
	h := &Webhook{
		Name:      "web",
		Events:    events,
		Config:    in.Config,
		Active:    active,
		CreatedAt: now,
		UpdatedAt: now,
		OwnerID:   o.ID,
		OwnerType: "org",
	}

	if err := s.store.Create(ctx, h); err != nil {
		return nil, err
	}

	s.populateOrgURLs(h, org)
	return h, nil
}

// UpdateForOrg updates a webhook for an org
func (s *Service) UpdateForOrg(ctx context.Context, org string, hookID int64, in *UpdateIn) (*Webhook, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil || h.OwnerID != o.ID || h.OwnerType != "org" {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, hookID, in); err != nil {
		return nil, err
	}

	return s.GetForOrg(ctx, org, hookID)
}

// DeleteForOrg deletes a webhook for an org
func (s *Service) DeleteForOrg(ctx context.Context, org string, hookID int64) error {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return err
	}
	if o == nil {
		return orgs.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return err
	}
	if h == nil || h.OwnerID != o.ID || h.OwnerType != "org" {
		return ErrNotFound
	}

	return s.store.Delete(ctx, hookID)
}

// PingOrg sends a ping event to an org webhook
func (s *Service) PingOrg(ctx context.Context, org string, hookID int64) error {
	h, err := s.GetForOrg(ctx, org, hookID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"zen":     "Keep it logically awesome.",
		"hook_id": h.ID,
		"hook":    h,
	}

	_, err = s.Dispatch(ctx, hookID, "ping", payload)
	return err
}

// ListDeliveriesForOrg returns deliveries for an org webhook
func (s *Service) ListDeliveriesForOrg(ctx context.Context, org string, hookID int64, opts *ListOpts) ([]*Delivery, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil || h.OwnerID != o.ID || h.OwnerType != "org" {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListDeliveries(ctx, hookID, opts)
}

// GetDeliveryForOrg retrieves a delivery for an org webhook
func (s *Service) GetDeliveryForOrg(ctx context.Context, org string, hookID int64, deliveryID int64) (*Delivery, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil || h.OwnerID != o.ID || h.OwnerType != "org" {
		return nil, ErrNotFound
	}

	return s.store.GetDeliveryByID(ctx, deliveryID)
}

// RedeliverForOrg redelivers a webhook for an org
func (s *Service) RedeliverForOrg(ctx context.Context, org string, hookID int64, deliveryID int64) (*Delivery, error) {
	d, err := s.GetDeliveryForOrg(ctx, org, hookID, deliveryID)
	if err != nil {
		return nil, err
	}

	if d.Request != nil && d.Request.Payload != nil {
		return s.Dispatch(ctx, hookID, d.Event, d.Request.Payload)
	}

	return nil, ErrDeliveryNotFound
}

// Dispatch dispatches a webhook event
func (s *Service) Dispatch(ctx context.Context, hookID int64, event string, payload interface{}) (*Delivery, error) {
	h, err := s.store.GetByID(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, ErrNotFound
	}

	if !h.Active {
		return nil, nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.Config.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-GitHub-Delivery", fmt.Sprintf("%d", time.Now().UnixNano()))

	// Sign payload if secret is set
	if h.Config.Secret != "" {
		mac := hmac.New(sha256.New, []byte(h.Config.Secret))
		mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Hub-Signature-256", "sha256="+signature)
	}

	start := time.Now()
	resp, err := s.client.Do(req)
	duration := time.Since(start).Seconds()

	delivery := &Delivery{
		GUID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		DeliveredAt: time.Now(),
		Duration:    duration,
		Event:       event,
		Request: &DeliveryRequest{
			Headers: map[string]string{
				"Content-Type":      "application/json",
				"X-GitHub-Event":    event,
				"X-GitHub-Delivery": req.Header.Get("X-GitHub-Delivery"),
			},
			Payload: payload,
		},
	}

	if err != nil {
		delivery.Status = "failed"
		delivery.StatusCode = 0
	} else {
		defer resp.Body.Close()
		delivery.StatusCode = resp.StatusCode
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			delivery.Status = "delivered"
		} else {
			delivery.Status = "failed"
		}
	}

	if err := s.store.CreateDelivery(ctx, delivery); err != nil {
		return nil, err
	}

	return delivery, nil
}

// populateRepoURLs fills in URL fields for a repo webhook
func (s *Service) populateRepoURLs(h *Webhook, owner, repo string) {
	h.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/hooks/%d", s.baseURL, owner, repo, h.ID)
	h.TestURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/hooks/%d/tests", s.baseURL, owner, repo, h.ID)
	h.PingURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/hooks/%d/pings", s.baseURL, owner, repo, h.ID)
}

// populateOrgURLs fills in URL fields for an org webhook
func (s *Service) populateOrgURLs(h *Webhook, org string) {
	h.URL = fmt.Sprintf("%s/api/v3/orgs/%s/hooks/%d", s.baseURL, org, h.ID)
	h.TestURL = fmt.Sprintf("%s/api/v3/orgs/%s/hooks/%d/tests", s.baseURL, org, h.ID)
	h.PingURL = fmt.Sprintf("%s/api/v3/orgs/%s/hooks/%d/pings", s.baseURL, org, h.ID)
}
