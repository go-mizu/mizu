/**
 * News API Routes.
 * Provides endpoints for news home, categories, search, Full Coverage, and personalization.
 */

import { Hono } from 'hono';
import { NewsService } from '../services/news';
import { getSessionId } from '../middleware/session';
import type { NewsCategory, UserLocation } from '../types';

type Env = {
  Bindings: {
    SEARCH_KV: KVNamespace;
  };
  Variables: {
    sessionId: string;
  };
};

const newsRoutes = new Hono<Env>();

// Lazy-init news service per request (uses KV from env)
function getNewsService(kv: KVNamespace): NewsService {
  return new NewsService(kv);
}

// Valid news categories
const VALID_CATEGORIES: NewsCategory[] = [
  'top',
  'world',
  'nation',
  'business',
  'technology',
  'science',
  'health',
  'sports',
  'entertainment',
];

/**
 * GET /api/news/home
 * Get aggregated news home feed with Top Stories, For You, Local News, and category previews.
 */
newsRoutes.get('/home', async (c) => {
  const userId = getSessionId(c);
  const service = getNewsService(c.env.SEARCH_KV);
  await service.initCache();

  const response = await service.getHomeFeed(userId);
  return c.json(response);
});

/**
 * GET /api/news/category/:category
 * Get news feed for a specific category.
 */
newsRoutes.get('/category/:category', async (c) => {
  const category = c.req.param('category') as NewsCategory;

  if (!VALID_CATEGORIES.includes(category)) {
    return c.json({ error: `Invalid category: ${category}` }, 400);
  }

  const page = parseInt(c.req.query('page') || '1', 10);
  const userId = getSessionId(c);
  const service = getNewsService(c.env.SEARCH_KV);
  await service.initCache();

  const response = await service.getCategoryFeed(userId, category, page);
  return c.json(response);
});

/**
 * GET /api/news/search
 * Search news articles.
 */
newsRoutes.get('/search', async (c) => {
  const query = c.req.query('q');

  if (!query) {
    return c.json({ error: 'Query parameter "q" is required' }, 400);
  }

  const page = parseInt(c.req.query('page') || '1', 10);
  const timeRange = c.req.query('time');
  const source = c.req.query('source');
  const category = c.req.query('category') as NewsCategory | undefined;

  const userId = getSessionId(c);
  const service = getNewsService(c.env.SEARCH_KV);
  await service.initCache();

  const response = await service.searchNews(userId, query, {
    page,
    timeRange,
    source,
    category,
  });

  return c.json(response);
});

/**
 * GET /api/news/story/:storyId
 * Get Full Coverage for a story cluster.
 */
newsRoutes.get('/story/:storyId', async (c) => {
  const storyId = c.req.param('storyId');

  if (!storyId) {
    return c.json({ error: 'Story ID is required' }, 400);
  }

  const userId = getSessionId(c);
  const service = getNewsService(c.env.SEARCH_KV);
  await service.initCache();

  const cluster = await service.getFullCoverage(userId, storyId);

  if (!cluster) {
    return c.json({ error: 'Story not found' }, 404);
  }

  return c.json(cluster);
});

/**
 * GET /api/news/local
 * Get local news for user's location or specified location.
 */
newsRoutes.get('/local', async (c) => {
  const city = c.req.query('city');
  const state = c.req.query('state');
  const country = c.req.query('country');
  const lat = c.req.query('lat');
  const lng = c.req.query('lng');

  let location: UserLocation | undefined;

  if (city && country) {
    location = {
      city,
      state: state || undefined,
      country,
      lat: lat ? parseFloat(lat) : undefined,
      lng: lng ? parseFloat(lng) : undefined,
    };
  }

  const userId = getSessionId(c);
  const service = getNewsService(c.env.SEARCH_KV);
  await service.initCache();

  const articles = await service.getLocalNews(userId, location);
  return c.json({ articles });
});

/**
 * GET /api/news/following
 * Get news from followed topics and sources.
 */
newsRoutes.get('/following', async (c) => {
  const userId = getSessionId(c);
  const service = getNewsService(c.env.SEARCH_KV);
  await service.initCache();

  const articles = await service.getFollowingFeed(userId);
  return c.json({ articles });
});

/**
 * GET /api/news/preferences
 * Get user's news preferences.
 */
newsRoutes.get('/preferences', async (c) => {
  const userId = getSessionId(c);
  const service = getNewsService(c.env.SEARCH_KV);

  const preferences = await service.getPreferences(userId);
  return c.json(preferences);
});

/**
 * PUT /api/news/preferences
 * Update user's news preferences.
 */
newsRoutes.put('/preferences', async (c) => {
  const userId = getSessionId(c);
  const updates = await c.req.json();
  const service = getNewsService(c.env.SEARCH_KV);

  const preferences = await service.updatePreferences(userId, updates);
  return c.json(preferences);
});

/**
 * POST /api/news/follow
 * Follow a topic or source.
 */
newsRoutes.post('/follow', async (c) => {
  const userId = getSessionId(c);
  const body = await c.req.json<{ type: 'topic' | 'source'; id: string }>();

  if (!body.type || !body.id) {
    return c.json({ error: 'type and id are required' }, 400);
  }

  const service = getNewsService(c.env.SEARCH_KV);

  if (body.type === 'topic') {
    await service.followTopic(userId, body.id);
  } else {
    await service.followSource(userId, body.id);
  }

  return c.json({ success: true });
});

/**
 * DELETE /api/news/follow
 * Unfollow a topic or source.
 */
newsRoutes.delete('/follow', async (c) => {
  const userId = getSessionId(c);
  const body = await c.req.json<{ type: 'topic' | 'source'; id: string }>();

  if (!body.type || !body.id) {
    return c.json({ error: 'type and id are required' }, 400);
  }

  const service = getNewsService(c.env.SEARCH_KV);

  if (body.type === 'topic') {
    await service.unfollowTopic(userId, body.id);
  } else {
    await service.unfollowSource(userId, body.id);
  }

  return c.json({ success: true });
});

/**
 * POST /api/news/hide
 * Hide a source.
 */
newsRoutes.post('/hide', async (c) => {
  const userId = getSessionId(c);
  const body = await c.req.json<{ source: string }>();

  if (!body.source) {
    return c.json({ error: 'source is required' }, 400);
  }

  const service = getNewsService(c.env.SEARCH_KV);
  await service.hideSource(userId, body.source);

  return c.json({ success: true });
});

/**
 * DELETE /api/news/hide
 * Unhide a source.
 */
newsRoutes.delete('/hide', async (c) => {
  const userId = getSessionId(c);
  const body = await c.req.json<{ source: string }>();

  if (!body.source) {
    return c.json({ error: 'source is required' }, 400);
  }

  const service = getNewsService(c.env.SEARCH_KV);
  await service.unhideSource(userId, body.source);

  return c.json({ success: true });
});

/**
 * POST /api/news/location
 * Set user's primary location.
 */
newsRoutes.post('/location', async (c) => {
  const userId = getSessionId(c);
  const location = await c.req.json<UserLocation>();

  if (!location.city || !location.country) {
    return c.json({ error: 'city and country are required' }, 400);
  }

  const service = getNewsService(c.env.SEARCH_KV);
  await service.setLocation(userId, location);

  return c.json({ success: true });
});

/**
 * POST /api/news/read
 * Record article read for personalization.
 */
newsRoutes.post('/read', async (c) => {
  const userId = getSessionId(c);
  const body = await c.req.json<{
    article: {
      id: string;
      url: string;
      title: string;
      snippet: string;
      source: string;
      category: NewsCategory;
    };
    duration?: number;
  }>();

  if (!body.article) {
    return c.json({ error: 'article is required' }, 400);
  }

  const service = getNewsService(c.env.SEARCH_KV);
  await service.recordRead(
    userId,
    {
      ...body.article,
      sourceUrl: '',
      publishedAt: new Date().toISOString(),
      engines: [],
      score: 1,
    },
    body.duration
  );

  return c.json({ success: true });
});

export default newsRoutes;
