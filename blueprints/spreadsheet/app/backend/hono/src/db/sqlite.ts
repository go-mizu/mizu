import type { Database } from './types.js';
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
 * Generic SQLite executor interface (works with D1 and better-sqlite3)
 */
interface SqliteExecutor {
  run(sql: string, params?: unknown[]): Promise<void>;
  get<T>(sql: string, params?: unknown[]): Promise<T | null>;
  all<T>(sql: string, params?: unknown[]): Promise<T[]>;
}

/**
 * D1 adapter - wraps Cloudflare D1 database
 */
export class D1Adapter implements SqliteExecutor {
  constructor(private db: D1Database) {}

  async run(sql: string, params: unknown[] = []): Promise<void> {
    await this.db.prepare(sql).bind(...params).run();
  }

  async get<T>(sql: string, params: unknown[] = []): Promise<T | null> {
    const result = await this.db.prepare(sql).bind(...params).first();
    return result as T | null;
  }

  async all<T>(sql: string, params: unknown[] = []): Promise<T[]> {
    const result = await this.db.prepare(sql).bind(...params).all();
    return result.results as T[];
  }
}

/**
 * SQLite database implementation
 */
export class SqliteDatabase implements Database {
  constructor(private executor: SqliteExecutor) {}

  // ============================================================================
  // Users
  // ============================================================================

  async createUser(input: CreateUserInput & { id: string; password_hash: string }): Promise<User> {
    const now = new Date().toISOString();
    await this.executor.run(
      `INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
       VALUES (?, ?, ?, ?, ?, ?)`,
      [input.id, input.email, input.name, input.password_hash, now, now]
    );
    const user = await this.getUserById(input.id);
    if (!user) throw new Error('Failed to create user');
    return user;
  }

  async getUserById(id: string): Promise<User | null> {
    return this.executor.get<User>(
      'SELECT * FROM users WHERE id = ?',
      [id]
    );
  }

  async getUserByEmail(email: string): Promise<User | null> {
    return this.executor.get<User>(
      'SELECT * FROM users WHERE email = ?',
      [email]
    );
  }

  // ============================================================================
  // Sessions
  // ============================================================================

  async createSession(input: CreateSessionInput): Promise<Session> {
    const now = new Date().toISOString();
    await this.executor.run(
      `INSERT INTO sessions (id, user_id, token, expires_at, created_at)
       VALUES (?, ?, ?, ?, ?)`,
      [input.id, input.user_id, input.token, input.expires_at, now]
    );
    const session = await this.executor.get<Session>(
      'SELECT * FROM sessions WHERE id = ?',
      [input.id]
    );
    if (!session) throw new Error('Failed to create session');
    return session;
  }

  async getSessionByToken(token: string): Promise<Session | null> {
    return this.executor.get<Session>(
      `SELECT * FROM sessions WHERE token = ? AND expires_at > datetime('now')`,
      [token]
    );
  }

  async deleteSession(token: string): Promise<void> {
    await this.executor.run('DELETE FROM sessions WHERE token = ?', [token]);
  }

  async deleteExpiredSessions(): Promise<void> {
    await this.executor.run(`DELETE FROM sessions WHERE expires_at <= datetime('now')`);
  }

  // ============================================================================
  // Workbooks
  // ============================================================================

  async createWorkbook(input: CreateWorkbookInput & { id: string; user_id: string }): Promise<Workbook> {
    const now = new Date().toISOString();
    await this.executor.run(
      `INSERT INTO workbooks (id, user_id, name, created_at, updated_at)
       VALUES (?, ?, ?, ?, ?)`,
      [input.id, input.user_id, input.name, now, now]
    );
    const workbook = await this.getWorkbook(input.id);
    if (!workbook) throw new Error('Failed to create workbook');
    return workbook;
  }

  async getWorkbook(id: string): Promise<Workbook | null> {
    return this.executor.get<Workbook>(
      'SELECT * FROM workbooks WHERE id = ?',
      [id]
    );
  }

  async getWorkbooksByUser(userId: string): Promise<Workbook[]> {
    return this.executor.all<Workbook>(
      'SELECT * FROM workbooks WHERE user_id = ? ORDER BY created_at DESC',
      [userId]
    );
  }

  async updateWorkbook(id: string, data: UpdateWorkbookInput): Promise<Workbook | null> {
    const updates: string[] = [];
    const params: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name = ?');
      params.push(data.name);
    }

    if (updates.length === 0) {
      return this.getWorkbook(id);
    }

    updates.push('updated_at = ?');
    params.push(new Date().toISOString());
    params.push(id);

    await this.executor.run(
      `UPDATE workbooks SET ${updates.join(', ')} WHERE id = ?`,
      params
    );

    return this.getWorkbook(id);
  }

  async deleteWorkbook(id: string): Promise<void> {
    await this.executor.run('DELETE FROM workbooks WHERE id = ?', [id]);
  }

  // ============================================================================
  // Sheets
  // ============================================================================

  async createSheet(input: CreateSheetInput & { id: string }): Promise<Sheet> {
    const now = new Date().toISOString();
    const indexNum = input.index_num ?? await this.getMaxSheetIndex(input.workbook_id) + 1;

    await this.executor.run(
      `INSERT INTO sheets (id, workbook_id, name, index_num, row_count, col_count, created_at, updated_at)
       VALUES (?, ?, ?, ?, 1000, 26, ?, ?)`,
      [input.id, input.workbook_id, input.name, indexNum, now, now]
    );
    const sheet = await this.getSheet(input.id);
    if (!sheet) throw new Error('Failed to create sheet');
    return sheet;
  }

  async getSheet(id: string): Promise<Sheet | null> {
    return this.executor.get<Sheet>(
      'SELECT * FROM sheets WHERE id = ?',
      [id]
    );
  }

  async getSheetsByWorkbook(workbookId: string): Promise<Sheet[]> {
    return this.executor.all<Sheet>(
      'SELECT * FROM sheets WHERE workbook_id = ? ORDER BY index_num',
      [workbookId]
    );
  }

  async updateSheet(id: string, data: UpdateSheetInput): Promise<Sheet | null> {
    const updates: string[] = [];
    const params: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name = ?');
      params.push(data.name);
    }
    if (data.index_num !== undefined) {
      updates.push('index_num = ?');
      params.push(data.index_num);
    }

    if (updates.length === 0) {
      return this.getSheet(id);
    }

    updates.push('updated_at = ?');
    params.push(new Date().toISOString());
    params.push(id);

    await this.executor.run(
      `UPDATE sheets SET ${updates.join(', ')} WHERE id = ?`,
      params
    );

    return this.getSheet(id);
  }

  async deleteSheet(id: string): Promise<void> {
    await this.executor.run('DELETE FROM sheets WHERE id = ?', [id]);
  }

  async getMaxSheetIndex(workbookId: string): Promise<number> {
    const result = await this.executor.get<{ max_index: number | null }>(
      'SELECT MAX(index_num) as max_index FROM sheets WHERE workbook_id = ?',
      [workbookId]
    );
    return result?.max_index ?? -1;
  }

  // ============================================================================
  // Cells
  // ============================================================================

  async getCell(sheetId: string, row: number, col: number): Promise<Cell | null> {
    return this.executor.get<Cell>(
      'SELECT * FROM cells WHERE sheet_id = ? AND row_num = ? AND col_num = ?',
      [sheetId, row, col]
    );
  }

  async getCellsBySheet(sheetId: string): Promise<Cell[]> {
    return this.executor.all<Cell>(
      'SELECT * FROM cells WHERE sheet_id = ? ORDER BY row_num, col_num',
      [sheetId]
    );
  }

  async upsertCell(input: UpsertCellInput & { id: string }): Promise<Cell> {
    const now = new Date().toISOString();
    const existing = await this.getCell(input.sheet_id, input.row_num, input.col_num);

    if (existing) {
      const updates: string[] = [];
      const params: unknown[] = [];

      if (input.value !== undefined) {
        updates.push('value = ?');
        params.push(input.value);
      }
      if (input.formula !== undefined) {
        updates.push('formula = ?');
        params.push(input.formula);
      }
      if (input.display !== undefined) {
        updates.push('display = ?');
        params.push(input.display);
      }
      if (input.format !== undefined) {
        updates.push('format = ?');
        params.push(input.format);
      }

      updates.push('updated_at = ?');
      params.push(now);
      params.push(existing.id);

      await this.executor.run(
        `UPDATE cells SET ${updates.join(', ')} WHERE id = ?`,
        params
      );

      return (await this.getCell(input.sheet_id, input.row_num, input.col_num))!;
    }

    await this.executor.run(
      `INSERT INTO cells (id, sheet_id, row_num, col_num, value, formula, display, format, created_at, updated_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      [
        input.id,
        input.sheet_id,
        input.row_num,
        input.col_num,
        input.value ?? null,
        input.formula ?? null,
        input.display ?? null,
        input.format ?? null,
        now,
        now,
      ]
    );

    return (await this.getCell(input.sheet_id, input.row_num, input.col_num))!;
  }

  async upsertCells(inputs: Array<UpsertCellInput & { id: string }>): Promise<Cell[]> {
    const results: Cell[] = [];
    for (const input of inputs) {
      results.push(await this.upsertCell(input));
    }
    return results;
  }

  async deleteCell(sheetId: string, row: number, col: number): Promise<void> {
    await this.executor.run(
      'DELETE FROM cells WHERE sheet_id = ? AND row_num = ? AND col_num = ?',
      [sheetId, row, col]
    );
  }

  async deleteCellsInRange(
    sheetId: string,
    startRow: number,
    endRow: number,
    startCol: number,
    endCol: number
  ): Promise<void> {
    await this.executor.run(
      `DELETE FROM cells
       WHERE sheet_id = ?
       AND row_num >= ? AND row_num <= ?
       AND col_num >= ? AND col_num <= ?`,
      [sheetId, startRow, endRow, startCol, endCol]
    );
  }

  async shiftCellsDown(sheetId: string, startRow: number, count: number): Promise<void> {
    await this.executor.run(
      `UPDATE cells SET row_num = row_num + ?, updated_at = ?
       WHERE sheet_id = ? AND row_num >= ?`,
      [count, new Date().toISOString(), sheetId, startRow]
    );
  }

  async shiftCellsUp(sheetId: string, startRow: number, count: number): Promise<void> {
    await this.executor.run(
      `UPDATE cells SET row_num = row_num - ?, updated_at = ?
       WHERE sheet_id = ? AND row_num >= ?`,
      [count, new Date().toISOString(), sheetId, startRow + count]
    );
  }

  async shiftCellsRight(sheetId: string, startCol: number, count: number): Promise<void> {
    await this.executor.run(
      `UPDATE cells SET col_num = col_num + ?, updated_at = ?
       WHERE sheet_id = ? AND col_num >= ?`,
      [count, new Date().toISOString(), sheetId, startCol]
    );
  }

  async shiftCellsLeft(sheetId: string, startCol: number, count: number): Promise<void> {
    await this.executor.run(
      `UPDATE cells SET col_num = col_num - ?, updated_at = ?
       WHERE sheet_id = ? AND col_num >= ?`,
      [count, new Date().toISOString(), sheetId, startCol + count]
    );
  }

  // ============================================================================
  // Merged Regions
  // ============================================================================

  async getMergedRegions(sheetId: string): Promise<MergedRegion[]> {
    return this.executor.all<MergedRegion>(
      'SELECT * FROM merged_regions WHERE sheet_id = ?',
      [sheetId]
    );
  }

  async createMergedRegion(input: CreateMergeInput & { id: string; sheet_id: string }): Promise<MergedRegion> {
    await this.executor.run(
      `INSERT INTO merged_regions (id, sheet_id, start_row, start_col, end_row, end_col)
       VALUES (?, ?, ?, ?, ?, ?)`,
      [input.id, input.sheet_id, input.start_row, input.start_col, input.end_row, input.end_col]
    );
    const region = await this.executor.get<MergedRegion>(
      'SELECT * FROM merged_regions WHERE id = ?',
      [input.id]
    );
    if (!region) throw new Error('Failed to create merged region');
    return region;
  }

  async deleteMergedRegion(id: string): Promise<void> {
    await this.executor.run('DELETE FROM merged_regions WHERE id = ?', [id]);
  }

  async deleteMergedRegionsInRange(
    sheetId: string,
    startRow: number,
    endRow: number,
    startCol: number,
    endCol: number
  ): Promise<void> {
    await this.executor.run(
      `DELETE FROM merged_regions
       WHERE sheet_id = ?
       AND start_row >= ? AND end_row <= ?
       AND start_col >= ? AND end_col <= ?`,
      [sheetId, startRow, endRow, startCol, endCol]
    );
  }

  // ============================================================================
  // Charts
  // ============================================================================

  async createChart(input: CreateChartInput & { id: string }): Promise<Chart> {
    const now = new Date().toISOString();
    await this.executor.run(
      `INSERT INTO charts (id, sheet_id, title, chart_type, data_range, config, position_x, position_y, width, height, created_at, updated_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      [
        input.id,
        input.sheet_id,
        input.title ?? '',
        input.chart_type ?? 'bar',
        input.data_range,
        input.config ?? null,
        input.position_x ?? 0,
        input.position_y ?? 0,
        input.width ?? 400,
        input.height ?? 300,
        now,
        now,
      ]
    );
    const chart = await this.getChart(input.id);
    if (!chart) throw new Error('Failed to create chart');
    return chart;
  }

  async getChart(id: string): Promise<Chart | null> {
    return this.executor.get<Chart>(
      'SELECT * FROM charts WHERE id = ?',
      [id]
    );
  }

  async getChartsBySheet(sheetId: string): Promise<Chart[]> {
    return this.executor.all<Chart>(
      'SELECT * FROM charts WHERE sheet_id = ? ORDER BY created_at',
      [sheetId]
    );
  }

  async updateChart(id: string, data: UpdateChartInput): Promise<Chart | null> {
    const updates: string[] = [];
    const params: unknown[] = [];

    if (data.title !== undefined) {
      updates.push('title = ?');
      params.push(data.title);
    }
    if (data.chart_type !== undefined) {
      updates.push('chart_type = ?');
      params.push(data.chart_type);
    }
    if (data.data_range !== undefined) {
      updates.push('data_range = ?');
      params.push(data.data_range);
    }
    if (data.config !== undefined) {
      updates.push('config = ?');
      params.push(data.config);
    }
    if (data.position_x !== undefined) {
      updates.push('position_x = ?');
      params.push(data.position_x);
    }
    if (data.position_y !== undefined) {
      updates.push('position_y = ?');
      params.push(data.position_y);
    }
    if (data.width !== undefined) {
      updates.push('width = ?');
      params.push(data.width);
    }
    if (data.height !== undefined) {
      updates.push('height = ?');
      params.push(data.height);
    }

    if (updates.length === 0) {
      return this.getChart(id);
    }

    updates.push('updated_at = ?');
    params.push(new Date().toISOString());
    params.push(id);

    await this.executor.run(
      `UPDATE charts SET ${updates.join(', ')} WHERE id = ?`,
      params
    );

    return this.getChart(id);
  }

  async deleteChart(id: string): Promise<void> {
    await this.executor.run('DELETE FROM charts WHERE id = ?', [id]);
  }

  // ============================================================================
  // Utilities
  // ============================================================================

  async close(): Promise<void> {
    // D1 doesn't need explicit close
  }
}
