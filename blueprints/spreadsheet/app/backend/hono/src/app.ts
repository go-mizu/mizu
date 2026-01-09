import { Hono } from 'hono';
import { logger } from 'hono/logger';
import { prettyJSON } from 'hono/pretty-json';
import { secureHeaders } from 'hono/secure-headers';
import type { Env, Variables } from './types/index.js';
import type { Database } from './db/types.js';
import { createDatabase } from './db/index.js';
import { corsMiddleware } from './middleware/cors.js';
import { errorHandler } from './middleware/error.js';
import { auth } from './routes/auth.js';
import { workbooks } from './routes/workbooks.js';
import { sheets } from './routes/sheets.js';
import { cells } from './routes/cells.js';
import { charts, sheetCharts } from './routes/charts.js';

/**
 * Create the Hono application
 */
export function createApp() {
  const app = new Hono<{
    Bindings: Env;
    Variables: Variables & { db: Database };
  }>();

  // Global middleware
  app.use('*', logger());
  app.use('*', prettyJSON());
  app.use('*', secureHeaders());
  app.use('*', corsMiddleware);

  // Error handler
  app.onError(errorHandler);

  // Database middleware - inject db into context
  app.use('*', async (c, next) => {
    const db = createDatabase(c.env);
    c.set('db', db);
    await next();
  });

  // Health check
  app.get('/health', (c) => {
    return c.json({
      status: 'ok',
      timestamp: new Date().toISOString(),
    });
  });

  // API v1 routes
  const api = new Hono<{
    Bindings: Env;
    Variables: Variables & { db: Database };
  }>();

  // Mount route modules
  api.route('/auth', auth);
  api.route('/workbooks', workbooks);
  api.route('/sheets', sheets);

  // Cell routes are mounted under /sheets/:sheetId
  api.route('/sheets', cells);

  // Chart routes
  api.route('/charts', charts);
  api.route('/sheets', sheetCharts);

  // Mount API under /api/v1
  app.route('/api/v1', api);

  // Catch-all for API 404s
  app.all('/api/*', (c) => {
    return c.json(
      {
        error: 'Not Found',
        message: 'The requested endpoint does not exist',
        status: 404,
      },
      404
    );
  });

  return app;
}

export type App = ReturnType<typeof createApp>;
