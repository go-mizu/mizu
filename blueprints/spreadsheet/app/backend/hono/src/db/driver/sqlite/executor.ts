/**
 * SQLite executor interface and adapters
 *
 * Provides a unified interface for D1 (Cloudflare Workers) and better-sqlite3 (Node.js/Bun)
 */

/**
 * Generic SQLite executor interface
 */
export interface SqliteExecutor {
  run(sql: string, params?: unknown[]): Promise<void>;
  get<T>(sql: string, params?: unknown[]): Promise<T | null>;
  all<T>(sql: string, params?: unknown[]): Promise<T[]>;
}

/**
 * D1 adapter - wraps Cloudflare D1 database
 */
export class D1Adapter implements SqliteExecutor {
  constructor(private db: D1Database) {}

  async run(sql: string, params: unknown[] = []): Promise<void> {
    await this.db.prepare(sql).bind(...params).run();
  }

  async get<T>(sql: string, params: unknown[] = []): Promise<T | null> {
    const result = await this.db.prepare(sql).bind(...params).first();
    return result as T | null;
  }

  async all<T>(sql: string, params: unknown[] = []): Promise<T[]> {
    const result = await this.db.prepare(sql).bind(...params).all();
    return result.results as T[];
  }
}

/**
 * Better-sqlite3 adapter type (for Node.js/Bun)
 * The actual adapter is created dynamically to avoid bundling better-sqlite3 in Workers
 */
export interface BetterSqlite3Database {
  prepare(sql: string): {
    run(...params: unknown[]): unknown;
    get(...params: unknown[]): unknown;
    all(...params: unknown[]): unknown[];
  };
}

/**
 * Create a better-sqlite3 adapter
 */
export function createBetterSqliteAdapter(db: BetterSqlite3Database): SqliteExecutor {
  return {
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
}

/**
 * Create an in-memory SQLite database for testing (Node.js only)
 * Requires better-sqlite3 to be installed
 */
export async function createInMemoryExecutor(): Promise<SqliteExecutor> {
  const BetterSqlite3 = await import('better-sqlite3');
  const db = new BetterSqlite3.default(':memory:');
  return createBetterSqliteAdapter(db);
}

/**
 * Create a file-based SQLite database (Node.js only)
 * Requires better-sqlite3 to be installed
 */
export async function createFileExecutor(path: string): Promise<SqliteExecutor> {
  const BetterSqlite3 = await import('better-sqlite3');
  const db = new BetterSqlite3.default(path);
  return createBetterSqliteAdapter(db);
}
