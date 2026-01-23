// Data Source types
export interface DataSource {
  id: string
  name: string
  engine: 'sqlite' | 'postgres' | 'mysql' | 'clickhouse' | 'duckdb'

  // Basic connection
  host?: string
  port?: number
  database: string
  username?: string

  // SSL/TLS Configuration
  ssl: boolean
  ssl_mode?: 'disable' | 'allow' | 'prefer' | 'require' | 'verify-ca' | 'verify-full'
  ssl_root_cert?: string
  ssl_client_cert?: string

  // SSH Tunnel
  tunnel_enabled?: boolean
  tunnel_host?: string
  tunnel_port?: number
  tunnel_user?: string
  tunnel_auth_method?: 'password' | 'ssh-key'

  // Schema Filtering
  schema_filter_type?: 'all' | 'inclusion' | 'exclusion'
  schema_filter_patterns?: string[]

  // Sync Configuration
  auto_sync?: boolean
  sync_schedule?: string
  last_sync_at?: string
  last_sync_status?: 'success' | 'failed' | 'partial' | 'running'
  last_sync_error?: string

  // Cache Configuration
  cache_ttl?: number

  // Connection Pool
  max_open_conns?: number
  max_idle_conns?: number
  conn_max_lifetime?: number
  conn_max_idle_time?: number

  // Additional options
  options?: Record<string, string>

  // Metadata
  created_at: string
  updated_at: string
}

export interface DataSourceTestResult {
  success?: boolean
  valid?: boolean
  error?: string
  error_code?: string
  suggestions?: string[]
  version?: string
  schemas?: string[]
  latency_ms?: number
}

export interface DataSourceStatus {
  connected: boolean
  error?: string
  latency_ms?: number
  last_sync_at?: string
  last_sync_status?: string
  last_sync_error?: string
  capabilities?: DriverCapabilities
}

export interface DriverCapabilities {
  supports_schemas: boolean
  supports_ssl: boolean
  supports_ssh: boolean
  supports_ctes: boolean
  supports_json: boolean
  supports_arrays: boolean
  supports_window_functions: boolean
  max_query_timeout?: number
  default_port?: number
}

export interface SyncResult {
  status: string
  duration_ms: number
  schemas_synced?: number
  tables_synced?: number
  columns_synced?: number
  tables_added?: number
  tables_removed?: number
  columns_added?: number
  columns_removed?: number
  fields_scanned?: number
  values_cached?: number
  columns_fingerprinted?: number
  errors: string[]
}

export interface SyncLog {
  id: string
  type: 'schema_sync' | 'field_scan' | 'fingerprint'
  status: 'completed' | 'failed' | 'running'
  started_at?: string
  completed_at?: string
  duration_ms?: number
  error?: string
  details?: Record<string, any>
}

export interface CacheStats {
  datasource_id: string
  columns_with_cache: number
  total_cached_values: number
  cache_ttl?: number
}

export interface Table {
  id: string
  datasource_id: string
  schema: string
  name: string
  display_name: string
  description?: string
  visible: boolean
  field_order?: 'database' | 'alphabetical' | 'custom' | 'smart'
  row_count: number
  created_at: string
  updated_at: string
}

export interface Column {
  id: string
  table_id: string
  name: string
  display_name: string

  // Types
  type: string  // Database type
  mapped_type?: 'string' | 'number' | 'boolean' | 'datetime' | 'date' | 'json'
  semantic?: SemanticType

  // Metadata
  description?: string
  position: number

  // Visibility
  visibility?: 'everywhere' | 'detail_only' | 'hidden'

  // Filter widget type
  filter_widget_type?: 'search' | 'dropdown' | 'input' | 'none'

  // Constraints
  nullable?: boolean
  primary_key?: boolean
  foreign_key?: boolean
  foreign_table?: string
  foreign_column?: string

  // Fingerprint (statistics)
  distinct_count?: number
  null_count?: number
  min_value?: string
  max_value?: string
  avg_length?: number

  // Cached values (for dropdowns)
  cached_values?: string[]
  values_cached_at?: string

  // Custom mappings
  value_mappings?: Record<string, string>
}

export interface ColumnScanResult {
  column_id: string
  values: string[]
  total_distinct: number
  duration_ms: number
  cached_at: string
}

// Complete Semantic Types (matching Metabase)
export type SemanticType =
  // Keys
  | 'type/PK' | 'type/FK'
  // Numbers
  | 'type/Price' | 'type/Currency' | 'type/Score' | 'type/Percentage' | 'type/Quantity'
  | 'type/Cost' | 'type/GrossMargin' | 'type/Discount'
  // Text
  | 'type/Name' | 'type/Title' | 'type/Description' | 'type/Comment'
  | 'type/Category' | 'type/Company' | 'type/Product' | 'type/Source'
  | 'type/AvatarURL' | 'type/ImageURL' | 'type/URL' | 'type/Email' | 'type/Phone'
  | 'type/SerializedJSON'
  // Dates
  | 'type/CreationDate' | 'type/CreationTime' | 'type/CreationTimestamp'
  | 'type/UpdateDate' | 'type/UpdateTime' | 'type/UpdateTimestamp'
  | 'type/JoinDate' | 'type/JoinTime' | 'type/JoinTimestamp'
  | 'type/Birthdate'
  | 'type/CancelationDate' | 'type/CancelationTime' | 'type/CancelationTimestamp'
  | 'type/DeletionDate' | 'type/DeletionTime' | 'type/DeletionTimestamp'
  // Geo
  | 'type/Latitude' | 'type/Longitude' | 'type/Coordinate'
  | 'type/City' | 'type/State' | 'type/Country' | 'type/ZipCode' | 'type/Address'
  // Legacy compatibility
  | 'pk' | 'fk' | 'name' | 'category' | 'quantity'
  | 'price' | 'percentage' | 'latitude' | 'longitude'
  | 'email' | 'url' | 'image' | 'created_at' | 'updated_at'

// Question types
export interface Question {
  id: string
  name: string
  description?: string
  collection_id?: string
  datasource_id: string
  query_type: 'native' | 'query'
  query: QueryDefinition
  visualization: VisualizationSettings
  created_by?: string
  created_at: string
  updated_at: string
}

export interface QueryDefinition {
  sql?: string
  table?: string
  columns?: string[]
  filters?: Filter[]
  joins?: Join[]
  group_by?: string[]
  order_by?: OrderBy[]
  limit?: number
  aggregations?: Aggregation[]
}

export interface Filter {
  id: string
  column: string
  operator: FilterOperator
  value: any
}

export type FilterOperator =
  | 'equals' | 'not-equals'
  | 'contains' | 'not-contains' | 'starts-with' | 'ends-with'
  | 'is-null' | 'is-not-null'
  | 'greater-than' | 'less-than' | 'greater-or-equal' | 'less-or-equal' | 'between'
  | 'is-in-previous' | 'is-in-next' | 'is-in-current'

export interface Join {
  id: string
  source_table: string
  target_table: string
  type: 'left' | 'inner' | 'right' | 'full'
  conditions: JoinCondition[]
}

export interface JoinCondition {
  source_column: string
  target_column: string
  operator: '=' | '!=' | '>' | '<' | '>=' | '<='
}

export interface OrderBy {
  column: string
  direction: 'asc' | 'desc'
}

export interface Aggregation {
  id: string
  column?: string
  function: 'count' | 'sum' | 'avg' | 'min' | 'max' | 'distinct'
  alias?: string
}

// Visualization types
export interface VisualizationSettings {
  type: VisualizationType
  settings?: Record<string, any>
}

export type VisualizationType =
  | 'table' | 'number' | 'trend' | 'progress' | 'gauge'
  | 'line' | 'area' | 'bar' | 'row' | 'combo' | 'waterfall'
  | 'funnel' | 'pie' | 'donut' | 'scatter' | 'bubble'
  | 'map-pin' | 'map-grid' | 'map-region' | 'pivot' | 'sankey'

// Dashboard types
export interface Dashboard {
  id: string
  name: string
  description?: string
  collection_id?: string
  auto_refresh?: number
  width?: 'fixed' | 'full'
  cards?: DashboardCard[]
  filters?: DashboardFilter[]
  tabs?: DashboardTab[]
  created_by?: string
  created_at: string
  updated_at: string
}

export interface DashboardCard {
  id: string
  dashboard_id: string
  question_id?: string
  card_type: 'question' | 'text' | 'heading' | 'link' | 'action'
  tab_id?: string
  row: number
  col: number
  width: number
  height: number
  title?: string
  content?: string
  settings?: Record<string, any>
}

export interface DashboardFilter {
  id: string
  dashboard_id: string
  name: string
  type: 'time' | 'location' | 'id' | 'category' | 'text' | 'number'
  display_type?: 'search' | 'dropdown' | 'date' | 'input'
  default?: any
  required: boolean
  targets: FilterTarget[]
}

export interface FilterTarget {
  card_id: string
  column_id: string
}

export interface DashboardTab {
  id: string
  dashboard_id: string
  name: string
  position: number
}

// Collection types
export interface Collection {
  id: string
  name: string
  description?: string
  parent_id?: string
  color?: string
  created_by?: string
  created_at: string
}

// Model types
export interface Model {
  id: string
  name: string
  description?: string
  collection_id?: string
  datasource_id: string
  query: QueryDefinition
  columns?: ModelColumn[]
  created_by?: string
  created_at: string
  updated_at: string
}

export interface ModelColumn {
  id: string
  model_id: string
  name: string
  display_name: string
  description?: string
  semantic?: SemanticType
  visible: boolean
}

// Metric types
export interface Metric {
  id: string
  name: string
  description?: string
  table_id: string
  definition: MetricDefinition
  created_by?: string
  created_at: string
  updated_at: string
}

export interface MetricDefinition {
  aggregation: 'count' | 'sum' | 'avg' | 'min' | 'max' | 'distinct-count'
  column?: string
  filter?: Filter[]
}

// Alert types
export interface Alert {
  id: string
  name: string
  question_id: string
  alert_type: 'goal' | 'progress' | 'rows'
  condition: AlertCondition
  channels: AlertChannel[]
  enabled: boolean
  created_by?: string
  created_at: string
}

export interface AlertCondition {
  operator?: 'above' | 'below' | 'reaches'
  value?: number
  row_condition?: 'has-rows' | 'no-rows'
}

export interface AlertChannel {
  type: 'email' | 'slack' | 'webhook'
  targets: string[]
}

// Subscription types
export interface Subscription {
  id: string
  dashboard_id: string
  schedule: string
  format: 'png' | 'pdf' | 'csv'
  recipients: string[]
  enabled: boolean
  created_by?: string
  created_at: string
}

// User types
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'user' | 'viewer'
  created_at: string
  last_login?: string
}

// Query result types
export interface QueryResult {
  columns: ResultColumn[]
  rows: Record<string, any>[]
  row_count: number
  duration_ms: number
  cached?: boolean
}

export interface ResultColumn {
  name: string
  display_name: string
  type: string
}

// Settings types
export interface Settings {
  site_name?: string
  site_url?: string
  admin_email?: string
  enable_public_sharing?: boolean
  enable_embedding?: boolean
  enable_alerts?: boolean
}

// Search types
export interface SearchResult {
  id: string
  type: 'question' | 'dashboard' | 'collection' | 'table' | 'column'
  name: string
  description?: string
  collection?: string
  score: number
}

// Bookmark types
export interface Bookmark {
  id: string
  user_id: string
  item_type: 'question' | 'dashboard' | 'collection'
  item_id: string
  position: number
  created_at: string
}

// Recent item types
export interface RecentItem {
  id: string
  user_id: string
  item_type: 'question' | 'dashboard' | 'collection' | 'table'
  item_id: string
  item_name: string
  viewed_at: string
}

// Pin types (for home page)
export interface Pin {
  id: string
  item_type: 'question' | 'dashboard'
  item_id: string
  position: number
  created_by: string
  created_at: string
}

// Activity types
export interface Activity {
  id: string
  user_id: string
  action: 'view' | 'create' | 'update' | 'delete' | 'query' | 'export'
  item_type: string
  item_id: string
  details?: Record<string, any>
  created_at: string
}
