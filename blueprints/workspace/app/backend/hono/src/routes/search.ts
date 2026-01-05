import { Hono } from 'hono';
import type { Env, Variables } from '../env';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const searchRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

searchRoutes.use('/*', authMiddleware);

// Full search
searchRoutes.get('/workspaces/:id/search', async (c) => {
  const workspaceId = c.req.param('id');
  const query = c.req.query('q') ?? '';
  const store = c.get('store');
  const services = createServices(store);

  const results = await services.search.search(workspaceId, query);
  return c.json({ results });
});

// Quick search
searchRoutes.get('/workspaces/:id/quick-search', async (c) => {
  const workspaceId = c.req.param('id');
  const query = c.req.query('q') ?? '';
  const store = c.get('store');
  const services = createServices(store);

  const results = await services.search.quickSearch(workspaceId, query);
  return c.json({ results });
});

// Recent items
searchRoutes.get('/workspaces/:id/recent', async (c) => {
  const workspaceId = c.req.param('id');
  const userId = c.get('userId')!;
  const limit = c.req.query('limit');
  const store = c.get('store');
  const services = createServices(store);

  const items = await services.search.getRecent(
    workspaceId,
    userId,
    limit ? parseInt(limit, 10) : undefined
  );

  return c.json({ items });
});
