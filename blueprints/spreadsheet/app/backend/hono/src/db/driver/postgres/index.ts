/**
 * PostgreSQL database driver
 *
 * Uses the postgres library for connections
 */

import postgres from 'postgres';
import type { Database } from '../../types.js';
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
} from '../../types.js';
import { PostgresUsersStore } from './users.js';
import { PostgresSessionsStore } from './sessions.js';
import { PostgresWorkbooksStore } from './workbooks.js';
import { PostgresSheetsStore } from './sheets.js';
import { PostgresTilesStore } from './tiles.js';
import { PostgresChartsStore } from './charts.js';

// Schema for migrations
const schema = `
-- PostgreSQL Schema for Spreadsheet (Tile-based storage)

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP DEFAULT NOW(),
    updated_at    TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS workbooks (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workbooks_user ON workbooks(user_id);

CREATE TABLE IF NOT EXISTS sheets (
    id          TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL REFERENCES workbooks(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    index_num   INTEGER NOT NULL DEFAULT 0,
    row_count   INTEGER DEFAULT 1000,
    col_count   INTEGER DEFAULT 26,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);

CREATE TABLE IF NOT EXISTS sheet_tiles (
    sheet_id    TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    tile_row    INTEGER NOT NULL,
    tile_col    INTEGER NOT NULL,
    tile_h      INTEGER NOT NULL DEFAULT 256,
    tile_w      INTEGER NOT NULL DEFAULT 64,
    encoding    TEXT NOT NULL DEFAULT 'json_v1',
    values_blob TEXT NOT NULL,
    updated_at  TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (sheet_id, tile_row, tile_col)
);

CREATE INDEX IF NOT EXISTS idx_sheet_tiles_lookup
ON sheet_tiles(sheet_id, tile_row, tile_col);

CREATE TABLE IF NOT EXISTS merged_regions (
    id        TEXT PRIMARY KEY,
    sheet_id  TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    start_row INTEGER NOT NULL,
    start_col INTEGER NOT NULL,
    end_row   INTEGER NOT NULL,
    end_col   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_merged_sheet ON merged_regions(sheet_id);

CREATE TABLE IF NOT EXISTS charts (
    id         TEXT PRIMARY KEY,
    sheet_id   TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT '',
    chart_type TEXT NOT NULL DEFAULT 'bar',
    data_range TEXT NOT NULL,
    config     TEXT,
    position_x INTEGER DEFAULT 0,
    position_y INTEGER DEFAULT 0,
    width      INTEGER DEFAULT 400,
    height     INTEGER DEFAULT 300,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);
`;

/**
 * PostgreSQL database driver implementing the Database interface
 */
export class PostgresDriver implements Database {
  private sql: postgres.Sql;
  private users: PostgresUsersStore;
  private sessions: PostgresSessionsStore;
  private workbooks: PostgresWorkbooksStore;
  private sheets: PostgresSheetsStore;
  private tiles: PostgresTilesStore;
  private charts: PostgresChartsStore;

  constructor(connectionString: string) {
    this.sql = postgres(connectionString, {
      max: 10,
      idle_timeout: 20,
      connect_timeout: 10,
    });
    this.users = new PostgresUsersStore(this.sql);
    this.sessions = new PostgresSessionsStore(this.sql);
    this.workbooks = new PostgresWorkbooksStore(this.sql);
    this.sheets = new PostgresSheetsStore(this.sql);
    this.tiles = new PostgresTilesStore(this.sql);
    this.charts = new PostgresChartsStore(this.sql);
  }

  /**
   * Run schema migrations
   */
  async ensure(): Promise<void> {
    await this.sql.unsafe(schema);
  }

  // ============================================================================
  // Users
  // ============================================================================

  createUser(input: CreateUserInput & { id: string; password_hash: string }): Promise<User> {
    return this.users.createUser(input);
  }

  getUserById(id: string): Promise<User | null> {
    return this.users.getUserById(id);
  }

  getUserByEmail(email: string): Promise<User | null> {
    return this.users.getUserByEmail(email);
  }

  // ============================================================================
  // Sessions
  // ============================================================================

  createSession(input: CreateSessionInput): Promise<Session> {
    return this.sessions.createSession(input);
  }

  getSessionByToken(token: string): Promise<Session | null> {
    return this.sessions.getSessionByToken(token);
  }

  deleteSession(token: string): Promise<void> {
    return this.sessions.deleteSession(token);
  }

  deleteExpiredSessions(): Promise<void> {
    return this.sessions.deleteExpiredSessions();
  }

  // ============================================================================
  // Workbooks
  // ============================================================================

  createWorkbook(input: CreateWorkbookInput & { id: string; user_id: string }): Promise<Workbook> {
    return this.workbooks.createWorkbook(input);
  }

  getWorkbook(id: string): Promise<Workbook | null> {
    return this.workbooks.getWorkbook(id);
  }

  getWorkbooksByUser(userId: string): Promise<Workbook[]> {
    return this.workbooks.getWorkbooksByUser(userId);
  }

  updateWorkbook(id: string, data: UpdateWorkbookInput): Promise<Workbook | null> {
    return this.workbooks.updateWorkbook(id, data);
  }

  deleteWorkbook(id: string): Promise<void> {
    return this.workbooks.deleteWorkbook(id);
  }

  // ============================================================================
  // Sheets
  // ============================================================================

  createSheet(input: CreateSheetInput & { id: string }): Promise<Sheet> {
    return this.sheets.createSheet(input);
  }

  getSheet(id: string): Promise<Sheet | null> {
    return this.sheets.getSheet(id);
  }

  getSheetsByWorkbook(workbookId: string): Promise<Sheet[]> {
    return this.sheets.getSheetsByWorkbook(workbookId);
  }

  updateSheet(id: string, data: UpdateSheetInput): Promise<Sheet | null> {
    return this.sheets.updateSheet(id, data);
  }

  deleteSheet(id: string): Promise<void> {
    return this.sheets.deleteSheet(id);
  }

  getMaxSheetIndex(workbookId: string): Promise<number> {
    return this.sheets.getMaxSheetIndex(workbookId);
  }

  // ============================================================================
  // Cells (via tiles)
  // ============================================================================

  getCell(sheetId: string, row: number, col: number): Promise<Cell | null> {
    return this.tiles.getCell(sheetId, row, col);
  }

  getCellsBySheet(sheetId: string): Promise<Cell[]> {
    return this.tiles.getCellsBySheet(sheetId);
  }

  upsertCell(input: UpsertCellInput & { id: string }): Promise<Cell> {
    return this.tiles.upsertCell(input);
  }

  upsertCells(inputs: Array<UpsertCellInput & { id: string }>): Promise<Cell[]> {
    return this.tiles.upsertCells(inputs);
  }

  deleteCell(sheetId: string, row: number, col: number): Promise<void> {
    return this.tiles.deleteCell(sheetId, row, col);
  }

  deleteCellsInRange(
    sheetId: string,
    startRow: number,
    endRow: number,
    startCol: number,
    endCol: number
  ): Promise<void> {
    return this.tiles.deleteCellsInRange(sheetId, startRow, endRow, startCol, endCol);
  }

  shiftCellsDown(sheetId: string, startRow: number, count: number): Promise<void> {
    return this.tiles.shiftCellsDown(sheetId, startRow, count);
  }

  shiftCellsUp(sheetId: string, startRow: number, count: number): Promise<void> {
    return this.tiles.shiftCellsUp(sheetId, startRow, count);
  }

  shiftCellsRight(sheetId: string, startCol: number, count: number): Promise<void> {
    return this.tiles.shiftCellsRight(sheetId, startCol, count);
  }

  shiftCellsLeft(sheetId: string, startCol: number, count: number): Promise<void> {
    return this.tiles.shiftCellsLeft(sheetId, startCol, count);
  }

  // ============================================================================
  // Merged Regions
  // ============================================================================

  getMergedRegions(sheetId: string): Promise<MergedRegion[]> {
    return this.tiles.getMergedRegions(sheetId);
  }

  createMergedRegion(input: CreateMergeInput & { id: string; sheet_id: string }): Promise<MergedRegion> {
    return this.tiles.createMergedRegion(input);
  }

  deleteMergedRegion(id: string): Promise<void> {
    return this.tiles.deleteMergedRegion(id);
  }

  deleteMergedRegionsInRange(
    sheetId: string,
    startRow: number,
    endRow: number,
    startCol: number,
    endCol: number
  ): Promise<void> {
    return this.tiles.deleteMergedRegionsInRange(sheetId, startRow, endRow, startCol, endCol);
  }

  // ============================================================================
  // Charts
  // ============================================================================

  createChart(input: CreateChartInput & { id: string }): Promise<Chart> {
    return this.charts.createChart(input);
  }

  getChart(id: string): Promise<Chart | null> {
    return this.charts.getChart(id);
  }

  getChartsBySheet(sheetId: string): Promise<Chart[]> {
    return this.charts.getChartsBySheet(sheetId);
  }

  updateChart(id: string, data: UpdateChartInput): Promise<Chart | null> {
    return this.charts.updateChart(id, data);
  }

  deleteChart(id: string): Promise<void> {
    return this.charts.deleteChart(id);
  }

  // ============================================================================
  // Utilities
  // ============================================================================

  async close(): Promise<void> {
    await this.sql.end();
  }
}
