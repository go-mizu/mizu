import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import { z } from 'zod';
import type { Env, Variables } from '../types/index.js';
import { CreateTableSchema, UpdateTableSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const tables = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All table routes require authentication
tables.use('*', authRequired);

/**
 * POST /tables - Create table in base
 */
tables.post('/', zValidator('json', CreateTableSchema.extend({ base_id: z.string() })), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { base_id, ...input } = c.req.valid('json');

  // Verify base exists
  const base = await db.getBase(base_id);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  const table = await db.createTable({
    id: ulid(),
    base_id,
    ...input,
    created_by: user.id,
  });

  // Get fields and views for the response
  const fields = await db.getFieldsByTable(table.id);
  const views = await db.getViewsByTable(table.id);

  return c.json({ table: { ...table, fields, views } }, 201);
});

/**
 * GET /tables/:id - Get table with fields and views
 */
tables.get('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const table = await db.getTable(id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  const fields = await db.getFieldsByTable(id);
  const views = await db.getViewsByTable(id);
  const recordCount = await db.getRecordCount(id);

  return c.json({
    table: {
      ...table,
      fields,
      views,
      record_count: recordCount,
    },
  });
});

/**
 * PATCH /tables/:id - Update table
 */
tables.patch('/:id', zValidator('json', UpdateTableSchema), async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();
  const input = c.req.valid('json');

  const table = await db.updateTable(id, input);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  return c.json({ table });
});

/**
 * DELETE /tables/:id - Delete table
 */
tables.delete('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const table = await db.getTable(id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  await db.deleteTable(id);
  return c.json({ message: 'Table deleted' });
});

/**
 * POST /tables/:baseId/reorder - Reorder tables
 */
tables.post('/:baseId/reorder', zValidator('json', z.object({ table_ids: z.array(z.string()) })), async (c) => {
  const db = c.get('db');
  const { baseId } = c.req.param();
  const { table_ids } = c.req.valid('json');

  await db.reorderTables(baseId, table_ids);
  return c.json({ message: 'Tables reordered' });
});

/**
 * GET /tables/:id/fields - List fields
 */
tables.get('/:id/fields', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const table = await db.getTable(id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  const fields = await db.getFieldsByTable(id);
  return c.json({ fields });
});

/**
 * GET /tables/:id/views - List views
 */
tables.get('/:id/views', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const table = await db.getTable(id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  const views = await db.getViewsByTable(id);
  return c.json({ views });
});

export { tables };
