import { z } from 'zod';

// ============================================================================
// User Types
// ============================================================================

export const UserSchema = z.object({
  id: z.string(),
  email: z.string().email(),
  name: z.string(),
  avatar_url: z.string().nullable().optional(),
  password_hash: z.string(),
  settings: z.record(z.unknown()).optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type User = z.infer<typeof UserSchema>;

export const CreateUserSchema = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(100),
  password: z.string().min(6).max(100),
});

export type CreateUserInput = z.infer<typeof CreateUserSchema>;

export const LoginSchema = z.object({
  email: z.string().email(),
  password: z.string(),
});

export type LoginInput = z.infer<typeof LoginSchema>;

export interface UserPublic {
  id: string;
  email: string;
  name: string;
  avatar_url?: string | null;
  created_at: string;
}

// ============================================================================
// Session Types
// ============================================================================

export interface Session {
  id: string;
  user_id: string;
  expires_at: string;
  created_at: string;
}

export interface CreateSessionInput {
  id: string;
  user_id: string;
  expires_at: string;
}

// ============================================================================
// Workspace Types
// ============================================================================

export const WorkspaceSchema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  icon: z.string().nullable().optional(),
  plan: z.string().optional(),
  settings: z.record(z.unknown()).optional(),
  owner_id: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Workspace = z.infer<typeof WorkspaceSchema>;

export const CreateWorkspaceSchema = z.object({
  name: z.string().min(1).max(100),
  slug: z.string().min(1).max(50).regex(/^[a-z0-9-]+$/),
  icon: z.string().nullable().optional(),
});

export type CreateWorkspaceInput = z.infer<typeof CreateWorkspaceSchema>;

export const UpdateWorkspaceSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  icon: z.string().nullable().optional(),
  settings: z.record(z.unknown()).optional(),
});

export type UpdateWorkspaceInput = z.infer<typeof UpdateWorkspaceSchema>;

// ============================================================================
// Base Types
// ============================================================================

export const BaseSchema = z.object({
  id: z.string(),
  workspace_id: z.string(),
  name: z.string(),
  description: z.string().nullable().optional(),
  icon: z.string().nullable().optional(),
  color: z.string().optional(),
  settings: z.record(z.unknown()).optional(),
  is_template: z.boolean().optional(),
  created_by: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Base = z.infer<typeof BaseSchema>;

export const CreateBaseSchema = z.object({
  name: z.string().min(1).max(100),
  description: z.string().max(500).nullable().optional(),
  icon: z.string().nullable().optional(),
  color: z.string().optional(),
});

export type CreateBaseInput = z.infer<typeof CreateBaseSchema>;

export const UpdateBaseSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  description: z.string().max(500).nullable().optional(),
  icon: z.string().nullable().optional(),
  color: z.string().optional(),
  settings: z.record(z.unknown()).optional(),
});

export type UpdateBaseInput = z.infer<typeof UpdateBaseSchema>;

// ============================================================================
// Table Types
// ============================================================================

export const TableSchema = z.object({
  id: z.string(),
  base_id: z.string(),
  name: z.string(),
  description: z.string().nullable().optional(),
  icon: z.string().nullable().optional(),
  position: z.number(),
  primary_field_id: z.string().nullable().optional(),
  settings: z.record(z.unknown()).optional(),
  created_by: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Table = z.infer<typeof TableSchema>;

export const CreateTableSchema = z.object({
  name: z.string().min(1).max(100),
  description: z.string().max(500).nullable().optional(),
  icon: z.string().nullable().optional(),
});

export type CreateTableInput = z.infer<typeof CreateTableSchema>;

export const UpdateTableSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  description: z.string().max(500).nullable().optional(),
  icon: z.string().nullable().optional(),
  settings: z.record(z.unknown()).optional(),
});

export type UpdateTableInput = z.infer<typeof UpdateTableSchema>;

// ============================================================================
// Field Types
// ============================================================================

export const FieldTypeEnum = z.enum([
  'text', 'long_text', 'number', 'currency', 'percent',
  'single_select', 'multi_select', 'date', 'checkbox', 'rating',
  'duration', 'phone', 'email', 'url', 'attachment', 'user',
  'created_time', 'modified_time', 'created_by', 'modified_by',
  'autonumber', 'barcode', 'formula', 'rollup', 'count', 'lookup', 'link',
]);

export type FieldType = z.infer<typeof FieldTypeEnum>;

export const FieldSchema = z.object({
  id: z.string(),
  table_id: z.string(),
  name: z.string(),
  type: FieldTypeEnum,
  description: z.string().nullable().optional(),
  options: z.record(z.unknown()).optional(),
  position: z.number(),
  is_primary: z.boolean().optional(),
  is_computed: z.boolean().optional(),
  is_hidden: z.boolean().optional(),
  width: z.number().optional(),
  created_by: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Field = z.infer<typeof FieldSchema>;

export const CreateFieldSchema = z.object({
  name: z.string().min(1).max(100),
  type: FieldTypeEnum,
  description: z.string().max(500).nullable().optional(),
  options: z.record(z.unknown()).optional(),
});

export type CreateFieldInput = z.infer<typeof CreateFieldSchema>;

export const UpdateFieldSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  description: z.string().max(500).nullable().optional(),
  options: z.record(z.unknown()).optional(),
  is_hidden: z.boolean().optional(),
  width: z.number().optional(),
});

export type UpdateFieldInput = z.infer<typeof UpdateFieldSchema>;

// ============================================================================
// Select Option Types
// ============================================================================

export const SelectOptionSchema = z.object({
  id: z.string(),
  field_id: z.string(),
  name: z.string(),
  color: z.string(),
  position: z.number(),
});

export type SelectOption = z.infer<typeof SelectOptionSchema>;

export const CreateSelectOptionSchema = z.object({
  name: z.string().min(1).max(100),
  color: z.string().optional(),
});

export type CreateSelectOptionInput = z.infer<typeof CreateSelectOptionSchema>;

// ============================================================================
// Record Types
// ============================================================================

export const RecordSchema = z.object({
  id: z.string(),
  table_id: z.string(),
  position: z.number(),
  created_by: z.string(),
  created_at: z.string(),
  updated_by: z.string(),
  updated_at: z.string(),
});

export type Record = z.infer<typeof RecordSchema>;

export const CreateRecordSchema = z.object({
  fields: z.record(z.unknown()).optional(),
});

export type CreateRecordInput = z.infer<typeof CreateRecordSchema>;

export const UpdateRecordSchema = z.object({
  fields: z.record(z.unknown()),
});

export type UpdateRecordInput = z.infer<typeof UpdateRecordSchema>;

export const BatchCreateRecordsSchema = z.object({
  records: z.array(z.object({
    fields: z.record(z.unknown()).optional(),
  })),
});

export type BatchCreateRecordsInput = z.infer<typeof BatchCreateRecordsSchema>;

export const BatchUpdateRecordsSchema = z.object({
  records: z.array(z.object({
    id: z.string(),
    fields: z.record(z.unknown()),
  })),
});

export type BatchUpdateRecordsInput = z.infer<typeof BatchUpdateRecordsSchema>;

export const BatchDeleteRecordsSchema = z.object({
  ids: z.array(z.string()),
});

export type BatchDeleteRecordsInput = z.infer<typeof BatchDeleteRecordsSchema>;

// ============================================================================
// Cell Value Types
// ============================================================================

export const CellValueSchema = z.object({
  id: z.string(),
  record_id: z.string(),
  field_id: z.string(),
  value: z.unknown(),
  text_value: z.string().nullable().optional(),
  number_value: z.number().nullable().optional(),
  date_value: z.string().nullable().optional(),
  updated_at: z.string(),
});

export type CellValue = z.infer<typeof CellValueSchema>;

// ============================================================================
// View Types
// ============================================================================

export const ViewTypeEnum = z.enum([
  'grid', 'kanban', 'calendar', 'gallery', 'form', 'timeline', 'list',
]);

export type ViewType = z.infer<typeof ViewTypeEnum>;

export const ViewSchema = z.object({
  id: z.string(),
  table_id: z.string(),
  name: z.string(),
  type: ViewTypeEnum,
  filters: z.array(z.unknown()).optional(),
  sorts: z.array(z.unknown()).optional(),
  groups: z.array(z.unknown()).optional(),
  field_config: z.array(z.unknown()).optional(),
  settings: z.record(z.unknown()).optional(),
  position: z.number(),
  is_default: z.boolean().optional(),
  is_locked: z.boolean().optional(),
  created_by: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type View = z.infer<typeof ViewSchema>;

export const CreateViewSchema = z.object({
  name: z.string().min(1).max(100),
  type: ViewTypeEnum,
  filters: z.array(z.unknown()).optional(),
  sorts: z.array(z.unknown()).optional(),
  groups: z.array(z.unknown()).optional(),
  field_config: z.array(z.unknown()).optional(),
  settings: z.record(z.unknown()).optional(),
});

export type CreateViewInput = z.infer<typeof CreateViewSchema>;

export const UpdateViewSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  filters: z.array(z.unknown()).optional(),
  sorts: z.array(z.unknown()).optional(),
  groups: z.array(z.unknown()).optional(),
  field_config: z.array(z.unknown()).optional(),
  settings: z.record(z.unknown()).optional(),
  is_locked: z.boolean().optional(),
});

export type UpdateViewInput = z.infer<typeof UpdateViewSchema>;

// ============================================================================
// Comment Types
// ============================================================================

export const CommentSchema = z.object({
  id: z.string(),
  record_id: z.string(),
  parent_id: z.string().nullable().optional(),
  author_id: z.string(),
  content: z.record(z.unknown()),
  is_resolved: z.boolean().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Comment = z.infer<typeof CommentSchema>;

export const CreateCommentSchema = z.object({
  content: z.record(z.unknown()),
  parent_id: z.string().nullable().optional(),
});

export type CreateCommentInput = z.infer<typeof CreateCommentSchema>;

export const UpdateCommentSchema = z.object({
  content: z.record(z.unknown()).optional(),
  is_resolved: z.boolean().optional(),
});

export type UpdateCommentInput = z.infer<typeof UpdateCommentSchema>;

// ============================================================================
// Share Types
// ============================================================================

export const ShareSchema = z.object({
  id: z.string(),
  base_id: z.string(),
  type: z.string(),
  permission: z.string(),
  user_id: z.string().nullable().optional(),
  email: z.string().nullable().optional(),
  token: z.string().nullable().optional(),
  password: z.string().nullable().optional(),
  expires_at: z.string().nullable().optional(),
  created_by: z.string(),
  created_at: z.string(),
});

export type Share = z.infer<typeof ShareSchema>;

export const CreateShareSchema = z.object({
  type: z.enum(['invite', 'public_link']),
  permission: z.enum(['read', 'comment', 'edit', 'creator', 'owner']),
  email: z.string().email().optional(),
  password: z.string().optional(),
  expires_at: z.string().optional(),
});

export type CreateShareInput = z.infer<typeof CreateShareSchema>;

// ============================================================================
// Pagination Types
// ============================================================================

export const PaginationSchema = z.object({
  cursor: z.string().optional(),
  limit: z.number().int().min(1).max(100).optional().default(50),
});

export type PaginationInput = z.infer<typeof PaginationSchema>;

export interface PaginatedResult<T> {
  items: T[];
  next_cursor?: string;
  has_more: boolean;
}

// ============================================================================
// API Response Types
// ============================================================================

export interface ApiError {
  error: string;
  message: string;
  status: number;
}

export interface AuthResponse {
  token: string;
  user: UserPublic;
}

// ============================================================================
// Environment Types
// ============================================================================

export interface Env {
  // SQLite file path
  DATABASE_PATH?: string;
  // PostgreSQL connection string
  DATABASE_URL?: string;
  // JWT secret
  JWT_SECRET: string;
  // Node environment
  NODE_ENV?: string;
}

// ============================================================================
// Context Variables
// ============================================================================

export interface Variables {
  user: UserPublic;
  session: Session;
}
