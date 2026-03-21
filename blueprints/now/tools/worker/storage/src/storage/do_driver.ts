// ── Durable Object SQLite + R2 driver for StorageEngine ──────────────
//
// Each actor gets a dedicated Durable Object with its own SQLite database.
// Zero table sharding — the DO IS the actor's namespace.
//
// SQL is synchronous (local SQLite), transactions via transactionSync().
// R2 is handled by the DOEngine adapter (not inside the DO) to avoid
// routing blobs through the DO's stub.
//
// Architecture:
//   DOEngine (Worker) ─── RPC ───► StorageDO (Durable Object)
//      │                                │
//      │ R2 blob ops                    │ SQLite metadata ops
//      ▼                                ▼
//   R2 Bucket                    DO-local SQLite

import { DurableObject } from "cloudflare:workers";
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

// ── Helpers (shared with d1_driver) ──────────────────────────────────

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

// ── Durable Object class ─────────────────────────────────────────────
//
// Each instance serves exactly one actor. SQLite is local to the DO.
// Methods are called via RPC from DOEngine.

interface DOEnv {
  BUCKET: R2Bucket;
  [key: string]: unknown;
}

export class StorageDO extends DurableObject<DOEnv> {
  private schemaReady = false;

  private ensureSchema(): void {
    if (this.schemaReady) return;
    const sql = this.ctx.storage.sql;
    sql.exec(`CREATE TABLE IF NOT EXISTS files (
      path TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      size INTEGER NOT NULL DEFAULT 0,
      type TEXT NOT NULL DEFAULT 'application/octet-stream',
      addr TEXT,
      tx INTEGER,
      tx_time INTEGER,
      updated_at INTEGER NOT NULL
    )`);
    sql.exec(`CREATE TABLE IF NOT EXISTS events (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      tx INTEGER NOT NULL,
      action TEXT NOT NULL CHECK(action IN ('write','move','delete')),
      path TEXT NOT NULL,
      addr TEXT,
      size INTEGER NOT NULL DEFAULT 0,
      type TEXT,
      meta TEXT,
      msg TEXT,
      ts INTEGER NOT NULL
    )`);
    sql.exec(`CREATE TABLE IF NOT EXISTS blobs (
      addr TEXT PRIMARY KEY,
      size INTEGER NOT NULL,
      ref_count INTEGER NOT NULL DEFAULT 1,
      created_at INTEGER NOT NULL
    )`);
    sql.exec(`CREATE TABLE IF NOT EXISTS meta (
      key TEXT PRIMARY KEY,
      value TEXT NOT NULL
    )`);
    sql.exec(`INSERT OR IGNORE INTO meta (key, value) VALUES ('next_tx', '0')`);
    sql.exec(`CREATE INDEX IF NOT EXISTS idx_events_tx ON events(tx)`);
    sql.exec(`CREATE INDEX IF NOT EXISTS idx_events_path ON events(path, tx)`);
    sql.exec(`CREATE INDEX IF NOT EXISTS idx_files_name ON files(name COLLATE NOCASE)`);
    this.schemaReady = true;
  }

  private nextTx(): number {
    const sql = this.ctx.storage.sql;
    sql.exec("UPDATE meta SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'next_tx'");
    const row = [...sql.exec<{ value: string }>("SELECT value FROM meta WHERE key = 'next_tx'")][0];
    return parseInt(row.value, 10);
  }

  // ── RPC methods called from DOEngine ────────────────────────────────

  /** Record a file write in SQLite (blob already in R2). */
  async recordWrite(
    path: string,
    addr: string,
    size: number,
    contentType: string,
    msg?: string,
  ): Promise<WriteResult> {
    this.ensureSchema();
    const sql = this.ctx.storage.sql;
    const now = Date.now();
    const name = path.split("/").pop() || path;

    const oldRows = [...sql.exec<{ addr: string | null }>("SELECT addr FROM files WHERE path = ?", path)];
    const oldAddr = oldRows[0]?.addr ?? null;

    let tx: number;
    this.ctx.storage.transactionSync(() => {
      tx = this.nextTx();

      sql.exec(
        "INSERT INTO events (tx, action, path, addr, size, type, msg, ts) VALUES (?, 'write', ?, ?, ?, ?, ?, ?)",
        tx, path, addr, size, contentType, msg || `write ${path}`, now,
      );

      sql.exec(
        `INSERT INTO files (path, name, size, type, addr, tx, tx_time, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT (path) DO UPDATE SET name=excluded.name, size=excluded.size, type=excluded.type, addr=excluded.addr, tx=excluded.tx, tx_time=excluded.tx_time, updated_at=excluded.updated_at`,
        path, name, size, contentType, addr, tx!, now, now,
      );

      sql.exec(
        `INSERT INTO blobs (addr, size, ref_count, created_at) VALUES (?, ?, 1, ?)
         ON CONFLICT (addr) DO UPDATE SET ref_count = ref_count + 1`,
        addr, size, now,
      );

      if (oldAddr && oldAddr !== addr) {
        sql.exec("UPDATE blobs SET ref_count = MAX(ref_count - 1, 0) WHERE addr = ?", oldAddr);
      }
      if (oldAddr && oldAddr === addr) {
        sql.exec("UPDATE blobs SET ref_count = MAX(ref_count - 1, 0) WHERE addr = ?", addr);
      }
    });

    return { tx: tx!, time: now, size };
  }

  /** Record a file move in SQLite (no blob operations). */
  async recordMove(from: string, to: string, msg?: string): Promise<MutationResult & { addr: string | null; size: number; type: string }> {
    this.ensureSchema();
    const sql = this.ctx.storage.sql;

    const rows = [...sql.exec<{ addr: string | null; size: number; type: string }>(
      "SELECT addr, size, type FROM files WHERE path = ?", from,
    )];
    if (!rows.length) throw new Error("Source not found: " + from);
    const file = rows[0];

    const now = Date.now();
    const newName = to.split("/").pop() || to;
    const meta = JSON.stringify({ from });

    let tx: number;
    this.ctx.storage.transactionSync(() => {
      tx = this.nextTx();

      sql.exec(
        "INSERT INTO events (tx, action, path, addr, size, type, meta, msg, ts) VALUES (?, 'move', ?, ?, ?, ?, ?, ?, ?)",
        tx, to, file.addr, file.size, file.type, meta, msg || `move ${from} → ${to}`, now,
      );
      sql.exec("DELETE FROM files WHERE path = ?", from);
      sql.exec(
        `INSERT INTO files (path, name, size, type, addr, tx, tx_time, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT (path) DO UPDATE SET name=excluded.name, size=excluded.size, type=excluded.type, addr=excluded.addr, tx=excluded.tx, tx_time=excluded.tx_time, updated_at=excluded.updated_at`,
        to, newName, file.size, file.type, file.addr, tx!, now, now,
      );
    });

    return { tx: tx!, time: now, addr: file.addr, size: file.size, type: file.type };
  }

  /** Record file deletion(s) in SQLite. Returns paths + addrs for R2 GC. */
  async recordDelete(
    paths: string[],
    msg?: string,
  ): Promise<DeleteResult> {
    this.ensureSchema();
    const sql = this.ctx.storage.sql;
    const now = Date.now();
    let deleted = 0;

    let tx: number;
    this.ctx.storage.transactionSync(() => {
      tx = this.nextTx();

      for (const path of paths) {
        if (path.endsWith("/")) {
          const rows = [...sql.exec<{ path: string; addr: string | null }>(
            "SELECT path, addr FROM files WHERE path LIKE ?", `${path}%`,
          )];
          for (const row of rows) {
            sql.exec(
              "INSERT INTO events (tx, action, path, addr, size, type, msg, ts) VALUES (?, 'delete', ?, NULL, 0, NULL, ?, ?)",
              tx, row.path, msg || `delete ${path}*`, now,
            );
            if (row.addr) {
              sql.exec("UPDATE blobs SET ref_count = MAX(ref_count - 1, 0) WHERE addr = ?", row.addr);
            }
            deleted++;
          }
          sql.exec("DELETE FROM files WHERE path LIKE ?", `${path}%`);
        } else {
          const rows = [...sql.exec<{ addr: string | null }>("SELECT addr FROM files WHERE path = ?", path)];
          sql.exec(
            "INSERT INTO events (tx, action, path, addr, size, type, msg, ts) VALUES (?, 'delete', ?, NULL, 0, NULL, ?, ?)",
            tx, path, msg || `delete ${path}`, now,
          );
          if (rows[0]?.addr) {
            sql.exec("UPDATE blobs SET ref_count = MAX(ref_count - 1, 0) WHERE addr = ?", rows[0].addr);
          }
          sql.exec("DELETE FROM files WHERE path = ?", path);
          deleted++;
        }
      }
    });

    return { tx: tx!, time: now, deleted };
  }

  /** Get file metadata (no body). */
  async getFileMeta(path: string): Promise<FileMeta | null> {
    this.ensureSchema();
    const rows = [...this.ctx.storage.sql.exec<{
      path: string; name: string; size: number; type: string; addr: string | null; tx: number; tx_time: number;
    }>("SELECT path, name, size, type, addr, tx, tx_time FROM files WHERE path = ?", path)];

    if (!rows.length) return null;
    const f = rows[0];
    return { path: f.path, name: f.name, size: f.size, type: f.type, tx: f.tx || 0, tx_time: f.tx_time || 0 };
  }

  /** Get file addr for R2 lookup. */
  async getFileAddr(path: string): Promise<{ addr: string | null; meta: FileMeta } | null> {
    this.ensureSchema();
    const rows = [...this.ctx.storage.sql.exec<{
      path: string; name: string; size: number; type: string; addr: string | null; tx: number; tx_time: number;
    }>("SELECT path, name, size, type, addr, tx, tx_time FROM files WHERE path = ?", path)];

    if (!rows.length) return null;
    const f = rows[0];
    return {
      addr: f.addr,
      meta: { path: f.path, name: f.name, size: f.size, type: f.type, tx: f.tx || 0, tx_time: f.tx_time || 0 },
    };
  }

  /** List files under a prefix. */
  async listFiles(opts?: ListOptions): Promise<{ entries: FileEntry[]; truncated: boolean }> {
    this.ensureSchema();
    const prefix = opts?.prefix || "";
    const limit = Math.min(opts?.limit || 200, 1000);
    const offset = opts?.offset || 0;

    const rows = [...this.ctx.storage.sql.exec<{
      path: string; name: string; size: number; type: string; updated_at: number; tx: number; tx_time: number;
    }>(
      "SELECT path, name, size, type, updated_at, tx, tx_time FROM files WHERE path LIKE ? ORDER BY path LIMIT ? OFFSET ?",
      `${prefix}%`, limit + 1, offset,
    )];

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
          size: row.size,
          updated_at: row.updated_at,
          tx: row.tx,
          tx_time: row.tx_time,
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

  /** Search files by name/path. */
  async searchFiles(
    query: string,
    opts?: { limit?: number; prefix?: string },
  ): Promise<SearchResult[]> {
    this.ensureSchema();
    const limit = Math.min(opts?.limit || 50, 200);
    const pfx = opts?.prefix || "";

    let q = "SELECT path, name, size, type, tx FROM files WHERE (name LIKE ? OR path LIKE ?)";
    const binds: any[] = [`%${query}%`, `%${query}%`];

    if (pfx) {
      q += " AND path LIKE ?";
      binds.push(`${pfx}%`);
    }

    q += " ORDER BY updated_at DESC LIMIT ?";
    binds.push(limit);

    const rows = [...this.ctx.storage.sql.exec<{
      path: string; name: string; size: number; type: string; tx: number;
    }>(q, ...binds)];

    return rows.map((r) => ({
      path: r.path,
      name: r.name,
      size: r.size,
      type: r.type,
      tx: r.tx || 0,
    }));
  }

  /** Get storage stats. */
  async getStats(): Promise<{ files: number; bytes: number }> {
    this.ensureSchema();
    const rows = [...this.ctx.storage.sql.exec<{ files: number; bytes: number }>(
      "SELECT COUNT(*) as files, COALESCE(SUM(size), 0) as bytes FROM files",
    )];
    return { files: rows[0]?.files || 0, bytes: rows[0]?.bytes || 0 };
  }

  /** Get all file path/name pairs. */
  async getAllNames(): Promise<{ path: string; name: string }[]> {
    this.ensureSchema();
    return [...this.ctx.storage.sql.exec<{ path: string; name: string }>(
      "SELECT path, name FROM files",
    )];
  }

  /** Get event log. */
  async getLog(opts?: LogOptions): Promise<StorageEvent[]> {
    this.ensureSchema();
    const limit = Math.min(opts?.limit || 50, 500);

    let q = "SELECT tx, action, path, size, msg, meta, ts FROM events WHERE 1=1";
    const binds: any[] = [];

    if (opts?.path) {
      q += " AND path = ?";
      binds.push(opts.path);
    }
    if (opts?.since_tx) {
      q += " AND tx > ?";
      binds.push(opts.since_tx);
    }
    if (opts?.before_tx) {
      q += " AND tx < ?";
      binds.push(opts.before_tx);
    }

    q += " ORDER BY tx DESC, id DESC LIMIT ?";
    binds.push(limit);

    const rows = [...this.ctx.storage.sql.exec<{
      tx: number; action: string; path: string; size: number; msg: string | null; meta: string | null; ts: number;
    }>(q, ...binds)];

    return rows.map((r) => ({
      tx: r.tx,
      action: r.action as "write" | "move" | "delete",
      path: r.path,
      size: r.size,
      msg: r.msg,
      ts: r.ts,
      meta: r.meta ? JSON.parse(r.meta) : null,
    }));
  }
}

// ── DOEngine adapter (implements StorageEngine) ──────────────────────
//
// Sits in the Worker. Routes metadata ops to the actor's DO via RPC.
// Handles R2 blob operations directly (blobs never pass through the DO).

interface DOConfig {
  ns: DurableObjectNamespace<StorageDO>;
  bucket: R2Bucket;
  r2Endpoint?: string;
  r2AccessKeyId?: string;
  r2SecretAccessKey?: string;
  r2BucketName?: string;
}

export class DOEngine implements StorageEngine {
  private ns: DurableObjectNamespace<StorageDO>;
  private bucket: R2Bucket;
  private r2Endpoint?: string;
  private r2AccessKeyId?: string;
  private r2SecretAccessKey?: string;
  private r2BucketName: string;

  constructor(config: DOConfig) {
    this.ns = config.ns;
    this.bucket = config.bucket;
    this.r2Endpoint = config.r2Endpoint;
    this.r2AccessKeyId = config.r2AccessKeyId;
    this.r2SecretAccessKey = config.r2SecretAccessKey;
    this.r2BucketName = config.r2BucketName || "storage-files";
  }

  private stub(actor: string): DurableObjectStub<StorageDO> {
    return this.ns.get(this.ns.idFromName(actor));
  }

  private get presignConfigured(): boolean {
    return !!(this.r2Endpoint && this.r2AccessKeyId && this.r2SecretAccessKey);
  }

  private async presign(
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

  // ── write ────────────────────────────────────────────────────────

  async write(
    actor: string,
    path: string,
    body: ArrayBuffer | ReadableStream,
    contentType: string,
    msg?: string,
  ): Promise<WriteResult> {
    const buf = body instanceof ArrayBuffer ? body : await streamToBuffer(body);
    const addr = await sha256(buf);

    // Upload blob to R2 (dedup check)
    const key = blobKey(actor, addr);
    const existing = await this.bucket.head(key);
    if (!existing) {
      await this.bucket.put(key, buf, { httpMetadata: { contentType } });
    }

    // Record metadata in DO's SQLite
    return this.stub(actor).recordWrite(path, addr, buf.byteLength, contentType, msg);
  }

  // ── move ─────────────────────────────────────────────────────────

  async move(
    actor: string,
    from: string,
    to: string,
    msg?: string,
  ): Promise<MutationResult> {
    const result = await this.stub(actor).recordMove(from, to, msg);
    return { tx: result.tx, time: result.time };
  }

  // ── delete ───────────────────────────────────────────────────────

  async delete(
    actor: string,
    paths: string[],
    msg?: string,
  ): Promise<DeleteResult> {
    return this.stub(actor).recordDelete(paths, msg);
  }

  // ── read ─────────────────────────────────────────────────────────

  async read(actor: string, path: string): Promise<ReadResult | null> {
    const info = await this.stub(actor).getFileAddr(path);
    if (!info) return null;

    if (info.addr) {
      const key = blobKey(actor, info.addr);
      const obj = await this.bucket.get(key);
      if (!obj) {
        // Fallback to legacy key
        const legacy = await this.bucket.get(`${actor}/${path}`);
        if (!legacy) return null;
        return { body: legacy.body, meta: info.meta };
      }
      return { body: obj.body, meta: info.meta };
    }

    // Legacy (no addr)
    const obj = await this.bucket.get(`${actor}/${path}`);
    if (!obj) return null;
    return { body: obj.body, meta: info.meta };
  }

  // ── head ─────────────────────────────────────────────────────────

  async head(actor: string, path: string): Promise<FileMeta | null> {
    return this.stub(actor).getFileMeta(path);
  }

  // ── list ─────────────────────────────────────────────────────────

  async list(
    actor: string,
    opts?: ListOptions,
  ): Promise<{ entries: FileEntry[]; truncated: boolean }> {
    return this.stub(actor).listFiles(opts);
  }

  // ── search ───────────────────────────────────────────────────────

  async search(
    actor: string,
    query: string,
    opts?: { limit?: number; prefix?: string },
  ): Promise<SearchResult[]> {
    return this.stub(actor).searchFiles(query, opts);
  }

  // ── stats ────────────────────────────────────────────────────────

  async stats(actor: string): Promise<{ files: number; bytes: number }> {
    return this.stub(actor).getStats();
  }

  // ── allNames ─────────────────────────────────────────────────────

  async allNames(actor: string): Promise<{ path: string; name: string }[]> {
    return this.stub(actor).getAllNames();
  }

  // ── log ──────────────────────────────────────────────────────────

  async log(actor: string, opts?: LogOptions): Promise<StorageEvent[]> {
    return this.stub(actor).getLog(opts);
  }

  // ── presign read ─────────────────────────────────────────────────

  async presignRead(
    actor: string,
    path: string,
    expiresIn = 3600,
  ): Promise<string | null> {
    if (!this.presignConfigured) return null;

    const info = await this.stub(actor).getFileAddr(path);
    if (!info) return null;

    const key = info.addr ? blobKey(actor, info.addr) : `${actor}/${path}`;
    return this.presign("GET", key, expiresIn);
  }

  // ── presign upload ───────────────────────────────────────────────

  async presignUpload(
    actor: string,
    path: string,
    contentType: string,
    expiresIn = 3600,
    contentHash?: string,
  ): Promise<string> {
    if (contentHash) {
      return this.presign("PUT", blobKey(actor, contentHash), expiresIn, { contentType });
    }
    const key = `${actor}/${path}`;
    return this.presign("PUT", key, expiresIn, { contentType });
  }

  // ── confirm upload ───────────────────────────────────────────────

  async confirmUpload(
    actor: string,
    path: string,
    msg?: string,
    contentHash?: string,
  ): Promise<WriteResult> {
    if (contentHash) {
      // Client provided hash — verify blob exists via HEAD, no data pull
      const key = blobKey(actor, contentHash);
      const head = await this.bucket.head(key);
      if (!head) throw new Error("Upload not found at content-addressed location");
      const ct = head.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);
      return this.stub(actor).recordWrite(path, contentHash, head.size, ct, msg);
    }
    // Legacy flow: pull data, compute hash, re-store
    const legacyKey = `${actor}/${path}`;
    const obj = await this.bucket.get(legacyKey);
    if (!obj) throw new Error("Upload not found in storage");

    const buf = await obj.arrayBuffer();
    const contentType = obj.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);

    const result = await this.write(actor, path, buf, contentType, msg);
    await this.bucket.delete(legacyKey);
    return result;
  }

  async blobExists(actor: string, contentHash: string): Promise<number | null> {
    const head = await this.bucket.head(blobKey(actor, contentHash));
    return head ? head.size : null;
  }

  // ── multipart ────────────────────────────────────────────────────

  async initiateMultipart(
    actor: string,
    path: string,
    contentType: string,
    partCount: number,
    contentHash?: string,
  ): Promise<{ upload_id: string; part_urls: string[]; expires_in: number }> {
    if (!this.presignConfigured) throw new Error("Presigned URLs not configured");

    const key = contentHash ? blobKey(actor, contentHash) : `${actor}/${path}`;
    const mpu = await this.bucket.createMultipartUpload(key, {
      httpMetadata: { contentType },
    });

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
    contentHash?: string,
  ): Promise<WriteResult> {
    const key = contentHash ? blobKey(actor, contentHash) : `${actor}/${path}`;
    const mpu = this.bucket.resumeMultipartUpload(key, uploadId);
    await mpu.complete(parts.map((p) => ({ partNumber: p.part_number, etag: p.etag })));

    if (contentHash) {
      // Client provided hash — verify assembled blob via HEAD, no data pull
      const head = await this.bucket.head(key);
      if (!head) throw new Error("Multipart upload not found after completion");
      const ct = head.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);
      return this.stub(actor).recordWrite(path, contentHash, head.size, ct, msg);
    }
    // Legacy flow: pull assembled data, compute hash, re-store
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
