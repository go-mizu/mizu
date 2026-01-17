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
  columns: string[];
  rows: Record<string, any>[];
  row_count: number;
  duration_ms: number;
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
  created_at: string;
  updated_at: string;
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
