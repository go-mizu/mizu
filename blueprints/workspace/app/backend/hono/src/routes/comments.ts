import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreateCommentSchema, UpdateCommentSchema } from '../models/comment';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const commentRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

commentRoutes.use('/*', authMiddleware);

// Create comment
commentRoutes.post(
  '/',
  zValidator('json', CreateCommentSchema),
  async (c) => {
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const comment = await services.comments.create(input, userId);
    return c.json({ comment }, 201);
  }
);

// Update comment
commentRoutes.patch(
  '/:id',
  zValidator('json', UpdateCommentSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const comment = await services.comments.update(id, input, userId);
    return c.json({ comment });
  }
);

// Delete comment
commentRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  await services.comments.delete(id, userId);
  return c.json({ success: true });
});

// Resolve comment
commentRoutes.post('/:id/resolve', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const comment = await services.comments.resolve(id);
  return c.json({ comment });
});
