package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func (s *Store) ListBindings(ctx context.Context) ([]types.Binding, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, agent_id, channel_type, channel_id, peer_id, priority FROM bindings ORDER BY priority DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []types.Binding
	for rows.Next() {
		var b types.Binding
		if err := rows.Scan(&b.ID, &b.AgentID, &b.ChannelType, &b.ChannelID, &b.PeerID, &b.Priority); err != nil {
			return nil, err
		}
		bindings = append(bindings, b)
	}
	return bindings, rows.Err()
}

func (s *Store) CreateBinding(ctx context.Context, b *types.Binding) error {
	if b.ID == "" {
		b.ID = generateID()
	}
	if b.ChannelType == "" {
		b.ChannelType = "*"
	}
	if b.ChannelID == "" {
		b.ChannelID = "*"
	}
	if b.PeerID == "" {
		b.PeerID = "*"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO bindings (id, agent_id, channel_type, channel_id, peer_id, priority) VALUES (?, ?, ?, ?, ?, ?)`,
		b.ID, b.AgentID, b.ChannelType, b.ChannelID, b.PeerID, b.Priority)
	return err
}

func (s *Store) DeleteBinding(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bindings WHERE id = ?`, id)
	return err
}

// ResolveAgent finds the best-matching agent for an inbound message.
// Priority order: exact peer > exact channel > channel type > wildcard.
func (s *Store) ResolveAgent(ctx context.Context, channelType, channelID, peerID string) (*types.Agent, error) {
	// Query bindings ordered by specificity (most specific first)
	var agentID string
	err := s.db.QueryRowContext(ctx, `
		SELECT b.agent_id FROM bindings b
		JOIN agents a ON a.id = b.agent_id AND a.status = 'active'
		WHERE
			(b.channel_type = ? OR b.channel_type = '*') AND
			(b.channel_id = ? OR b.channel_id = '*') AND
			(b.peer_id = ? OR b.peer_id = '*')
		ORDER BY
			CASE WHEN b.peer_id != '*' THEN 3 ELSE 0 END +
			CASE WHEN b.channel_id != '*' THEN 2 ELSE 0 END +
			CASE WHEN b.channel_type != '*' THEN 1 ELSE 0 END
		DESC, b.priority DESC
		LIMIT 1
	`, channelType, channelID, peerID).Scan(&agentID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no agent bound for %s/%s/%s", channelType, channelID, peerID)
	}
	if err != nil {
		return nil, err
	}

	return s.GetAgent(ctx, agentID)
}
