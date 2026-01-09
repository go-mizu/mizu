import type { Database } from './types.js';
import type { Env } from '../types/index.js';
import { SqliteDatabase, D1Adapter } from './sqlite.js';
import { PostgresDatabase } from './postgres.js';

export type { Database } from './types.js';
export { SqliteDatabase, D1Adapter } from './sqlite.js';
export { PostgresDatabase } from './postgres.js';

/**
 * Create database instance based on environment
 *
 * Priority:
 * 1. D1 binding (Cloudflare Workers)
 * 2. DATABASE_URL (PostgreSQL - Vercel/Node)
 * 3. In-memory SQLite (testing/development)
 */
export function createDatabase(env: Env): Database {
  // Cloudflare D1
  if (env.DB) {
    return new SqliteDatabase(new D1Adapter(env.DB));
  }

  // PostgreSQL via connection string
  if (env.DATABASE_URL) {
    return new PostgresDatabase(env.DATABASE_URL);
  }

  throw new Error('No database configuration found. Set DB (D1) or DATABASE_URL (PostgreSQL).');
}

/**
 * Create in-memory SQLite database for testing
 * Requires better-sqlite3 (Node.js only)
 */
export async function createTestDatabase(): Promise<Database> {
  // Dynamic import to avoid bundling better-sqlite3 in Workers
  const BetterSqlite3 = await import('better-sqlite3');
  const db = new BetterSqlite3.default(':memory:');

  // Create adapter for better-sqlite3
  const adapter = {
    async run(sql: string, params: unknown[] = []): Promise<void> {
      db.prepare(sql).run(...params);
    },
    async get<T>(sql: string, params: unknown[] = []): Promise<T | null> {
      return db.prepare(sql).get(...params) as T | null;
    },
    async all<T>(sql: string, params: unknown[] = []): Promise<T[]> {
      return db.prepare(sql).all(...params) as T[];
    },
  };

  const database = new SqliteDatabase(adapter);

  // Run migrations
  const { schema } = await import('./schema.js');
  for (const statement of schema.split(';').filter(s => s.trim())) {
    await adapter.run(statement);
  }

  return database;
}

/**
 * Run database migrations
 */
export async function runMigrations(_db: Database, _schema: string): Promise<void> {
  // For D1/SQLite, we need to execute statements one by one
  // For PostgreSQL, the schema can be executed as a whole
  // This is handled by the respective implementations
}
