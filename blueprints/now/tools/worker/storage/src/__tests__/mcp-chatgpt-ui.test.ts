/**
 * ChatGPT Rich UI — widget HTML + MCP protocol tests
 *
 * Tests:
 *   1. Widget HTML generators produce valid, theme-aware HTML
 *   2. getWidgetHtml maps URIs correctly
 *   3. TOOL_WIDGET_MAP covers all 8 tools
 *   4. MCP protocol: tools/list includes _meta, resources/list + resources/read work
 *   5. MCP protocol: tool results include structuredContent
 */
import { describe, it, expect, beforeAll } from "vitest";
import { SELF } from "cloudflare:test";
import {
  filesWidget,
  viewerWidget,
  resultWidget,
  shareWidget,
  statsWidget,
  getWidgetHtml,
  WIDGET_RESOURCES,
  TOOL_WIDGET_MAP,
  WIDGET_RESOURCE_META,
} from "../mcp-widgets";

// ── Helpers ──────────────────────────────────────────────────────────────

/** Seed a file directly into R2 + D1 */
async function seedFile(actor: string, path: string, content: string, contentType = "text/plain") {
  const { env } = await import("cloudflare:test");
  const key = `${actor}/${path}`;
  const data = new TextEncoder().encode(content);
  await env.BUCKET.put(key, data, { httpMetadata: { contentType } });
  const name = path.split("/").pop()!;
  await env.DB.prepare(
    "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT (owner, path) DO UPDATE SET size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
  ).bind(actor, path, name, data.byteLength, contentType, Date.now()).run();
}

/** Set up schema + create test actor with session token */
async function setupDb(actor: string): Promise<string> {
  const { env } = await import("cloudflare:test");
  const db = env.DB;
  const stmts = [
    `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL DEFAULT 'human' CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS challenges (id TEXT PRIMARY KEY, actor TEXT NOT NULL, nonce TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, actor TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_sessions_actor ON sessions(actor, expires_at)`,
    `CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, actor TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, name TEXT NOT NULL DEFAULT '', prefix TEXT NOT NULL DEFAULT '', expires_at INTEGER, created_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_api_keys_actor ON api_keys(actor)`,
    `CREATE TABLE IF NOT EXISTS files (owner TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, size INTEGER NOT NULL DEFAULT 0, type TEXT NOT NULL DEFAULT 'application/octet-stream', updated_at INTEGER NOT NULL, PRIMARY KEY (owner, path))`,
    `CREATE INDEX IF NOT EXISTS idx_files_name ON files(owner, name COLLATE NOCASE)`,
    `CREATE TABLE IF NOT EXISTS audit_log (id INTEGER PRIMARY KEY AUTOINCREMENT, actor TEXT, action TEXT NOT NULL, path TEXT, ip TEXT, ts INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(actor, ts)`,
    `CREATE TABLE IF NOT EXISTS share_links (token TEXT PRIMARY KEY, actor TEXT NOT NULL, path TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_share_links_actor ON share_links(actor, created_at)`,
    `CREATE INDEX IF NOT EXISTS idx_share_links_expires ON share_links(expires_at)`,
    `CREATE TABLE IF NOT EXISTS oauth_clients (client_id TEXT PRIMARY KEY, redirect_uris TEXT NOT NULL DEFAULT '[]', client_name TEXT NOT NULL DEFAULT '', token_endpoint_auth_method TEXT NOT NULL DEFAULT 'none', created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS oauth_codes (code TEXT PRIMARY KEY, actor TEXT NOT NULL, client_id TEXT NOT NULL, redirect_uri TEXT NOT NULL, scope TEXT NOT NULL DEFAULT '*', code_challenge TEXT NOT NULL, code_challenge_method TEXT NOT NULL DEFAULT 'S256', expires_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires ON oauth_codes(expires_at)`,
    `CREATE TABLE IF NOT EXISTS magic_tokens (token TEXT PRIMARY KEY, email TEXT NOT NULL, actor TEXT, expires_at INTEGER NOT NULL)`,
  ];
  for (const sql of stmts) await db.exec(sql);
  // Create actor and session
  await db.prepare("INSERT OR IGNORE INTO actors (actor, type, created_at) VALUES (?, 'human', ?)").bind(actor, Date.now()).run();
  const sessionToken = "test-chatgpt-ui-session";
  await db.prepare("INSERT OR REPLACE INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)").bind(sessionToken, actor, Date.now() + 86400000).run();
  return sessionToken;
}

function mcpRequest(method: string, params?: any, id: number | string = 1) {
  return { jsonrpc: "2.0", id, method, params };
}

async function mcpCall(method: string, params: any, token: string) {
  const res = await SELF.fetch("http://localhost/mcp", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
    },
    body: JSON.stringify(mcpRequest(method, params)),
  });
  return res.json() as Promise<any>;
}

// ── Unit Tests: Widget HTML Generation ───────────────────────────────────

describe("ChatGPT UI — Widget HTML", () => {
  const widgets = [
    { name: "filesWidget", fn: filesWidget, uri: "ui://storage/files" },
    { name: "viewerWidget", fn: viewerWidget, uri: "ui://storage/viewer" },
    { name: "resultWidget", fn: resultWidget, uri: "ui://storage/result" },
    { name: "shareWidget", fn: shareWidget, uri: "ui://storage/share" },
    { name: "statsWidget", fn: statsWidget, uri: "ui://storage/stats" },
  ];

  for (const { name, fn, uri } of widgets) {
    describe(name, () => {
      const html = fn();

      it("returns valid HTML document", () => {
        expect(html).toContain("<!DOCTYPE html>");
        expect(html).toContain("<html>");
        expect(html).toContain("</html>");
        expect(html).toContain("<head>");
        expect(html).toContain("<body>");
      });

      it("includes theme detection via window.openai", () => {
        expect(html).toContain("window.openai");
        expect(html).toContain("data-theme");
      });

      it("includes dark mode CSS variables", () => {
        expect(html).toContain('[data-theme="dark"]');
      });

      it("reads structuredContent from window.openai.toolOutput", () => {
        expect(html).toContain("toolOutput");
      });

      it("reports intrinsic height", () => {
        expect(html).toContain("notifyIntrinsicHeight");
      });
    });
  }

  it("getWidgetHtml returns correct HTML for known URIs", () => {
    for (const r of WIDGET_RESOURCES) {
      const html = getWidgetHtml(r.uri);
      expect(html).not.toBeNull();
      expect(html).toContain("<!DOCTYPE html>");
    }
  });

  it("getWidgetHtml returns null for unknown URIs", () => {
    expect(getWidgetHtml("ui://unknown/widget")).toBeNull();
    expect(getWidgetHtml("")).toBeNull();
    expect(getWidgetHtml("http://example.com")).toBeNull();
  });
});

describe("ChatGPT UI — Widget Content", () => {
  it("filesWidget includes file browser elements", () => {
    const html = filesWidget();
    expect(html).toContain("fb-list");
    expect(html).toContain("fb-item");
    expect(html).toContain("storage_list");  // callTool reference
    expect(html).toContain("storage_read");  // callTool reference
  });

  it("viewerWidget includes code display and copy button", () => {
    const html = viewerWidget();
    expect(html).toContain("fv-code");
    expect(html).toContain("copyBtn");
    expect(html).toContain("clipboard");
  });

  it("viewerWidget includes markdown renderer", () => {
    const html = viewerWidget();
    expect(html).toContain("md2html");
    expect(html).toContain("fv-md");
    expect(html).toContain("markdown");
  });

  it("viewerWidget includes syntax highlighting", () => {
    const html = viewerWidget();
    expect(html).toContain("highlight");
    expect(html).toContain("tk-kw");
    expect(html).toContain("tk-str");
    expect(html).toContain("tk-cm");
  });

  it("resultWidget handles write/delete/move operations", () => {
    const html = resultWidget();
    expect(html).toContain("deleted");    // detects delete
    expect(html).toContain("old_path");   // detects move
    expect(html).toContain("File saved"); // write title
    expect(html).toContain("File moved"); // move title
  });

  it("shareWidget includes URL display and copy", () => {
    const html = shareWidget();
    expect(html).toContain("sh-url");
    expect(html).toContain("copyUrl");
    expect(html).toContain("clipboard");
    expect(html).toContain("Expires");
  });

  it("statsWidget includes file count and storage cards", () => {
    const html = statsWidget();
    expect(html).toContain("st-grid");
    expect(html).toContain("Files");
    expect(html).toContain("Storage");
    expect(html).toContain("file_count");
  });
});

describe("ChatGPT UI — Tool Widget Map", () => {
  const expectedTools = [
    "storage_list", "storage_read", "storage_write", "storage_delete",
    "storage_search", "storage_move", "storage_share", "storage_stats",
  ];

  it("covers all 8 tools", () => {
    for (const tool of expectedTools) {
      expect(TOOL_WIDGET_MAP[tool]).toBeDefined();
    }
  });

  it("each tool has required fields", () => {
    for (const tool of expectedTools) {
      const mapping = TOOL_WIDGET_MAP[tool];
      expect(mapping.uri).toBeTruthy();
      expect(mapping.invoking).toBeTruthy();
      expect(mapping.invoked).toBeTruthy();
      expect(mapping.widgetDescription).toBeTruthy();
    }
  });

  it("invoking/invoked messages are under 64 chars", () => {
    for (const tool of expectedTools) {
      const mapping = TOOL_WIDGET_MAP[tool];
      expect(mapping.invoking.length).toBeLessThanOrEqual(64);
      expect(mapping.invoked.length).toBeLessThanOrEqual(64);
    }
  });

  it("all widget URIs map to existing widgets", () => {
    const uris = new Set(Object.values(TOOL_WIDGET_MAP).map((m) => m.uri));
    for (const uri of uris) {
      expect(getWidgetHtml(uri)).not.toBeNull();
    }
  });
});

describe("ChatGPT UI — Resource Metadata", () => {
  it("has correct domain", () => {
    expect(WIDGET_RESOURCE_META.ui.domain).toBe("https://storage.liteio.dev");
  });

  it("has minimal CSP (no external domains)", () => {
    expect(WIDGET_RESOURCE_META.ui.csp.connectDomains).toEqual([]);
    expect(WIDGET_RESOURCE_META.ui.csp.resourceDomains).toEqual([]);
    expect(WIDGET_RESOURCE_META.ui.csp.frameDomains).toEqual([]);
  });

  it("prefers border", () => {
    expect(WIDGET_RESOURCE_META.ui.prefersBorder).toBe(true);
  });
});

// ── Integration Tests: MCP Protocol ─────────────────────────────────────

describe("ChatGPT UI — MCP Protocol", () => {
  let token: string;
  const testActor = "h/chatgpt-ui-test";

  beforeAll(async () => {
    token = await setupDb(testActor);
    await seedFile(testActor, "hello.txt", "Hello, world!", "text/plain");
    await seedFile(testActor, "docs/readme.md", "# Readme", "text/markdown");
  });

  describe("tools/list", () => {
    it("returns _meta with widget URIs for all tools", async () => {
      const res = await mcpCall("tools/list", {}, token);
      expect(res.result).toBeDefined();
      const tools = res.result.tools;
      expect(tools.length).toBe(8);

      for (const tool of tools) {
        expect(tool._meta).toBeDefined();
        expect(tool._meta.ui).toBeDefined();
        expect(tool._meta.ui.resourceUri).toBeTruthy();
        expect(tool._meta["openai/outputTemplate"]).toBeTruthy();
        expect(tool._meta["openai/toolInvocation/invoking"]).toBeTruthy();
        expect(tool._meta["openai/toolInvocation/invoked"]).toBeTruthy();
        expect(tool._meta["openai/widgetDescription"]).toBeTruthy();
      }
    });

    it("widget URIs match TOOL_WIDGET_MAP", async () => {
      const res = await mcpCall("tools/list", {}, token);
      for (const tool of res.result.tools) {
        const expected = TOOL_WIDGET_MAP[tool.name];
        expect(expected).toBeDefined();
        expect(tool._meta.ui.resourceUri).toBe(expected.uri);
      }
    });
  });

  describe("resources/list", () => {
    it("returns all widget resources", async () => {
      const res = await mcpCall("resources/list", {}, token);
      expect(res.result).toBeDefined();
      expect(res.result.resources.length).toBe(WIDGET_RESOURCES.length);

      for (const r of res.result.resources) {
        expect(r.uri).toBeTruthy();
        expect(r.name).toBeTruthy();
        expect(r.mimeType).toBe("text/html;profile=mcp-app");
      }
    });
  });

  describe("resources/read", () => {
    for (const resource of WIDGET_RESOURCES) {
      it(`returns HTML for ${resource.uri}`, async () => {
        const res = await mcpCall("resources/read", { uri: resource.uri }, token);
        expect(res.result).toBeDefined();
        expect(res.result.contents).toHaveLength(1);
        const content = res.result.contents[0];
        expect(content.uri).toBe(resource.uri);
        expect(content.mimeType).toBe("text/html;profile=mcp-app");
        expect(content.text).toContain("<!DOCTYPE html>");
        expect(content._meta.ui.domain).toBe("https://storage.liteio.dev");
      });
    }

    it("returns error for unknown URI", async () => {
      const res = await mcpCall("resources/read", { uri: "ui://unknown/x" }, token);
      expect(res.error).toBeDefined();
      expect(res.error.code).toBe(-32602);
    });
  });

  describe("initialize", () => {
    it("includes resources capability", async () => {
      const res = await mcpCall("initialize", {
        protocolVersion: "2025-06-18",
        clientInfo: { name: "test", version: "1.0" },
        capabilities: {},
      }, token);
      expect(res.result.capabilities.resources).toBeDefined();
      expect(res.result.capabilities.resources.listChanged).toBe(false);
    });
  });

  describe("tools/call — structuredContent", () => {
    it("storage_list returns structuredContent with entries", async () => {
      const res = await mcpCall("tools/call", { name: "storage_list", arguments: {} }, token);
      expect(res.result.structuredContent).toBeDefined();
      expect(res.result.structuredContent.prefix).toBeDefined();
      expect(res.result.structuredContent.entries).toBeInstanceOf(Array);
      // Should find hello.txt and docs/
      const names = res.result.structuredContent.entries.map((e: any) => e.name);
      expect(names).toContain("hello.txt");
      expect(names).toContain("docs/");
    });

    it("storage_list still returns text content for non-ChatGPT clients", async () => {
      const res = await mcpCall("tools/call", { name: "storage_list", arguments: {} }, token);
      expect(res.result.content).toBeDefined();
      expect(res.result.content[0].type).toBe("text");
      expect(res.result.content[0].text).toContain("hello.txt");
    });

    it("storage_read returns structuredContent with file metadata", async () => {
      const res = await mcpCall("tools/call", { name: "storage_read", arguments: { path: "hello.txt" } }, token);
      expect(res.result.structuredContent).toBeDefined();
      expect(res.result.structuredContent.path).toBe("hello.txt");
      expect(res.result.structuredContent.size).toBe(13);
      expect(res.result.structuredContent.content_type).toBe("text/plain");
      expect(res.result.structuredContent.is_text).toBe(true);
    });

    it("storage_read includes file content in _meta for widget", async () => {
      const res = await mcpCall("tools/call", { name: "storage_read", arguments: { path: "hello.txt" } }, token);
      expect(res.result._meta).toBeDefined();
      expect(res.result._meta.fileContent).toBe("Hello, world!");
    });

    it("storage_write returns structuredContent with file info", async () => {
      const res = await mcpCall("tools/call", {
        name: "storage_write",
        arguments: { path: "test-write.txt", content: "test content" },
      }, token);
      expect(res.result.structuredContent).toBeDefined();
      expect(res.result.structuredContent.path).toBe("test-write.txt");
      expect(res.result.structuredContent.size).toBeGreaterThan(0);
      expect(res.result.structuredContent.name).toBe("test-write.txt");
    });

    it("storage_search returns structuredContent with query and items", async () => {
      const res = await mcpCall("tools/call", {
        name: "storage_search",
        arguments: { query: "hello" },
      }, token);
      expect(res.result.structuredContent).toBeDefined();
      expect(res.result.structuredContent.query).toBe("hello");
      expect(res.result.structuredContent.count).toBeGreaterThanOrEqual(1);
      expect(res.result.structuredContent.items).toBeInstanceOf(Array);
    });

    it("storage_stats returns structuredContent with counts", async () => {
      const res = await mcpCall("tools/call", { name: "storage_stats", arguments: {} }, token);
      expect(res.result.structuredContent).toBeDefined();
      expect(res.result.structuredContent.file_count).toBeGreaterThanOrEqual(2);
      expect(res.result.structuredContent.total_size).toBeGreaterThan(0);
      expect(res.result.structuredContent.total_size_human).toBeTruthy();
    });

    it("storage_share returns structuredContent with URL", async () => {
      const res = await mcpCall("tools/call", {
        name: "storage_share",
        arguments: { path: "hello.txt" },
      }, token);
      expect(res.result.structuredContent).toBeDefined();
      expect(res.result.structuredContent.url).toContain("/s/");
      expect(res.result.structuredContent.path).toBe("hello.txt");
      expect(res.result.structuredContent.expires_at).toBeTruthy();
    });

    it("storage_move returns structuredContent with paths", async () => {
      // First write a file to move
      await mcpCall("tools/call", {
        name: "storage_write",
        arguments: { path: "moveme.txt", content: "move test" },
      }, token);
      const res = await mcpCall("tools/call", {
        name: "storage_move",
        arguments: { from: "moveme.txt", to: "moved.txt" },
      }, token);
      expect(res.result.structuredContent).toBeDefined();
      expect(res.result.structuredContent.old_path).toBe("moveme.txt");
      expect(res.result.structuredContent.new_path).toBe("moved.txt");
    });

    it("storage_delete returns structuredContent with deleted paths", async () => {
      // Write a file to delete
      await mcpCall("tools/call", {
        name: "storage_write",
        arguments: { path: "deleteme.txt", content: "delete test" },
      }, token);
      const res = await mcpCall("tools/call", {
        name: "storage_delete",
        arguments: { paths: ["deleteme.txt"] },
      }, token);
      expect(res.result.structuredContent).toBeDefined();
      expect(res.result.structuredContent.deleted).toContain("deleteme.txt");
    });
  });
});
