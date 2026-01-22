package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// SubscriptionStore implements store.SubscriptionStore.
type SubscriptionStore struct {
	db *sql.DB
}

func (s *SubscriptionStore) Create(ctx context.Context, sub *store.Subscription) error {
	if sub.ID == "" {
		sub.ID = generateID()
	}
	sub.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions (id, dashboard_id, schedule, format, recipients, enabled, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, sub.ID, sub.DashboardID, sub.Schedule, sub.Format, toJSON(sub.Recipients), sub.Enabled, sub.CreatedBy, sub.CreatedAt)
	return err
}

func (s *SubscriptionStore) GetByID(ctx context.Context, id string) (*store.Subscription, error) {
	var sub store.Subscription
	var recipients string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, dashboard_id, schedule, format, recipients, enabled, created_by, created_at
		FROM subscriptions WHERE id = ?
	`, id).Scan(&sub.ID, &sub.DashboardID, &sub.Schedule, &sub.Format, &recipients, &sub.Enabled, &sub.CreatedBy, &sub.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(recipients, &sub.Recipients)
	return &sub, nil
}

func (s *SubscriptionStore) List(ctx context.Context) ([]*store.Subscription, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, schedule, format, recipients, enabled, created_by, created_at
		FROM subscriptions
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Subscription
	for rows.Next() {
		var sub store.Subscription
		var recipients string
		if err := rows.Scan(&sub.ID, &sub.DashboardID, &sub.Schedule, &sub.Format, &recipients, &sub.Enabled, &sub.CreatedBy, &sub.CreatedAt); err != nil {
			return nil, err
		}
		fromJSON(recipients, &sub.Recipients)
		result = append(result, &sub)
	}
	return result, rows.Err()
}

func (s *SubscriptionStore) ListByDashboard(ctx context.Context, dashboardID string) ([]*store.Subscription, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, schedule, format, recipients, enabled, created_by, created_at
		FROM subscriptions WHERE dashboard_id = ?
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Subscription
	for rows.Next() {
		var sub store.Subscription
		var recipients string
		if err := rows.Scan(&sub.ID, &sub.DashboardID, &sub.Schedule, &sub.Format, &recipients, &sub.Enabled, &sub.CreatedBy, &sub.CreatedAt); err != nil {
			return nil, err
		}
		fromJSON(recipients, &sub.Recipients)
		result = append(result, &sub)
	}
	return result, rows.Err()
}

func (s *SubscriptionStore) Update(ctx context.Context, sub *store.Subscription) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE subscriptions SET schedule=?, format=?, recipients=?, enabled=?
		WHERE id=?
	`, sub.Schedule, sub.Format, toJSON(sub.Recipients), sub.Enabled, sub.ID)
	return err
}

func (s *SubscriptionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id=?`, id)
	return err
}
