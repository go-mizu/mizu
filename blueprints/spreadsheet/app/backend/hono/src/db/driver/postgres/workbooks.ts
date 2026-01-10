/**
 * PostgreSQL workbooks store
 */

import type postgres from 'postgres';
import type { Workbook, CreateWorkbookInput, UpdateWorkbookInput } from '../../types.js';

export class PostgresWorkbooksStore {
  constructor(private sql: postgres.Sql) {}

  async createWorkbook(input: CreateWorkbookInput & { id: string; user_id: string }): Promise<Workbook> {
    const [workbook] = await this.sql<[Workbook]>`
      INSERT INTO workbooks (id, user_id, name, created_at, updated_at)
      VALUES (${input.id}, ${input.user_id}, ${input.name}, NOW(), NOW())
      RETURNING *
    `;
    return workbook;
  }

  async getWorkbook(id: string): Promise<Workbook | null> {
    const [workbook] = await this.sql<[Workbook | undefined]>`
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
    const [workbook] = await this.sql<[Workbook | undefined]>`
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
}
