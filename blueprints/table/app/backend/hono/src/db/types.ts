/**
 * Database types and interfaces for the table blueprint
 */

// Re-export entity types
export type {
  User,
  CreateUserInput,
  Session,
  CreateSessionInput,
  Workspace,
  CreateWorkspaceInput,
  UpdateWorkspaceInput,
  Base,
  CreateBaseInput,
  UpdateBaseInput,
  Table,
  CreateTableInput,
  UpdateTableInput,
  Field,
  CreateFieldInput,
  UpdateFieldInput,
  SelectOption,
  CreateSelectOptionInput,
  Record,
  CreateRecordInput,
  UpdateRecordInput,
  CellValue,
  View,
  CreateViewInput,
  UpdateViewInput,
  Comment,
  CreateCommentInput,
  UpdateCommentInput,
  Share,
  CreateShareInput,
  PaginatedResult,
} from '../types/index.js';

import type {
  User,
  CreateUserInput,
  Session,
  CreateSessionInput,
  Workspace,
  CreateWorkspaceInput,
  UpdateWorkspaceInput,
  Base,
  CreateBaseInput,
  UpdateBaseInput,
  Table,
  CreateTableInput,
  UpdateTableInput,
  Field,
  CreateFieldInput,
  UpdateFieldInput,
  SelectOption,
  CreateSelectOptionInput,
  Record,
  CellValue,
  View,
  CreateViewInput,
  UpdateViewInput,
  Comment,
  CreateCommentInput,
  UpdateCommentInput,
  Share,
  CreateShareInput,
} from '../types/index.js';

/**
 * Record with cell values populated
 */
export interface RecordWithFields extends Record {
  fields: { [fieldId: string]: unknown };
}

/**
 * Filter condition for querying records
 */
export interface FilterCondition {
  field_id: string;
  operator: string;
  value: unknown;
}

/**
 * Sort condition for querying records
 */
export interface SortCondition {
  field_id: string;
  direction: 'asc' | 'desc';
}

/**
 * Query options for listing records
 */
export interface RecordQueryOptions {
  filters?: FilterCondition[];
  sorts?: SortCondition[];
  cursor?: string;
  limit?: number;
}

/**
 * Database interface - abstracts SQLite and PostgreSQL
 */
export interface Database {
  // ============================================================================
  // Users
  // ============================================================================
  createUser(input: CreateUserInput & { id: string; password_hash: string }): Promise<User>;
  getUserById(id: string): Promise<User | null>;
  getUserByEmail(email: string): Promise<User | null>;

  // ============================================================================
  // Sessions
  // ============================================================================
  createSession(session: CreateSessionInput): Promise<Session>;
  getSessionById(id: string): Promise<Session | null>;
  deleteSession(id: string): Promise<void>;
  deleteExpiredSessions(): Promise<void>;

  // ============================================================================
  // Workspaces
  // ============================================================================
  createWorkspace(input: CreateWorkspaceInput & { id: string; owner_id: string }): Promise<Workspace>;
  getWorkspace(id: string): Promise<Workspace | null>;
  getWorkspaceBySlug(slug: string): Promise<Workspace | null>;
  getWorkspacesByUser(userId: string): Promise<Workspace[]>;
  updateWorkspace(id: string, data: UpdateWorkspaceInput): Promise<Workspace | null>;
  deleteWorkspace(id: string): Promise<void>;

  // ============================================================================
  // Bases
  // ============================================================================
  createBase(input: CreateBaseInput & { id: string; workspace_id: string; created_by: string }): Promise<Base>;
  getBase(id: string): Promise<Base | null>;
  getBasesByWorkspace(workspaceId: string): Promise<Base[]>;
  updateBase(id: string, data: UpdateBaseInput): Promise<Base | null>;
  deleteBase(id: string): Promise<void>;

  // ============================================================================
  // Tables
  // ============================================================================
  createTable(input: CreateTableInput & { id: string; base_id: string; created_by: string }): Promise<Table>;
  getTable(id: string): Promise<Table | null>;
  getTablesByBase(baseId: string): Promise<Table[]>;
  updateTable(id: string, data: UpdateTableInput): Promise<Table | null>;
  deleteTable(id: string): Promise<void>;
  reorderTables(baseId: string, tableIds: string[]): Promise<void>;

  // ============================================================================
  // Fields
  // ============================================================================
  createField(input: CreateFieldInput & { id: string; table_id: string; created_by: string; position: number }): Promise<Field>;
  getField(id: string): Promise<Field | null>;
  getFieldsByTable(tableId: string): Promise<Field[]>;
  updateField(id: string, data: UpdateFieldInput): Promise<Field | null>;
  deleteField(id: string): Promise<void>;
  reorderFields(tableId: string, fieldIds: string[]): Promise<void>;
  getMaxFieldPosition(tableId: string): Promise<number>;

  // ============================================================================
  // Select Options
  // ============================================================================
  createSelectOption(input: CreateSelectOptionInput & { id: string; field_id: string; position: number }): Promise<SelectOption>;
  getSelectOptionsByField(fieldId: string): Promise<SelectOption[]>;
  updateSelectOption(id: string, name: string, color: string): Promise<SelectOption | null>;
  deleteSelectOption(id: string): Promise<void>;
  reorderSelectOptions(fieldId: string, optionIds: string[]): Promise<void>;

  // ============================================================================
  // Records
  // ============================================================================
  createRecord(input: { id: string; table_id: string; created_by: string; position: number }): Promise<Record>;
  getRecord(id: string): Promise<Record | null>;
  getRecordWithFields(id: string): Promise<RecordWithFields | null>;
  getRecordsByTable(tableId: string, options?: RecordQueryOptions): Promise<{ records: RecordWithFields[]; next_cursor?: string; has_more: boolean }>;
  updateRecord(id: string, updated_by: string): Promise<Record | null>;
  deleteRecord(id: string): Promise<void>;
  deleteRecordsByTable(tableId: string): Promise<void>;
  getMaxRecordPosition(tableId: string): Promise<number>;
  getRecordCount(tableId: string): Promise<number>;

  // ============================================================================
  // Cell Values
  // ============================================================================
  setCellValue(recordId: string, fieldId: string, value: unknown): Promise<CellValue>;
  getCellValue(recordId: string, fieldId: string): Promise<CellValue | null>;
  getCellValuesByRecord(recordId: string): Promise<CellValue[]>;
  deleteCellValue(recordId: string, fieldId: string): Promise<void>;
  deleteCellValuesByRecord(recordId: string): Promise<void>;
  deleteCellValuesByField(fieldId: string): Promise<void>;

  // ============================================================================
  // Views
  // ============================================================================
  createView(input: CreateViewInput & { id: string; table_id: string; created_by: string; position: number }): Promise<View>;
  getView(id: string): Promise<View | null>;
  getViewsByTable(tableId: string): Promise<View[]>;
  updateView(id: string, data: UpdateViewInput): Promise<View | null>;
  deleteView(id: string): Promise<void>;
  reorderViews(tableId: string, viewIds: string[]): Promise<void>;
  getMaxViewPosition(tableId: string): Promise<number>;

  // ============================================================================
  // Comments
  // ============================================================================
  createComment(input: CreateCommentInput & { id: string; record_id: string; author_id: string }): Promise<Comment>;
  getComment(id: string): Promise<Comment | null>;
  getCommentsByRecord(recordId: string): Promise<Comment[]>;
  updateComment(id: string, data: UpdateCommentInput): Promise<Comment | null>;
  deleteComment(id: string): Promise<void>;

  // ============================================================================
  // Shares
  // ============================================================================
  createShare(input: CreateShareInput & { id: string; base_id: string; created_by: string; token?: string }): Promise<Share>;
  getShare(id: string): Promise<Share | null>;
  getShareByToken(token: string): Promise<Share | null>;
  getSharesByBase(baseId: string): Promise<Share[]>;
  deleteShare(id: string): Promise<void>;

  // ============================================================================
  // Utilities
  // ============================================================================
  close(): Promise<void>;
  ensure(): Promise<void>;
}

/**
 * Row result from database - used internally
 */
export interface DbRow {
  [key: string]: unknown;
}
