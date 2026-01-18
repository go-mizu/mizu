import { api } from './client';
import type { Table, Column, Policy, Extension, QueryResult } from '../types';

export interface CreateTableRequest {
  schema: string;
  name: string;
  columns: Column[];
}

export interface CreateColumnRequest {
  name: string;
  type: string;
  default_value?: string;
  is_nullable?: boolean;
  is_primary_key?: boolean;
  is_unique?: boolean;
}

export interface CreatePolicyRequest {
  name: string;
  schema: string;
  table: string;
  command: 'ALL' | 'SELECT' | 'INSERT' | 'UPDATE' | 'DELETE';
  definition: string;
  check_expression?: string;
  roles?: string[];
}

export const databaseApi = {
  // Schema operations
  listSchemas: (): Promise<string[]> => {
    return api.get<string[]>('/api/database/schemas');
  },

  createSchema: (name: string): Promise<void> => {
    return api.post('/api/database/schemas', { name });
  },

  // Table operations
  listTables: (schema = 'public'): Promise<Table[]> => {
    return api.get<Table[]>(`/api/database/tables?schema=${schema}`);
  },

  getTable: (schema: string, name: string): Promise<Table> => {
    return api.get<Table>(`/api/database/tables/${schema}/${name}`);
  },

  createTable: (data: CreateTableRequest): Promise<void> => {
    return api.post('/api/database/tables', data);
  },

  dropTable: (schema: string, name: string): Promise<void> => {
    return api.delete(`/api/database/tables/${schema}/${name}`);
  },

  // Column operations
  listColumns: (schema: string, table: string): Promise<Column[]> => {
    return api.get<Column[]>(`/api/database/tables/${schema}/${table}/columns`);
  },

  addColumn: (schema: string, table: string, column: CreateColumnRequest): Promise<void> => {
    return api.post(`/api/database/tables/${schema}/${table}/columns`, column);
  },

  alterColumn: (
    schema: string,
    table: string,
    columnName: string,
    updates: Partial<Column>
  ): Promise<void> => {
    return api.put(`/api/database/tables/${schema}/${table}/columns/${columnName}`, updates);
  },

  dropColumn: (schema: string, table: string, column: string): Promise<void> => {
    return api.delete(`/api/database/tables/${schema}/${table}/columns/${column}`);
  },

  // Policy operations
  listPolicies: (schema: string, table: string): Promise<Policy[]> => {
    return api.get<Policy[]>(`/api/database/policies/${schema}/${table}`);
  },

  createPolicy: (data: CreatePolicyRequest): Promise<void> => {
    return api.post('/api/database/policies', data);
  },

  dropPolicy: (schema: string, table: string, name: string): Promise<void> => {
    return api.delete(`/api/database/policies/${schema}/${table}/${name}`);
  },

  // Extension operations
  listExtensions: (): Promise<Extension[]> => {
    return api.get<Extension[]>('/api/database/extensions');
  },

  enableExtension: (name: string): Promise<void> => {
    return api.post('/api/database/extensions', { name });
  },

  // Query execution
  executeQuery: (query: string): Promise<QueryResult> => {
    return api.post<QueryResult>('/api/database/query', { query });
  },

  // PostgREST-compatible REST API
  selectTable: (table: string, query?: string): Promise<any[]> => {
    const path = query ? `/rest/v1/${table}?${query}` : `/rest/v1/${table}`;
    return api.get<any[]>(path);
  },

  insertRow: (table: string, data: Record<string, any>): Promise<any> => {
    return api.post(`/rest/v1/${table}`, data, {
      headers: { 'Prefer': 'return=representation' },
    });
  },

  updateRow: (table: string, query: string, data: Record<string, any>): Promise<any> => {
    return api.patch(`/rest/v1/${table}?${query}`, data, {
      headers: { 'Prefer': 'return=representation' },
    });
  },

  deleteRow: (table: string, query: string): Promise<void> => {
    return api.delete(`/rest/v1/${table}?${query}`);
  },

  // Enhanced Table Editor API
  getTableData: async (
    schema: string,
    table: string,
    options?: {
      limit?: number;
      offset?: number;
      order?: string;
      select?: string;
      filters?: Record<string, string>;
      includeCount?: boolean;
    }
  ): Promise<{ data: any[]; totalCount?: number }> => {
    const params = new URLSearchParams();
    if (options?.limit) params.set('limit', String(options.limit));
    if (options?.offset) params.set('offset', String(options.offset));
    if (options?.order) params.set('order', options.order);
    if (options?.select) params.set('select', options.select);
    if (options?.includeCount) params.set('count', 'true');
    if (options?.filters) {
      Object.entries(options.filters).forEach(([key, value]) => {
        params.set(key, value);
      });
    }
    const queryString = params.toString();
    const path = `/api/database/tables/${schema}/${table}/data${queryString ? '?' + queryString : ''}`;

    // Use fetch directly to access headers
    const serviceKey = localStorage.getItem('supabase_service_key') || 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJsb2NhbGJhc2UiLCJyb2xlIjoic2VydmljZV9yb2xlIiwiaWF0IjoxNzA0MDY3MjAwLCJleHAiOjE4NjE4MzM2MDB9.service_role_key_signature';
    const response = await fetch(path, {
      headers: {
        'Content-Type': 'application/json',
        'apikey': serviceKey,
        'Authorization': `Bearer ${serviceKey}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch table data: ${response.statusText}`);
    }

    const data = await response.json();
    const totalCount = response.headers.get('X-Total-Count');

    return {
      data,
      totalCount: totalCount ? parseInt(totalCount, 10) : undefined,
    };
  },

  exportTableData: (
    schema: string,
    table: string,
    format: 'json' | 'csv' | 'sql' = 'json',
    filters?: Record<string, string>
  ): string => {
    const params = new URLSearchParams();
    params.set('format', format);
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        params.set(key, value);
      });
    }
    return `/api/database/tables/${schema}/${table}/export?${params.toString()}`;
  },

  bulkOperation: (
    schema: string,
    table: string,
    operation: 'delete' | 'update',
    ids: any[],
    options?: { column?: string; data?: Record<string, any> }
  ): Promise<{ operation: string; rows_affected: number }> => {
    return api.post(`/api/database/tables/${schema}/${table}/bulk`, {
      operation,
      ids,
      column: options?.column || 'id',
      data: options?.data,
    });
  },
};
