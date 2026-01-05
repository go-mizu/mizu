import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreateDatabaseSchema, UpdateDatabaseSchema, AddPropertySchema, UpdatePropertySchema } from '../models/database';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const databaseRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

databaseRoutes.use('/*', authMiddleware);

// Create database
databaseRoutes.post(
  '/',
  zValidator('json', CreateDatabaseSchema),
  async (c) => {
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const { database, page } = await services.databases.create(input, userId);
    return c.json({ database, page }, 201);
  }
);

// Get database
databaseRoutes.get('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const database = await services.databases.getById(id);
  if (!database) {
    return c.json({ error: 'Database not found' }, 404);
  }

  return c.json({ database });
});

// Update database
databaseRoutes.patch(
  '/:id',
  zValidator('json', UpdateDatabaseSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const database = await services.databases.update(id, input);
    return c.json({ database });
  }
);

// Delete database
databaseRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  await services.databases.delete(id);
  return c.json({ success: true });
});

// Add property
databaseRoutes.post(
  '/:id/properties',
  zValidator('json', AddPropertySchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const database = await services.databases.addProperty(id, input);
    return c.json({ database }, 201);
  }
);

// Update property
databaseRoutes.patch(
  '/:id/properties/:propId',
  zValidator('json', UpdatePropertySchema),
  async (c) => {
    const id = c.req.param('id');
    const propId = c.req.param('propId');
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const database = await services.databases.updateProperty(id, propId, input);
    return c.json({ database });
  }
);

// Delete property
databaseRoutes.delete('/:id/properties/:propId', async (c) => {
  const id = c.req.param('id');
  const propId = c.req.param('propId');
  const store = c.get('store');
  const services = createServices(store);

  const database = await services.databases.deleteProperty(id, propId);
  return c.json({ database });
});

// List views
databaseRoutes.get('/:id/views', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  const views = await services.views.listByDatabase(id);
  return c.json({ views });
});

// Create view
databaseRoutes.post('/:id/views', async (c) => {
  const databaseId = c.req.param('id');
  const body = await c.req.json();
  const store = c.get('store');
  const services = createServices(store);

  const view = await services.views.create({ ...body, databaseId });
  return c.json({ view }, 201);
});

// List rows
databaseRoutes.get('/:id/rows', async (c) => {
  const id = c.req.param('id');
  const cursor = c.req.query('cursor');
  const limit = c.req.query('limit');
  const store = c.get('store');
  const services = createServices(store);

  const result = await services.rows.listByDatabase(id, {
    cursor,
    limit: limit ? parseInt(limit, 10) : undefined,
  });

  return c.json(result);
});

// Create row
databaseRoutes.post('/:id/rows', async (c) => {
  const databaseId = c.req.param('id');
  const body = await c.req.json();
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const row = await services.rows.create({ ...body, databaseId }, userId);
  return c.json({ row }, 201);
});
