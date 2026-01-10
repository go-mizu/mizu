import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import { z } from 'zod';
import type { Env, Variables } from '../types/index.js';
import { CreateViewSchema, UpdateViewSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const views = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All view routes require authentication
views.use('*', authRequired);

/**
 * POST /views - Create view in table
 */
views.post('/', zValidator('json', CreateViewSchema.extend({ table_id: z.string() })), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { table_id, ...input } = c.req.valid('json');

  // Verify table exists
  const table = await db.getTable(table_id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  // Get next position
  const maxPos = await db.getMaxViewPosition(table_id);
  const position = maxPos + 1;

  const view = await db.createView({
    id: ulid(),
    table_id,
    ...input,
    position,
    created_by: user.id,
  });

  return c.json({ view }, 201);
});

/**
 * GET /views/:id - Get view
 */
views.get('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const view = await db.getView(id);
  if (!view) {
    throw ApiError.notFound('View not found');
  }

  return c.json({ view });
});

/**
 * PATCH /views/:id - Update view
 */
views.patch('/:id', zValidator('json', UpdateViewSchema), async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();
  const input = c.req.valid('json');

  const view = await db.updateView(id, input);
  if (!view) {
    throw ApiError.notFound('View not found');
  }

  return c.json({ view });
});

/**
 * DELETE /views/:id - Delete view
 */
views.delete('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const view = await db.getView(id);
  if (!view) {
    throw ApiError.notFound('View not found');
  }

  // Prevent deleting the last view
  const tableViews = await db.getViewsByTable(view.table_id);
  if (tableViews.length <= 1) {
    throw ApiError.badRequest('Cannot delete the last view');
  }

  await db.deleteView(id);
  return c.json({ message: 'View deleted' });
});

/**
 * POST /views/:id/duplicate - Duplicate view
 */
views.post('/:id/duplicate', async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { id } = c.req.param();

  const view = await db.getView(id);
  if (!view) {
    throw ApiError.notFound('View not found');
  }

  // Get next position
  const maxPos = await db.getMaxViewPosition(view.table_id);
  const position = maxPos + 1;

  const newView = await db.createView({
    id: ulid(),
    table_id: view.table_id,
    name: `${view.name} (copy)`,
    type: view.type,
    filters: view.filters,
    sorts: view.sorts,
    groups: view.groups,
    field_config: view.field_config,
    settings: view.settings,
    position,
    created_by: user.id,
  });

  return c.json({ view: newView }, 201);
});

/**
 * POST /views/:tableId/reorder - Reorder views
 */
views.post('/:tableId/reorder', zValidator('json', z.object({ view_ids: z.array(z.string()) })), async (c) => {
  const db = c.get('db');
  const { tableId } = c.req.param();
  const { view_ids } = c.req.valid('json');

  await db.reorderViews(tableId, view_ids);
  return c.json({ message: 'Views reordered' });
});

export { views };
