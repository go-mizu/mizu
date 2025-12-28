package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/webhooks"
)

// WebhooksStore handles webhook data access.
type WebhooksStore struct {
	db *sql.DB
}

// NewWebhooksStore creates a new webhooks store.
func NewWebhooksStore(db *sql.DB) *WebhooksStore {
	return &WebhooksStore{db: db}
}

func (s *WebhooksStore) Create(ctx context.Context, w *webhooks.Webhook) error {
	now := time.Now()
	w.CreatedAt = now
	w.UpdatedAt = now

	eventsJSON, _ := json.Marshal(w.Events)

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO webhooks (node_id, owner_id, owner_type, name, url, content_type, secret,
			insecure_ssl, events, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, "", w.OwnerID, w.OwnerType, w.Name, w.Config.URL, w.Config.ContentType, w.Config.Secret,
		w.Config.InsecureSSL, string(eventsJSON), w.Active, w.CreatedAt, w.UpdatedAt).Scan(&w.ID)
	if err != nil {
		return err
	}

	w.NodeID = generateNodeID("H", w.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE webhooks SET node_id = $1 WHERE id = $2`, w.NodeID, w.ID)
	return err
}

func (s *WebhooksStore) GetByID(ctx context.Context, id int64) (*webhooks.Webhook, error) {
	w := &webhooks.Webhook{Config: &webhooks.Config{}}
	var eventsJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, owner_id, owner_type, name, url, content_type, secret, insecure_ssl,
			events, active, created_at, updated_at
		FROM webhooks WHERE id = $1
	`, id).Scan(&w.ID, &w.NodeID, &w.OwnerID, &w.OwnerType, &w.Name, &w.Config.URL,
		&w.Config.ContentType, &w.Config.Secret, &w.Config.InsecureSSL,
		&eventsJSON, &w.Active, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(eventsJSON), &w.Events)
	return w, nil
}

func (s *WebhooksStore) Update(ctx context.Context, id int64, in *webhooks.UpdateIn) error {
	w, err := s.GetByID(ctx, id)
	if err != nil || w == nil {
		return err
	}

	// Update config fields
	if in.Config != nil {
		if in.Config.URL != "" {
			w.Config.URL = in.Config.URL
		}
		if in.Config.ContentType != "" {
			w.Config.ContentType = in.Config.ContentType
		}
		if in.Config.Secret != "" {
			w.Config.Secret = in.Config.Secret
		}
		if in.Config.InsecureSSL != "" {
			w.Config.InsecureSSL = in.Config.InsecureSSL
		}
	}

	events := w.Events
	if in.Events != nil {
		events = in.Events
	}
	if in.AddEvents != nil {
		for _, e := range in.AddEvents {
			found := false
			for _, existing := range events {
				if existing == e {
					found = true
					break
				}
			}
			if !found {
				events = append(events, e)
			}
		}
	}
	if in.RemoveEvents != nil {
		var filtered []string
		for _, e := range events {
			remove := false
			for _, r := range in.RemoveEvents {
				if e == r {
					remove = true
					break
				}
			}
			if !remove {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	active := w.Active
	if in.Active != nil {
		active = *in.Active
	}

	eventsJSON, _ := json.Marshal(events)

	_, err = s.db.ExecContext(ctx, `
		UPDATE webhooks SET
			url = $2,
			content_type = $3,
			secret = $4,
			insecure_ssl = $5,
			events = $6,
			active = $7,
			updated_at = $8
		WHERE id = $1
	`, id, w.Config.URL, w.Config.ContentType, w.Config.Secret, w.Config.InsecureSSL,
		string(eventsJSON), active, time.Now())
	return err
}

func (s *WebhooksStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	return err
}

func (s *WebhooksStore) ListByOwner(ctx context.Context, ownerID int64, ownerType string, opts *webhooks.ListOpts) ([]*webhooks.Webhook, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT id, node_id, owner_id, owner_type, name, url, content_type, secret, insecure_ssl,
			events, active, created_at, updated_at
		FROM webhooks WHERE owner_id = $1 AND owner_type = $2
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, ownerID, ownerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*webhooks.Webhook
	for rows.Next() {
		w := &webhooks.Webhook{Config: &webhooks.Config{}}
		var eventsJSON string
		if err := rows.Scan(&w.ID, &w.NodeID, &w.OwnerID, &w.OwnerType, &w.Name, &w.Config.URL,
			&w.Config.ContentType, &w.Config.Secret, &w.Config.InsecureSSL,
			&eventsJSON, &w.Active, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(eventsJSON), &w.Events)
		list = append(list, w)
	}
	return list, rows.Err()
}

// Delivery methods

func (s *WebhooksStore) CreateDelivery(ctx context.Context, d *webhooks.Delivery) error {
	if d.DeliveredAt.IsZero() {
		d.DeliveredAt = time.Now()
	}

	var reqHeaders, respHeaders, reqPayload []byte
	if d.Request != nil {
		reqHeaders, _ = json.Marshal(d.Request.Headers)
		reqPayload, _ = json.Marshal(d.Request.Payload)
	}
	if d.Response != nil {
		respHeaders, _ = json.Marshal(d.Response.Headers)
	}
	respPayload := ""
	if d.Response != nil {
		respPayload = d.Response.Payload
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO webhook_deliveries (hook_id, guid, delivered_at, redelivery, duration, status,
			status_code, event, action, request_headers, request_payload, response_headers, response_payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, d.HookID, d.GUID, d.DeliveredAt, d.Redelivery, d.Duration, d.Status, d.StatusCode,
		d.Event, d.Action, string(reqHeaders), string(reqPayload),
		string(respHeaders), respPayload).Scan(&d.ID)
	return err
}

func (s *WebhooksStore) GetDeliveryByID(ctx context.Context, id int64) (*webhooks.Delivery, error) {
	d := &webhooks.Delivery{
		Request:  &webhooks.DeliveryRequest{},
		Response: &webhooks.DeliveryResponse{},
	}
	var reqHeaders, respHeaders, reqPayload string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, hook_id, guid, delivered_at, redelivery, duration, status, status_code,
			event, action, request_headers, request_payload, response_headers, response_payload
		FROM webhook_deliveries WHERE id = $1
	`, id).Scan(&d.ID, &d.HookID, &d.GUID, &d.DeliveredAt, &d.Redelivery, &d.Duration, &d.Status,
		&d.StatusCode, &d.Event, &d.Action, &reqHeaders, &reqPayload,
		&respHeaders, &d.Response.Payload)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(reqHeaders), &d.Request.Headers)
	_ = json.Unmarshal([]byte(respHeaders), &d.Response.Headers)
	if reqPayload != "" {
		_ = json.Unmarshal([]byte(reqPayload), &d.Request.Payload)
	}
	return d, nil
}

func (s *WebhooksStore) ListDeliveries(ctx context.Context, hookID int64, opts *webhooks.ListOpts) ([]*webhooks.Delivery, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT id, hook_id, guid, delivered_at, redelivery, duration, status, status_code,
			event, action, request_headers, request_payload, response_headers, response_payload
		FROM webhook_deliveries WHERE hook_id = $1
		ORDER BY delivered_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, hookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*webhooks.Delivery
	for rows.Next() {
		d := &webhooks.Delivery{
			Request:  &webhooks.DeliveryRequest{},
			Response: &webhooks.DeliveryResponse{},
		}
		var reqHeaders, respHeaders, reqPayload string
		if err := rows.Scan(&d.ID, &d.HookID, &d.GUID, &d.DeliveredAt, &d.Redelivery, &d.Duration, &d.Status,
			&d.StatusCode, &d.Event, &d.Action, &reqHeaders, &reqPayload,
			&respHeaders, &d.Response.Payload); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(reqHeaders), &d.Request.Headers)
		_ = json.Unmarshal([]byte(respHeaders), &d.Response.Headers)
		if reqPayload != "" {
			_ = json.Unmarshal([]byte(reqPayload), &d.Request.Payload)
		}
		list = append(list, d)
	}
	return list, rows.Err()
}
