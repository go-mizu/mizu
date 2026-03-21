// ── D1 inode-based driver (v2) ────────────────────────────────────────
//
// Per-actor table sharding with inode-style directory tree:
//   nodes_{shard}             — node identity (stable across rename/move)
//   directory_entries_{shard} — parent→child directory links
//   file_versions_{shard}    — immutable version history per file node
//   file_current_state_{shard} — latest version projection (fast reads)
//   blob_refs_{shard}        — content-addressed blob refcounting
//   events_{shard}           — event log (all mutations)
//   transactions_{shard}     — tx registry
//   path_cache_{shard}       — denormalized path→node_id cache
//
// Shard = sha256(actor).slice(0, 16) — deterministic, no lookup needed.
// Registry: `shards` table maps actor → shard + next_tx.
//
// Blobs in R2:  blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}
//
// Key difference from DO v2: async D1 batch() instead of transactionSync.

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

interface D1V2Config {
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

/** Split "docs/readme.md" → { dir: "docs/", name: "readme.md" } */
function splitPath(path: string): { dir: string; name: string } {
  const p = path.replace(/^\/+/, "");
  const i = p.lastIndexOf("/");
  if (i === -1) return { dir: "", name: p };
  return { dir: p.slice(0, i + 1), name: p.slice(i + 1) };
}

const NID_CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-";
function nanoid(): string {
  const b = crypto.getRandomValues(new Uint8Array(21));
  let id = "";
  for (let i = 0; i < 21; i++) id += NID_CHARS[b[i] & 63];
  return id;
}

// ── Implementation ───────────────────────────────────────────────────

export class D1V2Engine implements StorageEngine {
  private db: D1Database;
  private bucket: R2Bucket;
  private r2Endpoint?: string;
  private r2AccessKeyId?: string;
  private r2SecretAccessKey?: string;
  private r2BucketName: string;

  /** Per-request cache: actor → shard hex string */
  private shardCache = new Map<string, string>();

  constructor(config: D1V2Config) {
    this.db = config.db;
    this.bucket = config.bucket;
    this.r2Endpoint = config.r2Endpoint;
    this.r2AccessKeyId = config.r2AccessKeyId;
    this.r2SecretAccessKey = config.r2SecretAccessKey;
    this.r2BucketName = config.r2BucketName || "storage-files";
  }

  // ── shard management ──────────────────────────────────────────────

  /** Compute deterministic shard from actor name (no DB lookup). */
  private async actorShard(actor: string): Promise<string> {
    const data = new TextEncoder().encode(actor);
    const hash = await crypto.subtle.digest("SHA-256", data);
    return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0"))
      .join("")
      .slice(0, 16);
  }

  /**
   * Ensure per-actor tables exist. On first access for an actor:
   * 1. Register in `shards` table
   * 2. CREATE TABLE IF NOT EXISTS for all v2 inode tables
   * 3. Seed root node
   */
  private async ensureShard(actor: string): Promise<string> {
    const cached = this.shardCache.get(actor);
    if (cached) return cached;

    // Check registry
    const row = await this.db
      .prepare("SELECT shard FROM shards WHERE actor = ?")
      .bind(actor)
      .first<{ shard: string }>();

    if (row) {
      this.shardCache.set(actor, row.shard);
      return row.shard;
    }

    // New actor — compute shard, register, create tables
    const shard = await this.actorShard(actor);
    const now = Date.now();

    // Check for v1 shards entry (migration continuity)
    const legacyTx = await this.db
      .prepare("SELECT next_tx FROM shards WHERE actor = ?")
      .bind(actor)
      .first<{ next_tx: number }>();

    // Register shard (handle race: another request may have created it)
    try {
      await this.db
        .prepare("INSERT INTO shards (actor, shard, next_tx, created_at) VALUES (?, ?, ?, ?)")
        .bind(actor, shard, legacyTx?.next_tx || 1, now)
        .run();
    } catch {
      // Race condition: another request already created this shard
      const existing = await this.db
        .prepare("SELECT shard FROM shards WHERE actor = ?")
        .bind(actor)
        .first<{ shard: string }>();
      if (existing) {
        this.shardCache.set(actor, existing.shard);
        return existing.shard;
      }
      throw new Error("Failed to create shard for actor: " + actor);
    }

    // Create per-actor inode tables + indexes
    const s = shard;
    await this.db.batch([
      this.db.prepare(
        `CREATE TABLE IF NOT EXISTS nodes_${s} (` +
          `node_id TEXT PRIMARY KEY,` +
          `kind TEXT NOT NULL CHECK(kind IN ('file','dir')),` +
          `created_at INTEGER NOT NULL,` +
          `deleted_at INTEGER` +
          `)`,
      ),
      this.db.prepare(
        `CREATE TABLE IF NOT EXISTS directory_entries_${s} (` +
          `parent_id TEXT NOT NULL,` +
          `name TEXT NOT NULL,` +
          `child_id TEXT NOT NULL,` +
          `created_tx INTEGER NOT NULL,` +
          `deleted_tx INTEGER,` +
          `PRIMARY KEY (parent_id, name)` +
          `)`,
      ),
      this.db.prepare(
        `CREATE INDEX IF NOT EXISTS idx_de_${s}_child ON directory_entries_${s}(child_id)`,
      ),
      this.db.prepare(
        `CREATE TABLE IF NOT EXISTS file_versions_${s} (` +
          `node_id TEXT NOT NULL,` +
          `version INTEGER NOT NULL,` +
          `content_hash TEXT NOT NULL,` +
          `size INTEGER NOT NULL,` +
          `content_type TEXT,` +
          `created_tx INTEGER NOT NULL,` +
          `PRIMARY KEY (node_id, version)` +
          `)`,
      ),
      this.db.prepare(
        `CREATE TABLE IF NOT EXISTS file_current_state_${s} (` +
          `node_id TEXT PRIMARY KEY,` +
          `content_hash TEXT NOT NULL,` +
          `size INTEGER NOT NULL,` +
          `content_type TEXT,` +
          `version INTEGER NOT NULL,` +
          `updated_tx INTEGER NOT NULL,` +
          `updated_at INTEGER NOT NULL` +
          `)`,
      ),
    ]);

    // Second batch (D1 limit is 100 statements per batch — split across calls)
    await this.db.batch([
      this.db.prepare(
        `CREATE TABLE IF NOT EXISTS blob_refs_${s} (` +
          `content_hash TEXT PRIMARY KEY,` +
          `size INTEGER NOT NULL,` +
          `ref_count INTEGER NOT NULL DEFAULT 1,` +
          `created_at INTEGER NOT NULL` +
          `)`,
      ),
      this.db.prepare(
        `CREATE TABLE IF NOT EXISTS events_${s} (` +
          `id INTEGER PRIMARY KEY AUTOINCREMENT,` +
          `tx INTEGER NOT NULL,` +
          `action TEXT NOT NULL CHECK(action IN ('write','move','delete')),` +
          `node_id TEXT NOT NULL,` +
          `path TEXT NOT NULL,` +
          `content_hash TEXT,` +
          `size INTEGER NOT NULL DEFAULT 0,` +
          `content_type TEXT,` +
          `meta TEXT,` +
          `msg TEXT,` +
          `ts INTEGER NOT NULL` +
          `)`,
      ),
      this.db.prepare(
        `CREATE INDEX IF NOT EXISTS idx_ev_${s}_tx ON events_${s}(tx DESC)`,
      ),
      this.db.prepare(
        `CREATE INDEX IF NOT EXISTS idx_ev_${s}_node ON events_${s}(node_id)`,
      ),
      this.db.prepare(
        `CREATE TABLE IF NOT EXISTS transactions_${s} (` +
          `tx INTEGER PRIMARY KEY, ts INTEGER NOT NULL, msg TEXT` +
          `)`,
      ),
      this.db.prepare(
        `CREATE TABLE IF NOT EXISTS path_cache_${s} (` +
          `path TEXT PRIMARY KEY,` +
          `node_id TEXT NOT NULL,` +
          `parent_id TEXT NOT NULL,` +
          `name TEXT NOT NULL,` +
          `kind TEXT NOT NULL DEFAULT 'file',` +
          `updated_tx INTEGER NOT NULL` +
          `)`,
      ),
      this.db.prepare(
        `CREATE INDEX IF NOT EXISTS idx_pc_${s}_node ON path_cache_${s}(node_id)`,
      ),
    ]);

    // Seed root node
    await this.db
      .prepare(
        `INSERT OR IGNORE INTO nodes_${s} (node_id, kind, created_at) VALUES ('root', 'dir', ?)`,
      )
      .bind(now)
      .run();

    this.shardCache.set(actor, shard);
    return shard;
  }

  // ── tx allocation ────────────────────────────────────────────────

  /** Atomic increment on shards table — returns allocated tx number. */
  private async nextTx(actor: string): Promise<number> {
    await this.db
      .prepare("UPDATE shards SET next_tx = next_tx + 1 WHERE actor = ?")
      .bind(actor)
      .run();

    const row = await this.db
      .prepare("SELECT next_tx FROM shards WHERE actor = ?")
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

  // ── path resolution ─────────────────────────────────────────────

  /**
   * Resolve a path to its node info. Returns null if not found.
   * Checks path_cache first, then walks directory_entries from root.
   */
  private async resolvePath(
    shard: string,
    path: string,
  ): Promise<{ node_id: string; parent_id: string; name: string } | null> {
    if (!path || path === "/") return { node_id: "root", parent_id: "", name: "" };
    const p = path.replace(/^\/+/, "").replace(/\/+$/, "");
    if (!p) return { node_id: "root", parent_id: "", name: "" };

    // Check cache
    const cached = await this.db
      .prepare(`SELECT node_id, parent_id, name FROM path_cache_${shard} WHERE path = ?`)
      .bind(p)
      .first<{ node_id: string; parent_id: string; name: string }>();

    if (cached) {
      // Verify node is alive
      const alive = await this.db
        .prepare(`SELECT node_id FROM nodes_${shard} WHERE node_id = ? AND deleted_at IS NULL`)
        .bind(cached.node_id)
        .first<{ node_id: string }>();

      if (alive) return cached;

      // Stale cache — remove
      await this.db
        .prepare(`DELETE FROM path_cache_${shard} WHERE path = ?`)
        .bind(p)
        .run();
    }

    // Walk from root
    const segments = p.split("/");
    let current = "root";
    let parentId = "";

    for (let i = 0; i < segments.length; i++) {
      const seg = segments[i];
      const row = await this.db
        .prepare(
          `SELECT child_id FROM directory_entries_${shard} WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL`,
        )
        .bind(current, seg)
        .first<{ child_id: string }>();

      if (!row) return null;
      parentId = current;
      current = row.child_id;
    }

    return { node_id: current, parent_id: parentId, name: segments[segments.length - 1] };
  }

  /**
   * Ensure all directories in a path exist, creating missing ones.
   * Returns the node_id of the deepest directory.
   * Collects batch statements into the provided array instead of executing immediately.
   */
  private async ensureDirChain(
    shard: string,
    dirPath: string,
    tx: number,
    now: number,
  ): Promise<{ parentId: string; stmts: D1PreparedStatement[] }> {
    const stmts: D1PreparedStatement[] = [];
    if (!dirPath) return { parentId: "root", stmts };

    const segments = dirPath
      .replace(/^\/+/, "")
      .replace(/\/+$/, "")
      .split("/")
      .filter(Boolean);
    if (!segments.length) return { parentId: "root", stmts };

    let current = "root";
    let accPath = "";

    for (const seg of segments) {
      accPath = accPath ? accPath + "/" + seg : seg;

      const row = await this.db
        .prepare(
          `SELECT child_id FROM directory_entries_${shard} WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL`,
        )
        .bind(current, seg)
        .first<{ child_id: string }>();

      if (row) {
        current = row.child_id;
      } else {
        const nid = nanoid();
        stmts.push(
          this.db
            .prepare(`INSERT INTO nodes_${shard} (node_id, kind, created_at) VALUES (?, 'dir', ?)`)
            .bind(nid, now),
        );
        stmts.push(
          this.db
            .prepare(
              `INSERT INTO directory_entries_${shard} (parent_id, name, child_id, created_tx) VALUES (?, ?, ?, ?)`,
            )
            .bind(current, seg, nid, tx),
        );
        stmts.push(
          this.db
            .prepare(
              `INSERT OR REPLACE INTO path_cache_${shard} (path, node_id, parent_id, name, kind, updated_tx) VALUES (?, ?, ?, ?, 'dir', ?)`,
            )
            .bind(accPath, nid, current, seg, tx),
        );
        current = nid;
      }
    }

    return { parentId: current, stmts };
  }

  /**
   * Recursively collect all file descendant node_ids under a directory.
   * D1 is async so we walk level by level.
   */
  private async collectDescendantFiles(
    shard: string,
    parentId: string,
  ): Promise<{ node_id: string; name: string; path: string; content_hash: string | null }[]> {
    const result: { node_id: string; name: string; path: string; content_hash: string | null }[] = [];

    const { results: children } = await this.db
      .prepare(
        `SELECT child_id, name FROM directory_entries_${shard} WHERE parent_id = ? AND deleted_tx IS NULL`,
      )
      .bind(parentId)
      .all<{ child_id: string; name: string }>();

    for (const child of children || []) {
      const node = await this.db
        .prepare(
          `SELECT kind FROM nodes_${shard} WHERE node_id = ? AND deleted_at IS NULL`,
        )
        .bind(child.child_id)
        .first<{ kind: string }>();

      if (!node) continue;

      if (node.kind === "file") {
        const fcs = await this.db
          .prepare(
            `SELECT content_hash FROM file_current_state_${shard} WHERE node_id = ?`,
          )
          .bind(child.child_id)
          .first<{ content_hash: string }>();

        result.push({
          node_id: child.child_id,
          name: child.name,
          path: "",
          content_hash: fcs?.content_hash ?? null,
        });
      } else {
        // Recurse into sub-directory
        const nested = await this.collectDescendantFiles(shard, child.child_id);
        result.push(...nested);
      }
    }

    return result;
  }

  // ── write ────────────────────────────────────────────────────────

  async write(
    actor: string,
    path: string,
    body: ArrayBuffer | ReadableStream,
    contentType: string,
    msg?: string,
  ): Promise<WriteResult> {
    const s = await this.ensureShard(actor);
    const buf = body instanceof ArrayBuffer ? body : await streamToBuffer(body);
    const addr = await sha256(buf);
    const size = buf.byteLength;
    const now = Date.now();
    const p = path.replace(/^\/+/, "");
    const { dir, name } = splitPath(p);
    const autoMsg = msg || `write ${p}`;

    // Upload blob to R2 (content-addressed dedup)
    const key = blobKey(actor, addr);
    const existing = await this.bucket.head(key);
    if (!existing) {
      await this.bucket.put(key, buf, { httpMetadata: { contentType } });
    }

    // Allocate tx
    const tx = await this.nextTx(actor);

    // Ensure directory chain exists
    const { parentId, stmts } = await this.ensureDirChain(s, dir, tx, now);

    // Check if file already exists at this path
    const existingEntry = await this.db
      .prepare(
        `SELECT child_id FROM directory_entries_${s} WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL`,
      )
      .bind(parentId, name)
      .first<{ child_id: string }>();

    if (existingEntry) {
      // Overwrite existing file
      const nodeId = existingEntry.child_id;
      const cur = await this.db
        .prepare(
          `SELECT version, content_hash FROM file_current_state_${s} WHERE node_id = ?`,
        )
        .bind(nodeId)
        .first<{ version: number; content_hash: string }>();

      const newVer = (cur?.version || 0) + 1;
      const oldHash = cur?.content_hash ?? null;

      stmts.push(
        this.db
          .prepare(
            `INSERT INTO file_versions_${s} (node_id, version, content_hash, size, content_type, created_tx) VALUES (?, ?, ?, ?, ?, ?)`,
          )
          .bind(nodeId, newVer, addr, size, contentType, tx),
      );
      stmts.push(
        this.db
          .prepare(
            `INSERT OR REPLACE INTO file_current_state_${s} (node_id, content_hash, size, content_type, version, updated_tx, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
          )
          .bind(nodeId, addr, size, contentType, newVer, tx, now),
      );
      stmts.push(
        this.db
          .prepare(
            `INSERT INTO blob_refs_${s} (content_hash, size, ref_count, created_at) VALUES (?, ?, 1, ?) ` +
              `ON CONFLICT (content_hash) DO UPDATE SET ref_count = ref_count + 1`,
          )
          .bind(addr, size, now),
      );

      // Decrement old blob ref if content changed
      if (oldHash && oldHash !== addr) {
        stmts.push(
          this.db
            .prepare(
              `UPDATE blob_refs_${s} SET ref_count = MAX(ref_count - 1, 0) WHERE content_hash = ?`,
            )
            .bind(oldHash),
        );
      }
      // Same-addr overwrite: upsert incremented, but old file already held a ref — net 0
      if (oldHash && oldHash === addr) {
        stmts.push(
          this.db
            .prepare(
              `UPDATE blob_refs_${s} SET ref_count = MAX(ref_count - 1, 0) WHERE content_hash = ?`,
            )
            .bind(addr),
        );
      }

      // Event + transaction + path cache
      stmts.push(
        this.db
          .prepare(
            `INSERT INTO events_${s} (tx, action, node_id, path, content_hash, size, content_type, msg, ts) VALUES (?, 'write', ?, ?, ?, ?, ?, ?, ?)`,
          )
          .bind(tx, nodeId, p, addr, size, contentType, autoMsg, now),
      );
      stmts.push(
        this.db
          .prepare(`INSERT INTO transactions_${s} (tx, ts, msg) VALUES (?, ?, ?)`)
          .bind(tx, now, autoMsg),
      );
      stmts.push(
        this.db
          .prepare(
            `INSERT OR REPLACE INTO path_cache_${s} (path, node_id, parent_id, name, kind, updated_tx) VALUES (?, ?, ?, ?, 'file', ?)`,
          )
          .bind(p, nodeId, parentId, name, tx),
      );
    } else {
      // New file
      const nodeId = nanoid();

      stmts.push(
        this.db
          .prepare(
            `INSERT INTO nodes_${s} (node_id, kind, created_at) VALUES (?, 'file', ?)`,
          )
          .bind(nodeId, now),
      );
      stmts.push(
        this.db
          .prepare(
            `INSERT INTO directory_entries_${s} (parent_id, name, child_id, created_tx) VALUES (?, ?, ?, ?)`,
          )
          .bind(parentId, name, nodeId, tx),
      );
      stmts.push(
        this.db
          .prepare(
            `INSERT INTO file_versions_${s} (node_id, version, content_hash, size, content_type, created_tx) VALUES (?, 1, ?, ?, ?, ?)`,
          )
          .bind(nodeId, addr, size, contentType, tx),
      );
      stmts.push(
        this.db
          .prepare(
            `INSERT INTO file_current_state_${s} (node_id, content_hash, size, content_type, version, updated_tx, updated_at) VALUES (?, ?, ?, ?, 1, ?, ?)`,
          )
          .bind(nodeId, addr, size, contentType, tx, now),
      );
      stmts.push(
        this.db
          .prepare(
            `INSERT INTO blob_refs_${s} (content_hash, size, ref_count, created_at) VALUES (?, ?, 1, ?) ` +
              `ON CONFLICT (content_hash) DO UPDATE SET ref_count = ref_count + 1`,
          )
          .bind(addr, size, now),
      );

      // Event + transaction + path cache
      stmts.push(
        this.db
          .prepare(
            `INSERT INTO events_${s} (tx, action, node_id, path, content_hash, size, content_type, msg, ts) VALUES (?, 'write', ?, ?, ?, ?, ?, ?, ?)`,
          )
          .bind(tx, nodeId, p, addr, size, contentType, autoMsg, now),
      );
      stmts.push(
        this.db
          .prepare(`INSERT INTO transactions_${s} (tx, ts, msg) VALUES (?, ?, ?)`)
          .bind(tx, now, autoMsg),
      );
      stmts.push(
        this.db
          .prepare(
            `INSERT OR REPLACE INTO path_cache_${s} (path, node_id, parent_id, name, kind, updated_tx) VALUES (?, ?, ?, ?, 'file', ?)`,
          )
          .bind(p, nodeId, parentId, name, tx),
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
    const s = await this.ensureShard(actor);
    const pFrom = from.replace(/^\/+/, "");
    const pTo = to.replace(/^\/+/, "");
    const autoMsg = msg || `move ${pFrom} → ${pTo}`;

    const src = await this.resolvePath(s, pFrom);
    if (!src) throw new Error("Source not found: " + pFrom);

    const { dir: dstDir, name: dstName } = splitPath(pTo);

    // Get file info for event
    const fcs = await this.db
      .prepare(
        `SELECT content_hash, size, content_type FROM file_current_state_${s} WHERE node_id = ?`,
      )
      .bind(src.node_id)
      .first<{ content_hash: string; size: number; content_type: string }>();
    const fileInfo = fcs ?? { content_hash: null, size: 0, content_type: "application/octet-stream" };

    const now = Date.now();
    const tx = await this.nextTx(actor);

    // Ensure destination directory chain
    const { parentId: dstParentId, stmts } = await this.ensureDirChain(s, dstDir, tx, now);

    // Determine if this is a dir or file for cache invalidation
    const nodeKind = await this.db
      .prepare(`SELECT kind FROM nodes_${s} WHERE node_id = ?`)
      .bind(src.node_id)
      .first<{ kind: string }>();
    const isDir = nodeKind?.kind === "dir";

    // Unlink from old parent
    stmts.push(
      this.db
        .prepare(
          `UPDATE directory_entries_${s} SET deleted_tx = ? WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL`,
        )
        .bind(tx, src.parent_id, src.name),
    );

    // Link into new parent
    stmts.push(
      this.db
        .prepare(
          `INSERT INTO directory_entries_${s} (parent_id, name, child_id, created_tx) VALUES (?, ?, ?, ?)`,
        )
        .bind(dstParentId, dstName, src.node_id, tx),
    );

    const meta = JSON.stringify({ from: pFrom });
    stmts.push(
      this.db
        .prepare(
          `INSERT INTO events_${s} (tx, action, node_id, path, size, meta, msg, ts) VALUES (?, 'move', ?, ?, ?, ?, ?, ?)`,
        )
        .bind(tx, src.node_id, pTo, fileInfo.size, meta, autoMsg, now),
    );
    stmts.push(
      this.db
        .prepare(`INSERT INTO transactions_${s} (tx, ts, msg) VALUES (?, ?, ?)`)
        .bind(tx, now, autoMsg),
    );

    // Invalidate old path cache
    stmts.push(
      this.db
        .prepare(`DELETE FROM path_cache_${s} WHERE path = ?`)
        .bind(pFrom),
    );
    if (isDir) {
      stmts.push(
        this.db
          .prepare(`DELETE FROM path_cache_${s} WHERE path LIKE ?`)
          .bind(pFrom.replace(/\/$/, "") + "/%"),
      );
    }

    // Insert new path cache entry
    stmts.push(
      this.db
        .prepare(
          `INSERT OR REPLACE INTO path_cache_${s} (path, node_id, parent_id, name, kind, updated_tx) VALUES (?, ?, ?, ?, ?, ?)`,
        )
        .bind(
          pTo.replace(/\/$/, ""),
          src.node_id,
          dstParentId,
          dstName,
          isDir ? "dir" : "file",
          tx,
        ),
    );

    await this.db.batch(stmts);

    return { tx, time: now };
  }

  // ── delete ───────────────────────────────────────────────────────

  async delete(
    actor: string,
    paths: string[],
    msg?: string,
  ): Promise<DeleteResult> {
    const s = await this.ensureShard(actor);
    const now = Date.now();
    const tx = await this.nextTx(actor);
    let deleted = 0;

    const stmts: D1PreparedStatement[] = [];

    for (const rawPath of paths) {
      const p = rawPath.replace(/^\/+/, "");

      if (p.endsWith("/")) {
        // Directory delete — find the directory node and collect descendants
        const dirNode = await this.resolvePath(s, p.replace(/\/$/, ""));
        if (!dirNode) continue;

        const fileNodes = await this.collectDescendantFiles(s, dirNode.node_id);
        for (const fn of fileNodes) {
          stmts.push(
            this.db
              .prepare(`UPDATE nodes_${s} SET deleted_at = ? WHERE node_id = ?`)
              .bind(now, fn.node_id),
          );
          if (fn.content_hash) {
            stmts.push(
              this.db
                .prepare(
                  `UPDATE blob_refs_${s} SET ref_count = MAX(ref_count - 1, 0) WHERE content_hash = ?`,
                )
                .bind(fn.content_hash),
            );
          }
          stmts.push(
            this.db
              .prepare(
                `INSERT INTO events_${s} (tx, action, node_id, path, size, msg, ts) VALUES (?, 'delete', ?, ?, 0, ?, ?)`,
              )
              .bind(tx, fn.node_id, fn.path || p + fn.name, msg || `delete ${p}*`, now),
          );
          deleted++;
        }

        // Soft-delete the directory node and unlink
        stmts.push(
          this.db
            .prepare(`UPDATE nodes_${s} SET deleted_at = ? WHERE node_id = ?`)
            .bind(now, dirNode.node_id),
        );
        stmts.push(
          this.db
            .prepare(
              `UPDATE directory_entries_${s} SET deleted_tx = ? WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL`,
            )
            .bind(tx, dirNode.parent_id, dirNode.name),
        );

        // Invalidate cache
        stmts.push(
          this.db
            .prepare(`DELETE FROM path_cache_${s} WHERE path LIKE ?`)
            .bind(p + "%"),
        );
        stmts.push(
          this.db
            .prepare(`DELETE FROM path_cache_${s} WHERE path = ?`)
            .bind(p.replace(/\/$/, "")),
        );
      } else {
        // Single file delete
        const node = await this.resolvePath(s, p);
        if (!node) {
          deleted++;
          continue;
        }

        const fcs = await this.db
          .prepare(
            `SELECT content_hash FROM file_current_state_${s} WHERE node_id = ?`,
          )
          .bind(node.node_id)
          .first<{ content_hash: string }>();

        stmts.push(
          this.db
            .prepare(`UPDATE nodes_${s} SET deleted_at = ? WHERE node_id = ?`)
            .bind(now, node.node_id),
        );
        stmts.push(
          this.db
            .prepare(
              `UPDATE directory_entries_${s} SET deleted_tx = ? WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL`,
            )
            .bind(tx, node.parent_id, node.name),
        );
        if (fcs?.content_hash) {
          stmts.push(
            this.db
              .prepare(
                `UPDATE blob_refs_${s} SET ref_count = MAX(ref_count - 1, 0) WHERE content_hash = ?`,
              )
              .bind(fcs.content_hash),
          );
        }
        stmts.push(
          this.db
            .prepare(
              `INSERT INTO events_${s} (tx, action, node_id, path, size, msg, ts) VALUES (?, 'delete', ?, ?, 0, ?, ?)`,
            )
            .bind(tx, node.node_id, p, msg || `delete ${p}`, now),
        );
        stmts.push(
          this.db
            .prepare(`DELETE FROM path_cache_${s} WHERE path = ?`)
            .bind(p),
        );
        deleted++;
      }
    }

    // Transaction record
    stmts.push(
      this.db
        .prepare(`INSERT INTO transactions_${s} (tx, ts, msg) VALUES (?, ?, ?)`)
        .bind(tx, now, msg || "delete"),
    );

    if (stmts.length) {
      // Respect D1 batch limit of 100 — split into chunks
      for (let i = 0; i < stmts.length; i += 100) {
        await this.db.batch(stmts.slice(i, i + 100));
      }
    }

    return { tx, time: now, deleted };
  }

  // ── read ─────────────────────────────────────────────────────────

  async read(actor: string, path: string): Promise<ReadResult | null> {
    const s = await this.ensureShard(actor);
    const p = path.replace(/^\/+/, "");

    const node = await this.resolvePath(s, p);
    if (!node) return null;

    const fcs = await this.db
      .prepare(
        `SELECT content_hash, size, content_type, updated_tx, updated_at FROM file_current_state_${s} WHERE node_id = ?`,
      )
      .bind(node.node_id)
      .first<{
        content_hash: string;
        size: number;
        content_type: string;
        updated_tx: number;
        updated_at: number;
      }>();
    if (!fcs) return null;

    const meta: FileMeta = {
      path: p,
      name: node.name,
      size: fcs.size,
      type: fcs.content_type,
      tx: fcs.updated_tx,
      tx_time: fcs.updated_at,
    };

    // Content-addressed read
    if (fcs.content_hash) {
      const key = blobKey(actor, fcs.content_hash);
      const obj = await this.bucket.get(key);
      if (obj) return { body: obj.body, meta };

      // Fallback to legacy path
      const legacy = await this.bucket.get(`${actor}/${p}`);
      if (!legacy) return null;
      return { body: legacy.body, meta };
    }

    // Legacy: no content_hash yet
    const obj = await this.bucket.get(`${actor}/${p}`);
    if (!obj) return null;
    return { body: obj.body, meta };
  }

  // ── head ─────────────────────────────────────────────────────────

  async head(actor: string, path: string): Promise<FileMeta | null> {
    const s = await this.ensureShard(actor);
    const p = path.replace(/^\/+/, "");

    const node = await this.resolvePath(s, p);
    if (!node) return null;

    const fcs = await this.db
      .prepare(
        `SELECT size, content_type, updated_tx, updated_at FROM file_current_state_${s} WHERE node_id = ?`,
      )
      .bind(node.node_id)
      .first<{ size: number; content_type: string; updated_tx: number; updated_at: number }>();
    if (!fcs) return null;

    return {
      path: p,
      name: node.name,
      size: fcs.size,
      type: fcs.content_type,
      tx: fcs.updated_tx,
      tx_time: fcs.updated_at,
    };
  }

  // ── list ─────────────────────────────────────────────────────────

  async list(
    actor: string,
    opts?: ListOptions,
  ): Promise<{ entries: FileEntry[]; truncated: boolean }> {
    const s = await this.ensureShard(actor);
    const prefix = (opts?.prefix || "").replace(/^\/+/, "");
    const limit = Math.min(opts?.limit || 200, 1000);
    const offset = opts?.offset || 0;

    // Resolve prefix to a directory node
    let parentId = "root";
    if (prefix) {
      const dirPath = prefix.replace(/\/$/, "");
      const node = await this.resolvePath(s, dirPath);
      if (!node) return { entries: [], truncated: false };
      parentId = node.node_id;
    }

    // List children of the directory
    const { results: children } = await this.db
      .prepare(
        `SELECT child_id, name FROM directory_entries_${s} WHERE parent_id = ? AND deleted_tx IS NULL ORDER BY name LIMIT ? OFFSET ?`,
      )
      .bind(parentId, limit + 1, offset)
      .all<{ child_id: string; name: string }>();

    const rows = children || [];
    const truncated = rows.length > limit;
    if (truncated) rows.pop();

    const entries: FileEntry[] = [];
    for (const child of rows) {
      const node = await this.db
        .prepare(
          `SELECT kind FROM nodes_${s} WHERE node_id = ? AND deleted_at IS NULL`,
        )
        .bind(child.child_id)
        .first<{ kind: string }>();
      if (!node) continue;

      if (node.kind === "dir") {
        entries.push({ name: child.name + "/", type: "directory" });
      } else {
        const fcs = await this.db
          .prepare(
            `SELECT size, content_type, updated_at, updated_tx FROM file_current_state_${s} WHERE node_id = ?`,
          )
          .bind(child.child_id)
          .first<{
            size: number;
            content_type: string;
            updated_at: number;
            updated_tx: number;
          }>();
        if (fcs) {
          entries.push({
            name: child.name,
            type: fcs.content_type,
            size: fcs.size,
            updated_at: fcs.updated_at,
            tx: fcs.updated_tx,
            tx_time: fcs.updated_at,
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
    const s = await this.ensureShard(actor);
    const limit = Math.min(opts?.limit || 50, 200);

    let sql = `SELECT path, name, node_id FROM path_cache_${s} WHERE kind = 'file' AND (name LIKE ? OR path LIKE ?)`;
    const binds: any[] = [`%${query}%`, `%${query}%`];

    if (opts?.prefix) {
      sql += " AND path LIKE ?";
      binds.push(`${opts.prefix}%`);
    }

    sql += " LIMIT ?";
    binds.push(limit);

    const { results } = await this.db.prepare(sql).bind(...binds).all<{
      path: string;
      name: string;
      node_id: string;
    }>();

    const searchResults: SearchResult[] = [];
    for (const r of results || []) {
      const fcs = await this.db
        .prepare(
          `SELECT size, content_type, updated_tx FROM file_current_state_${s} WHERE node_id = ?`,
        )
        .bind(r.node_id)
        .first<{ size: number; content_type: string; updated_tx: number }>();
      if (fcs) {
        searchResults.push({
          path: r.path,
          name: r.name,
          size: fcs.size,
          type: fcs.content_type,
          tx: fcs.updated_tx,
        });
      }
    }

    return searchResults;
  }

  // ── stats ────────────────────────────────────────────────────────

  async stats(actor: string): Promise<{ files: number; bytes: number }> {
    const s = await this.ensureShard(actor);

    const row = await this.db
      .prepare(
        `SELECT COUNT(*) as files, COALESCE(SUM(size), 0) as bytes FROM file_current_state_${s}`,
      )
      .first<{ files: number; bytes: number }>();

    return { files: row?.files || 0, bytes: row?.bytes || 0 };
  }

  // ── allNames ─────────────────────────────────────────────────────

  async allNames(actor: string): Promise<{ path: string; name: string }[]> {
    const s = await this.ensureShard(actor);

    const { results } = await this.db
      .prepare(`SELECT path, name FROM path_cache_${s} WHERE kind = 'file'`)
      .all<{ path: string; name: string }>();

    return (results || []).map((r) => ({ path: r.path, name: r.name }));
  }

  // ── log ──────────────────────────────────────────────────────────

  async log(actor: string, opts?: LogOptions): Promise<StorageEvent[]> {
    const s = await this.ensureShard(actor);
    const limit = Math.min(opts?.limit || 50, 500);

    let sql = `SELECT tx, action, path, size, msg, meta, ts FROM events_${s} WHERE 1=1`;
    const binds: any[] = [];

    if (opts?.path) {
      sql += " AND path = ?";
      binds.push(opts.path);
    }
    if (opts?.since_tx) {
      sql += " AND tx > ?";
      binds.push(opts.since_tx);
    }
    if (opts?.before_tx) {
      sql += " AND tx < ?";
      binds.push(opts.before_tx);
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

    const s = await this.ensureShard(actor);
    const p = path.replace(/^\/+/, "");

    const node = await this.resolvePath(s, p);
    if (!node) return null;

    const fcs = await this.db
      .prepare(
        `SELECT content_hash FROM file_current_state_${s} WHERE node_id = ?`,
      )
      .bind(node.node_id)
      .first<{ content_hash: string }>();

    if (!fcs) return null;

    const key = fcs.content_hash ? blobKey(actor, fcs.content_hash) : `${actor}/${p}`;
    return this.presign("GET", key, expiresIn);
  }

  // ── presign upload ───────────────────────────────────────────────

  async presignUpload(
    actor: string,
    path: string,
    contentType: string,
    expiresIn = 3600,
  ): Promise<string> {
    // Ensure shard exists (creates tables if needed for new actor)
    await this.ensureShard(actor);

    // Presigned uploads go to legacy path — confirmUpload will content-address them
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
    const contentType =
      obj.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);

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

    // Ensure shard exists
    await this.ensureShard(actor);

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
    await mpu.complete(
      parts.map((p) => ({ partNumber: p.part_number, etag: p.etag })),
    );

    // Now content-address the assembled object
    const obj = await this.bucket.get(key);
    if (!obj) throw new Error("Multipart upload not found after completion");

    const buf = await obj.arrayBuffer();
    const contentType =
      obj.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);

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

export default D1V2Engine;
