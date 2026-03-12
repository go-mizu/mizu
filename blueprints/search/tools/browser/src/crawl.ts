import type { Context } from "hono";
import type { Env, CrawlRequest, JobConfig, JobRow, PageRow, ApiResponse, JobResult } from "./types";

function uuid(): string {
  return crypto.randomUUID();
}

// POST /api/crawl
export async function handlePost(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: CrawlRequest;
  try {
    body = await c.req.json<CrawlRequest>();
  } catch {
    return c.json<ApiResponse>({
      success: false,
      errors: [{ code: 1001, message: "Invalid JSON body" }],
      result: null,
    }, 400);
  }

  if (!body.url) {
    return c.json<ApiResponse>({
      success: false,
      errors: [{ code: 1001, message: "url is required" }],
      result: null,
    }, 400);
  }

  // Validate URL
  try {
    new URL(body.url);
  } catch {
    return c.json<ApiResponse>({
      success: false,
      errors: [{ code: 1001, message: "url is not a valid URL" }],
      result: null,
    }, 400);
  }

  const config: JobConfig = {
    url: body.url,
    limit: Math.min(body.limit ?? 10, 100_000),
    depth: body.depth ?? 100,
    formats: body.formats ?? ["markdown"],
    userAgent: body.userAgent ?? "mizu-browser/1.0",
    extraHeaders: body.setExtraHTTPHeaders ?? {},
    options: {
      includeSubdomains: body.options?.includeSubdomains ?? false,
      includeExternalLinks: body.options?.includeExternalLinks ?? false,
      includePatterns: body.options?.includePatterns ?? [],
      excludePatterns: body.options?.excludePatterns ?? [],
    },
  };

  const jobId = uuid();
  const now = Date.now();

  // Create job row
  await c.env.DB.prepare(
    `INSERT INTO jobs (id, url, status, config, total, finished, created_at, updated_at)
     VALUES (?, ?, 'running', ?, 1, 0, ?, ?)`
  )
    .bind(jobId, config.url, JSON.stringify(config), now, now)
    .run();

  // Insert seed page
  await c.env.DB.prepare(
    `INSERT INTO pages (job_id, url, status, depth, created_at)
     VALUES (?, ?, 'queued', 0, ?)`
  )
    .bind(jobId, config.url, now)
    .run();

  // Enqueue seed URL
  await c.env.CRAWL_QUEUE.send({ jobId, url: config.url, depth: 0 });

  return c.json<ApiResponse<string>>({ success: true, result: jobId });
}

// GET /api/crawl/:id
export async function handleGet(c: Context<{ Bindings: Env }>): Promise<Response> {
  const id = c.req.param("id");
  const cursor = parseInt(c.req.query("cursor") ?? "0", 10) || 0;
  const limit = parseInt(c.req.query("limit") ?? "100", 10) || 100;

  const job = await c.env.DB.prepare("SELECT * FROM jobs WHERE id = ?")
    .bind(id)
    .first<JobRow>();

  if (!job) {
    return c.json<ApiResponse>({
      success: false,
      errors: [{ code: 1002, message: "Job not found" }],
      result: null,
    }, 404);
  }

  const rows = await c.env.DB.prepare(
    `SELECT * FROM pages WHERE job_id = ? AND id > ? ORDER BY id ASC LIMIT ?`
  )
    .bind(id, cursor, limit)
    .all<PageRow>();

  const records = (rows.results ?? []).map((p) => ({
    url: p.url,
    status: p.status,
    markdown: p.markdown ?? null,
    html: p.html ?? null,
    metadata: {
      status: p.http_status,
      title: p.title,
      url: p.url,
    },
  }));

  const newCursor = records.length > 0 ? (rows.results[rows.results.length - 1]?.id ?? cursor) : cursor;

  const result: JobResult = {
    id: job.id,
    status: job.status,
    total: job.total,
    finished: job.finished,
    cursor: newCursor,
    records,
  };

  return c.json<ApiResponse<JobResult>>({ success: true, result });
}

// DELETE /api/crawl/:id
export async function handleDelete(c: Context<{ Bindings: Env }>): Promise<Response> {
  const id = c.req.param("id");

  const job = await c.env.DB.prepare("SELECT id FROM jobs WHERE id = ?")
    .bind(id)
    .first<{ id: string }>();

  if (!job) {
    return c.json<ApiResponse>({
      success: false,
      errors: [{ code: 1002, message: "Job not found" }],
      result: null,
    }, 404);
  }

  await c.env.DB.prepare(
    "UPDATE jobs SET status = 'cancelled_by_user', updated_at = ? WHERE id = ?"
  ).bind(Date.now(), id).run();

  return c.json<ApiResponse>({
    success: true,
    result: { id, status: "cancelled_by_user" },
  });
}
