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
	"github.com/oklog/ulid/v2"

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

// Helper functions

func generateID() string {
	return ulid.Make().String()
}

func toJSON(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func fromJSON(s string, v interface{}) error {
	if s == "" {
		return nil
	}
	return json.Unmarshal([]byte(s), v)
}

// DataSourceStore implements store.DataSourceStore.
type DataSourceStore struct {
	db *sql.DB
}

func (s *DataSourceStore) Create(ctx context.Context, ds *store.DataSource) error {
	if ds.ID == "" {
		ds.ID = generateID()
	}
	now := time.Now()
	ds.CreatedAt = now
	ds.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO datasources (id, name, engine, host, port, database_name, username, password, ssl, options, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ds.ID, ds.Name, ds.Engine, ds.Host, ds.Port, ds.Database, ds.Username, ds.Password, ds.SSL, toJSON(ds.Options), ds.CreatedAt, ds.UpdatedAt)
	return err
}

func (s *DataSourceStore) GetByID(ctx context.Context, id string) (*store.DataSource, error) {
	var ds store.DataSource
	var options string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, engine, host, port, database_name, username, password, ssl, options, created_at, updated_at
		FROM datasources WHERE id = ?
	`, id).Scan(&ds.ID, &ds.Name, &ds.Engine, &ds.Host, &ds.Port, &ds.Database, &ds.Username, &ds.Password, &ds.SSL, &options, &ds.CreatedAt, &ds.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(options, &ds.Options)
	return &ds, nil
}

func (s *DataSourceStore) List(ctx context.Context) ([]*store.DataSource, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, engine, host, port, database_name, username, password, ssl, options, created_at, updated_at
		FROM datasources ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DataSource
	for rows.Next() {
		var ds store.DataSource
		var options string
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.Engine, &ds.Host, &ds.Port, &ds.Database, &ds.Username, &ds.Password, &ds.SSL, &options, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			return nil, err
		}
		fromJSON(options, &ds.Options)
		result = append(result, &ds)
	}
	return result, rows.Err()
}

func (s *DataSourceStore) Update(ctx context.Context, ds *store.DataSource) error {
	ds.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE datasources SET name=?, engine=?, host=?, port=?, database_name=?, username=?, password=?, ssl=?, options=?, updated_at=?
		WHERE id=?
	`, ds.Name, ds.Engine, ds.Host, ds.Port, ds.Database, ds.Username, ds.Password, ds.SSL, toJSON(ds.Options), ds.UpdatedAt, ds.ID)
	return err
}

func (s *DataSourceStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM datasources WHERE id=?`, id)
	return err
}

// TableStore implements store.TableStore.
type TableStore struct {
	db *sql.DB
}

func (s *TableStore) Create(ctx context.Context, t *store.Table) error {
	if t.ID == "" {
		t.ID = generateID()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tables (id, datasource_id, schema_name, name, display_name, description, row_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.DataSourceID, t.Schema, t.Name, t.DisplayName, t.Description, t.RowCount, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *TableStore) GetByID(ctx context.Context, id string) (*store.Table, error) {
	var t store.Table
	err := s.db.QueryRowContext(ctx, `
		SELECT id, datasource_id, schema_name, name, display_name, description, row_count, created_at, updated_at
		FROM tables WHERE id = ?
	`, id).Scan(&t.ID, &t.DataSourceID, &t.Schema, &t.Name, &t.DisplayName, &t.Description, &t.RowCount, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *TableStore) ListByDataSource(ctx context.Context, dsID string) ([]*store.Table, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, datasource_id, schema_name, name, display_name, description, row_count, created_at, updated_at
		FROM tables WHERE datasource_id = ? ORDER BY name
	`, dsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Table
	for rows.Next() {
		var t store.Table
		if err := rows.Scan(&t.ID, &t.DataSourceID, &t.Schema, &t.Name, &t.DisplayName, &t.Description, &t.RowCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}

func (s *TableStore) Update(ctx context.Context, t *store.Table) error {
	t.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE tables SET display_name=?, description=?, row_count=?, updated_at=?
		WHERE id=?
	`, t.DisplayName, t.Description, t.RowCount, t.UpdatedAt, t.ID)
	return err
}

func (s *TableStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tables WHERE id=?`, id)
	return err
}

func (s *TableStore) CreateColumn(ctx context.Context, col *store.Column) error {
	if col.ID == "" {
		col.ID = generateID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO columns (id, table_id, name, display_name, type, semantic, description, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, col.ID, col.TableID, col.Name, col.DisplayName, col.Type, col.Semantic, col.Description, col.Position)
	return err
}

func (s *TableStore) ListColumns(ctx context.Context, tableID string) ([]*store.Column, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, name, display_name, type, semantic, description, position
		FROM columns WHERE table_id = ? ORDER BY position
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Column
	for rows.Next() {
		var c store.Column
		if err := rows.Scan(&c.ID, &c.TableID, &c.Name, &c.DisplayName, &c.Type, &c.Semantic, &c.Description, &c.Position); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *TableStore) DeleteColumnsByTable(ctx context.Context, tableID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM columns WHERE table_id=?`, tableID)
	return err
}

// QuestionStore implements store.QuestionStore.
type QuestionStore struct {
	db *sql.DB
}

func (s *QuestionStore) Create(ctx context.Context, q *store.Question) error {
	if q.ID == "" {
		q.ID = generateID()
	}
	now := time.Now()
	q.CreatedAt = now
	q.UpdatedAt = now

	var collID interface{}
	if q.CollectionID != "" {
		collID = q.CollectionID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO questions (id, name, description, collection_id, datasource_id, query_type, query, visualization, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, q.ID, q.Name, q.Description, collID, q.DataSourceID, q.QueryType, toJSON(q.Query), toJSON(q.Visualization), q.CreatedBy, q.CreatedAt, q.UpdatedAt)
	return err
}

func (s *QuestionStore) GetByID(ctx context.Context, id string) (*store.Question, error) {
	var q store.Question
	var query, viz string
	var collID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query_type, query, visualization, created_by, created_at, updated_at
		FROM questions WHERE id = ?
	`, id).Scan(&q.ID, &q.Name, &q.Description, &collID, &q.DataSourceID, &q.QueryType, &query, &viz, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	q.CollectionID = collID.String
	fromJSON(query, &q.Query)
	fromJSON(viz, &q.Visualization)
	return &q, nil
}

func (s *QuestionStore) List(ctx context.Context) ([]*store.Question, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query_type, query, visualization, created_by, created_at, updated_at
		FROM questions ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Question
	for rows.Next() {
		var q store.Question
		var query, viz string
		var collID sql.NullString
		if err := rows.Scan(&q.ID, &q.Name, &q.Description, &collID, &q.DataSourceID, &q.QueryType, &query, &viz, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		q.CollectionID = collID.String
		fromJSON(query, &q.Query)
		fromJSON(viz, &q.Visualization)
		result = append(result, &q)
	}
	return result, rows.Err()
}

func (s *QuestionStore) ListByCollection(ctx context.Context, collectionID string) ([]*store.Question, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query_type, query, visualization, created_by, created_at, updated_at
		FROM questions WHERE collection_id = ? ORDER BY name
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Question
	for rows.Next() {
		var q store.Question
		var query, viz string
		var collID sql.NullString
		if err := rows.Scan(&q.ID, &q.Name, &q.Description, &collID, &q.DataSourceID, &q.QueryType, &query, &viz, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		q.CollectionID = collID.String
		fromJSON(query, &q.Query)
		fromJSON(viz, &q.Visualization)
		result = append(result, &q)
	}
	return result, rows.Err()
}

func (s *QuestionStore) Update(ctx context.Context, q *store.Question) error {
	q.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE questions SET name=?, description=?, collection_id=?, query_type=?, query=?, visualization=?, updated_at=?
		WHERE id=?
	`, q.Name, q.Description, q.CollectionID, q.QueryType, toJSON(q.Query), toJSON(q.Visualization), q.UpdatedAt, q.ID)
	return err
}

func (s *QuestionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM questions WHERE id=?`, id)
	return err
}

// DashboardStore implements store.DashboardStore.
type DashboardStore struct {
	db *sql.DB
}

func (s *DashboardStore) Create(ctx context.Context, d *store.Dashboard) error {
	if d.ID == "" {
		d.ID = generateID()
	}
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now

	var collID interface{}
	if d.CollectionID != "" {
		collID = d.CollectionID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboards (id, name, description, collection_id, auto_refresh, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.Name, d.Description, collID, d.AutoRefresh, d.CreatedBy, d.CreatedAt, d.UpdatedAt)
	return err
}

func (s *DashboardStore) GetByID(ctx context.Context, id string) (*store.Dashboard, error) {
	var d store.Dashboard
	var collID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, collection_id, auto_refresh, created_by, created_at, updated_at
		FROM dashboards WHERE id = ?
	`, id).Scan(&d.ID, &d.Name, &d.Description, &collID, &d.AutoRefresh, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.CollectionID = collID.String
	return &d, nil
}

func (s *DashboardStore) List(ctx context.Context) ([]*store.Dashboard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, auto_refresh, created_by, created_at, updated_at
		FROM dashboards ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Dashboard
	for rows.Next() {
		var d store.Dashboard
		var collID sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &collID, &d.AutoRefresh, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.CollectionID = collID.String
		result = append(result, &d)
	}
	return result, rows.Err()
}

func (s *DashboardStore) ListByCollection(ctx context.Context, collectionID string) ([]*store.Dashboard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, auto_refresh, created_by, created_at, updated_at
		FROM dashboards WHERE collection_id = ? ORDER BY name
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Dashboard
	for rows.Next() {
		var d store.Dashboard
		var collID sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &collID, &d.AutoRefresh, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.CollectionID = collID.String
		result = append(result, &d)
	}
	return result, rows.Err()
}

func (s *DashboardStore) Update(ctx context.Context, d *store.Dashboard) error {
	d.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE dashboards SET name=?, description=?, collection_id=?, auto_refresh=?, updated_at=?
		WHERE id=?
	`, d.Name, d.Description, d.CollectionID, d.AutoRefresh, d.UpdatedAt, d.ID)
	return err
}

func (s *DashboardStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dashboards WHERE id=?`, id)
	return err
}

func (s *DashboardStore) CreateCard(ctx context.Context, card *store.DashboardCard) error {
	if card.ID == "" {
		card.ID = generateID()
	}

	var questionID interface{}
	if card.QuestionID != "" {
		questionID = card.QuestionID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboard_cards (id, dashboard_id, question_id, card_type, tab_id, row_num, col_num, width, height, settings)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, card.ID, card.DashboardID, questionID, card.CardType, card.TabID, card.Row, card.Col, card.Width, card.Height, toJSON(card.Settings))
	return err
}

func (s *DashboardStore) GetCard(ctx context.Context, id string) (*store.DashboardCard, error) {
	var card store.DashboardCard
	var settings string
	var qID, tabID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, dashboard_id, question_id, card_type, tab_id, row_num, col_num, width, height, settings
		FROM dashboard_cards WHERE id = ?
	`, id).Scan(&card.ID, &card.DashboardID, &qID, &card.CardType, &tabID, &card.Row, &card.Col, &card.Width, &card.Height, &settings)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	card.QuestionID = qID.String
	card.TabID = tabID.String
	fromJSON(settings, &card.Settings)
	return &card, nil
}

func (s *DashboardStore) ListCards(ctx context.Context, dashboardID string) ([]*store.DashboardCard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, question_id, card_type, tab_id, row_num, col_num, width, height, settings
		FROM dashboard_cards WHERE dashboard_id = ? ORDER BY row_num, col_num
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DashboardCard
	for rows.Next() {
		var card store.DashboardCard
		var settings string
		var qID, tabID sql.NullString
		if err := rows.Scan(&card.ID, &card.DashboardID, &qID, &card.CardType, &tabID, &card.Row, &card.Col, &card.Width, &card.Height, &settings); err != nil {
			return nil, err
		}
		card.QuestionID = qID.String
		card.TabID = tabID.String
		fromJSON(settings, &card.Settings)
		result = append(result, &card)
	}
	return result, rows.Err()
}

func (s *DashboardStore) UpdateCard(ctx context.Context, card *store.DashboardCard) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE dashboard_cards SET question_id=?, card_type=?, tab_id=?, row_num=?, col_num=?, width=?, height=?, settings=?
		WHERE id=?
	`, card.QuestionID, card.CardType, card.TabID, card.Row, card.Col, card.Width, card.Height, toJSON(card.Settings), card.ID)
	return err
}

func (s *DashboardStore) DeleteCard(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dashboard_cards WHERE id=?`, id)
	return err
}

func (s *DashboardStore) CreateFilter(ctx context.Context, filter *store.DashboardFilter) error {
	if filter.ID == "" {
		filter.ID = generateID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboard_filters (id, dashboard_id, name, type, default_value, required, targets)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, filter.ID, filter.DashboardID, filter.Name, filter.Type, filter.Default, filter.Required, toJSON(filter.Targets))
	return err
}

func (s *DashboardStore) ListFilters(ctx context.Context, dashboardID string) ([]*store.DashboardFilter, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, name, type, default_value, required, targets
		FROM dashboard_filters WHERE dashboard_id = ?
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DashboardFilter
	for rows.Next() {
		var f store.DashboardFilter
		var targets string
		if err := rows.Scan(&f.ID, &f.DashboardID, &f.Name, &f.Type, &f.Default, &f.Required, &targets); err != nil {
			return nil, err
		}
		fromJSON(targets, &f.Targets)
		result = append(result, &f)
	}
	return result, rows.Err()
}

func (s *DashboardStore) DeleteFilter(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dashboard_filters WHERE id=?`, id)
	return err
}

func (s *DashboardStore) CreateTab(ctx context.Context, tab *store.DashboardTab) error {
	if tab.ID == "" {
		tab.ID = generateID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboard_tabs (id, dashboard_id, name, position)
		VALUES (?, ?, ?, ?)
	`, tab.ID, tab.DashboardID, tab.Name, tab.Position)
	return err
}

func (s *DashboardStore) ListTabs(ctx context.Context, dashboardID string) ([]*store.DashboardTab, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, name, position
		FROM dashboard_tabs WHERE dashboard_id = ? ORDER BY position
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DashboardTab
	for rows.Next() {
		var t store.DashboardTab
		if err := rows.Scan(&t.ID, &t.DashboardID, &t.Name, &t.Position); err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}

func (s *DashboardStore) DeleteTab(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dashboard_tabs WHERE id=?`, id)
	return err
}

// CollectionStore implements store.CollectionStore.
type CollectionStore struct {
	db *sql.DB
}

func (s *CollectionStore) Create(ctx context.Context, c *store.Collection) error {
	if c.ID == "" {
		c.ID = generateID()
	}
	c.CreatedAt = time.Now()

	var parentID interface{}
	if c.ParentID != "" {
		parentID = c.ParentID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collections (id, name, description, parent_id, color, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, c.Description, parentID, c.Color, c.CreatedBy, c.CreatedAt)
	return err
}

func (s *CollectionStore) GetByID(ctx context.Context, id string) (*store.Collection, error) {
	var c store.Collection
	var parentID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, parent_id, color, created_by, created_at
		FROM collections WHERE id = ?
	`, id).Scan(&c.ID, &c.Name, &c.Description, &parentID, &c.Color, &c.CreatedBy, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	return &c, nil
}

func (s *CollectionStore) List(ctx context.Context) ([]*store.Collection, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, parent_id, color, created_by, created_at
		FROM collections ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Collection
	for rows.Next() {
		var c store.Collection
		var parentID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &parentID, &c.Color, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ParentID = parentID.String
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *CollectionStore) ListByParent(ctx context.Context, parentID string) ([]*store.Collection, error) {
	var rows *sql.Rows
	var err error
	if parentID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, description, parent_id, color, created_by, created_at
			FROM collections WHERE parent_id IS NULL ORDER BY name
		`)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, description, parent_id, color, created_by, created_at
			FROM collections WHERE parent_id = ? ORDER BY name
		`, parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Collection
	for rows.Next() {
		var c store.Collection
		var pID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &pID, &c.Color, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ParentID = pID.String
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *CollectionStore) Update(ctx context.Context, c *store.Collection) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE collections SET name=?, description=?, parent_id=?, color=?
		WHERE id=?
	`, c.Name, c.Description, c.ParentID, c.Color, c.ID)
	return err
}

func (s *CollectionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id=?`, id)
	return err
}

// ModelStore implements store.ModelStore.
type ModelStore struct {
	db *sql.DB
}

func (s *ModelStore) Create(ctx context.Context, m *store.Model) error {
	if m.ID == "" {
		m.ID = generateID()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	var collID interface{}
	if m.CollectionID != "" {
		collID = m.CollectionID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO models (id, name, description, collection_id, datasource_id, query, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Name, m.Description, collID, m.DataSourceID, toJSON(m.Query), m.CreatedBy, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *ModelStore) GetByID(ctx context.Context, id string) (*store.Model, error) {
	var m store.Model
	var query string
	var collID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query, created_by, created_at, updated_at
		FROM models WHERE id = ?
	`, id).Scan(&m.ID, &m.Name, &m.Description, &collID, &m.DataSourceID, &query, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.CollectionID = collID.String
	fromJSON(query, &m.Query)
	return &m, nil
}

func (s *ModelStore) List(ctx context.Context) ([]*store.Model, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query, created_by, created_at, updated_at
		FROM models ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Model
	for rows.Next() {
		var m store.Model
		var query string
		var collID sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &collID, &m.DataSourceID, &query, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.CollectionID = collID.String
		fromJSON(query, &m.Query)
		result = append(result, &m)
	}
	return result, rows.Err()
}

func (s *ModelStore) Update(ctx context.Context, m *store.Model) error {
	m.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE models SET name=?, description=?, collection_id=?, query=?, updated_at=?
		WHERE id=?
	`, m.Name, m.Description, m.CollectionID, toJSON(m.Query), m.UpdatedAt, m.ID)
	return err
}

func (s *ModelStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM models WHERE id=?`, id)
	return err
}

func (s *ModelStore) CreateColumn(ctx context.Context, col *store.ModelColumn) error {
	if col.ID == "" {
		col.ID = generateID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO model_columns (id, model_id, name, display_name, description, semantic, visible)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, col.ID, col.ModelID, col.Name, col.DisplayName, col.Description, col.Semantic, col.Visible)
	return err
}

func (s *ModelStore) ListColumns(ctx context.Context, modelID string) ([]*store.ModelColumn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, model_id, name, display_name, description, semantic, visible
		FROM model_columns WHERE model_id = ?
	`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.ModelColumn
	for rows.Next() {
		var c store.ModelColumn
		if err := rows.Scan(&c.ID, &c.ModelID, &c.Name, &c.DisplayName, &c.Description, &c.Semantic, &c.Visible); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *ModelStore) UpdateColumn(ctx context.Context, col *store.ModelColumn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE model_columns SET display_name=?, description=?, semantic=?, visible=?
		WHERE id=?
	`, col.DisplayName, col.Description, col.Semantic, col.Visible, col.ID)
	return err
}

// MetricStore implements store.MetricStore.
type MetricStore struct {
	db *sql.DB
}

func (s *MetricStore) Create(ctx context.Context, m *store.Metric) error {
	if m.ID == "" {
		m.ID = generateID()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO metrics (id, name, description, table_id, definition, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Name, m.Description, m.TableID, toJSON(m.Definition), m.CreatedBy, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *MetricStore) GetByID(ctx context.Context, id string) (*store.Metric, error) {
	var m store.Metric
	var def string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, table_id, definition, created_by, created_at, updated_at
		FROM metrics WHERE id = ?
	`, id).Scan(&m.ID, &m.Name, &m.Description, &m.TableID, &def, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(def, &m.Definition)
	return &m, nil
}

func (s *MetricStore) List(ctx context.Context) ([]*store.Metric, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, table_id, definition, created_by, created_at, updated_at
		FROM metrics ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Metric
	for rows.Next() {
		var m store.Metric
		var def string
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.TableID, &def, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		fromJSON(def, &m.Definition)
		result = append(result, &m)
	}
	return result, rows.Err()
}

func (s *MetricStore) ListByTable(ctx context.Context, tableID string) ([]*store.Metric, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, table_id, definition, created_by, created_at, updated_at
		FROM metrics WHERE table_id = ? ORDER BY name
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Metric
	for rows.Next() {
		var m store.Metric
		var def string
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.TableID, &def, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		fromJSON(def, &m.Definition)
		result = append(result, &m)
	}
	return result, rows.Err()
}

func (s *MetricStore) Update(ctx context.Context, m *store.Metric) error {
	m.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE metrics SET name=?, description=?, definition=?, updated_at=?
		WHERE id=?
	`, m.Name, m.Description, toJSON(m.Definition), m.UpdatedAt, m.ID)
	return err
}

func (s *MetricStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM metrics WHERE id=?`, id)
	return err
}

// AlertStore implements store.AlertStore.
type AlertStore struct {
	db *sql.DB
}

func (s *AlertStore) Create(ctx context.Context, a *store.Alert) error {
	if a.ID == "" {
		a.ID = generateID()
	}
	a.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO alerts (id, name, question_id, alert_type, condition, channels, enabled, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.Name, a.QuestionID, a.AlertType, toJSON(a.Condition), toJSON(a.Channels), a.Enabled, a.CreatedBy, a.CreatedAt)
	return err
}

func (s *AlertStore) GetByID(ctx context.Context, id string) (*store.Alert, error) {
	var a store.Alert
	var cond, channels string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, question_id, alert_type, condition, channels, enabled, created_by, created_at
		FROM alerts WHERE id = ?
	`, id).Scan(&a.ID, &a.Name, &a.QuestionID, &a.AlertType, &cond, &channels, &a.Enabled, &a.CreatedBy, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(cond, &a.Condition)
	fromJSON(channels, &a.Channels)
	return &a, nil
}

func (s *AlertStore) List(ctx context.Context) ([]*store.Alert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, question_id, alert_type, condition, channels, enabled, created_by, created_at
		FROM alerts ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Alert
	for rows.Next() {
		var a store.Alert
		var cond, channels string
		if err := rows.Scan(&a.ID, &a.Name, &a.QuestionID, &a.AlertType, &cond, &channels, &a.Enabled, &a.CreatedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		fromJSON(cond, &a.Condition)
		fromJSON(channels, &a.Channels)
		result = append(result, &a)
	}
	return result, rows.Err()
}

func (s *AlertStore) ListByQuestion(ctx context.Context, questionID string) ([]*store.Alert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, question_id, alert_type, condition, channels, enabled, created_by, created_at
		FROM alerts WHERE question_id = ? ORDER BY name
	`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Alert
	for rows.Next() {
		var a store.Alert
		var cond, channels string
		if err := rows.Scan(&a.ID, &a.Name, &a.QuestionID, &a.AlertType, &cond, &channels, &a.Enabled, &a.CreatedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		fromJSON(cond, &a.Condition)
		fromJSON(channels, &a.Channels)
		result = append(result, &a)
	}
	return result, rows.Err()
}

func (s *AlertStore) Update(ctx context.Context, a *store.Alert) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE alerts SET name=?, alert_type=?, condition=?, channels=?, enabled=?
		WHERE id=?
	`, a.Name, a.AlertType, toJSON(a.Condition), toJSON(a.Channels), a.Enabled, a.ID)
	return err
}

func (s *AlertStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM alerts WHERE id=?`, id)
	return err
}

// SubscriptionStore implements store.SubscriptionStore.
type SubscriptionStore struct {
	db *sql.DB
}

func (s *SubscriptionStore) Create(ctx context.Context, sub *store.Subscription) error {
	if sub.ID == "" {
		sub.ID = generateID()
	}
	sub.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions (id, dashboard_id, schedule, format, recipients, enabled, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, sub.ID, sub.DashboardID, sub.Schedule, sub.Format, toJSON(sub.Recipients), sub.Enabled, sub.CreatedBy, sub.CreatedAt)
	return err
}

func (s *SubscriptionStore) GetByID(ctx context.Context, id string) (*store.Subscription, error) {
	var sub store.Subscription
	var recipients string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, dashboard_id, schedule, format, recipients, enabled, created_by, created_at
		FROM subscriptions WHERE id = ?
	`, id).Scan(&sub.ID, &sub.DashboardID, &sub.Schedule, &sub.Format, &recipients, &sub.Enabled, &sub.CreatedBy, &sub.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(recipients, &sub.Recipients)
	return &sub, nil
}

func (s *SubscriptionStore) List(ctx context.Context) ([]*store.Subscription, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, schedule, format, recipients, enabled, created_by, created_at
		FROM subscriptions
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Subscription
	for rows.Next() {
		var sub store.Subscription
		var recipients string
		if err := rows.Scan(&sub.ID, &sub.DashboardID, &sub.Schedule, &sub.Format, &recipients, &sub.Enabled, &sub.CreatedBy, &sub.CreatedAt); err != nil {
			return nil, err
		}
		fromJSON(recipients, &sub.Recipients)
		result = append(result, &sub)
	}
	return result, rows.Err()
}

func (s *SubscriptionStore) ListByDashboard(ctx context.Context, dashboardID string) ([]*store.Subscription, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, schedule, format, recipients, enabled, created_by, created_at
		FROM subscriptions WHERE dashboard_id = ?
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Subscription
	for rows.Next() {
		var sub store.Subscription
		var recipients string
		if err := rows.Scan(&sub.ID, &sub.DashboardID, &sub.Schedule, &sub.Format, &recipients, &sub.Enabled, &sub.CreatedBy, &sub.CreatedAt); err != nil {
			return nil, err
		}
		fromJSON(recipients, &sub.Recipients)
		result = append(result, &sub)
	}
	return result, rows.Err()
}

func (s *SubscriptionStore) Update(ctx context.Context, sub *store.Subscription) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE subscriptions SET schedule=?, format=?, recipients=?, enabled=?
		WHERE id=?
	`, sub.Schedule, sub.Format, toJSON(sub.Recipients), sub.Enabled, sub.ID)
	return err
}

func (s *SubscriptionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id=?`, id)
	return err
}

// UserStore implements store.UserStore.
type UserStore struct {
	db *sql.DB
}

func (s *UserStore) Create(ctx context.Context, user *store.User) error {
	if user.ID == "" {
		user.ID = generateID()
	}
	user.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, user.ID, user.Email, user.Name, user.PasswordHash, user.Role, user.CreatedAt)
	return err
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*store.User, error) {
	var user store.User
	var lastLogin sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, role, created_at, last_login
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	return &user, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	var user store.User
	var lastLogin sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, role, created_at, last_login
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	return &user, nil
}

func (s *UserStore) List(ctx context.Context) ([]*store.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, name, password_hash, role, created_at, last_login
		FROM users ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.User
	for rows.Next() {
		var user store.User
		var lastLogin sql.NullTime
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin); err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			user.LastLogin = lastLogin.Time
		}
		result = append(result, &user)
	}
	return result, rows.Err()
}

func (s *UserStore) Update(ctx context.Context, user *store.User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET email=?, name=?, password_hash=?, role=?
		WHERE id=?
	`, user.Email, user.Name, user.PasswordHash, user.Role, user.ID)
	return err
}

func (s *UserStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id=?`, id)
	return err
}

func (s *UserStore) CreateSession(ctx context.Context, session *store.Session) error {
	if session.ID == "" {
		session.ID = generateID()
	}
	session.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt)
	return err
}

func (s *UserStore) GetSession(ctx context.Context, token string) (*store.Session, error) {
	var session store.Session
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions WHERE token = ? AND expires_at > datetime('now')
	`, token).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *UserStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token=?`, token)
	return err
}

func (s *UserStore) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET last_login=datetime('now') WHERE id=?`, userID)
	return err
}

// SettingsStore implements store.SettingsStore.
type SettingsStore struct {
	db *sql.DB
}

func (s *SettingsStore) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *SettingsStore) Set(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

func (s *SettingsStore) List(ctx context.Context) ([]*store.Settings, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Settings
	for rows.Next() {
		var s store.Settings
		if err := rows.Scan(&s.Key, &s.Value); err != nil {
			return nil, err
		}
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (s *SettingsStore) WriteAuditLog(ctx context.Context, log *store.AuditLog) error {
	if log.ID == "" {
		log.ID = generateID()
	}
	log.Timestamp = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, actor_id, actor_email, action, resource_type, resource_id, metadata, ip_address, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.ID, log.ActorID, log.ActorEmail, log.Action, log.ResourceType, log.ResourceID, toJSON(log.Metadata), log.IPAddress, log.Timestamp)
	return err
}

func (s *SettingsStore) ListAuditLogs(ctx context.Context, limit, offset int) ([]*store.AuditLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor_id, actor_email, action, resource_type, resource_id, metadata, ip_address, timestamp
		FROM audit_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.AuditLog
	for rows.Next() {
		var log store.AuditLog
		var metadata string
		if err := rows.Scan(&log.ID, &log.ActorID, &log.ActorEmail, &log.Action, &log.ResourceType, &log.ResourceID, &metadata, &log.IPAddress, &log.Timestamp); err != nil {
			return nil, err
		}
		fromJSON(metadata, &log.Metadata)
		result = append(result, &log)
	}
	return result, rows.Err()
}

// QueryHistoryStore implements store.QueryHistoryStore.
type QueryHistoryStore struct {
	db *sql.DB
}

func (s *QueryHistoryStore) Create(ctx context.Context, qh *store.QueryHistory) error {
	if qh.ID == "" {
		qh.ID = generateID()
	}
	qh.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO query_history (id, user_id, datasource_id, query, duration, row_count, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, qh.ID, qh.UserID, qh.DataSourceID, qh.Query, qh.Duration, qh.RowCount, qh.Error, qh.CreatedAt)
	return err
}

func (s *QueryHistoryStore) List(ctx context.Context, userID string, limit int) ([]*store.QueryHistory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, datasource_id, query, duration, row_count, error, created_at
		FROM query_history WHERE user_id = ? ORDER BY created_at DESC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.QueryHistory
	for rows.Next() {
		var qh store.QueryHistory
		if err := rows.Scan(&qh.ID, &qh.UserID, &qh.DataSourceID, &qh.Query, &qh.Duration, &qh.RowCount, &qh.Error, &qh.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &qh)
	}
	return result, rows.Err()
}
