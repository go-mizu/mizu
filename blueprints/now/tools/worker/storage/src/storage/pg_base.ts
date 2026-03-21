// ── Shared PostgreSQL base for Hyperdrive and Neon drivers ────────────
//
// Both drivers share the same PostgreSQL schema and R2 blob logic.
// Subclasses provide query execution; this class handles SQL and R2.
//
// Schema uses `stg_` prefix (stg_files, stg_events, stg_blobs, stg_tx)
// to avoid conflicts in shared databases.
//
// Unlike the D1 driver (per-actor table sharding), Postgres uses a
// standard `owner` column with indexes — Postgres query planner handles
// the rest.

import type {
  StorageEngine,
  FileEntry,
  FileMeta,
  WriteResult,
  MutationResult,
  DeleteResult,
  ReadResult,
  SearchResult,
  StorageEvent,
  ListOptions,
  LogOptions,
} from "./engine";
import { presignUrl } from "../lib/presign";
import { mimeFromName } from "../lib/mime";

// ── Types ────────────────────────────────────────────────────────────

export type QueryFn = <T extends Record<string, any> = Record<string, any>>(
  text: string,
  params?: any[],
) => Promise<T[]>;

export interface PgBaseConfig {
  bucket: R2Bucket;
  r2Endpoint?: string;
  r2AccessKeyId?: string;
  r2SecretAccessKey?: string;
  r2BucketName?: string;
}

// ── Helpers ──────────────────────────────────────────────────────────

async function sha256(data: ArrayBuffer): Promise<string> {
  const hash = await crypto.subtle.digest("SHA-256", data);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}

function blobKey(actor: string, addr: string): string {
  return `blobs/${actor}/${addr.slice(0, 2)}/${addr.slice(2, 4)}/${addr}`;
}

async function streamToBuffer(stream: ReadableStream): Promise<ArrayBuffer> {
  const reader = stream.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    total += value.byteLength;
  }
  const result = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    result.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return result.buffer as ArrayBuffer;
}

// ── Schema DDL ──────────────────────────────────────────────────────

const SCHEMA_STMTS = [
  `CREATE TABLE IF NOT EXISTS stg_files (
    owner TEXT NOT NULL,
    path TEXT NOT NULL,
    name TEXT NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    type TEXT NOT NULL DEFAULT 'application/octet-stream',
    addr TEXT,
    tx INTEGER,
    tx_time BIGINT,
    updated_at BIGINT NOT NULL,
    PRIMARY KEY (owner, path)
  )`,
  `CREATE INDEX IF NOT EXISTS idx_stg_files_name ON stg_files(owner, lower(name))`,
  `CREATE INDEX IF NOT EXISTS idx_stg_files_updated ON stg_files(owner, updated_at DESC)`,
  `CREATE TABLE IF NOT EXISTS stg_events (
    id BIGSERIAL PRIMARY KEY,
    tx INTEGER NOT NULL,
    actor TEXT NOT NULL,
    action TEXT NOT NULL CHECK(action IN ('write','move','delete')),
    path TEXT NOT NULL,
    addr TEXT,
    size BIGINT NOT NULL DEFAULT 0,
    type TEXT,
    meta TEXT,
    msg TEXT,
    ts BIGINT NOT NULL
  )`,
  `CREATE INDEX IF NOT EXISTS idx_stg_events_actor_tx ON stg_events(actor, tx DESC)`,
  `CREATE INDEX IF NOT EXISTS idx_stg_events_path ON stg_events(actor, path, tx DESC)`,
  `CREATE TABLE IF NOT EXISTS stg_blobs (
    addr TEXT NOT NULL,
    actor TEXT NOT NULL,
    size BIGINT NOT NULL,
    ref_count INTEGER NOT NULL DEFAULT 1,
    created_at BIGINT NOT NULL,
    PRIMARY KEY (addr, actor)
  )`,
  `CREATE TABLE IF NOT EXISTS stg_tx (
    actor TEXT PRIMARY KEY,
    next_tx INTEGER NOT NULL DEFAULT 1
  )`,
];

/** Module-level flag — schema only checked once per isolate. */
let schemaReady = false;

/** Reset schema-ready flag (for tests). */
export function resetSchemaFlag(): void {
  schemaReady = false;
}

// ── Abstract base class ─────────────────────────────────────────────

export abstract class PgEngineBase implements StorageEngine {
  protected bucket: R2Bucket;
  protected r2Endpoint?: string;
  protected r2AccessKeyId?: string;
  protected r2SecretAccessKey?: string;
  protected r2BucketName: string;

  constructor(config: PgBaseConfig) {
    this.bucket = config.bucket;
    this.r2Endpoint = config.r2Endpoint;
    this.r2AccessKeyId = config.r2AccessKeyId;
    this.r2SecretAccessKey = config.r2SecretAccessKey;
    this.r2BucketName = config.r2BucketName || "storage-files";
  }

  /** Execute a read query. Returns rows. */
  protected abstract query<T extends Record<string, any>>(text: string, params?: any[]): Promise<T[]>;

  /** Execute writes in a transaction. Callback receives a query fn bound to the tx. */
  protected abstract transaction<R>(fn: (q: QueryFn) => Promise<R>): Promise<R>;

  /** Benchmark-only: run write SQL transaction without R2 blob ops. */
  async benchWriteMeta(actor: string, path: string, size: number, contentType: string): Promise<void> {
    await this.ensureSchema();
    const addr = "bench_" + Date.now().toString(16);
    const now = Date.now();
    const name = path.split("/").pop() || path;

    await this.transaction(async (q) => {
      await q("SELECT addr FROM stg_files WHERE owner = $1 AND path = $2", [actor, path]);

      const txRows = await q<{ next_tx: number }>(
        `INSERT INTO stg_tx (actor, next_tx) VALUES ($1, 1)
         ON CONFLICT (actor) DO UPDATE SET next_tx = stg_tx.next_tx + 1
         RETURNING next_tx`,
        [actor],
      );
      const tx = txRows[0].next_tx;

      await q(
        `INSERT INTO stg_events (tx, actor, action, path, addr, size, type, msg, ts)
         VALUES ($1, $2, 'write', $3, $4, $5, $6, $7, $8)`,
        [tx, actor, path, addr, size, contentType, `bench meta ${path}`, now],
      );

      await q(
        `INSERT INTO stg_files (owner, path, name, size, type, addr, tx, tx_time, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
         ON CONFLICT (owner, path) DO UPDATE SET
           name=EXCLUDED.name, size=EXCLUDED.size, type=EXCLUDED.type,
           addr=EXCLUDED.addr, tx=EXCLUDED.tx, tx_time=EXCLUDED.tx_time,
           updated_at=EXCLUDED.updated_at`,
        [actor, path, name, size, contentType, addr, tx, now],
      );

      await q(
        `INSERT INTO stg_blobs (addr, actor, size, ref_count, created_at)
         VALUES ($1, $2, $3, 1, $4)
         ON CONFLICT (addr, actor) DO UPDATE SET ref_count = stg_blobs.ref_count + 1`,
        [addr, actor, size, now],
      );
    });
  }

  /** Ensure PostgreSQL schema exists (idempotent). */
  async ensureSchema(): Promise<void> {
    if (schemaReady) return;
    for (const stmt of SCHEMA_STMTS) {
      await this.query(stmt);
    }
    schemaReady = true;
  }

  // ── presign helpers ──────────────────────────────────────────────

  protected get presignConfigured(): boolean {
    return !!(this.r2Endpoint && this.r2AccessKeyId && this.r2SecretAccessKey);
  }

  protected async presign(
    method: "GET" | "PUT" | "POST" | "HEAD" | "DELETE",
    key: string,
    expiresIn: number,
    opts?: { contentType?: string; queryParams?: Record<string, string> },
  ): Promise<string> {
    if (!this.presignConfigured) throw new Error("Presigned URLs not configured");
    return presignUrl({
      method,
      key,
      bucket: this.r2BucketName,
      endpoint: this.r2Endpoint!,
      accessKeyId: this.r2AccessKeyId!,
      secretAccessKey: this.r2SecretAccessKey!,
      expiresIn,
      contentType: opts?.contentType,
      queryParams: opts?.queryParams,
    });
  }

  // ── write ─────────────────────────────────────────────────────────

  async write(
    actor: string,
    path: string,
    body: ArrayBuffer | ReadableStream,
    contentType: string,
    msg?: string,
  ): Promise<WriteResult> {
    await this.ensureSchema();
    const buf = body instanceof ArrayBuffer ? body : await streamToBuffer(body);
    const addr = await sha256(buf);
    const size = buf.byteLength;
    const now = Date.now();
    const name = path.split("/").pop() || path;

    // R2 dedup
    const key = blobKey(actor, addr);
    const existing = await this.bucket.head(key);
    if (!existing) {
      await this.bucket.put(key, buf, { httpMetadata: { contentType } });
    }

    return this.transaction(async (q) => {
      const oldRows = await q<{ addr: string | null }>(
        "SELECT addr FROM stg_files WHERE owner = $1 AND path = $2",
        [actor, path],
      );
      const oldAddr = oldRows[0]?.addr ?? null;

      // Atomic tx counter (upsert + return)
      const txRows = await q<{ next_tx: number }>(
        `INSERT INTO stg_tx (actor, next_tx) VALUES ($1, 1)
         ON CONFLICT (actor) DO UPDATE SET next_tx = stg_tx.next_tx + 1
         RETURNING next_tx`,
        [actor],
      );
      const tx = txRows[0].next_tx;

      await q(
        `INSERT INTO stg_events (tx, actor, action, path, addr, size, type, msg, ts)
         VALUES ($1, $2, 'write', $3, $4, $5, $6, $7, $8)`,
        [tx, actor, path, addr, size, contentType, msg || `write ${path}`, now],
      );

      await q(
        `INSERT INTO stg_files (owner, path, name, size, type, addr, tx, tx_time, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
         ON CONFLICT (owner, path) DO UPDATE SET
           name=EXCLUDED.name, size=EXCLUDED.size, type=EXCLUDED.type,
           addr=EXCLUDED.addr, tx=EXCLUDED.tx, tx_time=EXCLUDED.tx_time,
           updated_at=EXCLUDED.updated_at`,
        [actor, path, name, size, contentType, addr, tx, now],
      );

      await q(
        `INSERT INTO stg_blobs (addr, actor, size, ref_count, created_at)
         VALUES ($1, $2, $3, 1, $4)
         ON CONFLICT (addr, actor) DO UPDATE SET ref_count = stg_blobs.ref_count + 1`,
        [addr, actor, size, now],
      );

      if (oldAddr && oldAddr !== addr) {
        await q(
          "UPDATE stg_blobs SET ref_count = GREATEST(ref_count - 1, 0) WHERE addr = $1 AND actor = $2",
          [oldAddr, actor],
        );
      }
      if (oldAddr && oldAddr === addr) {
        await q(
          "UPDATE stg_blobs SET ref_count = GREATEST(ref_count - 1, 0) WHERE addr = $1 AND actor = $2",
          [addr, actor],
        );
      }

      return { tx, time: now, size };
    });
  }

  // ── move ─────────────────────────────────────────────────────────

  async move(
    actor: string,
    from: string,
    to: string,
    msg?: string,
  ): Promise<MutationResult> {
    await this.ensureSchema();

    return this.transaction(async (q) => {
      const rows = await q<{ addr: string | null; size: number; type: string }>(
        "SELECT addr, size, type FROM stg_files WHERE owner = $1 AND path = $2",
        [actor, from],
      );
      if (!rows.length) throw new Error("Source not found: " + from);
      const file = rows[0];

      const now = Date.now();
      const newName = to.split("/").pop() || to;
      const meta = JSON.stringify({ from });

      const txRows = await q<{ next_tx: number }>(
        `INSERT INTO stg_tx (actor, next_tx) VALUES ($1, 1)
         ON CONFLICT (actor) DO UPDATE SET next_tx = stg_tx.next_tx + 1
         RETURNING next_tx`,
        [actor],
      );
      const tx = txRows[0].next_tx;

      await q(
        `INSERT INTO stg_events (tx, actor, action, path, addr, size, type, meta, msg, ts)
         VALUES ($1, $2, 'move', $3, $4, $5, $6, $7, $8, $9)`,
        [tx, actor, to, file.addr, file.size, file.type, meta, msg || `move ${from} → ${to}`, now],
      );

      await q("DELETE FROM stg_files WHERE owner = $1 AND path = $2", [actor, from]);

      await q(
        `INSERT INTO stg_files (owner, path, name, size, type, addr, tx, tx_time, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
         ON CONFLICT (owner, path) DO UPDATE SET
           name=EXCLUDED.name, size=EXCLUDED.size, type=EXCLUDED.type,
           addr=EXCLUDED.addr, tx=EXCLUDED.tx, tx_time=EXCLUDED.tx_time,
           updated_at=EXCLUDED.updated_at`,
        [actor, to, newName, file.size, file.type, file.addr, tx, now],
      );

      return { tx, time: now };
    });
  }

  // ── delete ───────────────────────────────────────────────────────

  async delete(
    actor: string,
    paths: string[],
    msg?: string,
  ): Promise<DeleteResult> {
    await this.ensureSchema();

    return this.transaction(async (q) => {
      const now = Date.now();
      let deleted = 0;

      const txRows = await q<{ next_tx: number }>(
        `INSERT INTO stg_tx (actor, next_tx) VALUES ($1, 1)
         ON CONFLICT (actor) DO UPDATE SET next_tx = stg_tx.next_tx + 1
         RETURNING next_tx`,
        [actor],
      );
      const tx = txRows[0].next_tx;

      for (const path of paths) {
        if (path.endsWith("/")) {
          const rows = await q<{ path: string; addr: string | null }>(
            "SELECT path, addr FROM stg_files WHERE owner = $1 AND path LIKE $2",
            [actor, `${path}%`],
          );
          for (const row of rows) {
            await q(
              `INSERT INTO stg_events (tx, actor, action, path, addr, size, type, msg, ts)
               VALUES ($1, $2, 'delete', $3, NULL, 0, NULL, $4, $5)`,
              [tx, actor, row.path, msg || `delete ${path}*`, now],
            );
            if (row.addr) {
              await q(
                "UPDATE stg_blobs SET ref_count = GREATEST(ref_count - 1, 0) WHERE addr = $1 AND actor = $2",
                [row.addr, actor],
              );
            }
            deleted++;
          }
          await q("DELETE FROM stg_files WHERE owner = $1 AND path LIKE $2", [actor, `${path}%`]);
        } else {
          const rows = await q<{ addr: string | null }>(
            "SELECT addr FROM stg_files WHERE owner = $1 AND path = $2",
            [actor, path],
          );
          await q(
            `INSERT INTO stg_events (tx, actor, action, path, addr, size, type, msg, ts)
             VALUES ($1, $2, 'delete', $3, NULL, 0, NULL, $4, $5)`,
            [tx, actor, path, msg || `delete ${path}`, now],
          );
          if (rows[0]?.addr) {
            await q(
              "UPDATE stg_blobs SET ref_count = GREATEST(ref_count - 1, 0) WHERE addr = $1 AND actor = $2",
              [rows[0].addr, actor],
            );
          }
          await q("DELETE FROM stg_files WHERE owner = $1 AND path = $2", [actor, path]);
          deleted++;
        }
      }

      return { tx, time: now, deleted };
    });
  }

  // ── read ─────────────────────────────────────────────────────────

  async read(actor: string, path: string): Promise<ReadResult | null> {
    await this.ensureSchema();

    const rows = await this.query<{
      path: string; name: string; size: string; type: string;
      addr: string | null; tx: number; tx_time: string;
    }>(
      "SELECT path, name, size, type, addr, tx, tx_time FROM stg_files WHERE owner = $1 AND path = $2",
      [actor, path],
    );
    if (!rows.length) return null;
    const file = rows[0];
    const meta: FileMeta = {
      path: file.path, name: file.name, size: Number(file.size),
      type: file.type, tx: file.tx || 0, tx_time: Number(file.tx_time) || 0,
    };

    if (file.addr) {
      const key = blobKey(actor, file.addr);
      const obj = await this.bucket.get(key);
      if (!obj) {
        const legacy = await this.bucket.get(`${actor}/${path}`);
        if (!legacy) return null;
        return { body: legacy.body, meta };
      }
      return { body: obj.body, meta };
    }

    const obj = await this.bucket.get(`${actor}/${path}`);
    if (!obj) return null;
    return { body: obj.body, meta };
  }

  // ── head ─────────────────────────────────────────────────────────

  async head(actor: string, path: string): Promise<FileMeta | null> {
    await this.ensureSchema();

    const rows = await this.query<{
      path: string; name: string; size: string; type: string; tx: number; tx_time: string;
    }>(
      "SELECT path, name, size, type, tx, tx_time FROM stg_files WHERE owner = $1 AND path = $2",
      [actor, path],
    );
    if (!rows.length) return null;
    const f = rows[0];
    return {
      path: f.path, name: f.name, size: Number(f.size),
      type: f.type, tx: f.tx || 0, tx_time: Number(f.tx_time) || 0,
    };
  }

  // ── list ─────────────────────────────────────────────────────────

  async list(
    actor: string,
    opts?: ListOptions,
  ): Promise<{ entries: FileEntry[]; truncated: boolean }> {
    await this.ensureSchema();
    const prefix = opts?.prefix || "";
    const limit = Math.min(opts?.limit || 200, 1000);
    const offset = opts?.offset || 0;

    const rows = await this.query<{
      path: string; name: string; size: string; type: string;
      updated_at: string; tx: number; tx_time: string;
    }>(
      "SELECT path, name, size, type, updated_at, tx, tx_time FROM stg_files WHERE owner = $1 AND path LIKE $2 ORDER BY path LIMIT $3 OFFSET $4",
      [actor, `${prefix}%`, limit + 1, offset],
    );

    const truncated = rows.length > limit;
    if (truncated) rows.pop();

    const entries: FileEntry[] = [];
    const dirs = new Set<string>();

    for (const row of rows) {
      const relative = row.path.slice(prefix.length);
      const slash = relative.indexOf("/");
      if (slash === -1) {
        entries.push({
          name: relative,
          type: row.type,
          size: Number(row.size),
          updated_at: Number(row.updated_at),
          tx: row.tx,
          tx_time: Number(row.tx_time),
        });
      } else {
        const dir = relative.slice(0, slash + 1);
        if (!dirs.has(dir)) {
          dirs.add(dir);
          entries.push({ name: dir, type: "directory" });
        }
      }
    }

    return { entries, truncated };
  }

  // ── search ───────────────────────────────────────────────────────

  async search(
    actor: string,
    query: string,
    opts?: { limit?: number; prefix?: string },
  ): Promise<SearchResult[]> {
    await this.ensureSchema();
    const limit = Math.min(opts?.limit || 50, 200);
    const pfx = opts?.prefix || "";

    let sql = "SELECT path, name, size, type, tx FROM stg_files WHERE owner = $1 AND (name ILIKE $2 OR path ILIKE $2)";
    const params: any[] = [actor, `%${query}%`];
    let idx = 3;

    if (pfx) {
      sql += ` AND path LIKE $${idx}`;
      params.push(`${pfx}%`);
      idx++;
    }

    sql += ` ORDER BY updated_at DESC LIMIT $${idx}`;
    params.push(limit);

    const rows = await this.query<{
      path: string; name: string; size: string; type: string; tx: number;
    }>(sql, params);

    return rows.map((r) => ({
      path: r.path,
      name: r.name,
      size: Number(r.size),
      type: r.type,
      tx: r.tx || 0,
    }));
  }

  // ── stats ────────────────────────────────────────────────────────

  async stats(actor: string): Promise<{ files: number; bytes: number }> {
    await this.ensureSchema();
    const rows = await this.query<{ files: string; bytes: string }>(
      "SELECT COUNT(*) as files, COALESCE(SUM(size), 0) as bytes FROM stg_files WHERE owner = $1",
      [actor],
    );
    return { files: Number(rows[0]?.files || 0), bytes: Number(rows[0]?.bytes || 0) };
  }

  // ── allNames ─────────────────────────────────────────────────────

  async allNames(actor: string): Promise<{ path: string; name: string }[]> {
    await this.ensureSchema();
    return this.query<{ path: string; name: string }>(
      "SELECT path, name FROM stg_files WHERE owner = $1",
      [actor],
    );
  }

  // ── log ──────────────────────────────────────────────────────────

  async log(actor: string, opts?: LogOptions): Promise<StorageEvent[]> {
    await this.ensureSchema();
    const limit = Math.min(opts?.limit || 50, 500);

    let sql = "SELECT tx, action, path, size, msg, meta, ts FROM stg_events WHERE actor = $1";
    const params: any[] = [actor];
    let idx = 2;

    if (opts?.path) {
      sql += ` AND path = $${idx}`;
      params.push(opts.path);
      idx++;
    }
    if (opts?.since_tx) {
      sql += ` AND tx > $${idx}`;
      params.push(opts.since_tx);
      idx++;
    }

    sql += ` ORDER BY tx DESC, id DESC LIMIT $${idx}`;
    params.push(limit);

    const rows = await this.query<{
      tx: number; action: string; path: string; size: string;
      msg: string | null; meta: string | null; ts: string;
    }>(sql, params);

    return rows.map((r) => ({
      tx: r.tx,
      action: r.action as "write" | "move" | "delete",
      path: r.path,
      size: Number(r.size),
      msg: r.msg,
      ts: Number(r.ts),
      meta: r.meta ? (typeof r.meta === "string" ? JSON.parse(r.meta) : r.meta) : null,
    }));
  }

  // ── presign read ─────────────────────────────────────────────────

  async presignRead(
    actor: string,
    path: string,
    expiresIn = 3600,
  ): Promise<string | null> {
    if (!this.presignConfigured) return null;
    await this.ensureSchema();

    const rows = await this.query<{ addr: string | null }>(
      "SELECT addr FROM stg_files WHERE owner = $1 AND path = $2",
      [actor, path],
    );
    if (!rows.length) return null;

    const key = rows[0].addr ? blobKey(actor, rows[0].addr) : `${actor}/${path}`;
    return this.presign("GET", key, expiresIn);
  }

  // ── presign upload ───────────────────────────────────────────────

  async presignUpload(
    actor: string,
    path: string,
    contentType: string,
    expiresIn = 3600,
  ): Promise<string> {
    await this.ensureSchema();
    const key = `${actor}/${path}`;
    return this.presign("PUT", key, expiresIn, { contentType });
  }

  // ── confirm upload ───────────────────────────────────────────────

  async confirmUpload(
    actor: string,
    path: string,
    msg?: string,
  ): Promise<WriteResult> {
    const legacyKey = `${actor}/${path}`;
    const obj = await this.bucket.get(legacyKey);
    if (!obj) throw new Error("Upload not found in storage");

    const buf = await obj.arrayBuffer();
    const contentType = obj.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);
    const result = await this.write(actor, path, buf, contentType, msg);
    await this.bucket.delete(legacyKey);
    return result;
  }

  // ── multipart ────────────────────────────────────────────────────

  async initiateMultipart(
    actor: string,
    path: string,
    contentType: string,
    partCount: number,
  ): Promise<{ upload_id: string; part_urls: string[]; expires_in: number }> {
    if (!this.presignConfigured) throw new Error("Presigned URLs not configured");
    await this.ensureSchema();

    const key = `${actor}/${path}`;
    const mpu = await this.bucket.createMultipartUpload(key, { httpMetadata: { contentType } });

    const partUrls: string[] = [];
    for (let i = 1; i <= Math.min(partCount, 10000); i++) {
      const url = await this.presign("PUT", key, 86400, {
        queryParams: { partNumber: String(i), uploadId: mpu.uploadId },
      });
      partUrls.push(url);
    }

    return { upload_id: mpu.uploadId, part_urls: partUrls, expires_in: 86400 };
  }

  async completeMultipart(
    actor: string,
    path: string,
    uploadId: string,
    parts: { part_number: number; etag: string }[],
    msg?: string,
  ): Promise<WriteResult> {
    const key = `${actor}/${path}`;
    const mpu = this.bucket.resumeMultipartUpload(key, uploadId);
    await mpu.complete(parts.map((p) => ({ partNumber: p.part_number, etag: p.etag })));

    const obj = await this.bucket.get(key);
    if (!obj) throw new Error("Multipart upload not found after completion");

    const buf = await obj.arrayBuffer();
    const contentType = obj.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);
    const result = await this.write(actor, path, buf, contentType, msg);
    await this.bucket.delete(key);
    return result;
  }

  async abortMultipart(
    actor: string,
    path: string,
    uploadId: string,
  ): Promise<void> {
    const key = `${actor}/${path}`;
    const mpu = this.bucket.resumeMultipartUpload(key, uploadId);
    await mpu.abort();
  }
}
