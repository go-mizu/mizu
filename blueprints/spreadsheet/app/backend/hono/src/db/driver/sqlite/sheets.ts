/**
 * SQLite sheets store
 */

import type { SqliteExecutor } from './executor.js';
import type { Sheet, CreateSheetInput, UpdateSheetInput } from '../../types.js';

export class SqliteSheetsStore {
  constructor(private executor: SqliteExecutor) {}

  async createSheet(input: CreateSheetInput & { id: string }): Promise<Sheet> {
    const now = new Date().toISOString();
    const indexNum = input.index_num ?? (await this.getMaxSheetIndex(input.workbook_id)) + 1;

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
      `SELECT * FROM sheets WHERE id = ?`,
      [id]
    );
  }

  async getSheetsByWorkbook(workbookId: string): Promise<Sheet[]> {
    return this.executor.all<Sheet>(
      `SELECT * FROM sheets WHERE workbook_id = ? ORDER BY index_num`,
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
    await this.executor.run(`DELETE FROM sheets WHERE id = ?`, [id]);
  }

  async getMaxSheetIndex(workbookId: string): Promise<number> {
    const result = await this.executor.get<{ max_index: number | null }>(
      `SELECT MAX(index_num) as max_index FROM sheets WHERE workbook_id = ?`,
      [workbookId]
    );
    return result?.max_index ?? -1;
  }
}
