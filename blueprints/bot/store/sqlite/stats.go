package sqlite

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/bot/store"
)

func (s *Store) Stats(ctx context.Context) (*store.Stats, error) {
	st := &store.Stats{}

	queries := []struct {
		q   string
		dst *int
	}{
		{"SELECT COUNT(*) FROM agents", &st.Agents},
		{"SELECT COUNT(*) FROM channels", &st.Channels},
		{"SELECT COUNT(*) FROM sessions", &st.Sessions},
		{"SELECT COUNT(*) FROM messages", &st.Messages},
		{"SELECT COUNT(*) FROM bindings", &st.Bindings},
	}

	for _, q := range queries {
		if err := s.db.QueryRowContext(ctx, q.q).Scan(q.dst); err != nil {
			return nil, err
		}
	}

	return st, nil
}
