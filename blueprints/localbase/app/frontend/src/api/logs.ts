import { api } from './client';

export interface Log {
  id: string;
  type: string;
  level: string;
  message: string;
  metadata?: Record<string, any>;
  timestamp: string;
}

export interface LogType {
  id: string;
  name: string;
  description: string;
}

export interface LogsResponse {
  logs: Log[];
  total: number;
  page: number;
  limit: number;
}

export interface LogSearchRequest {
  type?: string;
  levels?: string[];
  query?: string;
  start_time?: string;
  end_time?: string;
  limit?: number;
  offset?: number;
}

export const logsApi = {
  // List logs with optional filters
  listLogs: (params?: {
    type?: string;
    level?: string;
    limit?: number;
    offset?: number;
  }): Promise<LogsResponse> => {
    const searchParams = new URLSearchParams();
    if (params?.type) searchParams.set('type', params.type);
    if (params?.level) searchParams.set('level', params.level);
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());
    const query = searchParams.toString();
    return api.get(`/api/logs${query ? `?${query}` : ''}`);
  },

  // Get available log types
  listLogTypes: (): Promise<LogType[]> => {
    return api.get('/api/logs/types');
  },

  // Search logs with advanced filters
  searchLogs: (request: LogSearchRequest): Promise<LogsResponse> => {
    return api.post('/api/logs/search', request);
  },

  // Export logs
  exportLogs: (format: 'json' | 'csv', params?: {
    type?: string;
    level?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<Blob> => {
    const searchParams = new URLSearchParams();
    searchParams.set('format', format);
    if (params?.type) searchParams.set('type', params.type);
    if (params?.level) searchParams.set('level', params.level);
    if (params?.start_time) searchParams.set('start_time', params.start_time);
    if (params?.end_time) searchParams.set('end_time', params.end_time);
    return api.get(`/api/logs/export?${searchParams.toString()}`, { responseType: 'blob' } as any);
  },
};
