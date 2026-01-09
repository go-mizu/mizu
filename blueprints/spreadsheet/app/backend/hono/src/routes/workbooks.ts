import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import type { Env, Variables } from '../types/index.js';
import { CreateWorkbookSchema, UpdateWorkbookSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const workbooks = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All workbook routes require authentication
workbooks.use('*', authRequired);

/**
 * GET /workbooks - List user's workbooks
 */
workbooks.get('/', async (c) => {
  const user = c.get('user');
  const db = c.get('db');

  const workbookList = await db.getWorkbooksByUser(user.id);
  return c.json({ workbooks: workbookList });
});

/**
 * POST /workbooks - Create workbook
 */
workbooks.post('/', zValidator('json', CreateWorkbookSchema), async (c) => {
  const { name } = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  const workbookId = ulid();
  const workbook = await db.createWorkbook({
    id: workbookId,
    user_id: user.id,
    name,
  });

  // Create default sheet
  const sheetId = ulid();
  await db.createSheet({
    id: sheetId,
    workbook_id: workbookId,
    name: 'Sheet1',
    index_num: 0,
  });

  return c.json({ workbook }, 201);
});

/**
 * GET /workbooks/:id - Get workbook
 */
workbooks.get('/:id', async (c) => {
  const { id } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  const workbook = await db.getWorkbook(id);
  if (!workbook) {
    throw ApiError.notFound('Workbook not found');
  }

  // Check ownership
  if (workbook.user_id !== user.id) {
    throw ApiError.forbidden('Access denied');
  }

  return c.json({ workbook });
});

/**
 * PATCH /workbooks/:id - Update workbook
 */
workbooks.patch('/:id', zValidator('json', UpdateWorkbookSchema), async (c) => {
  const { id } = c.req.param();
  const data = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  const workbook = await db.getWorkbook(id);
  if (!workbook) {
    throw ApiError.notFound('Workbook not found');
  }

  // Check ownership
  if (workbook.user_id !== user.id) {
    throw ApiError.forbidden('Access denied');
  }

  const updated = await db.updateWorkbook(id, data);
  return c.json({ workbook: updated });
});

/**
 * DELETE /workbooks/:id - Delete workbook
 */
workbooks.delete('/:id', async (c) => {
  const { id } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  const workbook = await db.getWorkbook(id);
  if (!workbook) {
    throw ApiError.notFound('Workbook not found');
  }

  // Check ownership
  if (workbook.user_id !== user.id) {
    throw ApiError.forbidden('Access denied');
  }

  await db.deleteWorkbook(id);
  return c.json({ message: 'Workbook deleted' });
});

/**
 * GET /workbooks/:id/sheets - Get workbook sheets
 */
workbooks.get('/:id/sheets', async (c) => {
  const { id } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  const workbook = await db.getWorkbook(id);
  if (!workbook) {
    throw ApiError.notFound('Workbook not found');
  }

  // Check ownership
  if (workbook.user_id !== user.id) {
    throw ApiError.forbidden('Access denied');
  }

  const sheets = await db.getSheetsByWorkbook(id);
  return c.json({ sheets });
});

export { workbooks };
