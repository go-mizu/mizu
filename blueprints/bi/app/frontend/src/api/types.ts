// Data Source types
export interface DataSource {
  id: string
  name: string
  engine: 'sqlite' | 'postgres' | 'mysql'
  host?: string
  port?: number
  database: string
  username?: string
  ssl: boolean
  created_at: string
  updated_at: string
}

export interface Table {
  id: string
  datasource_id: string
  schema: string
  name: string
  display_name: string
  description?: string
  row_count: number
  created_at: string
  updated_at: string
}

export interface Column {
  id: string
  table_id: string
  name: string
  display_name: string
  type: 'string' | 'number' | 'boolean' | 'datetime' | 'date'
  semantic?: SemanticType
  description?: string
  position: number
}

export type SemanticType =
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
