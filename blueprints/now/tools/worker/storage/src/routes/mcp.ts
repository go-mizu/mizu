import type { Context } from "hono";
import type { Env, Variables, ObjectRow } from "../types";
import { objectId, publicLinkId, publicLinkToken } from "../lib/id";
import { mimeFromName } from "../lib/mime";
import { ensureParentFolders } from "./files";
import { requireScope } from "../middleware/authorize";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// ── JSON-RPC 2.0 wire types ──────────────────────────────────────────

interface RPCRequest {
  jsonrpc: string;
  id?: number | string | null;
  method: string;
  params?: any;
}

interface RPCResponse {
  jsonrpc: "2.0";
  id?: number | string | null;
  result?: any;
  error?: { code: number; message: string };
}

interface ToolDef {
  name: string;
  description: string;
  inputSchema: Record<string, any>;
}

// ── Protocol constants ────────────────────────────────────────────────

const PROTOCOL_VERSION = "2025-06-18";
const SERVER_NAME = "storage.now";
const SERVER_VERSION = "1.0.0";

// ── Tool definitions ──────────────────────────────────────────────────

const TOOLS: ToolDef[] = [
  {
    name: "storage_list",
    description: "List folder contents. Returns direct children of a folder with name, path, type, size, and metadata.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string", description: "Folder path to list (default: root). Must end with / for folders." },
      },
    },
  },
  {
    name: "storage_read",
    description: "Read a file's content. Returns text content for text files, base64 for binary. Includes file metadata.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string", description: "File path to read." },
      },
      required: ["path"],
    },
  },
  {
    name: "storage_write",
    description: "Create or overwrite a file. Two modes: (1) pass 'content' with text or base64 data, or (2) pass 'url' and the SERVER will download it for you — you do NOT need to fetch the URL yourself, just pass it as the 'url' parameter.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string", description: "File path to write." },
        content: { type: "string", description: "File content (text or base64). Not needed when using url mode." },
        url: { type: "string", description: "The server will download this URL and save it to path. You do not need to fetch it yourself — just pass the URL string here." },
        encoding: { type: "string", description: "Content encoding: 'utf-8' (default) or 'base64'. Only used with content, ignored with url." },
        content_type: { type: "string", description: "MIME type. Auto-detected from extension or URL response if omitted." },
      },
      required: ["path"],
    },
  },
  {
    name: "storage_delete",
    description: "Delete a file permanently.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string", description: "File path to delete." },
      },
      required: ["path"],
    },
  },
  {
    name: "storage_search",
    description: "Search files by name pattern. Optionally filter by content type or starred status.",
    inputSchema: {
      type: "object",
      properties: {
        query: { type: "string", description: "Search term to match against filenames." },
        type: { type: "string", description: "Content-type prefix filter (e.g. 'text/', 'image/')." },
        starred: { type: "boolean", description: "If true, only return starred items." },
      },
      required: ["query"],
    },
  },
  {
    name: "storage_move",
    description: "Move or rename a file/folder. Provide new_name to rename in place, or destination to move to another folder.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string", description: "Current path of the item." },
        new_name: { type: "string", description: "New name (rename in place). Cannot contain /." },
        destination: { type: "string", description: "Destination folder path (move). Must end with /." },
      },
      required: ["path"],
    },
  },
  {
    name: "storage_share",
    description: "Create a public shareable URL for a file or folder. Returns a link (https://...//p/TOKEN) that anyone can open in a browser to view or download the file. Use this whenever the user asks to share, get a link, or make a file public.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string", description: "File or folder path to share." },
        expires_in: { type: "number", description: "Link expiry in seconds (optional). Omit for no expiry." },
        password: { type: "string", description: "Optional password to protect the link." },
      },
      required: ["path"],
    },
  },
  {
    name: "storage_stats",
    description: "Get storage usage statistics: file count, folder count, total size, trash count, and quota.",
    inputSchema: {
      type: "object",
      properties: {},
    },
  },
];

// ── GET /mcp — transport discovery ────────────────────────────────────

export async function mcpInfo(c: AppContext) {
  return c.json({
    jsonrpc: "2.0",
    transport: "streamable-http-subset",
    server: SERVER_NAME,
  });
}

// ── POST /mcp — JSON-RPC 2.0 handler ─────────────────────────────────

export async function mcpHandler(c: AppContext) {
  const actor = c.get("actor");
  const scopes = c.get("scopes");
  const ip = c.req.header("cf-connecting-ip") || "unknown";

  let req: RPCRequest;
  try {
    req = await c.req.json<RPCRequest>();
  } catch {
    return rpcResponse(c, { jsonrpc: "2.0", error: { code: -32700, message: "Parse error" } });
  }

  console.log(`[mcp] actor=${actor} method=${req.method} ip=${ip} scopes=${scopes} params=${JSON.stringify(req.params)}`);

  if (req.jsonrpc !== "2.0") {
    return rpcResponse(c, { jsonrpc: "2.0", id: req.id, error: { code: -32600, message: "Invalid request: missing jsonrpc=2.0" } });
  }

  switch (req.method) {
    case "initialize":
      return rpcResponse(c, {
        jsonrpc: "2.0",
        id: req.id,
        result: {
          protocolVersion: PROTOCOL_VERSION,
          serverInfo: { name: SERVER_NAME, version: SERVER_VERSION },
          capabilities: { tools: { listChanged: false } },
          instructions: "storage.now file storage. Tools: storage_list (browse), storage_read/storage_write (file I/O — write supports uploading from a URL), storage_search (find files), storage_move (rename/move), storage_delete, storage_share (create a public shareable link), storage_stats. When asked to share a file, use storage_share.",
        },
      });

    case "notifications/initialized":
      return c.body(null, 204);

    case "tools/list":
      return rpcResponse(c, {
        jsonrpc: "2.0",
        id: req.id,
        result: { tools: TOOLS },
      });

    case "tools/call": {
      const params = req.params as { name: string; arguments?: Record<string, any> } | undefined;
      if (!params?.name) {
        return rpcResponse(c, { jsonrpc: "2.0", id: req.id, error: { code: -32602, message: "Invalid params: missing tool name" } });
      }
      const result = await callTool(c, params.name, params.arguments || {});
      return rpcResponse(c, { jsonrpc: "2.0", id: req.id, result });
    }

    default:
      if (req.id == null) return c.body(null, 204); // notification — no response
      return rpcResponse(c, { jsonrpc: "2.0", id: req.id, error: { code: -32601, message: "Method not found: " + req.method } });
  }
}

// ── Tool dispatcher ───────────────────────────────────────────────────

async function callTool(c: AppContext, name: string, args: Record<string, any>): Promise<any> {
  try {
    switch (name) {
      case "storage_list":   return await toolList(c, args);
      case "storage_read":   return await toolRead(c, args);
      case "storage_write":  return await toolWrite(c, args);
      case "storage_delete": return await toolDelete(c, args);
      case "storage_search": return await toolSearch(c, args);
      case "storage_move":   return await toolMove(c, args);
      case "storage_share":  return await toolShare(c, args);
      case "storage_stats":  return await toolStats(c);
      default:
        return toolError("Unknown tool: " + name);
    }
  } catch (err: any) {
    return toolError(err.message || "Internal error");
  }
}

// ── Tool implementations ──────────────────────────────────────────────

async function toolList(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "folders:read");
  if (scopeErr) return toolError("Token lacks required scope: folders:read");

  const actor = c.get("actor");
  let prefix = normPath((args.path as string) || "");
  if (prefix && !prefix.endsWith("/")) prefix += "/";

  const { results } = await c.env.DB.prepare(`
    SELECT id, path, name, is_folder, content_type, size, starred, created_at, updated_at
    FROM objects
    WHERE owner = ? AND path LIKE ? AND path != ? AND trashed_at IS NULL
    ORDER BY is_folder DESC, name ASC
  `)
    .bind(actor, prefix + "%", prefix || "")
    .all<ObjectRow>();

  // Filter to direct children only
  const items = (results || []).filter((obj) => {
    const rest = obj.path.slice(prefix.length);
    if (obj.is_folder) {
      return rest.replace(/\/$/, "").indexOf("/") === -1;
    }
    return rest.indexOf("/") === -1;
  });

  const mapped = items.map((o) => ({
    name: o.name,
    path: o.path,
    is_folder: !!o.is_folder,
    content_type: o.content_type || "",
    size: o.size,
    starred: !!o.starred,
    updated_at: o.updated_at,
  }));

  return toolResult(JSON.stringify(mapped, null, 2));
}

async function toolRead(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "files:read");
  if (scopeErr) return toolError("Token lacks required scope: files:read");

  const actor = c.get("actor");
  const filePath = normPath(args.path as string || "");
  if (!filePath) return toolError("path is required");

  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND is_folder = 0 AND trashed_at IS NULL",
  )
    .bind(actor, filePath)
    .first<ObjectRow>();

  if (!obj) return toolError("File not found: " + filePath);

  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (!r2Obj) return toolError("File data not found in storage");

  // Track access
  c.executionCtx.waitUntil(
    c.env.DB.prepare("UPDATE objects SET accessed_at = ? WHERE id = ?")
      .bind(Date.now(), obj.id).run(),
  );

  const isText = obj.content_type.startsWith("text/") ||
    obj.content_type === "application/json" ||
    obj.content_type === "application/xml" ||
    obj.content_type === "application/javascript";

  const meta = `path: ${obj.path}\nsize: ${obj.size} bytes\ncontent_type: ${obj.content_type}\nupdated: ${new Date(obj.updated_at).toISOString()}`;

  if (isText) {
    const text = await r2Obj.text();
    // Cap at 100KB to avoid overwhelming the LLM context
    const truncated = text.length > 102400;
    const content = truncated ? text.slice(0, 102400) + "\n... (truncated, " + text.length + " bytes total)" : text;
    return toolResult(meta + "\n---\n" + content);
  }

  // Binary file — return metadata only
  return toolResult(meta + "\n---\n[Binary file, " + obj.size + " bytes. Use the storage REST API to download.]");
}

async function toolWrite(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "files:write");
  if (scopeErr) return toolError("Token lacks required scope: files:write");

  const actor = c.get("actor");
  const filePath = normPath(args.path as string || "");
  const content = args.content as string | undefined;
  const url = args.url as string | undefined;
  const encoding = (args.encoding as string) || "utf-8";

  if (!filePath) return toolError("path is required");
  if (content == null && !url) return toolError("Either content or url is required");
  if (filePath.endsWith("/")) return toolError("path must be a file, not a folder");

  let body: ArrayBuffer;
  let detectedContentType: string | undefined;

  if (url) {
    // Fetch from remote URL
    let res: Response;
    try {
      res = await fetch(url, { redirect: "follow" });
    } catch (err: any) {
      return toolError("Failed to fetch URL: " + err.message);
    }
    if (!res.ok) {
      return toolError(`URL returned ${res.status} ${res.statusText}`);
    }
    body = await res.arrayBuffer();
    detectedContentType = res.headers.get("content-type")?.split(";")[0].trim() || undefined;
  } else if (encoding === "base64") {
    const binary = atob(content!);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    body = bytes.buffer as ArrayBuffer;
  } else {
    body = new TextEncoder().encode(content!).buffer as ArrayBuffer;
  }

  // 10MB limit for MCP writes (smaller than REST API's 100MB)
  if (body.byteLength > 10 * 1024 * 1024) {
    return toolError("Content too large for MCP write (10MB max). Use presigned upload for larger files.");
  }

  const name = filePath.split("/").pop() || filePath;
  const contentType = (args.content_type as string) || detectedContentType || mimeFromName(name);
  const r2Key = `${actor}/${filePath}`;

  await c.env.BUCKET.put(r2Key, body, { httpMetadata: { contentType } });
  await ensureParentFolders(c.env.DB, actor, filePath);

  const now = Date.now();
  const existing = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ?",
  )
    .bind(actor, filePath)
    .first<{ id: string }>();

  let id: string;
  if (existing) {
    id = existing.id;
    await c.env.DB.prepare(
      "UPDATE objects SET content_type = ?, size = ?, r2_key = ?, updated_at = ? WHERE id = ?",
    )
      .bind(contentType, body.byteLength, r2Key, now, id)
      .run();
  } else {
    id = objectId();
    await c.env.DB.prepare(
      "INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?, ?)",
    )
      .bind(id, actor, filePath, name, contentType, body.byteLength, r2Key, now, now)
      .run();
  }

  return toolResult(JSON.stringify({
    id, path: filePath, name, content_type: contentType,
    size: body.byteLength, created: !existing,
  }));
}

async function toolDelete(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "files:write");
  if (scopeErr) return toolError("Token lacks required scope: files:write");

  const actor = c.get("actor");
  const filePath = normPath(args.path as string || "");
  if (!filePath) return toolError("path is required");

  const obj = await c.env.DB.prepare(
    "SELECT id, r2_key FROM objects WHERE owner = ? AND path = ? AND is_folder = 0 AND trashed_at IS NULL",
  )
    .bind(actor, filePath)
    .first<{ id: string; r2_key: string }>();

  if (!obj) return toolError("File not found: " + filePath);

  await c.env.BUCKET.delete(obj.r2_key);
  await c.env.DB.prepare("DELETE FROM shares WHERE object_id = ?").bind(obj.id).run();
  await c.env.DB.prepare("DELETE FROM objects WHERE id = ?").bind(obj.id).run();

  return toolResult(JSON.stringify({ deleted: true, path: filePath }));
}

async function toolSearch(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "drive:read");
  if (scopeErr) return toolError("Token lacks required scope: drive:read");

  const actor = c.get("actor");
  const q = (args.query as string) || "";
  const type = (args.type as string) || "";
  const starred = args.starred as boolean;

  if (!q && !type && !starred) return toolError("At least one of query, type, or starred is required");

  let sql = `SELECT id, path, name, is_folder, content_type, size, starred, created_at, updated_at
    FROM objects WHERE owner = ? AND trashed_at IS NULL`;
  const binds: any[] = [actor];

  if (q) {
    sql += " AND name LIKE ?";
    binds.push(`%${q}%`);
  }
  if (type) {
    sql += " AND content_type LIKE ?";
    binds.push(`${type}%`);
  }
  if (starred) {
    sql += " AND starred = 1";
  }

  sql += " ORDER BY updated_at DESC LIMIT 50";

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all<ObjectRow>();

  const items = (results || []).map((o) => ({
    name: o.name,
    path: o.path,
    is_folder: !!o.is_folder,
    content_type: o.content_type,
    size: o.size,
    starred: !!o.starred,
    updated_at: o.updated_at,
  }));

  return toolResult(JSON.stringify({ count: items.length, items }, null, 2));
}

async function toolMove(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return toolError("Token lacks required scope: drive:write");

  const actor = c.get("actor");
  const path = normPath(args.path as string || "");
  const newName = args.new_name as string;
  const destination = args.destination ? normPath(args.destination as string) : undefined;

  if (!path) return toolError("path is required");
  if (!newName && !destination) return toolError("Provide new_name (rename) or destination (move)");

  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND trashed_at IS NULL",
  )
    .bind(actor, path)
    .first<ObjectRow>();
  if (!obj) return toolError("Not found: " + path);

  const now = Date.now();

  if (newName) {
    // Rename
    if (newName.includes("/")) return toolError("new_name cannot contain /");

    const parts = path.replace(/\/$/, "").split("/");
    parts[parts.length - 1] = newName;
    let newPath = parts.join("/");
    if (obj.is_folder) newPath += "/";

    const conflict = await c.env.DB.prepare(
      "SELECT 1 FROM objects WHERE owner = ? AND path = ?",
    ).bind(actor, newPath).first();
    if (conflict) return toolError("An item named '" + newName + "' already exists at that location");

    if (obj.is_folder) {
      const oldPrefix = path.endsWith("/") ? path : path + "/";
      const newPrefix = newPath.endsWith("/") ? newPath : newPath + "/";
      await c.env.DB.prepare(
        "UPDATE objects SET path = ? || substr(path, ?), updated_at = ? WHERE owner = ? AND path LIKE ?",
      ).bind(newPrefix, oldPrefix.length + 1, now, actor, oldPrefix + "%").run();
      await c.env.DB.prepare(
        "UPDATE objects SET path = ?, name = ?, updated_at = ? WHERE id = ?",
      ).bind(newPath, newName, now, obj.id).run();
    } else {
      const newR2Key = `${actor}/${newPath}`;
      const r2Obj = await c.env.BUCKET.get(obj.r2_key);
      if (r2Obj) {
        await c.env.BUCKET.put(newR2Key, r2Obj.body, {
          httpMetadata: { contentType: obj.content_type },
        });
        await c.env.BUCKET.delete(obj.r2_key);
      }
      await c.env.DB.prepare(
        "UPDATE objects SET path = ?, name = ?, r2_key = ?, updated_at = ? WHERE id = ?",
      ).bind(newPath, newName, newR2Key, now, obj.id).run();
    }

    return toolResult(JSON.stringify({ id: obj.id, old_path: path, new_path: newPath }));
  }

  // Move to destination
  const dest = destination!.endsWith("/") ? destination! : destination! + "/";
  const newPath = dest + obj.name + (obj.is_folder ? "/" : "");

  const conflict = await c.env.DB.prepare(
    "SELECT 1 FROM objects WHERE owner = ? AND path = ?",
  ).bind(actor, newPath).first();
  if (conflict) return toolError("An item already exists at " + newPath);

  if (obj.is_folder) {
    const oldPrefix = path.endsWith("/") ? path : path + "/";
    const newPrefix = newPath.endsWith("/") ? newPath : newPath + "/";
    await c.env.DB.prepare(
      "UPDATE objects SET path = ? || substr(path, ?), updated_at = ? WHERE owner = ? AND path LIKE ?",
    ).bind(newPrefix, oldPrefix.length + 1, now, actor, oldPrefix + "%").run();
    await c.env.DB.prepare(
      "UPDATE objects SET path = ?, updated_at = ? WHERE id = ?",
    ).bind(newPath, now, obj.id).run();
  } else {
    const newR2Key = `${actor}/${newPath}`;
    const r2Obj = await c.env.BUCKET.get(obj.r2_key);
    if (r2Obj) {
      await c.env.BUCKET.put(newR2Key, r2Obj.body, {
        httpMetadata: { contentType: obj.content_type },
      });
      await c.env.BUCKET.delete(obj.r2_key);
    }
    await c.env.DB.prepare(
      "UPDATE objects SET path = ?, r2_key = ?, updated_at = ? WHERE id = ?",
    ).bind(newPath, newR2Key, now, obj.id).run();
  }

  return toolResult(JSON.stringify({ id: obj.id, old_path: path, new_path: newPath }));
}

async function toolShare(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "links:manage");
  if (scopeErr) return toolError("Token lacks required scope: links:manage");

  const actor = c.get("actor");
  const filePath = normPath(args.path as string || "");
  if (!filePath) return toolError("path is required");

  const obj = await c.env.DB.prepare(
    "SELECT id, is_folder, name FROM objects WHERE owner = ? AND path = ? AND trashed_at IS NULL",
  ).bind(actor, filePath).first<{ id: string; is_folder: number; name: string }>();

  if (!obj) return toolError("Not found: " + filePath);

  const token = publicLinkToken();
  const id = publicLinkId();
  const now = Date.now();
  const expiresIn = args.expires_in as number | undefined;
  const expiresAt = expiresIn ? now + expiresIn * 1000 : null;

  let passwordHash: string | null = null;
  if (args.password) {
    const encoded = new TextEncoder().encode(args.password as string);
    const hash = await crypto.subtle.digest("SHA-256", encoded);
    passwordHash = Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
  }

  await c.env.DB.prepare(
    `INSERT INTO public_links (id, object_id, owner, token, permission, password_hash, expires_at, max_downloads, download_count, created_at)
     VALUES (?, ?, ?, ?, 'viewer', ?, ?, NULL, 0, ?)`,
  ).bind(id, obj.id, actor, token, passwordHash, expiresAt, now).run();

  const origin = new URL(c.req.url).origin;
  const url = `${origin}/p/${token}`;

  return toolResult(JSON.stringify({
    url,
    token,
    path: filePath,
    name: obj.name,
    password_protected: !!passwordHash,
    expires_at: expiresAt ? new Date(expiresAt).toISOString() : null,
  }, null, 2));
}

async function toolStats(c: AppContext) {
  const scopeErr = requireScope(c, "drive:read");
  if (scopeErr) return toolError("Token lacks required scope: drive:read");

  const actor = c.get("actor");

  const stats = await c.env.DB.prepare(
    `SELECT COUNT(*) as file_count, COALESCE(SUM(size),0) as total_size
     FROM objects WHERE owner = ? AND is_folder = 0 AND trashed_at IS NULL`,
  ).bind(actor).first<{ file_count: number; total_size: number }>();

  const folderCount = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM objects WHERE owner = ? AND is_folder = 1 AND trashed_at IS NULL",
  ).bind(actor).first<{ count: number }>();

  const trashCount = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM objects WHERE owner = ? AND trashed_at IS NOT NULL",
  ).bind(actor).first<{ count: number }>();

  return toolResult(JSON.stringify({
    actor,
    file_count: stats?.file_count || 0,
    folder_count: folderCount?.count || 0,
    total_size: stats?.total_size || 0,
    total_size_human: humanSize(stats?.total_size || 0),
    trash_count: trashCount?.count || 0,
    quota: 5 * 1024 * 1024 * 1024,
    quota_human: "5 GB",
  }, null, 2));
}

// ── Helpers ───────────────────────────────────────────────────────────

/** Strip leading / — storage paths are relative (e.g. "docs/readme.md" not "/docs/readme.md") */
function normPath(p: string): string {
  while (p.startsWith("/")) p = p.slice(1);
  return p;
}

function toolResult(text: string): any {
  return { content: [{ type: "text", text }], isError: false };
}

function toolError(message: string): any {
  return { content: [{ type: "text", text: message }], isError: true };
}

function rpcResponse(c: AppContext, resp: RPCResponse) {
  return c.json(resp);
}

function humanSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const val = bytes / Math.pow(1024, i);
  return val.toFixed(i === 0 ? 0 : 1) + " " + units[i];
}
