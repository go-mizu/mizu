import { Hono } from "hono";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Env = { AUTH_TOKEN: string; AI: any };

interface CrawlRequest {
  urls: string[];
  browser?: boolean;
  timeout?: number;
}

interface CrawlResult {
  url: string;
  status: number;
  html: string | null;
  markdown: string | null;
  title: string | null;
  content_type: string | null;
  content_length: number;
  redirect_url: string | null;
  fetch_time_ms: number;
  error: string | null;
}

const MAX_URLS = 10;
const DEFAULT_TIMEOUT = 15_000;
const MAX_HTML_SIZE = 5_000_000; // 5MB
const UA =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36";

// In-memory cache per isolate (survives across requests within same isolate)
const htmlCache = new Map<string, { html: string; status: number; contentType: string; redirectUrl: string | null }>();
const mdCache = new Map<string, string>();

const app = new Hono<{ Bindings: Env }>();

// Health check
app.get("/", (c) => {
  return c.json({
    status: "ok",
    version: "1.0.0",
    cache: { html: htmlCache.size, markdown: mdCache.size },
  });
});

// Auth middleware
app.use("/crawl", async (c, next) => {
  const auth = c.req.header("Authorization");
  if (!auth || !auth.startsWith("Bearer ")) {
    return c.json({ error: "Missing or invalid Authorization header" }, 401);
  }
  if (auth.slice(7) !== c.env.AUTH_TOKEN) {
    return c.json({ error: "Invalid token" }, 403);
  }
  await next();
});

// Batch crawl endpoint
app.post("/crawl", async (c) => {
  const body = await c.req.json<CrawlRequest>();

  if (!body.urls || !Array.isArray(body.urls) || body.urls.length === 0) {
    return c.json({ error: "urls must be a non-empty array" }, 400);
  }
  if (body.urls.length > MAX_URLS) {
    return c.json({ error: `Maximum ${MAX_URLS} URLs per batch` }, 400);
  }

  const timeout = body.timeout ?? DEFAULT_TIMEOUT;

  // Process URLs with controlled concurrency (3 at a time to stay within CPU limits)
  const results: CrawlResult[] = [];
  const concurrency = 3;
  for (let i = 0; i < body.urls.length; i += concurrency) {
    const chunk = body.urls.slice(i, i + concurrency);
    const chunkResults = await Promise.allSettled(
      chunk.map((url) => crawlOne(url, timeout, c.env))
    );
    for (let j = 0; j < chunkResults.length; j++) {
      const r = chunkResults[j];
      if (r.status === "fulfilled") {
        results.push(r.value);
      } else {
        results.push({
          url: chunk[j],
          status: 0,
          html: null,
          markdown: null,
          title: null,
          content_type: null,
          content_length: 0,
          redirect_url: null,
          fetch_time_ms: 0,
          error: r.reason?.message ?? "Unknown error",
        });
      }
    }
  }

  return c.json(results);
});

async function crawlOne(
  url: string,
  timeout: number,
  env: Env
): Promise<CrawlResult> {
  const start = Date.now();

  // Check HTML cache
  const cached = htmlCache.get(url);
  if (cached) {
    const md = mdCache.get(url) ?? null;
    return {
      url,
      status: cached.status,
      html: cached.html,
      markdown: md,
      title: extractTitle(cached.html),
      content_type: cached.contentType,
      content_length: cached.html.length,
      redirect_url: cached.redirectUrl,
      fetch_time_ms: Date.now() - start,
      error: null,
    };
  }

  // Fetch HTML
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeout);

  try {
    const resp = await fetch(url, {
      method: "GET",
      headers: {
        "User-Agent": UA,
        Accept:
          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.9",
      },
      signal: controller.signal,
      redirect: "follow",
    });

    const contentType = resp.headers.get("content-type") ?? "";
    const contentLength = parseInt(
      resp.headers.get("content-length") ?? "0",
      10
    );

    // Skip non-HTML responses
    if (
      !contentType.includes("text/html") &&
      !contentType.includes("application/xhtml")
    ) {
      return {
        url,
        status: resp.status,
        html: null,
        markdown: null,
        title: null,
        content_type: contentType,
        content_length: contentLength,
        redirect_url: resp.url !== url ? resp.url : null,
        fetch_time_ms: Date.now() - start,
        error: null,
      };
    }

    // Read body with size limit
    if (contentLength > MAX_HTML_SIZE) {
      return {
        url,
        status: resp.status,
        html: null,
        markdown: null,
        title: null,
        content_type: contentType,
        content_length: contentLength,
        redirect_url: resp.url !== url ? resp.url : null,
        fetch_time_ms: Date.now() - start,
        error: `Response too large: ${contentLength} bytes`,
      };
    }

    const html = await resp.text();
    if (html.length > MAX_HTML_SIZE) {
      return {
        url,
        status: resp.status,
        html: null,
        markdown: null,
        title: null,
        content_type: contentType,
        content_length: html.length,
        redirect_url: resp.url !== url ? resp.url : null,
        fetch_time_ms: Date.now() - start,
        error: `Response too large: ${html.length} bytes`,
      };
    }

    const redirectUrl = resp.url !== url ? resp.url : null;

    // Cache HTML
    htmlCache.set(url, {
      html,
      status: resp.status,
      contentType,
      redirectUrl,
    });

    // Convert to markdown via Workers AI (best-effort, don't fail the request)
    let markdown: string | null = null;
    try {
      markdown = await toMarkdown(html, env);
      if (markdown) {
        mdCache.set(url, markdown);
      }
    } catch {
      // markdown conversion failed, continue with html only
    }

    return {
      url,
      status: resp.status,
      html,
      markdown,
      title: extractTitle(html),
      content_type: contentType,
      content_length: html.length,
      redirect_url: redirectUrl,
      fetch_time_ms: Date.now() - start,
      error: null,
    };
  } catch (err: unknown) {
    const msg =
      err instanceof Error
        ? err.name === "AbortError"
          ? `Timeout after ${timeout}ms`
          : err.message
        : "Unknown error";
    return {
      url,
      status: 0,
      html: null,
      markdown: null,
      title: null,
      content_type: null,
      content_length: 0,
      redirect_url: null,
      fetch_time_ms: Date.now() - start,
      error: msg,
    };
  } finally {
    clearTimeout(timer);
  }
}

// Workers AI HTML → Markdown conversion
async function toMarkdown(
  html: string,
  env: Env
): Promise<string | null> {
  if (!env.AI) return null;
  try {
    const blob = new Blob([html], { type: "text/html" });
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const results: any[] = await env.AI.toMarkdown([
      { name: "page.html", blob },
    ]);
    if (!results || results.length === 0) return null;
    const r = results[0];
    const md = (r.data ?? r.result ?? "") as string;
    return md.trim() || null;
  } catch {
    return null;
  }
}

function extractTitle(html: string): string | null {
  const m = html.match(/<title[^>]*>([^<]+)<\/title>/i);
  return m ? m[1].trim() : null;
}

export default app;
