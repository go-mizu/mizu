package sqlite

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func (s *Store) ListMessages(ctx context.Context, sessionID string, limit int) ([]types.Message, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, agent_id, channel_id, peer_id, role, content, metadata, created_at
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY created_at ASC
		 LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var m types.Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.AgentID, &m.ChannelID, &m.PeerID, &m.Role, &m.Content, &m.Metadata, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (s *Store) CreateMessage(ctx context.Context, m *types.Message) error {
	if m.ID == "" {
		m.ID = generateID()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	if m.Metadata == "" {
		m.Metadata = "{}"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (id, session_id, agent_id, channel_id, peer_id, role, content, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.SessionID, m.AgentID, m.ChannelID, m.PeerID, m.Role, m.Content, m.Metadata, m.CreatedAt)
	return err
}

func (s *Store) CountMessages(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages`).Scan(&count)
	return count, err
}
