/**
 * Database types and interfaces
 *
 * Re-exports entity types from main types file and defines the Database interface
 */

// Re-export entity types
export type {
  User,
  CreateUserInput,
  Session,
  CreateSessionInput,
  Workbook,
  CreateWorkbookInput,
  UpdateWorkbookInput,
  Sheet,
  CreateSheetInput,
  UpdateSheetInput,
  Cell,
  UpsertCellInput,
  MergedRegion,
  CreateMergeInput,
  Chart,
  CreateChartInput,
  UpdateChartInput,
} from '../types/index.js';

import type {
  User,
  CreateUserInput,
  Session,
  CreateSessionInput,
  Workbook,
  CreateWorkbookInput,
  UpdateWorkbookInput,
  Sheet,
  CreateSheetInput,
  UpdateSheetInput,
  Cell,
  UpsertCellInput,
  MergedRegion,
  CreateMergeInput,
  Chart,
  CreateChartInput,
  UpdateChartInput,
} from '../types/index.js';

/**
 * Database interface - abstracts PostgreSQL and SQLite/D1
 *
 * Both drivers (SqliteDriver and PostgresDriver) implement this interface.
 * Cell storage uses tile-based storage internally for better performance.
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
  getSessionByToken(token: string): Promise<Session | null>;
  deleteSession(token: string): Promise<void>;
  deleteExpiredSessions(): Promise<void>;

  // ============================================================================
  // Workbooks
  // ============================================================================
  createWorkbook(input: CreateWorkbookInput & { id: string; user_id: string }): Promise<Workbook>;
  getWorkbook(id: string): Promise<Workbook | null>;
  getWorkbooksByUser(userId: string): Promise<Workbook[]>;
  updateWorkbook(id: string, data: UpdateWorkbookInput): Promise<Workbook | null>;
  deleteWorkbook(id: string): Promise<void>;

  // ============================================================================
  // Sheets
  // ============================================================================
  createSheet(input: CreateSheetInput & { id: string }): Promise<Sheet>;
  getSheet(id: string): Promise<Sheet | null>;
  getSheetsByWorkbook(workbookId: string): Promise<Sheet[]>;
  updateSheet(id: string, data: UpdateSheetInput): Promise<Sheet | null>;
  deleteSheet(id: string): Promise<void>;
  getMaxSheetIndex(workbookId: string): Promise<number>;

  // ============================================================================
  // Cells (tile-based storage internally)
  // ============================================================================
  getCell(sheetId: string, row: number, col: number): Promise<Cell | null>;
  getCellsBySheet(sheetId: string): Promise<Cell[]>;
  upsertCell(input: UpsertCellInput & { id: string }): Promise<Cell>;
  upsertCells(inputs: Array<UpsertCellInput & { id: string }>): Promise<Cell[]>;
  deleteCell(sheetId: string, row: number, col: number): Promise<void>;
  deleteCellsInRange(sheetId: string, startRow: number, endRow: number, startCol: number, endCol: number): Promise<void>;
  shiftCellsDown(sheetId: string, startRow: number, count: number): Promise<void>;
  shiftCellsUp(sheetId: string, startRow: number, count: number): Promise<void>;
  shiftCellsRight(sheetId: string, startCol: number, count: number): Promise<void>;
  shiftCellsLeft(sheetId: string, startCol: number, count: number): Promise<void>;

  // ============================================================================
  // Merged Regions
  // ============================================================================
  getMergedRegions(sheetId: string): Promise<MergedRegion[]>;
  createMergedRegion(input: CreateMergeInput & { id: string; sheet_id: string }): Promise<MergedRegion>;
  deleteMergedRegion(id: string): Promise<void>;
  deleteMergedRegionsInRange(sheetId: string, startRow: number, endRow: number, startCol: number, endCol: number): Promise<void>;

  // ============================================================================
  // Charts
  // ============================================================================
  createChart(input: CreateChartInput & { id: string }): Promise<Chart>;
  getChart(id: string): Promise<Chart | null>;
  getChartsBySheet(sheetId: string): Promise<Chart[]>;
  updateChart(id: string, data: UpdateChartInput): Promise<Chart | null>;
  deleteChart(id: string): Promise<void>;

  // ============================================================================
  // Utilities
  // ============================================================================
  close(): Promise<void>;
}

/**
 * Row result from database - used internally
 */
export interface DbRow {
  [key: string]: unknown;
}
