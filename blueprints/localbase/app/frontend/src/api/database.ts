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
};
