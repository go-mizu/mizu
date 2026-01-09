/**
 * Vercel Serverless Function entry point
 *
 * This file exports the Hono app as a Vercel serverless function.
 * It uses PostgreSQL for storage via DATABASE_URL environment variable.
 */

import { handle } from 'hono/vercel';
import { createApp } from '../src/app.js';

const app = createApp();

export const GET = handle(app);
export const POST = handle(app);
export const PUT = handle(app);
export const PATCH = handle(app);
export const DELETE = handle(app);
export const OPTIONS = handle(app);
