import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import { z } from 'zod';
import type { Env, Variables } from '../types/index.js';
import { CreateShareSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const shares = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All share routes require authentication
shares.use('*', authRequired);

/**
 * GET /shares/base/:baseId - List shares for a base
 */
shares.get('/base/:baseId', async (c) => {
  const db = c.get('db');
  const { baseId } = c.req.param();

  // Verify base exists
  const base = await db.getBase(baseId);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  const items = await db.getSharesByBase(baseId);
  return c.json({ shares: items });
});

/**
 * POST /shares - Create share
 */
shares.post('/', zValidator('json', CreateShareSchema.extend({ base_id: z.string() })), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { base_id, ...input } = c.req.valid('json');

  // Verify base exists
  const base = await db.getBase(base_id);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  // Generate token for public links
  let token: string | undefined;
  if (input.type === 'public_link') {
    token = ulid();
  }

  const share = await db.createShare({
    id: ulid(),
    base_id,
    ...input,
    token,
    created_by: user.id,
  });

  return c.json({ share }, 201);
});

/**
 * GET /shares/:id - Get share
 */
shares.get('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const share = await db.getShare(id);
  if (!share) {
    throw ApiError.notFound('Share not found');
  }

  return c.json({ share });
});

/**
 * DELETE /shares/:id - Delete share
 */
shares.delete('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const share = await db.getShare(id);
  if (!share) {
    throw ApiError.notFound('Share not found');
  }

  await db.deleteShare(id);
  return c.json({ message: 'Share deleted' });
});

/**
 * GET /shares/token/:token - Get shared base by token (public endpoint)
 */
shares.get('/token/:token', async (c) => {
  const db = c.get('db');
  const { token } = c.req.param();

  const share = await db.getShareByToken(token);
  if (!share) {
    throw ApiError.notFound('Share not found');
  }

  // Check expiration
  if (share.expires_at && new Date(share.expires_at) < new Date()) {
    throw ApiError.forbidden('Share link has expired');
  }

  // Get base
  const base = await db.getBase(share.base_id);
  if (!base) {
    throw ApiError.notFound('Base not found');
  }

  // Get tables
  const tables = await db.getTablesByBase(base.id);

  return c.json({
    share: {
      ...share,
      password: undefined, // Don't expose password
    },
    base,
    tables,
  });
});

export { shares };
