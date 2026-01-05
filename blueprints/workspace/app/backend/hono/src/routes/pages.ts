import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreatePageSchema, UpdatePageSchema } from '../models/page';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const pageRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

pageRoutes.use('/*', authMiddleware);

// Create page
pageRoutes.post(
  '/',
  zValidator('json', CreatePageSchema),
  async (c) => {
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const page = await services.pages.create(input, userId);
    return c.json({ page }, 201);
  }
);

// Get page
pageRoutes.get('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const page = await services.pages.getById(id);
  if (!page) {
    return c.json({ error: 'Page not found' }, 404);
  }

  return c.json({ page });
});

// Update page
pageRoutes.patch(
  '/:id',
  zValidator('json', UpdatePageSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const page = await services.pages.update(id, input);
    return c.json({ page });
  }
);

// Delete page
pageRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  await services.pages.delete(id);
  return c.json({ success: true });
});

// Get page blocks
pageRoutes.get('/:id/blocks', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const blocks = await services.blocks.getBlockTree(id);
  return c.json({ blocks });
});

// Update page blocks
pageRoutes.put('/:id/blocks', async (c) => {
  const id = c.req.param('id');
  const body = await c.req.json();
  const store = c.get('store');
  const services = createServices(store);

  const blocks = await services.blocks.updateBlocks(id, body);
  return c.json({ blocks });
});

// Archive page
pageRoutes.post('/:id/archive', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const page = await services.pages.archive(id);
  return c.json({ page });
});

// Restore page
pageRoutes.post('/:id/restore', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const page = await services.pages.restore(id);
  return c.json({ page });
});

// Duplicate page
pageRoutes.post('/:id/duplicate', async (c) => {
  const id = c.req.param('id');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const page = await services.pages.duplicate(id, userId);
  return c.json({ page }, 201);
});

// Get page hierarchy
pageRoutes.get('/:id/hierarchy', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const hierarchy = await services.pages.getHierarchy(id);
  return c.json({ hierarchy });
});

// Get page comments
pageRoutes.get('/:id/comments', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const comments = await services.comments.listWithAuthors('page', id);
  return c.json({ comments });
});

// Create share for page
pageRoutes.post('/:id/shares', async (c) => {
  const pageId = c.req.param('id');
  const body = await c.req.json();
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const share = await services.sharing.create({ ...body, pageId }, userId);
  return c.json({ share }, 201);
});

// List shares for page
pageRoutes.get('/:id/shares', async (c) => {
  const pageId = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const shares = await services.sharing.listByPage(pageId);
  return c.json({ shares });
});

// Get synced blocks for page
pageRoutes.get('/:id/synced-blocks', async (c) => {
  const pageId = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const syncedBlocks = await services.syncedBlocks.listByPage(pageId);
  return c.json({ syncedBlocks });
});
