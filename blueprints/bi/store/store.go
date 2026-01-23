package store

import (
	"context"
	"time"
)

// Store defines the interface for all storage operations.
type Store interface {
	// Schema management
	Ensure(ctx context.Context) error
	Close() error

	// Data sources
	DataSources() DataSourceStore
	// Tables
	Tables() TableStore
	// Questions
	Questions() QuestionStore
	// Dashboards
	Dashboards() DashboardStore
	// Collections
	Collections() CollectionStore
	// Models
	Models() ModelStore
	// Metrics
	Metrics() MetricStore
	// Alerts
	Alerts() AlertStore
	// Subscriptions
	Subscriptions() SubscriptionStore
	// Users
	Users() UserStore
	// Settings
	Settings() SettingsStore
	// Query History
	QueryHistory() QueryHistoryStore
}

// DataSource represents a database connection.
type DataSource struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Engine   string `json:"engine"` // sqlite, postgres, mysql, mariadb
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database"`
	Username string `json:"username,omitempty"`
	Password string `json:"-"` // encrypted, never exposed

	// SSL/TLS Configuration
	SSL           bool   `json:"ssl"`
	SSLMode       string `json:"ssl_mode,omitempty"`        // disable, allow, prefer, require, verify-ca, verify-full
	SSLRootCert   string `json:"ssl_root_cert,omitempty"`   // PEM certificate content
	SSLClientCert string `json:"ssl_client_cert,omitempty"` // Client certificate
	SSLClientKey  string `json:"-"`                         // Client private key, never exposed

	// SSH Tunnel Configuration
	TunnelEnabled    bool   `json:"tunnel_enabled"`
	TunnelHost       string `json:"tunnel_host,omitempty"`
	TunnelPort       int    `json:"tunnel_port,omitempty"` // default 22
	TunnelUser       string `json:"tunnel_user,omitempty"`
	TunnelAuthMethod string `json:"tunnel_auth_method,omitempty"` // password, ssh-key
	TunnelPassword   string `json:"-"`                            // never exposed
	TunnelPrivateKey string `json:"-"`                            // never exposed
	TunnelPassphrase string `json:"-"`                            // never exposed

	// Schema Filtering
	SchemaFilterType     string   `json:"schema_filter_type,omitempty"` // inclusion, exclusion
	SchemaFilterPatterns []string `json:"schema_filter_patterns,omitempty"`

	// Sync Configuration
	AutoSync       bool       `json:"auto_sync"`
	SyncSchedule   string     `json:"sync_schedule,omitempty"` // cron expression
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus string     `json:"last_sync_status,omitempty"` // success, failed
	LastSyncError  string     `json:"last_sync_error,omitempty"`

	// Cache Configuration
	CacheTTL int `json:"cache_ttl,omitempty"` // seconds, 0 = disabled

	// Connection Pool Configuration
	MaxOpenConns    int `json:"max_open_conns,omitempty"`
	MaxIdleConns    int `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime int `json:"conn_max_lifetime,omitempty"` // seconds
	ConnMaxIdleTime int `json:"conn_max_idle_time,omitempty"` // seconds

	// Additional driver-specific options
	Options map[string]string `json:"options,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Table represents table metadata.
type Table struct {
	ID           string    `json:"id"`
	DataSourceID string    `json:"datasource_id"`
	Schema       string    `json:"schema"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description,omitempty"`
	Visible      bool      `json:"visible"`           // Whether table is visible in query builder
	FieldOrder   string    `json:"field_order,omitempty"` // database, alphabetical, custom, smart
	RowCount     int64     `json:"row_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Column represents column metadata.
type Column struct {
	ID          string `json:"id"`
	TableID     string `json:"table_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`       // Database type
	MappedType  string `json:"mapped_type"` // Normalized: string, number, boolean, datetime
	Semantic    string `json:"semantic,omitempty"`
	Description string `json:"description,omitempty"`
	Position    int    `json:"position"`

	// Visibility
	Visibility string `json:"visibility"` // everywhere, detail_only, hidden

	// Constraints
	Nullable      bool   `json:"nullable"`
	PrimaryKey    bool   `json:"primary_key"`
	ForeignKey    bool   `json:"foreign_key"`
	ForeignTable  string `json:"foreign_table,omitempty"`
	ForeignColumn string `json:"foreign_column,omitempty"`

	// Fingerprint Data (statistics)
	DistinctCount int64   `json:"distinct_count,omitempty"`
	NullCount     int64   `json:"null_count,omitempty"`
	MinValue      string  `json:"min_value,omitempty"`
	MaxValue      string  `json:"max_value,omitempty"`
	AvgLength     float64 `json:"avg_length,omitempty"` // for string types

	// Cached Values (for filter dropdowns)
	CachedValues   []string   `json:"cached_values,omitempty"`
	ValuesCachedAt *time.Time `json:"values_cached_at,omitempty"`
}

// Semantic type constants for columns.
const (
	// Keys
	SemanticPK = "type/PK"
	SemanticFK = "type/FK"

	// Numbers
	SemanticPrice    = "type/Price"
	SemanticCurrency = "type/Currency"
	SemanticScore    = "type/Score"
	SemanticPercent  = "type/Percentage"
	SemanticQuantity = "type/Quantity"

	// Text
	SemanticName        = "type/Name"
	SemanticTitle       = "type/Title"
	SemanticDescription = "type/Description"
	SemanticCategory    = "type/Category"
	SemanticURL         = "type/URL"
	SemanticEmail       = "type/Email"
	SemanticPhone       = "type/Phone"

	// Dates
	SemanticCreated  = "type/CreationDate"
	SemanticUpdated  = "type/UpdateDate"
	SemanticJoined   = "type/JoinDate"
	SemanticBirthday = "type/Birthdate"

	// Geo
	SemanticLatitude  = "type/Latitude"
	SemanticLongitude = "type/Longitude"
	SemanticZipCode   = "type/ZipCode"
	SemanticCity      = "type/City"
	SemanticState     = "type/State"
	SemanticCountry   = "type/Country"
	SemanticAddress   = "type/Address"
)

// ValidSemanticTypes returns all valid semantic types.
func ValidSemanticTypes() []string {
	return []string{
		SemanticPK, SemanticFK,
		SemanticPrice, SemanticCurrency, SemanticScore, SemanticPercent, SemanticQuantity,
		SemanticName, SemanticTitle, SemanticDescription, SemanticCategory, SemanticURL, SemanticEmail, SemanticPhone,
		SemanticCreated, SemanticUpdated, SemanticJoined, SemanticBirthday,
		SemanticLatitude, SemanticLongitude, SemanticZipCode, SemanticCity, SemanticState, SemanticCountry, SemanticAddress,
	}
}

// Question represents a saved query.
type Question struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	CollectionID  string                 `json:"collection_id,omitempty"`
	DataSourceID  string                 `json:"datasource_id"`
	QueryType     string                 `json:"query_type"` // native, query
	Query         map[string]interface{} `json:"query"`
	Visualization map[string]interface{} `json:"visualization"`
	CreatedBy     string                 `json:"created_by"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// Dashboard represents a dashboard.
type Dashboard struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	CollectionID string    `json:"collection_id,omitempty"`
	AutoRefresh  int       `json:"auto_refresh"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DashboardCard represents a card on a dashboard.
type DashboardCard struct {
	ID          string                 `json:"id"`
	DashboardID string                 `json:"dashboard_id"`
	QuestionID  string                 `json:"question_id,omitempty"`
	CardType    string                 `json:"card_type"` // question, text, filter
	TabID       string                 `json:"tab_id,omitempty"`
	Row         int                    `json:"row"`
	Col         int                    `json:"col"`
	Width       int                    `json:"width"`
	Height      int                    `json:"height"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// DashboardTab represents a tab on a dashboard.
type DashboardTab struct {
	ID          string `json:"id"`
	DashboardID string `json:"dashboard_id"`
	Name        string `json:"name"`
	Position    int    `json:"position"`
}

// DashboardFilter represents a filter on a dashboard.
type DashboardFilter struct {
	ID          string         `json:"id"`
	DashboardID string         `json:"dashboard_id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Default     string         `json:"default,omitempty"`
	Required    bool           `json:"required"`
	Targets     []FilterTarget `json:"targets"`
}

// FilterTarget specifies which card/column a filter targets.
type FilterTarget struct {
	CardID   string `json:"card_id"`
	ColumnID string `json:"column_id"`
}

// Collection type constants.
const (
	CollectionTypeRoot     = "root"
	CollectionTypePersonal = "personal"
	CollectionTypeTrash    = "trash"
	CollectionTypeRegular  = ""
)

// Collection represents a folder for organizing items.
type Collection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ParentID    string    `json:"parent_id,omitempty"`
	Color       string    `json:"color,omitempty"`
	Type        string    `json:"type,omitempty"`     // root, personal, trash, or empty for regular
	OwnerID     string    `json:"owner_id,omitempty"` // for personal collections
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// Model represents a curated dataset.
type Model struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	CollectionID string                 `json:"collection_id,omitempty"`
	DataSourceID string                 `json:"datasource_id"`
	Query        map[string]interface{} `json:"query"`
	CreatedBy    string                 `json:"created_by"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// ModelColumn represents a column in a model.
type ModelColumn struct {
	ID          string `json:"id"`
	ModelID     string `json:"model_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	Semantic    string `json:"semantic,omitempty"`
	Visible     bool   `json:"visible"`
}

// Metric represents a canonical calculation.
type Metric struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	TableID     string                 `json:"table_id"`
	Definition  map[string]interface{} `json:"definition"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Alert represents an alert configuration.
type Alert struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	QuestionID string         `json:"question_id"`
	AlertType  string         `json:"alert_type"` // goal, rows
	Condition  AlertCondition `json:"condition"`
	Channels   []AlertChannel `json:"channels"`
	Enabled    bool           `json:"enabled"`
	CreatedBy  string         `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
}

// AlertCondition defines when an alert triggers.
type AlertCondition struct {
	Operator string  `json:"operator"` // above, below, reaches
	Value    float64 `json:"value"`
}

// AlertChannel defines how to send an alert.
type AlertChannel struct {
	Type    string   `json:"type"` // email, slack
	Targets []string `json:"targets"`
}

// Subscription represents a scheduled delivery.
type Subscription struct {
	ID          string    `json:"id"`
	DashboardID string    `json:"dashboard_id"`
	Schedule    string    `json:"schedule"` // cron expression
	Format      string    `json:"format"`   // pdf, png, csv
	Recipients  []string  `json:"recipients"`
	Enabled     bool      `json:"enabled"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// User represents a user account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`   // admin, user, viewer
	Active       bool      `json:"active"` // whether user account is active
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
}

// Session represents a user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// QueryHistory represents a query execution record.
type QueryHistory struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	DataSourceID string    `json:"datasource_id"`
	Query        string    `json:"query"`
	Duration     float64   `json:"duration_ms"`
	RowCount     int64     `json:"row_count"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// QueryResult represents query execution results.
type QueryResult struct {
	Columns    []ResultColumn           `json:"columns"`
	Rows       []map[string]interface{} `json:"rows"`
	RowCount   int64                    `json:"row_count"`
	TotalRows  int64                    `json:"total_rows,omitempty"`  // Total rows before pagination
	Page       int                      `json:"page,omitempty"`        // Current page (1-indexed)
	PageSize   int                      `json:"page_size,omitempty"`   // Page size
	TotalPages int                      `json:"total_pages,omitempty"` // Total pages
	Duration   float64                  `json:"duration_ms"`
	Cached     bool                     `json:"cached"`
}

// ResultColumn represents a column in query results.
type ResultColumn struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

// Settings represents application settings.
type Settings struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID           string            `json:"id"`
	ActorID      string            `json:"actor_id,omitempty"`
	ActorEmail   string            `json:"actor_email,omitempty"`
	Action       string            `json:"action"`
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IPAddress    string            `json:"ip_address,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// Store interfaces

type DataSourceStore interface {
	Create(ctx context.Context, ds *DataSource) error
	GetByID(ctx context.Context, id string) (*DataSource, error)
	List(ctx context.Context) ([]*DataSource, error)
	Update(ctx context.Context, ds *DataSource) error
	Delete(ctx context.Context, id string) error
}

type TableStore interface {
	Create(ctx context.Context, table *Table) error
	GetByID(ctx context.Context, id string) (*Table, error)
	ListByDataSource(ctx context.Context, dsID string) ([]*Table, error)
	Update(ctx context.Context, table *Table) error
	Delete(ctx context.Context, id string) error
	CreateColumn(ctx context.Context, col *Column) error
	GetColumn(ctx context.Context, id string) (*Column, error)
	ListColumns(ctx context.Context, tableID string) ([]*Column, error)
	UpdateColumn(ctx context.Context, col *Column) error
	DeleteColumnsByTable(ctx context.Context, tableID string) error
}

type QuestionStore interface {
	Create(ctx context.Context, q *Question) error
	GetByID(ctx context.Context, id string) (*Question, error)
	List(ctx context.Context) ([]*Question, error)
	ListByCollection(ctx context.Context, collectionID string) ([]*Question, error)
	Update(ctx context.Context, q *Question) error
	Delete(ctx context.Context, id string) error
}

type DashboardStore interface {
	Create(ctx context.Context, d *Dashboard) error
	GetByID(ctx context.Context, id string) (*Dashboard, error)
	List(ctx context.Context) ([]*Dashboard, error)
	ListByCollection(ctx context.Context, collectionID string) ([]*Dashboard, error)
	Update(ctx context.Context, d *Dashboard) error
	Delete(ctx context.Context, id string) error
	// Cards
	CreateCard(ctx context.Context, card *DashboardCard) error
	GetCard(ctx context.Context, id string) (*DashboardCard, error)
	ListCards(ctx context.Context, dashboardID string) ([]*DashboardCard, error)
	UpdateCard(ctx context.Context, card *DashboardCard) error
	DeleteCard(ctx context.Context, id string) error
	// Filters
	CreateFilter(ctx context.Context, filter *DashboardFilter) error
	ListFilters(ctx context.Context, dashboardID string) ([]*DashboardFilter, error)
	DeleteFilter(ctx context.Context, id string) error
	// Tabs
	CreateTab(ctx context.Context, tab *DashboardTab) error
	ListTabs(ctx context.Context, dashboardID string) ([]*DashboardTab, error)
	DeleteTab(ctx context.Context, id string) error
}

type CollectionStore interface {
	Create(ctx context.Context, c *Collection) error
	GetByID(ctx context.Context, id string) (*Collection, error)
	List(ctx context.Context) ([]*Collection, error)
	ListByParent(ctx context.Context, parentID string) ([]*Collection, error)
	Update(ctx context.Context, c *Collection) error
	Delete(ctx context.Context, id string) error
	// Special collections
	GetRootCollection(ctx context.Context) (*Collection, error)
	GetPersonalCollection(ctx context.Context, userID string) (*Collection, error)
	EnsureRootCollection(ctx context.Context) (*Collection, error)
	EnsurePersonalCollection(ctx context.Context, userID, userName string) (*Collection, error)
}

type ModelStore interface {
	Create(ctx context.Context, m *Model) error
	GetByID(ctx context.Context, id string) (*Model, error)
	List(ctx context.Context) ([]*Model, error)
	Update(ctx context.Context, m *Model) error
	Delete(ctx context.Context, id string) error
	CreateColumn(ctx context.Context, col *ModelColumn) error
	ListColumns(ctx context.Context, modelID string) ([]*ModelColumn, error)
	UpdateColumn(ctx context.Context, col *ModelColumn) error
}

type MetricStore interface {
	Create(ctx context.Context, m *Metric) error
	GetByID(ctx context.Context, id string) (*Metric, error)
	List(ctx context.Context) ([]*Metric, error)
	ListByTable(ctx context.Context, tableID string) ([]*Metric, error)
	Update(ctx context.Context, m *Metric) error
	Delete(ctx context.Context, id string) error
}

type AlertStore interface {
	Create(ctx context.Context, a *Alert) error
	GetByID(ctx context.Context, id string) (*Alert, error)
	List(ctx context.Context) ([]*Alert, error)
	ListByQuestion(ctx context.Context, questionID string) ([]*Alert, error)
	Update(ctx context.Context, a *Alert) error
	Delete(ctx context.Context, id string) error
}

type SubscriptionStore interface {
	Create(ctx context.Context, s *Subscription) error
	GetByID(ctx context.Context, id string) (*Subscription, error)
	List(ctx context.Context) ([]*Subscription, error)
	ListByDashboard(ctx context.Context, dashboardID string) ([]*Subscription, error)
	Update(ctx context.Context, s *Subscription) error
	Delete(ctx context.Context, id string) error
}

type UserStore interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	UpdateLastLogin(ctx context.Context, userID string) error
}

type SettingsStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	List(ctx context.Context) ([]*Settings, error)
	WriteAuditLog(ctx context.Context, log *AuditLog) error
	ListAuditLogs(ctx context.Context, limit, offset int) ([]*AuditLog, error)
}

type QueryHistoryStore interface {
	Create(ctx context.Context, qh *QueryHistory) error
	List(ctx context.Context, userID string, limit int) ([]*QueryHistory, error)
}
