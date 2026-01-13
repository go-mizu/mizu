package sqlite

import (
	"context"
	"database/sql"
	"sync"

	"github.com/go-mizu/blueprints/localflare/store"
)

// HyperdriveStoreImpl implements store.HyperdriveStore.
type HyperdriveStoreImpl struct {
	db    *sql.DB
	mu    sync.RWMutex
	stats map[string]*store.HyperdriveStats
}

func (s *HyperdriveStoreImpl) CreateConfig(ctx context.Context, cfg *store.HyperdriveConfig) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO hyperdrive_configs (id, name, origin_database, origin_host, origin_port, origin_scheme, origin_user, origin_password, cache_disabled, cache_max_age, cache_stale_while_revalidate, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cfg.ID, cfg.Name, cfg.Origin.Database, cfg.Origin.Host, cfg.Origin.Port,
		cfg.Origin.Scheme, cfg.Origin.User, cfg.Origin.Password,
		cfg.Caching.Disabled, cfg.Caching.MaxAge, cfg.Caching.StaleWhileRevalidate, cfg.CreatedAt)
	return err
}

func (s *HyperdriveStoreImpl) GetConfig(ctx context.Context, id string) (*store.HyperdriveConfig, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, origin_database, origin_host, origin_port, origin_scheme, origin_user, origin_password, cache_disabled, cache_max_age, cache_stale_while_revalidate, created_at
		FROM hyperdrive_configs WHERE id = ?`, id)
	return s.scanConfig(row)
}

func (s *HyperdriveStoreImpl) GetConfigByName(ctx context.Context, name string) (*store.HyperdriveConfig, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, origin_database, origin_host, origin_port, origin_scheme, origin_user, origin_password, cache_disabled, cache_max_age, cache_stale_while_revalidate, created_at
		FROM hyperdrive_configs WHERE name = ?`, name)
	return s.scanConfig(row)
}

func (s *HyperdriveStoreImpl) scanConfig(row *sql.Row) (*store.HyperdriveConfig, error) {
	var cfg store.HyperdriveConfig
	if err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Origin.Database, &cfg.Origin.Host,
		&cfg.Origin.Port, &cfg.Origin.Scheme, &cfg.Origin.User, &cfg.Origin.Password,
		&cfg.Caching.Disabled, &cfg.Caching.MaxAge, &cfg.Caching.StaleWhileRevalidate,
		&cfg.CreatedAt); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *HyperdriveStoreImpl) ListConfigs(ctx context.Context) ([]*store.HyperdriveConfig, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, origin_database, origin_host, origin_port, origin_scheme, origin_user, origin_password, cache_disabled, cache_max_age, cache_stale_while_revalidate, created_at
		FROM hyperdrive_configs ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*store.HyperdriveConfig
	for rows.Next() {
		var cfg store.HyperdriveConfig
		if err := rows.Scan(&cfg.ID, &cfg.Name, &cfg.Origin.Database, &cfg.Origin.Host,
			&cfg.Origin.Port, &cfg.Origin.Scheme, &cfg.Origin.User, &cfg.Origin.Password,
			&cfg.Caching.Disabled, &cfg.Caching.MaxAge, &cfg.Caching.StaleWhileRevalidate,
			&cfg.CreatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, &cfg)
	}
	return configs, rows.Err()
}

func (s *HyperdriveStoreImpl) UpdateConfig(ctx context.Context, cfg *store.HyperdriveConfig) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE hyperdrive_configs SET name = ?, origin_database = ?, origin_host = ?, origin_port = ?, origin_scheme = ?, origin_user = ?, origin_password = ?, cache_disabled = ?, cache_max_age = ?, cache_stale_while_revalidate = ?
		WHERE id = ?`,
		cfg.Name, cfg.Origin.Database, cfg.Origin.Host, cfg.Origin.Port,
		cfg.Origin.Scheme, cfg.Origin.User, cfg.Origin.Password,
		cfg.Caching.Disabled, cfg.Caching.MaxAge, cfg.Caching.StaleWhileRevalidate, cfg.ID)
	return err
}

func (s *HyperdriveStoreImpl) DeleteConfig(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM hyperdrive_configs WHERE id = ?`, id)
	return err
}

func (s *HyperdriveStoreImpl) GetStats(ctx context.Context, configID string) (*store.HyperdriveStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats == nil {
		s.stats = make(map[string]*store.HyperdriveStats)
	}

	if stats, ok := s.stats[configID]; ok {
		return stats, nil
	}

	// Return default stats
	return &store.HyperdriveStats{
		ActiveConnections: 0,
		IdleConnections:   0,
		TotalConnections:  0,
		QueriesPerSecond:  0,
		CacheHitRate:      0,
	}, nil
}

// UpdateStats updates the stats for a config (called by the proxy layer).
func (s *HyperdriveStoreImpl) UpdateStats(configID string, stats *store.HyperdriveStats) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats == nil {
		s.stats = make(map[string]*store.HyperdriveStats)
	}
	s.stats[configID] = stats
}

// Schema for Hyperdrive
const hyperdriveSchema = `
	-- Hyperdrive Configs
	CREATE TABLE IF NOT EXISTS hyperdrive_configs (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		origin_database TEXT NOT NULL,
		origin_host TEXT NOT NULL,
		origin_port INTEGER NOT NULL,
		origin_scheme TEXT DEFAULT 'postgres',
		origin_user TEXT NOT NULL,
		origin_password TEXT NOT NULL,
		cache_disabled INTEGER DEFAULT 0,
		cache_max_age INTEGER DEFAULT 60,
		cache_stale_while_revalidate INTEGER DEFAULT 15,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
`
