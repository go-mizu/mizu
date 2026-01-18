// User types
export interface User {
  id: string;
  aud?: string;
  role: string;
  email: string;
  phone: string;
  email_confirmed_at?: string;
  phone_confirmed_at?: string;
  last_sign_in_at?: string;
  app_metadata: Record<string, any>;
  user_metadata: Record<string, any>;
  identities?: Identity[];
  created_at: string;
  updated_at: string;
  is_anonymous: boolean;
}

export interface Identity {
  identity_id: string;
  id: string;
  user_id: string;
  identity_data: Record<string, any>;
  provider: string;
  last_sign_in_at?: string;
  created_at: string;
  updated_at: string;
  email?: string;
}

export interface Session {
  id: string;
  user_id: string;
  created_at: string;
  updated_at: string;
  aal: string;
  not_after: string;
}

// Storage types
export interface Bucket {
  id: string;
  name: string;
  public: boolean;
  file_size_limit?: number;
  allowed_mime_types?: string[];
  created_at: string;
  updated_at: string;
}

export interface StorageObject {
  id: string;
  bucket_id: string;
  name: string;
  owner?: string;
  path_tokens?: string[];
  version?: string;
  metadata?: Record<string, string>;
  content_type?: string;
  size: number;
  created_at: string;
  updated_at: string;
  last_accessed_at?: string;
  // Additional fields from enhanced API
  etag?: string;
  cache_control?: string;
}

// Database types
export interface Table {
  id: number;
  schema: string;
  name: string;
  row_count: number;
  size_bytes: number;
  comment?: string;
  rls_enabled: boolean;
  columns?: Column[];
}

export interface Column {
  name: string;
  type: string;
  default_value?: string;
  is_nullable: boolean;
  is_primary_key: boolean;
  is_unique: boolean;
  is_identity?: boolean;
  comment?: string;
}

export interface Policy {
  id: number;
  name: string;
  schema: string;
  table: string;
  command: string;
  definition: string;
  check_expression?: string;
  roles: string[];
}

export interface Extension {
  name: string;
  installed_version?: string;
  default_version: string;
  comment?: string;
}

export interface QueryResult {
  query_id?: string;
  columns: string[];
  rows: Record<string, any>[];
  row_count: number;
  duration_ms: number;
}

export interface QueryHistoryEntry {
  id: string;
  query: string;
  executed_at: string;
  duration_ms: number;
  role: string;
  row_count: number;
  success: boolean;
  error?: string;
}

export interface SQLSnippet {
  id: string;
  name: string;
  query: string;
  folder_id?: string;
  is_shared: boolean;
  created_at: string;
  updated_at: string;
}

export interface SQLFolder {
  id: string;
  name: string;
  parent_id?: string;
  created_at: string;
}

// Functions types
export interface EdgeFunction {
  id: string;
  name: string;
  slug: string;
  version: number;
  status: 'active' | 'inactive';
  entrypoint: string;
  import_map?: string;
  verify_jwt: boolean;
  draft_source?: string;
  draft_import_map?: string;
  created_at: string;
  updated_at: string;
  latest_deployment?: Deployment;
}

export interface Deployment {
  id: string;
  function_id: string;
  version: number;
  source_code: string;
  bundle_path?: string;
  status: 'pending' | 'deploying' | 'deployed' | 'failed';
  deployed_at: string;
}

export interface Secret {
  id: string;
  name: string;
  created_at: string;
}

export interface FunctionLog {
  id: string;
  function_id: string;
  request_id?: string;
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  duration_ms?: number;
  status_code?: number;
  region?: string;
  metadata?: Record<string, any>;
}

export interface FunctionMetrics {
  function_id: string;
  period: string;
  invocations: {
    total: number;
    success: number;
    error: number;
    by_hour: Array<{ hour: string; count: number }>;
  };
  latency: {
    avg: number;
    p50?: number;
    p95?: number;
    p99?: number;
  };
}

export interface FunctionTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  icon: string;
}

export interface FunctionSource {
  function_id: string;
  entrypoint: string;
  source_code: string;
  import_map?: string;
  version: number;
  is_draft: boolean;
}

export interface FunctionTestRequest {
  method: string;
  path: string;
  headers: Record<string, string>;
  body?: any;
}

export interface FunctionTestResponse {
  status: number;
  headers: Record<string, string>;
  body: any;
  duration_ms: number;
  logs?: Array<{
    level: string;
    message: string;
    timestamp: string;
  }>;
}

// Realtime types
export interface Channel {
  id: string;
  name: string;
  inserted_at: string;
}

export interface RealtimeStats {
  connections: number;
  channels: number;
  messagesPerSecond: number;
  server_time: string;
}

// Dashboard types
export interface DashboardStats {
  users: {
    total: number;
  };
  storage: {
    buckets: number;
  };
  functions: {
    total: number;
    active: number;
  };
  database: {
    tables: number;
  };
  timestamp: string;
}

export interface HealthStatus {
  status: 'healthy' | 'unhealthy';
  services: {
    database: boolean;
    auth: boolean;
    storage: boolean;
    realtime: boolean;
  };
  version: string;
  timestamp: string;
}

// API Response types
export interface ApiError {
  code?: number;
  error_code?: string;
  msg?: string;
  message?: string;
  error?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
}

// Schema Visualization types
export interface SchemaVisualizationTable {
  id: number;
  schema: string;
  name: string;
  comment?: string;
  columns: SchemaVisualizationColumn[];
}

export interface SchemaVisualizationColumn {
  name: string;
  type: string;
  is_nullable: boolean;
  is_primary_key: boolean;
  is_unique: boolean;
  is_identity: boolean;
  default_value?: string;
}

export interface SchemaVisualizationRelationship {
  id: number;
  source_schema: string;
  source_table: string;
  source_columns: string[];
  target_schema: string;
  target_table: string;
  target_columns: string[];
  constraint_name: string;
}

export interface SchemaVisualizationData {
  tables: SchemaVisualizationTable[];
  relationships: SchemaVisualizationRelationship[];
}
