package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func (s *Store) ListSessions(ctx context.Context) ([]types.Session, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, agent_id, channel_id, channel_type, peer_id, display_name, origin, status, metadata, created_at, updated_at FROM sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []types.Session
	for rows.Next() {
		var ss types.Session
		if err := rows.Scan(&ss.ID, &ss.AgentID, &ss.ChannelID, &ss.ChannelType, &ss.PeerID, &ss.DisplayName, &ss.Origin, &ss.Status, &ss.Metadata, &ss.CreatedAt, &ss.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, ss)
	}
	return sessions, rows.Err()
}

func (s *Store) GetSession(ctx context.Context, id string) (*types.Session, error) {
	var ss types.Session
	err := s.db.QueryRowContext(ctx, `SELECT id, agent_id, channel_id, channel_type, peer_id, display_name, origin, status, metadata, created_at, updated_at FROM sessions WHERE id = ?`, id).
		Scan(&ss.ID, &ss.AgentID, &ss.ChannelID, &ss.ChannelType, &ss.PeerID, &ss.DisplayName, &ss.Origin, &ss.Status, &ss.Metadata, &ss.CreatedAt, &ss.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

func (s *Store) GetOrCreateSession(ctx context.Context, agentID, channelID, channelType, peerID, displayName, origin string) (*types.Session, error) {
	// Look for existing active session
	var ss types.Session
	err := s.db.QueryRowContext(ctx,
		`SELECT id, agent_id, channel_id, channel_type, peer_id, display_name, origin, status, metadata, created_at, updated_at
		 FROM sessions
		 WHERE agent_id = ? AND channel_type = ? AND peer_id = ? AND status = 'active'
		 ORDER BY updated_at DESC LIMIT 1`,
		agentID, channelType, peerID,
	).Scan(&ss.ID, &ss.AgentID, &ss.ChannelID, &ss.ChannelType, &ss.PeerID, &ss.DisplayName, &ss.Origin, &ss.Status, &ss.Metadata, &ss.CreatedAt, &ss.UpdatedAt)

	if err == nil {
		// Touch the session
		ss.UpdatedAt = time.Now().UTC()
		s.db.ExecContext(ctx, `UPDATE sessions SET updated_at = ? WHERE id = ?`, ss.UpdatedAt, ss.ID)
		return &ss, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new session
	id := generateID()
	now := time.Now().UTC()
	ss = types.Session{
		ID:          id,
		AgentID:     agentID,
		ChannelID:   channelID,
		ChannelType: channelType,
		PeerID:      peerID,
		DisplayName: displayName,
		Origin:      origin,
		Status:      "active",
		Metadata:    "{}",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, agent_id, channel_id, channel_type, peer_id, display_name, origin, status, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ss.ID, ss.AgentID, ss.ChannelID, ss.ChannelType, ss.PeerID, ss.DisplayName, ss.Origin, ss.Status, ss.Metadata, ss.CreatedAt, ss.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

func (s *Store) UpdateSession(ctx context.Context, ss *types.Session) error {
	ss.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET display_name=?, status=?, metadata=?, updated_at=? WHERE id=?`,
		ss.DisplayName, ss.Status, ss.Metadata, ss.UpdatedAt, ss.ID)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// PatchSession updates specific fields of a session by ID.
func (s *Store) PatchSession(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	// Map JSON field names to database column names.
	colMap := map[string]string{
		"label":                         "display_name",
		"display_name":                  "display_name",
		"thinking_level":                "thinking_level",
		"verbose_level":                 "verbose_level",
		"reasoning_level":               "reasoning_level",
		"model":                         "model",
		"model_override":                "model_override",
		"response_usage":                "response_usage",
		"send_policy":                   "send_policy",
		"memory_flush_at":               "memory_flush_at",
		"memory_flush_compaction_count": "memory_flush_compaction_count",
		"metadata":                      "metadata",
		"status":                        "status",
	}

	var setClauses []string
	var args []any
	for field, val := range updates {
		col, ok := colMap[field]
		if !ok {
			col = field // use field name directly as column name
		}
		setClauses = append(setClauses, col+" = ?")
		args = append(args, val)
	}
	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE sessions SET %s, updated_at = datetime('now') WHERE id = ?",
		strings.Join(setClauses, ", "),
	)

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// CreateSession inserts a new session into the database.
func (s *Store) CreateSession(ctx context.Context, ss *types.Session) error {
	if ss.Metadata == "" {
		ss.Metadata = "{}"
	}
	now := time.Now().UTC()
	ss.CreatedAt = now
	ss.UpdatedAt = now

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, agent_id, channel_id, channel_type, peer_id, display_name, origin, status, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ss.ID, ss.AgentID, ss.ChannelID, ss.ChannelType, ss.PeerID, ss.DisplayName, ss.Origin, ss.Status, ss.Metadata, ss.CreatedAt, ss.UpdatedAt,
	)
	return err
}

func (s *Store) ExpireSessions(ctx context.Context, mode string, idleMinutes int) (int, error) {
	var res sql.Result
	var err error

	switch mode {
	case "idle":
		cutoff := time.Now().UTC().Add(-time.Duration(idleMinutes) * time.Minute)
		res, err = s.db.ExecContext(ctx, `UPDATE sessions SET status = 'expired' WHERE status = 'active' AND updated_at < ?`, cutoff)
	case "daily":
		// Expire sessions created before today
		today := time.Now().UTC().Truncate(24 * time.Hour)
		res, err = s.db.ExecContext(ctx, `UPDATE sessions SET status = 'expired' WHERE status = 'active' AND created_at < ?`, today)
	default:
		return 0, nil
	}

	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func generateID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}
