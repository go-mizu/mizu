package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func (s *Store) ListChannels(ctx context.Context) ([]types.Channel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, type, name, config, status, created_at, updated_at FROM channels ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []types.Channel
	for rows.Next() {
		var c types.Channel
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.Config, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, c)
	}
	return channels, rows.Err()
}

func (s *Store) GetChannel(ctx context.Context, id string) (*types.Channel, error) {
	var c types.Channel
	err := s.db.QueryRowContext(ctx, `SELECT id, type, name, config, status, created_at, updated_at FROM channels WHERE id = ?`, id).
		Scan(&c.ID, &c.Type, &c.Name, &c.Config, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("channel not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CreateChannel(ctx context.Context, c *types.Channel) error {
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = "disconnected"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO channels (id, type, name, config, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Type, c.Name, c.Config, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *Store) UpdateChannel(ctx context.Context, c *types.Channel) error {
	c.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE channels SET type=?, name=?, config=?, status=?, updated_at=? WHERE id=?`,
		c.Type, c.Name, c.Config, c.Status, c.UpdatedAt, c.ID)
	return err
}

func (s *Store) DeleteChannel(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM channels WHERE id = ?`, id)
	return err
}
