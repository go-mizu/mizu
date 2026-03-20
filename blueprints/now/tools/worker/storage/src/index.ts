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

const TAG_LABELS: Record<string, string> = {
  files: "Files",
  uploads: "Uploads",
  sharing: "Sharing",
  auth: "Authentication",
  keys: "API Keys",
};

const METHOD_COLORS: Record<string, string> = {
  GET: "#10b981",
  POST: "#3b82f6",
  PUT: "#f59e0b",
  PATCH: "#f59e0b",
  DELETE: "#ef4444",
  HEAD: "#6b7280",
};

function renderApiReference(spec: any, origin: string): string {
  const resolved = resolveRefs(spec, spec);

  // Group by tag
  const byTag = new Map<string, { method: string; path: string; op: any }[]>();
  const tagOrder: string[] = [];
  for (const [path, methods] of Object.entries(resolved.paths || {})) {
    for (const [method, rawOp] of Object.entries(methods as any)) {
      if (typeof rawOp !== "object" || !rawOp) continue;
      const op = rawOp as any;
      const tag = op.tags?.[0] || "other";
      if (!byTag.has(tag)) { byTag.set(tag, []); tagOrder.push(tag); }
      byTag.get(tag)!.push({ method: method.toUpperCase(), path, op });
    }
  }

  // ── Sidebar ──────────────────────────────────────────────────────
  let sidebar = `<nav class="sidebar" id="sidebar">
<div class="sidebar-header">
  <a href="/" class="sidebar-logo">Storage API</a>
  <button class="sidebar-close" onclick="document.getElementById('sidebar').classList.remove('open')" aria-label="Close">&times;</button>
</div>
<ul class="sidebar-nav">
  <li><a href="#overview" class="sidebar-link">Overview</a></li>
  <li><a href="#authentication" class="sidebar-link">Authentication</a></li>
  <li><a href="#errors" class="sidebar-link">Errors</a></li>`;

  for (const tag of tagOrder) {
    const routes = byTag.get(tag)!;
    const label = TAG_LABELS[tag] || tag.charAt(0).toUpperCase() + tag.slice(1);
    sidebar += `\n  <li class="sidebar-group">
    <span class="sidebar-group-label">${esc(label)}</span>
    <ul>`;
    for (const { method, path, op } of routes) {
      const id = slug(method, path);
      sidebar += `\n      <li><a href="#${id}" class="sidebar-link"><span class="method-dot" style="background:${METHOD_COLORS[method] || "#888"}"></span>${esc(op.summary || `${method} ${path}`)}</a></li>`;
    }
    sidebar += `\n    </ul>
  </li>`;
  }
  sidebar += `\n</ul></nav>`;

  // ── Overview section ─────────────────────────────────────────────
  let content = `<main class="content">
<button class="menu-btn" onclick="document.getElementById('sidebar').classList.toggle('open')" aria-label="Menu">&#9776; API Reference</button>

<section id="overview" class="endpoint-section">
<div class="desc-col">
  <h1>Storage API Reference</h1>
  <p>File storage on the edge. Upload via presigned URLs, download via redirect.</p>
  <h3>Base URL</h3>
  <p><code>${esc(origin)}</code></p>
</div>
<div class="example-col">
  <div class="example-block">
    <div class="example-label">Base URL</div>
    <pre><code>${esc(origin)}</code></pre>
  </div>
</div>
</section>

<section id="authentication" class="endpoint-section">
<div class="desc-col">
  <h2>Authentication</h2>
  <p>Requests require a Bearer token passed in the <code>Authorization</code> header. Obtain a token through the authentication flow or by creating an API key.</p>
  <div class="param-list">
    <div class="param-item">
      <div class="param-header"><code class="param-name">Authorization</code> <span class="param-type">header</span></div>
      <p class="param-desc">Bearer token. Format: <code>Bearer &lt;token&gt;</code></p>
    </div>
  </div>
  <p>API keys can be scoped to a path prefix and set to expire. Create them via <a href="#POST-auth-keys"><code>POST /auth/keys</code></a>.</p>
</div>
<div class="example-col">
  <div class="example-block">
    <div class="example-label">Authenticated request</div>
    <pre><code>curl ${esc(origin)}/files \\
  -H "Authorization: Bearer $STORAGE_API_KEY"</code></pre>
  </div>
</div>
</section>

<section id="errors" class="endpoint-section">
<div class="desc-col">
  <h2>Errors</h2>
  <p>The API returns errors as JSON with a consistent shape. HTTP status codes follow standard conventions.</p>
  <div class="param-list">
    <div class="param-item">
      <div class="param-header"><code class="param-name">error</code> <span class="param-type">string</span></div>
      <p class="param-desc">Machine-readable error code (e.g. <code>not_found</code>, <code>unauthorized</code>, <code>bad_request</code>).</p>
    </div>
    <div class="param-item">
      <div class="param-header"><code class="param-name">message</code> <span class="param-type">string</span></div>
      <p class="param-desc">Human-readable description of the error.</p>
    </div>
  </div>
</div>
<div class="example-col">
  <div class="example-block">
    <div class="example-label">Error response</div>
    <pre><code>{
  "error": "not_found",
  "message": "File not found"
}</code></pre>
  </div>
</div>
</section>`;

  // ── Endpoints ────────────────────────────────────────────────────
  for (const tag of tagOrder) {
    const routes = byTag.get(tag)!;
    const label = TAG_LABELS[tag] || tag.charAt(0).toUpperCase() + tag.slice(1);
    content += `\n<h2 class="tag-heading" id="tag-${esc(tag)}">${esc(label)}</h2>`;

    for (const { method, path, op } of routes) {
      const id = slug(method, path);
      const color = METHOD_COLORS[method] || "#888";
      const needsAuth = op.security?.length > 0;

      // Parameters (query + path)
      let paramsHtml = "";
      const queryParams = (op.parameters || []).filter((p: any) => p.in === "query");
      const pathParams = (op.parameters || []).filter((p: any) => p.in === "path");

      if (pathParams.length) {
        paramsHtml += `<h4>Path parameters</h4><div class="param-list">`;
        for (const p of pathParams) {
          const pType = p.schema?.type || "string";
          const req = p.required !== false ? "" : `<span class="param-optional">optional</span> `;
          paramsHtml += `<div class="param-item">
            <div class="param-header"><code class="param-name">${esc(p.name)}</code> ${req}<span class="param-type">${esc(pType)}</span></div>
            ${p.description ? `<p class="param-desc">${esc(p.description)}</p>` : ""}
          </div>`;
        }
        paramsHtml += `</div>`;
      }

      if (queryParams.length) {
        paramsHtml += `<h4>Query parameters</h4><div class="param-list">`;
        for (const p of queryParams) {
          const pSchema = p.schema || {};
          const pType = pSchema.type || "string";
          const req = p.required ? "" : `<span class="param-optional">optional</span> `;
          const desc = p.description || pSchema.description || "";
          const def = pSchema.default !== undefined ? ` Defaults to <code>${esc(String(pSchema.default))}</code>.` : "";
          paramsHtml += `<div class="param-item">
            <div class="param-header"><code class="param-name">${esc(p.name)}</code> ${req}<span class="param-type">${esc(pType)}</span></div>
            ${desc || def ? `<p class="param-desc">${esc(desc)}${def}</p>` : ""}
          </div>`;
        }
        paramsHtml += `</div>`;
      }

      // Request body
      const bodySchema = op.requestBody?.content?.["application/json"]?.schema;
      if (bodySchema && bodySchema.properties) {
        paramsHtml += `<h4>Request body</h4><div class="param-list">`;
        const required = new Set(bodySchema.required || []);
        for (const [key, rawProp] of Object.entries(bodySchema.properties)) {
          const prop = rawProp as any;
          let pType = prop.type || "string";
          if (prop.enum) pType = prop.enum.map((v: any) => `"${v}"`).join(" | ");
          const req = required.has(key) ? "" : `<span class="param-optional">optional</span> `;
          const desc = prop.description || "";
          const def = prop.default !== undefined ? ` Defaults to <code>${esc(String(prop.default))}</code>.` : "";
          paramsHtml += `<div class="param-item">
            <div class="param-header"><code class="param-name">${esc(key)}</code> ${req}<span class="param-type">${esc(pType)}</span></div>
            ${desc || def ? `<p class="param-desc">${esc(desc)}${def}</p>` : ""}
          </div>`;
        }
        paramsHtml += `</div>`;
      }

      // Returns
      let returnsHtml = "";
      for (const [code, rawResp] of Object.entries(op.responses || {})) {
        const resp = rawResp as any;
        const rSchema = resp.content?.["application/json"]?.schema;
        if (rSchema && rSchema.properties) {
          returnsHtml += `<h4>Returns <span class="response-code code-${code[0]}">${esc(code)}</span></h4>`;
          if (resp.description) returnsHtml += `<p class="param-desc">${esc(resp.description)}</p>`;
          returnsHtml += `<div class="param-list">`;
          for (const [key, rawProp] of Object.entries(rSchema.properties)) {
            const prop = rawProp as any;
            let pType = prop.type || "string";
            if (prop.type === "array") pType = `array of ${(prop.items as any)?.type || "object"}`;
            if (prop.nullable) pType += " | null";
            const desc = prop.description || "";
            returnsHtml += `<div class="param-item">
              <div class="param-header"><code class="param-name">${esc(key)}</code> <span class="param-type">${esc(pType)}</span></div>
              ${desc ? `<p class="param-desc">${esc(desc)}</p>` : ""}
            </div>`;
          }
          returnsHtml += `</div>`;
        } else if (resp.description && code !== "200" && code !== "201") {
          returnsHtml += `<p class="response-line"><span class="response-code code-${code[0]}">${esc(code)}</span> ${esc(resp.description)}</p>`;
        } else if (resp.description && !rSchema) {
          returnsHtml += `<h4>Returns <span class="response-code code-${code[0]}">${esc(code)}</span></h4><p class="param-desc">${esc(resp.description)}</p>`;
        }
      }

      // Curl example
      let curl = `curl`;
      if (method !== "GET" && method !== "HEAD") curl += ` -X ${method}`;
      curl += ` ${origin}${path.replace(/\{(\w+)\}/g, ":$1")}`;
      if (needsAuth) curl += ` \\\n  -H "Authorization: Bearer $STORAGE_API_KEY"`;
      if (bodySchema) {
        curl += ` \\\n  -H "Content-Type: application/json"`;
        const example = exampleFromSchema(bodySchema);
        if (example) curl += ` \\\n  -d '${JSON.stringify(example)}'`;
      }

      // Response example
      let responseJson = "";
      const successResp = (op.responses?.["200"] || op.responses?.["201"]) as any;
      if (successResp?.content?.["application/json"]?.schema) {
        const example = exampleFromSchema(successResp.content["application/json"].schema);
        if (example) responseJson = JSON.stringify(example, null, 2);
      }

      content += `
<section id="${id}" class="endpoint-section">
<div class="desc-col">
  <h3>${esc(op.summary || `${method} ${path}`)}</h3>
  <div class="method-path">
    <span class="method-badge" style="background:${color}">${method}</span>
    <code class="endpoint-path">${esc(path)}</code>
    ${needsAuth ? `<span class="auth-badge" title="Requires authentication">&#128274;</span>` : ""}
  </div>
  ${paramsHtml}
  ${returnsHtml}
</div>
<div class="example-col">
  <div class="example-block">
    <div class="example-header"><span class="example-label">Request</span><button class="copy-btn" onclick="copyCode(this)" title="Copy">&#128203;</button></div>
    <pre><code>${esc(curl)}</code></pre>
  </div>
  ${responseJson ? `<div class="example-block">
    <div class="example-header"><span class="example-label">Response</span></div>
    <pre><code>${esc(responseJson)}</code></pre>
  </div>` : ""}
</div>
</section>`;
    }
  }

  content += `\n</main>`;

  // ── Full HTML ────────────────────────────────────────────────────
  return `<!doctype html><html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>API Reference — Storage</title>
<style>
:root{
  --bg:#fff;--bg2:#f9fafb;--bg3:#f3f4f6;--fg:#1a1a2e;--fg2:#4b5563;--fg3:#6b7280;
  --border:#e5e7eb;--accent:#10b981;--code-bg:#f3f4f6;--code-fg:#1a1a2e;
  --example-bg:#1e1e2e;--example-fg:#cdd6f4;--example-border:#313244;
}
@media(prefers-color-scheme:dark){:root{
  --bg:#0f0f1a;--bg2:#1a1a2e;--bg3:#252540;--fg:#e5e7eb;--fg2:#9ca3af;--fg3:#6b7280;
  --border:#2d2d44;--accent:#34d399;--code-bg:#252540;--code-fg:#e5e7eb;
  --example-bg:#0f0f1a;--example-fg:#cdd6f4;--example-border:#2d2d44;
}}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:system-ui,-apple-system,'Segoe UI',sans-serif;background:var(--bg);color:var(--fg);line-height:1.6;font-size:15px}
code,pre{font-family:'SF Mono','Fira Code','Cascadia Code','Consolas',monospace;font-size:13px}
a{color:var(--accent);text-decoration:none}a:hover{text-decoration:underline}

/* Layout */
.sidebar{position:fixed;top:0;left:0;width:240px;height:100vh;background:var(--bg2);border-right:1px solid var(--border);overflow-y:auto;z-index:100;padding:0 0 40px}
.sidebar-header{padding:20px 16px 12px;display:flex;align-items:center;justify-content:space-between}
.sidebar-logo{font-weight:700;font-size:15px;color:var(--fg);text-decoration:none}
.sidebar-close{display:none;background:none;border:none;font-size:22px;color:var(--fg3);cursor:pointer}
.sidebar-nav{list-style:none;padding:0 8px}
.sidebar-nav li{margin:0}
.sidebar-link{display:flex;align-items:center;gap:6px;padding:5px 8px;color:var(--fg2);font-size:13px;border-radius:4px;text-decoration:none}
.sidebar-link:hover{background:var(--bg3);color:var(--fg);text-decoration:none}
.sidebar-group{margin-top:16px}
.sidebar-group-label{display:block;padding:4px 8px;font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.05em;color:var(--fg3)}
.sidebar-group ul{list-style:none;padding:0}
.method-dot{width:6px;height:6px;border-radius:50%;flex-shrink:0}

.content{margin-left:240px;max-width:1200px}
.menu-btn{display:none;position:sticky;top:0;z-index:50;width:100%;padding:10px 16px;background:var(--bg2);border:none;border-bottom:1px solid var(--border);color:var(--fg);font-size:14px;text-align:left;cursor:pointer}

/* Endpoint sections */
.endpoint-section{display:grid;grid-template-columns:1fr 1fr;gap:0;border-bottom:1px solid var(--border);min-height:0}
.desc-col{padding:32px 40px 32px 40px}
.example-col{padding:32px 40px 32px 24px;background:var(--example-bg);color:var(--example-fg);border-left:1px solid var(--example-border)}

.desc-col h1{font-size:24px;font-weight:700;margin-bottom:8px}
.desc-col h2{font-size:20px;font-weight:700;margin-bottom:8px}
.desc-col h3{font-size:17px;font-weight:600;margin-bottom:8px}
.desc-col h4{font-size:12px;font-weight:600;text-transform:uppercase;letter-spacing:.05em;color:var(--fg3);margin:20px 0 8px}
.desc-col p{color:var(--fg2);margin-bottom:12px}
.desc-col code{background:var(--code-bg);color:var(--code-fg);padding:2px 6px;border-radius:3px;font-size:13px}

.tag-heading{padding:32px 40px 12px;font-size:18px;font-weight:700;border-bottom:1px solid var(--border)}

.method-path{display:flex;align-items:center;gap:8px;margin-bottom:16px;flex-wrap:wrap}
.method-badge{color:#fff;font-size:11px;font-weight:700;padding:3px 8px;border-radius:4px;letter-spacing:.02em}
.endpoint-path{font-size:14px;color:var(--fg);background:none;padding:0}
.auth-badge{font-size:14px;cursor:help}

/* Parameters */
.param-list{margin-bottom:12px}
.param-item{padding:10px 0;border-top:1px solid var(--border)}
.param-item:first-child{border-top:none}
.param-header{display:flex;align-items:baseline;gap:6px;flex-wrap:wrap}
.param-name{background:var(--code-bg);color:var(--code-fg);padding:1px 5px;border-radius:3px;font-size:13px;font-weight:600}
.param-type{font-size:12px;color:var(--fg3)}
.param-optional{font-size:11px;color:var(--fg3);font-style:italic}
.param-desc{font-size:13px;color:var(--fg2);margin:4px 0 0}
.response-line{font-size:13px;color:var(--fg2);margin:6px 0}
.response-code{font-size:11px;font-weight:700;padding:2px 6px;border-radius:3px;background:var(--bg3);color:var(--fg)}
.code-2{background:#d1fae5;color:#065f46}
.code-3{background:#dbeafe;color:#1e40af}
.code-4{background:#fee2e2;color:#991b1b}
.code-5{background:#fef3c7;color:#92400e}

/* Examples */
.example-block{margin-bottom:20px;border:1px solid var(--example-border);border-radius:6px;overflow:hidden}
.example-header{display:flex;align-items:center;justify-content:space-between;padding:6px 12px;background:rgba(255,255,255,.05);border-bottom:1px solid var(--example-border)}
.example-label{font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.05em;color:var(--example-fg);opacity:.6}
.example-block pre{margin:0;padding:14px;overflow-x:auto;font-size:12.5px;line-height:1.5;background:transparent;color:var(--example-fg)}
.example-block code{background:none;padding:0;font-size:inherit;color:inherit}
.copy-btn{background:none;border:1px solid var(--example-border);border-radius:4px;color:var(--example-fg);opacity:.5;cursor:pointer;padding:2px 6px;font-size:13px}
.copy-btn:hover{opacity:1}

/* Responsive */
@media(max-width:1024px){
  .sidebar{transform:translateX(-100%);transition:transform .2s}
  .sidebar.open{transform:translateX(0);box-shadow:4px 0 20px rgba(0,0,0,.3)}
  .sidebar-close{display:block}
  .content{margin-left:0}
  .menu-btn{display:block}
  .endpoint-section{grid-template-columns:1fr}
  .example-col{border-left:none;border-top:1px solid var(--example-border)}
  .desc-col,.example-col{padding:24px 20px}
  .tag-heading{padding:24px 20px 12px}
}
@media print{.sidebar,.menu-btn{display:none}.content{margin-left:0}.endpoint-section{grid-template-columns:1fr}}
</style>
</head><body>
${sidebar}
${content}
<script>
function copyCode(btn){var pre=btn.closest('.example-block').querySelector('pre');navigator.clipboard.writeText(pre.textContent).then(function(){btn.textContent='\\u2713';setTimeout(function(){btn.innerHTML='\\u{1F4CB}'},1500)})}
// Highlight current sidebar link on scroll
var links=document.querySelectorAll('.sidebar-link[href^="#"]');
var sections=[];links.forEach(function(a){var t=document.getElementById(a.getAttribute('href').slice(1));if(t)sections.push({el:t,link:a})});
var io=new IntersectionObserver(function(entries){entries.forEach(function(e){if(e.isIntersecting){links.forEach(function(l){l.style.color=''});var m=sections.find(function(s){return s.el===e.target});if(m){m.link.style.color='var(--accent)';m.link.style.fontWeight='600'}}})},{threshold:0.1,rootMargin:'-80px 0px -60% 0px'});
sections.forEach(function(s){io.observe(s.el)});
</script>
</body></html>`;
}
