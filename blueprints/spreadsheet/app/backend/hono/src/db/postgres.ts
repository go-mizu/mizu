import postgres from 'postgres';
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
 * PostgreSQL database implementation
 */
export class PostgresDatabase implements Database {
  private sql: postgres.Sql;

  constructor(connectionString: string) {
    this.sql = postgres(connectionString, {
      max: 10,
      idle_timeout: 20,
      connect_timeout: 10,
    });
  }

  // ============================================================================
  // Users
  // ============================================================================

  async createUser(input: CreateUserInput & { id: string; password_hash: string }): Promise<User> {
    const [user] = await this.sql<User[]>`
      INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
      VALUES (${input.id}, ${input.email}, ${input.name}, ${input.password_hash}, NOW(), NOW())
      RETURNING *
    `;
    return user;
  }

  async getUserById(id: string): Promise<User | null> {
    const [user] = await this.sql<User[]>`
      SELECT * FROM users WHERE id = ${id}
    `;
    return user ?? null;
  }

  async getUserByEmail(email: string): Promise<User | null> {
    const [user] = await this.sql<User[]>`
      SELECT * FROM users WHERE email = ${email}
    `;
    return user ?? null;
  }

  // ============================================================================
  // Sessions
  // ============================================================================

  async createSession(input: CreateSessionInput): Promise<Session> {
    const [session] = await this.sql<Session[]>`
      INSERT INTO sessions (id, user_id, token, expires_at, created_at)
      VALUES (${input.id}, ${input.user_id}, ${input.token}, ${input.expires_at}, NOW())
      RETURNING *
    `;
    return session;
  }

  async getSessionByToken(token: string): Promise<Session | null> {
    const [session] = await this.sql<Session[]>`
      SELECT * FROM sessions WHERE token = ${token} AND expires_at > NOW()
    `;
    return session ?? null;
  }

  async deleteSession(token: string): Promise<void> {
    await this.sql`DELETE FROM sessions WHERE token = ${token}`;
  }

  async deleteExpiredSessions(): Promise<void> {
    await this.sql`DELETE FROM sessions WHERE expires_at <= NOW()`;
  }

  // ============================================================================
  // Workbooks
  // ============================================================================

  async createWorkbook(input: CreateWorkbookInput & { id: string; user_id: string }): Promise<Workbook> {
    const [workbook] = await this.sql<Workbook[]>`
      INSERT INTO workbooks (id, user_id, name, created_at, updated_at)
      VALUES (${input.id}, ${input.user_id}, ${input.name}, NOW(), NOW())
      RETURNING *
    `;
    return workbook;
  }

  async getWorkbook(id: string): Promise<Workbook | null> {
    const [workbook] = await this.sql<Workbook[]>`
      SELECT * FROM workbooks WHERE id = ${id}
    `;
    return workbook ?? null;
  }

  async getWorkbooksByUser(userId: string): Promise<Workbook[]> {
    return this.sql<Workbook[]>`
      SELECT * FROM workbooks WHERE user_id = ${userId} ORDER BY created_at DESC
    `;
  }

  async updateWorkbook(id: string, data: UpdateWorkbookInput): Promise<Workbook | null> {
    if (data.name === undefined) {
      return this.getWorkbook(id);
    }
    const [workbook] = await this.sql<Workbook[]>`
      UPDATE workbooks
      SET name = ${data.name}, updated_at = NOW()
      WHERE id = ${id}
      RETURNING *
    `;
    return workbook ?? null;
  }

  async deleteWorkbook(id: string): Promise<void> {
    await this.sql`DELETE FROM workbooks WHERE id = ${id}`;
  }

  // ============================================================================
  // Sheets
  // ============================================================================

  async createSheet(input: CreateSheetInput & { id: string }): Promise<Sheet> {
    const indexNum = input.index_num ?? (await this.getMaxSheetIndex(input.workbook_id)) + 1;
    const [sheet] = await this.sql<Sheet[]>`
      INSERT INTO sheets (id, workbook_id, name, index_num, row_count, col_count, created_at, updated_at)
      VALUES (${input.id}, ${input.workbook_id}, ${input.name}, ${indexNum}, 1000, 26, NOW(), NOW())
      RETURNING *
    `;
    return sheet;
  }

  async getSheet(id: string): Promise<Sheet | null> {
    const [sheet] = await this.sql<Sheet[]>`
      SELECT * FROM sheets WHERE id = ${id}
    `;
    return sheet ?? null;
  }

  async getSheetsByWorkbook(workbookId: string): Promise<Sheet[]> {
    return this.sql<Sheet[]>`
      SELECT * FROM sheets WHERE workbook_id = ${workbookId} ORDER BY index_num
    `;
  }

  async updateSheet(id: string, data: UpdateSheetInput): Promise<Sheet | null> {
    const values: Record<string, unknown> = {};

    if (data.name !== undefined) {
      values.name = data.name;
    }
    if (data.index_num !== undefined) {
      values.index_num = data.index_num;
    }

    if (Object.keys(values).length === 0) {
      return this.getSheet(id);
    }

    // Build update dynamically
    const [sheet] = await this.sql<Sheet[]>`
      UPDATE sheets
      SET ${this.sql(values)}, updated_at = NOW()
      WHERE id = ${id}
      RETURNING *
    `;
    return sheet ?? null;
  }

  async deleteSheet(id: string): Promise<void> {
    await this.sql`DELETE FROM sheets WHERE id = ${id}`;
  }

  async getMaxSheetIndex(workbookId: string): Promise<number> {
    const [result] = await this.sql<[{ max_index: number | null }]>`
      SELECT COALESCE(MAX(index_num), -1) as max_index FROM sheets WHERE workbook_id = ${workbookId}
    `;
    return result?.max_index ?? -1;
  }

  // ============================================================================
  // Cells
  // ============================================================================

  async getCell(sheetId: string, row: number, col: number): Promise<Cell | null> {
    const [cell] = await this.sql<Cell[]>`
      SELECT * FROM cells WHERE sheet_id = ${sheetId} AND row_num = ${row} AND col_num = ${col}
    `;
    return cell ?? null;
  }

  async getCellsBySheet(sheetId: string): Promise<Cell[]> {
    return this.sql<Cell[]>`
      SELECT * FROM cells WHERE sheet_id = ${sheetId} ORDER BY row_num, col_num
    `;
  }

  async upsertCell(input: UpsertCellInput & { id: string }): Promise<Cell> {
    const [cell] = await this.sql<Cell[]>`
      INSERT INTO cells (id, sheet_id, row_num, col_num, value, formula, display, format, created_at, updated_at)
      VALUES (
        ${input.id},
        ${input.sheet_id},
        ${input.row_num},
        ${input.col_num},
        ${input.value ?? null},
        ${input.formula ?? null},
        ${input.display ?? null},
        ${input.format ?? null},
        NOW(),
        NOW()
      )
      ON CONFLICT (sheet_id, row_num, col_num)
      DO UPDATE SET
        value = COALESCE(${input.value ?? null}, cells.value),
        formula = COALESCE(${input.formula ?? null}, cells.formula),
        display = COALESCE(${input.display ?? null}, cells.display),
        format = COALESCE(${input.format ?? null}, cells.format),
        updated_at = NOW()
      RETURNING *
    `;
    return cell;
  }

  async upsertCells(inputs: Array<UpsertCellInput & { id: string }>): Promise<Cell[]> {
    if (inputs.length === 0) return [];

    // For simplicity, do sequential upserts (could be optimized with bulk insert)
    const results: Cell[] = [];
    for (const input of inputs) {
      results.push(await this.upsertCell(input));
    }
    return results;
  }

  async deleteCell(sheetId: string, row: number, col: number): Promise<void> {
    await this.sql`
      DELETE FROM cells WHERE sheet_id = ${sheetId} AND row_num = ${row} AND col_num = ${col}
    `;
  }

  async deleteCellsInRange(
    sheetId: string,
    startRow: number,
    endRow: number,
    startCol: number,
    endCol: number
  ): Promise<void> {
    await this.sql`
      DELETE FROM cells
      WHERE sheet_id = ${sheetId}
      AND row_num >= ${startRow} AND row_num <= ${endRow}
      AND col_num >= ${startCol} AND col_num <= ${endCol}
    `;
  }

  async shiftCellsDown(sheetId: string, startRow: number, count: number): Promise<void> {
    await this.sql`
      UPDATE cells
      SET row_num = row_num + ${count}, updated_at = NOW()
      WHERE sheet_id = ${sheetId} AND row_num >= ${startRow}
    `;
  }

  async shiftCellsUp(sheetId: string, startRow: number, count: number): Promise<void> {
    await this.sql`
      UPDATE cells
      SET row_num = row_num - ${count}, updated_at = NOW()
      WHERE sheet_id = ${sheetId} AND row_num >= ${startRow + count}
    `;
  }

  async shiftCellsRight(sheetId: string, startCol: number, count: number): Promise<void> {
    await this.sql`
      UPDATE cells
      SET col_num = col_num + ${count}, updated_at = NOW()
      WHERE sheet_id = ${sheetId} AND col_num >= ${startCol}
    `;
  }

  async shiftCellsLeft(sheetId: string, startCol: number, count: number): Promise<void> {
    await this.sql`
      UPDATE cells
      SET col_num = col_num - ${count}, updated_at = NOW()
      WHERE sheet_id = ${sheetId} AND col_num >= ${startCol + count}
    `;
  }

  // ============================================================================
  // Merged Regions
  // ============================================================================

  async getMergedRegions(sheetId: string): Promise<MergedRegion[]> {
    return this.sql<MergedRegion[]>`
      SELECT * FROM merged_regions WHERE sheet_id = ${sheetId}
    `;
  }

  async createMergedRegion(input: CreateMergeInput & { id: string; sheet_id: string }): Promise<MergedRegion> {
    const [region] = await this.sql<MergedRegion[]>`
      INSERT INTO merged_regions (id, sheet_id, start_row, start_col, end_row, end_col)
      VALUES (${input.id}, ${input.sheet_id}, ${input.start_row}, ${input.start_col}, ${input.end_row}, ${input.end_col})
      RETURNING *
    `;
    return region;
  }

  async deleteMergedRegion(id: string): Promise<void> {
    await this.sql`DELETE FROM merged_regions WHERE id = ${id}`;
  }

  async deleteMergedRegionsInRange(
    sheetId: string,
    startRow: number,
    endRow: number,
    startCol: number,
    endCol: number
  ): Promise<void> {
    await this.sql`
      DELETE FROM merged_regions
      WHERE sheet_id = ${sheetId}
      AND start_row >= ${startRow} AND end_row <= ${endRow}
      AND start_col >= ${startCol} AND end_col <= ${endCol}
    `;
  }

  // ============================================================================
  // Charts
  // ============================================================================

  async createChart(input: CreateChartInput & { id: string }): Promise<Chart> {
    const [chart] = await this.sql<Chart[]>`
      INSERT INTO charts (id, sheet_id, title, chart_type, data_range, config, position_x, position_y, width, height, created_at, updated_at)
      VALUES (
        ${input.id},
        ${input.sheet_id},
        ${input.title ?? ''},
        ${input.chart_type ?? 'bar'},
        ${input.data_range},
        ${input.config ?? null},
        ${input.position_x ?? 0},
        ${input.position_y ?? 0},
        ${input.width ?? 400},
        ${input.height ?? 300},
        NOW(),
        NOW()
      )
      RETURNING *
    `;
    return chart;
  }

  async getChart(id: string): Promise<Chart | null> {
    const [chart] = await this.sql<Chart[]>`
      SELECT * FROM charts WHERE id = ${id}
    `;
    return chart ?? null;
  }

  async getChartsBySheet(sheetId: string): Promise<Chart[]> {
    return this.sql<Chart[]>`
      SELECT * FROM charts WHERE sheet_id = ${sheetId} ORDER BY created_at
    `;
  }

  async updateChart(id: string, data: UpdateChartInput): Promise<Chart | null> {
    const values: Record<string, unknown> = {};

    if (data.title !== undefined) values.title = data.title;
    if (data.chart_type !== undefined) values.chart_type = data.chart_type;
    if (data.data_range !== undefined) values.data_range = data.data_range;
    if (data.config !== undefined) values.config = data.config;
    if (data.position_x !== undefined) values.position_x = data.position_x;
    if (data.position_y !== undefined) values.position_y = data.position_y;
    if (data.width !== undefined) values.width = data.width;
    if (data.height !== undefined) values.height = data.height;

    if (Object.keys(values).length === 0) {
      return this.getChart(id);
    }

    const [chart] = await this.sql<Chart[]>`
      UPDATE charts
      SET ${this.sql(values)}, updated_at = NOW()
      WHERE id = ${id}
      RETURNING *
    `;
    return chart ?? null;
  }

  async deleteChart(id: string): Promise<void> {
    await this.sql`DELETE FROM charts WHERE id = ${id}`;
  }

  // ============================================================================
  // Utilities
  // ============================================================================

  async close(): Promise<void> {
    await this.sql.end();
  }
}
