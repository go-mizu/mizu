/**
 * Node.js entry point
 *
 * This file runs the Hono app on Node.js using @hono/node-server.
 * It uses PostgreSQL or SQLite for storage.
 */

import { serve } from '@hono/node-server';
import { createApp } from './app.js';

const app = createApp();

const port = parseInt(process.env.PORT || '3000', 10);

console.log(`Starting server on port ${port}...`);
console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
console.log(`Database: ${process.env.DATABASE_URL ? 'PostgreSQL' : 'SQLite'}`);

serve({
  fetch: app.fetch,
  port,
}, (info) => {
  console.log(`Server running at http://localhost:${info.port}`);
  console.log(`Health check: http://localhost:${info.port}/health`);
  console.log(`API: http://localhost:${info.port}/api/v1`);
});
