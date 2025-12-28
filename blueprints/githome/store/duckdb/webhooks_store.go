package duckdb

import (
	"context"
	"database/sql"
	"strings"

	"github.com/go-mizu/blueprints/githome/feature/webhooks"
)

// WebhooksStore implements webhooks.Store
type WebhooksStore struct {
	db *sql.DB
}

// NewWebhooksStore creates a new webhooks store
func NewWebhooksStore(db *sql.DB) *WebhooksStore {
	return &WebhooksStore{db: db}
}

// Create creates a new webhook
func (s *WebhooksStore) Create(ctx context.Context, w *webhooks.Webhook) error {
	events := strings.Join(w.Events, ",")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhooks (id, repo_id, org_id, url, secret, content_type, events, active, insecure_ssl, created_at, updated_at, last_response_code, last_response_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, w.ID, nullString(w.RepoID), nullString(w.OrgID), w.URL, w.Secret, w.ContentType, events, w.Active, w.InsecureSSL, w.CreatedAt, w.UpdatedAt, w.LastResponseCode, w.LastResponseAt)
	return err
}

// GetByID retrieves a webhook by ID
func (s *WebhooksStore) GetByID(ctx context.Context, id string) (*webhooks.Webhook, error) {
	w := &webhooks.Webhook{}
	var repoID, orgID sql.NullString
	var events string
	var lastResponseCode sql.NullInt64
	var lastResponseAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, org_id, url, secret, content_type, events, active, insecure_ssl, created_at, updated_at, last_response_code, last_response_at
		FROM webhooks WHERE id = $1
	`, id).Scan(&w.ID, &repoID, &orgID, &w.URL, &w.Secret, &w.ContentType, &events, &w.Active, &w.InsecureSSL, &w.CreatedAt, &w.UpdatedAt, &lastResponseCode, &lastResponseAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if repoID.Valid {
		w.RepoID = repoID.String
	}
	if orgID.Valid {
		w.OrgID = orgID.String
	}
	if events != "" {
		w.Events = strings.Split(events, ",")
	}
	if lastResponseCode.Valid {
		w.LastResponseCode = int(lastResponseCode.Int64)
	}
	if lastResponseAt.Valid {
		w.LastResponseAt = &lastResponseAt.Time
	}
	return w, nil
}

// Update updates a webhook
func (s *WebhooksStore) Update(ctx context.Context, w *webhooks.Webhook) error {
	events := strings.Join(w.Events, ",")
	_, err := s.db.ExecContext(ctx, `
		UPDATE webhooks SET url = $2, secret = $3, content_type = $4, events = $5, active = $6, insecure_ssl = $7, updated_at = $8, last_response_code = $9, last_response_at = $10
		WHERE id = $1
	`, w.ID, w.URL, w.Secret, w.ContentType, events, w.Active, w.InsecureSSL, w.UpdatedAt, w.LastResponseCode, w.LastResponseAt)
	return err
}

// Delete deletes a webhook
func (s *WebhooksStore) Delete(ctx context.Context, id string) error {
	// Delete deliveries first
	s.db.ExecContext(ctx, `DELETE FROM webhook_deliveries WHERE webhook_id = $1`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	return err
}

// ListByRepo lists webhooks for a repository
func (s *WebhooksStore) ListByRepo(ctx context.Context, repoID string, limit, offset int) ([]*webhooks.Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, org_id, url, secret, content_type, events, active, insecure_ssl, created_at, updated_at, last_response_code, last_response_at
		FROM webhooks WHERE repo_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, repoID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanWebhooks(rows)
}

// ListByOrg lists webhooks for an organization
func (s *WebhooksStore) ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]*webhooks.Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, org_id, url, secret, content_type, events, active, insecure_ssl, created_at, updated_at, last_response_code, last_response_at
		FROM webhooks WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanWebhooks(rows)
}

// ListByEvent lists active webhooks for a repository that subscribe to a specific event
func (s *WebhooksStore) ListByEvent(ctx context.Context, repoID, event string) ([]*webhooks.Webhook, error) {
	// This needs to check if the event is in the events list or if '*' is in the list
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, org_id, url, secret, content_type, events, active, insecure_ssl, created_at, updated_at, last_response_code, last_response_at
		FROM webhooks WHERE repo_id = $1 AND active = TRUE
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allWebhooks, err := s.scanWebhooks(rows)
	if err != nil {
		return nil, err
	}

	// Filter webhooks that subscribe to this event
	var filtered []*webhooks.Webhook
	for _, w := range allWebhooks {
		for _, e := range w.Events {
			if e == event || e == "*" {
				filtered = append(filtered, w)
				break
			}
		}
	}
	return filtered, nil
}

func (s *WebhooksStore) scanWebhooks(rows *sql.Rows) ([]*webhooks.Webhook, error) {
	var list []*webhooks.Webhook
	for rows.Next() {
		w := &webhooks.Webhook{}
		var repoID, orgID sql.NullString
		var events string
		var lastResponseCode sql.NullInt64
		var lastResponseAt sql.NullTime
		if err := rows.Scan(&w.ID, &repoID, &orgID, &w.URL, &w.Secret, &w.ContentType, &events, &w.Active, &w.InsecureSSL, &w.CreatedAt, &w.UpdatedAt, &lastResponseCode, &lastResponseAt); err != nil {
			return nil, err
		}
		if repoID.Valid {
			w.RepoID = repoID.String
		}
		if orgID.Valid {
			w.OrgID = orgID.String
		}
		if events != "" {
			w.Events = strings.Split(events, ",")
		}
		if lastResponseCode.Valid {
			w.LastResponseCode = int(lastResponseCode.Int64)
		}
		if lastResponseAt.Valid {
			w.LastResponseAt = &lastResponseAt.Time
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

// CreateDelivery creates a delivery record
func (s *WebhooksStore) CreateDelivery(ctx context.Context, d *webhooks.Delivery) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (id, webhook_id, event, guid, payload, request_headers, response_headers, response_body, status_code, delivered, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, d.ID, d.WebhookID, d.Event, d.GUID, d.Payload, d.RequestHeaders, d.ResponseHeaders, d.ResponseBody, d.StatusCode, d.Delivered, d.DurationMS, d.CreatedAt)
	return err
}

// GetDelivery retrieves a delivery by ID
func (s *WebhooksStore) GetDelivery(ctx context.Context, id string) (*webhooks.Delivery, error) {
	d := &webhooks.Delivery{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, webhook_id, event, guid, payload, request_headers, response_headers, response_body, status_code, delivered, duration_ms, created_at
		FROM webhook_deliveries WHERE id = $1
	`, id).Scan(&d.ID, &d.WebhookID, &d.Event, &d.GUID, &d.Payload, &d.RequestHeaders, &d.ResponseHeaders, &d.ResponseBody, &d.StatusCode, &d.Delivered, &d.DurationMS, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// UpdateDelivery updates a delivery
func (s *WebhooksStore) UpdateDelivery(ctx context.Context, d *webhooks.Delivery) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE webhook_deliveries SET response_headers = $2, response_body = $3, status_code = $4, delivered = $5, duration_ms = $6
		WHERE id = $1
	`, d.ID, d.ResponseHeaders, d.ResponseBody, d.StatusCode, d.Delivered, d.DurationMS)
	return err
}

// ListDeliveries lists deliveries for a webhook
func (s *WebhooksStore) ListDeliveries(ctx context.Context, webhookID string, limit, offset int) ([]*webhooks.Delivery, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, webhook_id, event, guid, payload, request_headers, response_headers, response_body, status_code, delivered, duration_ms, created_at
		FROM webhook_deliveries WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, webhookID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*webhooks.Delivery
	for rows.Next() {
		d := &webhooks.Delivery{}
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.Event, &d.GUID, &d.Payload, &d.RequestHeaders, &d.ResponseHeaders, &d.ResponseBody, &d.StatusCode, &d.Delivered, &d.DurationMS, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}
