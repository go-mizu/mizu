/**
 * Database factory and driver exports
 *
 * Provides a unified interface for SQLite (D1/better-sqlite3) and PostgreSQL
 */

import type { Database } from './types.js';
import type { Env } from '../types/index.js';
import { SqliteDriver } from './driver/sqlite/index.js';
import { PostgresDriver } from './driver/postgres/index.js';

// Re-export types and drivers
export type { Database } from './types.js';
export { SqliteDriver, D1Adapter } from './driver/sqlite/index.js';
export { PostgresDriver } from './driver/postgres/index.js';

// Re-export tile utilities
export {
  TILE_HEIGHT,
  TILE_WIDTH,
  cellToTile,
  tileCellKey,
  parseTileCellKey,
  tileToCell,
  getTileRange,
  createEmptyTile,
  serializeTile,
  deserializeTile,
  isTileEmpty,
} from './tile.js';
export type { Tile, TileCell, TilePosition } from './tile.js';

/**
 * Create database instance based on environment
 *
 * Priority:
 * 1. D1 binding (Cloudflare Workers)
 * 2. DATABASE_URL (PostgreSQL - Vercel/Node)
 * 3. Throws error if no database configured
 *
 * @param env - Environment bindings
 * @returns Database instance
 */
export function createDatabase(env: Env): Database {
  // Cloudflare D1
  if (env.DB) {
    return SqliteDriver.fromD1(env.DB);
  }

  // PostgreSQL via connection string
  if (env.DATABASE_URL) {
    return new PostgresDriver(env.DATABASE_URL);
  }

  throw new Error('No database configuration found. Set DB (D1) or DATABASE_URL (PostgreSQL).');
}

/**
 * Create in-memory SQLite database for testing
 * Requires better-sqlite3 (Node.js only)
 */
export async function createTestDatabase(): Promise<Database> {
  return SqliteDriver.createInMemory();
}

/**
 * Run database migrations for the given driver
 *
 * @param db - Database instance (must have ensure() method)
 */
export async function runMigrations(db: Database & { ensure?: () => Promise<void> }): Promise<void> {
  if (typeof db.ensure === 'function') {
    await db.ensure();
  }
}
