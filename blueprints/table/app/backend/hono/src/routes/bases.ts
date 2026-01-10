import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import { z } from 'zod';
import type { Env, Variables } from '../types/index.js';
import { CreateBaseSchema, UpdateBaseSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const bases = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All base routes require authentication
bases.use('*', authRequired);

/**
 * POST /bases - Create base in workspace
 */
bases.post('/', zValidator('json', CreateBaseSchema.extend({ workspace_id: z.string() })), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { workspace_id, ...input } = c.req.valid('json');

  // Verify workspace exists
  const workspace = await db.getWorkspace(workspace_id);
  if (!workspace) {
    throw ApiError.notFound('Workspace not found');
  }

  const base = await db.createBase({
    id: ulid(),
    workspace_id,
    ...input,
    created_by: user.id,
  });

  return c.json({ base }, 201);
});

/**
 * GET /bases/:id - Get base with tables
 */
bases.get('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const base = await db.getBase(id);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  const tables = await db.getTablesByBase(id);

  return c.json({ base, tables });
});

/**
 * PATCH /bases/:id - Update base
 */
bases.patch('/:id', zValidator('json', UpdateBaseSchema), async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();
  const input = c.req.valid('json');

  const base = await db.updateBase(id, input);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  return c.json({ base });
});

/**
 * DELETE /bases/:id - Delete base
 */
bases.delete('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const base = await db.getBase(id);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  await db.deleteBase(id);
  return c.json({ message: 'Base deleted' });
});

/**
 * POST /bases/:id/duplicate - Duplicate base
 */
bases.post('/:id/duplicate', async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { id } = c.req.param();

  const base = await db.getBase(id);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  // Create new base with same properties
  const newBase = await db.createBase({
    id: ulid(),
    workspace_id: base.workspace_id,
    name: `${base.name} (copy)`,
    description: base.description || undefined,
    icon: base.icon || undefined,
    color: base.color,
    created_by: user.id,
  });

  // TODO: Copy tables, fields, records

  return c.json({ base: newBase }, 201);
});

/**
 * GET /bases/:id/tables - List tables in base
 */
bases.get('/:id/tables', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const base = await db.getBase(id);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  const tables = await db.getTablesByBase(id);
  return c.json({ tables });
});

export { bases };
