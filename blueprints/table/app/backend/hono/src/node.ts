/**
 * Node.js entry point
 *
 * This file runs the Hono app on Node.js using @hono/node-server.
 * It uses SQLite for storage.
 */

import { serve } from '@hono/node-server';
import { existsSync, mkdirSync } from 'fs';
import { dirname } from 'path';
import { createApp } from './app.js';
import { SqliteDriver } from './db/sqlite.js';

// Ensure data directory exists
const dbPath = process.env.DATABASE_PATH || './data/table.db';
const dbDir = dirname(dbPath);
if (!existsSync(dbDir)) {
  mkdirSync(dbDir, { recursive: true });
}

// Initialize database schema
const db = new SqliteDriver(dbPath);
await db.ensure();
await db.close();

const app = createApp();

const port = parseInt(process.env.PORT || '3000', 10);

console.log(`Starting Table API server on port ${port}...`);
console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
console.log(`Database: ${dbPath}`);

serve({
  fetch: app.fetch,
  port,
}, (info) => {
  console.log(`Server running at http://localhost:${info.port}`);
  console.log(`Health check: http://localhost:${info.port}/health`);
  console.log(`API: http://localhost:${info.port}/api/v1`);
});
