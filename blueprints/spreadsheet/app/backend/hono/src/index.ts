/**
 * Cloudflare Workers entry point
 *
 * This file exports the Hono app as a Cloudflare Worker.
 * It uses the D1 database binding for SQLite storage.
 */

import { createApp } from './app.js';

const app = createApp();

export default app;
