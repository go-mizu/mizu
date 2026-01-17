import { api } from './client';

// Severity levels for Postgres logs
export type LogSeverity = 'DEBUG' | 'INFO' | 'NOTICE' | 'WARNING' | 'ERROR' | 'FATAL' | 'PANIC';

// Log entry from the API
export interface LogEntry {
  id: string;
  timestamp: string;
  event_message: string;
  request_id?: string;
  method?: string;
  path?: string;
  status_code?: number;
  source: string;
  severity?: LogSeverity;
  user_id?: string;
  user_agent?: string;
  apikey?: string;
  request_headers?: Record<string, string>;
  response_headers?: Record<string, string>;
  duration_ms?: number;
  metadata?: Record<string, any>;
  search?: string;
}

// Log source/collection type
export interface LogSource {
  id: string;
  name: string;
  description: string;
}

// Histogram bucket for charts
export interface LogHistogramBucket {
  timestamp: string;
  count: number;
}

// Saved query
export interface SavedQuery {
  id: string;
  name: string;
  description?: string;
  query_params: Record<string, any>;
  created_at: string;
  updated_at: string;
}

// Query template
export interface QueryTemplate {
  id: string;
  name: string;
  description?: string;
  query_params: Record<string, any>;
  category?: string;
}

// Response types
export interface LogsResponse {
  logs: LogEntry[];
  total: number;
  limit: number;
  offset: number;
}

export interface HistogramResponse {
  buckets: LogHistogramBucket[];
  total: number;
}

// Filter parameters
export interface LogFilter {
  source?: string;
  severity?: LogSeverity;
  severities?: LogSeverity[];
  status_min?: number;
  status_max?: number;
  methods?: string[];
  method?: string;
  path_pattern?: string;
  query?: string;
  regex?: string;
  user_id?: string;
  request_id?: string;
  from?: string;
  to?: string;
  time_range?: string;
  limit?: number;
  offset?: number;
}

// Search request body
export interface LogSearchRequest {
  source?: string;
  severity?: LogSeverity;
  severities?: LogSeverity[];
  status_min?: number;
  status_max?: number;
  methods?: string[];
  path_pattern?: string;
  query?: string;
  regex?: string;
  user_id?: string;
  request_id?: string;
  from?: string;
  to?: string;
  time_range?: string;
  limit?: number;
  offset?: number;
}

// Export params
export interface LogExportParams {
  format: 'json' | 'csv';
  source?: string;
  from?: string;
  to?: string;
  time_range?: string;
}

// Saved query create/update request
export interface SavedQueryRequest {
  name: string;
  description?: string;
  query_params: Record<string, any>;
}

// Build URL search params from filter
function buildSearchParams(filter: LogFilter): URLSearchParams {
  const params = new URLSearchParams();
  if (filter.source) params.set('source', filter.source);
  if (filter.severity) params.set('severity', filter.severity);
  if (filter.severities && filter.severities.length > 0) params.set('severities', filter.severities.join(','));
  if (filter.status_min) params.set('status_min', filter.status_min.toString());
  if (filter.status_max) params.set('status_max', filter.status_max.toString());
  if (filter.method) params.set('method', filter.method);
  if (filter.methods && filter.methods.length > 0) params.set('methods', filter.methods.join(','));
  if (filter.path_pattern) params.set('path', filter.path_pattern);
  if (filter.query) params.set('query', filter.query);
  if (filter.regex) params.set('regex', filter.regex);
  if (filter.user_id) params.set('user_id', filter.user_id);
  if (filter.request_id) params.set('request_id', filter.request_id);
  if (filter.from) params.set('from', filter.from);
  if (filter.to) params.set('to', filter.to);
  if (filter.time_range) params.set('time_range', filter.time_range);
  if (filter.limit) params.set('limit', filter.limit.toString());
  if (filter.offset) params.set('offset', filter.offset.toString());
  return params;
}

export const logsApi = {
  // List logs with optional filters
  listLogs: (filter?: LogFilter): Promise<LogsResponse> => {
    const params = filter ? buildSearchParams(filter) : new URLSearchParams();
    const query = params.toString();
    return api.get(`/api/logs${query ? `?${query}` : ''}`);
  },

  // Get a single log by ID
  getLog: (id: string): Promise<LogEntry> => {
    return api.get(`/api/logs/${id}`);
  },

  // Get log histogram for charts
  getHistogram: (filter?: LogFilter, interval?: string): Promise<HistogramResponse> => {
    const params = filter ? buildSearchParams(filter) : new URLSearchParams();
    if (interval) params.set('interval', interval);
    const query = params.toString();
    return api.get(`/api/logs/histogram${query ? `?${query}` : ''}`);
  },

  // Get available log sources/collections
  listSources: (): Promise<LogSource[]> => {
    return api.get('/api/logs/sources');
  },

  // Search logs with advanced filters
  searchLogs: (request: LogSearchRequest): Promise<LogsResponse> => {
    return api.post('/api/logs/search', request);
  },

  // Export logs as JSON or CSV
  exportLogs: async (params: LogExportParams): Promise<Blob> => {
    const searchParams = new URLSearchParams();
    searchParams.set('format', params.format);
    if (params.source) searchParams.set('source', params.source);
    if (params.from) searchParams.set('from', params.from);
    if (params.to) searchParams.set('to', params.to);
    if (params.time_range) searchParams.set('time_range', params.time_range);
    return api.get(`/api/logs/export?${searchParams.toString()}`, { responseType: 'blob' } as any);
  },

  // List saved queries
  listSavedQueries: (): Promise<SavedQuery[]> => {
    return api.get('/api/logs/queries');
  },

  // Create a saved query
  createSavedQuery: (request: SavedQueryRequest): Promise<SavedQuery> => {
    return api.post('/api/logs/queries', request);
  },

  // Get a saved query by ID
  getSavedQuery: (id: string): Promise<SavedQuery> => {
    return api.get(`/api/logs/queries/${id}`);
  },

  // Update a saved query
  updateSavedQuery: (id: string, request: SavedQueryRequest): Promise<SavedQuery> => {
    return api.put(`/api/logs/queries/${id}`, request);
  },

  // Delete a saved query
  deleteSavedQuery: (id: string): Promise<void> => {
    return api.delete(`/api/logs/queries/${id}`);
  },

  // List query templates
  listQueryTemplates: (): Promise<QueryTemplate[]> => {
    return api.get('/api/logs/templates');
  },
};

// Legacy exports for backward compatibility
export type Log = LogEntry;
export type LogType = LogSource;
