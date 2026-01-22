package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-mizu/blueprints/bi/store"
)

// Store implements the store.Store interface using SQLite.
type Store struct {
	db      *sql.DB
	dataDir string

	datasources   *DataSourceStore
	tables        *TableStore
	questions     *QuestionStore
	dashboards    *DashboardStore
	collections   *CollectionStore
	models        *ModelStore
	metrics       *MetricStore
	alerts        *AlertStore
	subscriptions *SubscriptionStore
	users         *UserStore
	settings      *SettingsStore
	queryHistory  *QueryHistoryStore
}

// New creates a new SQLite store.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "bi.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{
		db:      db,
		dataDir: dataDir,
	}

	s.datasources = &DataSourceStore{db: db}
	s.tables = &TableStore{db: db}
	s.questions = &QuestionStore{db: db}
	s.dashboards = &DashboardStore{db: db}
	s.collections = &CollectionStore{db: db}
	s.models = &ModelStore{db: db}
	s.metrics = &MetricStore{db: db}
	s.alerts = &AlertStore{db: db}
	s.subscriptions = &SubscriptionStore{db: db}
	s.users = &UserStore{db: db}
	s.settings = &SettingsStore{db: db}
	s.queryHistory = &QueryHistoryStore{db: db}

	return s, nil
}

// Ensure creates the database schema.
func (s *Store) Ensure(ctx context.Context) error {
	schema := `
	-- Data Sources
	CREATE TABLE IF NOT EXISTS datasources (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		engine TEXT NOT NULL,
		host TEXT,
		port INTEGER,
		database_name TEXT NOT NULL,
		username TEXT,
		password TEXT,
		ssl INTEGER DEFAULT 0,
		options TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Tables
	CREATE TABLE IF NOT EXISTS tables (
		id TEXT PRIMARY KEY,
		datasource_id TEXT NOT NULL REFERENCES datasources(id) ON DELETE CASCADE,
		schema_name TEXT,
		name TEXT NOT NULL,
		display_name TEXT,
		description TEXT,
		row_count INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Columns
	CREATE TABLE IF NOT EXISTS columns (
		id TEXT PRIMARY KEY,
		table_id TEXT NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		display_name TEXT,
		type TEXT NOT NULL,
		semantic TEXT,
		description TEXT,
		position INTEGER DEFAULT 0
	);

	-- Collections
	CREATE TABLE IF NOT EXISTS collections (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		parent_id TEXT REFERENCES collections(id) ON DELETE CASCADE,
		color TEXT,
		created_by TEXT,
		created_at DATETIME NOT NULL
	);

	-- Questions
	CREATE TABLE IF NOT EXISTS questions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		collection_id TEXT REFERENCES collections(id) ON DELETE SET NULL,
		datasource_id TEXT NOT NULL REFERENCES datasources(id) ON DELETE CASCADE,
		query_type TEXT NOT NULL,
		query TEXT NOT NULL,
		visualization TEXT,
		created_by TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Dashboards
	CREATE TABLE IF NOT EXISTS dashboards (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		collection_id TEXT REFERENCES collections(id) ON DELETE SET NULL,
		auto_refresh INTEGER DEFAULT 0,
		created_by TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Dashboard Cards
	CREATE TABLE IF NOT EXISTS dashboard_cards (
		id TEXT PRIMARY KEY,
		dashboard_id TEXT NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
		question_id TEXT REFERENCES questions(id) ON DELETE CASCADE,
		card_type TEXT NOT NULL,
		tab_id TEXT,
		row_num INTEGER NOT NULL,
		col_num INTEGER NOT NULL,
		width INTEGER NOT NULL,
		height INTEGER NOT NULL,
		settings TEXT
	);

	-- Dashboard Tabs
	CREATE TABLE IF NOT EXISTS dashboard_tabs (
		id TEXT PRIMARY KEY,
		dashboard_id TEXT NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		position INTEGER NOT NULL
	);

	-- Dashboard Filters
	CREATE TABLE IF NOT EXISTS dashboard_filters (
		id TEXT PRIMARY KEY,
		dashboard_id TEXT NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		default_value TEXT,
		required INTEGER DEFAULT 0,
		targets TEXT
	);

	-- Models
	CREATE TABLE IF NOT EXISTS models (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		collection_id TEXT REFERENCES collections(id) ON DELETE SET NULL,
		datasource_id TEXT NOT NULL REFERENCES datasources(id) ON DELETE CASCADE,
		query TEXT NOT NULL,
		created_by TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Model Columns
	CREATE TABLE IF NOT EXISTS model_columns (
		id TEXT PRIMARY KEY,
		model_id TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		display_name TEXT,
		description TEXT,
		semantic TEXT,
		visible INTEGER DEFAULT 1
	);

	-- Metrics
	CREATE TABLE IF NOT EXISTS metrics (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		table_id TEXT NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
		definition TEXT NOT NULL,
		created_by TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Alerts
	CREATE TABLE IF NOT EXISTS alerts (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		question_id TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
		alert_type TEXT NOT NULL,
		condition TEXT NOT NULL,
		channels TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		created_by TEXT,
		created_at DATETIME NOT NULL
	);

	-- Subscriptions
	CREATE TABLE IF NOT EXISTS subscriptions (
		id TEXT PRIMARY KEY,
		dashboard_id TEXT NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
		schedule TEXT NOT NULL,
		format TEXT NOT NULL,
		recipients TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		created_by TEXT,
		created_at DATETIME NOT NULL
	);

	-- Users
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		last_login DATETIME
	);

	-- Sessions
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL
	);

	-- Settings
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	-- Audit Logs
	CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		actor_id TEXT,
		actor_email TEXT,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id TEXT,
		metadata TEXT,
		ip_address TEXT,
		timestamp DATETIME NOT NULL
	);

	-- Query History
	CREATE TABLE IF NOT EXISTS query_history (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		datasource_id TEXT NOT NULL,
		query TEXT NOT NULL,
		duration REAL NOT NULL,
		row_count INTEGER NOT NULL,
		error TEXT,
		created_at DATETIME NOT NULL
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_tables_datasource ON tables(datasource_id);
	CREATE INDEX IF NOT EXISTS idx_columns_table ON columns(table_id);
	CREATE INDEX IF NOT EXISTS idx_questions_collection ON questions(collection_id);
	CREATE INDEX IF NOT EXISTS idx_questions_datasource ON questions(datasource_id);
	CREATE INDEX IF NOT EXISTS idx_dashboards_collection ON dashboards(collection_id);
	CREATE INDEX IF NOT EXISTS idx_dashboard_cards_dashboard ON dashboard_cards(dashboard_id);
	CREATE INDEX IF NOT EXISTS idx_collections_parent ON collections(parent_id);
	CREATE INDEX IF NOT EXISTS idx_models_collection ON models(collection_id);
	CREATE INDEX IF NOT EXISTS idx_metrics_table ON metrics(table_id);
	CREATE INDEX IF NOT EXISTS idx_alerts_question ON alerts(question_id);
	CREATE INDEX IF NOT EXISTS idx_subscriptions_dashboard ON subscriptions(dashboard_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_query_history_user ON query_history(user_id);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DataSources returns the data source store.
func (s *Store) DataSources() store.DataSourceStore { return s.datasources }

// Tables returns the table store.
func (s *Store) Tables() store.TableStore { return s.tables }

// Questions returns the question store.
func (s *Store) Questions() store.QuestionStore { return s.questions }

// Dashboards returns the dashboard store.
func (s *Store) Dashboards() store.DashboardStore { return s.dashboards }

// Collections returns the collection store.
func (s *Store) Collections() store.CollectionStore { return s.collections }

// Models returns the model store.
func (s *Store) Models() store.ModelStore { return s.models }

// Metrics returns the metric store.
func (s *Store) Metrics() store.MetricStore { return s.metrics }

// Alerts returns the alert store.
func (s *Store) Alerts() store.AlertStore { return s.alerts }

// Subscriptions returns the subscription store.
func (s *Store) Subscriptions() store.SubscriptionStore { return s.subscriptions }

// Users returns the user store.
func (s *Store) Users() store.UserStore { return s.users }

// Settings returns the settings store.
func (s *Store) Settings() store.SettingsStore { return s.settings }

// QueryHistory returns the query history store.
func (s *Store) QueryHistory() store.QueryHistoryStore { return s.queryHistory }

// DB returns the underlying database connection for query execution.
func (s *Store) DB() *sql.DB { return s.db }

// DataDir returns the data directory path.
func (s *Store) DataDir() string { return s.dataDir }
