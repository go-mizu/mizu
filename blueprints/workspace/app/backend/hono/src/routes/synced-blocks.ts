import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreateSyncedBlockSchema, UpdateSyncedBlockSchema } from '../models/share';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const syncedBlockRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

syncedBlockRoutes.use('/*', authMiddleware);

// Get synced block
syncedBlockRoutes.get('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const syncedBlock = await services.syncedBlocks.getById(id);
  if (!syncedBlock) {
    return c.json({ error: 'Synced block not found' }, 404);
  }

  return c.json({ syncedBlock });
});

// Create synced block
syncedBlockRoutes.post(
  '/',
  zValidator('json', CreateSyncedBlockSchema),
  async (c) => {
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const syncedBlock = await services.syncedBlocks.create(input, userId);
    return c.json({ syncedBlock }, 201);
  }
);

// Update synced block
syncedBlockRoutes.patch(
  '/:id',
  zValidator('json', UpdateSyncedBlockSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const syncedBlock = await services.syncedBlocks.update(id, input);
    return c.json({ syncedBlock });
  }
);

// Delete synced block
syncedBlockRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  await services.syncedBlocks.delete(id);
  return c.json({ success: true });
});
