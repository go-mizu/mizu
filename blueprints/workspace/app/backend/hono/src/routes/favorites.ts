import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreateFavoriteSchema } from '../models/share';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const favoriteRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

favoriteRoutes.use('/*', authMiddleware);

// Add favorite
favoriteRoutes.post(
  '/',
  zValidator('json', CreateFavoriteSchema),
  async (c) => {
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const favorite = await services.favorites.add(input.pageId, userId);
    return c.json({ favorite }, 201);
  }
);

// Remove favorite
favoriteRoutes.delete('/:pageId', async (c) => {
  const pageId = c.req.param('pageId');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  await services.favorites.remove(pageId, userId);
  return c.json({ success: true });
});

// Check if favorited
favoriteRoutes.get('/:pageId', async (c) => {
  const pageId = c.req.param('pageId');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const isFavorite = await services.favorites.isFavorite(pageId, userId);
  return c.json({ isFavorite });
});
