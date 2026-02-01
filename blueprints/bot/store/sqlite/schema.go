package sqlite

import "context"

func (s *Store) createSchema(ctx context.Context) error {
	queries := []string{
		// Agents table
		`CREATE TABLE IF NOT EXISTS agents (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			model       TEXT NOT NULL DEFAULT 'claude-sonnet-4-20250514',
			system_prompt TEXT NOT NULL DEFAULT '',
			workspace   TEXT NOT NULL DEFAULT '',
			max_tokens  INTEGER NOT NULL DEFAULT 4096,
			temperature REAL NOT NULL DEFAULT 0.7,
			status      TEXT NOT NULL DEFAULT 'active',
			created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		// Channels table
		`CREATE TABLE IF NOT EXISTS channels (
			id          TEXT PRIMARY KEY,
			type        TEXT NOT NULL,
			name        TEXT NOT NULL,
			config      TEXT NOT NULL DEFAULT '{}',
			status      TEXT NOT NULL DEFAULT 'disconnected',
			created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		// Sessions table
		`CREATE TABLE IF NOT EXISTS sessions (
			id           TEXT PRIMARY KEY,
			agent_id     TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			channel_id   TEXT NOT NULL DEFAULT '',
			channel_type TEXT NOT NULL DEFAULT '',
			peer_id      TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			origin       TEXT NOT NULL DEFAULT 'dm',
			status       TEXT NOT NULL DEFAULT 'active',
			metadata     TEXT NOT NULL DEFAULT '{}',
			created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at   DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_peer ON sessions(channel_type, channel_id, peer_id)`,

		// Messages table
		`CREATE TABLE IF NOT EXISTS messages (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			agent_id   TEXT NOT NULL DEFAULT '',
			channel_id TEXT NOT NULL DEFAULT '',
			peer_id    TEXT NOT NULL DEFAULT '',
			role       TEXT NOT NULL,
			content    TEXT NOT NULL,
			metadata   TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, created_at)`,

		// Bindings table (agent-channel routing rules)
		`CREATE TABLE IF NOT EXISTS bindings (
			id           TEXT PRIMARY KEY,
			agent_id     TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			channel_type TEXT NOT NULL DEFAULT '*',
			channel_id   TEXT NOT NULL DEFAULT '*',
			peer_id      TEXT NOT NULL DEFAULT '*',
			priority     INTEGER NOT NULL DEFAULT 0
		)`,

		`CREATE INDEX IF NOT EXISTS idx_bindings_route ON bindings(channel_type, channel_id, peer_id)`,

		// Pairing codes for DM security
		`CREATE TABLE IF NOT EXISTS pairing_codes (
			id           TEXT PRIMARY KEY,
			channel_type TEXT NOT NULL,
			peer_id      TEXT NOT NULL,
			code         TEXT NOT NULL,
			status       TEXT NOT NULL DEFAULT 'pending',
			created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
			expires_at   DATETIME NOT NULL
		)`,

		// Gateway config key-value store
		`CREATE TABLE IF NOT EXISTS config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,

		// Cron jobs table
		`CREATE TABLE IF NOT EXISTS cron_jobs (
			id             TEXT PRIMARY KEY,
			name           TEXT NOT NULL,
			description    TEXT NOT NULL DEFAULT '',
			agent_id       TEXT NOT NULL DEFAULT '',
			enabled        INTEGER NOT NULL DEFAULT 1,
			schedule       TEXT NOT NULL DEFAULT '{}',
			session_target TEXT NOT NULL DEFAULT 'main',
			wake_mode      TEXT NOT NULL DEFAULT 'next-heartbeat',
			payload        TEXT NOT NULL DEFAULT '{}',
			last_run_at    DATETIME,
			last_status    TEXT NOT NULL DEFAULT '',
			created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at     DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		// Cron run history
		`CREATE TABLE IF NOT EXISTS cron_runs (
			id          TEXT PRIMARY KEY,
			job_id      TEXT NOT NULL REFERENCES cron_jobs(id) ON DELETE CASCADE,
			status      TEXT NOT NULL DEFAULT 'running',
			started_at  DATETIME NOT NULL DEFAULT (datetime('now')),
			ended_at    DATETIME,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			summary     TEXT NOT NULL DEFAULT '',
			error       TEXT NOT NULL DEFAULT ''
		)`,

		`CREATE INDEX IF NOT EXISTS idx_cron_runs_job ON cron_runs(job_id, started_at DESC)`,
	}

	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}

	return nil
}
