package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Store implements store.Store with SQLite.
type Store struct {
	db      *sql.DB
	dataDir string

	zones           *ZoneStore
	dns             *DNSStoreImpl
	ssl             *SSLStoreImpl
	firewall        *FirewallStoreImpl
	cache           *CacheStoreImpl
	workers         *WorkerStoreImpl
	kv              *KVStoreImpl
	r2              *R2StoreImpl
	d1              *D1StoreImpl
	loadBalancer    *LoadBalancerStoreImpl
	analytics       *AnalyticsStoreImpl
	rules           *RulesStoreImpl
	users           *UserStoreImpl
	durableObjects  *DurableObjectStoreImpl
	queues          *QueueStoreImpl
	vectorize       *VectorizeStoreImpl
	analyticsEngine *AnalyticsEngineStoreImpl
	ai              *AIStoreImpl
	aiGateway       *AIGatewayStoreImpl
	hyperdrive      *HyperdriveStoreImpl
	cron            *CronStoreImpl
}

// New creates a new SQLite store.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "localflare.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{
		db:      db,
		dataDir: dataDir,
	}

	// Initialize sub-stores
	s.zones = &ZoneStore{db: db}
	s.dns = &DNSStoreImpl{db: db}
	s.ssl = &SSLStoreImpl{db: db}
	s.firewall = &FirewallStoreImpl{db: db}
	s.cache = &CacheStoreImpl{db: db}
	s.workers = &WorkerStoreImpl{db: db}
	s.kv = &KVStoreImpl{db: db}
	s.r2 = &R2StoreImpl{db: db, dataDir: filepath.Join(dataDir, "r2")}
	s.d1 = &D1StoreImpl{db: db, dataDir: filepath.Join(dataDir, "d1")}
	s.loadBalancer = &LoadBalancerStoreImpl{db: db}
	s.analytics = &AnalyticsStoreImpl{db: db}
	s.rules = &RulesStoreImpl{db: db}
	s.users = &UserStoreImpl{db: db}
	s.durableObjects = &DurableObjectStoreImpl{db: db, dataDir: filepath.Join(dataDir, "do")}
	s.queues = &QueueStoreImpl{db: db}
	s.vectorize = &VectorizeStoreImpl{db: db}
	s.analyticsEngine = &AnalyticsEngineStoreImpl{db: db}
	s.ai = NewAIStore(db)
	s.aiGateway = &AIGatewayStoreImpl{db: db}
	s.hyperdrive = &HyperdriveStoreImpl{db: db}
	s.cron = &CronStoreImpl{db: db}

	return s, nil
}

// Ensure creates all required tables.
func (s *Store) Ensure(ctx context.Context) error {
	schema := `
	-- Zones
	CREATE TABLE IF NOT EXISTS zones (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		status TEXT DEFAULT 'active',
		plan TEXT DEFAULT 'free',
		name_servers TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- DNS Records
	CREATE TABLE IF NOT EXISTS dns_records (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		ttl INTEGER DEFAULT 300,
		priority INTEGER DEFAULT 0,
		proxied INTEGER DEFAULT 0,
		comment TEXT,
		tags TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_dns_records_zone ON dns_records(zone_id);
	CREATE INDEX IF NOT EXISTS idx_dns_records_name ON dns_records(name, type);

	-- SSL Certificates
	CREATE TABLE IF NOT EXISTS certificates (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		type TEXT NOT NULL,
		hosts TEXT NOT NULL,
		issuer TEXT,
		serial_number TEXT,
		signature TEXT,
		status TEXT DEFAULT 'active',
		expires_at DATETIME,
		certificate TEXT,
		private_key TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- SSL Settings
	CREATE TABLE IF NOT EXISTS ssl_settings (
		zone_id TEXT PRIMARY KEY,
		mode TEXT DEFAULT 'full',
		always_https INTEGER DEFAULT 0,
		min_tls_version TEXT DEFAULT '1.2',
		opportunistic_encryption INTEGER DEFAULT 0,
		tls_1_3 INTEGER DEFAULT 1,
		automatic_https_rewrites INTEGER DEFAULT 1,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- Firewall Rules
	CREATE TABLE IF NOT EXISTS firewall_rules (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		description TEXT,
		expression TEXT NOT NULL,
		action TEXT NOT NULL,
		priority INTEGER DEFAULT 0,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- IP Access Rules
	CREATE TABLE IF NOT EXISTS ip_access_rules (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		mode TEXT NOT NULL,
		target TEXT NOT NULL,
		value TEXT NOT NULL,
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- Rate Limit Rules
	CREATE TABLE IF NOT EXISTS rate_limit_rules (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		description TEXT,
		expression TEXT NOT NULL,
		threshold INTEGER NOT NULL,
		period INTEGER NOT NULL,
		action TEXT NOT NULL,
		action_timeout INTEGER DEFAULT 60,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- Cache Settings
	CREATE TABLE IF NOT EXISTS cache_settings (
		zone_id TEXT PRIMARY KEY,
		cache_level TEXT DEFAULT 'standard',
		browser_ttl INTEGER DEFAULT 14400,
		edge_ttl INTEGER DEFAULT 7200,
		development_mode INTEGER DEFAULT 0,
		always_online INTEGER DEFAULT 1,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- Cache Rules
	CREATE TABLE IF NOT EXISTS cache_rules (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		name TEXT NOT NULL,
		expression TEXT NOT NULL,
		cache_level TEXT,
		edge_ttl INTEGER,
		browser_ttl INTEGER,
		bypass_cache INTEGER DEFAULT 0,
		priority INTEGER DEFAULT 0,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- Workers
	CREATE TABLE IF NOT EXISTS workers (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		script TEXT NOT NULL,
		routes TEXT DEFAULT '[]',
		bindings TEXT DEFAULT '{}',
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Worker Routes
	CREATE TABLE IF NOT EXISTS worker_routes (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		pattern TEXT NOT NULL,
		worker_id TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE,
		FOREIGN KEY (worker_id) REFERENCES workers(id) ON DELETE CASCADE
	);

	-- KV Namespaces
	CREATE TABLE IF NOT EXISTS kv_namespaces (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- KV Pairs
	CREATE TABLE IF NOT EXISTS kv_pairs (
		namespace_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value BLOB,
		metadata TEXT,
		expiration DATETIME,
		PRIMARY KEY (namespace_id, key),
		FOREIGN KEY (namespace_id) REFERENCES kv_namespaces(id) ON DELETE CASCADE
	);

	-- R2 Buckets
	CREATE TABLE IF NOT EXISTS r2_buckets (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		location TEXT DEFAULT 'auto',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- R2 Objects (metadata only, data stored on disk)
	CREATE TABLE IF NOT EXISTS r2_objects (
		bucket_id TEXT NOT NULL,
		key TEXT NOT NULL,
		size INTEGER,
		etag TEXT,
		content_type TEXT,
		metadata TEXT,
		last_modified DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (bucket_id, key),
		FOREIGN KEY (bucket_id) REFERENCES r2_buckets(id) ON DELETE CASCADE
	);

	-- D1 Databases (metadata only)
	CREATE TABLE IF NOT EXISTS d1_databases (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		version TEXT DEFAULT '1',
		num_tables INTEGER DEFAULT 0,
		file_size INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Load Balancers
	CREATE TABLE IF NOT EXISTS load_balancers (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		name TEXT NOT NULL,
		fallback_pool TEXT,
		default_pools TEXT DEFAULT '[]',
		session_affinity TEXT DEFAULT 'none',
		steering_policy TEXT DEFAULT 'off',
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- Origin Pools
	CREATE TABLE IF NOT EXISTS origin_pools (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		origins TEXT NOT NULL,
		check_regions TEXT DEFAULT '[]',
		description TEXT,
		enabled INTEGER DEFAULT 1,
		minimum_origins INTEGER DEFAULT 1,
		monitor TEXT,
		notification_email TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Health Checks
	CREATE TABLE IF NOT EXISTS health_checks (
		id TEXT PRIMARY KEY,
		description TEXT,
		type TEXT DEFAULT 'http',
		method TEXT DEFAULT 'GET',
		path TEXT DEFAULT '/',
		header TEXT DEFAULT '{}',
		port INTEGER DEFAULT 80,
		timeout INTEGER DEFAULT 5,
		retries INTEGER DEFAULT 2,
		interval_seconds INTEGER DEFAULT 60,
		expected_body TEXT,
		expected_codes TEXT DEFAULT '200',
		follow_redirects INTEGER DEFAULT 1,
		allow_insecure INTEGER DEFAULT 0
	);

	-- Analytics
	CREATE TABLE IF NOT EXISTS analytics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		zone_id TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		requests INTEGER DEFAULT 0,
		bandwidth INTEGER DEFAULT 0,
		threats INTEGER DEFAULT 0,
		page_views INTEGER DEFAULT 0,
		unique_visits INTEGER DEFAULT 0,
		cache_hits INTEGER DEFAULT 0,
		cache_misses INTEGER DEFAULT 0,
		status_codes TEXT DEFAULT '{}',
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_analytics_zone_time ON analytics(zone_id, timestamp);

	-- Page Rules
	CREATE TABLE IF NOT EXISTS page_rules (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		targets TEXT NOT NULL,
		actions TEXT NOT NULL,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- Transform Rules
	CREATE TABLE IF NOT EXISTS transform_rules (
		id TEXT PRIMARY KEY,
		zone_id TEXT NOT NULL,
		type TEXT NOT NULL,
		expression TEXT NOT NULL,
		action TEXT NOT NULL,
		action_value TEXT,
		priority INTEGER DEFAULT 0,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	-- Users
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		name TEXT,
		password_hash TEXT NOT NULL,
		role TEXT DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Sessions
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);

	-- Durable Object Namespaces
	CREATE TABLE IF NOT EXISTS durable_object_namespaces (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		script TEXT NOT NULL,
		class_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Durable Object Instances
	CREATE TABLE IF NOT EXISTS durable_object_instances (
		id TEXT PRIMARY KEY,
		namespace_id TEXT NOT NULL,
		name TEXT,
		has_storage INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_access DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (namespace_id) REFERENCES durable_object_namespaces(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_do_instances_namespace ON durable_object_instances(namespace_id);

	-- Durable Object Storage
	CREATE TABLE IF NOT EXISTS durable_object_storage (
		object_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value BLOB,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (object_id, key)
	);

	-- Durable Object Alarms
	CREATE TABLE IF NOT EXISTS durable_object_alarms (
		object_id TEXT PRIMARY KEY,
		scheduled_time DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_do_alarms_time ON durable_object_alarms(scheduled_time);

	-- Queues
	CREATE TABLE IF NOT EXISTS queues (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		settings TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Queue Messages
	CREATE TABLE IF NOT EXISTS queue_messages (
		id TEXT PRIMARY KEY,
		queue_id TEXT NOT NULL,
		body BLOB NOT NULL,
		content_type TEXT DEFAULT 'json',
		attempts INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		visible_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (queue_id) REFERENCES queues(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_queue_messages_queue ON queue_messages(queue_id);
	CREATE INDEX IF NOT EXISTS idx_queue_messages_visible ON queue_messages(queue_id, visible_at, expires_at);

	-- Queue Consumers
	CREATE TABLE IF NOT EXISTS queue_consumers (
		id TEXT PRIMARY KEY,
		queue_id TEXT NOT NULL,
		script_name TEXT NOT NULL,
		type TEXT DEFAULT 'worker',
		max_batch_size INTEGER DEFAULT 10,
		max_batch_timeout INTEGER DEFAULT 30,
		max_retries INTEGER DEFAULT 3,
		dead_letter_queue TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (queue_id) REFERENCES queues(id) ON DELETE CASCADE
	);

	-- Vector Indexes
	CREATE TABLE IF NOT EXISTS vector_indexes (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		dimensions INTEGER NOT NULL,
		metric TEXT DEFAULT 'cosine',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		vector_count INTEGER DEFAULT 0
	);

	-- Vectors
	CREATE TABLE IF NOT EXISTS vectors (
		id TEXT NOT NULL,
		index_id TEXT NOT NULL,
		namespace TEXT,
		values_json TEXT NOT NULL,
		metadata TEXT DEFAULT '{}',
		PRIMARY KEY (index_id, id),
		FOREIGN KEY (index_id) REFERENCES vector_indexes(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_vectors_namespace ON vectors(index_id, namespace);

	-- Analytics Engine Datasets
	CREATE TABLE IF NOT EXISTS analytics_engine_datasets (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Analytics Engine Data Points
	CREATE TABLE IF NOT EXISTS analytics_engine_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dataset TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		indexes TEXT DEFAULT '[]',
		doubles TEXT DEFAULT '[]',
		blobs TEXT DEFAULT '[]'
	);
	CREATE INDEX IF NOT EXISTS idx_ae_data_dataset ON analytics_engine_data(dataset);
	CREATE INDEX IF NOT EXISTS idx_ae_data_timestamp ON analytics_engine_data(dataset, timestamp);

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

	-- AI Gateway Cache
	CREATE TABLE IF NOT EXISTS ai_gateway_cache (
		gateway_id TEXT NOT NULL,
		cache_key TEXT NOT NULL,
		response BLOB NOT NULL,
		expires_at DATETIME NOT NULL,
		PRIMARY KEY (gateway_id, cache_key)
	);

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

	-- Cron Triggers
	CREATE TABLE IF NOT EXISTS cron_triggers (
		id TEXT PRIMARY KEY,
		script_name TEXT NOT NULL,
		cron TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_cron_triggers_script ON cron_triggers(script_name);

	-- Cron Executions
	CREATE TABLE IF NOT EXISTS cron_executions (
		id TEXT PRIMARY KEY,
		trigger_id TEXT NOT NULL,
		scheduled_at DATETIME NOT NULL,
		started_at DATETIME NOT NULL,
		finished_at DATETIME,
		status TEXT NOT NULL,
		error TEXT,
		FOREIGN KEY (trigger_id) REFERENCES cron_triggers(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_cron_executions_trigger ON cron_executions(trigger_id);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Store interface implementations
func (s *Store) Zones() store.ZoneStore                     { return s.zones }
func (s *Store) DNS() store.DNSStore                        { return s.dns }
func (s *Store) SSL() store.SSLStore                        { return s.ssl }
func (s *Store) Firewall() store.FirewallStore              { return s.firewall }
func (s *Store) Cache() store.CacheStore                    { return s.cache }
func (s *Store) Workers() store.WorkerStore                 { return s.workers }
func (s *Store) KV() store.KVStore                          { return s.kv }
func (s *Store) R2() store.R2Store                          { return s.r2 }
func (s *Store) D1() store.D1Store                          { return s.d1 }
func (s *Store) LoadBalancer() store.LoadBalancerStore      { return s.loadBalancer }
func (s *Store) Analytics() store.AnalyticsStore            { return s.analytics }
func (s *Store) Rules() store.RulesStore                    { return s.rules }
func (s *Store) Users() store.UserStore                     { return s.users }
func (s *Store) DurableObjects() store.DurableObjectStore   { return s.durableObjects }
func (s *Store) Queues() store.QueueStore                   { return s.queues }
func (s *Store) Vectorize() store.VectorizeStore            { return s.vectorize }
func (s *Store) AnalyticsEngine() store.AnalyticsEngineStore { return s.analyticsEngine }
func (s *Store) AI() store.AIStore                          { return s.ai }
func (s *Store) AIGateway() store.AIGatewayStore            { return s.aiGateway }
func (s *Store) Hyperdrive() store.HyperdriveStore          { return s.hyperdrive }
func (s *Store) Cron() store.CronStore                      { return s.cron }

// ZoneStore implementation
type ZoneStore struct {
	db *sql.DB
}

func (s *ZoneStore) Create(ctx context.Context, zone *store.Zone) error {
	ns, _ := json.Marshal(zone.NameServers)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO zones (id, name, status, plan, name_servers, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		zone.ID, zone.Name, zone.Status, zone.Plan, string(ns), zone.CreatedAt, zone.UpdatedAt)
	return err
}

func (s *ZoneStore) GetByID(ctx context.Context, id string) (*store.Zone, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, status, plan, name_servers, created_at, updated_at FROM zones WHERE id = ?`, id)
	return s.scanZone(row)
}

func (s *ZoneStore) GetByName(ctx context.Context, name string) (*store.Zone, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, status, plan, name_servers, created_at, updated_at FROM zones WHERE name = ?`, name)
	return s.scanZone(row)
}

func (s *ZoneStore) List(ctx context.Context) ([]*store.Zone, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, status, plan, name_servers, created_at, updated_at FROM zones ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []*store.Zone
	for rows.Next() {
		zone, err := s.scanZone(rows)
		if err != nil {
			return nil, err
		}
		zones = append(zones, zone)
	}
	return zones, rows.Err()
}

func (s *ZoneStore) Update(ctx context.Context, zone *store.Zone) error {
	ns, _ := json.Marshal(zone.NameServers)
	_, err := s.db.ExecContext(ctx,
		`UPDATE zones SET name = ?, status = ?, plan = ?, name_servers = ?, updated_at = ? WHERE id = ?`,
		zone.Name, zone.Status, zone.Plan, string(ns), time.Now(), zone.ID)
	return err
}

func (s *ZoneStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM zones WHERE id = ?`, id)
	return err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (s *ZoneStore) scanZone(row scanner) (*store.Zone, error) {
	var zone store.Zone
	var ns string
	if err := row.Scan(&zone.ID, &zone.Name, &zone.Status, &zone.Plan, &ns, &zone.CreatedAt, &zone.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(ns), &zone.NameServers)
	return &zone, nil
}

// DNSStoreImpl implementation
type DNSStoreImpl struct {
	db *sql.DB
}

func (s *DNSStoreImpl) Create(ctx context.Context, record *store.DNSRecord) error {
	tags, _ := json.Marshal(record.Tags)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO dns_records (id, zone_id, type, name, content, ttl, priority, proxied, comment, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.ZoneID, record.Type, record.Name, record.Content, record.TTL,
		record.Priority, record.Proxied, record.Comment, string(tags), record.CreatedAt, record.UpdatedAt)
	return err
}

func (s *DNSStoreImpl) GetByID(ctx context.Context, id string) (*store.DNSRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, zone_id, type, name, content, ttl, priority, proxied, comment, tags, created_at, updated_at
		FROM dns_records WHERE id = ?`, id)
	return s.scanRecord(row)
}

func (s *DNSStoreImpl) ListByZone(ctx context.Context, zoneID string) ([]*store.DNSRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, type, name, content, ttl, priority, proxied, comment, tags, created_at, updated_at
		FROM dns_records WHERE zone_id = ? ORDER BY type, name`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*store.DNSRecord
	for rows.Next() {
		record, err := s.scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *DNSStoreImpl) Update(ctx context.Context, record *store.DNSRecord) error {
	tags, _ := json.Marshal(record.Tags)
	_, err := s.db.ExecContext(ctx,
		`UPDATE dns_records SET type = ?, name = ?, content = ?, ttl = ?, priority = ?,
		proxied = ?, comment = ?, tags = ?, updated_at = ? WHERE id = ?`,
		record.Type, record.Name, record.Content, record.TTL, record.Priority,
		record.Proxied, record.Comment, string(tags), time.Now(), record.ID)
	return err
}

func (s *DNSStoreImpl) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dns_records WHERE id = ?`, id)
	return err
}

func (s *DNSStoreImpl) scanRecord(row scanner) (*store.DNSRecord, error) {
	var record store.DNSRecord
	var tags string
	if err := row.Scan(&record.ID, &record.ZoneID, &record.Type, &record.Name, &record.Content,
		&record.TTL, &record.Priority, &record.Proxied, &record.Comment, &tags,
		&record.CreatedAt, &record.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(tags), &record.Tags)
	return &record, nil
}

// SSLStoreImpl implementation
type SSLStoreImpl struct {
	db *sql.DB
}

func (s *SSLStoreImpl) CreateCertificate(ctx context.Context, cert *store.Certificate) error {
	hosts, _ := json.Marshal(cert.Hosts)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO certificates (id, zone_id, type, hosts, issuer, serial_number, signature, status, expires_at, certificate, private_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cert.ID, cert.ZoneID, cert.Type, string(hosts), cert.Issuer, cert.SerialNum,
		cert.Signature, cert.Status, cert.ExpiresAt, cert.Certificate, cert.PrivateKey, cert.CreatedAt)
	return err
}

func (s *SSLStoreImpl) GetCertificate(ctx context.Context, id string) (*store.Certificate, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, zone_id, type, hosts, issuer, serial_number, signature, status, expires_at, certificate, private_key, created_at
		FROM certificates WHERE id = ?`, id)
	var cert store.Certificate
	var hosts string
	if err := row.Scan(&cert.ID, &cert.ZoneID, &cert.Type, &hosts, &cert.Issuer, &cert.SerialNum,
		&cert.Signature, &cert.Status, &cert.ExpiresAt, &cert.Certificate, &cert.PrivateKey, &cert.CreatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(hosts), &cert.Hosts)
	return &cert, nil
}

func (s *SSLStoreImpl) ListCertificates(ctx context.Context, zoneID string) ([]*store.Certificate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, type, hosts, issuer, serial_number, signature, status, expires_at, certificate, private_key, created_at
		FROM certificates WHERE zone_id = ?`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*store.Certificate
	for rows.Next() {
		var cert store.Certificate
		var hosts string
		if err := rows.Scan(&cert.ID, &cert.ZoneID, &cert.Type, &hosts, &cert.Issuer, &cert.SerialNum,
			&cert.Signature, &cert.Status, &cert.ExpiresAt, &cert.Certificate, &cert.PrivateKey, &cert.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(hosts), &cert.Hosts)
		certs = append(certs, &cert)
	}
	return certs, rows.Err()
}

func (s *SSLStoreImpl) DeleteCertificate(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM certificates WHERE id = ?`, id)
	return err
}

func (s *SSLStoreImpl) GetSettings(ctx context.Context, zoneID string) (*store.SSLSettings, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT zone_id, mode, always_https, min_tls_version, opportunistic_encryption, tls_1_3, automatic_https_rewrites
		FROM ssl_settings WHERE zone_id = ?`, zoneID)
	var settings store.SSLSettings
	if err := row.Scan(&settings.ZoneID, &settings.Mode, &settings.AlwaysHTTPS, &settings.MinTLSVersion,
		&settings.OpportunisticEncryption, &settings.TLS13, &settings.AutomaticHTTPSRewrites); err != nil {
		if err == sql.ErrNoRows {
			return &store.SSLSettings{ZoneID: zoneID, Mode: "full", MinTLSVersion: "1.2", TLS13: true, AutomaticHTTPSRewrites: true}, nil
		}
		return nil, err
	}
	return &settings, nil
}

func (s *SSLStoreImpl) UpdateSettings(ctx context.Context, settings *store.SSLSettings) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO ssl_settings (zone_id, mode, always_https, min_tls_version, opportunistic_encryption, tls_1_3, automatic_https_rewrites)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		settings.ZoneID, settings.Mode, settings.AlwaysHTTPS, settings.MinTLSVersion,
		settings.OpportunisticEncryption, settings.TLS13, settings.AutomaticHTTPSRewrites)
	return err
}

// FirewallStoreImpl implementation
type FirewallStoreImpl struct {
	db *sql.DB
}

func (s *FirewallStoreImpl) CreateRule(ctx context.Context, rule *store.FirewallRule) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO firewall_rules (id, zone_id, description, expression, action, priority, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.ZoneID, rule.Description, rule.Expression, rule.Action, rule.Priority, rule.Enabled, rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (s *FirewallStoreImpl) GetRule(ctx context.Context, id string) (*store.FirewallRule, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, zone_id, description, expression, action, priority, enabled, created_at, updated_at
		FROM firewall_rules WHERE id = ?`, id)
	var rule store.FirewallRule
	if err := row.Scan(&rule.ID, &rule.ZoneID, &rule.Description, &rule.Expression, &rule.Action,
		&rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *FirewallStoreImpl) ListRules(ctx context.Context, zoneID string) ([]*store.FirewallRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, description, expression, action, priority, enabled, created_at, updated_at
		FROM firewall_rules WHERE zone_id = ? ORDER BY priority`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*store.FirewallRule
	for rows.Next() {
		var rule store.FirewallRule
		if err := rows.Scan(&rule.ID, &rule.ZoneID, &rule.Description, &rule.Expression, &rule.Action,
			&rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (s *FirewallStoreImpl) UpdateRule(ctx context.Context, rule *store.FirewallRule) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE firewall_rules SET description = ?, expression = ?, action = ?, priority = ?, enabled = ?, updated_at = ?
		WHERE id = ?`,
		rule.Description, rule.Expression, rule.Action, rule.Priority, rule.Enabled, time.Now(), rule.ID)
	return err
}

func (s *FirewallStoreImpl) DeleteRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM firewall_rules WHERE id = ?`, id)
	return err
}

func (s *FirewallStoreImpl) CreateIPAccessRule(ctx context.Context, rule *store.IPAccessRule) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ip_access_rules (id, zone_id, mode, target, value, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.ZoneID, rule.Mode, rule.Target, rule.Value, rule.Notes, rule.CreatedAt)
	return err
}

func (s *FirewallStoreImpl) ListIPAccessRules(ctx context.Context, zoneID string) ([]*store.IPAccessRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, mode, target, value, notes, created_at FROM ip_access_rules WHERE zone_id = ?`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*store.IPAccessRule
	for rows.Next() {
		var rule store.IPAccessRule
		if err := rows.Scan(&rule.ID, &rule.ZoneID, &rule.Mode, &rule.Target, &rule.Value, &rule.Notes, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (s *FirewallStoreImpl) DeleteIPAccessRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ip_access_rules WHERE id = ?`, id)
	return err
}

func (s *FirewallStoreImpl) CreateRateLimitRule(ctx context.Context, rule *store.RateLimitRule) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO rate_limit_rules (id, zone_id, description, expression, threshold, period, action, action_timeout, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.ZoneID, rule.Description, rule.Expression, rule.Threshold, rule.Period,
		rule.Action, rule.ActionTimeout, rule.Enabled, rule.CreatedAt)
	return err
}

func (s *FirewallStoreImpl) ListRateLimitRules(ctx context.Context, zoneID string) ([]*store.RateLimitRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, description, expression, threshold, period, action, action_timeout, enabled, created_at
		FROM rate_limit_rules WHERE zone_id = ?`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*store.RateLimitRule
	for rows.Next() {
		var rule store.RateLimitRule
		if err := rows.Scan(&rule.ID, &rule.ZoneID, &rule.Description, &rule.Expression, &rule.Threshold,
			&rule.Period, &rule.Action, &rule.ActionTimeout, &rule.Enabled, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

// CacheStoreImpl implementation
type CacheStoreImpl struct {
	db *sql.DB
}

func (s *CacheStoreImpl) GetSettings(ctx context.Context, zoneID string) (*store.CacheSettings, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT zone_id, cache_level, browser_ttl, edge_ttl, development_mode, always_online
		FROM cache_settings WHERE zone_id = ?`, zoneID)
	var settings store.CacheSettings
	if err := row.Scan(&settings.ZoneID, &settings.CacheLevel, &settings.BrowserTTL,
		&settings.EdgeTTL, &settings.DevelopmentMode, &settings.AlwaysOnline); err != nil {
		if err == sql.ErrNoRows {
			return &store.CacheSettings{ZoneID: zoneID, CacheLevel: "standard", BrowserTTL: 14400, EdgeTTL: 7200, AlwaysOnline: true}, nil
		}
		return nil, err
	}
	return &settings, nil
}

func (s *CacheStoreImpl) UpdateSettings(ctx context.Context, settings *store.CacheSettings) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO cache_settings (zone_id, cache_level, browser_ttl, edge_ttl, development_mode, always_online)
		VALUES (?, ?, ?, ?, ?, ?)`,
		settings.ZoneID, settings.CacheLevel, settings.BrowserTTL, settings.EdgeTTL, settings.DevelopmentMode, settings.AlwaysOnline)
	return err
}

func (s *CacheStoreImpl) CreateRule(ctx context.Context, rule *store.CacheRule) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cache_rules (id, zone_id, name, expression, cache_level, edge_ttl, browser_ttl, bypass_cache, priority, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.ZoneID, rule.Name, rule.Expression, rule.CacheLevel, rule.EdgeTTL,
		rule.BrowserTTL, rule.BypassCache, rule.Priority, rule.Enabled, rule.CreatedAt)
	return err
}

func (s *CacheStoreImpl) ListRules(ctx context.Context, zoneID string) ([]*store.CacheRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, name, expression, cache_level, edge_ttl, browser_ttl, bypass_cache, priority, enabled, created_at
		FROM cache_rules WHERE zone_id = ? ORDER BY priority`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*store.CacheRule
	for rows.Next() {
		var rule store.CacheRule
		if err := rows.Scan(&rule.ID, &rule.ZoneID, &rule.Name, &rule.Expression, &rule.CacheLevel,
			&rule.EdgeTTL, &rule.BrowserTTL, &rule.BypassCache, &rule.Priority, &rule.Enabled, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (s *CacheStoreImpl) DeleteRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cache_rules WHERE id = ?`, id)
	return err
}

func (s *CacheStoreImpl) PurgeAll(ctx context.Context, zoneID string) error {
	// In a real implementation, this would clear the actual cache
	return nil
}

func (s *CacheStoreImpl) PurgeURLs(ctx context.Context, zoneID string, urls []string) error {
	// In a real implementation, this would clear specific cached URLs
	return nil
}

// WorkerStoreImpl implementation
type WorkerStoreImpl struct {
	db *sql.DB
}

func (s *WorkerStoreImpl) Create(ctx context.Context, worker *store.Worker) error {
	routes, _ := json.Marshal(worker.Routes)
	bindings, _ := json.Marshal(worker.Bindings)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workers (id, name, script, routes, bindings, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		worker.ID, worker.Name, worker.Script, string(routes), string(bindings), worker.Enabled, worker.CreatedAt, worker.UpdatedAt)
	return err
}

func (s *WorkerStoreImpl) GetByID(ctx context.Context, id string) (*store.Worker, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, script, routes, bindings, enabled, created_at, updated_at FROM workers WHERE id = ?`, id)
	return s.scanWorker(row)
}

func (s *WorkerStoreImpl) GetByName(ctx context.Context, name string) (*store.Worker, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, script, routes, bindings, enabled, created_at, updated_at FROM workers WHERE name = ?`, name)
	return s.scanWorker(row)
}

func (s *WorkerStoreImpl) List(ctx context.Context) ([]*store.Worker, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, script, routes, bindings, enabled, created_at, updated_at FROM workers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*store.Worker
	for rows.Next() {
		worker, err := s.scanWorker(rows)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}
	return workers, rows.Err()
}

func (s *WorkerStoreImpl) Update(ctx context.Context, worker *store.Worker) error {
	routes, _ := json.Marshal(worker.Routes)
	bindings, _ := json.Marshal(worker.Bindings)
	_, err := s.db.ExecContext(ctx,
		`UPDATE workers SET name = ?, script = ?, routes = ?, bindings = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		worker.Name, worker.Script, string(routes), string(bindings), worker.Enabled, time.Now(), worker.ID)
	return err
}

func (s *WorkerStoreImpl) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workers WHERE id = ?`, id)
	return err
}

func (s *WorkerStoreImpl) CreateRoute(ctx context.Context, route *store.WorkerRoute) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO worker_routes (id, zone_id, pattern, worker_id, enabled) VALUES (?, ?, ?, ?, ?)`,
		route.ID, route.ZoneID, route.Pattern, route.WorkerID, route.Enabled)
	return err
}

func (s *WorkerStoreImpl) ListRoutes(ctx context.Context, zoneID string) ([]*store.WorkerRoute, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, pattern, worker_id, enabled FROM worker_routes WHERE zone_id = ?`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []*store.WorkerRoute
	for rows.Next() {
		var route store.WorkerRoute
		if err := rows.Scan(&route.ID, &route.ZoneID, &route.Pattern, &route.WorkerID, &route.Enabled); err != nil {
			return nil, err
		}
		routes = append(routes, &route)
	}
	return routes, rows.Err()
}

func (s *WorkerStoreImpl) DeleteRoute(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM worker_routes WHERE id = ?`, id)
	return err
}

func (s *WorkerStoreImpl) scanWorker(row scanner) (*store.Worker, error) {
	var worker store.Worker
	var routes, bindings string
	if err := row.Scan(&worker.ID, &worker.Name, &worker.Script, &routes, &bindings, &worker.Enabled, &worker.CreatedAt, &worker.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(routes), &worker.Routes)
	json.Unmarshal([]byte(bindings), &worker.Bindings)
	return &worker, nil
}

// KVStoreImpl implementation
type KVStoreImpl struct {
	db *sql.DB
}

func (s *KVStoreImpl) CreateNamespace(ctx context.Context, ns *store.KVNamespace) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kv_namespaces (id, title, created_at) VALUES (?, ?, ?)`,
		ns.ID, ns.Title, ns.CreatedAt)
	return err
}

func (s *KVStoreImpl) GetNamespace(ctx context.Context, id string) (*store.KVNamespace, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, title, created_at FROM kv_namespaces WHERE id = ?`, id)
	var ns store.KVNamespace
	if err := row.Scan(&ns.ID, &ns.Title, &ns.CreatedAt); err != nil {
		return nil, err
	}
	return &ns, nil
}

func (s *KVStoreImpl) ListNamespaces(ctx context.Context) ([]*store.KVNamespace, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, created_at FROM kv_namespaces ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var namespaces []*store.KVNamespace
	for rows.Next() {
		var ns store.KVNamespace
		if err := rows.Scan(&ns.ID, &ns.Title, &ns.CreatedAt); err != nil {
			return nil, err
		}
		namespaces = append(namespaces, &ns)
	}
	return namespaces, rows.Err()
}

func (s *KVStoreImpl) DeleteNamespace(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM kv_namespaces WHERE id = ?`, id)
	return err
}

func (s *KVStoreImpl) Put(ctx context.Context, nsID string, pair *store.KVPair) error {
	metadata, _ := json.Marshal(pair.Metadata)
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO kv_pairs (namespace_id, key, value, metadata, expiration) VALUES (?, ?, ?, ?, ?)`,
		nsID, pair.Key, pair.Value, string(metadata), pair.Expiration)
	return err
}

func (s *KVStoreImpl) Get(ctx context.Context, nsID, key string) (*store.KVPair, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT key, value, metadata, expiration FROM kv_pairs WHERE namespace_id = ? AND key = ?`, nsID, key)
	var pair store.KVPair
	var metadata string
	if err := row.Scan(&pair.Key, &pair.Value, &metadata, &pair.Expiration); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(metadata), &pair.Metadata)
	return &pair, nil
}

func (s *KVStoreImpl) Delete(ctx context.Context, nsID, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM kv_pairs WHERE namespace_id = ? AND key = ?`, nsID, key)
	return err
}

func (s *KVStoreImpl) List(ctx context.Context, nsID, prefix string, limit int) ([]*store.KVPair, error) {
	query := `SELECT key, value, metadata, expiration FROM kv_pairs WHERE namespace_id = ?`
	args := []interface{}{nsID}
	if prefix != "" {
		query += ` AND key LIKE ?`
		args = append(args, prefix+"%")
	}
	query += ` ORDER BY key`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []*store.KVPair
	for rows.Next() {
		var pair store.KVPair
		var metadata string
		if err := rows.Scan(&pair.Key, &pair.Value, &metadata, &pair.Expiration); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(metadata), &pair.Metadata)
		pairs = append(pairs, &pair)
	}
	return pairs, rows.Err()
}

// R2StoreImpl implementation
type R2StoreImpl struct {
	db      *sql.DB
	dataDir string
}

func (s *R2StoreImpl) CreateBucket(ctx context.Context, bucket *store.R2Bucket) error {
	bucketDir := filepath.Join(s.dataDir, bucket.ID)
	if err := os.MkdirAll(bucketDir, 0755); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO r2_buckets (id, name, location, created_at) VALUES (?, ?, ?, ?)`,
		bucket.ID, bucket.Name, bucket.Location, bucket.CreatedAt)
	return err
}

func (s *R2StoreImpl) GetBucket(ctx context.Context, id string) (*store.R2Bucket, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, location, created_at FROM r2_buckets WHERE id = ?`, id)
	var bucket store.R2Bucket
	if err := row.Scan(&bucket.ID, &bucket.Name, &bucket.Location, &bucket.CreatedAt); err != nil {
		return nil, err
	}
	return &bucket, nil
}

func (s *R2StoreImpl) ListBuckets(ctx context.Context) ([]*store.R2Bucket, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, location, created_at FROM r2_buckets ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []*store.R2Bucket
	for rows.Next() {
		var bucket store.R2Bucket
		if err := rows.Scan(&bucket.ID, &bucket.Name, &bucket.Location, &bucket.CreatedAt); err != nil {
			return nil, err
		}
		buckets = append(buckets, &bucket)
	}
	return buckets, rows.Err()
}

func (s *R2StoreImpl) DeleteBucket(ctx context.Context, id string) error {
	bucketDir := filepath.Join(s.dataDir, id)
	os.RemoveAll(bucketDir)
	_, err := s.db.ExecContext(ctx, `DELETE FROM r2_buckets WHERE id = ?`, id)
	return err
}

func (s *R2StoreImpl) PutObject(ctx context.Context, bucketID, key string, data []byte, metadata map[string]string) error {
	// Ensure bucket directory exists
	objectPath := filepath.Join(s.dataDir, bucketID, key)
	if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
		return err
	}

	// Write data to file
	if err := os.WriteFile(objectPath, data, 0644); err != nil {
		return err
	}

	// Update metadata in database
	meta, _ := json.Marshal(metadata)
	contentType := "application/octet-stream"
	if ct, ok := metadata["content-type"]; ok {
		contentType = ct
	}
	etag := fmt.Sprintf("%x", len(data)) // Simple ETag

	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO r2_objects (bucket_id, key, size, etag, content_type, metadata, last_modified)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		bucketID, key, len(data), etag, contentType, string(meta))
	return err
}

func (s *R2StoreImpl) GetObject(ctx context.Context, bucketID, key string) ([]byte, *store.R2Object, error) {
	// Get metadata
	row := s.db.QueryRowContext(ctx,
		`SELECT key, size, etag, content_type, metadata, last_modified FROM r2_objects WHERE bucket_id = ? AND key = ?`,
		bucketID, key)
	var obj store.R2Object
	var meta string
	if err := row.Scan(&obj.Key, &obj.Size, &obj.ETag, &obj.ContentType, &meta, &obj.LastModified); err != nil {
		return nil, nil, err
	}
	json.Unmarshal([]byte(meta), &obj.Metadata)

	// Read data from file
	objectPath := filepath.Join(s.dataDir, bucketID, key)
	data, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, nil, err
	}

	return data, &obj, nil
}

func (s *R2StoreImpl) DeleteObject(ctx context.Context, bucketID, key string) error {
	objectPath := filepath.Join(s.dataDir, bucketID, key)
	os.Remove(objectPath)
	_, err := s.db.ExecContext(ctx, `DELETE FROM r2_objects WHERE bucket_id = ? AND key = ?`, bucketID, key)
	return err
}

func (s *R2StoreImpl) ListObjects(ctx context.Context, bucketID, prefix, delimiter string, limit int) ([]*store.R2Object, error) {
	query := `SELECT key, size, etag, content_type, metadata, last_modified FROM r2_objects WHERE bucket_id = ?`
	args := []interface{}{bucketID}
	if prefix != "" {
		query += ` AND key LIKE ?`
		args = append(args, prefix+"%")
	}
	query += ` ORDER BY key`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []*store.R2Object
	for rows.Next() {
		var obj store.R2Object
		var meta string
		if err := rows.Scan(&obj.Key, &obj.Size, &obj.ETag, &obj.ContentType, &meta, &obj.LastModified); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(meta), &obj.Metadata)
		objects = append(objects, &obj)
	}
	return objects, rows.Err()
}

// D1StoreImpl implementation
type D1StoreImpl struct {
	db      *sql.DB
	dataDir string
}

func (s *D1StoreImpl) CreateDatabase(ctx context.Context, database *store.D1Database) error {
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	// Create the actual SQLite database file
	dbPath := filepath.Join(s.dataDir, database.ID+".db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	db.Close()

	// Store metadata
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO d1_databases (id, name, version, num_tables, file_size, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		database.ID, database.Name, database.Version, database.NumTables, database.FileSize, database.CreatedAt)
	return err
}

func (s *D1StoreImpl) GetDatabase(ctx context.Context, id string) (*store.D1Database, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, version, num_tables, file_size, created_at FROM d1_databases WHERE id = ?`, id)
	var database store.D1Database
	if err := row.Scan(&database.ID, &database.Name, &database.Version, &database.NumTables, &database.FileSize, &database.CreatedAt); err != nil {
		return nil, err
	}
	return &database, nil
}

func (s *D1StoreImpl) ListDatabases(ctx context.Context) ([]*store.D1Database, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, version, num_tables, file_size, created_at FROM d1_databases ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []*store.D1Database
	for rows.Next() {
		var database store.D1Database
		if err := rows.Scan(&database.ID, &database.Name, &database.Version, &database.NumTables, &database.FileSize, &database.CreatedAt); err != nil {
			return nil, err
		}
		databases = append(databases, &database)
	}
	return databases, rows.Err()
}

func (s *D1StoreImpl) DeleteDatabase(ctx context.Context, id string) error {
	dbPath := filepath.Join(s.dataDir, id+".db")
	os.Remove(dbPath)
	_, err := s.db.ExecContext(ctx, `DELETE FROM d1_databases WHERE id = ?`, id)
	return err
}

func (s *D1StoreImpl) Query(ctx context.Context, dbID, sqlQuery string, params []interface{}) ([]map[string]interface{}, error) {
	dbPath := filepath.Join(s.dataDir, dbID+".db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, sqlQuery, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func (s *D1StoreImpl) Exec(ctx context.Context, dbID, sqlQuery string, params []interface{}) (int64, error) {
	dbPath := filepath.Join(s.dataDir, dbID+".db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	result, err := db.ExecContext(ctx, sqlQuery, params...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// LoadBalancerStoreImpl implementation
type LoadBalancerStoreImpl struct {
	db *sql.DB
}

func (s *LoadBalancerStoreImpl) Create(ctx context.Context, lb *store.LoadBalancer) error {
	pools, _ := json.Marshal(lb.DefaultPools)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO load_balancers (id, zone_id, name, fallback_pool, default_pools, session_affinity, steering_policy, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		lb.ID, lb.ZoneID, lb.Name, lb.Fallback, string(pools), lb.SessionAffinity, lb.SteeringPolicy, lb.Enabled, lb.CreatedAt)
	return err
}

func (s *LoadBalancerStoreImpl) GetByID(ctx context.Context, id string) (*store.LoadBalancer, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, zone_id, name, fallback_pool, default_pools, session_affinity, steering_policy, enabled, created_at
		FROM load_balancers WHERE id = ?`, id)
	var lb store.LoadBalancer
	var pools string
	if err := row.Scan(&lb.ID, &lb.ZoneID, &lb.Name, &lb.Fallback, &pools, &lb.SessionAffinity, &lb.SteeringPolicy, &lb.Enabled, &lb.CreatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(pools), &lb.DefaultPools)
	return &lb, nil
}

func (s *LoadBalancerStoreImpl) List(ctx context.Context, zoneID string) ([]*store.LoadBalancer, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, name, fallback_pool, default_pools, session_affinity, steering_policy, enabled, created_at
		FROM load_balancers WHERE zone_id = ?`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lbs []*store.LoadBalancer
	for rows.Next() {
		var lb store.LoadBalancer
		var pools string
		if err := rows.Scan(&lb.ID, &lb.ZoneID, &lb.Name, &lb.Fallback, &pools, &lb.SessionAffinity, &lb.SteeringPolicy, &lb.Enabled, &lb.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(pools), &lb.DefaultPools)
		lbs = append(lbs, &lb)
	}
	return lbs, rows.Err()
}

func (s *LoadBalancerStoreImpl) Update(ctx context.Context, lb *store.LoadBalancer) error {
	pools, _ := json.Marshal(lb.DefaultPools)
	_, err := s.db.ExecContext(ctx,
		`UPDATE load_balancers SET name = ?, fallback_pool = ?, default_pools = ?, session_affinity = ?, steering_policy = ?, enabled = ?
		WHERE id = ?`,
		lb.Name, lb.Fallback, string(pools), lb.SessionAffinity, lb.SteeringPolicy, lb.Enabled, lb.ID)
	return err
}

func (s *LoadBalancerStoreImpl) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM load_balancers WHERE id = ?`, id)
	return err
}

func (s *LoadBalancerStoreImpl) CreatePool(ctx context.Context, pool *store.OriginPool) error {
	origins, _ := json.Marshal(pool.Origins)
	regions, _ := json.Marshal(pool.CheckRegions)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO origin_pools (id, name, origins, check_regions, description, enabled, minimum_origins, monitor, notification_email, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		pool.ID, pool.Name, string(origins), string(regions), pool.Description, pool.Enabled,
		pool.MinOrigins, pool.Monitor, pool.NotifyEmail, pool.CreatedAt)
	return err
}

func (s *LoadBalancerStoreImpl) GetPool(ctx context.Context, id string) (*store.OriginPool, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, origins, check_regions, description, enabled, minimum_origins, monitor, notification_email, created_at
		FROM origin_pools WHERE id = ?`, id)
	var pool store.OriginPool
	var origins, regions string
	if err := row.Scan(&pool.ID, &pool.Name, &origins, &regions, &pool.Description, &pool.Enabled,
		&pool.MinOrigins, &pool.Monitor, &pool.NotifyEmail, &pool.CreatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(origins), &pool.Origins)
	json.Unmarshal([]byte(regions), &pool.CheckRegions)
	return &pool, nil
}

func (s *LoadBalancerStoreImpl) ListPools(ctx context.Context) ([]*store.OriginPool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, origins, check_regions, description, enabled, minimum_origins, monitor, notification_email, created_at
		FROM origin_pools ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*store.OriginPool
	for rows.Next() {
		var pool store.OriginPool
		var origins, regions string
		if err := rows.Scan(&pool.ID, &pool.Name, &origins, &regions, &pool.Description, &pool.Enabled,
			&pool.MinOrigins, &pool.Monitor, &pool.NotifyEmail, &pool.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(origins), &pool.Origins)
		json.Unmarshal([]byte(regions), &pool.CheckRegions)
		pools = append(pools, &pool)
	}
	return pools, rows.Err()
}

func (s *LoadBalancerStoreImpl) UpdatePool(ctx context.Context, pool *store.OriginPool) error {
	origins, _ := json.Marshal(pool.Origins)
	regions, _ := json.Marshal(pool.CheckRegions)
	_, err := s.db.ExecContext(ctx,
		`UPDATE origin_pools SET name = ?, origins = ?, check_regions = ?, description = ?, enabled = ?,
		minimum_origins = ?, monitor = ?, notification_email = ? WHERE id = ?`,
		pool.Name, string(origins), string(regions), pool.Description, pool.Enabled,
		pool.MinOrigins, pool.Monitor, pool.NotifyEmail, pool.ID)
	return err
}

func (s *LoadBalancerStoreImpl) DeletePool(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM origin_pools WHERE id = ?`, id)
	return err
}

func (s *LoadBalancerStoreImpl) CreateHealthCheck(ctx context.Context, hc *store.HealthCheck) error {
	header, _ := json.Marshal(hc.Header)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO health_checks (id, description, type, method, path, header, port, timeout, retries, interval_seconds, expected_body, expected_codes, follow_redirects, allow_insecure)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hc.ID, hc.Description, hc.Type, hc.Method, hc.Path, string(header), hc.Port, hc.Timeout,
		hc.Retries, hc.Interval, hc.ExpectedBody, hc.ExpectedCodes, hc.FollowRedirects, hc.AllowInsecure)
	return err
}

func (s *LoadBalancerStoreImpl) ListHealthChecks(ctx context.Context) ([]*store.HealthCheck, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, description, type, method, path, header, port, timeout, retries, interval_seconds, expected_body, expected_codes, follow_redirects, allow_insecure
		FROM health_checks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []*store.HealthCheck
	for rows.Next() {
		var hc store.HealthCheck
		var header string
		if err := rows.Scan(&hc.ID, &hc.Description, &hc.Type, &hc.Method, &hc.Path, &header, &hc.Port, &hc.Timeout,
			&hc.Retries, &hc.Interval, &hc.ExpectedBody, &hc.ExpectedCodes, &hc.FollowRedirects, &hc.AllowInsecure); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(header), &hc.Header)
		checks = append(checks, &hc)
	}
	return checks, rows.Err()
}

// AnalyticsStoreImpl implementation
type AnalyticsStoreImpl struct {
	db *sql.DB
}

func (s *AnalyticsStoreImpl) Record(ctx context.Context, zoneID string, data *store.AnalyticsData) error {
	codes, _ := json.Marshal(data.StatusCodes)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO analytics (zone_id, timestamp, requests, bandwidth, threats, page_views, unique_visits, cache_hits, cache_misses, status_codes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		zoneID, data.Timestamp, data.Requests, data.Bandwidth, data.Threats, data.PageViews,
		data.UniqueVisits, data.CacheHits, data.CacheMisses, string(codes))
	return err
}

func (s *AnalyticsStoreImpl) Query(ctx context.Context, zoneID string, start, end time.Time) ([]*store.AnalyticsData, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT timestamp, requests, bandwidth, threats, page_views, unique_visits, cache_hits, cache_misses, status_codes
		FROM analytics WHERE zone_id = ? AND timestamp BETWEEN ? AND ? ORDER BY timestamp`,
		zoneID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*store.AnalyticsData
	for rows.Next() {
		var data store.AnalyticsData
		var codes string
		if err := rows.Scan(&data.Timestamp, &data.Requests, &data.Bandwidth, &data.Threats, &data.PageViews,
			&data.UniqueVisits, &data.CacheHits, &data.CacheMisses, &codes); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(codes), &data.StatusCodes)
		results = append(results, &data)
	}
	return results, rows.Err()
}

func (s *AnalyticsStoreImpl) GetSummary(ctx context.Context, zoneID string, period string) (*store.AnalyticsData, error) {
	var since time.Time
	switch period {
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	default:
		since = time.Now().Add(-24 * time.Hour)
	}

	row := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(requests), 0), COALESCE(SUM(bandwidth), 0), COALESCE(SUM(threats), 0),
		COALESCE(SUM(page_views), 0), COALESCE(SUM(unique_visits), 0), COALESCE(SUM(cache_hits), 0), COALESCE(SUM(cache_misses), 0)
		FROM analytics WHERE zone_id = ? AND timestamp >= ?`,
		zoneID, since)

	var data store.AnalyticsData
	if err := row.Scan(&data.Requests, &data.Bandwidth, &data.Threats, &data.PageViews,
		&data.UniqueVisits, &data.CacheHits, &data.CacheMisses); err != nil {
		return nil, err
	}
	return &data, nil
}

// RulesStoreImpl implementation
type RulesStoreImpl struct {
	db *sql.DB
}

func (s *RulesStoreImpl) CreatePageRule(ctx context.Context, rule *store.PageRule) error {
	targets, _ := json.Marshal(rule.Targets)
	actions, _ := json.Marshal(rule.Actions)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO page_rules (id, zone_id, targets, actions, priority, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.ZoneID, string(targets), string(actions), rule.Priority, rule.Status, rule.CreatedAt)
	return err
}

func (s *RulesStoreImpl) ListPageRules(ctx context.Context, zoneID string) ([]*store.PageRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, targets, actions, priority, status, created_at FROM page_rules WHERE zone_id = ? ORDER BY priority`,
		zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*store.PageRule
	for rows.Next() {
		var rule store.PageRule
		var targets, actions string
		if err := rows.Scan(&rule.ID, &rule.ZoneID, &targets, &actions, &rule.Priority, &rule.Status, &rule.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(targets), &rule.Targets)
		json.Unmarshal([]byte(actions), &rule.Actions)
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (s *RulesStoreImpl) DeletePageRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM page_rules WHERE id = ?`, id)
	return err
}

func (s *RulesStoreImpl) CreateTransformRule(ctx context.Context, rule *store.TransformRule) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO transform_rules (id, zone_id, type, expression, action, action_value, priority, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.ZoneID, rule.Type, rule.Expression, rule.Action, rule.ActionValue, rule.Priority, rule.Enabled, rule.CreatedAt)
	return err
}

func (s *RulesStoreImpl) ListTransformRules(ctx context.Context, zoneID string) ([]*store.TransformRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, zone_id, type, expression, action, action_value, priority, enabled, created_at
		FROM transform_rules WHERE zone_id = ? ORDER BY priority`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*store.TransformRule
	for rows.Next() {
		var rule store.TransformRule
		if err := rows.Scan(&rule.ID, &rule.ZoneID, &rule.Type, &rule.Expression, &rule.Action,
			&rule.ActionValue, &rule.Priority, &rule.Enabled, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (s *RulesStoreImpl) DeleteTransformRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM transform_rules WHERE id = ?`, id)
	return err
}

// UserStoreImpl implementation
type UserStoreImpl struct {
	db *sql.DB
}

func (s *UserStoreImpl) Create(ctx context.Context, user *store.User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.Name, user.PasswordHash, user.Role, user.CreatedAt)
	return err
}

func (s *UserStoreImpl) GetByID(ctx context.Context, id string) (*store.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE id = ?`, id)
	var user store.User
	if err := row.Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStoreImpl) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE email = ?`, email)
	var user store.User
	if err := row.Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStoreImpl) Update(ctx context.Context, user *store.User) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET email = ?, name = ?, password_hash = ?, role = ? WHERE id = ?`,
		user.Email, user.Name, user.PasswordHash, user.Role, user.ID)
	return err
}

func (s *UserStoreImpl) CreateSession(ctx context.Context, session *store.Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt)
	return err
}

func (s *UserStoreImpl) GetSession(ctx context.Context, token string) (*store.Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, token, expires_at, created_at FROM sessions WHERE token = ?`, token)
	var session store.Session
	if err := row.Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *UserStoreImpl) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}
