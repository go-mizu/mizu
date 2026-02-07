import { Hono } from "hono";

type Env = {
  AUTH_TOKEN: string;
};

interface FetchRequest {
  urls: string[];
  mode?: "status" | "full";
  timeout?: number;
  user_agent?: string;
}

interface FetchResult {
  url: string;
  status: number;
  content_type: string | null;
  content_length: number;
  redirect_url: string | null;
  fetch_time_ms: number;
  error: string | null;
  body: string | null;
}

const MAX_URLS = 500;
const DEFAULT_TIMEOUT = 10_000;
const DEFAULT_USER_AGENT = "mizu-crawler/1.0";

const app = new Hono<{ Bindings: Env }>();

// Health check
app.get("/", (c) => {
  return c.json({ status: "ok", version: "1.0.0" });
});

// Auth middleware for /fetch
app.use("/fetch", async (c, next) => {
  const auth = c.req.header("Authorization");
  if (!auth || !auth.startsWith("Bearer ")) {
    return c.json({ error: "Missing or invalid Authorization header" }, 401);
  }
  const token = auth.slice(7);
  if (token !== c.env.AUTH_TOKEN) {
    return c.json({ error: "Invalid token" }, 403);
  }
  await next();
});

// Batch URL fetcher
app.post("/fetch", async (c) => {
  const body = await c.req.json<FetchRequest>();

  if (!body.urls || !Array.isArray(body.urls) || body.urls.length === 0) {
    return c.json({ error: "urls must be a non-empty array" }, 400);
  }
  if (body.urls.length > MAX_URLS) {
    return c.json({ error: `Maximum ${MAX_URLS} URLs per batch` }, 400);
  }

  const mode = body.mode ?? "status";
  const timeout = body.timeout ?? DEFAULT_TIMEOUT;
  const userAgent = body.user_agent ?? DEFAULT_USER_AGENT;

  const results = await Promise.allSettled(
    body.urls.map((url) => fetchOne(url, mode, timeout, userAgent))
  );

  const response: FetchResult[] = results.map((r, i) => {
    if (r.status === "fulfilled") {
      return r.value;
    }
    return {
      url: body.urls[i],
      status: 0,
      content_type: null,
      content_length: 0,
      redirect_url: null,
      fetch_time_ms: 0,
      error: r.reason?.message ?? "Unknown error",
      body: null,
    };
  });

  return c.json(response);
});

async function fetchOne(
  url: string,
  mode: "status" | "full",
  timeout: number,
  userAgent: string
): Promise<FetchResult> {
  const start = Date.now();
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeout);

  try {
    const resp = await fetch(url, {
      method: "GET",
      headers: { "User-Agent": userAgent },
      signal: controller.signal,
      redirect: "follow",
    });

    const contentType = resp.headers.get("content-type");
    let contentLength = parseInt(resp.headers.get("content-length") ?? "0", 10);
    let body: string | null = null;

    if (mode === "full") {
      const text = await resp.text();
      contentLength = contentLength || text.length;
      body = text;
    }

    const redirectUrl = resp.url !== url ? resp.url : null;

    return {
      url,
      status: resp.status,
      content_type: contentType,
      content_length: contentLength || 0,
      redirect_url: redirectUrl,
      fetch_time_ms: Date.now() - start,
      error: null,
      body,
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
      content_type: null,
      content_length: 0,
      redirect_url: null,
      fetch_time_ms: Date.now() - start,
      error: msg,
      body: null,
    };
  } finally {
    clearTimeout(timer);
  }
}

export default app;
