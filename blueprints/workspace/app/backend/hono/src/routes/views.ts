import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreateViewSchema, UpdateViewSchema, QueryViewSchema } from '../models/view';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const viewRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

viewRoutes.use('/*', authMiddleware);

// Create view
viewRoutes.post(
  '/',
  zValidator('json', CreateViewSchema),
  async (c) => {
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const view = await services.views.create(input);
    return c.json({ view }, 201);
  }
);

// Get view
viewRoutes.get('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const view = await services.views.getById(id);
  if (!view) {
    return c.json({ error: 'View not found' }, 404);
  }

  return c.json({ view });
});

// Update view
viewRoutes.patch(
  '/:id',
  zValidator('json', UpdateViewSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const view = await services.views.update(id, input);
    return c.json({ view });
  }
);

// Delete view
viewRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  await services.views.delete(id);
  return c.json({ success: true });
});

// Query view
viewRoutes.post(
  '/:id/query',
  zValidator('json', QueryViewSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const result = await services.views.query(id, input);
    return c.json(result);
  }
);
