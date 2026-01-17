import { api } from './client';

// Types for pgmeta API
export interface PGIndex {
  id: number;
  schema: string;
  table: string;
  name: string;
  columns: string[];
  is_unique: boolean;
  is_primary: boolean;
  using: string;
  definition: string;
  size: string;
}

export interface PGView {
  id: number;
  schema: string;
  name: string;
  definition: string;
  is_materialized: boolean;
  columns: Array<{ name: string; type: string }>;
}

export interface PGTrigger {
  id: number;
  schema: string;
  table: string;
  name: string;
  function_schema: string;
  function_name: string;
  events: string[];
  activation: string; // BEFORE, AFTER, INSTEAD OF
  orientation: string; // ROW, STATEMENT
  condition: string | null;
  enabled: boolean;
}

export interface PGRole {
  id: number;
  name: string;
  is_superuser: boolean;
  can_create_db: boolean;
  can_create_role: boolean;
  inherit: boolean;
  can_login: boolean;
  is_replication_role: boolean;
  can_bypass_rls: boolean;
  active_connections: number;
  connection_limit: number;
  password: string | null;
  valid_until: string | null;
  config: string | null;
}

export interface PGType {
  id: number;
  schema: string;
  name: string;
  type: string; // enum, composite, range, etc.
  values?: string[]; // for enums
  attributes?: Array<{ name: string; type: string }>; // for composites
}

export interface PGPublication {
  id: number;
  name: string;
  owner: string;
  tables: string[] | null;
  publish_insert: boolean;
  publish_update: boolean;
  publish_delete: boolean;
  publish_truncate: boolean;
}

export interface PGPrivilege {
  schema: string;
  table: string;
  grantee: string;
  privilege_type: string;
  is_grantable: boolean;
}

export interface PGConstraint {
  id: number;
  schema: string;
  table: string;
  name: string;
  type: string; // p (primary), f (foreign), u (unique), c (check), x (exclusion)
  definition: string;
}

export interface PGForeignKey {
  id: number;
  schema: string;
  table: string;
  name: string;
  columns: string[];
  target_schema: string;
  target_table: string;
  target_columns: string[];
  update_action: string;
  delete_action: string;
}

export interface PGDBFunction {
  id: number;
  schema: string;
  name: string;
  language: string;
  return_type: string;
  argument_types: string;
  definition: string;
  is_security_definer: boolean;
  volatility: string;
}

export interface CreateIndexRequest {
  schema: string;
  table: string;
  name: string;
  columns: string[];
  unique?: boolean;
  using?: string;
  where?: string;
}

export interface CreateViewRequest {
  schema: string;
  name: string;
  definition: string;
}

export interface CreateTriggerRequest {
  schema: string;
  table: string;
  name: string;
  function_name: string;
  function_schema?: string;
  activation: 'BEFORE' | 'AFTER' | 'INSTEAD OF';
  events: Array<'INSERT' | 'UPDATE' | 'DELETE' | 'TRUNCATE'>;
  orientation?: 'ROW' | 'STATEMENT';
  condition?: string;
}

export interface CreateRoleRequest {
  name: string;
  password?: string;
  is_superuser?: boolean;
  can_create_db?: boolean;
  can_create_role?: boolean;
  can_login?: boolean;
  inherit?: boolean;
  connection_limit?: number;
  valid_until?: string;
}

export interface CreateTypeRequest {
  schema: string;
  name: string;
  type: 'enum' | 'composite';
  values?: string[]; // for enums
  attributes?: Array<{ name: string; type: string }>; // for composites
}

export interface CreatePublicationRequest {
  name: string;
  tables?: string[];
  publish_insert?: boolean;
  publish_update?: boolean;
  publish_delete?: boolean;
  publish_truncate?: boolean;
}

export const pgmetaApi = {
  // Version
  getVersion: (): Promise<{ version: string; version_number: number }> => {
    return api.get('/api/pg/config/version');
  },

  // Indexes
  listIndexes: (schemas = 'public'): Promise<PGIndex[]> => {
    return api.get(`/api/pg/indexes?included_schemas=${schemas}`);
  },

  createIndex: (data: CreateIndexRequest): Promise<PGIndex> => {
    return api.post('/api/pg/indexes', data);
  },

  dropIndex: (schema: string, name: string): Promise<void> => {
    return api.delete(`/api/pg/indexes/${schema}.${name}`);
  },

  // Views
  listViews: (schemas = 'public'): Promise<PGView[]> => {
    return api.get(`/api/pg/views?included_schemas=${schemas}`);
  },

  createView: (data: CreateViewRequest): Promise<PGView> => {
    return api.post('/api/pg/views', data);
  },

  updateView: (id: string, definition: string): Promise<PGView> => {
    return api.patch(`/api/pg/views/${id}`, { definition });
  },

  dropView: (schema: string, name: string): Promise<void> => {
    return api.delete(`/api/pg/views/${schema}.${name}`);
  },

  // Materialized Views
  listMaterializedViews: (schemas = 'public'): Promise<PGView[]> => {
    return api.get(`/api/pg/materialized-views?included_schemas=${schemas}`);
  },

  createMaterializedView: (data: CreateViewRequest): Promise<PGView> => {
    return api.post('/api/pg/materialized-views', data);
  },

  refreshMaterializedView: (schema: string, name: string): Promise<void> => {
    return api.post(`/api/pg/materialized-views/${schema}.${name}/refresh`, {});
  },

  dropMaterializedView: (schema: string, name: string): Promise<void> => {
    return api.delete(`/api/pg/materialized-views/${schema}.${name}`);
  },

  // Triggers
  listTriggers: (schemas = 'public'): Promise<PGTrigger[]> => {
    return api.get(`/api/pg/triggers?included_schemas=${schemas}`);
  },

  createTrigger: (data: CreateTriggerRequest): Promise<PGTrigger> => {
    return api.post('/api/pg/triggers', data);
  },

  dropTrigger: (schema: string, table: string, name: string): Promise<void> => {
    return api.delete(`/api/pg/triggers/${schema}.${table}.${name}`);
  },

  // Roles
  listRoles: (): Promise<PGRole[]> => {
    return api.get('/api/pg/roles');
  },

  createRole: (data: CreateRoleRequest): Promise<PGRole> => {
    return api.post('/api/pg/roles', data);
  },

  updateRole: (id: string, data: Partial<CreateRoleRequest>): Promise<PGRole> => {
    return api.patch(`/api/pg/roles/${id}`, data);
  },

  dropRole: (name: string): Promise<void> => {
    return api.delete(`/api/pg/roles/${name}`);
  },

  // Types
  listTypes: (schemas = 'public'): Promise<PGType[]> => {
    return api.get(`/api/pg/types?included_schemas=${schemas}`);
  },

  createType: (data: CreateTypeRequest): Promise<PGType> => {
    return api.post('/api/pg/types', data);
  },

  dropType: (schema: string, name: string): Promise<void> => {
    return api.delete(`/api/pg/types/${schema}.${name}`);
  },

  // Publications
  listPublications: (): Promise<PGPublication[]> => {
    return api.get('/api/pg/publications');
  },

  createPublication: (data: CreatePublicationRequest): Promise<PGPublication> => {
    return api.post('/api/pg/publications', data);
  },

  dropPublication: (name: string): Promise<void> => {
    return api.delete(`/api/pg/publications/${name}`);
  },

  // Privileges
  listTablePrivileges: (schemas = 'public'): Promise<PGPrivilege[]> => {
    return api.get(`/api/pg/table-privileges?included_schemas=${schemas}`);
  },

  listColumnPrivileges: (schemas = 'public'): Promise<PGPrivilege[]> => {
    return api.get(`/api/pg/column-privileges?included_schemas=${schemas}`);
  },

  // Constraints
  listConstraints: (schemas = 'public'): Promise<PGConstraint[]> => {
    return api.get(`/api/pg/constraints?included_schemas=${schemas}`);
  },

  listPrimaryKeys: (schemas = 'public'): Promise<PGConstraint[]> => {
    return api.get(`/api/pg/primary-keys?included_schemas=${schemas}`);
  },

  listForeignKeys: (schemas = 'public'): Promise<PGForeignKey[]> => {
    return api.get(`/api/pg/foreign-keys?included_schemas=${schemas}`);
  },

  listRelationships: (schemas = 'public'): Promise<PGForeignKey[]> => {
    return api.get(`/api/pg/relationships?included_schemas=${schemas}`);
  },

  // Database Functions
  listDatabaseFunctions: (schemas = 'public'): Promise<PGDBFunction[]> => {
    return api.get(`/api/pg/functions?included_schemas=${schemas}`);
  },

  // SQL Utilities
  formatSQL: (query: string): Promise<{ formatted: string }> => {
    return api.post('/api/pg/format', { query });
  },

  explainQuery: (query: string, format = 'json'): Promise<any> => {
    return api.post('/api/pg/explain', { query, format });
  },

  // Type Generators
  generateTypeScript: (schemas = 'public'): Promise<string> => {
    return api.get(`/api/pg/generators/typescript?included_schemas=${schemas}`, { responseType: 'text' } as any);
  },

  generateOpenAPI: (schemas = 'public'): Promise<any> => {
    return api.get(`/api/pg/generators/openapi?included_schemas=${schemas}`);
  },

  generateGo: (schemas = 'public', packageName = 'models'): Promise<string> => {
    return api.get(`/api/pg/generators/go?included_schemas=${schemas}&package=${packageName}`, { responseType: 'text' } as any);
  },

  generateSwift: (schemas = 'public'): Promise<string> => {
    return api.get(`/api/pg/generators/swift?included_schemas=${schemas}`, { responseType: 'text' } as any);
  },

  generatePython: (schemas = 'public'): Promise<string> => {
    return api.get(`/api/pg/generators/python?included_schemas=${schemas}`, { responseType: 'text' } as any);
  },

  // Foreign Tables
  listForeignTables: (schemas = 'public'): Promise<any[]> => {
    return api.get(`/api/pg/foreign-tables?included_schemas=${schemas}`);
  },
};
