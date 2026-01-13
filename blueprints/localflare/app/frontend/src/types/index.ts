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
  value: unknown
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
  batch_size?: number        // Frontend field name
  max_batch_size?: number    // Backend field name
  max_batch_timeout: number
  message_retention_seconds?: number  // Frontend field name
  message_ttl?: number                // Backend field name
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
  metadata?: Record<string, unknown>
  namespace?: string
}

export interface VectorQueryRequest {
  vector?: number[]
  text?: string
  topK: number
  namespace?: string
  returnValues: boolean
  returnMetadata: boolean
  filter?: Record<string, unknown>
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
  rows: unknown[][]
  row_count: number
  execution_time_ms: number
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
  request?: unknown
  response?: unknown
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
