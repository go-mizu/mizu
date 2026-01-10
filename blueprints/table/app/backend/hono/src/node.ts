/**
 * Node.js entry point
 *
 * This file runs the Hono app on Node.js using @hono/node-server.
 * It uses SQLite for storage.
 */

import { serve } from '@hono/node-server';
import { serveStatic } from '@hono/node-server/serve-static';
import { existsSync, mkdirSync, readFileSync } from 'fs';
import { dirname, join, resolve } from 'path';
import { createApp } from './app.js';
import { SqliteDriver } from './db/sqlite.js';

// Ensure data directory exists
const dbPath = process.env.DATABASE_PATH || process.env.DB_PATH || './data/table.db';
const dbDir = dirname(dbPath);
if (!existsSync(dbDir)) {
  mkdirSync(dbDir, { recursive: true });
}

// Initialize database schema
const db = new SqliteDriver(dbPath);
await db.ensure();
await db.close();

const app = createApp();

// Find static directory (assets/static/dist)
const findStaticDir = () => {
  // Try various paths relative to the backend
  const candidates = [
    '../../../assets/static',
    '../../assets/static',
    './assets/static',
    resolve(process.cwd(), 'assets/static'),
  ];

  for (const candidate of candidates) {
    const distPath = join(candidate, 'dist', 'js', 'main.js');
    if (existsSync(distPath)) {
      return candidate;
    }
  }
  return null;
};

const staticDir = findStaticDir();

// Serve static files in production
if (staticDir) {
  console.log(`Serving static files from: ${staticDir}`);

  // Serve static assets
  app.use('/static/*', serveStatic({
    root: staticDir,
    rewriteRequestPath: (path) => path.replace('/static', ''),
  }));

  // Serve index.html for SPA routes
  app.get('/', (c) => {
    try {
      const indexPath = join(staticDir, 'dist', 'index.html');
      if (existsSync(indexPath)) {
        const html = readFileSync(indexPath, 'utf-8');
        return c.html(html);
      }
    } catch {
      // Fall through
    }
    // Redirect to app.html
    return c.redirect('/static/dist/index.html');
  });

  // Serve app for all non-API routes (SPA fallback)
  app.get('*', (c) => {
    // Skip API and static routes
    if (c.req.path.startsWith('/api') || c.req.path.startsWith('/static') || c.req.path.startsWith('/health')) {
      return c.notFound();
    }

    try {
      const indexPath = join(staticDir, 'dist', 'index.html');
      if (existsSync(indexPath)) {
        const html = readFileSync(indexPath, 'utf-8');
        return c.html(html);
      }
    } catch {
      // Fall through
    }
    return c.redirect('/static/dist/index.html');
  });
}

const port = parseInt(process.env.PORT || '3000', 10);

console.log(`Starting Table API server on port ${port}...`);
console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
console.log(`Database: ${dbPath}`);
if (!staticDir) {
  console.log(`Static files: not found (dev mode - use Vite)`);
}

serve({
  fetch: app.fetch,
  port,
}, (info) => {
  console.log(`Server running at http://localhost:${info.port}`);
  console.log(`Health check: http://localhost:${info.port}/health`);
  console.log(`API: http://localhost:${info.port}/api/v1`);
});
