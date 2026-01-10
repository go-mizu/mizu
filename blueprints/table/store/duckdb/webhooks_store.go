package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/table/feature/webhooks"
)

// WebhooksStore provides DuckDB-based webhook storage.
type WebhooksStore struct {
	db *sql.DB
}

// NewWebhooksStore creates a new webhooks store.
func NewWebhooksStore(db *sql.DB) *WebhooksStore {
	return &WebhooksStore{db: db}
}

// Create creates a new webhook.
func (s *WebhooksStore) Create(ctx context.Context, webhook *webhooks.Webhook) error {
	webhook.CreatedAt = time.Now()
	if !webhook.IsActive {
		webhook.IsActive = true
	}

	eventsJSON, _ := json.Marshal(webhook.Events)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhooks (id, base_id, table_id, url, events, secret, is_active, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, webhook.ID, webhook.BaseID, webhook.TableID, webhook.URL, string(eventsJSON), webhook.Secret, webhook.IsActive, webhook.CreatedBy, webhook.CreatedAt)
	return err
}

// GetByID retrieves a webhook by ID.
func (s *WebhooksStore) GetByID(ctx context.Context, id string) (*webhooks.Webhook, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, base_id, table_id, url, events, secret, is_active, created_by, created_at
		FROM webhooks WHERE id = $1
	`, id)
	return s.scanWebhook(row)
}

// Update updates a webhook.
func (s *WebhooksStore) Update(ctx context.Context, webhook *webhooks.Webhook) error {
	eventsJSON, _ := json.Marshal(webhook.Events)

	_, err := s.db.ExecContext(ctx, `
		UPDATE webhooks SET
			url = $1, events = $2, secret = $3, is_active = $4
		WHERE id = $5
	`, webhook.URL, string(eventsJSON), webhook.Secret, webhook.IsActive, webhook.ID)
	return err
}

// Delete deletes a webhook.
func (s *WebhooksStore) Delete(ctx context.Context, id string) error {
	// Delete deliveries first
	_, _ = s.db.ExecContext(ctx, `DELETE FROM webhook_deliveries WHERE webhook_id = $1`, id)

	_, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	return err
}

// ListByBase lists all webhooks for a base.
func (s *WebhooksStore) ListByBase(ctx context.Context, baseID string) ([]*webhooks.Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, base_id, table_id, url, events, secret, is_active, created_by, created_at
		FROM webhooks WHERE base_id = $1
		ORDER BY created_at DESC
	`, baseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanWebhooks(rows)
}

// ListByTable lists all webhooks for a table.
func (s *WebhooksStore) ListByTable(ctx context.Context, tableID string) ([]*webhooks.Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, base_id, table_id, url, events, secret, is_active, created_by, created_at
		FROM webhooks WHERE table_id = $1
		ORDER BY created_at DESC
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanWebhooks(rows)
}

// CreateDelivery creates a new webhook delivery.
func (s *WebhooksStore) CreateDelivery(ctx context.Context, delivery *webhooks.Delivery) error {
	delivery.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status_code, response, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, delivery.ID, delivery.WebhookID, delivery.Event, delivery.Payload, delivery.StatusCode, delivery.Response, delivery.DurationMs, delivery.CreatedAt)
	return err
}

// GetDelivery retrieves a delivery by ID.
func (s *WebhooksStore) GetDelivery(ctx context.Context, id string) (*webhooks.Delivery, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, webhook_id, event, payload, status_code, response, duration_ms, created_at
		FROM webhook_deliveries WHERE id = $1
	`, id)

	delivery := &webhooks.Delivery{}
	var response sql.NullString
	var statusCode, durationMs sql.NullInt64

	err := row.Scan(&delivery.ID, &delivery.WebhookID, &delivery.Event, &delivery.Payload, &statusCode, &response, &durationMs, &delivery.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, webhooks.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if statusCode.Valid {
		delivery.StatusCode = int(statusCode.Int64)
	}
	if response.Valid {
		delivery.Response = response.String
	}
	if durationMs.Valid {
		delivery.DurationMs = int(durationMs.Int64)
	}

	return delivery, nil
}

// ListDeliveries lists deliveries for a webhook.
func (s *WebhooksStore) ListDeliveries(ctx context.Context, webhookID string, opts webhooks.ListOpts) ([]*webhooks.Delivery, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, webhook_id, event, payload, status_code, response, duration_ms, created_at
		FROM webhook_deliveries WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, webhookID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*webhooks.Delivery
	for rows.Next() {
		delivery := &webhooks.Delivery{}
		var response sql.NullString
		var statusCode, durationMs sql.NullInt64

		err := rows.Scan(&delivery.ID, &delivery.WebhookID, &delivery.Event, &delivery.Payload, &statusCode, &response, &durationMs, &delivery.CreatedAt)
		if err != nil {
			return nil, err
		}

		if statusCode.Valid {
			delivery.StatusCode = int(statusCode.Int64)
		}
		if response.Valid {
			delivery.Response = response.String
		}
		if durationMs.Valid {
			delivery.DurationMs = int(durationMs.Int64)
		}

		deliveries = append(deliveries, delivery)
	}
	return deliveries, rows.Err()
}

func (s *WebhooksStore) scanWebhook(row *sql.Row) (*webhooks.Webhook, error) {
	webhook := &webhooks.Webhook{}
	var tableID, secret sql.NullString
	var eventsStr string

	err := row.Scan(&webhook.ID, &webhook.BaseID, &tableID, &webhook.URL, &eventsStr, &secret, &webhook.IsActive, &webhook.CreatedBy, &webhook.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, webhooks.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if tableID.Valid {
		webhook.TableID = tableID.String
	}
	if secret.Valid {
		webhook.Secret = secret.String
	}
	json.Unmarshal([]byte(eventsStr), &webhook.Events)

	return webhook, nil
}

func (s *WebhooksStore) scanWebhooks(rows *sql.Rows) ([]*webhooks.Webhook, error) {
	var webhookList []*webhooks.Webhook
	for rows.Next() {
		webhook := &webhooks.Webhook{}
		var tableID, secret sql.NullString
		var eventsStr string

		err := rows.Scan(&webhook.ID, &webhook.BaseID, &tableID, &webhook.URL, &eventsStr, &secret, &webhook.IsActive, &webhook.CreatedBy, &webhook.CreatedAt)
		if err != nil {
			return nil, err
		}

		if tableID.Valid {
			webhook.TableID = tableID.String
		}
		if secret.Valid {
			webhook.Secret = secret.String
		}
		json.Unmarshal([]byte(eventsStr), &webhook.Events)

		webhookList = append(webhookList, webhook)
	}
	return webhookList, rows.Err()
}
