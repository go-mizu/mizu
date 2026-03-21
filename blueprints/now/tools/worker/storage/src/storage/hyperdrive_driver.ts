// ── Hyperdrive + R2 driver for StorageEngine ──────────────────────────
//
// Uses Cloudflare Hyperdrive (TCP connection pool) to connect to
// PostgreSQL. Hyperdrive provides:
//   - Connection pooling at the edge (sub-ms connection setup)
//   - Query caching for identical reads
//   - Automatic retry on transient errors
//
// Architecture:
//   Worker  ──► Hyperdrive (edge proxy) ──► PostgreSQL (Neon/Supabase/etc)
//      │
//      └──► R2 Bucket (blob storage)
//
// The `postgres` (postgresjs) library handles SQL. Hyperdrive provides
// the connection string that routes through Cloudflare's edge proxy.

import postgres from "postgres";
import { PgEngineBase, type QueryFn, type PgBaseConfig } from "./pg_base";

// ── Config ───────────────────────────────────────────────────────────

interface HyperdriveConfig extends PgBaseConfig {
  /** Connection string from env.HYPERDRIVE.connectionString */
  connectionString: string;
}

// ── Implementation ───────────────────────────────────────────────────

export class HyperdriveEngine extends PgEngineBase {
  private sql: ReturnType<typeof postgres>;

  constructor(config: HyperdriveConfig) {
    super(config);
    this.sql = postgres(config.connectionString, {
      // Hyperdrive handles pooling; we only need one connection per request.
      max: 1,
      // Workers don't persist between requests — no keepalive needed.
      idle_timeout: 0,
      // Disable prepare — Hyperdrive doesn't support named prepared statements.
      prepare: false,
    });
  }

  protected async query<T extends Record<string, any>>(
    text: string,
    params?: any[],
  ): Promise<T[]> {
    const rows = await this.sql.unsafe(text, params || []);
    return rows as unknown as T[];
  }

  protected async transaction<R>(fn: (q: QueryFn) => Promise<R>): Promise<R> {
    return this.sql.begin(async (tx) => {
      const q: QueryFn = async <T extends Record<string, any>>(
        text: string,
        params?: any[],
      ): Promise<T[]> => {
        const rows = await tx.unsafe(text, params || []);
        return rows as unknown as T[];
      };
      return fn(q);
    });
  }
}
