/**
 * PostgreSQL sheets store
 */

import type postgres from 'postgres';
import type { Sheet, CreateSheetInput, UpdateSheetInput } from '../../types.js';

export class PostgresSheetsStore {
  constructor(private sql: postgres.Sql) {}

  async createSheet(input: CreateSheetInput & { id: string }): Promise<Sheet> {
    const indexNum = input.index_num ?? (await this.getMaxSheetIndex(input.workbook_id)) + 1;
    const [sheet] = await this.sql<[Sheet]>`
      INSERT INTO sheets (id, workbook_id, name, index_num, row_count, col_count, created_at, updated_at)
      VALUES (${input.id}, ${input.workbook_id}, ${input.name}, ${indexNum}, 1000, 26, NOW(), NOW())
      RETURNING *
    `;
    return sheet;
  }

  async getSheet(id: string): Promise<Sheet | null> {
    const [sheet] = await this.sql<[Sheet | undefined]>`
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

    const [sheet] = await this.sql<[Sheet | undefined]>`
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
}
