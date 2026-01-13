package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

// AIGatewayStoreImpl implements store.AIGatewayStore.
type AIGatewayStoreImpl struct {
	db    *sql.DB
	cache map[string]cachedResponse
}

type cachedResponse struct {
	data      []byte
	expiresAt time.Time
}

func (s *AIGatewayStoreImpl) CreateGateway(ctx context.Context, gw *store.AIGateway) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_gateways (id, name, collect_logs, cache_enabled, cache_ttl, rate_limit_enabled, rate_limit_count, rate_limit_period, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		gw.ID, gw.Name, gw.CollectLogs, gw.CacheEnabled, gw.CacheTTL,
		gw.RateLimitEnabled, gw.RateLimitCount, gw.RateLimitPeriod, gw.CreatedAt)
	return err
}

func (s *AIGatewayStoreImpl) GetGateway(ctx context.Context, id string) (*store.AIGateway, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, collect_logs, cache_enabled, cache_ttl, rate_limit_enabled, rate_limit_count, rate_limit_period, created_at
		FROM ai_gateways WHERE id = ?`, id)
	return s.scanGateway(row)
}

func (s *AIGatewayStoreImpl) GetGatewayByName(ctx context.Context, name string) (*store.AIGateway, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, collect_logs, cache_enabled, cache_ttl, rate_limit_enabled, rate_limit_count, rate_limit_period, created_at
		FROM ai_gateways WHERE name = ?`, name)
	return s.scanGateway(row)
}

func (s *AIGatewayStoreImpl) scanGateway(row *sql.Row) (*store.AIGateway, error) {
	var gw store.AIGateway
	if err := row.Scan(&gw.ID, &gw.Name, &gw.CollectLogs, &gw.CacheEnabled, &gw.CacheTTL,
		&gw.RateLimitEnabled, &gw.RateLimitCount, &gw.RateLimitPeriod, &gw.CreatedAt); err != nil {
		return nil, err
	}
	return &gw, nil
}

func (s *AIGatewayStoreImpl) ListGateways(ctx context.Context) ([]*store.AIGateway, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, collect_logs, cache_enabled, cache_ttl, rate_limit_enabled, rate_limit_count, rate_limit_period, created_at
		FROM ai_gateways ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []*store.AIGateway
	for rows.Next() {
		var gw store.AIGateway
		if err := rows.Scan(&gw.ID, &gw.Name, &gw.CollectLogs, &gw.CacheEnabled, &gw.CacheTTL,
			&gw.RateLimitEnabled, &gw.RateLimitCount, &gw.RateLimitPeriod, &gw.CreatedAt); err != nil {
			return nil, err
		}
		gateways = append(gateways, &gw)
	}
	return gateways, rows.Err()
}

func (s *AIGatewayStoreImpl) UpdateGateway(ctx context.Context, gw *store.AIGateway) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE ai_gateways SET name = ?, collect_logs = ?, cache_enabled = ?, cache_ttl = ?,
		rate_limit_enabled = ?, rate_limit_count = ?, rate_limit_period = ?
		WHERE id = ?`,
		gw.Name, gw.CollectLogs, gw.CacheEnabled, gw.CacheTTL,
		gw.RateLimitEnabled, gw.RateLimitCount, gw.RateLimitPeriod, gw.ID)
	return err
}

func (s *AIGatewayStoreImpl) DeleteGateway(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_gateways WHERE id = ?`, id)
	return err
}

func (s *AIGatewayStoreImpl) LogRequest(ctx context.Context, log *store.AIGatewayLog) error {
	metadataJSON, _ := json.Marshal(log.Metadata)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_gateway_logs (id, gateway_id, provider, model, cached, status, duration_ms, tokens, cost, request, response, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.GatewayID, log.Provider, log.Model, log.Cached, log.Status,
		log.Duration, log.Tokens, log.Cost, log.Request, log.Response, string(metadataJSON), log.CreatedAt)
	return err
}

func (s *AIGatewayStoreImpl) GetLogs(ctx context.Context, gatewayID string, limit, offset int) ([]*store.AIGatewayLog, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, gateway_id, provider, model, cached, status, duration_ms, tokens, cost, request, response, metadata, created_at
		FROM ai_gateway_logs WHERE gateway_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		gatewayID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*store.AIGatewayLog
	for rows.Next() {
		var log store.AIGatewayLog
		var metadataJSON string
		if err := rows.Scan(&log.ID, &log.GatewayID, &log.Provider, &log.Model, &log.Cached,
			&log.Status, &log.Duration, &log.Tokens, &log.Cost, &log.Request, &log.Response,
			&metadataJSON, &log.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(metadataJSON), &log.Metadata)
		logs = append(logs, &log)
	}
	return logs, rows.Err()
}

func (s *AIGatewayStoreImpl) GetCachedResponse(ctx context.Context, gatewayID, cacheKey string) ([]byte, bool, error) {
	if s.cache == nil {
		s.cache = make(map[string]cachedResponse)
	}

	key := gatewayID + ":" + cacheKey
	if cached, ok := s.cache[key]; ok {
		if time.Now().Before(cached.expiresAt) {
			return cached.data, true, nil
		}
		delete(s.cache, key)
	}

	// Also check database for persistent cache
	var data []byte
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT response, expires_at FROM ai_gateway_cache WHERE gateway_id = ? AND cache_key = ? AND expires_at > ?`,
		gatewayID, cacheKey, time.Now()).Scan(&data, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	// Store in memory cache
	s.cache[key] = cachedResponse{data: data, expiresAt: expiresAt}
	return data, true, nil
}

func (s *AIGatewayStoreImpl) SetCachedResponse(ctx context.Context, gatewayID, cacheKey string, response []byte, ttl int) error {
	if s.cache == nil {
		s.cache = make(map[string]cachedResponse)
	}

	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)

	// Store in memory cache
	key := gatewayID + ":" + cacheKey
	s.cache[key] = cachedResponse{data: response, expiresAt: expiresAt}

	// Store in database for persistence
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO ai_gateway_cache (gateway_id, cache_key, response, expires_at)
		VALUES (?, ?, ?, ?)`,
		gatewayID, cacheKey, response, expiresAt)
	return err
}

// Schema for AI Gateway
const aiGatewaySchema = `
	-- AI Gateways
	CREATE TABLE IF NOT EXISTS ai_gateways (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		collect_logs INTEGER DEFAULT 1,
		cache_enabled INTEGER DEFAULT 0,
		cache_ttl INTEGER DEFAULT 3600,
		rate_limit_enabled INTEGER DEFAULT 0,
		rate_limit_count INTEGER DEFAULT 100,
		rate_limit_period INTEGER DEFAULT 60,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- AI Gateway Logs
	CREATE TABLE IF NOT EXISTS ai_gateway_logs (
		id TEXT PRIMARY KEY,
		gateway_id TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		cached INTEGER DEFAULT 0,
		status INTEGER NOT NULL,
		duration_ms INTEGER DEFAULT 0,
		tokens INTEGER DEFAULT 0,
		cost REAL DEFAULT 0,
		request BLOB,
		response BLOB,
		metadata TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (gateway_id) REFERENCES ai_gateways(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_ai_gateway_logs_gateway ON ai_gateway_logs(gateway_id);
	CREATE INDEX IF NOT EXISTS idx_ai_gateway_logs_created ON ai_gateway_logs(gateway_id, created_at);

	-- AI Gateway Cache
	CREATE TABLE IF NOT EXISTS ai_gateway_cache (
		gateway_id TEXT NOT NULL,
		cache_key TEXT NOT NULL,
		response BLOB NOT NULL,
		expires_at DATETIME NOT NULL,
		PRIMARY KEY (gateway_id, cache_key),
		FOREIGN KEY (gateway_id) REFERENCES ai_gateways(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_ai_gateway_cache_expires ON ai_gateway_cache(expires_at);
`
