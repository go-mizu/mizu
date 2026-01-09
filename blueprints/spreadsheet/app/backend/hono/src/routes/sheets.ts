import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import type { Env, Variables } from '../types/index.js';
import { CreateSheetSchema, UpdateSheetSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const sheets = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All sheet routes require authentication
sheets.use('*', authRequired);

/**
 * Helper to verify user owns the workbook containing the sheet
 */
async function verifySheetAccess(
  db: Database,
  sheetId: string,
  userId: string
): Promise<void> {
  const sheet = await db.getSheet(sheetId);
  if (!sheet) {
    throw ApiError.notFound('Sheet not found');
  }

  const workbook = await db.getWorkbook(sheet.workbook_id);
  if (!workbook || workbook.user_id !== userId) {
    throw ApiError.forbidden('Access denied');
  }
}

/**
 * POST /sheets - Create sheet
 */
sheets.post('/', zValidator('json', CreateSheetSchema), async (c) => {
  const data = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  // Verify workbook access
  const workbook = await db.getWorkbook(data.workbook_id);
  if (!workbook) {
    throw ApiError.notFound('Workbook not found');
  }
  if (workbook.user_id !== user.id) {
    throw ApiError.forbidden('Access denied');
  }

  const sheetId = ulid();
  const sheet = await db.createSheet({
    id: sheetId,
    ...data,
  });

  return c.json({ sheet }, 201);
});

/**
 * GET /sheets/:id - Get sheet
 */
sheets.get('/:id', async (c) => {
  const { id } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, id, user.id);

  const sheet = await db.getSheet(id);
  return c.json({ sheet });
});

/**
 * PATCH /sheets/:id - Update sheet
 */
sheets.patch('/:id', zValidator('json', UpdateSheetSchema), async (c) => {
  const { id } = c.req.param();
  const data = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, id, user.id);

  const updated = await db.updateSheet(id, data);
  return c.json({ sheet: updated });
});

/**
 * DELETE /sheets/:id - Delete sheet
 */
sheets.delete('/:id', async (c) => {
  const { id } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, id, user.id);

  // Don't allow deleting last sheet
  const sheet = await db.getSheet(id);
  if (sheet) {
    const allSheets = await db.getSheetsByWorkbook(sheet.workbook_id);
    if (allSheets.length <= 1) {
      throw ApiError.badRequest('Cannot delete the last sheet in a workbook');
    }
  }

  await db.deleteSheet(id);
  return c.json({ message: 'Sheet deleted' });
});

export { sheets, verifySheetAccess };
