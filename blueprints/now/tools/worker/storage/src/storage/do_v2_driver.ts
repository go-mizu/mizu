// ── Durable Object inode-based driver (v2) ────────────────────────────
//
// Each actor gets a dedicated DO with local SQLite.
// node_id = identity (stable across rename/move).
// path = derived from directory_entries tree + path_cache projection.
//
// Key win: directory move/rename is O(1) in source of truth.

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

/** Split "docs/readme.md" → { dir: "docs/", name: "readme.md" } */
function splitPath(path: string): { dir: string; name: string } {
  const p = path.replace(/^\/+/, "");
  const i = p.lastIndexOf("/");
  if (i === -1) return { dir: "", name: p };
  return { dir: p.slice(0, i + 1), name: p.slice(i + 1) };
}

// ── Durable Object class ─────────────────────────────────────────────

interface DOEnv {
  BUCKET: R2Bucket;
  [key: string]: unknown;
}

export class StorageDOv2 extends DurableObject<DOEnv> {
  private ready = false;

  private ensureSchema(): void {
    if (this.ready) return;
    const sql = this.ctx.storage.sql;

    sql.exec(`CREATE TABLE IF NOT EXISTS nodes (
      node_id TEXT PRIMARY KEY,
      kind TEXT NOT NULL CHECK(kind IN ('file','dir')),
      created_at INTEGER NOT NULL,
      deleted_at INTEGER
    )`);

    sql.exec(`CREATE TABLE IF NOT EXISTS directory_entries (
      parent_id TEXT NOT NULL,
      name TEXT NOT NULL,
      child_id TEXT NOT NULL,
      created_tx INTEGER NOT NULL,
      deleted_tx INTEGER,
      PRIMARY KEY (parent_id, name)
    )`);
    sql.exec(`CREATE INDEX IF NOT EXISTS idx_de_child ON directory_entries(child_id)`);

    sql.exec(`CREATE TABLE IF NOT EXISTS file_versions (
      node_id TEXT NOT NULL,
      version INTEGER NOT NULL,
      content_hash TEXT NOT NULL,
      size INTEGER NOT NULL,
      content_type TEXT,
      created_tx INTEGER NOT NULL,
      PRIMARY KEY (node_id, version)
    )`);

    sql.exec(`CREATE TABLE IF NOT EXISTS file_current_state (
      node_id TEXT PRIMARY KEY,
      content_hash TEXT NOT NULL,
      size INTEGER NOT NULL,
      content_type TEXT,
      version INTEGER NOT NULL,
      updated_tx INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )`);

    sql.exec(`CREATE TABLE IF NOT EXISTS blob_refs (
      content_hash TEXT PRIMARY KEY,
      size INTEGER NOT NULL,
      ref_count INTEGER NOT NULL DEFAULT 1,
      created_at INTEGER NOT NULL
    )`);

    sql.exec(`CREATE TABLE IF NOT EXISTS events (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      tx INTEGER NOT NULL,
      action TEXT NOT NULL CHECK(action IN ('write','move','delete')),
      node_id TEXT NOT NULL,
      path TEXT NOT NULL,
      content_hash TEXT,
      size INTEGER NOT NULL DEFAULT 0,
      content_type TEXT,
      meta TEXT,
      msg TEXT,
      ts INTEGER NOT NULL
    )`);
    sql.exec(`CREATE INDEX IF NOT EXISTS idx_events_tx ON events(tx DESC)`);
    sql.exec(`CREATE INDEX IF NOT EXISTS idx_events_node ON events(node_id)`);

    sql.exec(`CREATE TABLE IF NOT EXISTS transactions (
      tx INTEGER PRIMARY KEY, ts INTEGER NOT NULL, msg TEXT
    )`);

    sql.exec(`CREATE TABLE IF NOT EXISTS path_cache (
      path TEXT PRIMARY KEY,
      node_id TEXT NOT NULL,
      parent_id TEXT NOT NULL,
      name TEXT NOT NULL,
      kind TEXT NOT NULL DEFAULT 'file',
      updated_tx INTEGER NOT NULL
    )`);
    sql.exec(`CREATE INDEX IF NOT EXISTS idx_pc_node ON path_cache(node_id)`);

    sql.exec(`CREATE TABLE IF NOT EXISTS meta (
      key TEXT PRIMARY KEY, value TEXT NOT NULL
    )`);
    sql.exec(`INSERT OR IGNORE INTO meta (key, value) VALUES ('next_tx', '0')`);
    sql.exec(`INSERT OR IGNORE INTO nodes (node_id, kind, created_at) VALUES ('root', 'dir', ${Date.now()})`);

    this.ready = true;
  }

  private nextTx(): number {
    const sql = this.ctx.storage.sql;
    sql.exec("UPDATE meta SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'next_tx'");
    return parseInt([...sql.exec<{ value: string }>("SELECT value FROM meta WHERE key = 'next_tx'")][0].value, 10);
  }

  // ── Path resolution ────────────────────────────────────────────────

  /** Resolve a path to node info. Returns null if not found. */
  private resolve(path: string): { node_id: string; parent_id: string; name: string } | null {
    if (!path || path === "/") return { node_id: "root", parent_id: "", name: "" };
    const p = path.replace(/^\/+/, "").replace(/\/+$/, "");
    if (!p) return { node_id: "root", parent_id: "", name: "" };

    const sql = this.ctx.storage.sql;

    // Check cache
    const cached = [...sql.exec<{ node_id: string; parent_id: string; name: string }>(
      "SELECT node_id, parent_id, name FROM path_cache WHERE path = ?", p,
    )];
    if (cached.length) {
      // Verify node is alive
      const alive = [...sql.exec<{ node_id: string }>(
        "SELECT node_id FROM nodes WHERE node_id = ? AND deleted_at IS NULL", cached[0].node_id,
      )];
      if (alive.length) return cached[0];
      // Stale cache
      sql.exec("DELETE FROM path_cache WHERE path = ?", p);
    }

    // Walk from root
    const segments = p.split("/");
    let current = "root";
    let parentId = "";

    for (let i = 0; i < segments.length; i++) {
      const seg = segments[i];
      const rows = [...sql.exec<{ child_id: string }>(
        "SELECT child_id FROM directory_entries WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL",
        current, seg,
      )];
      if (!rows.length) return null;
      parentId = current;
      current = rows[0].child_id;
    }

    return { node_id: current, parent_id: parentId, name: segments[segments.length - 1] };
  }

  /** Ensure all directories in a path exist, creating as needed. Returns deepest dir node_id. */
  private ensureDirChain(dirPath: string, tx: number, now: number): string {
    if (!dirPath) return "root";
    const segments = dirPath.replace(/^\/+/, "").replace(/\/+$/, "").split("/").filter(Boolean);
    if (!segments.length) return "root";

    const sql = this.ctx.storage.sql;
    let current = "root";
    let accPath = "";

    for (const seg of segments) {
      accPath = accPath ? accPath + "/" + seg : seg;

      const rows = [...sql.exec<{ child_id: string }>(
        "SELECT child_id FROM directory_entries WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL",
        current, seg,
      )];

      if (rows.length) {
        current = rows[0].child_id;
      } else {
        const nid = nanoid();
        sql.exec("INSERT INTO nodes (node_id, kind, created_at) VALUES (?, 'dir', ?)", nid, now);
        sql.exec(
          "INSERT INTO directory_entries (parent_id, name, child_id, created_tx) VALUES (?, ?, ?, ?)",
          current, seg, nid, tx,
        );
        sql.exec(
          "INSERT OR REPLACE INTO path_cache (path, node_id, parent_id, name, kind, updated_tx) VALUES (?, ?, ?, ?, 'dir', ?)",
          accPath, nid, current, seg, tx,
        );
        current = nid;
      }
    }
    return current;
  }

  // ── RPC methods ────────────────────────────────────────────────────

  async recordWrite(
    path: string,
    addr: string,
    size: number,
    contentType: string,
    msg?: string,
  ): Promise<WriteResult> {
    this.ensureSchema();
    const sql = this.ctx.storage.sql;
    const p = path.replace(/^\/+/, "");
    const { dir, name } = splitPath(p);
    const autoMsg = msg || `write ${p}`;

    let tx!: number;
    const now = Date.now();

    this.ctx.storage.transactionSync(() => {
      tx = this.nextTx();
      const parentId = this.ensureDirChain(dir, tx, now);

      // Check if file exists
      const existing = [...sql.exec<{ child_id: string }>(
        "SELECT child_id FROM directory_entries WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL",
        parentId, name,
      )];

      let nodeId: string;
      if (existing.length) {
        nodeId = existing[0].child_id;
        const cur = [...sql.exec<{ version: number; content_hash: string }>(
          "SELECT version, content_hash FROM file_current_state WHERE node_id = ?", nodeId,
        )];
        const newVer = (cur[0]?.version || 0) + 1;
        const oldHash = cur[0]?.content_hash ?? null;

        sql.exec(
          "INSERT INTO file_versions (node_id, version, content_hash, size, content_type, created_tx) VALUES (?, ?, ?, ?, ?, ?)",
          nodeId, newVer, addr, size, contentType, tx,
        );
        sql.exec(
          `INSERT OR REPLACE INTO file_current_state (node_id, content_hash, size, content_type, version, updated_tx, updated_at)
           VALUES (?, ?, ?, ?, ?, ?, ?)`,
          nodeId, addr, size, contentType, newVer, tx, now,
        );
        sql.exec(
          `INSERT INTO blob_refs (content_hash, size, ref_count, created_at) VALUES (?, ?, 1, ?)
           ON CONFLICT (content_hash) DO UPDATE SET ref_count = ref_count + 1`,
          addr, size, now,
        );
        if (oldHash && oldHash !== addr) {
          sql.exec("UPDATE blob_refs SET ref_count = MAX(ref_count - 1, 0) WHERE content_hash = ?", oldHash);
        }
        if (oldHash && oldHash === addr) {
          sql.exec("UPDATE blob_refs SET ref_count = MAX(ref_count - 1, 0) WHERE content_hash = ?", addr);
        }
      } else {
        nodeId = nanoid();
        sql.exec("INSERT INTO nodes (node_id, kind, created_at) VALUES (?, 'file', ?)", nodeId, now);
        sql.exec(
          "INSERT INTO directory_entries (parent_id, name, child_id, created_tx) VALUES (?, ?, ?, ?)",
          parentId, name, nodeId, tx,
        );
        sql.exec(
          "INSERT INTO file_versions (node_id, version, content_hash, size, content_type, created_tx) VALUES (?, 1, ?, ?, ?, ?)",
          nodeId, addr, size, contentType, tx,
        );
        sql.exec(
          "INSERT INTO file_current_state (node_id, content_hash, size, content_type, version, updated_tx, updated_at) VALUES (?, ?, ?, ?, 1, ?, ?)",
          nodeId, addr, size, contentType, tx, now,
        );
        sql.exec(
          `INSERT INTO blob_refs (content_hash, size, ref_count, created_at) VALUES (?, ?, 1, ?)
           ON CONFLICT (content_hash) DO UPDATE SET ref_count = ref_count + 1`,
          addr, size, now,
        );
      }

      sql.exec(
        "INSERT INTO events (tx, action, node_id, path, content_hash, size, content_type, msg, ts) VALUES (?, 'write', ?, ?, ?, ?, ?, ?, ?)",
        tx, nodeId, p, addr, size, contentType, autoMsg, now,
      );
      sql.exec("INSERT INTO transactions (tx, ts, msg) VALUES (?, ?, ?)", tx, now, autoMsg);
      sql.exec(
        "INSERT OR REPLACE INTO path_cache (path, node_id, parent_id, name, kind, updated_tx) VALUES (?, ?, ?, ?, 'file', ?)",
        p, nodeId, parentId, name, tx,
      );
    });

    return { tx, time: now, size };
  }

  async recordMove(
    from: string,
    to: string,
    msg?: string,
  ): Promise<MutationResult & { content_hash: string | null; size: number; type: string }> {
    this.ensureSchema();
    const sql = this.ctx.storage.sql;
    const pFrom = from.replace(/^\/+/, "");
    const pTo = to.replace(/^\/+/, "");
    const autoMsg = msg || `move ${pFrom} → ${pTo}`;

    const src = this.resolve(pFrom);
    if (!src) throw new Error("Source not found: " + pFrom);

    const { dir: dstDir, name: dstName } = splitPath(pTo);

    // Get file info for return value
    const fcs = [...sql.exec<{ content_hash: string; size: number; content_type: string }>(
      "SELECT content_hash, size, content_type FROM file_current_state WHERE node_id = ?", src.node_id,
    )];
    const fileInfo = fcs[0] ?? { content_hash: null, size: 0, content_type: "application/octet-stream" };

    let tx!: number;
    const now = Date.now();

    this.ctx.storage.transactionSync(() => {
      tx = this.nextTx();
      const dstParentId = this.ensureDirChain(dstDir, tx, now);

      // Unlink from old parent
      sql.exec(
        "UPDATE directory_entries SET deleted_tx = ? WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL",
        tx, src.parent_id, src.name,
      );

      // Link into new parent
      sql.exec(
        `INSERT INTO directory_entries (parent_id, name, child_id, created_tx) VALUES (?, ?, ?, ?)`,
        dstParentId, dstName, src.node_id, tx,
      );

      const meta = JSON.stringify({ from: pFrom });
      sql.exec(
        "INSERT INTO events (tx, action, node_id, path, size, meta, msg, ts) VALUES (?, 'move', ?, ?, ?, ?, ?, ?)",
        tx, src.node_id, pTo, fileInfo.size, meta, autoMsg, now,
      );
      sql.exec("INSERT INTO transactions (tx, ts, msg) VALUES (?, ?, ?)", tx, now, autoMsg);

      // Invalidate path cache
      const isDir = [...sql.exec<{ kind: string }>("SELECT kind FROM nodes WHERE node_id = ?", src.node_id)][0]?.kind === "dir";
      sql.exec("DELETE FROM path_cache WHERE path = ?", pFrom);
      if (isDir) {
        sql.exec("DELETE FROM path_cache WHERE path LIKE ?", pFrom.replace(/\/$/, "") + "/%");
      }
      sql.exec(
        "INSERT OR REPLACE INTO path_cache (path, node_id, parent_id, name, kind, updated_tx) VALUES (?, ?, ?, ?, ?, ?)",
        pTo.replace(/\/$/, ""), src.node_id, dstParentId, dstName, isDir ? "dir" : "file", tx,
      );
    });

    return { tx, time: now, content_hash: fileInfo.content_hash, size: fileInfo.size, type: fileInfo.content_type };
  }

  async recordDelete(paths: string[], msg?: string): Promise<DeleteResult> {
    this.ensureSchema();
    const sql = this.ctx.storage.sql;
    const now = Date.now();
    let deleted = 0;

    let tx!: number;
    this.ctx.storage.transactionSync(() => {
      tx = this.nextTx();

      for (const rawPath of paths) {
        const p = rawPath.replace(/^\/+/, "");

        if (p.endsWith("/")) {
          // Directory delete — find all files under this prefix via path_cache + walk
          const dirNode = this.resolve(p.replace(/\/$/, ""));
          if (!dirNode) continue;

          // Collect all file nodes under this directory recursively
          const fileNodes = this.collectDescendantFiles(dirNode.node_id);
          for (const fn of fileNodes) {
            sql.exec("UPDATE nodes SET deleted_at = ? WHERE node_id = ?", now, fn.node_id);
            if (fn.content_hash) {
              sql.exec("UPDATE blob_refs SET ref_count = MAX(ref_count - 1, 0) WHERE content_hash = ?", fn.content_hash);
            }
            sql.exec(
              "INSERT INTO events (tx, action, node_id, path, size, msg, ts) VALUES (?, 'delete', ?, ?, 0, ?, ?)",
              tx, fn.node_id, fn.path || p + fn.name, msg || `delete ${p}*`, now,
            );
            deleted++;
          }

          // Soft-delete the directory and unlink
          sql.exec("UPDATE nodes SET deleted_at = ? WHERE node_id = ?", now, dirNode.node_id);
          sql.exec(
            "UPDATE directory_entries SET deleted_tx = ? WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL",
            tx, dirNode.parent_id, dirNode.name,
          );
          // Invalidate cache
          sql.exec("DELETE FROM path_cache WHERE path LIKE ?", p + "%");
          sql.exec("DELETE FROM path_cache WHERE path = ?", p.replace(/\/$/, ""));
        } else {
          const node = this.resolve(p);
          if (!node) { deleted++; continue; }

          const fcs = [...sql.exec<{ content_hash: string }>(
            "SELECT content_hash FROM file_current_state WHERE node_id = ?", node.node_id,
          )];

          sql.exec("UPDATE nodes SET deleted_at = ? WHERE node_id = ?", now, node.node_id);
          sql.exec(
            "UPDATE directory_entries SET deleted_tx = ? WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL",
            tx, node.parent_id, node.name,
          );
          if (fcs[0]?.content_hash) {
            sql.exec("UPDATE blob_refs SET ref_count = MAX(ref_count - 1, 0) WHERE content_hash = ?", fcs[0].content_hash);
          }
          sql.exec(
            "INSERT INTO events (tx, action, node_id, path, size, msg, ts) VALUES (?, 'delete', ?, ?, 0, ?, ?)",
            tx, node.node_id, p, msg || `delete ${p}`, now,
          );
          sql.exec("DELETE FROM path_cache WHERE path = ?", p);
          deleted++;
        }
      }

      sql.exec("INSERT INTO transactions (tx, ts, msg) VALUES (?, ?, ?)", tx, now, msg || "delete");
    });

    return { tx, time: now, deleted };
  }

  /** Recursively collect file descendant info under a directory node. */
  private collectDescendantFiles(parentId: string): { node_id: string; name: string; path: string; content_hash: string | null }[] {
    const sql = this.ctx.storage.sql;
    const result: { node_id: string; name: string; path: string; content_hash: string | null }[] = [];

    const children = [...sql.exec<{ child_id: string; name: string }>(
      "SELECT child_id, name FROM directory_entries WHERE parent_id = ? AND deleted_tx IS NULL", parentId,
    )];

    for (const child of children) {
      const kind = [...sql.exec<{ kind: string }>("SELECT kind FROM nodes WHERE node_id = ? AND deleted_at IS NULL", child.child_id)];
      if (!kind.length) continue;

      if (kind[0].kind === "file") {
        const fcs = [...sql.exec<{ content_hash: string }>(
          "SELECT content_hash FROM file_current_state WHERE node_id = ?", child.child_id,
        )];
        result.push({ node_id: child.child_id, name: child.name, path: "", content_hash: fcs[0]?.content_hash ?? null });
      } else {
        result.push(...this.collectDescendantFiles(child.child_id));
      }
    }

    return result;
  }

  async getFileAddr(path: string): Promise<{ addr: string | null; meta: FileMeta } | null> {
    this.ensureSchema();
    const p = path.replace(/^\/+/, "");
    const node = this.resolve(p);
    if (!node) return null;

    const sql = this.ctx.storage.sql;
    const fcs = [...sql.exec<{
      content_hash: string; size: number; content_type: string; updated_tx: number; updated_at: number;
    }>("SELECT content_hash, size, content_type, updated_tx, updated_at FROM file_current_state WHERE node_id = ?", node.node_id)];
    if (!fcs.length) return null;

    const f = fcs[0];
    return {
      addr: f.content_hash,
      meta: { path: p, name: node.name, size: f.size, type: f.content_type, tx: f.updated_tx, tx_time: f.updated_at },
    };
  }

  async getFileMeta(path: string): Promise<FileMeta | null> {
    this.ensureSchema();
    const p = path.replace(/^\/+/, "");
    const node = this.resolve(p);
    if (!node) return null;

    const sql = this.ctx.storage.sql;
    const fcs = [...sql.exec<{
      size: number; content_type: string; updated_tx: number; updated_at: number;
    }>("SELECT size, content_type, updated_tx, updated_at FROM file_current_state WHERE node_id = ?", node.node_id)];
    if (!fcs.length) return null;

    const f = fcs[0];
    return { path: p, name: node.name, size: f.size, type: f.content_type, tx: f.updated_tx, tx_time: f.updated_at };
  }

  async listDir(opts?: ListOptions): Promise<{ entries: FileEntry[]; truncated: boolean }> {
    this.ensureSchema();
    const prefix = (opts?.prefix || "").replace(/^\/+/, "");
    const limit = Math.min(opts?.limit || 200, 1000);
    const offset = opts?.offset || 0;

    const sql = this.ctx.storage.sql;

    // Resolve prefix to a directory node
    let parentId = "root";
    if (prefix) {
      const dirPath = prefix.replace(/\/$/, "");
      const node = this.resolve(dirPath);
      if (!node) return { entries: [], truncated: false };
      parentId = node.node_id;
    }

    // List children
    const children = [...sql.exec<{ child_id: string; name: string }>(
      "SELECT child_id, name FROM directory_entries WHERE parent_id = ? AND deleted_tx IS NULL ORDER BY name LIMIT ? OFFSET ?",
      parentId, limit + 1, offset,
    )];

    const truncated = children.length > limit;
    if (truncated) children.pop();

    const entries: FileEntry[] = [];
    for (const child of children) {
      const node = [...sql.exec<{ kind: string }>(
        "SELECT kind FROM nodes WHERE node_id = ? AND deleted_at IS NULL", child.child_id,
      )];
      if (!node.length) continue;

      if (node[0].kind === "dir") {
        entries.push({ name: child.name + "/", type: "directory" });
      } else {
        const fcs = [...sql.exec<{
          size: number; content_type: string; updated_at: number; updated_tx: number;
        }>("SELECT size, content_type, updated_at, updated_tx FROM file_current_state WHERE node_id = ?", child.child_id)];
        if (fcs.length) {
          entries.push({
            name: child.name,
            type: fcs[0].content_type,
            size: fcs[0].size,
            updated_at: fcs[0].updated_at,
            tx: fcs[0].updated_tx,
            tx_time: fcs[0].updated_at,
          });
        }
      }
    }

    return { entries, truncated };
  }

  async searchFiles(query: string, opts?: { limit?: number; prefix?: string }): Promise<SearchResult[]> {
    this.ensureSchema();
    const limit = Math.min(opts?.limit || 50, 200);
    const sql = this.ctx.storage.sql;

    const rows = [...sql.exec<{
      path: string; name: string; node_id: string;
    }>(
      "SELECT path, name, node_id FROM path_cache WHERE kind = 'file' AND (name LIKE ? OR path LIKE ?) LIMIT ?",
      `%${query}%`, `%${query}%`, limit,
    )];

    const results: SearchResult[] = [];
    for (const r of rows) {
      if (opts?.prefix && !r.path.startsWith(opts.prefix)) continue;
      const fcs = [...sql.exec<{ size: number; content_type: string; updated_tx: number }>(
        "SELECT size, content_type, updated_tx FROM file_current_state WHERE node_id = ?", r.node_id,
      )];
      if (fcs.length) {
        results.push({ path: r.path, name: r.name, size: fcs[0].size, type: fcs[0].content_type, tx: fcs[0].updated_tx });
      }
    }
    return results;
  }

  async getStats(): Promise<{ files: number; bytes: number }> {
    this.ensureSchema();
    const rows = [...this.ctx.storage.sql.exec<{ files: number; bytes: number }>(
      "SELECT COUNT(*) as files, COALESCE(SUM(size), 0) as bytes FROM file_current_state",
    )];
    return { files: rows[0]?.files || 0, bytes: rows[0]?.bytes || 0 };
  }

  async getAllNames(): Promise<{ path: string; name: string }[]> {
    this.ensureSchema();
    return [...this.ctx.storage.sql.exec<{ path: string; name: string }>(
      "SELECT path, name FROM path_cache WHERE kind = 'file'",
    )];
  }

  async getLog(opts?: LogOptions): Promise<StorageEvent[]> {
    this.ensureSchema();
    const limit = Math.min(opts?.limit || 50, 500);
    const sql = this.ctx.storage.sql;

    let q = "SELECT tx, action, path, size, msg, meta, ts FROM events WHERE 1=1";
    const binds: any[] = [];

    if (opts?.path) { q += " AND path = ?"; binds.push(opts.path); }
    if (opts?.since_tx) { q += " AND tx > ?"; binds.push(opts.since_tx); }
    if (opts?.before_tx) { q += " AND tx < ?"; binds.push(opts.before_tx); }

    q += " ORDER BY tx DESC, id DESC LIMIT ?";
    binds.push(limit);

    const rows = [...sql.exec<{
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

// ── DOV2Engine adapter (implements StorageEngine) ────────────────────

interface DOV2Config {
  ns: DurableObjectNamespace<StorageDOv2>;
  bucket: R2Bucket;
  r2Endpoint?: string;
  r2AccessKeyId?: string;
  r2SecretAccessKey?: string;
  r2BucketName?: string;
}

export class DOV2Engine implements StorageEngine {
  private ns: DurableObjectNamespace<StorageDOv2>;
  private bucket: R2Bucket;
  private r2Endpoint?: string;
  private r2AccessKeyId?: string;
  private r2SecretAccessKey?: string;
  private r2BucketName: string;

  constructor(config: DOV2Config) {
    this.ns = config.ns;
    this.bucket = config.bucket;
    this.r2Endpoint = config.r2Endpoint;
    this.r2AccessKeyId = config.r2AccessKeyId;
    this.r2SecretAccessKey = config.r2SecretAccessKey;
    this.r2BucketName = config.r2BucketName || "storage-files";
  }

  private stub(actor: string): DurableObjectStub<StorageDOv2> {
    return this.ns.get(this.ns.idFromName(actor));
  }

  private get presignConfigured(): boolean {
    return !!(this.r2Endpoint && this.r2AccessKeyId && this.r2SecretAccessKey);
  }

  private async presign(
    method: "GET" | "PUT",
    key: string,
    expiresIn: number,
    opts?: { contentType?: string; queryParams?: Record<string, string> },
  ): Promise<string> {
    if (!this.presignConfigured) throw new Error("Presigned URLs not configured");
    return presignUrl({
      method, key, bucket: this.r2BucketName,
      endpoint: this.r2Endpoint!, accessKeyId: this.r2AccessKeyId!, secretAccessKey: this.r2SecretAccessKey!,
      expiresIn, contentType: opts?.contentType, queryParams: opts?.queryParams,
    });
  }

  async write(actor: string, path: string, body: ArrayBuffer | ReadableStream, contentType: string, msg?: string): Promise<WriteResult> {
    const buf = body instanceof ArrayBuffer ? body : await streamToBuffer(body);
    const addr = await sha256(buf);
    const key = blobKey(actor, addr);
    if (!(await this.bucket.head(key))) {
      await this.bucket.put(key, buf, { httpMetadata: { contentType } });
    }
    return this.stub(actor).recordWrite(path, addr, buf.byteLength, contentType, msg);
  }

  async move(actor: string, from: string, to: string, msg?: string): Promise<MutationResult> {
    const result = await this.stub(actor).recordMove(from, to, msg);
    return { tx: result.tx, time: result.time };
  }

  async delete(actor: string, paths: string[], msg?: string): Promise<DeleteResult> {
    return this.stub(actor).recordDelete(paths, msg);
  }

  async read(actor: string, path: string): Promise<ReadResult | null> {
    const info = await this.stub(actor).getFileAddr(path);
    if (!info) return null;
    if (info.addr) {
      const obj = await this.bucket.get(blobKey(actor, info.addr));
      if (!obj) { const l = await this.bucket.get(`${actor}/${path}`); if (!l) return null; return { body: l.body, meta: info.meta }; }
      return { body: obj.body, meta: info.meta };
    }
    const obj = await this.bucket.get(`${actor}/${path}`);
    if (!obj) return null;
    return { body: obj.body, meta: info.meta };
  }

  async head(actor: string, path: string): Promise<FileMeta | null> {
    return this.stub(actor).getFileMeta(path);
  }

  async list(actor: string, opts?: ListOptions): Promise<{ entries: FileEntry[]; truncated: boolean }> {
    return this.stub(actor).listDir(opts);
  }

  async search(actor: string, query: string, opts?: { limit?: number; prefix?: string }): Promise<SearchResult[]> {
    return this.stub(actor).searchFiles(query, opts);
  }

  async stats(actor: string): Promise<{ files: number; bytes: number }> {
    return this.stub(actor).getStats();
  }

  async allNames(actor: string): Promise<{ path: string; name: string }[]> {
    return this.stub(actor).getAllNames();
  }

  async log(actor: string, opts?: LogOptions): Promise<StorageEvent[]> {
    return this.stub(actor).getLog(opts);
  }

  async presignRead(actor: string, path: string, expiresIn = 3600): Promise<string | null> {
    if (!this.presignConfigured) return null;
    const info = await this.stub(actor).getFileAddr(path);
    if (!info) return null;
    const key = info.addr ? blobKey(actor, info.addr) : `${actor}/${path}`;
    return this.presign("GET", key, expiresIn);
  }

  async presignUpload(actor: string, path: string, contentType: string, expiresIn = 3600, contentHash?: string): Promise<string> {
    if (contentHash) {
      // Presign directly to content-addressed blob location
      return this.presign("PUT", blobKey(actor, contentHash), expiresIn, { contentType });
    }
    return this.presign("PUT", `${actor}/${path}`, expiresIn, { contentType });
  }

  async confirmUpload(actor: string, path: string, msg?: string, contentHash?: string): Promise<WriteResult> {
    if (contentHash) {
      // Client provided hash — verify blob exists via HEAD, no data pull
      const key = blobKey(actor, contentHash);
      const head = await this.bucket.head(key);
      if (!head) throw new Error("Upload not found at content-addressed location");
      const ct = head.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);
      return this.stub(actor).recordWrite(path, contentHash, head.size, ct, msg);
    }
    // Legacy flow: pull data, compute hash, re-store
    const key = `${actor}/${path}`;
    const obj = await this.bucket.get(key);
    if (!obj) throw new Error("Upload not found in storage");
    const buf = await obj.arrayBuffer();
    const ct = obj.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);
    const result = await this.write(actor, path, buf, ct, msg);
    await this.bucket.delete(key);
    return result;
  }

  async blobExists(actor: string, contentHash: string): Promise<number | null> {
    const head = await this.bucket.head(blobKey(actor, contentHash));
    return head ? head.size : null;
  }

  async initiateMultipart(actor: string, path: string, contentType: string, partCount: number, contentHash?: string) {
    if (!this.presignConfigured) throw new Error("Presigned URLs not configured");
    const key = contentHash ? blobKey(actor, contentHash) : `${actor}/${path}`;
    const mpu = await this.bucket.createMultipartUpload(key, { httpMetadata: { contentType } });
    const partUrls: string[] = [];
    for (let i = 1; i <= Math.min(partCount, 10000); i++) {
      partUrls.push(await this.presign("PUT", key, 86400, { queryParams: { partNumber: String(i), uploadId: mpu.uploadId } }));
    }
    return { upload_id: mpu.uploadId, part_urls: partUrls, expires_in: 86400 };
  }

  async completeMultipart(actor: string, path: string, uploadId: string, parts: { part_number: number; etag: string }[], msg?: string, contentHash?: string): Promise<WriteResult> {
    const key = contentHash ? blobKey(actor, contentHash) : `${actor}/${path}`;
    const mpu = this.bucket.resumeMultipartUpload(key, uploadId);
    await mpu.complete(parts.map((p) => ({ partNumber: p.part_number, etag: p.etag })));
    if (contentHash) {
      // Client provided hash — verify assembled blob exists via HEAD, no data pull
      const head = await this.bucket.head(key);
      if (!head) throw new Error("Multipart upload not found after completion");
      const ct = head.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);
      return this.stub(actor).recordWrite(path, contentHash, head.size, ct, msg);
    }
    // Legacy flow: pull assembled data, compute hash, re-store
    const obj = await this.bucket.get(key);
    if (!obj) throw new Error("Multipart upload not found after completion");
    const buf = await obj.arrayBuffer();
    const ct = obj.httpMetadata?.contentType || mimeFromName(path.split("/").pop() || path);
    const result = await this.write(actor, path, buf, ct, msg);
    await this.bucket.delete(key);
    return result;
  }

  async abortMultipart(actor: string, path: string, uploadId: string): Promise<void> {
    this.bucket.resumeMultipartUpload(`${actor}/${path}`, uploadId).abort();
  }
}
