import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreateBlockSchema, UpdateBlockSchema, MoveBlockSchema } from '../models/block';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const blockRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

blockRoutes.use('/*', authMiddleware);

// Create block
blockRoutes.post(
  '/',
  zValidator('json', CreateBlockSchema),
  async (c) => {
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const block = await services.blocks.create(input);
    return c.json({ block }, 201);
  }
);

// Update block
blockRoutes.patch(
  '/:id',
  zValidator('json', UpdateBlockSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const block = await services.blocks.update(id, input);
    return c.json({ block });
  }
);

// Delete block
blockRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  await services.blocks.delete(id);
  return c.json({ success: true });
});

// Move block
blockRoutes.post(
  '/:id/move',
  zValidator('json', MoveBlockSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const block = await services.blocks.move(id, input);
    return c.json({ block });
  }
);
