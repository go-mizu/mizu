/**
 * SQLite database driver
 *
 * Supports D1 (Cloudflare Workers), better-sqlite3 (Node.js), and bun:sqlite
 */

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
import { SqliteExecutor, D1Adapter, createBetterSqliteAdapter, type BetterSqlite3Database } from './executor.js';
import { SqliteUsersStore } from './users.js';
import { SqliteSessionsStore } from './sessions.js';
import { SqliteWorkbooksStore } from './workbooks.js';
import { SqliteSheetsStore } from './sheets.js';
import { SqliteTilesStore } from './tiles.js';
import { SqliteChartsStore } from './charts.js';

// Embed schema for migrations
const schema = `
-- SQLite Schema for Spreadsheet (Tile-based storage)

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TEXT DEFAULT (datetime('now')),
    updated_at    TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    token      TEXT UNIQUE NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS workbooks (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    name       TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workbooks_user ON workbooks(user_id);

CREATE TABLE IF NOT EXISTS sheets (
    id          TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL,
    name        TEXT NOT NULL,
    index_num   INTEGER NOT NULL DEFAULT 0,
    row_count   INTEGER DEFAULT 1000,
    col_count   INTEGER DEFAULT 26,
    created_at  TEXT DEFAULT (datetime('now')),
    updated_at  TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);

CREATE TABLE IF NOT EXISTS sheet_tiles (
    sheet_id    TEXT NOT NULL,
    tile_row    INTEGER NOT NULL,
    tile_col    INTEGER NOT NULL,
    tile_h      INTEGER NOT NULL DEFAULT 256,
    tile_w      INTEGER NOT NULL DEFAULT 64,
    encoding    TEXT NOT NULL DEFAULT 'json_v1',
    values_blob TEXT NOT NULL,
    updated_at  TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (sheet_id, tile_row, tile_col),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sheet_tiles_lookup
ON sheet_tiles(sheet_id, tile_row, tile_col);

CREATE TABLE IF NOT EXISTS merged_regions (
    id        TEXT PRIMARY KEY,
    sheet_id  TEXT NOT NULL,
    start_row INTEGER NOT NULL,
    start_col INTEGER NOT NULL,
    end_row   INTEGER NOT NULL,
    end_col   INTEGER NOT NULL,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_merged_sheet ON merged_regions(sheet_id);

CREATE TABLE IF NOT EXISTS charts (
    id         TEXT PRIMARY KEY,
    sheet_id   TEXT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    chart_type TEXT NOT NULL DEFAULT 'bar',
    data_range TEXT NOT NULL,
    config     TEXT,
    position_x INTEGER DEFAULT 0,
    position_y INTEGER DEFAULT 0,
    width      INTEGER DEFAULT 400,
    height     INTEGER DEFAULT 300,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);
`;

/**
 * SQLite database driver implementing the Database interface
 */
export class SqliteDriver implements Database {
  private executor: SqliteExecutor;
  private users: SqliteUsersStore;
  private sessions: SqliteSessionsStore;
  private workbooks: SqliteWorkbooksStore;
  private sheets: SqliteSheetsStore;
  private tiles: SqliteTilesStore;
  private charts: SqliteChartsStore;

  constructor(executor: SqliteExecutor) {
    this.executor = executor;
    this.users = new SqliteUsersStore(executor);
    this.sessions = new SqliteSessionsStore(executor);
    this.workbooks = new SqliteWorkbooksStore(executor);
    this.sheets = new SqliteSheetsStore(executor);
    this.tiles = new SqliteTilesStore(executor);
    this.charts = new SqliteChartsStore(executor);
  }

  /**
   * Create a SQLite driver from a D1 database
   */
  static fromD1(db: D1Database): SqliteDriver {
    return new SqliteDriver(new D1Adapter(db));
  }

  /**
   * Create a SQLite driver from a better-sqlite3 database
   */
  static fromBetterSqlite(db: BetterSqlite3Database): SqliteDriver {
    return new SqliteDriver(createBetterSqliteAdapter(db));
  }

  /**
   * Create a SQLite driver with an in-memory database (for testing)
   */
  static async createInMemory(): Promise<SqliteDriver> {
    const BetterSqlite3 = await import('better-sqlite3');
    const db = new BetterSqlite3.default(':memory:');
    const driver = SqliteDriver.fromBetterSqlite(db);
    await driver.ensure();
    return driver;
  }

  /**
   * Run schema migrations
   */
  async ensure(): Promise<void> {
    for (const statement of schema.split(';').filter(s => s.trim())) {
      await this.executor.run(statement);
    }
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
    // SQLite/D1 doesn't need explicit close
  }
}

// Re-exports
export { D1Adapter, createBetterSqliteAdapter, createInMemoryExecutor, createFileExecutor } from './executor.js';
export type { SqliteExecutor, BetterSqlite3Database } from './executor.js';
