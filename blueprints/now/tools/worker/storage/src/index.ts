import { OpenAPIHono } from "@hono/zod-openapi";
import { cors } from "hono/cors";
import { register as registerFiles } from "./routes/files-v2";
import { register as registerAuth } from "./routes/auth";
import { register as registerKeys } from "./routes/keys";
import { register as registerMcp } from "./routes/mcp";
import { register as registerOAuth } from "./routes/oauth";
import { register as registerCli } from "./routes/cli";
import { register as registerMagic } from "./routes/magic";
import { register as registerShare } from "./routes/share";
import { getSessionActor } from "./pages/session";
import { homePage } from "./pages/home";
import { developersPage } from "./pages/developers";
import { pricingPage } from "./pages/pricing";
import { aiPage } from "./pages/ai";
import { cliPage } from "./pages/cli";
import { browsePage } from "./pages/browse";
import { privacyPage } from "./pages/privacy";
import type { Env, Variables } from "./types";

const app = new OpenAPIHono<{ Bindings: Env; Variables: Variables }>();

app.use("*", cors());

// ── Pages (no auth — session read optionally for signed-in state) ───
app.get("/", async (c) => {
  const actor = await getSessionActor(c);
  return c.html(homePage(actor));
});
app.get("/developers", async (c) => {
  const actor = await getSessionActor(c);
  return c.html(developersPage(actor));
});
app.get("/pricing", (c) => c.html(pricingPage()));
app.get("/ai", (c) => c.html(aiPage()));
app.get("/privacy", (c) => c.html(privacyPage()));
app.get("/cli", async (c) => {
  const actor = await getSessionActor(c);
  return c.html(cliPage(actor));
});
app.get("/browse", browsePage as any);
app.get("/browse/*", browsePage as any);

// ── Storage API routes (/files/*) ────────────────────────────────────
registerFiles(app);

// ── Share access (public, no auth) ──────────────────────────────────
registerShare(app);

// ── Auth / MCP / OAuth / CLI ────────────────────────────────────────
registerAuth(app);
registerKeys(app);
registerMcp(app);
registerOAuth(app);
registerCli(app);
registerMagic(app);

// ── OpenAPI spec (auto-generated from route definitions) ────────────
app.doc("/openapi.json", {
  openapi: "3.1.0",
  info: {
    title: "Storage API",
    version: "1.0.0",
    description: "File storage on the edge. Upload via presigned URLs, download via redirect.",
  },
});

// ── Swagger UI ──────────────────────────────────────────────────────
app.get("/docs", (c) => {
  const origin = new URL(c.req.url).origin;
  return c.html(`<!doctype html><html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Storage API</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head><body>
<div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({url:"${origin}/openapi.json",dom_id:"#swagger-ui",deepLinking:true})</script>
</body></html>`);
});

// ── API Reference (auto-generated from OpenAPI spec) ─────────────────
app.get("/api", (c) => {
  const origin = new URL(c.req.url).origin;
  const spec = app.getOpenAPIDocument({
    openapi: "3.1.0",
    info: { title: "Storage API", version: "1.0.0", description: "File storage on the edge." },
  });
  return c.html(renderApiReference(spec, origin));
});

// ── Fallbacks ───────────────────────────────────────────────────────
app.notFound((c) => c.json({ error: "not_found", message: "Not found" }, 404));

app.onError((e, c) => {
  const method = c.req.method;
  const url = c.req.url;
  const actor = (() => { try { return c.get("actor"); } catch { return "anon"; } })();
  console.error(JSON.stringify({
    level: "error",
    method,
    url,
    actor,
    error: e?.message || String(e),
    stack: e?.stack,
    ts: Date.now(),
  }));
  return c.json({ error: "internal", message: "Internal server error" }, 500);
});

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>;

// ── API Reference renderer ───────────────────────────────────────────

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

function resolveRefs(obj: any, root: any): any {
  if (!obj || typeof obj !== "object") return obj;
  if (obj.$ref) {
    const path = (obj.$ref as string).replace("#/", "").split("/");
    let resolved = root;
    for (const seg of path) resolved = resolved?.[seg];
    return resolved ?? obj;
  }
  if (Array.isArray(obj)) return obj.map((v) => resolveRefs(v, root));
  const out: any = {};
  for (const [k, v] of Object.entries(obj)) out[k] = resolveRefs(v, root);
  return out;
}

function exampleFromSchema(schema: any): any {
  if (!schema || typeof schema !== "object") return null;
  if (schema.example !== undefined) return schema.example;
  if (schema.type === "array") {
    const item = exampleFromSchema(schema.items);
    return item != null ? [item] : [];
  }
  if (schema.type === "object" && schema.properties) {
    const obj: Record<string, any> = {};
    for (const [k, v] of Object.entries(schema.properties)) {
      obj[k] = exampleFromSchema(v as any);
    }
    return obj;
  }
  if (schema.enum?.length) return schema.enum[0];
  if (schema.type === "string") return "string";
  if (schema.type === "number" || schema.type === "integer") return 0;
  if (schema.type === "boolean") return true;
  return null;
}

function slug(method: string, path: string): string {
  return `${method}-${path.replace(/[^a-z0-9]+/gi, "-").replace(/^-|-$/g, "")}`;
}

// Syntax highlighting for curl commands
function highlightCurl(code: string): string {
  return esc(code)
    .replace(/^(curl)\b/gm, `<span class="sh-cmd">$1</span>`)
    .replace(/(-[XHdF])\s/g, `<span class="sh-flag">$1</span> `)
    .replace(/(&quot;[^&]*&quot;)/g, `<span class="sh-str">$1</span>`)
    .replace(/(\$[A-Z_]+)/g, `<span class="sh-var">$1</span>`);
}

// Syntax highlighting for JSON
function highlightJson(code: string): string {
  return esc(code)
    .replace(/(&quot;[^&]*&quot;)\s*:/g, `<span class="js-key">$1</span>:`)
    .replace(/:\s*(&quot;[^&]*&quot;)/g, `: <span class="js-str">$1</span>`)
    .replace(/:\s*(\d+)/g, `: <span class="js-num">$1</span>`)
    .replace(/:\s*(true|false|null)\b/g, `: <span class="js-bool">$1</span>`)
    .replace(/\[\s*(&quot;[^&]*&quot;)\s*\]/g, `[<span class="js-str">$1</span>]`);
}

// Add line numbers to code
function withLineNumbers(html: string): string {
  const lines = html.split("\n");
  return lines.map((l, i) => `<span class="ln">${i + 1}</span>${l}`).join("\n");
}

const TAG_LABELS: Record<string, string> = {
  files: "Files",
  uploads: "Uploads",
  sharing: "Sharing",
  auth: "Authentication",
  keys: "API Keys",
};

const TAG_ORDER = ["files", "uploads", "sharing", "auth", "keys"];

const METHOD_COLORS: Record<string, [string, string]> = {
  GET: ["#10b981", "#ecfdf5"],
  POST: ["#3b82f6", "#eff6ff"],
  PUT: ["#f59e0b", "#fffbeb"],
  PATCH: ["#f59e0b", "#fffbeb"],
  DELETE: ["#ef4444", "#fef2f2"],
  HEAD: ["#8b5cf6", "#f5f3ff"],
};

const METHOD_COLORS_DARK: Record<string, [string, string]> = {
  GET: ["#34d399", "#064e3b"],
  POST: ["#60a5fa", "#1e3a5f"],
  PUT: ["#fbbf24", "#451a03"],
  PATCH: ["#fbbf24", "#451a03"],
  DELETE: ["#f87171", "#450a0a"],
  HEAD: ["#a78bfa", "#2e1065"],
};

const DESCRIPTIONS: Record<string, string> = {
  "List files": "Returns a list of files and folders in the authenticated user's storage. Results are paginated and can be filtered by prefix to list contents of a specific folder.",
  "Search files": "Searches for files by name across the authenticated user's storage. Supports multi-word queries with relevance scoring. Results are sorted by match quality.",
  "Retrieve storage stats": "Returns aggregate storage statistics for the authenticated user, including total file count and total bytes used.",
  "Move a file": "Moves or renames a file within the authenticated user's storage. Both the source and destination paths must be valid. The file's content and metadata are preserved.",
  "Share a file": "Creates a temporary, public share link for a file. The link expires after the specified TTL (default 1 hour, maximum 7 days). Anyone with the link can download the file.",
  "Retrieve a file": "Downloads a file by path. By default returns a <code>302</code> redirect to a presigned R2 URL. With <code>Accept: application/json</code>, returns metadata including the presigned URL, file size, content type, and ETag.",
  "Retrieve file metadata": "Returns file metadata in HTTP response headers (<code>Content-Type</code>, <code>Content-Length</code>, <code>ETag</code>) without downloading the file body. Useful for checking if a file exists or getting its size.",
  "Delete a file": "Permanently deletes a file or folder. For folders (paths ending with <code>/</code>), recursively deletes all contents. This action cannot be undone.",
  "Create a folder": "Creates an empty folder marker in storage. The path must end with <code>/</code>. Folders are virtual — they exist as zero-byte objects in R2.",
  "Create an upload": "Initiates a file upload by generating a presigned PUT URL. Upload the file directly to this URL using an HTTP PUT request, then call Complete an upload to index it in the database.",
  "Complete an upload": "Confirms a file upload after the file has been uploaded to the presigned URL. Verifies the object exists in R2 and indexes it in the database with its metadata.",
  "Create a multipart upload": "Initiates a multipart upload for large files. Returns presigned URLs for each part. Upload parts in parallel for faster transfers, then call Complete a multipart upload.",
  "Complete a multipart upload": "Finalizes a multipart upload by assembling all uploaded parts into a single object. You must provide the ETag returned from each part upload.",
  "Abort a multipart upload": "Cancels an in-progress multipart upload and cleans up any parts that have already been uploaded.",
  "Register an account": "Creates a new account with an Ed25519 public key for cryptographic signature-based authentication. The actor name must be 1-64 characters, alphanumeric with hyphens and underscores.",
  "Create a challenge": "Issues a cryptographic nonce for Ed25519 signature verification. The challenge expires after 5 minutes. Sign the nonce with your private key and submit via Verify a signature.",
  "Verify a signature": "Verifies an Ed25519 signature of the challenge nonce and issues a session token. The session token can be used as a Bearer token for authenticated requests.",
  "Log out": "Invalidates the current session token. Accepts the token from either the Authorization header or a session cookie.",
  "Request a magic link": "Sends a passwordless sign-in link to the provided email address. The link is single-use and expires after 15 minutes. A generic response is always returned to prevent email enumeration.",
  "Create an API key": "Creates a new API key for programmatic access. The key token is returned exactly once in the response — store it securely, as it cannot be retrieved again.",
  "List API keys": "Returns all API keys for the authenticated user. Key tokens (secrets) are never included in the response — only metadata like name, prefix scope, and expiry.",
  "Delete an API key": "Permanently revokes an API key. Any requests using this key will immediately start returning 401 Unauthorized.",
  "Access a shared file": "Accesses a file using a share token created via Share a file. Returns a <code>302</code> redirect to a presigned R2 URL for downloading. No authentication is required.",
};

function renderParamType(prop: any): string {
  if (!prop) return "string";
  if (prop.enum) return prop.enum.map((v: any) => `<code>${esc(String(v))}</code>`).join(" or ");
  let t = prop.type || "string";
  if (t === "array") {
    const itemType = (prop.items as any)?.type || "object";
    if (prop.items?.properties) {
      const fields = Object.keys(prop.items.properties).slice(0, 3).join(", ");
      t = `array of object { ${fields}, ... }`;
    } else {
      t = `array of ${itemType}`;
    }
  }
  if (prop.nullable) t += " | null";
  return esc(t);
}

function renderParams(params: any[], label: string): string {
  if (!params.length) return "";
  let h = `<h4>${esc(label)}</h4><div class="param-list">`;
  for (const p of params) {
    const s = p.schema || {};
    const req = p.required ? "" : `<span class="param-optional">optional</span> `;
    const desc = p.description || s.description || "";
    const def = s.default !== undefined ? ` Defaults to <code>${esc(String(s.default))}</code>.` : "";
    h += `<div class="param-item"><div class="param-header"><span class="param-name">${esc(p.name)}</span>: ${req}<span class="param-type">${renderParamType(s)}</span></div>${desc || def ? `<p class="param-desc">${esc(desc)}${def}</p>` : ""}</div>`;
  }
  return h + `</div>`;
}

function renderBodyParams(schema: any): string {
  if (!schema?.properties) return "";
  const required = new Set(schema.required || []);
  let h = `<h4>Body Parameters</h4><div class="param-list">`;
  for (const [key, rawProp] of Object.entries(schema.properties)) {
    const prop = rawProp as any;
    const req = required.has(key) ? "" : `<span class="param-optional">optional</span> `;
    const desc = prop.description || "";
    const def = prop.default !== undefined ? ` Defaults to <code>${esc(String(prop.default))}</code>.` : "";
    h += `<div class="param-item"><div class="param-header"><span class="param-name">${esc(key)}</span>: ${req}<span class="param-type">${renderParamType(prop)}</span></div>${desc || def ? `<p class="param-desc">${esc(desc)}${def}</p>` : ""}</div>`;
  }
  return h + `</div>`;
}

function renderReturns(responses: any): string {
  if (!responses) return "";
  let h = "";
  for (const [code, rawResp] of Object.entries(responses)) {
    const resp = rawResp as any;
    const rSchema = resp.content?.["application/json"]?.schema;
    const badge = `<span class="response-code code-${code[0]}">${esc(code)}</span>`;
    if (rSchema?.properties) {
      h += `<h4>Returns ${badge}</h4>`;
      if (resp.description) h += `<p class="param-desc">${esc(resp.description)}</p>`;
      h += `<div class="param-list">`;
      for (const [key, rawProp] of Object.entries(rSchema.properties)) {
        const prop = rawProp as any;
        const desc = prop.description || "";
        h += `<div class="param-item"><div class="param-header"><span class="param-name">${esc(key)}</span>: <span class="param-type">${renderParamType(prop)}</span></div>${desc ? `<p class="param-desc">${esc(desc)}</p>` : ""}</div>`;
      }
      h += `</div>`;
    } else if (resp.description && code !== "200" && code !== "201") {
      h += `<p class="response-line">${badge} ${esc(resp.description)}</p>`;
    } else if (resp.description && !rSchema) {
      h += `<h4>Returns ${badge}</h4><p class="param-desc">${esc(resp.description)}</p>`;
    }
  }
  return h;
}

// Generate full markdown representation of the API docs
function specToMarkdown(spec: any, origin: string): string {
  const md: string[] = [];
  md.push("# Storage API Reference\n");
  md.push(`Base URL: ${origin}\n`);
  md.push("## Authentication\n");
  md.push("All authenticated endpoints require a Bearer token:\n");
  md.push("```\nAuthorization: Bearer STORAGE_API_KEY\n```\n");
  const byTag = new Map<string, { method: string; path: string; op: any }[]>();
  for (const [path, methods] of Object.entries(spec.paths || {})) {
    for (const [method, rawOp] of Object.entries(methods as any)) {
      if (typeof rawOp !== "object" || !rawOp) continue;
      const op = rawOp as any;
      const tag = op.tags?.[0] || "other";
      if (!byTag.has(tag)) byTag.set(tag, []);
      byTag.get(tag)!.push({ method: method.toUpperCase(), path, op });
    }
  }
  for (const tag of TAG_ORDER) {
    const routes = byTag.get(tag);
    if (!routes) continue;
    md.push(`## ${TAG_LABELS[tag] || tag}\n`);
    for (const { method, path, op } of routes) {
      md.push(`### ${op.summary || `${method} ${path}`}\n`);
      md.push(`\`${method} ${path}\`\n`);
      const desc = DESCRIPTIONS[op.summary || ""];
      if (desc) md.push(`${desc.replace(/<[^>]+>/g, "")}\n`);
      if (op.security?.length) md.push("**Requires authentication**\n");
      const params = op.parameters || [];
      if (params.length) {
        md.push("**Parameters:**\n");
        for (const p of params) md.push(`- \`${p.name}\` (${p.in}, ${p.required ? "required" : "optional"}) — ${p.description || p.schema?.description || p.schema?.type || ""}`);
        md.push("");
      }
      const body = op.requestBody?.content?.["application/json"]?.schema;
      if (body?.properties) {
        md.push("**Body:**\n");
        const req = new Set(body.required || []);
        for (const [k, v] of Object.entries(body.properties)) { const p = v as any; md.push(`- \`${k}\` (${req.has(k) ? "required" : "optional"}, ${p.type || "string"}) — ${p.description || ""}`); }
        md.push("");
      }
      let curl = `curl`;
      if (method !== "GET" && method !== "HEAD") curl += ` -X ${method}`;
      curl += ` ${origin}${path}`;
      if (op.security?.length) curl += ` \\\n  -H "Authorization: Bearer $STORAGE_API_KEY"`;
      if (body) { curl += ` \\\n  -H "Content-Type: application/json"`; const ex = exampleFromSchema(body); if (ex) curl += ` \\\n  -d '${JSON.stringify(ex)}'`; }
      md.push("```bash\n" + curl + "\n```\n");
      const sr = (op.responses?.["200"] || op.responses?.["201"]) as any;
      if (sr?.content?.["application/json"]?.schema) { const ex = exampleFromSchema(sr.content["application/json"].schema); if (ex) md.push("```json\n" + JSON.stringify(ex, null, 2) + "\n```\n"); }
      md.push("---\n");
    }
  }
  return md.join("\n");
}

const COPY_ICON = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`;

function renderApiReference(spec: any, origin: string): string {
  const resolved = resolveRefs(spec, spec);

  // Group by tag
  const byTag = new Map<string, { method: string; path: string; op: any }[]>();
  for (const [path, methods] of Object.entries(resolved.paths || {})) {
    for (const [method, rawOp] of Object.entries(methods as any)) {
      if (typeof rawOp !== "object" || !rawOp) continue;
      const op = rawOp as any;
      const tag = op.tags?.[0] || "other";
      if (!byTag.has(tag)) byTag.set(tag, []);
      byTag.get(tag)!.push({ method: method.toUpperCase(), path, op });
    }
  }
  const tagOrder = TAG_ORDER.filter((t) => byTag.has(t));
  for (const t of byTag.keys()) if (!tagOrder.includes(t)) tagOrder.push(t);

  // Sidebar
  let sidebar = `<nav class="sidebar" id="sidebar">
<div class="sidebar-header">
  <a href="/" class="sidebar-logo">Storage API</a>
  <div class="sidebar-actions">
    <button class="theme-toggle" id="theme-toggle" title="Toggle theme" aria-label="Toggle theme">
      <svg class="icon-sun" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
      <svg class="icon-moon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
    </button>
    <button class="sidebar-close" onclick="document.getElementById('sidebar').classList.remove('open')" aria-label="Close">&times;</button>
  </div>
</div>
<ul class="sidebar-nav">
  <li class="sidebar-group">
    <span class="sidebar-group-label-static">API Reference</span>
    <ul>
      <li><a href="#introduction" class="sidebar-link">Introduction</a></li>
      <li><a href="#authentication" class="sidebar-link">Authentication</a></li>
      <li><a href="#content-types" class="sidebar-link">Content types</a></li>
      <li><a href="#errors" class="sidebar-link">Errors</a></li>
    </ul>
  </li>`;

  for (const tag of tagOrder) {
    const routes = byTag.get(tag)!;
    const label = TAG_LABELS[tag] || tag.charAt(0).toUpperCase() + tag.slice(1);
    sidebar += `\n  <li class="sidebar-group">
    <details class="sidebar-details" open>
      <summary class="sidebar-group-label">${esc(label)}<svg class="chevron" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 9l6 6 6-6"/></svg></summary>
      <ul>`;
    for (const { method, path, op } of routes) {
      const id = slug(method, path);
      const [color] = METHOD_COLORS[method] || ["#888", "#f5f5f5"];
      sidebar += `\n        <li><a href="#${id}" class="sidebar-link"><span class="sb-badge" style="color:${color}">${method}</span>${esc(op.summary || `${method} ${path}`)}</a></li>`;
    }
    sidebar += `\n      </ul>
    </details>
  </li>`;
  }
  sidebar += `\n</ul></nav>`;

  // Prose sections
  const prose = `<main class="content" id="content">
<button class="menu-btn" onclick="document.getElementById('sidebar').classList.toggle('open')" aria-label="Menu">&#9776; API Reference</button>

<div class="top-actions">
  <div class="md-dropdown" id="md-dropdown">
    <button class="md-btn" onclick="document.getElementById('md-dropdown').classList.toggle('open')">
      ${COPY_ICON} Copy Markdown
      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 9l6 6 6-6"/></svg>
    </button>
    <div class="md-menu">
      <button onclick="copyMarkdown()">
        ${COPY_ICON} Copy Markdown
      </button>
      <button onclick="viewMarkdown()">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg> View as Markdown
      </button>
    </div>
  </div>
</div>

<article class="prose-section" id="introduction">
  <h1>API Overview</h1>

  <h2>Introduction</h2>
  <p>This API reference describes the REST APIs you can use to interact with the Storage platform. REST APIs are usable via HTTP in any environment that supports HTTP requests. The <a href="/cli">CLI</a> and MCP tools are also available for programmatic access.</p>
  <p>The base URL for all API requests is:</p>
  <div class="code-block"><pre><code>${esc(origin)}</code></pre><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
</article>

<article class="prose-section" id="authentication">
  <h2>Authentication</h2>
  <p>The Storage API uses API keys for authentication. Create, manage, and revoke API keys via <a href="#POST-auth-keys"><code>POST /auth/keys</code></a> or through the <a href="/browse">dashboard</a>.</p>
  <p><strong>Remember that your API key is a secret!</strong> Do not share it with others or expose it in any client-side code (browsers, apps). API keys should be securely loaded from an environment variable or key management service on the server.</p>
  <p>API keys should be provided via HTTP Bearer authentication.</p>
  <div class="code-block"><pre><code>Authorization: Bearer STORAGE_API_KEY</code></pre><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
  <p>All authenticated requests should include this header:</p>
  <div class="code-block"><pre><code>${highlightCurl(`curl ${origin}/files \\\n  -H "Authorization: Bearer $STORAGE_API_KEY"`)}</code></pre><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
  <p>API keys can be scoped to a path prefix (e.g. <code>docs/</code>) and set to expire after a specified duration. This lets you create restricted keys for specific use cases like CI/CD pipelines or shared access.</p>
</article>

<article class="prose-section" id="content-types">
  <h2>Content types</h2>
  <p>Request bodies should be sent as JSON with <code>Content-Type: application/json</code>.</p>
  <p>File downloads support content negotiation via the <code>Accept</code> header. By default, <code>GET /files/{path}</code> returns a <code>302</code> redirect to a presigned R2 URL &mdash; browsers and curl follow this redirect to download the file directly.</p>
  <p>Programmatic clients (CLIs, SDKs) can request JSON metadata instead by setting:</p>
  <div class="code-block"><pre><code>Accept: application/json</code></pre><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
  <p>This returns a JSON response with the presigned URL, file size, content type, and ETag &mdash; useful when you need the URL as a string value or want to add custom headers like <code>Range</code>.</p>
</article>

<article class="prose-section" id="errors">
  <h2>Errors</h2>
  <p>The API returns errors as JSON with a consistent shape. HTTP status codes follow standard conventions.</p>
  <div class="code-block"><pre><code>${highlightJson(`{\n  "error": "not_found",\n  "message": "File not found"\n}`)}</code></pre><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
  <table class="error-table">
    <thead><tr><th>Code</th><th>Error</th><th>Meaning</th></tr></thead>
    <tbody>
      <tr><td><span class="response-code code-4">400</span></td><td><code>bad_request</code></td><td>Invalid parameters or request body</td></tr>
      <tr><td><span class="response-code code-4">401</span></td><td><code>unauthorized</code></td><td>Missing or invalid authentication</td></tr>
      <tr><td><span class="response-code code-4">403</span></td><td><code>forbidden</code></td><td>Path not allowed for this token</td></tr>
      <tr><td><span class="response-code code-4">404</span></td><td><code>not_found</code></td><td>File or resource not found</td></tr>
      <tr><td><span class="response-code code-4">409</span></td><td><code>conflict</code></td><td>Resource already exists</td></tr>
      <tr><td><span class="response-code code-4">429</span></td><td><code>rate_limited</code></td><td>Too many requests</td></tr>
      <tr><td><span class="response-code code-5">500</span></td><td><code>internal</code></td><td>Internal server error</td></tr>
    </tbody>
  </table>
</article>`;

  // Endpoints
  let endpoints = "";
  for (const tag of tagOrder) {
    const routes = byTag.get(tag)!;
    const label = TAG_LABELS[tag] || tag.charAt(0).toUpperCase() + tag.slice(1);
    endpoints += `\n<div class="tag-heading" id="tag-${esc(tag)}">${esc(label)}</div>`;

    for (const { method, path, op } of routes) {
      const id = slug(method, path);
      const [mColor, mBg] = METHOD_COLORS[method] || ["#888", "#f5f5f5"];
      const needsAuth = op.security?.length > 0;
      const summary = op.summary || `${method} ${path}`;
      const desc = DESCRIPTIONS[summary] || "";

      const allParams = op.parameters || [];
      const pathP = allParams.filter((p: any) => p.in === "path");
      const queryP = allParams.filter((p: any) => p.in === "query");
      const bodySchema = op.requestBody?.content?.["application/json"]?.schema;

      // Curl
      let curlRaw = "curl";
      if (method !== "GET" && method !== "HEAD") curlRaw += ` -X ${method}`;
      curlRaw += ` ${origin}${path.replace(/\{(\w+)\}/g, ":$1")}`;
      if (needsAuth) curlRaw += ` \\\n  -H "Authorization: Bearer $STORAGE_API_KEY"`;
      if (bodySchema) {
        curlRaw += ` \\\n  -H "Content-Type: application/json"`;
        const ex = exampleFromSchema(bodySchema);
        if (ex) curlRaw += ` \\\n  -d '${JSON.stringify(ex)}'`;
      }

      // Response
      let responseRaw = "";
      const sr = (op.responses?.["200"] || op.responses?.["201"]) as any;
      if (sr?.content?.["application/json"]?.schema) {
        const ex = exampleFromSchema(sr.content["application/json"].schema);
        if (ex) responseRaw = JSON.stringify(ex, null, 2);
      }

      endpoints += `
<section id="${id}" class="endpoint-section">
<div class="desc-col">
  <h3 class="endpoint-title">${esc(summary)}</h3>
  <div class="method-path">
    <span class="method-badge" style="color:${mColor};background:${mBg};border-color:${mColor}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M7 17L17 7M17 7H7M17 7v10"/></svg> ${method}</span>
    <code class="endpoint-path">${esc(path)}</code>
  </div>
  ${desc ? `<p class="endpoint-desc">${desc}</p>` : ""}
  ${renderParams(pathP, "Path Parameters")}
  ${renderParams(queryP, "Query Parameters")}
  ${renderBodyParams(bodySchema)}
  ${renderReturns(op.responses)}
</div>
<div class="example-col">
  <div class="example-block">
    <div class="example-header">
      <span class="example-title">${esc(summary)}</span>
      <button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button>
    </div>
    <pre><code>${withLineNumbers(highlightCurl(curlRaw))}</code></pre>
  </div>
  ${responseRaw ? `<div class="example-block">
    <pre><code>${withLineNumbers(highlightJson(responseRaw))}</code></pre>
  </div>` : ""}
</div>
</section>`;
    }
  }

  // Generate markdown data for copy/view
  const markdownData = specToMarkdown(resolved, origin).replace(/\\/g, "\\\\").replace(/`/g, "\\`").replace(/\$/g, "\\$");

  return `<!doctype html><html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>API Reference &mdash; Storage</title>
<style>
:root{
  --bg:#fff;--bg2:#f9fafb;--bg3:#f3f4f6;--fg:#111;--fg2:#4b5563;--fg3:#9ca3af;
  --border:#e5e7eb;--code-bg:#f8f9fa;--code-fg:#24292e;
  --ex-bg:#fafafa;--ex-fg:#24292e;--ex-border:#e5e7eb;--ex-header:#f6f8fa;
  --link:#0969da;
}
html.dark{
  --bg:#0d1117;--bg2:#161b22;--bg3:#21262d;--fg:#e6edf3;--fg2:#8b949e;--fg3:#6e7681;
  --border:#30363d;--code-bg:#161b22;--code-fg:#e6edf3;
  --ex-bg:#0d1117;--ex-fg:#e6edf3;--ex-border:#30363d;--ex-header:#161b22;
  --link:#58a6ff;
}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Noto Sans,Helvetica,Arial,sans-serif;background:var(--bg);color:var(--fg);line-height:1.6;font-size:15px}
code,pre{font-family:ui-monospace,SFMono-Regular,'SF Mono',Menlo,Consolas,monospace;font-size:13px}
a{color:var(--link);text-decoration:none}a:hover{text-decoration:underline}

/* Auto-hide scrollbar */
*{scrollbar-width:thin;scrollbar-color:transparent transparent}
*:hover{scrollbar-color:var(--border) transparent}
::-webkit-scrollbar{width:6px;height:6px}
::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:transparent;border-radius:3px}
*:hover::-webkit-scrollbar-thumb{background:var(--border)}

/* Sidebar */
.sidebar{position:fixed;top:0;left:0;width:250px;height:100vh;background:var(--bg);border-right:1px solid var(--border);overflow-y:auto;z-index:100;padding:0 0 40px}
.sidebar-header{padding:20px 16px 16px;display:flex;align-items:center;justify-content:space-between}
.sidebar-logo{font-weight:700;font-size:15px;color:var(--fg);text-decoration:none}
.sidebar-actions{display:flex;align-items:center;gap:8px}
.sidebar-close{display:none;background:none;border:none;font-size:20px;color:var(--fg3);cursor:pointer}
.theme-toggle{background:none;border:none;color:var(--fg3);cursor:pointer;padding:4px;border-radius:6px;display:flex;align-items:center}
.theme-toggle:hover{color:var(--fg);background:var(--bg3)}
html:not(.dark) .icon-moon{display:none}
html.dark .icon-sun{display:none}
.sidebar-nav{list-style:none;padding:0 8px}
.sidebar-nav>li{margin:0}
.sidebar-group{margin-top:4px}
.sidebar-group-label-static,.sidebar-group-label{display:flex;align-items:center;justify-content:space-between;padding:10px 8px 4px;font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.06em;color:var(--fg3);list-style:none;user-select:none}
.sidebar-group-label{cursor:pointer}
.sidebar-group-label::-webkit-details-marker{display:none}
.sidebar-details{border:none}
.sidebar-details .chevron{transition:transform .15s;flex-shrink:0;opacity:.5}
.sidebar-details:not([open]) .chevron{transform:rotate(-90deg)}
.sidebar-link{display:flex;align-items:center;gap:7px;padding:5px 8px;color:var(--fg2);font-size:13px;border-radius:6px;text-decoration:none;line-height:1.4}
.sidebar-link:hover{background:var(--bg3);color:var(--fg);text-decoration:none}
.sidebar-link.active{background:var(--bg3);color:var(--fg);font-weight:500}
.sidebar-group ul{list-style:none;padding:0}
.sb-badge{font-size:10px;font-weight:600;flex-shrink:0;min-width:40px;font-family:ui-monospace,SFMono-Regular,monospace}

/* Content */
.content{margin-left:250px}
.menu-btn{display:none;position:sticky;top:0;z-index:50;width:100%;padding:10px 16px;background:var(--bg);border:none;border-bottom:1px solid var(--border);color:var(--fg);font-size:14px;text-align:left;cursor:pointer}

/* Top actions bar */
.top-actions{position:sticky;top:0;z-index:40;display:flex;justify-content:flex-end;padding:8px 24px;background:var(--bg);border-bottom:1px solid var(--border)}
.md-dropdown{position:relative}
.md-btn{display:flex;align-items:center;gap:6px;padding:6px 12px;border:1px solid var(--border);border-radius:8px;background:var(--bg);color:var(--fg);font-size:13px;cursor:pointer;font-family:inherit}
.md-btn:hover{background:var(--bg3)}
.md-menu{display:none;position:absolute;right:0;top:calc(100% + 4px);background:var(--bg);border:1px solid var(--border);border-radius:10px;box-shadow:0 4px 12px rgba(0,0,0,.1);min-width:200px;padding:4px;z-index:50}
.md-dropdown.open .md-menu{display:block}
.md-menu button{display:flex;align-items:center;gap:8px;width:100%;padding:8px 12px;border:none;background:none;color:var(--fg);font-size:13px;cursor:pointer;border-radius:6px;font-family:inherit;text-align:left}
.md-menu button:hover{background:var(--bg3)}

/* Prose sections */
.prose-section{max-width:820px;padding:40px 48px;border-bottom:1px solid var(--border)}
.prose-section h1{font-size:32px;font-weight:700;margin-bottom:28px;letter-spacing:-.5px}
.prose-section h2{font-size:22px;font-weight:600;margin:36px 0 12px;letter-spacing:-.3px}
.prose-section h2:first-child{margin-top:0}
.prose-section p{color:var(--fg2);margin-bottom:16px;font-size:15px;line-height:1.7}
.prose-section strong{color:var(--fg);font-weight:600}
.prose-section code{background:var(--code-bg);color:var(--code-fg);padding:2px 6px;border-radius:4px;font-size:13px}
.code-block{position:relative;background:var(--code-bg);border:1px solid var(--border);border-radius:8px;margin:16px 0;overflow:hidden}
.code-block pre{margin:0;padding:16px;overflow-x:auto;font-size:13px;line-height:1.7}
.code-block code{background:none;padding:0;color:var(--code-fg)}
.code-block .copy-btn{position:absolute;top:8px;right:8px;background:var(--bg);border:1px solid var(--border);border-radius:6px;color:var(--fg3);cursor:pointer;padding:4px 6px;opacity:.4;display:flex;align-items:center}
.code-block:hover .copy-btn{opacity:.8}
.code-block .copy-btn:hover{opacity:1}

/* Error table */
.error-table{width:100%;border-collapse:collapse;margin:16px 0;font-size:14px}
.error-table th{text-align:left;padding:10px 12px;font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.05em;color:var(--fg3);border-bottom:2px solid var(--border)}
.error-table td{padding:10px 12px;border-bottom:1px solid var(--border);color:var(--fg2)}
.error-table code{background:var(--code-bg);padding:2px 6px;border-radius:4px;font-size:12px}

/* Tag headings */
.tag-heading{padding:36px 48px 12px;font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.08em;color:var(--fg3);border-bottom:1px solid var(--border)}

/* Endpoint sections */
.endpoint-section{display:grid;grid-template-columns:1fr 1fr;gap:0;border-bottom:1px solid var(--border)}
.desc-col{padding:36px 40px 36px 48px}
.example-col{padding:36px 32px;background:var(--ex-bg);border-left:1px solid var(--ex-border)}

.endpoint-title{font-size:22px;font-weight:600;margin-bottom:10px;letter-spacing:-.3px}
.endpoint-desc{color:var(--fg2);margin-bottom:20px;font-size:14px;line-height:1.7}
.endpoint-desc code{background:var(--code-bg);padding:2px 5px;border-radius:3px;font-size:12px}
.desc-col h4{font-size:12px;font-weight:600;letter-spacing:.03em;color:var(--fg3);margin:28px 0 8px}
.desc-col p{color:var(--fg2);margin-bottom:12px;font-size:14px}
.desc-col code{background:var(--code-bg);color:var(--code-fg);padding:2px 6px;border-radius:4px;font-size:12px}

.method-path{display:flex;align-items:center;gap:10px;margin-bottom:16px}
.method-badge{font-size:11px;font-weight:700;padding:2px 8px;border-radius:6px;border:1.5px solid;display:inline-flex;align-items:center;gap:4px;letter-spacing:.01em}
.method-badge svg{opacity:.7}
.endpoint-path{font-size:14px;color:var(--fg2);background:none;padding:0}

/* Parameters */
.param-list{margin-bottom:8px}
.param-item{padding:12px 0;border-top:1px solid var(--border)}
.param-item:first-child{border-top:none}
.param-header{display:flex;align-items:baseline;gap:4px;flex-wrap:wrap}
.param-name{font-weight:600;font-size:14px;color:var(--fg)}
.param-type{font-size:13px;color:var(--fg3)}
.param-type code{font-size:12px;padding:1px 4px}
.param-optional{font-size:12px;color:var(--fg3);font-style:italic;margin-right:2px}
.param-desc{font-size:14px;color:var(--fg2);margin:4px 0 0;line-height:1.6}
.param-desc code{background:var(--code-bg);padding:1px 4px;border-radius:3px;font-size:12px}
.response-line{font-size:13px;color:var(--fg2);margin:6px 0}
.response-code{font-size:10px;font-weight:700;padding:2px 7px;border-radius:4px}
.code-2{background:#dafbe1;color:#116329}
.code-3{background:#ddf4ff;color:#0550ae}
.code-4{background:#ffebe9;color:#82071e}
.code-5{background:#fff8c5;color:#6a5505}
html.dark .code-2{background:#0f291c;color:#3fb950}
html.dark .code-3{background:#0c2d6b;color:#58a6ff}
html.dark .code-4{background:#2d0d14;color:#f85149}
html.dark .code-5{background:#2d1d00;color:#d29922}

/* Example blocks */
.example-block{margin-bottom:16px;border:1px solid var(--ex-border);border-radius:10px;overflow:hidden}
.example-header{display:flex;align-items:center;justify-content:space-between;padding:10px 16px;background:var(--ex-header);border-bottom:1px solid var(--ex-border)}
.example-title{font-size:13px;font-weight:500;color:var(--fg)}
.example-block pre{margin:0;padding:16px;overflow-x:auto;font-size:13px;line-height:1.7;background:transparent;color:var(--ex-fg);counter-reset:line}
.example-block code{background:none;padding:0;font-size:inherit;color:inherit}
.example-header .copy-btn{background:none;border:1px solid var(--ex-border);border-radius:6px;color:var(--fg3);opacity:.4;cursor:pointer;padding:3px 6px;display:flex;align-items:center}
.example-block:hover .copy-btn{opacity:.8}
.example-header .copy-btn:hover{opacity:1}

/* Line numbers */
.ln{display:inline-block;width:28px;text-align:right;margin-right:16px;color:var(--fg3);opacity:.5;user-select:none;font-size:12px}

/* Syntax highlighting */
.sh-cmd{color:#cf222e;font-weight:600}
.sh-flag{color:#8250df}
.sh-str{color:#0a3069}
.sh-var{color:#953800}
.js-key{color:#0550ae}
.js-str{color:#0a3069}
.js-num{color:#0550ae}
.js-bool{color:#cf222e}
html.dark .sh-cmd{color:#ff7b72;font-weight:600}
html.dark .sh-flag{color:#d2a8ff}
html.dark .sh-str{color:#a5d6ff}
html.dark .sh-var{color:#ffa657}
html.dark .js-key{color:#79c0ff}
html.dark .js-str{color:#a5d6ff}
html.dark .js-num{color:#79c0ff}
html.dark .js-bool{color:#ff7b72}

/* Responsive */
@media(max-width:1100px){
  .sidebar{transform:translateX(-100%);transition:transform .2s}
  .sidebar.open{transform:translateX(0);box-shadow:4px 0 24px rgba(0,0,0,.12)}
  .sidebar-close{display:block}
  .content{margin-left:0}
  .menu-btn{display:block}
  .endpoint-section{grid-template-columns:1fr}
  .example-col{border-left:none;border-top:1px solid var(--ex-border)}
  .desc-col,.example-col{padding:24px 20px}
  .prose-section{padding:24px 20px}
  .tag-heading{padding:24px 20px 12px}
}
@media print{.sidebar,.menu-btn,.top-actions{display:none}.content{margin-left:0}.endpoint-section{grid-template-columns:1fr}}
</style>
</head><body>
${sidebar}
${prose}
${endpoints}
</main>
<script>
// Theme
(function(){
  var h=document.documentElement,s=localStorage.getItem('api-theme');
  if(s==='dark')h.classList.add('dark');
  else if(!s&&window.matchMedia('(prefers-color-scheme:dark)').matches)h.classList.add('dark');
  document.getElementById('theme-toggle').onclick=function(){h.classList.toggle('dark');localStorage.setItem('api-theme',h.classList.contains('dark')?'dark':'light')};
})();
// Copy
function copyCode(btn){var p=btn.closest('.code-block,.example-block').querySelector('pre');navigator.clipboard.writeText(p.textContent.replace(/^\\s*\\d+\\s*/gm,'').trim()).then(function(){var o=btn.innerHTML;btn.textContent='Copied!';setTimeout(function(){btn.innerHTML=o},1200)})}
// Markdown
var MD=\`${markdownData}\`;
function copyMarkdown(){navigator.clipboard.writeText(MD).then(function(){var b=document.querySelector('.md-btn');b.textContent='Copied!';setTimeout(function(){b.innerHTML='${COPY_ICON} Copy Markdown <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 9l6 6 6-6"/></svg>'},1200)});document.getElementById('md-dropdown').classList.remove('open')}
function viewMarkdown(){var w=window.open('','_blank');w.document.write('<pre style="white-space:pre-wrap;font-family:monospace;padding:20px;max-width:900px;margin:0 auto">'+MD.replace(/</g,'&lt;')+'</pre>');w.document.close();document.getElementById('md-dropdown').classList.remove('open')}
document.addEventListener('click',function(e){if(!e.target.closest('.md-dropdown'))document.getElementById('md-dropdown').classList.remove('open')});
// Scroll highlight
(function(){
  var links=document.querySelectorAll('.sidebar-link[href^="#"]'),secs=[];
  links.forEach(function(a){var t=document.getElementById(a.getAttribute('href').slice(1));if(t)secs.push({el:t,link:a})});
  function u(){var y=window.scrollY+120,c=null;for(var i=secs.length-1;i>=0;i--){if(secs[i].el.offsetTop<=y){c=secs[i];break}}links.forEach(function(l){l.classList.remove('active')});if(c)c.link.classList.add('active')}
  window.addEventListener('scroll',u,{passive:true});u();
})();
</script>
</body></html>`;
}
