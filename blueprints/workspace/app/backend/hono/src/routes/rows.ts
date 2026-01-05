import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { UpdateRowSchema } from '../models/row';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const rowRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

rowRoutes.use('/*', authMiddleware);

// Get row
rowRoutes.get('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const row = await services.rows.getById(id);
  if (!row) {
    return c.json({ error: 'Row not found' }, 404);
  }

  return c.json({ row });
});

// Update row
rowRoutes.patch(
  '/:id',
  zValidator('json', UpdateRowSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const row = await services.rows.update(id, input);
    return c.json({ row });
  }
);

// Delete row
rowRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  await services.rows.delete(id);
  return c.json({ success: true });
});

// Duplicate row
rowRoutes.post('/:id/duplicate', async (c) => {
  const id = c.req.param('id');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const row = await services.rows.duplicate(id, userId);
  return c.json({ row }, 201);
});

// Get row blocks
rowRoutes.get('/:id/blocks', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const blocks = await services.rows.getBlocks(id);
  return c.json({ blocks });
});

// Create block in row
rowRoutes.post('/:id/blocks', async (c) => {
  const id = c.req.param('id');
  const body = await c.req.json();
  const store = c.get('store');
  const services = createServices(store);

  const block = await services.rows.createBlock(id, body);
  return c.json({ block }, 201);
});

// Get row comments
rowRoutes.get('/:id/comments', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const comments = await services.comments.listWithAuthors('database_row', id);
  return c.json({ comments });
});

// Create row comment
rowRoutes.post('/:id/comments', async (c) => {
  const id = c.req.param('id');
  const body = await c.req.json();
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  // Get row to find workspace
  const row = await services.rows.getById(id);
  if (!row) {
    return c.json({ error: 'Row not found' }, 404);
  }

  const comment = await services.comments.create(
    {
      workspaceId: row.workspaceId,
      targetType: 'database_row',
      targetId: id,
      content: body.content,
      parentId: body.parentId,
    },
    userId
  );

  return c.json({ comment }, 201);
});
