/**
 * Database factory and driver exports
 */

import type { Database } from './types.js';
import type { Env } from '../types/index.js';
import { SqliteDriver } from './sqlite.js';

// Re-export types and drivers
export type { Database } from './types.js';
export { SqliteDriver } from './sqlite.js';

/**
 * Create database instance based on environment
 *
 * @param env - Environment bindings
 * @returns Database instance
 */
export function createDatabase(env: Env): Database {
  // SQLite via file path
  const dbPath = env.DATABASE_PATH || './data/table.db';
  return new SqliteDriver(dbPath);
}

/**
 * Create in-memory SQLite database for testing
 */
export function createTestDatabase(): Database {
  return SqliteDriver.createInMemory();
}

/**
 * Run database migrations for the given driver
 */
export async function runMigrations(db: Database): Promise<void> {
  await db.ensure();
}
