/**
 * Main entry point - exports the app for various runtimes
 */

export { createApp } from './app.js';
export type { App } from './app.js';
export { createDatabase, createTestDatabase, runMigrations } from './db/index.js';
export type { Database } from './db/types.js';
