// ── Neon Serverless + R2 driver for StorageEngine ─────────────────────
//
// Uses @neondatabase/serverless for PostgreSQL access:
//   - HTTP mode (neon()) for reads — zero connection overhead
//   - WebSocket Pool for write transactions — interactive queries
//
// Architecture:
//   Worker  ──HTTP──►  Neon SQL Proxy  ──► PostgreSQL
//           ──WS────►  Neon WS Proxy   ──► PostgreSQL (transactions)
//      │
//      └──► R2 Bucket (blob storage)
//
// No TCP sockets needed — works in Cloudflare Workers natively.

import { neon, Pool } from "@neondatabase/serverless";
import { PgEngineBase, type QueryFn, type PgBaseConfig } from "./pg_base";

// ── Config ───────────────────────────────────────────────────────────

interface NeonConfig extends PgBaseConfig {
  /** Direct Neon PostgreSQL connection string. */
  connectionString: string;
}

// ── Implementation ───────────────────────────────────────────────────

export class NeonEngine extends PgEngineBase {
  /** HTTP-based SQL function — one HTTP request per query, zero connection state. */
  private httpSql: ReturnType<typeof neon>;

  /** WebSocket connection pool — used for interactive transactions. */
  private pool: Pool;

  constructor(config: NeonConfig) {
    super(config);
    this.httpSql = neon(config.connectionString);
    // Strip channel_binding param (not supported over WebSocket)
    const poolDsn = config.connectionString.replace(/([?&])channel_binding=[^&]*/g, "$1").replace(/[?&]$/, "");
    this.pool = new Pool({ connectionString: poolDsn });
  }

  protected async query<T extends Record<string, any>>(
    text: string,
    params?: any[],
  ): Promise<T[]> {
    // neon() HTTP mode: .query() for parameterized queries (one HTTP round trip)
    const rows = await this.httpSql.query(text, params || []);
    return rows as unknown as T[];
  }

  protected async transaction<R>(fn: (q: QueryFn) => Promise<R>): Promise<R> {
    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const q: QueryFn = async <T extends Record<string, any>>(
        text: string,
        params?: any[],
      ): Promise<T[]> => {
        const result = await client.query(text, params || []);
        return result.rows as T[];
      };
      const result = await fn(q);
      await client.query("COMMIT");
      return result;
    } catch (e) {
      await client.query("ROLLBACK");
      throw e;
    } finally {
      client.release();
    }
  }
}
