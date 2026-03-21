// ── PostgreSQL inode-based base (v2) ──────────────────────────────────
//
// Abstract base class for the inode-based storage engine using PostgreSQL.
// Port of DO v2 (do_v2_driver.ts) logic to PostgreSQL with the abstract
// query/transaction pattern from pg_base.ts.
//
// Schema uses `stg_` prefix with an `owner` column (not table sharding).
// Subclasses provide query execution; this class handles SQL and R2.

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

const NID_CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-";
function nanoid(): string {
  const b = crypto.getRandomValues(new Uint8Array(21));
  let id = "";
  for (let i = 0; i < 21; i++) id += NID_CHARS[b[i] & 63];
  return id;
}

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

/** Split "docs/readme.md" -> { dir: "docs/", name: "readme.md" } */
function splitPath(path: string): { dir: string; name: string } {
  const p = path.replace(/^\/+/, "");
  const i = p.lastIndexOf("/");
  if (i === -1) return { dir: "", name: p };
  return { dir: p.slice(0, i + 1), name: p.slice(i + 1) };
}

// ── Schema DDL ──────────────────────────────────────────────────────

const SCHEMA_STMTS = [
  `CREATE TABLE IF NOT EXISTS stg_nodes (
    owner TEXT NOT NULL,
    node_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK(kind IN ('file','dir')),
    created_at BIGINT NOT NULL,
    deleted_at BIGINT,
    PRIMARY KEY (owner, node_id)
  )`,
  `CREATE INDEX IF NOT EXISTS idx_stg_nodes_kind ON stg_nodes(owner, kind) WHERE deleted_at IS NULL`,

  `CREATE TABLE IF NOT EXISTS stg_directory_entries (
    owner TEXT NOT NULL,
    parent_id TEXT NOT NULL,
    name TEXT NOT NULL,
    child_id TEXT NOT NULL,
    created_tx BIGINT NOT NULL,
    deleted_tx BIGINT,
    PRIMARY KEY (owner, parent_id, name)
  )`,
  `CREATE INDEX IF NOT EXISTS idx_stg_de_child ON stg_directory_entries(owner, child_id)`,

  `CREATE TABLE IF NOT EXISTS stg_file_versions (
    owner TEXT NOT NULL,
    node_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    size BIGINT NOT NULL,
    content_type TEXT,
    created_tx BIGINT NOT NULL,
    PRIMARY KEY (owner, node_id, version)
  )`,

  `CREATE TABLE IF NOT EXISTS stg_file_current_state (
    owner TEXT NOT NULL,
    node_id TEXT NOT NULL PRIMARY KEY,
    content_hash TEXT NOT NULL,
    size BIGINT NOT NULL,
    content_type TEXT,
    version INTEGER NOT NULL,
    updated_tx BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
  )`,
  `CREATE INDEX IF NOT EXISTS idx_stg_fcs_owner ON stg_file_current_state(owner)`,

  `CREATE TABLE IF NOT EXISTS stg_blob_refs (
    owner TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    size BIGINT NOT NULL,
    ref_count INTEGER NOT NULL DEFAULT 1,
    created_at BIGINT NOT NULL,
    PRIMARY KEY (owner, content_hash)
  )`,

  `CREATE TABLE IF NOT EXISTS stg_events (
    id BIGSERIAL PRIMARY KEY,
    owner TEXT NOT NULL,
    tx BIGINT NOT NULL,
    action TEXT NOT NULL CHECK(action IN ('write','move','delete')),
    node_id TEXT NOT NULL,
    path TEXT NOT NULL,
    content_hash TEXT,
    size BIGINT NOT NULL DEFAULT 0,
    content_type TEXT,
    meta TEXT,
    msg TEXT,
    ts BIGINT NOT NULL
  )`,
  `CREATE INDEX IF NOT EXISTS idx_stg_events_owner_tx ON stg_events(owner, tx DESC)`,
  `CREATE INDEX IF NOT EXISTS idx_stg_events_path ON stg_events(owner, path, tx DESC)`,
  `CREATE INDEX IF NOT EXISTS idx_stg_events_node ON stg_events(owner, node_id)`,

  `CREATE TABLE IF NOT EXISTS stg_transactions (
    owner TEXT NOT NULL,
    tx BIGINT NOT NULL,
    ts BIGINT NOT NULL,
    msg TEXT,
    PRIMARY KEY (owner, tx)
  )`,
  `CREATE INDEX IF NOT EXISTS idx_stg_transactions_next ON stg_transactions(owner, tx DESC)`,

  `CREATE TABLE IF NOT EXISTS stg_path_cache (
    owner TEXT NOT NULL,
    path TEXT NOT NULL,
    node_id TEXT NOT NULL,
    parent_id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'file',
    updated_tx BIGINT NOT NULL,
    PRIMARY KEY (owner, path)
  )`,
  `CREATE INDEX IF NOT EXISTS idx_stg_pc_node ON stg_path_cache(owner, node_id)`,
  `CREATE INDEX IF NOT EXISTS idx_stg_pc_kind ON stg_path_cache(owner, kind) WHERE kind = 'file'`,
];

/** Module-level flag — schema only checked once per isolate. */
let schemaReady = false;

/** Reset schema-ready flag (for tests). */
export function resetSchemaFlag(): void {
  schemaReady = false;
}

// ── Abstract base class ─────────────────────────────────────────────

export abstract class PgV2EngineBase implements StorageEngine {
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

  /** Ensure PostgreSQL schema exists (idempotent). */
  async ensureSchema(): Promise<void> {
    if (schemaReady) return;
    for (const stmt of SCHEMA_STMTS) {
      await this.query(stmt);
    }
    // Ensure root node exists for every owner that already has data.
    // New owners get root lazily in ensureDirChain.
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

  // ── tx allocation ─────────────────────────────────────────────────

  /** Allocate next tx number atomically. Must be called within a transaction. */
  private async nextTx(owner: string, q: QueryFn): Promise<number> {
    const rows = await q<{ tx: string }>(
      `INSERT INTO stg_transactions (owner, tx, ts)
       SELECT $1, COALESCE(MAX(tx), 0) + 1, $2 FROM stg_transactions WHERE owner = $1
       RETURNING tx`,
      [owner, Date.now()],
    );
    return Number(rows[0].tx);
  }

  // ── path resolution ───────────────────────────────────────────────

  /** Resolve a path to { node_id, parent_id, name } or null. */
  private async resolvePath(
    owner: string,
    path: string,
    q?: QueryFn,
  ): Promise<{ node_id: string; parent_id: string; name: string } | null> {
    const query = q || this.query.bind(this);
    if (!path || path === "/") return { node_id: "root", parent_id: "", name: "" };
    const p = path.replace(/^\/+/, "").replace(/\/+$/, "");
    if (!p) return { node_id: "root", parent_id: "", name: "" };

    // Check cache
    const cached = await query<{ node_id: string; parent_id: string; name: string }>(
      "SELECT node_id, parent_id, name FROM stg_path_cache WHERE owner = $1 AND path = $2",
      [owner, p],
    );
    if (cached.length) {
      // Verify node is alive
      const alive = await query<{ node_id: string }>(
        "SELECT node_id FROM stg_nodes WHERE owner = $1 AND node_id = $2 AND deleted_at IS NULL",
        [owner, cached[0].node_id],
      );
      if (alive.length) return cached[0];
      // Stale cache — remove
      await query("DELETE FROM stg_path_cache WHERE owner = $1 AND path = $2", [owner, p]);
    }

    // Walk directory tree from root
    const segments = p.split("/");
    let current = "root";
    let parentId = "";

    for (let i = 0; i < segments.length; i++) {
      const seg = segments[i];
      const rows = await query<{ child_id: string }>(
        "SELECT child_id FROM stg_directory_entries WHERE owner = $1 AND parent_id = $2 AND name = $3 AND deleted_tx IS NULL",
        [owner, current, seg],
      );
      if (!rows.length) return null;
      parentId = current;
      current = rows[0].child_id;
    }

    return { node_id: current, parent_id: parentId, name: segments[segments.length - 1] };
  }

  /** Ensure all directories in a path exist, creating as needed. Returns deepest dir node_id. */
  private async ensureDirChain(
    owner: string,
    dirPath: string,
    tx: number,
    now: number,
    q: QueryFn,
  ): Promise<string> {
    if (!dirPath) return "root";
    const segments = dirPath.replace(/^\/+/, "").replace(/\/+$/, "").split("/").filter(Boolean);
    if (!segments.length) return "root";

    // Ensure root node exists for this owner
    await q(
      `INSERT INTO stg_nodes (owner, node_id, kind, created_at) VALUES ($1, 'root', 'dir', $2)
       ON CONFLICT (owner, node_id) DO NOTHING`,
      [owner, now],
    );

    let current = "root";
    let accPath = "";

    for (const seg of segments) {
      accPath = accPath ? accPath + "/" + seg : seg;

      const rows = await q<{ child_id: string }>(
        "SELECT child_id FROM stg_directory_entries WHERE owner = $1 AND parent_id = $2 AND name = $3 AND deleted_tx IS NULL",
        [owner, current, seg],
      );

      if (rows.length) {
        current = rows[0].child_id;
      } else {
        const nid = nanoid();
        await q(
          "INSERT INTO stg_nodes (owner, node_id, kind, created_at) VALUES ($1, $2, 'dir', $3)",
          [owner, nid, now],
        );
        await q(
          "INSERT INTO stg_directory_entries (owner, parent_id, name, child_id, created_tx) VALUES ($1, $2, $3, $4, $5)",
          [owner, current, seg, nid, tx],
        );
        await q(
          `INSERT INTO stg_path_cache (owner, path, node_id, parent_id, name, kind, updated_tx)
           VALUES ($1, $2, $3, $4, $5, 'dir', $6)
           ON CONFLICT (owner, path) DO UPDATE SET
             node_id = EXCLUDED.node_id, parent_id = EXCLUDED.parent_id,
             name = EXCLUDED.name, kind = EXCLUDED.kind, updated_tx = EXCLUDED.updated_tx`,
          [owner, accPath, nid, current, seg, tx],
        );
        current = nid;
      }
    }
    return current;
  }

  /** Recursively collect file descendant info under a directory node. */
  private async collectDescendantFiles(
    owner: string,
    parentId: string,
    q: QueryFn,
  ): Promise<{ node_id: string; name: string; path: string; content_hash: string | null }[]> {
    const result: { node_id: string; name: string; path: string; content_hash: string | null }[] = [];

    const children = await q<{ child_id: string; name: string }>(
      "SELECT child_id, name FROM stg_directory_entries WHERE owner = $1 AND parent_id = $2 AND deleted_tx IS NULL",
      [owner, parentId],
    );

    for (const child of children) {
      const kind = await q<{ kind: string }>(
        "SELECT kind FROM stg_nodes WHERE owner = $1 AND node_id = $2 AND deleted_at IS NULL",
        [owner, child.child_id],
      );
      if (!kind.length) continue;

      if (kind[0].kind === "file") {
        const fcs = await q<{ content_hash: string }>(
          "SELECT content_hash FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
          [owner, child.child_id],
        );
        result.push({
          node_id: child.child_id,
          name: child.name,
          path: "",
          content_hash: fcs[0]?.content_hash ?? null,
        });
      } else {
        result.push(...await this.collectDescendantFiles(owner, child.child_id, q));
      }
    }

    return result;
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
    const p = path.replace(/^\/+/, "");
    const { dir, name } = splitPath(p);
    const autoMsg = msg || `write ${p}`;

    // R2 dedup
    const key = blobKey(actor, addr);
    const existing = await this.bucket.head(key);
    if (!existing) {
      await this.bucket.put(key, buf, { httpMetadata: { contentType } });
    }

    return this.transaction(async (q) => {
      const tx = await this.nextTx(actor, q);

      // Update transaction msg
      await q(
        "UPDATE stg_transactions SET msg = $1 WHERE owner = $2 AND tx = $3",
        [autoMsg, actor, tx],
      );

      // Ensure root node
      await q(
        `INSERT INTO stg_nodes (owner, node_id, kind, created_at) VALUES ($1, 'root', 'dir', $2)
         ON CONFLICT (owner, node_id) DO NOTHING`,
        [actor, now],
      );

      const parentId = await this.ensureDirChain(actor, dir, tx, now, q);

      // Check if file exists
      const existingEntry = await q<{ child_id: string }>(
        "SELECT child_id FROM stg_directory_entries WHERE owner = $1 AND parent_id = $2 AND name = $3 AND deleted_tx IS NULL",
        [actor, parentId, name],
      );

      let nodeId: string;
      if (existingEntry.length) {
        nodeId = existingEntry[0].child_id;
        const cur = await q<{ version: string; content_hash: string }>(
          "SELECT version, content_hash FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
          [actor, nodeId],
        );
        const newVer = (Number(cur[0]?.version) || 0) + 1;
        const oldHash = cur[0]?.content_hash ?? null;

        await q(
          `INSERT INTO stg_file_versions (owner, node_id, version, content_hash, size, content_type, created_tx)
           VALUES ($1, $2, $3, $4, $5, $6, $7)`,
          [actor, nodeId, newVer, addr, size, contentType, tx],
        );
        await q(
          `INSERT INTO stg_file_current_state (owner, node_id, content_hash, size, content_type, version, updated_tx, updated_at)
           VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
           ON CONFLICT (node_id) DO UPDATE SET
             content_hash = EXCLUDED.content_hash, size = EXCLUDED.size,
             content_type = EXCLUDED.content_type, version = EXCLUDED.version,
             updated_tx = EXCLUDED.updated_tx, updated_at = EXCLUDED.updated_at`,
          [actor, nodeId, addr, size, contentType, newVer, tx, now],
        );
        await q(
          `INSERT INTO stg_blob_refs (owner, content_hash, size, ref_count, created_at)
           VALUES ($1, $2, $3, 1, $4)
           ON CONFLICT (owner, content_hash) DO UPDATE SET ref_count = stg_blob_refs.ref_count + 1`,
          [actor, addr, size, now],
        );
        if (oldHash && oldHash !== addr) {
          await q(
            "UPDATE stg_blob_refs SET ref_count = GREATEST(ref_count - 1, 0) WHERE owner = $1 AND content_hash = $2",
            [actor, oldHash],
          );
        }
        if (oldHash && oldHash === addr) {
          await q(
            "UPDATE stg_blob_refs SET ref_count = GREATEST(ref_count - 1, 0) WHERE owner = $1 AND content_hash = $2",
            [actor, addr],
          );
        }
      } else {
        nodeId = nanoid();
        await q(
          "INSERT INTO stg_nodes (owner, node_id, kind, created_at) VALUES ($1, $2, 'file', $3)",
          [actor, nodeId, now],
        );
        await q(
          "INSERT INTO stg_directory_entries (owner, parent_id, name, child_id, created_tx) VALUES ($1, $2, $3, $4, $5)",
          [actor, parentId, name, nodeId, tx],
        );
        await q(
          `INSERT INTO stg_file_versions (owner, node_id, version, content_hash, size, content_type, created_tx)
           VALUES ($1, $2, 1, $3, $4, $5, $6)`,
          [actor, nodeId, addr, size, contentType, tx],
        );
        await q(
          `INSERT INTO stg_file_current_state (owner, node_id, content_hash, size, content_type, version, updated_tx, updated_at)
           VALUES ($1, $2, $3, $4, $5, 1, $6, $7)`,
          [actor, nodeId, addr, size, contentType, tx, now],
        );
        await q(
          `INSERT INTO stg_blob_refs (owner, content_hash, size, ref_count, created_at)
           VALUES ($1, $2, $3, 1, $4)
           ON CONFLICT (owner, content_hash) DO UPDATE SET ref_count = stg_blob_refs.ref_count + 1`,
          [actor, addr, size, now],
        );
      }

      await q(
        `INSERT INTO stg_events (owner, tx, action, node_id, path, content_hash, size, content_type, msg, ts)
         VALUES ($1, $2, 'write', $3, $4, $5, $6, $7, $8, $9)`,
        [actor, tx, nodeId, p, addr, size, contentType, autoMsg, now],
      );
      await q(
        `INSERT INTO stg_path_cache (owner, path, node_id, parent_id, name, kind, updated_tx)
         VALUES ($1, $2, $3, $4, $5, 'file', $6)
         ON CONFLICT (owner, path) DO UPDATE SET
           node_id = EXCLUDED.node_id, parent_id = EXCLUDED.parent_id,
           name = EXCLUDED.name, kind = EXCLUDED.kind, updated_tx = EXCLUDED.updated_tx`,
        [actor, p, nodeId, parentId, name, tx],
      );

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
    const pFrom = from.replace(/^\/+/, "");
    const pTo = to.replace(/^\/+/, "");
    const autoMsg = msg || `move ${pFrom} → ${pTo}`;

    return this.transaction(async (q) => {
      const now = Date.now();

      // Ensure root node
      await q(
        `INSERT INTO stg_nodes (owner, node_id, kind, created_at) VALUES ($1, 'root', 'dir', $2)
         ON CONFLICT (owner, node_id) DO NOTHING`,
        [actor, now],
      );

      const src = await this.resolvePath(actor, pFrom, q);
      if (!src) throw new Error("Source not found: " + pFrom);

      const { dir: dstDir, name: dstName } = splitPath(pTo);

      // Get file info
      const fcs = await q<{ content_hash: string; size: string; content_type: string }>(
        "SELECT content_hash, size, content_type FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
        [actor, src.node_id],
      );
      const fileInfo = fcs[0] ?? { content_hash: null, size: "0", content_type: "application/octet-stream" };

      const tx = await this.nextTx(actor, q);

      await q(
        "UPDATE stg_transactions SET msg = $1 WHERE owner = $2 AND tx = $3",
        [autoMsg, actor, tx],
      );

      const dstParentId = await this.ensureDirChain(actor, dstDir, tx, now, q);

      // Unlink from old parent
      await q(
        "UPDATE stg_directory_entries SET deleted_tx = $1 WHERE owner = $2 AND parent_id = $3 AND name = $4 AND deleted_tx IS NULL",
        [tx, actor, src.parent_id, src.name],
      );

      // Link into new parent (handle conflict if destination name exists)
      await q(
        `INSERT INTO stg_directory_entries (owner, parent_id, name, child_id, created_tx)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (owner, parent_id, name) DO UPDATE SET
           child_id = EXCLUDED.child_id, created_tx = EXCLUDED.created_tx, deleted_tx = NULL`,
        [actor, dstParentId, dstName, src.node_id, tx],
      );

      const meta = JSON.stringify({ from: pFrom });
      await q(
        `INSERT INTO stg_events (owner, tx, action, node_id, path, size, meta, msg, ts)
         VALUES ($1, $2, 'move', $3, $4, $5, $6, $7, $8)`,
        [actor, tx, src.node_id, pTo, Number(fileInfo.size), meta, autoMsg, now],
      );

      // Invalidate path cache
      const kindRows = await q<{ kind: string }>(
        "SELECT kind FROM stg_nodes WHERE owner = $1 AND node_id = $2",
        [actor, src.node_id],
      );
      const isDir = kindRows[0]?.kind === "dir";

      await q("DELETE FROM stg_path_cache WHERE owner = $1 AND path = $2", [actor, pFrom]);
      if (isDir) {
        await q("DELETE FROM stg_path_cache WHERE owner = $1 AND path LIKE $2", [actor, pFrom.replace(/\/$/, "") + "/%"]);
      }
      await q(
        `INSERT INTO stg_path_cache (owner, path, node_id, parent_id, name, kind, updated_tx)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         ON CONFLICT (owner, path) DO UPDATE SET
           node_id = EXCLUDED.node_id, parent_id = EXCLUDED.parent_id,
           name = EXCLUDED.name, kind = EXCLUDED.kind, updated_tx = EXCLUDED.updated_tx`,
        [actor, pTo.replace(/\/$/, ""), src.node_id, dstParentId, dstName, isDir ? "dir" : "file", tx],
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

      // Ensure root node
      await q(
        `INSERT INTO stg_nodes (owner, node_id, kind, created_at) VALUES ($1, 'root', 'dir', $2)
         ON CONFLICT (owner, node_id) DO NOTHING`,
        [actor, now],
      );

      const tx = await this.nextTx(actor, q);
      await q(
        "UPDATE stg_transactions SET msg = $1 WHERE owner = $2 AND tx = $3",
        [msg || "delete", actor, tx],
      );

      for (const rawPath of paths) {
        const p = rawPath.replace(/^\/+/, "");

        if (p.endsWith("/")) {
          // Directory delete
          const dirNode = await this.resolvePath(actor, p.replace(/\/$/, ""), q);
          if (!dirNode) continue;

          // Collect all file nodes under this directory recursively
          const fileNodes = await this.collectDescendantFiles(actor, dirNode.node_id, q);
          for (const fn of fileNodes) {
            await q(
              "UPDATE stg_nodes SET deleted_at = $1 WHERE owner = $2 AND node_id = $3",
              [now, actor, fn.node_id],
            );
            if (fn.content_hash) {
              await q(
                "UPDATE stg_blob_refs SET ref_count = GREATEST(ref_count - 1, 0) WHERE owner = $1 AND content_hash = $2",
                [actor, fn.content_hash],
              );
            }
            await q(
              `INSERT INTO stg_events (owner, tx, action, node_id, path, size, msg, ts)
               VALUES ($1, $2, 'delete', $3, $4, 0, $5, $6)`,
              [actor, tx, fn.node_id, fn.path || p + fn.name, msg || `delete ${p}*`, now],
            );
            deleted++;
          }

          // Soft-delete the directory node and unlink
          await q(
            "UPDATE stg_nodes SET deleted_at = $1 WHERE owner = $2 AND node_id = $3",
            [now, actor, dirNode.node_id],
          );
          await q(
            "UPDATE stg_directory_entries SET deleted_tx = $1 WHERE owner = $2 AND parent_id = $3 AND name = $4 AND deleted_tx IS NULL",
            [tx, actor, dirNode.parent_id, dirNode.name],
          );
          // Invalidate cache
          await q("DELETE FROM stg_path_cache WHERE owner = $1 AND path LIKE $2", [actor, p + "%"]);
          await q("DELETE FROM stg_path_cache WHERE owner = $1 AND path = $2", [actor, p.replace(/\/$/, "")]);
        } else {
          const node = await this.resolvePath(actor, p, q);
          if (!node) { deleted++; continue; }

          const fcs = await q<{ content_hash: string }>(
            "SELECT content_hash FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
            [actor, node.node_id],
          );

          await q(
            "UPDATE stg_nodes SET deleted_at = $1 WHERE owner = $2 AND node_id = $3",
            [now, actor, node.node_id],
          );
          await q(
            "UPDATE stg_directory_entries SET deleted_tx = $1 WHERE owner = $2 AND parent_id = $3 AND name = $4 AND deleted_tx IS NULL",
            [tx, actor, node.parent_id, node.name],
          );
          if (fcs[0]?.content_hash) {
            await q(
              "UPDATE stg_blob_refs SET ref_count = GREATEST(ref_count - 1, 0) WHERE owner = $1 AND content_hash = $2",
              [actor, fcs[0].content_hash],
            );
          }
          await q(
            `INSERT INTO stg_events (owner, tx, action, node_id, path, size, msg, ts)
             VALUES ($1, $2, 'delete', $3, $4, 0, $5, $6)`,
            [actor, tx, node.node_id, p, msg || `delete ${p}`, now],
          );
          await q("DELETE FROM stg_path_cache WHERE owner = $1 AND path = $2", [actor, p]);
          deleted++;
        }
      }

      return { tx, time: now, deleted };
    });
  }

  // ── read ─────────────────────────────────────────────────────────

  async read(actor: string, path: string): Promise<ReadResult | null> {
    await this.ensureSchema();
    const p = path.replace(/^\/+/, "");

    const node = await this.resolvePath(actor, p);
    if (!node) return null;

    const fcs = await this.query<{
      content_hash: string; size: string; content_type: string;
      updated_tx: string; updated_at: string;
    }>(
      "SELECT content_hash, size, content_type, updated_tx, updated_at FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
      [actor, node.node_id],
    );
    if (!fcs.length) return null;

    const f = fcs[0];
    const meta: FileMeta = {
      path: p,
      name: node.name,
      size: Number(f.size),
      type: f.content_type,
      tx: Number(f.updated_tx),
      tx_time: Number(f.updated_at),
    };

    if (f.content_hash) {
      const key = blobKey(actor, f.content_hash);
      const obj = await this.bucket.get(key);
      if (!obj) {
        // Fallback to legacy key
        const legacy = await this.bucket.get(`${actor}/${p}`);
        if (!legacy) return null;
        return { body: legacy.body, meta };
      }
      return { body: obj.body, meta };
    }

    const obj = await this.bucket.get(`${actor}/${p}`);
    if (!obj) return null;
    return { body: obj.body, meta };
  }

  // ── head ─────────────────────────────────────────────────────────

  async head(actor: string, path: string): Promise<FileMeta | null> {
    await this.ensureSchema();
    const p = path.replace(/^\/+/, "");

    const node = await this.resolvePath(actor, p);
    if (!node) return null;

    const fcs = await this.query<{
      size: string; content_type: string; updated_tx: string; updated_at: string;
    }>(
      "SELECT size, content_type, updated_tx, updated_at FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
      [actor, node.node_id],
    );
    if (!fcs.length) return null;

    const f = fcs[0];
    return {
      path: p,
      name: node.name,
      size: Number(f.size),
      type: f.content_type,
      tx: Number(f.updated_tx),
      tx_time: Number(f.updated_at),
    };
  }

  // ── list ─────────────────────────────────────────────────────────

  async list(
    actor: string,
    opts?: ListOptions,
  ): Promise<{ entries: FileEntry[]; truncated: boolean }> {
    await this.ensureSchema();
    const prefix = (opts?.prefix || "").replace(/^\/+/, "");
    const limit = Math.min(opts?.limit || 200, 1000);
    const offset = opts?.offset || 0;

    // Resolve prefix to a directory node
    let parentId = "root";
    if (prefix) {
      const dirPath = prefix.replace(/\/$/, "");
      const node = await this.resolvePath(actor, dirPath);
      if (!node) return { entries: [], truncated: false };
      parentId = node.node_id;
    }

    // Ensure root exists (may not for empty owners)
    const rootCheck = await this.query<{ node_id: string }>(
      "SELECT node_id FROM stg_nodes WHERE owner = $1 AND node_id = $2 AND deleted_at IS NULL",
      [actor, parentId],
    );
    if (!rootCheck.length && parentId === "root") return { entries: [], truncated: false };

    // List children
    const children = await this.query<{ child_id: string; name: string }>(
      `SELECT child_id, name FROM stg_directory_entries
       WHERE owner = $1 AND parent_id = $2 AND deleted_tx IS NULL
       ORDER BY name LIMIT $3 OFFSET $4`,
      [actor, parentId, limit + 1, offset],
    );

    const truncated = children.length > limit;
    if (truncated) children.pop();

    const entries: FileEntry[] = [];
    for (const child of children) {
      const nodeRows = await this.query<{ kind: string }>(
        "SELECT kind FROM stg_nodes WHERE owner = $1 AND node_id = $2 AND deleted_at IS NULL",
        [actor, child.child_id],
      );
      if (!nodeRows.length) continue;

      if (nodeRows[0].kind === "dir") {
        entries.push({ name: child.name + "/", type: "directory" });
      } else {
        const fcs = await this.query<{
          size: string; content_type: string; updated_at: string; updated_tx: string;
        }>(
          "SELECT size, content_type, updated_at, updated_tx FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
          [actor, child.child_id],
        );
        if (fcs.length) {
          entries.push({
            name: child.name,
            type: fcs[0].content_type,
            size: Number(fcs[0].size),
            updated_at: Number(fcs[0].updated_at),
            tx: Number(fcs[0].updated_tx),
            tx_time: Number(fcs[0].updated_at),
          });
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

    let sql = `SELECT pc.path, pc.name, pc.node_id
               FROM stg_path_cache pc
               WHERE pc.owner = $1 AND pc.kind = 'file'
                 AND (pc.name ILIKE $2 OR pc.path ILIKE $2)`;
    const params: any[] = [actor, `%${query}%`];
    let idx = 3;

    if (pfx) {
      sql += ` AND pc.path LIKE $${idx}`;
      params.push(`${pfx}%`);
      idx++;
    }

    sql += ` LIMIT $${idx}`;
    params.push(limit);

    const rows = await this.query<{ path: string; name: string; node_id: string }>(sql, params);

    const results: SearchResult[] = [];
    for (const r of rows) {
      const fcs = await this.query<{ size: string; content_type: string; updated_tx: string }>(
        "SELECT size, content_type, updated_tx FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
        [actor, r.node_id],
      );
      if (fcs.length) {
        results.push({
          path: r.path,
          name: r.name,
          size: Number(fcs[0].size),
          type: fcs[0].content_type,
          tx: Number(fcs[0].updated_tx),
        });
      }
    }
    return results;
  }

  // ── stats ────────────────────────────────────────────────────────

  async stats(actor: string): Promise<{ files: number; bytes: number }> {
    await this.ensureSchema();
    const rows = await this.query<{ files: string; bytes: string }>(
      "SELECT COUNT(*) as files, COALESCE(SUM(size), 0) as bytes FROM stg_file_current_state WHERE owner = $1",
      [actor],
    );
    return { files: Number(rows[0]?.files || 0), bytes: Number(rows[0]?.bytes || 0) };
  }

  // ── allNames ─────────────────────────────────────────────────────

  async allNames(actor: string): Promise<{ path: string; name: string }[]> {
    await this.ensureSchema();
    return this.query<{ path: string; name: string }>(
      "SELECT path, name FROM stg_path_cache WHERE owner = $1 AND kind = 'file'",
      [actor],
    );
  }

  // ── log ──────────────────────────────────────────────────────────

  async log(actor: string, opts?: LogOptions): Promise<StorageEvent[]> {
    await this.ensureSchema();
    const limit = Math.min(opts?.limit || 50, 500);

    let sql = "SELECT tx, action, path, size, msg, meta, ts FROM stg_events WHERE owner = $1";
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
    if (opts?.before_tx) {
      sql += ` AND tx < $${idx}`;
      params.push(opts.before_tx);
      idx++;
    }

    sql += ` ORDER BY tx DESC, id DESC LIMIT $${idx}`;
    params.push(limit);

    const rows = await this.query<{
      tx: string; action: string; path: string; size: string;
      msg: string | null; meta: string | null; ts: string;
    }>(sql, params);

    return rows.map((r) => ({
      tx: Number(r.tx),
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
    const p = path.replace(/^\/+/, "");

    const node = await this.resolvePath(actor, p);
    if (!node) return null;

    const fcs = await this.query<{ content_hash: string | null }>(
      "SELECT content_hash FROM stg_file_current_state WHERE owner = $1 AND node_id = $2",
      [actor, node.node_id],
    );
    if (!fcs.length) return null;

    const key = fcs[0].content_hash ? blobKey(actor, fcs[0].content_hash) : `${actor}/${p}`;
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
