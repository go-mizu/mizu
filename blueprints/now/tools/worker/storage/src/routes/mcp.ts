import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { mimeFromName, isInlineType } from "../lib/mime";
import { shareToken } from "../lib/id";
import { invalidateCache } from "./find";
import { validatePath } from "../lib/path";

type C = Context<{ Bindings: Env; Variables: Variables }>;

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
const SERVER_VERSION = "3.0.0";

// Simple in-memory session tracking (maps session ID → actor)
const mcpSessions = new Map<string, { actor: string; created: number }>();
const SESSION_TTL_MS = 3_600_000; // 1 hour

// ── Tool definitions ──────────────────────────────────────────────────

const TOOLS: ToolDef[] = [
  {
    name: "storage_list",
    description:
      "List the user's files and folders in their cloud storage (storage.now). " +
      "Returns immediate children at the given path — each entry has a name, MIME type (or 'directory'), size in bytes, and last-modified timestamp. " +
      "Use this first to see what files the user has. Call with no prefix to list the root, or pass a folder path (e.g. 'photos/') to list that folder.",
    inputSchema: {
      type: "object",
      properties: {
        prefix: {
          type: "string",
          description:
            "Folder path to list. No leading slash. Use trailing slash for folders (e.g. 'documents/', 'projects/web/'). " +
            "Omit or leave empty to list the root directory.",
        },
        limit: { type: "number", description: "Maximum entries to return. Default 200, max 1000." },
      },
    },
  },
  {
    name: "storage_read",
    description:
      "Read a file from the user's cloud storage (storage.now) and return its contents. " +
      "For text files (plain text, markdown, JSON, code, etc.) the full content is returned inline so you can read and discuss it. " +
      "For binary files (images, PDFs, zips, etc.) only metadata is returned — suggest using storage_share to generate a download link instead. " +
      "Always call storage_list first if you are not sure of the exact file path.",
    inputSchema: {
      type: "object",
      properties: {
        path: {
          type: "string",
          description: "Full file path, no leading slash. Example: 'notes/todo.md', 'data/report.csv'.",
        },
      },
      required: ["path"],
    },
  },
  {
    name: "storage_write",
    description:
      "Save a file to the user's cloud storage (storage.now). Creates or overwrites. Two modes:\n" +
      "• Text mode — pass 'content' with the file body. Good for saving text, code, markdown, JSON, CSV, etc.\n" +
      "• URL mode — pass 'url' and the server downloads it for you. Good for saving images, PDFs, or any file from the web. " +
      "You do NOT need to fetch the URL yourself; just provide it.\n" +
      "The file path determines the folder structure automatically (e.g. 'reports/q1.pdf' puts the file in the 'reports' folder).",
    inputSchema: {
      type: "object",
      properties: {
        path: {
          type: "string",
          description:
            "Destination file path, no leading slash. The path doubles as the folder structure. " +
            "Examples: 'notes/meeting.md', 'images/photo.png', 'data/export.csv'.",
        },
        content: {
          type: "string",
          description:
            "The file content as a string. For text files, pass the text directly. " +
            "For binary data, base64-encode it and set encoding to 'base64'. " +
            "Not needed when using url mode.",
        },
        url: {
          type: "string",
          description:
            "A public URL to download. The server fetches it and saves the result. " +
            "Use this for images, PDFs, or any web resource — you do NOT need to download it yourself.",
        },
        encoding: {
          type: "string",
          enum: ["utf-8", "base64"],
          description: "How 'content' is encoded. Default 'utf-8'. Set to 'base64' for binary data passed via content.",
        },
        content_type: {
          type: "string",
          description: "MIME type (e.g. 'text/markdown', 'image/png'). Auto-detected from file extension if omitted.",
        },
      },
      required: ["path"],
    },
  },
  {
    name: "storage_delete",
    description:
      "Delete files or folders from the user's cloud storage (storage.now). " +
      "Pass exact file paths to delete individual files. " +
      "To delete an entire folder and everything inside it, add a trailing slash (e.g. 'old-project/'). " +
      "This action is permanent and cannot be undone — confirm with the user before deleting.",
    inputSchema: {
      type: "object",
      properties: {
        paths: {
          type: "array",
          items: { type: "string" },
          description:
            "List of paths to delete. " +
            "Examples: ['notes/draft.md'] deletes one file; ['temp/'] deletes the entire temp folder recursively.",
        },
      },
      required: ["paths"],
    },
  },
  {
    name: "storage_search",
    description:
      "Search the user's cloud storage (storage.now) by filename. " +
      "Performs a partial, case-insensitive match against file names — useful when you don't know the exact path. " +
      "For example, searching 'report' finds 'reports/q1-report.pdf', 'report.md', etc. " +
      "Returns matching file paths, names, sizes, and types, sorted by most recently modified.",
    inputSchema: {
      type: "object",
      properties: {
        query: {
          type: "string",
          description: "Search term to match against file names. Partial matches work: 'todo' matches 'my-todo-list.md'.",
        },
        limit: { type: "number", description: "Maximum results to return. Default 50, max 200." },
      },
      required: ["query"],
    },
  },
  {
    name: "storage_move",
    description:
      "Move or rename a file in the user's cloud storage (storage.now). " +
      "Use this to rename a file (same folder, different name), move it to a different folder, or both. " +
      "The source file must exist. If the destination already exists, it will be overwritten.",
    inputSchema: {
      type: "object",
      properties: {
        from: { type: "string", description: "Current file path. Example: 'drafts/post.md'." },
        to: { type: "string", description: "New file path. Example: 'published/post.md' or 'drafts/post-v2.md'." },
      },
      required: ["from", "to"],
    },
  },
  {
    name: "storage_share",
    description:
      "Generate a public share link for a file in the user's cloud storage (storage.now). " +
      "The returned URL can be opened by anyone — no login required. " +
      "IMPORTANT: Always show the full URL to the user so they can copy or share it. " +
      "Links expire after the specified duration (default 1 hour, max 7 days). " +
      "Use this when the user wants to share, download, or send a file to someone.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string", description: "File path to share. Example: 'docs/report.pdf'." },
        expires_in: {
          type: "number",
          description:
            "How long the link stays valid, in seconds. Default: 3600 (1 hour). Max: 604800 (7 days). " +
            "Common values: 3600 = 1 hour, 86400 = 1 day, 604800 = 1 week.",
        },
      },
      required: ["path"],
    },
  },
  {
    name: "storage_stats",
    description:
      "Show the user's storage usage on storage.now — total number of files and total size. " +
      "Use this when the user asks about their storage, how much space they are using, or how many files they have.",
    inputSchema: {
      type: "object",
      properties: {},
    },
  },
];

// ── GET /mcp — SSE endpoint for server-initiated notifications ───────

async function mcpGet(c: C) {
  // Per MCP Streamable HTTP spec: GET opens an SSE stream.
  // Minimal implementation — just keeps connection alive for clients that need it.
  const accept = c.req.header("Accept") || "";
  if (accept.includes("text/event-stream")) {
    const { readable, writable } = new TransformStream();
    const writer = writable.getWriter();
    const enc = new TextEncoder();
    // Send a comment to keep connection alive then close
    writer.write(enc.encode(": ok\n\n"));
    writer.close();
    return new Response(readable, {
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
      },
    });
  }
  // Fallback: return JSON info
  return c.json({
    jsonrpc: "2.0",
    transport: "streamable-http",
    server: SERVER_NAME,
    version: SERVER_VERSION,
  });
}

// ── POST /mcp — JSON-RPC 2.0 handler ─────────────────────────────────

async function mcpHandler(c: C) {
  const actor = c.get("actor");
  const ip = c.req.header("cf-connecting-ip") || "unknown";

  let req: RPCRequest;
  try {
    req = await c.req.json<RPCRequest>();
  } catch {
    return rpcResponse(c, { jsonrpc: "2.0", error: { code: -32700, message: "Parse error" } });
  }

  console.log(JSON.stringify({ level: "info", component: "mcp", actor, method: req.method, ip, ts: Date.now() }));

  if (req.jsonrpc !== "2.0") {
    return rpcResponse(c, { jsonrpc: "2.0", id: req.id, error: { code: -32600, message: "Invalid request: missing jsonrpc=2.0" } });
  }

  switch (req.method) {
    case "initialize": {
      // Create MCP session and return ID in header per Streamable HTTP spec
      const sessionId = crypto.randomUUID();
      mcpSessions.set(sessionId, { actor, created: Date.now() });
      // Cleanup old sessions
      const cutoff = Date.now() - SESSION_TTL_MS;
      for (const [k, v] of mcpSessions) { if (v.created < cutoff) mcpSessions.delete(k); }

      const resp = rpcResponse(c, {
        jsonrpc: "2.0",
        id: req.id,
        result: {
          protocolVersion: PROTOCOL_VERSION,
          serverInfo: { name: SERVER_NAME, version: SERVER_VERSION },
          capabilities: { tools: { listChanged: false } },
          instructions: [
            "You have access to the user's cloud file storage on storage.now (not their local device).",
            "These tools let you browse, read, write, organize, search, and share the user's cloud files.",
            "",
            "Common workflows:",
            "• To see what files exist: call storage_list (empty prefix = root).",
            "• To read a file: call storage_read with the path. Text files return content inline.",
            "• To save content (text, code, notes): call storage_write with path + content.",
            "• To save a file from a URL (image, PDF, webpage): call storage_write with path + url. The server downloads it — you do NOT need to fetch it yourself.",
            "• To find a file when you don't know the exact path: call storage_search.",
            "• To share a file: call storage_share, then ALWAYS show the returned URL to the user.",
            "• To check usage: call storage_stats.",
            "",
            "Path format: no leading slash, use forward slashes for folders. Example: 'docs/notes.md', 'images/logo.png'.",
            "Deleting is permanent — always confirm with the user before calling storage_delete.",
          ].join("\n"),
        },
      });
      resp.headers.set("Mcp-Session-Id", sessionId);
      return resp;
    }

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
      if (req.id == null) return c.body(null, 204);
      return rpcResponse(c, { jsonrpc: "2.0", id: req.id, error: { code: -32601, message: "Method not found: " + req.method } });
  }
}

// ── DELETE /mcp — terminate session ───────────────────────────────────

async function mcpDelete(c: C) {
  const sessionId = c.req.header("Mcp-Session-Id");
  if (sessionId) mcpSessions.delete(sessionId);
  return c.body(null, 204);
}

// ── Tool dispatcher ───────────────────────────────────────────────────

async function callTool(c: C, name: string, args: Record<string, any>): Promise<any> {
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

// ── Helpers ───────────────────────────────────────────────────────────

function toolResult(text: string): any {
  return { content: [{ type: "text", text }], isError: false };
}

function toolError(message: string): any {
  return { content: [{ type: "text", text: message }], isError: true };
}

function rpcResponse(c: C, resp: RPCResponse) {
  return c.json(resp);
}

function humanSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const val = bytes / Math.pow(1024, i);
  return val.toFixed(i === 0 ? 0 : 1) + " " + units[i];
}

function checkPrefix(c: C, path: string): string | null {
  const pfx = c.get("prefix");
  if (pfx && !path.startsWith(pfx)) return "Path not allowed for this token";
  return null;
}

// ── Tool implementations ──────────────────────────────────────────────

async function toolList(c: C, args: Record<string, any>) {
  const actor = c.get("actor");
  const prefix = (args.prefix as string) || "";
  const pfxErr = checkPrefix(c, prefix);
  if (pfxErr) return toolError(pfxErr);

  const limit = Math.min((args.limit as number) || 200, 1000);

  const { results } = await c.env.DB.prepare(
    "SELECT path, name, size, type, updated_at FROM files WHERE owner = ? AND path LIKE ? ORDER BY path LIMIT ?",
  ).bind(actor, `${prefix}%`, limit).all();

  const rows = results || [];
  const entries: { name: string; type: string; size?: number; updated_at?: number }[] = [];
  const dirs = new Set<string>();

  for (const row of rows) {
    const relative = (row.path as string).slice(prefix.length);
    const slash = relative.indexOf("/");
    if (slash === -1) {
      entries.push({ name: relative, type: row.type as string, size: row.size as number, updated_at: row.updated_at as number });
    } else {
      const dir = relative.slice(0, slash + 1);
      if (!dirs.has(dir)) {
        dirs.add(dir);
        entries.push({ name: dir, type: "directory" });
      }
    }
  }

  return toolResult(JSON.stringify({ prefix: prefix || "/", entries }, null, 2));
}

async function toolRead(c: C, args: Record<string, any>) {
  const filePath = args.path as string;
  if (!filePath) return toolError("path is required");
  const pfxErr = checkPrefix(c, filePath);
  if (pfxErr) return toolError(pfxErr);

  const actor = c.get("actor");
  const r2Obj = await c.env.BUCKET.get(`${actor}/${filePath}`);
  if (!r2Obj) return toolError("File not found: " + filePath);

  const ct = r2Obj.httpMetadata?.contentType || "application/octet-stream";
  const isText = ct.startsWith("text/") ||
    ct === "application/json" ||
    ct === "application/xml" ||
    ct === "application/javascript";

  const meta = `path: ${filePath}\nsize: ${r2Obj.size} bytes\ncontent_type: ${ct}\netag: ${r2Obj.etag}`;

  if (isText) {
    const text = await r2Obj.text();
    const truncated = text.length > 102400;
    const content = truncated ? text.slice(0, 102400) + "\n... (truncated, " + text.length + " bytes total)" : text;
    return toolResult(meta + "\n---\n" + content);
  }

  return toolResult(meta + "\n---\n[Binary file, " + r2Obj.size + " bytes. Use the storage REST API to download.]");
}

async function toolWrite(c: C, args: Record<string, any>) {
  const actor = c.get("actor");
  const filePath = args.path as string;
  const content = args.content as string | undefined;
  const url = args.url as string | undefined;
  const encoding = (args.encoding as string) || "utf-8";

  if (!filePath) return toolError("path is required");
  const pathErr = validatePath(filePath);
  if (pathErr) return toolError(pathErr);
  const pfxErr = checkPrefix(c, filePath);
  if (pfxErr) return toolError(pfxErr);
  if (content == null && !url) return toolError("Either content or url is required");

  let body: ArrayBuffer;
  let detectedContentType: string | undefined;

  if (url) {
    let res: Response;
    try {
      res = await fetch(url, { redirect: "follow" });
    } catch (err: any) {
      return toolError("Failed to fetch URL: " + err.message);
    }
    if (!res.ok) return toolError(`URL returned ${res.status} ${res.statusText}`);
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

  if (body.byteLength > 10 * 1024 * 1024) {
    return toolError("Content too large for MCP write (10 MB max). Use the REST API for larger files.");
  }

  const name = filePath.split("/").pop() || filePath;
  const contentType = (args.content_type as string) || detectedContentType || mimeFromName(name);
  const now = Date.now();

  await c.env.BUCKET.put(`${actor}/${filePath}`, body, { httpMetadata: { contentType } });
  await c.env.DB.prepare(
    "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT (owner, path) DO UPDATE SET size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
  ).bind(actor, filePath, name, body.byteLength, contentType, now).run();

  invalidateCache(actor);

  return toolResult(JSON.stringify({
    path: filePath, name, content_type: contentType, size: body.byteLength, updated_at: now,
  }));
}

async function toolDelete(c: C, args: Record<string, any>) {
  const paths = args.paths as string[];
  if (!paths || !paths.length) return toolError("paths is required");

  const actor = c.get("actor");
  const deleted: string[] = [];

  for (const path of paths) {
    const pfxErr = checkPrefix(c, path);
    if (pfxErr) continue;

    if (path.endsWith("/")) {
      const prefix = `${actor}/${path}`;
      let cursor: string | undefined;
      do {
        const list = await c.env.BUCKET.list({ prefix, cursor, limit: 1000 });
        if (list.objects.length) {
          await c.env.BUCKET.delete(list.objects.map((o) => o.key));
        }
        cursor = list.truncated ? list.cursor : undefined;
      } while (cursor);
      await c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path LIKE ?")
        .bind(actor, `${path}%`).run();
      deleted.push(path);
    } else {
      await c.env.BUCKET.delete(`${actor}/${path}`);
      await c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path = ?")
        .bind(actor, path).run();
      deleted.push(path);
    }
  }

  invalidateCache(actor);
  return toolResult(JSON.stringify({ deleted }));
}

async function toolSearch(c: C, args: Record<string, any>) {
  const q = (args.query as string) || "";
  if (!q) return toolError("query is required");

  const actor = c.get("actor");
  const limit = Math.min((args.limit as number) || 50, 200);
  const pfx = c.get("prefix") || "";

  let sql = "SELECT path, name, size, type, updated_at FROM files WHERE owner = ? AND name LIKE ?";
  const binds: any[] = [actor, `%${q}%`];

  if (pfx) {
    sql += " AND path LIKE ?";
    binds.push(`${pfx}%`);
  }

  sql += " ORDER BY updated_at DESC LIMIT ?";
  binds.push(limit);

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all();

  const items = (results || []).map((o: any) => ({
    name: o.name, path: o.path, type: o.type, size: o.size, updated_at: o.updated_at,
  }));

  return toolResult(JSON.stringify({ count: items.length, items }, null, 2));
}

async function toolMove(c: C, args: Record<string, any>) {
  const actor = c.get("actor");
  const from = args.from as string;
  const to = args.to as string;

  if (!from || !to) return toolError("from and to are required");
  const fromErr = checkPrefix(c, from);
  if (fromErr) return toolError(fromErr);
  const toErr = checkPrefix(c, to);
  if (toErr) return toolError(toErr);
  const pathErr = validatePath(to);
  if (pathErr) return toolError(pathErr);

  const r2Obj = await c.env.BUCKET.get(`${actor}/${from}`);
  if (!r2Obj) return toolError("File not found: " + from);

  await c.env.BUCKET.put(`${actor}/${to}`, r2Obj.body, {
    httpMetadata: r2Obj.httpMetadata,
  });
  await c.env.BUCKET.delete(`${actor}/${from}`);

  const newName = to.split("/").pop() || to;
  const now = Date.now();
  await c.env.DB.batch([
    c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path = ?").bind(actor, from),
    c.env.DB.prepare(
      "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
        "ON CONFLICT (owner, path) DO UPDATE SET name = excluded.name, size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
    ).bind(actor, to, newName, r2Obj.size, r2Obj.httpMetadata?.contentType || "application/octet-stream", now),
  ]);

  invalidateCache(actor);
  return toolResult(JSON.stringify({ old_path: from, new_path: to }));
}

async function toolShare(c: C, args: Record<string, any>) {
  const actor = c.get("actor");
  const filePath = args.path as string;
  if (!filePath) return toolError("path is required");
  const pfxErr = checkPrefix(c, filePath);
  if (pfxErr) return toolError(pfxErr);

  // Verify file exists
  const obj = await c.env.BUCKET.head(`${actor}/${filePath}`);
  if (!obj) return toolError("File not found: " + filePath);

  const expiresIn = Math.min(Math.max((args.expires_in as number) || 3600, 60), 7 * 24 * 3600);
  const now = Date.now();
  const expiresAt = now + expiresIn * 1000;

  const token = shareToken();
  await c.env.DB.prepare(
    "INSERT INTO share_links (token, actor, path, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
  ).bind(token, actor, filePath, expiresAt, now).run();

  const origin = new URL(c.req.url).origin;
  const url = `${origin}/s/${token}`;

  return toolResult(JSON.stringify({
    url,
    token,
    path: filePath,
    expires_at: new Date(expiresAt).toISOString(),
  }, null, 2));
}

async function toolStats(c: C) {
  const actor = c.get("actor");

  const stats = await c.env.DB
    .prepare("SELECT COUNT(*) as file_count, COALESCE(SUM(size),0) as total_size FROM files WHERE owner = ?")
    .bind(actor)
    .first<{ file_count: number; total_size: number }>();

  return toolResult(JSON.stringify({
    actor,
    file_count: stats?.file_count || 0,
    total_size: stats?.total_size || 0,
    total_size_human: humanSize(stats?.total_size || 0),
  }, null, 2));
}

// ── Registration ──────────────────────────────────────────────────────

export function register(app: App) {
  app.get("/mcp", auth, mcpGet);
  app.post("/mcp", auth, mcpHandler);
  app.delete("/mcp", auth, mcpDelete);
}
