import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import type { Env, Variables } from '../types/index.js';
import { CreateWorkspaceSchema, UpdateWorkspaceSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const workspaces = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All workspace routes require authentication
workspaces.use('*', authRequired);

/**
 * GET /workspaces - List user's workspaces
 */
workspaces.get('/', async (c) => {
  const user = c.get('user');
  const db = c.get('db');

  const items = await db.getWorkspacesByUser(user.id);
  return c.json({ workspaces: items });
});

/**
 * POST /workspaces - Create workspace
 */
workspaces.post('/', zValidator('json', CreateWorkspaceSchema), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const input = c.req.valid('json');

  // Check slug uniqueness
  const existing = await db.getWorkspaceBySlug(input.slug);
  if (existing) {
    throw ApiError.conflict('Workspace with this slug already exists');
  }

  const workspace = await db.createWorkspace({
    id: ulid(),
    ...input,
    owner_id: user.id,
  });

  return c.json({ workspace }, 201);
});

/**
 * GET /workspaces/:id - Get workspace
 */
workspaces.get('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const workspace = await db.getWorkspace(id);
  if (!workspace) {
    throw ApiError.notFound('Workspace not found');
  }

  return c.json({ workspace });
});

/**
 * PATCH /workspaces/:id - Update workspace
 */
workspaces.patch('/:id', zValidator('json', UpdateWorkspaceSchema), async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();
  const input = c.req.valid('json');

  const workspace = await db.updateWorkspace(id, input);
  if (!workspace) {
    throw ApiError.notFound('Workspace not found');
  }

  return c.json({ workspace });
});

/**
 * DELETE /workspaces/:id - Delete workspace
 */
workspaces.delete('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const workspace = await db.getWorkspace(id);
  if (!workspace) {
    throw ApiError.notFound('Workspace not found');
  }

  await db.deleteWorkspace(id);
  return c.json({ message: 'Workspace deleted' });
});

/**
 * GET /workspaces/:id/bases - List bases in workspace
 */
workspaces.get('/:id/bases', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const workspace = await db.getWorkspace(id);
  if (!workspace) {
    throw ApiError.notFound('Workspace not found');
  }

  const bases = await db.getBasesByWorkspace(id);
  return c.json({ bases });
});

export { workspaces };
