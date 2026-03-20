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

// ── Markdown API docs (auto-generated from OpenAPI spec) ────────────
app.get("/api", (c) => {
  const spec = app.getOpenAPIDocument({
    openapi: "3.1.0",
    info: { title: "Storage API", version: "1.0.0", description: "" },
  });
  return c.html(renderMarkdownDocs(spec));
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

// ── Markdown docs generator ─────────────────────────────────────────

function renderMarkdownDocs(spec: any): string {
  const md: string[] = [];
  md.push(`# ${spec.info?.title || "API"}\n`);
  if (spec.info?.description) md.push(`${spec.info.description}\n`);

  // Group paths by tag
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

  for (const [tag, routes] of byTag) {
    md.push(`## ${tag.charAt(0).toUpperCase() + tag.slice(1)}\n`);
    for (const { method, path, op } of routes) {
      md.push(`### \`${method} ${path}\`\n`);
      if (op.summary) md.push(`${op.summary}\n`);
      if (op.security?.length) md.push(`**Auth required**\n`);

      // Request body
      const bodySchema = op.requestBody?.content?.["application/json"]?.schema;
      if (bodySchema) {
        md.push(`**Request body**\n\`\`\`json\n${schemaToExample(bodySchema)}\n\`\`\`\n`);
      }

      // Parameters
      if (op.parameters?.length) {
        md.push("**Parameters**\n");
        for (const p of op.parameters) {
          md.push(`- \`${p.name}\` (${p.in}) — ${p.description || p.schema?.type || ""}`);
        }
        md.push("");
      }

      // Responses
      for (const [code, resp] of Object.entries(op.responses || {})) {
        const r = resp as any;
        const schema = r.content?.["application/json"]?.schema;
        if (schema) {
          md.push(`**Response ${code}**\n\`\`\`json\n${schemaToExample(schema)}\n\`\`\`\n`);
        } else if (r.description) {
          md.push(`**Response ${code}** — ${r.description}\n`);
        }
      }
      md.push("---\n");
    }
  }

  // Wrap in simple HTML
  const escaped = md.join("\n").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  return `<!doctype html><html><head><meta charset="utf-8"><title>API Reference</title>
<style>body{max-width:800px;margin:40px auto;padding:0 20px;font-family:system-ui;line-height:1.6;color:#333}
pre{background:#f5f5f5;padding:12px;border-radius:6px;overflow-x:auto}code{background:#f0f0f0;padding:2px 6px;border-radius:3px}
h2{margin-top:2em;border-bottom:1px solid #ddd;padding-bottom:4px}h3{margin-top:1.5em}hr{border:none;border-top:1px solid #eee;margin:1.5em 0}</style>
</head><body><pre style="white-space:pre-wrap">${escaped}</pre></body></html>`;
}

function schemaToExample(schema: any): string {
  if (!schema || typeof schema !== "object") return "{}";
  if (schema.$ref) return `{ "$ref": "${schema.$ref}" }`;
  if (schema.type === "array") return `[${schemaToExample(schema.items)}]`;
  if (schema.type !== "object" || !schema.properties) {
    return JSON.stringify(schema.example ?? schema.type ?? "?");
  }
  const obj: Record<string, any> = {};
  for (const [key, prop] of Object.entries(schema.properties)) {
    const p = prop as any;
    if (p.example !== undefined) obj[key] = p.example;
    else if (p.type === "string") obj[key] = "";
    else if (p.type === "number" || p.type === "integer") obj[key] = 0;
    else if (p.type === "boolean") obj[key] = false;
    else if (p.type === "array") obj[key] = [];
    else obj[key] = null;
  }
  return JSON.stringify(obj, null, 2);
}
