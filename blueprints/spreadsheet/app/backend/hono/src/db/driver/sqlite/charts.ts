/**
 * SQLite charts store
 */

import type { SqliteExecutor } from './executor.js';
import type { Chart, CreateChartInput, UpdateChartInput } from '../../types.js';

export class SqliteChartsStore {
  constructor(private executor: SqliteExecutor) {}

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
      `SELECT * FROM charts WHERE id = ?`,
      [id]
    );
  }

  async getChartsBySheet(sheetId: string): Promise<Chart[]> {
    return this.executor.all<Chart>(
      `SELECT * FROM charts WHERE sheet_id = ? ORDER BY created_at`,
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
    await this.executor.run(`DELETE FROM charts WHERE id = ?`, [id]);
  }
}
