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
// AI page removed — covered by home page
import { cliPage } from "./pages/cli";
import { browsePage } from "./pages/browse";
import { privacyPage } from "./pages/privacy";
import { CloudflareEngine } from "./storage/cloudflare";
import type { Env, Variables } from "./types";

const app = new OpenAPIHono<{ Bindings: Env; Variables: Variables }>();

app.use("*", cors());

// ── Inject storage engine into request context ──────────────────────
app.use("*", async (c, next) => {
  const engine = new CloudflareEngine({
    db: c.env.DB,
    bucket: c.env.BUCKET,
    r2Endpoint: c.env.R2_ENDPOINT,
    r2AccessKeyId: c.env.R2_ACCESS_KEY_ID,
    r2SecretAccessKey: c.env.R2_SECRET_ACCESS_KEY,
    r2BucketName: c.env.R2_BUCKET_NAME,
  });
  c.set("engine", engine);
  await next();
});

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
app.get("/ai", (c) => c.redirect("/", 302));
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

const COPY_ICON = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="0"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`;

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

  // Section tabs
  let sectionTabs = `<a href="#introduction" class="sec-tab active">Overview</a>`;
  for (const tag of tagOrder) {
    const label = TAG_LABELS[tag] || tag.charAt(0).toUpperCase() + tag.slice(1);
    sectionTabs += `<a href="#tag-${esc(tag)}" class="sec-tab">${esc(label)}</a>`;
  }

  // Prose sections
  const prose = `
<div class="page-header">
  <div class="page-header-inner">
    <div class="sec-label">API REFERENCE</div>
    <h1 class="page-title">Storage API</h1>
    <p class="page-sub">Base URL: <code>${esc(origin)}</code></p>
  </div>
</div>

<div class="sec-tabs" id="sec-tabs">${sectionTabs}</div>

<main class="api-main">

<article class="prose" id="introduction">
  <h2>Introduction</h2>
  <p>REST APIs usable via HTTP in any environment. The <a href="/cli">CLI</a> is also available.</p>
  <div class="code-block"><pre><code>${highlightCurl(`curl ${origin}/files \\\n  -H "Authorization: Bearer $STORAGE_API_KEY"`)}</code></pre><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
</article>

<article class="prose" id="authentication">
  <h2>Authentication</h2>
  <p>Create API keys via <a href="#POST-auth-keys"><code>POST /auth/keys</code></a> or the <a href="/browse">dashboard</a>. Provide them as Bearer tokens.</p>
  <div class="code-block"><pre><code>Authorization: Bearer STORAGE_API_KEY</code></pre><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
  <p>Keys can be scoped to a path prefix (e.g. <code>docs/</code>) and set to expire. <strong>Keep keys secret.</strong> Never expose in client-side code.</p>
</article>

<article class="prose" id="content-types">
  <h2>Content types</h2>
  <p>Send request bodies as JSON with <code>Content-Type: application/json</code>.</p>
  <p><code>GET /files/{path}</code> returns a <code>302</code> redirect to a presigned URL by default. Set <code>Accept: application/json</code> to get metadata instead.</p>
</article>

<article class="prose" id="errors">
  <h2>Errors</h2>
  <p>Errors return JSON with a consistent shape:</p>
  <div class="code-block"><pre><code>${highlightJson(`{\n  "error": "not_found",\n  "message": "File not found"\n}`)}</code></pre><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
  <table class="err-tbl">
    <thead><tr><th>Code</th><th>Error</th><th>Meaning</th></tr></thead>
    <tbody>
      <tr><td><span class="rc rc-4">400</span></td><td><code>bad_request</code></td><td>Invalid parameters or body</td></tr>
      <tr><td><span class="rc rc-4">401</span></td><td><code>unauthorized</code></td><td>Missing or invalid auth</td></tr>
      <tr><td><span class="rc rc-4">403</span></td><td><code>forbidden</code></td><td>Path not allowed</td></tr>
      <tr><td><span class="rc rc-4">404</span></td><td><code>not_found</code></td><td>Resource not found</td></tr>
      <tr><td><span class="rc rc-4">409</span></td><td><code>conflict</code></td><td>Already exists</td></tr>
      <tr><td><span class="rc rc-4">429</span></td><td><code>rate_limited</code></td><td>Too many requests</td></tr>
      <tr><td><span class="rc rc-5">500</span></td><td><code>internal</code></td><td>Server error</td></tr>
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
<section id="${id}" class="endpoint">
<div class="ep-desc">
  <h3 class="ep-title">${esc(summary)}</h3>
  <div class="ep-method">
    <span class="ep-badge ep-${method.toLowerCase()}">${method}</span>
    <code class="ep-path">${esc(path)}</code>
  </div>
  ${desc ? `<p class="ep-text">${desc}</p>` : ""}
  ${renderParams(pathP, "Path Parameters")}
  ${renderParams(queryP, "Query Parameters")}
  ${renderBodyParams(bodySchema)}
  ${renderReturns(op.responses)}
</div>
<div class="ep-ex">
  <div class="ex-block">
    <div class="ex-hdr"><span class="ex-title">${esc(summary)}</span><button class="copy-btn" onclick="copyCode(this)" title="Copy">${COPY_ICON}</button></div>
    <pre><code>${withLineNumbers(highlightCurl(curlRaw))}</code></pre>
  </div>
  ${responseRaw ? `<div class="ex-block"><pre><code>${withLineNumbers(highlightJson(responseRaw))}</code></pre></div>` : ""}
</div>
</section>`;
    }
  }

  // Generate markdown data for copy/view
  return `<!doctype html><html lang="en" class="dark"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>API Reference — Storage</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/base.css">
<style>
/* ── API-reference tokens (extends base.css) ─────────────── */
:root{--sf3:#E4E4E7;--purple:#8B5CF6}
html.dark{--sf3:#27272A;--purple:#A78BFA}
body{font-size:14px;line-height:1.6}
code,pre{font-family:'JetBrains Mono',monospace}
a{color:var(--text-2);text-decoration:underline;text-underline-offset:2px;transition:color .12s}
a:hover{color:var(--text)}

/* ── Page header ─────────────────────────────────────────── */
.page-header{position:relative;z-index:1;border-bottom:1px solid var(--border);padding:48px 0;background:var(--bg)}
.page-header-inner{max-width:1104px;margin:0 auto;padding:0 48px}
.sec-label{font-family:'JetBrains Mono',monospace;font-size:10px;letter-spacing:2px;color:var(--text-3);margin-bottom:12px;font-weight:500;text-transform:uppercase}
.page-title{font-size:32px;font-weight:800;letter-spacing:-.8px;margin-bottom:8px}
.page-sub{font-size:13px;color:var(--text-2);margin-bottom:16px}
.page-sub code{font-size:12px;background:var(--surface-alt);padding:3px 8px;border:1px solid var(--border)}

/* ── Section tabs ────────────────────────────────────────── */
.sec-tabs{position:sticky;top:48px;z-index:90;
  max-width:1104px;margin:0 auto;padding:0 48px;
  display:flex;gap:0;overflow-x:auto;
  background:color-mix(in srgb,var(--bg) 90%,transparent);
  backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);
  border-bottom:1px solid var(--border)}
.sec-tab{font-family:'JetBrains Mono',monospace;font-size:11px;font-weight:500;
  padding:14px 16px;color:var(--text-3);text-decoration:none;white-space:nowrap;
  border-bottom:2px solid transparent;transition:all .12s}
.sec-tab:hover{color:var(--text-2);text-decoration:none}
.sec-tab.active{color:var(--text);border-bottom-color:var(--text)}

/* ── Main content ────────────────────────────────────────── */
.api-main{position:relative;z-index:1;max-width:1104px;margin:0 auto;padding:0 48px;background:var(--bg)}

/* ── Prose ────────────────────────────────────────────────── */
.prose{padding:32px 0;border-bottom:1px solid var(--border)}
.prose h2{font-size:20px;font-weight:700;letter-spacing:-.4px;margin-bottom:12px}
.prose p{color:var(--text-2);margin-bottom:12px;font-size:13px;line-height:1.7}
.prose p:last-child{margin-bottom:0}
.prose strong{color:var(--text);font-weight:600}
.prose code{font-size:11.5px;background:var(--surface-alt);padding:2px 6px;border:1px solid var(--border)}
.code-block{position:relative;background:var(--surface);border:1px solid var(--border);margin:12px 0;overflow:hidden}
.code-block pre{margin:0;padding:14px 16px;overflow-x:auto;font-size:11.5px;line-height:1.7}
.code-block code{background:none;padding:0;border:none;color:var(--text-2)}
.copy-btn{position:absolute;top:6px;right:6px;background:var(--surface-alt);border:1px solid var(--border);
  color:var(--text-3);cursor:pointer;padding:3px 5px;opacity:0;transition:opacity .12s;display:flex;align-items:center}
.code-block:hover .copy-btn,.ex-block:hover .copy-btn{opacity:.7}
.copy-btn:hover{opacity:1 !important;color:var(--text)}

/* ── Error table ──────────────────────────────────────────── */
.err-tbl{width:100%;border-collapse:collapse;margin:12px 0;font-size:12px;border:1px solid var(--border)}
.err-tbl th{text-align:left;padding:8px 12px;font-size:10px;font-weight:600;text-transform:uppercase;
  letter-spacing:.05em;color:var(--text-3);border-bottom:1px solid var(--border);background:var(--surface-alt)}
.err-tbl td{padding:8px 12px;border-bottom:1px solid var(--border);color:var(--text-2)}
.err-tbl code{font-size:11px;background:var(--surface-alt);padding:1px 5px;border:1px solid var(--border)}
.rc{font-family:'JetBrains Mono',monospace;font-size:10px;font-weight:700;padding:2px 6px;border:1px solid var(--border)}
.rc-2{color:var(--green);background:color-mix(in srgb,var(--green) 8%,var(--surface))}
.rc-3{color:var(--blue);background:color-mix(in srgb,var(--blue) 8%,var(--surface))}
.rc-4{color:var(--red);background:color-mix(in srgb,var(--red) 8%,var(--surface))}
.rc-5{color:var(--amber);background:color-mix(in srgb,var(--amber) 8%,var(--surface))}

/* ── Tag headings ─────────────────────────────────────────── */
.tag-heading{padding:48px 0 12px;font-family:'JetBrains Mono',monospace;font-size:10px;font-weight:600;
  text-transform:uppercase;letter-spacing:2px;color:var(--text-3);border-bottom:1px solid var(--border)}

/* ── Endpoint sections ────────────────────────────────────── */
.endpoint{display:grid;grid-template-columns:1fr 1fr;gap:0;border-bottom:1px solid var(--border)}
.ep-desc{padding:28px 32px 28px 0}
.ep-ex{padding:28px 0 28px 32px;border-left:1px solid var(--border)}

.ep-title{font-size:18px;font-weight:700;letter-spacing:-.3px;margin-bottom:8px}
.ep-method{display:flex;align-items:center;gap:8px;margin-bottom:14px}
.ep-badge{font-family:'JetBrains Mono',monospace;font-size:10px;font-weight:700;padding:2px 8px;
  border:1px solid var(--border);letter-spacing:.3px}
.ep-get{color:var(--green);border-color:color-mix(in srgb,var(--green) 30%,var(--border))}
.ep-post{color:var(--blue);border-color:color-mix(in srgb,var(--blue) 30%,var(--border))}
.ep-put{color:var(--amber);border-color:color-mix(in srgb,var(--amber) 30%,var(--border))}
.ep-patch{color:var(--amber);border-color:color-mix(in srgb,var(--amber) 30%,var(--border))}
.ep-delete{color:var(--red);border-color:color-mix(in srgb,var(--red) 30%,var(--border))}
.ep-head{color:var(--purple);border-color:color-mix(in srgb,var(--purple) 30%,var(--border))}
.ep-path{font-size:12px;color:var(--text-2);background:none;padding:0;border:none}
.ep-text{color:var(--text-2);margin-bottom:16px;font-size:13px;line-height:1.7}
.ep-text code{font-size:11px;background:var(--surface-alt);padding:1px 4px;border:1px solid var(--border)}
.ep-desc h4{font-family:'JetBrains Mono',monospace;font-size:10px;font-weight:600;letter-spacing:1px;
  text-transform:uppercase;color:var(--text-3);margin:24px 0 6px}
.ep-desc p{color:var(--text-2);margin-bottom:8px;font-size:13px}
.ep-desc code{font-size:11px;background:var(--surface-alt);padding:1px 5px;border:1px solid var(--border)}

/* ── Parameters ───────────────────────────────────────────── */
.param-list{margin-bottom:4px}
.param-item{padding:10px 0;border-top:1px solid var(--border)}
.param-item:first-child{border-top:none}
.param-header{display:flex;align-items:baseline;gap:4px;flex-wrap:wrap}
.param-name{font-family:'JetBrains Mono',monospace;font-weight:600;font-size:12px;color:var(--text)}
.param-type{font-size:12px;color:var(--text-3)}
.param-type code{font-size:11px;padding:1px 3px;border:1px solid var(--border);background:var(--surface-alt)}
.param-optional{font-size:11px;color:var(--text-3);font-style:italic;margin-right:2px}
.param-desc{font-size:12px;color:var(--text-2);margin:3px 0 0;line-height:1.5}
.param-desc code{font-size:11px;background:var(--surface-alt);padding:1px 3px;border:1px solid var(--border)}
.response-line{font-size:12px;color:var(--text-2);margin:4px 0}
.response-code{font-family:'JetBrains Mono',monospace;font-size:10px;font-weight:700;padding:2px 6px;border:1px solid var(--border)}
.code-2{color:var(--green);background:color-mix(in srgb,var(--green) 8%,var(--surface))}
.code-3{color:var(--blue);background:color-mix(in srgb,var(--blue) 8%,var(--surface))}
.code-4{color:var(--red);background:color-mix(in srgb,var(--red) 8%,var(--surface))}
.code-5{color:var(--amber);background:color-mix(in srgb,var(--amber) 8%,var(--surface))}

/* ── Example blocks ───────────────────────────────────────── */
.ex-block{position:relative;margin-bottom:12px;border:1px solid var(--border);background:var(--surface);overflow:hidden}
.ex-hdr{display:flex;align-items:center;justify-content:space-between;padding:8px 14px;
  background:var(--surface-alt);border-bottom:1px solid var(--border)}
.ex-title{font-family:'JetBrains Mono',monospace;font-size:11px;font-weight:500;color:var(--text-2)}
.ex-hdr .copy-btn{position:static;opacity:0}
.ex-block:hover .ex-hdr .copy-btn{opacity:.7}
.ex-block pre{margin:0;padding:14px 16px;overflow-x:auto;font-size:11.5px;line-height:1.7;color:var(--text-2)}
.ex-block code{background:none;padding:0;border:none;font-size:inherit;color:inherit}

/* ── Line numbers ─────────────────────────────────────────── */
.ln{display:inline-block;width:24px;text-align:right;margin-right:14px;color:var(--text-3);opacity:.4;user-select:none;font-size:11px}

/* ── Syntax highlighting ──────────────────────────────────── */
.sh-cmd{color:var(--text);font-weight:600}
.sh-flag{color:var(--purple)}
.sh-str{color:var(--text-2);opacity:.85}
.sh-var{color:var(--amber)}
.js-key{color:var(--text)}
.js-str{color:var(--text-2);opacity:.85}
.js-num{color:var(--blue)}
.js-bool{color:var(--red)}

/* ── Responsive ───────────────────────────────────────────── */
@media(max-width:1024px){
  .endpoint{grid-template-columns:1fr}
  .ep-ex{border-left:none;border-top:1px solid var(--border);padding:20px 0}
  .ep-desc{padding:20px 0}
}
@media(max-width:640px){
  .page-header{padding:32px 0 24px}
  .page-header-inner{padding:0 20px}
  .page-title{font-size:24px}
  .sec-tabs{padding:0 20px;top:48px}
  .sec-tab{padding:10px 14px;font-size:10px}
  .api-main{padding:0 20px}
  .prose{padding:24px 0}
  .tag-heading{padding:32px 0 10px}
  .ep-desc,.ep-ex{padding:16px 0}
}
@media print{nav,.sec-tabs,.grid-bg,.copy-btn{display:none}.page-header{padding:20px 0}.endpoint{grid-template-columns:1fr}}
</style>
</head><body>

<div class="grid-bg"></div>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> Storage</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/developers">developers</a>
      <a href="/api" class="active">api</a>
      <a href="/cli">cli</a>
      <a href="/pricing">pricing</a>
    </div>
    <div class="nav-right">
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

${prose}
${endpoints}
</main>

<script>
function toggleTheme(){var d=document.documentElement.classList.toggle('dark');localStorage.setItem('theme',d?'dark':'light')}
(function(){var s=localStorage.getItem('theme');if(s==='light')document.documentElement.classList.remove('dark');else if(!s&&!window.matchMedia('(prefers-color-scheme:dark)').matches)document.documentElement.classList.remove('dark')})();
function copyCode(btn){var p=btn.closest('.code-block,.ex-block').querySelector('pre');navigator.clipboard.writeText(p.textContent.replace(/^\\s*\\d+\\s*/gm,'').trim()).then(function(){var o=btn.innerHTML;btn.textContent='Copied!';setTimeout(function(){btn.innerHTML=o},1200)})}
(function(){
  var tabs=document.querySelectorAll('.sec-tab'),anchors=[];
  tabs.forEach(function(a){var id=a.getAttribute('href');if(id){var el=document.getElementById(id.slice(1));if(el)anchors.push({el:el,tab:a})}});
  function u(){var y=window.scrollY+140,c=null;for(var i=anchors.length-1;i>=0;i--){if(anchors[i].el.offsetTop<=y){c=anchors[i];break}}tabs.forEach(function(t){t.classList.remove('active')});if(c)c.tab.classList.add('active')}
  window.addEventListener('scroll',u,{passive:true});u();
})();
</script>
</body></html>`;
}
