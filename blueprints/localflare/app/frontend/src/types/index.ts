// Generic JSON value type for dynamic data
export type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue }

// Durable Objects
export interface DurableObjectNamespace {
  id: string
  name: string
  class_name: string
  script_name?: string
  created_at: string
  object_count?: number
}

export interface DurableObjectInstance {
  id: string
  namespace_id: string
  last_access: string
  storage_size: number
}

export interface DurableObjectStorage {
  key: string
  value: JsonValue
  updated_at: string
}

// Queues
export interface Queue {
  id: string
  name: string
  created_at: string
  message_count?: number  // Optional - fetched separately via stats endpoint
  ready_count?: number
  delayed_count?: number
  failed_count?: number
  settings: QueueSettings
  consumers?: QueueConsumer[]  // Optional - may not be included in list response
}

export interface QueueSettings {
  max_retries: number
  max_batch_size: number
  max_batch_timeout: number
  message_ttl: number
  delivery_delay: number
}

export interface QueueConsumer {
  id: string
  queue_id: string
  script_name: string
  consumer_type: 'worker' | 'http'
  batch_size: number
  max_retries: number
  status: 'active' | 'paused'
}

export interface QueueMessage {
  id: string
  body: string
  content_type: 'json' | 'text' | 'bytes'
  delay_seconds?: number
}

// Vectorize
export interface VectorIndex {
  id: string
  name: string
  dimensions?: number
  metric?: 'cosine' | 'euclidean' | 'dot-product'
  description?: string
  created_at: string
  vector_count?: number
  namespace_count?: number
}

export interface VectorMatch {
  id: string
  score: number
  values?: number[]
  metadata?: Record<string, JsonValue>
  namespace?: string
}

export interface VectorQueryRequest {
  vector?: number[]
  text?: string
  topK: number
  namespace?: string
  returnValues: boolean
  returnMetadata: boolean
  filter?: Record<string, JsonValue>
}

export interface VectorInsertItem {
  id: string
  values: number[]
  metadata?: Record<string, JsonValue>
  namespace?: string
}

// Analytics Engine
export interface AnalyticsDataset {
  id: string
  name: string
  created_at: string
  data_points: number
  estimated_size_bytes: number
  last_write: string
}

export interface AnalyticsQueryResult {
  columns: string[]
  rows: JsonValue[][]
  row_count: number
  execution_time_ms: number
}

export interface AnalyticsDataPoint {
  indexes?: string[]
  doubles?: number[]
  blobs?: string[]
  timestamp?: string
}

// Workers AI
export interface AIModel {
  id: string
  name: string
  description?: string
  task: 'text-generation' | 'text-embeddings' | 'image-generation' | 'speech-to-text' | 'translation' | 'summarization'
  properties?: {
    max_tokens?: number
    context_length?: number
  }
}

export interface AIInferenceRequest {
  model: string
  prompt?: string
  messages?: Array<{ role: 'system' | 'user' | 'assistant'; content: string }>
  max_tokens?: number
  temperature?: number
  stream?: boolean
}

export interface AIInferenceResponse {
  response: string
  usage?: {
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
  latency_ms: number
}

// AI Gateway
// Note: Backend returns flat fields, frontend may use nested structure in fallback data
export interface AIGateway {
  id: string
  name: string
  created_at: string
  // Nested structure (used in fallback/mock data)
  settings?: AIGatewaySettings
  stats?: AIGatewayStats
  // Flat fields from backend API
  collect_logs?: boolean
  cache_enabled?: boolean
  cache_ttl?: number
  rate_limit_enabled?: boolean
  rate_limit_count?: number
  rate_limit_period?: number
}

export interface AIGatewaySettings {
  cache_enabled: boolean
  cache_ttl: number
  rate_limit_enabled: boolean
  rate_limit: number
  rate_limit_period: string
  logging_enabled: boolean
  retry_enabled?: boolean
  retry_count?: number
}

export interface AIGatewayStats {
  total_requests: number
  cached_requests: number
  error_count: number
  total_tokens: number
  total_cost: number
}

export interface AIGatewayLogRequest {
  model?: string
  messages?: Array<{ role: string; content: string }>
  prompt?: string
  max_tokens?: number
  temperature?: number
}

export interface AIGatewayLogResponse {
  id?: string
  choices?: Array<{ message?: { content: string }; text?: string }>
  usage?: { prompt_tokens: number; completion_tokens: number; total_tokens: number }
  error?: { message: string; code: string }
}

export interface AIGatewayLog {
  id: string
  gateway_id: string
  timestamp: string
  provider: string
  model: string
  status: number | 'CACHED'
  latency_ms: number
  tokens: number
  cost: number
  cached: boolean
  request?: AIGatewayLogRequest
  response?: AIGatewayLogResponse
}

// Hyperdrive
export interface HyperdriveConfig {
  id: string
  name: string
  created_at: string
  origin?: HyperdriveOrigin
  caching?: HyperdriveCaching
  status?: 'connected' | 'disconnected' | 'idle'
}

export interface HyperdriveOrigin {
  scheme: 'postgres' | 'mysql'
  host: string
  port: number
  database: string
  user: string
}

export interface HyperdriveCaching {
  enabled: boolean
  max_age: number
  stale_while_revalidate: number
}

export interface HyperdriveStats {
  active_connections: number
  idle_connections: number
  total_connections: number
  queries_per_second: number
  cache_hit_rate: number
}

// Cron Triggers
export interface CronTrigger {
  id: string
  cron: string
  script_name: string
  enabled: boolean
  created_at: string
  last_run?: string
  next_run?: string
}

export interface CronExecution {
  id: string
  trigger_id: string
  scheduled_at: string
  started_at: string
  finished_at?: string
  duration_ms?: number
  status: 'success' | 'failed' | 'running'
  error?: string
}

// Dashboard Stats
export interface DashboardStats {
  durable_objects: {
    namespaces: number
    objects: number
  }
  queues: {
    count: number
    total_messages: number
  }
  vectorize: {
    indexes: number
    total_vectors: number
  }
  analytics: {
    datasets: number
    data_points: number
  }
  ai: {
    requests_today: number
    tokens_today: number
  }
  ai_gateway: {
    gateways: number
    requests_today: number
  }
  hyperdrive: {
    configs: number
    active_connections: number
  }
  cron: {
    triggers: number
    executions_today: number
  }
}

export interface TimeSeriesData {
  timestamp: string
  value: number
}

export interface SystemStatus {
  service: string
  status: 'online' | 'degraded' | 'offline'
  latency_ms?: number
}

export interface ActivityEvent {
  id: string
  type: string
  message: string
  timestamp: string
  service: string
}

// API Response wrapper
export interface ApiResponse<T> {
  success: boolean
  result?: T
  errors?: Array<{ code: number; message: string }>
}

// Pagination
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

// Workers
export interface Worker {
  id: string
  name: string
  created_at: string
  modified_at: string
  status: 'active' | 'inactive' | 'error'
  routes?: string[]
  bindings?: WorkerBinding[]
  environment_variables?: Record<string, string>
  compatibility_date?: string
  usage_model?: string
  code?: string
}

export interface WorkerBinding {
  type: 'kv_namespace' | 'd1' | 'r2_bucket' | 'durable_object' | 'queue' | 'vectorize' | 'ai' | 'service' | 'secret' | 'text'
  name: string
  namespace_id?: string
  database_id?: string
  bucket_name?: string
}

export interface WorkerCreateRequest {
  name: string
  main_module?: string
  compatibility_date?: string
  code?: string
}

export interface WorkerBindingRequest {
  type: WorkerBinding['type']
  name: string
  namespace_id?: string
}

export interface WorkerVersion {
  id: string
  version: string
  created_at: string
  status: 'active' | 'inactive'
  message?: string
}

// KV
export interface KVNamespace {
  id: string
  title: string
  created_at: string
  key_count?: number
  storage_size?: number
}

export interface KVKey {
  name: string
  expiration?: number
  metadata?: Record<string, JsonValue>
}

// R2
export interface R2Bucket {
  name: string
  created_at: string
  location?: string
  object_count?: number
  storage_size?: number
  public_access?: boolean
}

export interface R2Object {
  key: string
  size: number
  last_modified: string
  etag: string
}

// D1
export interface D1Database {
  uuid: string
  name: string
  created_at: string
  version?: string
  num_tables?: number
  file_size?: number
}

export interface D1Table {
  name: string
  sql: string
  row_count?: number
}

export interface D1QueryResult {
  success: boolean
  results: Record<string, JsonValue>[]
  meta?: {
    duration: number
    rows_read: number
    rows_written: number
    changes: number
  }
}

// Pages
export interface PagesProject {
  name: string
  subdomain: string
  created_at: string
  production_branch: string
  latest_deployment?: PagesDeployment
  domains?: string[]
}

export interface PagesDeploymentTrigger {
  type: string
  metadata?: {
    branch?: string
    commit_hash?: string
    commit_message?: string
  }
}

export interface PagesDeployment {
  id: string
  url: string
  environment: 'production' | 'preview'
  deployment_trigger: PagesDeploymentTrigger
  created_at: string
  status: 'building' | 'success' | 'failed'
}

// Images
export interface CloudflareImage {
  id: string
  filename: string
  uploaded: string
  variants?: string[]
  meta?: {
    width?: number
    height?: number
  }
}

export interface ImageVariant {
  id: string
  name: string
  options: {
    fit: string
    width: number
    height: number
  }
  never_require_signed_urls?: boolean
}

// Stream
export interface StreamVideo {
  uid: string
  name: string
  created: string
  duration: number
  size: number
  status: {
    state: 'pendingupload' | 'queued' | 'inprogress' | 'ready' | 'error'
  }
  thumbnail?: string
  playback?: {
    hls?: string
    dash?: string
  }
}

export interface LiveInput {
  uid: string
  name: string
  created: string
  status: 'connected' | 'disconnected'
  rtmps: {
    url: string
    streamKey: string
  }
}

// Observability
export interface LogEntry {
  timestamp: string
  level: 'debug' | 'info' | 'warn' | 'error'
  message: string
  worker: string
  request_id: string
  duration_ms: number
}

export interface Trace {
  trace_id: string
  root_span: string
  worker: string
  timestamp: string
  duration_ms: number
  status: 'ok' | 'error'
  spans?: TraceSpan[]
}

export interface TraceSpan {
  name: string
  duration_ms: number
  start_ms: number
}

// Settings
export interface APIToken {
  id: string
  name: string
  permissions: string[]
  created_at: string
  last_used?: string
  status: 'active' | 'revoked'
}

export interface AccountMember {
  id: string
  email: string
  name?: string
  role: 'owner' | 'admin' | 'member'
  status: 'active' | 'pending'
  joined_at: string
}
