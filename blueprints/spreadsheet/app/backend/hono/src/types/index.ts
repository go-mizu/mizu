import { z } from 'zod';

// ============================================================================
// User Types
// ============================================================================

export const UserSchema = z.object({
  id: z.string(),
  email: z.string().email(),
  name: z.string(),
  password_hash: z.string(),
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
  created_at: string;
}

// ============================================================================
// Session Types
// ============================================================================

export interface Session {
  id: string;
  user_id: string;
  token: string;
  expires_at: string;
  created_at: string;
}

export interface CreateSessionInput {
  id: string;
  user_id: string;
  token: string;
  expires_at: string;
}

// ============================================================================
// Workbook Types
// ============================================================================

export const WorkbookSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  name: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Workbook = z.infer<typeof WorkbookSchema>;

export const CreateWorkbookSchema = z.object({
  name: z.string().min(1).max(255),
});

export type CreateWorkbookInput = z.infer<typeof CreateWorkbookSchema>;

export const UpdateWorkbookSchema = z.object({
  name: z.string().min(1).max(255).optional(),
});

export type UpdateWorkbookInput = z.infer<typeof UpdateWorkbookSchema>;

// ============================================================================
// Sheet Types
// ============================================================================

export const SheetSchema = z.object({
  id: z.string(),
  workbook_id: z.string(),
  name: z.string(),
  index_num: z.number(),
  row_count: z.number(),
  col_count: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Sheet = z.infer<typeof SheetSchema>;

export const CreateSheetSchema = z.object({
  workbook_id: z.string(),
  name: z.string().min(1).max(100),
  index_num: z.number().int().min(0).optional(),
});

export type CreateSheetInput = z.infer<typeof CreateSheetSchema>;

export const UpdateSheetSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  index_num: z.number().int().min(0).optional(),
});

export type UpdateSheetInput = z.infer<typeof UpdateSheetSchema>;

// ============================================================================
// Cell Types
// ============================================================================

export const CellSchema = z.object({
  id: z.string(),
  sheet_id: z.string(),
  row_num: z.number().int().min(0),
  col_num: z.number().int().min(0),
  value: z.string().nullable(),
  formula: z.string().nullable(),
  display: z.string().nullable(),
  format: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Cell = z.infer<typeof CellSchema>;

export const UpsertCellSchema = z.object({
  sheet_id: z.string(),
  row_num: z.number().int().min(0),
  col_num: z.number().int().min(0),
  value: z.string().nullable().optional(),
  formula: z.string().nullable().optional(),
  display: z.string().nullable().optional(),
  format: z.string().nullable().optional(),
});

export type UpsertCellInput = z.infer<typeof UpsertCellSchema>;

export const BatchCellUpdateSchema = z.object({
  cells: z.array(z.object({
    row: z.number().int().min(0),
    col: z.number().int().min(0),
    value: z.string().nullable().optional(),
    formula: z.string().nullable().optional(),
    display: z.string().nullable().optional(),
    format: z.string().nullable().optional(),
  })),
});

export type BatchCellUpdate = z.infer<typeof BatchCellUpdateSchema>;

// ============================================================================
// Merged Region Types
// ============================================================================

export const MergedRegionSchema = z.object({
  id: z.string(),
  sheet_id: z.string(),
  start_row: z.number().int().min(0),
  start_col: z.number().int().min(0),
  end_row: z.number().int().min(0),
  end_col: z.number().int().min(0),
});

export type MergedRegion = z.infer<typeof MergedRegionSchema>;

export const CreateMergeSchema = z.object({
  start_row: z.number().int().min(0),
  start_col: z.number().int().min(0),
  end_row: z.number().int().min(0),
  end_col: z.number().int().min(0),
});

export type CreateMergeInput = z.infer<typeof CreateMergeSchema>;

// ============================================================================
// Chart Types
// ============================================================================

export const ChartTypeEnum = z.enum([
  'bar', 'line', 'pie', 'doughnut', 'area', 'scatter',
  'radar', 'polarArea', 'bubble', 'horizontalBar',
  'stackedBar', 'stackedArea',
]);

export const ChartSchema = z.object({
  id: z.string(),
  sheet_id: z.string(),
  title: z.string(),
  chart_type: ChartTypeEnum,
  data_range: z.string(),
  config: z.string().nullable(),
  position_x: z.number(),
  position_y: z.number(),
  width: z.number(),
  height: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type Chart = z.infer<typeof ChartSchema>;

export const CreateChartSchema = z.object({
  sheet_id: z.string(),
  title: z.string().max(255).optional().default(''),
  chart_type: ChartTypeEnum.optional().default('bar'),
  data_range: z.string(),
  config: z.string().nullable().optional(),
  position_x: z.number().int().optional().default(0),
  position_y: z.number().int().optional().default(0),
  width: z.number().int().optional().default(400),
  height: z.number().int().optional().default(300),
});

export type CreateChartInput = z.infer<typeof CreateChartSchema>;

export const UpdateChartSchema = z.object({
  title: z.string().max(255).optional(),
  chart_type: ChartTypeEnum.optional(),
  data_range: z.string().optional(),
  config: z.string().nullable().optional(),
  position_x: z.number().int().optional(),
  position_y: z.number().int().optional(),
  width: z.number().int().optional(),
  height: z.number().int().optional(),
});

export type UpdateChartInput = z.infer<typeof UpdateChartSchema>;

// ============================================================================
// Row/Column Operations
// ============================================================================

export const InsertRowsSchema = z.object({
  start_row: z.number().int().min(0),
  count: z.number().int().min(1).max(100).default(1),
});

export type InsertRowsInput = z.infer<typeof InsertRowsSchema>;

export const DeleteRowsSchema = z.object({
  start_row: z.number().int().min(0),
  count: z.number().int().min(1).max(100).default(1),
});

export type DeleteRowsInput = z.infer<typeof DeleteRowsSchema>;

export const InsertColsSchema = z.object({
  start_col: z.number().int().min(0),
  count: z.number().int().min(1).max(100).default(1),
});

export type InsertColsInput = z.infer<typeof InsertColsSchema>;

export const DeleteColsSchema = z.object({
  start_col: z.number().int().min(0),
  count: z.number().int().min(1).max(100).default(1),
});

export type DeleteColsInput = z.infer<typeof DeleteColsSchema>;

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
  // Cloudflare D1 binding
  DB?: D1Database;
  // PostgreSQL connection string (Vercel/Node)
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
