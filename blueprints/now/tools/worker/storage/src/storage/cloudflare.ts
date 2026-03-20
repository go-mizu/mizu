// ── Cloudflare D1 + R2 implementation of StorageEngine ───────────────
//
// Blobs:  R2 at  blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}
// Events: D1 `events` table — append-only mutation log
// Files:  D1 `files` table  — materialized projection for fast reads

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

// ── Config ───────────────────────────────────────────────────────────

interface CloudflareConfig {
  db: D1Database;
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

/** Consume a ReadableStream into ArrayBuffer */
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

// ── Implementation ───────────────────────────────────────────────────

export class CloudflareEngine implements StorageEngine {
  private db: D1Database;
  private bucket: R2Bucket;
  private r2Endpoint?: string;
  private r2AccessKeyId?: string;
  private r2SecretAccessKey?: string;
  private r2BucketName: string;

  constructor(config: CloudflareConfig) {
    this.db = config.db;
    this.bucket = config.bucket;
    this.r2Endpoint = config.r2Endpoint;
    this.r2AccessKeyId = config.r2AccessKeyId;
    this.r2SecretAccessKey = config.r2SecretAccessKey;
    this.r2BucketName = config.r2BucketName || "storage-files";
  }

  // ── tx allocation ────────────────────────────────────────────────

  private async nextTx(actor: string): Promise<number> {
    // Atomic increment: upsert then read
    await this.db
      .prepare(
        "INSERT INTO tx_counter (actor, next_tx) VALUES (?, 1) " +
          "ON CONFLICT (actor) DO UPDATE SET next_tx = next_tx + 1",
      )
      .bind(actor)
      .run();

    const row = await this.db
      .prepare("SELECT next_tx FROM tx_counter WHERE actor = ?")
      .bind(actor)
      .first<{ next_tx: number }>();

    return row!.next_tx;
  }

  // ── presign helpers ──────────────────────────────────────────────

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
    const size = buf.byteLength;
    const now = Date.now();
    const name = path.split("/").pop() || path;

    // Check if blob already exists in R2 (dedup)
    const key = blobKey(actor, addr);
    const existing = await this.bucket.head(key);
    if (!existing) {
      await this.bucket.put(key, buf, { httpMetadata: { contentType } });
    }

    // Read old addr for ref_count decrement (if overwriting)
    const oldFile = await this.db
      .prepare("SELECT addr FROM files WHERE owner = ? AND path = ?")
      .bind(actor, path)
      .first<{ addr: string | null }>();

    const tx = await this.nextTx(actor);

    const stmts: D1PreparedStatement[] = [
      // Event
      this.db
        .prepare(
          "INSERT INTO events (tx, actor, action, path, addr, size, type, msg, ts) VALUES (?, ?, 'write', ?, ?, ?, ?, ?, ?)",
        )
        .bind(tx, actor, path, addr, size, contentType, msg || `write ${path}`, now),
      // Files projection
      this.db
        .prepare(
          "INSERT INTO files (owner, path, name, size, type, addr, tx, tx_time, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) " +
            "ON CONFLICT (owner, path) DO UPDATE SET name=excluded.name, size=excluded.size, type=excluded.type, addr=excluded.addr, tx=excluded.tx, tx_time=excluded.tx_time, updated_at=excluded.updated_at",
        )
        .bind(actor, path, name, size, contentType, addr, tx, now, now),
      // Blob tracking — upsert ref_count
      this.db
        .prepare(
          "INSERT INTO blobs (addr, actor, size, ref_count, created_at) VALUES (?, ?, ?, 1, ?) " +
            "ON CONFLICT (addr, actor) DO UPDATE SET ref_count = ref_count + 1",
        )
        .bind(addr, actor, size, now),
    ];

    // Decrement old blob ref_count if overwriting with different content
    if (oldFile?.addr && oldFile.addr !== addr) {
      stmts.push(
        this.db
          .prepare("UPDATE blobs SET ref_count = MAX(ref_count - 1, 0) WHERE addr = ? AND actor = ?")
          .bind(oldFile.addr, actor),
      );
    }
    // If same addr, the upsert above already incremented, but we should not
    // double-count — the file was already pointing at this addr. Correct the
    // ref_count by not incrementing (we only need one ref per file row).
    // Actually: upsert increments, but the old file entry already held a ref.
    // So for same-addr overwrite, net change is 0. We incremented above, so
    // decrement the extra.
    if (oldFile?.addr && oldFile.addr === addr) {
      stmts.push(
        this.db
          .prepare("UPDATE blobs SET ref_count = MAX(ref_count - 1, 0) WHERE addr = ? AND actor = ?")
          .bind(addr, actor),
      );
    }

    await this.db.batch(stmts);

    return { tx, time: now, size };
  }

  // ── move ─────────────────────────────────────────────────────────

  async move(
    actor: string,
    from: string,
    to: string,
    msg?: string,
  ): Promise<MutationResult> {
    const file = await this.db
      .prepare("SELECT addr, size, type FROM files WHERE owner = ? AND path = ?")
      .bind(actor, from)
      .first<{ addr: string | null; size: number; type: string }>();

    if (!file) throw new Error("Source not found: " + from);

    const now = Date.now();
    const tx = await this.nextTx(actor);
    const newName = to.split("/").pop() || to;
    const meta = JSON.stringify({ from });

    await this.db.batch([
      // Event
      this.db
        .prepare(
          "INSERT INTO events (tx, actor, action, path, addr, size, type, meta, msg, ts) VALUES (?, ?, 'move', ?, ?, ?, ?, ?, ?, ?)",
        )
        .bind(tx, actor, to, file.addr, file.size, file.type, meta, msg || `move ${from} → ${to}`, now),
      // Remove old file entry
      this.db.prepare("DELETE FROM files WHERE owner = ? AND path = ?").bind(actor, from),
      // Insert new file entry (same addr — no blob copy!)
      this.db
        .prepare(
          "INSERT INTO files (owner, path, name, size, type, addr, tx, tx_time, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) " +
            "ON CONFLICT (owner, path) DO UPDATE SET name=excluded.name, size=excluded.size, type=excluded.type, addr=excluded.addr, tx=excluded.tx, tx_time=excluded.tx_time, updated_at=excluded.updated_at",
        )
        .bind(actor, to, newName, file.size, file.type, file.addr, tx, now, now),
    ]);

    // No R2 operations needed — blob hasn't changed

    return { tx, time: now };
  }

  // ── delete ───────────────────────────────────────────────────────

  async delete(
    actor: string,
    paths: string[],
    msg?: string,
  ): Promise<DeleteResult> {
    const now = Date.now();
    const tx = await this.nextTx(actor);
    let deleted = 0;

    const stmts: D1PreparedStatement[] = [];

    for (const path of paths) {
      if (path.endsWith("/")) {
        // Recursive folder delete
        const { results } = await this.db
          .prepare("SELECT path, addr FROM files WHERE owner = ? AND path LIKE ?")
          .bind(actor, `${path}%`)
          .all();

        for (const row of results || []) {
          stmts.push(
            this.db
              .prepare(
                "INSERT INTO events (tx, actor, action, path, addr, size, type, msg, ts) VALUES (?, ?, 'delete', ?, NULL, 0, NULL, ?, ?)",
              )
              .bind(tx, actor, row.path as string, msg || `delete ${path}*`, now),
          );
          if (row.addr) {
            stmts.push(
              this.db
                .prepare("UPDATE blobs SET ref_count = MAX(ref_count - 1, 0) WHERE addr = ? AND actor = ?")
                .bind(row.addr as string, actor),
            );
          }
          deleted++;
        }

        stmts.push(
          this.db.prepare("DELETE FROM files WHERE owner = ? AND path LIKE ?").bind(actor, `${path}%`),
        );
      } else {
        // Single file delete
        const file = await this.db
          .prepare("SELECT addr FROM files WHERE owner = ? AND path = ?")
          .bind(actor, path)
          .first<{ addr: string | null }>();

        stmts.push(
          this.db
            .prepare(
              "INSERT INTO events (tx, actor, action, path, addr, size, type, msg, ts) VALUES (?, ?, 'delete', ?, NULL, 0, NULL, ?, ?)",
            )
            .bind(tx, actor, path, msg || `delete ${path}`, now),
        );

        if (file?.addr) {
          stmts.push(
            this.db
              .prepare("UPDATE blobs SET ref_count = MAX(ref_count - 1, 0) WHERE addr = ? AND actor = ?")
              .bind(file.addr, actor),
          );
        }

        stmts.push(
          this.db.prepare("DELETE FROM files WHERE owner = ? AND path = ?").bind(actor, path),
        );
        deleted++;
      }
    }

    if (stmts.length) await this.db.batch(stmts);

    // R2 blobs NOT deleted here — GC handles cleanup of ref_count=0 blobs

    return { tx, time: now, deleted };
  }

  // ── read ─────────────────────────────────────────────────────────

  async read(actor: string, path: string): Promise<ReadResult | null> {
    const file = await this.db
      .prepare("SELECT path, name, size, type, addr, tx, tx_time FROM files WHERE owner = ? AND path = ?")
      .bind(actor, path)
      .first<{ path: string; name: string; size: number; type: string; addr: string | null; tx: number; tx_time: number }>();

    if (!file) return null;

    // Content-addressed read
    if (file.addr) {
      const key = blobKey(actor, file.addr);
      const obj = await this.bucket.get(key);
      if (!obj) {
        // Blob missing — fall back to legacy path
        return this.readLegacy(actor, path, file);
      }
      return {
        body: obj.body,
        meta: { path: file.path, name: file.name, size: file.size, type: file.type, tx: file.tx, tx_time: file.tx_time },
      };
    }

    // Legacy: file has no addr yet (pre-migration)
    return this.readLegacy(actor, path, file);
  }

  private async readLegacy(
    actor: string,
    path: string,
    file: { path: string; name: string; size: number; type: string; tx: number; tx_time: number },
  ): Promise<ReadResult | null> {
    const obj = await this.bucket.get(`${actor}/${path}`);
    if (!obj) return null;
    return {
      body: obj.body,
      meta: { path: file.path, name: file.name, size: file.size, type: file.type, tx: file.tx || 0, tx_time: file.tx_time || 0 },
    };
  }

  // ── head ─────────────────────────────────────────────────────────

  async head(actor: string, path: string): Promise<FileMeta | null> {
    const file = await this.db
      .prepare("SELECT path, name, size, type, tx, tx_time FROM files WHERE owner = ? AND path = ?")
      .bind(actor, path)
      .first<{ path: string; name: string; size: number; type: string; tx: number; tx_time: number }>();

    if (!file) return null;
    return { path: file.path, name: file.name, size: file.size, type: file.type, tx: file.tx || 0, tx_time: file.tx_time || 0 };
  }

  // ── list ─────────────────────────────────────────────────────────

  async list(
    actor: string,
    opts?: ListOptions,
  ): Promise<{ entries: FileEntry[]; truncated: boolean }> {
    const prefix = opts?.prefix || "";
    const limit = Math.min(opts?.limit || 200, 1000);
    const offset = opts?.offset || 0;

    const { results } = await this.db
      .prepare(
        "SELECT path, name, size, type, updated_at, tx, tx_time FROM files WHERE owner = ? AND path LIKE ? ORDER BY path LIMIT ? OFFSET ?",
      )
      .bind(actor, `${prefix}%`, limit + 1, offset)
      .all();

    const rows = results || [];
    const truncated = rows.length > limit;
    if (truncated) rows.pop();

    const entries: FileEntry[] = [];
    const dirs = new Set<string>();

    for (const row of rows) {
      const relative = (row.path as string).slice(prefix.length);
      const slash = relative.indexOf("/");
      if (slash === -1) {
        entries.push({
          name: relative,
          type: row.type as string,
          size: row.size as number,
          updated_at: row.updated_at as number,
          tx: row.tx as number,
          tx_time: row.tx_time as number,
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
    const limit = Math.min(opts?.limit || 50, 200);
    const pfx = opts?.prefix || "";

    let sql =
      "SELECT path, name, size, type, tx FROM files WHERE owner = ? AND (name LIKE ? OR path LIKE ?)";
    const binds: any[] = [actor, `%${query}%`, `%${query}%`];

    if (pfx) {
      sql += " AND path LIKE ?";
      binds.push(`${pfx}%`);
    }

    sql += " ORDER BY updated_at DESC LIMIT ?";
    binds.push(limit);

    const { results } = await this.db.prepare(sql).bind(...binds).all();

    return (results || []).map((r) => ({
      path: r.path as string,
      name: r.name as string,
      size: r.size as number,
      type: r.type as string,
      tx: (r.tx as number) || 0,
    }));
  }

  // ── stats ────────────────────────────────────────────────────────

  async stats(actor: string): Promise<{ files: number; bytes: number }> {
    const row = await this.db
      .prepare("SELECT COUNT(*) as files, COALESCE(SUM(size), 0) as bytes FROM files WHERE owner = ?")
      .bind(actor)
      .first<{ files: number; bytes: number }>();

    return { files: row?.files || 0, bytes: row?.bytes || 0 };
  }

  // ── log ──────────────────────────────────────────────────────────

  async log(actor: string, opts?: LogOptions): Promise<StorageEvent[]> {
    const limit = Math.min(opts?.limit || 50, 500);
    let sql = "SELECT tx, action, path, size, msg, meta, ts FROM events WHERE actor = ?";
    const binds: any[] = [actor];

    if (opts?.path) {
      sql += " AND path = ?";
      binds.push(opts.path);
    }
    if (opts?.since_tx) {
      sql += " AND tx > ?";
      binds.push(opts.since_tx);
    }

    sql += " ORDER BY tx DESC, id DESC LIMIT ?";
    binds.push(limit);

    const { results } = await this.db.prepare(sql).bind(...binds).all();

    return (results || []).map((r) => ({
      tx: r.tx as number,
      action: r.action as "write" | "move" | "delete",
      path: r.path as string,
      size: r.size as number,
      msg: r.msg as string | null,
      ts: r.ts as number,
      meta: r.meta ? JSON.parse(r.meta as string) : null,
    }));
  }

  // ── presign read ─────────────────────────────────────────────────

  async presignRead(
    actor: string,
    path: string,
    expiresIn = 3600,
  ): Promise<string | null> {
    if (!this.presignConfigured) return null;

    const file = await this.db
      .prepare("SELECT addr FROM files WHERE owner = ? AND path = ?")
      .bind(actor, path)
      .first<{ addr: string | null }>();

    if (!file) return null;

    // Use content-addressed key if available, else legacy key
    const key = file.addr ? blobKey(actor, file.addr) : `${actor}/${path}`;
    return this.presign("GET", key, expiresIn);
  }

  // ── presign upload ───────────────────────────────────────────────

  async presignUpload(
    actor: string,
    path: string,
    contentType: string,
    expiresIn = 3600,
  ): Promise<string> {
    // Presigned uploads still go to legacy path — confirmUpload will
    // content-address them.
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

    // Write via the normal path (content-address it)
    const result = await this.write(actor, path, buf, contentType, msg);

    // Clean up the legacy key (the blob is now content-addressed)
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

    // Multipart uploads use legacy key — completeMultipart will content-address
    const key = `${actor}/${path}`;
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
  ): Promise<WriteResult> {
    const key = `${actor}/${path}`;
    const mpu = this.bucket.resumeMultipartUpload(key, uploadId);
    await mpu.complete(parts.map((p) => ({ partNumber: p.part_number, etag: p.etag })));

    // Now content-address the assembled object
    const obj = await this.bucket.get(key);
    if (!obj) throw new Error("Multipart upload not found after completion");

    const buf = await obj.arrayBuffer();
    const contentType = obj.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);

    const result = await this.write(actor, path, buf, contentType, msg);

    // Clean up legacy key
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
