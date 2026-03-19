import type { Context } from "hono";
import type { Env, Variables, ObjectRow, BucketRow } from "../types";
import { objectId, signedUrlId, signedUrlToken } from "../lib/id";
import { mimeFromName } from "../lib/mime";
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
const SERVER_VERSION = "2.0.0";

// ── Tool definitions ──────────────────────────────────────────────────

const TOOLS: ToolDef[] = [
  {
    name: "storage_list",
    description: "List objects in a bucket. Returns direct children with name, path, size, and metadata. If no bucket specified, lists all buckets.",
    inputSchema: {
      type: "object",
      properties: {
        bucket: { type: "string", description: "Bucket name to list objects in. Omit to list all buckets." },
        prefix: { type: "string", description: "Path prefix to filter objects (e.g. 'reports/'). Only used with bucket." },
      },
    },
  },
  {
    name: "storage_read",
    description: "Read a file's content. Returns text content for text files, metadata for binary. Includes file metadata.",
    inputSchema: {
      type: "object",
      properties: {
        bucket: { type: "string", description: "Bucket name." },
        path: { type: "string", description: "Object path to read." },
      },
      required: ["bucket", "path"],
    },
  },
  {
    name: "storage_write",
    description: "Create or overwrite a file in a bucket. Two modes: (1) pass 'content' with text or base64 data, or (2) pass 'url' and the SERVER will download it for you.",
    inputSchema: {
      type: "object",
      properties: {
        bucket: { type: "string", description: "Bucket name." },
        path: { type: "string", description: "Object path to write." },
        content: { type: "string", description: "File content (text or base64). Not needed when using url mode." },
        url: { type: "string", description: "The server will download this URL and save it. You do not need to fetch it yourself." },
        encoding: { type: "string", description: "Content encoding: 'utf-8' (default) or 'base64'. Only used with content." },
        content_type: { type: "string", description: "MIME type. Auto-detected from extension if omitted." },
      },
      required: ["bucket", "path"],
    },
  },
  {
    name: "storage_delete",
    description: "Delete one or more objects from a bucket.",
    inputSchema: {
      type: "object",
      properties: {
        bucket: { type: "string", description: "Bucket name." },
        paths: { type: "array", items: { type: "string" }, description: "Object paths to delete." },
      },
      required: ["bucket", "paths"],
    },
  },
  {
    name: "storage_search",
    description: "Search objects by name across a bucket.",
    inputSchema: {
      type: "object",
      properties: {
        bucket: { type: "string", description: "Bucket name." },
        query: { type: "string", description: "Search term to match against filenames." },
        type: { type: "string", description: "Content-type prefix filter (e.g. 'text/', 'image/')." },
      },
      required: ["bucket", "query"],
    },
  },
  {
    name: "storage_move",
    description: "Move or rename an object within a bucket.",
    inputSchema: {
      type: "object",
      properties: {
        bucket: { type: "string", description: "Bucket name." },
        from: { type: "string", description: "Current object path." },
        to: { type: "string", description: "New object path." },
      },
      required: ["bucket", "from", "to"],
    },
  },
  {
    name: "storage_share",
    description: "Create a signed URL for sharing a file. Returns a link that anyone can use to download the file.",
    inputSchema: {
      type: "object",
      properties: {
        bucket: { type: "string", description: "Bucket name." },
        path: { type: "string", description: "Object path to share." },
        expires_in: { type: "number", description: "Link expiry in seconds (default: 3600, max: 604800)." },
      },
      required: ["bucket", "path"],
    },
  },
  {
    name: "storage_stats",
    description: "Get storage usage statistics: bucket count, object count, total size.",
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
          instructions: "storage.now file storage (v2 — bucket model). Tools: storage_list (browse buckets/objects), storage_read/storage_write (file I/O — write supports uploading from a URL), storage_search (find objects), storage_move (rename/move), storage_delete, storage_share (create a signed URL for sharing), storage_stats. When asked to share a file, use storage_share.",
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
      if (req.id == null) return c.body(null, 204);
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

// ── Helpers ───────────────────────────────────────────────────────────

async function resolveBucket(c: AppContext, name: string): Promise<BucketRow | null> {
  const actor = c.get("actor");
  return c.env.DB
    .prepare("SELECT * FROM buckets WHERE owner = ? AND name = ?")
    .bind(actor, name)
    .first<BucketRow>();
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

// ── Tool implementations ──────────────────────────────────────────────

async function toolList(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return toolError("Token lacks required scope: object:read");

  const actor = c.get("actor");

  // No bucket specified → list all buckets
  if (!args.bucket) {
    const { results } = await c.env.DB
      .prepare("SELECT * FROM buckets WHERE owner = ? ORDER BY name")
      .bind(actor)
      .all<BucketRow>();

    const buckets = (results || []).map((b) => ({
      name: b.name,
      public: !!b.public,
      created_at: b.created_at,
    }));
    return toolResult(JSON.stringify(buckets, null, 2));
  }

  const bucket = await resolveBucket(c, args.bucket as string);
  if (!bucket) return toolError("Bucket not found: " + args.bucket);

  const prefix = (args.prefix as string) || "";

  let sql = "SELECT path, name, content_type, size, created_at, updated_at FROM objects WHERE bucket_id = ?";
  const binds: any[] = [bucket.id];

  if (prefix) {
    sql += " AND path LIKE ?";
    binds.push(prefix.replace(/%/g, "\\%") + "%");
  }

  sql += " ORDER BY name ASC LIMIT 100";

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all();

  // Filter to direct children only
  const items = (results || []).filter((obj: any) => {
    const rest = obj.path.slice(prefix.length);
    return rest.indexOf("/") === -1 || (rest.endsWith("/") && rest.indexOf("/") === rest.length - 1);
  }).map((o: any) => ({
    name: o.name,
    path: o.path,
    content_type: o.content_type || "",
    size: o.size,
    updated_at: o.updated_at,
  }));

  return toolResult(JSON.stringify(items, null, 2));
}

async function toolRead(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return toolError("Token lacks required scope: object:read");

  const bucketName = args.bucket as string;
  const filePath = args.path as string;
  if (!bucketName || !filePath) return toolError("bucket and path are required");

  const bucket = await resolveBucket(c, bucketName);
  if (!bucket) return toolError("Bucket not found: " + bucketName);

  const obj = await c.env.DB
    .prepare("SELECT * FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first<ObjectRow>();

  if (!obj) return toolError("Object not found: " + filePath);

  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (!r2Obj) return toolError("Object data not found in storage");

  c.executionCtx.waitUntil(
    c.env.DB.prepare("UPDATE objects SET accessed_at = ? WHERE id = ?")
      .bind(Date.now(), obj.id).run(),
  );

  const isText = obj.content_type.startsWith("text/") ||
    obj.content_type === "application/json" ||
    obj.content_type === "application/xml" ||
    obj.content_type === "application/javascript";

  const meta = `bucket: ${bucketName}\npath: ${obj.path}\nsize: ${obj.size} bytes\ncontent_type: ${obj.content_type}\nupdated: ${new Date(obj.updated_at).toISOString()}`;

  if (isText) {
    const text = await r2Obj.text();
    const truncated = text.length > 102400;
    const content = truncated ? text.slice(0, 102400) + "\n... (truncated, " + text.length + " bytes total)" : text;
    return toolResult(meta + "\n---\n" + content);
  }

  return toolResult(meta + "\n---\n[Binary file, " + obj.size + " bytes. Use the storage REST API to download.]");
}

async function toolWrite(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return toolError("Token lacks required scope: object:write");

  const actor = c.get("actor");
  const bucketName = args.bucket as string;
  const filePath = args.path as string;
  const content = args.content as string | undefined;
  const url = args.url as string | undefined;
  const encoding = (args.encoding as string) || "utf-8";

  if (!bucketName || !filePath) return toolError("bucket and path are required");
  if (content == null && !url) return toolError("Either content or url is required");

  const bucket = await resolveBucket(c, bucketName);
  if (!bucket) return toolError("Bucket not found: " + bucketName);

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
    return toolError("Content too large for MCP write (10MB max). Use signed upload URL for larger files.");
  }

  const name = filePath.split("/").pop() || filePath;
  const contentType = (args.content_type as string) || detectedContentType || mimeFromName(name);
  const r2Key = `${actor}/${bucket.name}/${filePath}`;

  await c.env.BUCKET.put(r2Key, body, { httpMetadata: { contentType } });

  const now = Date.now();
  const existing = await c.env.DB
    .prepare("SELECT id FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first<{ id: string }>();

  let id: string;
  if (existing) {
    id = existing.id;
    await c.env.DB.prepare(
      "UPDATE objects SET content_type = ?, size = ?, r2_key = ?, updated_at = ? WHERE id = ?",
    ).bind(contentType, body.byteLength, r2Key, now, id).run();
  } else {
    id = objectId();
    await c.env.DB.prepare(
      "INSERT INTO objects (id, owner, bucket_id, path, name, content_type, size, r2_key, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '{}', ?, ?)",
    ).bind(id, actor, bucket.id, filePath, name, contentType, body.byteLength, r2Key, now, now).run();
  }

  return toolResult(JSON.stringify({
    id, bucket: bucketName, path: filePath, name, content_type: contentType,
    size: body.byteLength, created: !existing,
  }));
}

async function toolDelete(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return toolError("Token lacks required scope: object:write");

  const bucketName = args.bucket as string;
  const paths = args.paths as string[];
  if (!bucketName || !paths || !paths.length) return toolError("bucket and paths are required");

  const bucket = await resolveBucket(c, bucketName);
  if (!bucket) return toolError("Bucket not found: " + bucketName);

  const deleted: string[] = [];
  for (const path of paths) {
    const obj = await c.env.DB
      .prepare("SELECT id, r2_key FROM objects WHERE bucket_id = ? AND path = ?")
      .bind(bucket.id, path)
      .first<{ id: string; r2_key: string }>();

    if (obj) {
      if (obj.r2_key) await c.env.BUCKET.delete(obj.r2_key);
      await c.env.DB.prepare("DELETE FROM objects WHERE id = ?").bind(obj.id).run();
      deleted.push(path);
    }
  }

  return toolResult(JSON.stringify({ deleted }));
}

async function toolSearch(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return toolError("Token lacks required scope: object:read");

  const bucketName = args.bucket as string;
  const q = (args.query as string) || "";
  const type = (args.type as string) || "";
  if (!bucketName) return toolError("bucket is required");
  if (!q && !type) return toolError("At least one of query or type is required");

  const bucket = await resolveBucket(c, bucketName);
  if (!bucket) return toolError("Bucket not found: " + bucketName);

  let sql = "SELECT path, name, content_type, size, created_at, updated_at FROM objects WHERE bucket_id = ?";
  const binds: any[] = [bucket.id];

  if (q) {
    sql += " AND name LIKE ?";
    binds.push(`%${q}%`);
  }
  if (type) {
    sql += " AND content_type LIKE ?";
    binds.push(`${type}%`);
  }

  sql += " ORDER BY updated_at DESC LIMIT 50";

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all();

  const items = (results || []).map((o: any) => ({
    name: o.name,
    path: o.path,
    content_type: o.content_type,
    size: o.size,
    updated_at: o.updated_at,
  }));

  return toolResult(JSON.stringify({ count: items.length, items }, null, 2));
}

async function toolMove(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return toolError("Token lacks required scope: object:write");

  const actor = c.get("actor");
  const bucketName = args.bucket as string;
  const from = args.from as string;
  const to = args.to as string;

  if (!bucketName || !from || !to) return toolError("bucket, from, and to are required");

  const bucket = await resolveBucket(c, bucketName);
  if (!bucket) return toolError("Bucket not found: " + bucketName);

  const obj = await c.env.DB
    .prepare("SELECT * FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, from)
    .first<ObjectRow>();

  if (!obj) return toolError("Object not found: " + from);

  const conflict = await c.env.DB
    .prepare("SELECT 1 FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, to)
    .first();
  if (conflict) return toolError("Target already exists: " + to);

  const newR2Key = `${actor}/${bucket.name}/${to}`;
  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (r2Obj) {
    await c.env.BUCKET.put(newR2Key, r2Obj.body, {
      httpMetadata: { contentType: obj.content_type },
    });
    await c.env.BUCKET.delete(obj.r2_key);
  }

  const newName = to.split("/").pop() || to;
  const now = Date.now();
  await c.env.DB
    .prepare("UPDATE objects SET path = ?, name = ?, r2_key = ?, updated_at = ? WHERE id = ?")
    .bind(to, newName, newR2Key, now, obj.id)
    .run();

  return toolResult(JSON.stringify({ old_path: from, new_path: to }));
}

async function toolShare(c: AppContext, args: Record<string, any>) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return toolError("Token lacks required scope: object:read");

  const actor = c.get("actor");
  const bucketName = args.bucket as string;
  const filePath = args.path as string;
  if (!bucketName || !filePath) return toolError("bucket and path are required");

  const bucket = await resolveBucket(c, bucketName);
  if (!bucket) return toolError("Bucket not found: " + bucketName);

  const obj = await c.env.DB
    .prepare("SELECT 1 FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first();
  if (!obj) return toolError("Object not found: " + filePath);

  const expiresIn = Math.min((args.expires_in as number) || 3600, 7 * 24 * 3600);
  const now = Date.now();
  const expiresAt = now + expiresIn * 1000;
  const id = signedUrlId();
  const token = signedUrlToken();

  await c.env.DB.prepare(
    "INSERT INTO signed_urls (id, owner, bucket_id, path, token, type, expires_at, created_at) VALUES (?, ?, ?, ?, ?, 'download', ?, ?)",
  ).bind(id, actor, bucket.id, filePath, token, expiresAt, now).run();

  const origin = new URL(c.req.url).origin;
  const url = `${origin}/sign/${token}`;

  return toolResult(JSON.stringify({
    url,
    token,
    bucket: bucketName,
    path: filePath,
    expires_at: new Date(expiresAt).toISOString(),
  }, null, 2));
}

async function toolStats(c: AppContext) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return toolError("Token lacks required scope: object:read");

  const actor = c.get("actor");

  const bucketCount = await c.env.DB
    .prepare("SELECT COUNT(*) as count FROM buckets WHERE owner = ?")
    .bind(actor)
    .first<{ count: number }>();

  const stats = await c.env.DB
    .prepare("SELECT COUNT(*) as file_count, COALESCE(SUM(size),0) as total_size FROM objects WHERE owner = ?")
    .bind(actor)
    .first<{ file_count: number; total_size: number }>();

  return toolResult(JSON.stringify({
    actor,
    bucket_count: bucketCount?.count || 0,
    object_count: stats?.file_count || 0,
    total_size: stats?.total_size || 0,
    total_size_human: humanSize(stats?.total_size || 0),
    quota: 5 * 1024 * 1024 * 1024,
    quota_human: "5 GB",
  }, null, 2));
}
