import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import { z } from 'zod';
import type { Env, Variables } from '../types/index.js';
import { CreateCommentSchema, UpdateCommentSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const comments = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All comment routes require authentication
comments.use('*', authRequired);

/**
 * GET /comments/record/:recordId - List comments for a record
 */
comments.get('/record/:recordId', async (c) => {
  const db = c.get('db');
  const { recordId } = c.req.param();

  // Verify record exists
  const record = await db.getRecord(recordId);
  if (!record) {
    throw ApiError.notFound('Record not found');
  }

  const items = await db.getCommentsByRecord(recordId);
  return c.json({ comments: items });
});

/**
 * POST /comments - Create comment
 */
comments.post('/', zValidator('json', CreateCommentSchema.extend({ record_id: z.string() })), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { record_id, ...input } = c.req.valid('json');

  // Verify record exists
  const record = await db.getRecord(record_id);
  if (!record) {
    throw ApiError.notFound('Record not found');
  }

  // If replying, verify parent exists
  if (input.parent_id) {
    const parent = await db.getComment(input.parent_id);
    if (!parent) {
      throw ApiError.notFound('Parent comment not found');
    }
  }

  const comment = await db.createComment({
    id: ulid(),
    record_id,
    ...input,
    author_id: user.id,
  });

  return c.json({ comment }, 201);
});

/**
 * GET /comments/:id - Get comment
 */
comments.get('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const comment = await db.getComment(id);
  if (!comment) {
    throw ApiError.notFound('Comment not found');
  }

  return c.json({ comment });
});

/**
 * PATCH /comments/:id - Update comment
 */
comments.patch('/:id', zValidator('json', UpdateCommentSchema), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { id } = c.req.param();
  const input = c.req.valid('json');

  const existing = await db.getComment(id);
  if (!existing) {
    throw ApiError.notFound('Comment not found');
  }

  // Only author can update content
  if (input.content && existing.author_id !== user.id) {
    throw ApiError.forbidden('You can only edit your own comments');
  }

  const comment = await db.updateComment(id, input);
  return c.json({ comment });
});

/**
 * DELETE /comments/:id - Delete comment
 */
comments.delete('/:id', async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { id } = c.req.param();

  const comment = await db.getComment(id);
  if (!comment) {
    throw ApiError.notFound('Comment not found');
  }

  // Only author can delete
  if (comment.author_id !== user.id) {
    throw ApiError.forbidden('You can only delete your own comments');
  }

  await db.deleteComment(id);
  return c.json({ message: 'Comment deleted' });
});

/**
 * POST /comments/:id/resolve - Resolve comment
 */
comments.post('/:id/resolve', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const comment = await db.updateComment(id, { is_resolved: true });
  if (!comment) {
    throw ApiError.notFound('Comment not found');
  }

  return c.json({ comment });
});

/**
 * POST /comments/:id/unresolve - Unresolve comment
 */
comments.post('/:id/unresolve', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const comment = await db.updateComment(id, { is_resolved: false });
  if (!comment) {
    throw ApiError.notFound('Comment not found');
  }

  return c.json({ comment });
});

export { comments };
