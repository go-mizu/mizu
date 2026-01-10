/**
 * PostgreSQL charts store
 */

import type postgres from 'postgres';
import type { Chart, CreateChartInput, UpdateChartInput } from '../../types.js';

export class PostgresChartsStore {
  constructor(private sql: postgres.Sql) {}

  async createChart(input: CreateChartInput & { id: string }): Promise<Chart> {
    const [chart] = await this.sql<[Chart]>`
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
    const [chart] = await this.sql<[Chart | undefined]>`
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

    const [chart] = await this.sql<[Chart | undefined]>`
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
}
