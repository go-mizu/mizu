import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import { z } from 'zod';
import type { Env, Variables } from '../types/index.js';
import { CreateRecordSchema, UpdateRecordSchema, BatchCreateRecordsSchema, BatchUpdateRecordsSchema, BatchDeleteRecordsSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const records = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All record routes require authentication
records.use('*', authRequired);

/**
 * GET /records - List records in table
 */
records.get('/', zValidator('query', z.object({
  table_id: z.string(),
  cursor: z.string().optional(),
  limit: z.coerce.number().int().min(1).max(100).optional().default(50),
})), async (c) => {
  const db = c.get('db');
  const { table_id, cursor, limit } = c.req.valid('query');

  const table = await db.getTable(table_id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  const result = await db.getRecordsByTable(table_id, { cursor, limit });

  return c.json({
    records: result.records,
    next_cursor: result.next_cursor,
    has_more: result.has_more,
  });
});

/**
 * POST /records - Create record
 */
records.post('/', zValidator('json', CreateRecordSchema.extend({ table_id: z.string() })), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { table_id, fields: fieldValues } = c.req.valid('json');

  // Verify table exists
  const table = await db.getTable(table_id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  // Get next position
  const maxPos = await db.getMaxRecordPosition(table_id);
  const position = maxPos + 1;

  // Create record
  const record = await db.createRecord({
    id: ulid(),
    table_id,
    created_by: user.id,
    position,
  });

  // Set cell values
  if (fieldValues) {
    for (const [fieldId, value] of Object.entries(fieldValues)) {
      if (value !== null && value !== undefined) {
        await db.setCellValue(record.id, fieldId, value);
      }
    }
  }

  // Get record with fields
  const recordWithFields = await db.getRecordWithFields(record.id);

  return c.json({ record: recordWithFields }, 201);
});

/**
 * GET /records/:id - Get record
 */
records.get('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const record = await db.getRecordWithFields(id);
  if (!record) {
    throw ApiError.notFound('Record not found');
  }

  return c.json({ record });
});

/**
 * PATCH /records/:id - Update record
 */
records.patch('/:id', zValidator('json', UpdateRecordSchema), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { id } = c.req.param();
  const { fields: fieldValues } = c.req.valid('json');

  const record = await db.getRecord(id);
  if (!record) {
    throw ApiError.notFound('Record not found');
  }

  // Update cell values
  for (const [fieldId, value] of Object.entries(fieldValues)) {
    if (value === null) {
      await db.deleteCellValue(id, fieldId);
    } else {
      await db.setCellValue(id, fieldId, value);
    }
  }

  // Update record timestamp
  await db.updateRecord(id, user.id);

  // Get updated record with fields
  const updatedRecord = await db.getRecordWithFields(id);

  return c.json({ record: updatedRecord });
});

/**
 * DELETE /records/:id - Delete record
 */
records.delete('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const record = await db.getRecord(id);
  if (!record) {
    throw ApiError.notFound('Record not found');
  }

  await db.deleteRecord(id);
  return c.json({ message: 'Record deleted' });
});

// ============================================================================
// Batch Operations
// ============================================================================

/**
 * POST /records/batch - Batch create records
 */
records.post('/batch', zValidator('json', BatchCreateRecordsSchema.extend({ table_id: z.string() })), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { table_id, records: recordInputs } = c.req.valid('json');

  // Verify table exists
  const table = await db.getTable(table_id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  const createdRecords = [];

  // Get starting position
  let position = await db.getMaxRecordPosition(table_id);

  for (const input of recordInputs) {
    position++;

    // Create record
    const record = await db.createRecord({
      id: ulid(),
      table_id,
      created_by: user.id,
      position,
    });

    // Set cell values
    if (input.fields) {
      for (const [fieldId, value] of Object.entries(input.fields)) {
        if (value !== null && value !== undefined) {
          await db.setCellValue(record.id, fieldId, value);
        }
      }
    }

    const recordWithFields = await db.getRecordWithFields(record.id);
    createdRecords.push(recordWithFields);
  }

  return c.json({ records: createdRecords }, 201);
});

/**
 * PATCH /records/batch - Batch update records
 */
records.patch('/batch', zValidator('json', BatchUpdateRecordsSchema), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { records: recordUpdates } = c.req.valid('json');

  const updatedRecords = [];

  for (const update of recordUpdates) {
    const record = await db.getRecord(update.id);
    if (!record) {
      continue; // Skip non-existent records
    }

    // Update cell values
    for (const [fieldId, value] of Object.entries(update.fields)) {
      if (value === null) {
        await db.deleteCellValue(update.id, fieldId);
      } else {
        await db.setCellValue(update.id, fieldId, value);
      }
    }

    // Update record timestamp
    await db.updateRecord(update.id, user.id);

    const recordWithFields = await db.getRecordWithFields(update.id);
    if (recordWithFields) {
      updatedRecords.push(recordWithFields);
    }
  }

  return c.json({ records: updatedRecords });
});

/**
 * DELETE /records/batch - Batch delete records
 */
records.delete('/batch', zValidator('json', BatchDeleteRecordsSchema), async (c) => {
  const db = c.get('db');
  const { ids } = c.req.valid('json');

  for (const id of ids) {
    await db.deleteRecord(id);
  }

  return c.json({ message: `Deleted ${ids.length} records` });
});

export { records };
