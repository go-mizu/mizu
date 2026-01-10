import { Hono } from 'hono';
import { logger } from 'hono/logger';
import { prettyJSON } from 'hono/pretty-json';
import { secureHeaders } from 'hono/secure-headers';
import BetterSqlite3 from 'better-sqlite3';
import type { Env, Variables } from '../src/types/index.js';
import type { Database } from '../src/db/types.js';
import { SqliteDriver } from '../src/db/driver/sqlite/index.js';
import { sqliteSchema } from '../src/db/schema.js';
import { corsMiddleware } from '../src/middleware/cors.js';
import { errorHandler } from '../src/middleware/error.js';
import { auth } from '../src/routes/auth.js';
import { workbooks } from '../src/routes/workbooks.js';
import { sheets } from '../src/routes/sheets.js';
import { cells } from '../src/routes/cells.js';
import { charts, sheetCharts } from '../src/routes/charts.js';

/**
 * Create a test database with schema applied
 */
export function createTestDb(): { db: Database; rawDb: BetterSqlite3.Database } {
  const rawDb = new BetterSqlite3(':memory:');

  // Enable foreign keys
  rawDb.pragma('foreign_keys = ON');

  // Apply schema - execute the entire script at once
  rawDb.exec(sqliteSchema);

  const db = SqliteDriver.fromBetterSqlite(rawDb);

  return { db, rawDb };
}

/**
 * Create test app with injected database
 */
export function createTestApp(db: Database): Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}> {
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

  // Inject test database and environment
  app.use('*', async (c, next) => {
    c.set('db', db);
    c.env = {
      JWT_SECRET: 'test-secret-key-for-testing-only',
      NODE_ENV: 'test',
    };
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
  api.route('/sheets', cells);
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

/**
 * Helper to make authenticated requests
 */
export async function registerAndLogin(
  app: Hono<any>,
  email: string = 'test@example.com',
  password: string = 'password123'
): Promise<{ token: string; userId: string }> {
  // Register
  const registerRes = await app.request('/api/v1/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email,
      name: 'Test User',
      password,
    }),
  });

  if (registerRes.status !== 201) {
    // Try login if user exists
    const loginRes = await app.request('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    const loginData = await loginRes.json() as { token: string; user: { id: string } };
    return { token: loginData.token, userId: loginData.user.id };
  }

  const data = await registerRes.json() as { token: string; user: { id: string } };
  return { token: data.token, userId: data.user.id };
}

/**
 * Helper to create a workbook with a sheet
 */
export async function createWorkbookWithSheet(
  app: Hono<any>,
  token: string,
  workbookName: string = 'Test Workbook'
): Promise<{ workbookId: string; sheetId: string }> {
  const res = await app.request('/api/v1/workbooks', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({ name: workbookName }),
  });

  const data = await res.json() as { workbook: { id: string } };
  const workbookId = data.workbook.id;

  // Get sheets
  const sheetsRes = await app.request(`/api/v1/workbooks/${workbookId}/sheets`, {
    headers: { 'Authorization': `Bearer ${token}` },
  });

  const sheetsData = await sheetsRes.json() as { sheets: Array<{ id: string }> };
  const sheetId = sheetsData.sheets[0].id;

  return { workbookId, sheetId };
}
