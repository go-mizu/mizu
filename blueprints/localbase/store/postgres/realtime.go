package postgres

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RealtimeStore implements store.RealtimeStore using PostgreSQL.
type RealtimeStore struct {
	pool *pgxpool.Pool
}

// CreateChannel creates a new realtime channel.
func (s *RealtimeStore) CreateChannel(ctx context.Context, channel *store.Channel) error {
	sql := `
	INSERT INTO realtime.channels (id, name)
	VALUES ($1, $2)
	ON CONFLICT (name) DO NOTHING
	`

	_, err := s.pool.Exec(ctx, sql, channel.ID, channel.Name)
	return err
}

// GetChannel retrieves a channel by name.
func (s *RealtimeStore) GetChannel(ctx context.Context, name string) (*store.Channel, error) {
	sql := `
	SELECT id, name, inserted_at
	FROM realtime.channels
	WHERE name = $1
	`

	channel := &store.Channel{}

	err := s.pool.QueryRow(ctx, sql, name).Scan(
		&channel.ID,
		&channel.Name,
		&channel.InsertedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("channel not found")
	}
	if err != nil {
		return nil, err
	}

	return channel, nil
}

// ListChannels lists all channels.
func (s *RealtimeStore) ListChannels(ctx context.Context) ([]*store.Channel, error) {
	sql := `
	SELECT id, name, inserted_at
	FROM realtime.channels
	ORDER BY name
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*store.Channel
	for rows.Next() {
		channel := &store.Channel{}

		err := rows.Scan(
			&channel.ID,
			&channel.Name,
			&channel.InsertedAt,
		)
		if err != nil {
			return nil, err
		}

		channels = append(channels, channel)
	}

	return channels, nil
}

// DeleteChannel deletes a channel.
func (s *RealtimeStore) DeleteChannel(ctx context.Context, name string) error {
	sql := `DELETE FROM realtime.channels WHERE name = $1`
	_, err := s.pool.Exec(ctx, sql, name)
	return err
}

// CreateSubscription creates a new subscription.
func (s *RealtimeStore) CreateSubscription(ctx context.Context, sub *store.Subscription) error {
	sql := `
	INSERT INTO realtime.subscriptions (id, channel_id, user_id, filters, claims)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.pool.Exec(ctx, sql,
		sub.ID,
		sub.ChannelID,
		nullIfEmpty(sub.UserID),
		sub.Filters,
		sub.Claims,
	)

	return err
}

// GetSubscription retrieves a subscription by ID.
func (s *RealtimeStore) GetSubscription(ctx context.Context, id string) (*store.Subscription, error) {
	sql := `
	SELECT id, channel_id, user_id, filters, claims, created_at
	FROM realtime.subscriptions
	WHERE id = $1
	`

	sub := &store.Subscription{}
	var userID *string

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&sub.ID,
		&sub.ChannelID,
		&userID,
		&sub.Filters,
		&sub.Claims,
		&sub.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("subscription not found")
	}
	if err != nil {
		return nil, err
	}

	if userID != nil {
		sub.UserID = *userID
	}

	return sub, nil
}

// ListSubscriptions lists subscriptions for a channel.
func (s *RealtimeStore) ListSubscriptions(ctx context.Context, channelID string) ([]*store.Subscription, error) {
	sql := `
	SELECT id, channel_id, user_id, filters, claims, created_at
	FROM realtime.subscriptions
	WHERE channel_id = $1
	ORDER BY created_at
	`

	rows, err := s.pool.Query(ctx, sql, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*store.Subscription
	for rows.Next() {
		sub := &store.Subscription{}
		var userID *string

		err := rows.Scan(
			&sub.ID,
			&sub.ChannelID,
			&userID,
			&sub.Filters,
			&sub.Claims,
			&sub.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if userID != nil {
			sub.UserID = *userID
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

// DeleteSubscription deletes a subscription.
func (s *RealtimeStore) DeleteSubscription(ctx context.Context, id string) error {
	sql := `DELETE FROM realtime.subscriptions WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, id)
	return err
}
