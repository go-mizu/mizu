/**
 * SQLite workbooks store
 */

import type { SqliteExecutor } from './executor.js';
import type { Workbook, CreateWorkbookInput, UpdateWorkbookInput } from '../../types.js';

export class SqliteWorkbooksStore {
  constructor(private executor: SqliteExecutor) {}

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
      `SELECT * FROM workbooks WHERE id = ?`,
      [id]
    );
  }

  async getWorkbooksByUser(userId: string): Promise<Workbook[]> {
    return this.executor.all<Workbook>(
      `SELECT * FROM workbooks WHERE user_id = ? ORDER BY created_at DESC`,
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
    await this.executor.run(`DELETE FROM workbooks WHERE id = ?`, [id]);
  }
}
