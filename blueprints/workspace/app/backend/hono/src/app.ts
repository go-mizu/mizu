import { Hono } from 'hono';
import { logger } from 'hono/logger';
import { secureHeaders } from 'hono/secure-headers';
import type { Env, Variables } from './env';
import { corsMiddleware } from './middleware/cors';
import { errorHandler } from './middleware/error';
import { createStore, type StoreConfig } from './store/factory';
import { createRoutes } from './routes';

export interface AppConfig {
  storeConfig?: StoreConfig;
}

export function createApp(config?: AppConfig) {
  const app = new Hono<{ Bindings: Env; Variables: Variables }>();

  // Cache store instance for reuse across requests (important for tests)
  let cachedStore: Awaited<ReturnType<typeof createStore>> | null = null;

  // Global middleware
  app.use('*', logger());
  app.use('*', secureHeaders());
  app.use('*', corsMiddleware);

  // Error handler
  app.onError(errorHandler);

  // Initialize store middleware
  app.use('*', async (c, next) => {
    // Use provided config or create from env
    const storeConfig: StoreConfig = config?.storeConfig ?? {
      driver: 'd1',
      d1: c.env.DB,
    };

    // Reuse cached store if available and config was provided (test mode)
    if (cachedStore && config?.storeConfig) {
      c.set('store', cachedStore);
    } else {
      const store = await createStore(storeConfig);
      if (config?.storeConfig) {
        cachedStore = store;
      }
      c.set('store', store);
    }

    await next();
  });

  // Static files - serve from ASSETS binding
  app.use('/static/*', async (c, next) => {
    // Use the ASSETS binding to fetch static files
    const assets = c.env.ASSETS;
    if (assets) {
      try {
        const url = new URL(c.req.url);
        // ASSETS binding expects paths without /static prefix
        url.pathname = url.pathname.replace('/static', '');
        const assetRequest = new Request(url.toString(), c.req.raw);
        const response = await assets.fetch(assetRequest);
        if (response.ok) {
          return response;
        }
      } catch {
        // Fall through to next handler
      }
    }
    await next();
  });

  // API and UI routes
  app.route('/', createRoutes());

  // 404 handler
  app.notFound((c) => {
    return c.json({ error: 'Not found' }, 404);
  });

  return app;
}
