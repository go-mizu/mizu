import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { mimeFromName, isInlineType } from "../lib/mime";
import { shareToken } from "../lib/id";
import { presignUrl } from "../lib/presign";
import { invalidateCache } from "./find";
import { validatePath } from "../lib/path";
import { getWidgetHtml, WIDGET_RESOURCES, WIDGET_RESOURCE_META, TOOL_WIDGET_MAP } from "../mcp-widgets";

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

interface ToolAnnotations {
  title?: string;
  readOnlyHint?: boolean;
  destructiveHint?: boolean;
  openWorldHint?: boolean;
  idempotentHint?: boolean;
}

interface ToolDef {
  name: string;
  description: string;
  inputSchema: Record<string, any>;
  annotations?: ToolAnnotations;
  _meta?: Record<string, any>;
}

// ── Protocol constants ────────────────────────────────────────────────

const PROTOCOL_VERSION = "2025-06-18";
const SERVER_NAME = "Storage";
const SERVER_VERSION = "3.0.0";

// Simple in-memory session tracking (maps session ID → actor)
const mcpSessions = new Map<string, { actor: string; created: number }>();
const SESSION_TTL_MS = 3_600_000; // 1 hour

// ── Tool definitions ──────────────────────────────────────────────────

const TOOLS: ToolDef[] = [
  {
    name: "storage_list",
    description:
      "List files and folders in the user's cloud storage. You MUST call this tool whenever the user asks what files they have, what's in a folder, or wants to browse their storage. " +
      "Call with no prefix to list the root. Pass a folder prefix (e.g. 'photos/') to list that folder's contents. " +
      "Returns immediate children — each entry has a name, MIME type (or 'directory'), size, and last-modified timestamp. " +
      "If an entry has type 'directory', you can call storage_list again with that folder as prefix to drill in.",
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
    annotations: { title: "List files", readOnlyHint: true, destructiveHint: false, openWorldHint: false, idempotentHint: true },
    _meta: {
      ui: { resourceUri: TOOL_WIDGET_MAP.storage_list.uri },
      "openai/outputTemplate": TOOL_WIDGET_MAP.storage_list.uri,
      "openai/toolInvocation/invoking": TOOL_WIDGET_MAP.storage_list.invoking,
      "openai/toolInvocation/invoked": TOOL_WIDGET_MAP.storage_list.invoked,
      "openai/widgetDescription": TOOL_WIDGET_MAP.storage_list.widgetDescription,
    },
  },
  {
    name: "storage_read",
    description:
      "Read a file from the user's cloud storage and return its contents. Call this when the user asks to see, view, open, or read a file. " +
      "For text files (plain text, markdown, JSON, code, etc.) the full content is returned inline so you can read and discuss it. " +
      "For binary files (images, PDFs, zips, etc.) only metadata is returned — suggest using storage_share to generate a download link instead. " +
      "If you are not sure of the exact path, call storage_list or storage_search first.",
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
    annotations: { title: "Read file", readOnlyHint: true, destructiveHint: false, openWorldHint: false, idempotentHint: true },
    _meta: {
      ui: { resourceUri: TOOL_WIDGET_MAP.storage_read.uri },
      "openai/outputTemplate": TOOL_WIDGET_MAP.storage_read.uri,
      "openai/toolInvocation/invoking": TOOL_WIDGET_MAP.storage_read.invoking,
      "openai/toolInvocation/invoked": TOOL_WIDGET_MAP.storage_read.invoked,
      "openai/widgetDescription": TOOL_WIDGET_MAP.storage_read.widgetDescription,
    },
  },
  {
    name: "storage_write",
    description:
      "Save a file to the user's cloud storage (Storage). Creates or overwrites. Two modes:\n" +
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
    annotations: { title: "Save file", readOnlyHint: false, destructiveHint: false, openWorldHint: false, idempotentHint: true },
    _meta: {
      ui: { resourceUri: TOOL_WIDGET_MAP.storage_write.uri },
      "openai/outputTemplate": TOOL_WIDGET_MAP.storage_write.uri,
      "openai/toolInvocation/invoking": TOOL_WIDGET_MAP.storage_write.invoking,
      "openai/toolInvocation/invoked": TOOL_WIDGET_MAP.storage_write.invoked,
      "openai/widgetDescription": TOOL_WIDGET_MAP.storage_write.widgetDescription,
    },
  },
  {
    name: "storage_delete",
    description:
      "Delete files or folders from the user's cloud storage (Storage). " +
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
    annotations: { title: "Delete files", readOnlyHint: false, destructiveHint: true, openWorldHint: false, idempotentHint: true },
    _meta: {
      ui: { resourceUri: TOOL_WIDGET_MAP.storage_delete.uri },
      "openai/outputTemplate": TOOL_WIDGET_MAP.storage_delete.uri,
      "openai/toolInvocation/invoking": TOOL_WIDGET_MAP.storage_delete.invoking,
      "openai/toolInvocation/invoked": TOOL_WIDGET_MAP.storage_delete.invoked,
      "openai/widgetDescription": TOOL_WIDGET_MAP.storage_delete.widgetDescription,
    },
  },
  {
    name: "storage_search",
    description:
      "Search the user's cloud storage by file name or folder name. Call this whenever the user mentions a specific file or folder name and you need to find it. " +
      "Performs a partial, case-insensitive match against both file names AND full paths — so searching 'taocp' finds all files inside a 'taocp/' folder. " +
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
    annotations: { title: "Search files", readOnlyHint: true, destructiveHint: false, openWorldHint: false, idempotentHint: true },
    _meta: {
      ui: { resourceUri: TOOL_WIDGET_MAP.storage_search.uri },
      "openai/outputTemplate": TOOL_WIDGET_MAP.storage_search.uri,
      "openai/toolInvocation/invoking": TOOL_WIDGET_MAP.storage_search.invoking,
      "openai/toolInvocation/invoked": TOOL_WIDGET_MAP.storage_search.invoked,
      "openai/widgetDescription": TOOL_WIDGET_MAP.storage_search.widgetDescription,
    },
  },
  {
    name: "storage_move",
    description:
      "Move or rename a file in the user's cloud storage (Storage). " +
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
    annotations: { title: "Move file", readOnlyHint: false, destructiveHint: false, openWorldHint: false, idempotentHint: true },
    _meta: {
      ui: { resourceUri: TOOL_WIDGET_MAP.storage_move.uri },
      "openai/outputTemplate": TOOL_WIDGET_MAP.storage_move.uri,
      "openai/toolInvocation/invoking": TOOL_WIDGET_MAP.storage_move.invoking,
      "openai/toolInvocation/invoked": TOOL_WIDGET_MAP.storage_move.invoked,
      "openai/widgetDescription": TOOL_WIDGET_MAP.storage_move.widgetDescription,
    },
  },
  {
    name: "storage_share",
    description:
      "Generate a public share link for a file in the user's cloud storage (Storage). " +
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
    annotations: { title: "Share file", readOnlyHint: false, destructiveHint: false, openWorldHint: true, idempotentHint: false },
    _meta: {
      ui: { resourceUri: TOOL_WIDGET_MAP.storage_share.uri },
      "openai/outputTemplate": TOOL_WIDGET_MAP.storage_share.uri,
      "openai/toolInvocation/invoking": TOOL_WIDGET_MAP.storage_share.invoking,
      "openai/toolInvocation/invoked": TOOL_WIDGET_MAP.storage_share.invoked,
      "openai/widgetDescription": TOOL_WIDGET_MAP.storage_share.widgetDescription,
    },
  },
  {
    name: "storage_stats",
    description:
      "Show the user's storage usage — total number of files and total size in bytes. " +
      "Call this when the user asks how much storage they're using, how many files they have, or about their quota.",
    inputSchema: {
      type: "object",
      properties: {},
    },
    annotations: { title: "Storage usage", readOnlyHint: true, destructiveHint: false, openWorldHint: false, idempotentHint: true },
    _meta: {
      ui: { resourceUri: TOOL_WIDGET_MAP.storage_stats.uri },
      "openai/outputTemplate": TOOL_WIDGET_MAP.storage_stats.uri,
      "openai/toolInvocation/invoking": TOOL_WIDGET_MAP.storage_stats.invoking,
      "openai/toolInvocation/invoked": TOOL_WIDGET_MAP.storage_stats.invoked,
      "openai/widgetDescription": TOOL_WIDGET_MAP.storage_stats.widgetDescription,
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
          capabilities: {
            tools: { listChanged: false },
            resources: { listChanged: false },
          },
          instructions: [
            "You are ALREADY connected to the user's cloud file storage (Storage). The connection is live and working — you do NOT need to ask the user to connect, upload, or link anything. Just call the tools directly.",
            "",
            "IMPORTANT: Whenever the user mentions files, folders, documents, storage, \"my files\", \"my storage\", \"cloud storage\", or asks about any file or folder by name — ALWAYS call a storage tool first. Never say you don't have access. You DO have access right now through these tools.",
            "",
            "When to use each tool:",
            "• User asks \"what's in my storage\" or \"list files\" or \"what files do I have\" → call storage_list",
            "• User asks \"what's in [folder]\" or \"show me [folder]\" → call storage_list with that folder as prefix",
            "• User mentions a file/folder name and you need to find it → call storage_search with the name",
            "• User wants to read/view/show file contents → call storage_read",
            "• User wants to save/write/create a file → call storage_write",
            "• User wants to share a file or get a link → call storage_share, then ALWAYS show the returned URL",
            "• User asks about storage usage or space → call storage_stats",
            "• User wants to rename or move a file → call storage_move",
            "• User wants to delete a file → confirm first, then call storage_delete",
            "",
            "Path format: no leading slash, forward slashes for folders. Example: 'docs/notes.md'.",
            "If storage_list returns a folder, you can list its contents by calling storage_list again with that folder as prefix (e.g. prefix: 'taocp/').",
            "To explore nested folders, call storage_list repeatedly, drilling into each subfolder.",
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
      const params = req.params as { name: string; arguments?: Record<string, any>; _meta?: Record<string, any> } | undefined;
      if (!params?.name) {
        return rpcResponse(c, { jsonrpc: "2.0", id: req.id, error: { code: -32602, message: "Invalid params: missing tool name" } });
      }
      const result = await callTool(c, params.name, params.arguments || {});
      return rpcResponse(c, { jsonrpc: "2.0", id: req.id, result });
    }

    // ── Resource methods (ChatGPT widget HTML) ──────────────────────
    case "resources/list":
      return rpcResponse(c, {
        jsonrpc: "2.0",
        id: req.id,
        result: {
          resources: WIDGET_RESOURCES.map((r) => ({
            uri: r.uri,
            name: r.name,
            description: r.description,
            mimeType: r.mimeType,
          })),
        },
      });

    case "resources/read": {
      const uri = (req.params as any)?.uri as string;
      if (!uri) {
        return rpcResponse(c, { jsonrpc: "2.0", id: req.id, error: { code: -32602, message: "Invalid params: missing uri" } });
      }
      const html = getWidgetHtml(uri);
      if (!html) {
        return rpcResponse(c, { jsonrpc: "2.0", id: req.id, error: { code: -32602, message: "Resource not found: " + uri } });
      }
      return rpcResponse(c, {
        jsonrpc: "2.0",
        id: req.id,
        result: {
          contents: [{
            uri,
            mimeType: "text/html;profile=mcp-app",
            text: html,
            _meta: WIDGET_RESOURCE_META,
          }],
        },
      });
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

/** Return both text (for standard MCP clients) and structured content (for ChatGPT widgets). */
function richResult(text: string, structured: any, privateMeta?: any): any {
  const result: any = { content: [{ type: "text", text }], isError: false, structuredContent: structured };
  if (privateMeta) result._meta = privateMeta;
  return result;
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

/** Strip leading slashes — LLMs often send /path instead of path */
function cleanPath(p: string): string {
  return p.replace(/^\/+/, "");
}

// ── Tool implementations ──────────────────────────────────────────────

async function toolList(c: C, args: Record<string, any>) {
  const actor = c.get("actor");
  // Accept prefix, path, or folder — LLMs often use "path" instead of "prefix"
  const rawPrefix = (args.prefix as string) || (args.path as string) || (args.folder as string) || "";
  const prefix = rawPrefix.replace(/^\/+/, ""); // strip leading slashes
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

  const structured = { prefix: prefix || "/", entries };
  return richResult(JSON.stringify(structured, null, 2), structured);
}

async function toolRead(c: C, args: Record<string, any>) {
  const filePath = cleanPath((args.path as string) || "");
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

  const meta = `path: ${filePath}\nsize: ${r2Obj.size} bytes\ncontent_type: ${ct}`;

  const structured: any = { path: filePath, size: r2Obj.size, content_type: ct, is_text: isText };

  if (isText) {
    const text = await r2Obj.text();
    const truncated = text.length > 102400;
    const content = truncated ? text.slice(0, 102400) + "\n... (truncated, " + text.length + " bytes total)" : text;
    return richResult(meta + "\n---\n" + content, structured, { fileContent: content });
  }

  // For binary files, return a presigned download URL if credentials are available
  const endpoint = c.env.R2_ENDPOINT;
  const accessKeyId = c.env.R2_ACCESS_KEY_ID;
  const secretAccessKey = c.env.R2_SECRET_ACCESS_KEY;
  if (endpoint && accessKeyId && secretAccessKey) {
    const dlUrl = await presignUrl({
      method: "GET",
      key: `${actor}/${filePath}`,
      bucket: c.env.R2_BUCKET_NAME || "storage-files",
      endpoint,
      accessKeyId,
      secretAccessKey,
      expiresIn: 3600,
    });
    structured.download_url = dlUrl;
    return richResult(meta + "\n---\n[Binary file — direct download link (expires in 1 hour):\n" + dlUrl + "]", structured, { downloadUrl: dlUrl });
  }

  return richResult(meta + "\n---\n[Binary file, " + r2Obj.size + " bytes. Use storage_share to generate a download link.]", structured);
}

async function toolWrite(c: C, args: Record<string, any>) {
  const actor = c.get("actor");
  const filePath = cleanPath((args.path as string) || "");
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

  const structured = { path: filePath, name, content_type: contentType, size: body.byteLength, updated_at: now };
  return richResult(JSON.stringify(structured), structured);
}

async function toolDelete(c: C, args: Record<string, any>) {
  const rawPaths = (args.paths as string[]) || (args.path ? [args.path as string] : []);
  if (!rawPaths.length) return toolError("paths is required");

  const actor = c.get("actor");
  const deleted: string[] = [];

  for (const rawPath of rawPaths) {
    const path = cleanPath(rawPath);
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
  const structured = { deleted };
  return richResult(JSON.stringify(structured), structured);
}

async function toolSearch(c: C, args: Record<string, any>) {
  const q = (args.query as string) || (args.q as string) || "";
  if (!q) return toolError("query is required");

  const actor = c.get("actor");
  const limit = Math.min((args.limit as number) || 50, 200);
  const pfx = c.get("prefix") || "";

  let sql = "SELECT path, name, size, type, updated_at FROM files WHERE owner = ? AND (name LIKE ? OR path LIKE ?)";
  const binds: any[] = [actor, `%${q}%`, `%${q}%`];

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

  const structured = { query: q, count: items.length, items };
  return richResult(JSON.stringify(structured, null, 2), structured);
}

async function toolMove(c: C, args: Record<string, any>) {
  const actor = c.get("actor");
  const from = cleanPath((args.from as string) || (args.source as string) || "");
  const to = cleanPath((args.to as string) || (args.destination as string) || (args.dest as string) || "");

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
  const structured = { old_path: from, new_path: to };
  return richResult(JSON.stringify(structured), structured);
}

async function toolShare(c: C, args: Record<string, any>) {
  const actor = c.get("actor");
  const filePath = cleanPath((args.path as string) || "");
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

  const structured = { url, token, path: filePath, expires_at: new Date(expiresAt).toISOString() };
  return richResult(JSON.stringify(structured, null, 2), structured);
}

async function toolStats(c: C) {
  const actor = c.get("actor");

  const stats = await c.env.DB
    .prepare("SELECT COUNT(*) as file_count, COALESCE(SUM(size),0) as total_size FROM files WHERE owner = ?")
    .bind(actor)
    .first<{ file_count: number; total_size: number }>();

  const structured = {
    file_count: stats?.file_count || 0,
    total_size: stats?.total_size || 0,
    total_size_human: humanSize(stats?.total_size || 0),
  };
  return richResult(JSON.stringify(structured, null, 2), structured);
}

// ── Registration ──────────────────────────────────────────────────────

export function register(app: App) {
  app.get("/mcp", auth, mcpGet);
  app.post("/mcp", auth, mcpHandler);
  app.delete("/mcp", auth, mcpDelete);
}
