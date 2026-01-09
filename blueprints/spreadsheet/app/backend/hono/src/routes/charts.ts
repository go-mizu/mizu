import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import type { Env, Variables } from '../types/index.js';
import { CreateChartSchema, UpdateChartSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';
import { verifySheetAccess } from './sheets.js';

const charts = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All chart routes require authentication
charts.use('*', authRequired);

/**
 * Helper to verify user owns the workbook containing the chart
 */
async function verifyChartAccess(
  db: Database,
  chartId: string,
  userId: string
): Promise<void> {
  const chart = await db.getChart(chartId);
  if (!chart) {
    throw ApiError.notFound('Chart not found');
  }

  const sheet = await db.getSheet(chart.sheet_id);
  if (!sheet) {
    throw ApiError.notFound('Sheet not found');
  }

  const workbook = await db.getWorkbook(sheet.workbook_id);
  if (!workbook || workbook.user_id !== userId) {
    throw ApiError.forbidden('Access denied');
  }
}

/**
 * POST /charts - Create chart
 */
charts.post('/', zValidator('json', CreateChartSchema), async (c) => {
  const data = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  // Verify sheet access
  await verifySheetAccess(db, data.sheet_id, user.id);

  const chartId = ulid();
  const chart = await db.createChart({
    id: chartId,
    ...data,
  });

  return c.json({
    chart: {
      id: chart.id,
      sheetId: chart.sheet_id,
      title: chart.title,
      chartType: chart.chart_type,
      dataRange: chart.data_range,
      config: chart.config ? JSON.parse(chart.config) : null,
      position: { x: chart.position_x, y: chart.position_y },
      size: { width: chart.width, height: chart.height },
      createdAt: chart.created_at,
      updatedAt: chart.updated_at,
    },
  }, 201);
});

/**
 * GET /charts/:id - Get chart
 */
charts.get('/:id', async (c) => {
  const { id } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifyChartAccess(db, id, user.id);

  const chart = await db.getChart(id);
  if (!chart) {
    throw ApiError.notFound('Chart not found');
  }

  return c.json({
    chart: {
      id: chart.id,
      sheetId: chart.sheet_id,
      title: chart.title,
      chartType: chart.chart_type,
      dataRange: chart.data_range,
      config: chart.config ? JSON.parse(chart.config) : null,
      position: { x: chart.position_x, y: chart.position_y },
      size: { width: chart.width, height: chart.height },
      createdAt: chart.created_at,
      updatedAt: chart.updated_at,
    },
  });
});

/**
 * PATCH /charts/:id - Update chart
 */
charts.patch('/:id', zValidator('json', UpdateChartSchema), async (c) => {
  const { id } = c.req.param();
  const data = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifyChartAccess(db, id, user.id);

  const updated = await db.updateChart(id, data);
  if (!updated) {
    throw ApiError.notFound('Chart not found');
  }

  return c.json({
    chart: {
      id: updated.id,
      sheetId: updated.sheet_id,
      title: updated.title,
      chartType: updated.chart_type,
      dataRange: updated.data_range,
      config: updated.config ? JSON.parse(updated.config) : null,
      position: { x: updated.position_x, y: updated.position_y },
      size: { width: updated.width, height: updated.height },
      createdAt: updated.created_at,
      updatedAt: updated.updated_at,
    },
  });
});

/**
 * DELETE /charts/:id - Delete chart
 */
charts.delete('/:id', async (c) => {
  const { id } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifyChartAccess(db, id, user.id);

  await db.deleteChart(id);
  return c.json({ message: 'Chart deleted' });
});

/**
 * GET /sheets/:sheetId/charts - Get charts for a sheet
 * Note: This is mounted at /sheets/:sheetId/charts in the main app
 */
const sheetCharts = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

sheetCharts.use('*', authRequired);

sheetCharts.get('/:sheetId/charts', async (c) => {
  const { sheetId } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  const chartList = await db.getChartsBySheet(sheetId);

  return c.json({
    charts: chartList.map(chart => ({
      id: chart.id,
      sheetId: chart.sheet_id,
      title: chart.title,
      chartType: chart.chart_type,
      dataRange: chart.data_range,
      config: chart.config ? JSON.parse(chart.config) : null,
      position: { x: chart.position_x, y: chart.position_y },
      size: { width: chart.width, height: chart.height },
      createdAt: chart.created_at,
      updatedAt: chart.updated_at,
    })),
  });
});

export { charts, sheetCharts };
