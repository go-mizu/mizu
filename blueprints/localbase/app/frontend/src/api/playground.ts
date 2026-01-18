import { api } from './client';

// Types
export interface EndpointCategory {
  name: string;
  icon: string;
  description: string;
  endpoints: Endpoint[];
}

export interface Endpoint {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  path: string;
  description: string;
  category: string;
  parameters?: Parameter[];
  requestBody?: Record<string, any>;
  example?: string;
}

export interface Parameter {
  name: string;
  type: string;
  description: string;
  required?: boolean;
  example?: string;
}

export interface TableInfo {
  schema: string;
  name: string;
  columns: ColumnInfo[];
  rls_enabled: boolean;
}

export interface ColumnInfo {
  name: string;
  type: string;
  is_nullable: boolean;
  is_primary_key: boolean;
}

export interface FunctionInfo {
  schema: string;
  name: string;
  arguments?: string;
  return_type: string;
}

export interface ExecuteRequest {
  method: string;
  path: string;
  headers: Record<string, string>;
  query: Record<string, string>;
  body: any;
}

export interface ExecuteResponse {
  status: number;
  statusText: string;
  headers: Record<string, string>;
  body: any;
  duration_ms: number;
}

export interface RequestHistoryEntry {
  id: string;
  method: string;
  path: string;
  status: number;
  duration_ms: number;
  timestamp: string;
  request: ExecuteRequest;
  response: ExecuteResponse;
}

export interface TableDocs {
  table: string;
  schema: string;
  columns: Array<{
    name: string;
    type: string;
    is_nullable: boolean;
    is_primary_key: boolean;
    is_unique: boolean;
    default_value: string | null;
    comment: string | null;
  }>;
  endpoints: Array<{
    method: string;
    path: string;
    description: string;
    parameters?: Array<{
      name: string;
      type: string;
      description: string;
    }>;
  }>;
  rls_enabled: boolean;
}

// API functions
export const playgroundApi = {
  // Get all available endpoints
  async getEndpoints(): Promise<{ categories: EndpointCategory[] }> {
    return api.get('/api/playground/endpoints');
  },

  // Get tables for dynamic REST API docs
  async getTables(): Promise<{ tables: TableInfo[] }> {
    return api.get('/api/playground/tables');
  },

  // Get available PostgreSQL RPC functions
  async getFunctions(): Promise<{ functions: FunctionInfo[] }> {
    return api.get('/api/playground/functions');
  },

  // Get auto-generated docs for a specific table
  async getTableDocs(schema: string, table: string): Promise<TableDocs> {
    return api.get(`/api/playground/docs/${schema}/${table}`);
  },

  // Execute an API request
  async execute(request: ExecuteRequest): Promise<ExecuteResponse> {
    return api.post('/api/playground/execute', request);
  },

  // Get request history
  async getHistory(limit = 50, offset = 0): Promise<{ history: RequestHistoryEntry[]; total: number }> {
    return api.get(`/api/playground/history?limit=${limit}&offset=${offset}`);
  },

  // Save request to history
  async saveHistory(entry: Partial<RequestHistoryEntry>): Promise<RequestHistoryEntry> {
    return api.post('/api/playground/history', entry);
  },

  // Clear request history
  async clearHistory(): Promise<{ status: string }> {
    return api.delete('/api/playground/history');
  },
};
