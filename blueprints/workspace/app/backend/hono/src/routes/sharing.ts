import { Hono } from 'hono';
import type { Env, Variables } from '../env';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const shareRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

// Validate share link (public, no auth required) - must be before auth middleware
shareRoutes.get('/validate/:token', async (c) => {
  const token = c.req.param('token');
  const password = c.req.query('password');
  const store = c.get('store');
  const services = createServices(store);

  const result = await services.sharing.validateLinkAccess(token, password);
  if (!result) {
    return c.json({ error: 'Invalid or expired share link' }, 404);
  }

  // Fetch page details for the response
  const page = await store.pages.getById(result.pageId);

  return c.json({
    page: page ? { id: page.id, title: page.title } : null,
    permission: result.share.permission,
  });
});

// All other share routes require authentication
shareRoutes.use('/*', authMiddleware);

// Delete share
shareRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const store = c.get('store');
  const services = createServices(store);

  await services.sharing.delete(id);
  return c.json({ success: true });
});
