import { extractLinks, filterLinks } from "./links";
import { htmlToMarkdown, extractTitle } from "./markdown";
import type { Env, CrawlMessage, JobConfig, JobRow } from "./types";

const DEFAULT_TIMEOUT_MS = 15_000;

export async function handleCrawlQueue(batch: MessageBatch<CrawlMessage>, env: Env): Promise<void> {
  for (const msg of batch.messages) {
    const { jobId, url, depth } = msg.body;

    try {
      await processUrl(env, jobId, url, depth);
      msg.ack();
    } catch (err) {
      console.error(`[queue] error processing ${url}:`, err);
      msg.retry();
    }
  }
}

async function processUrl(env: Env, jobId: string, url: string, depth: number): Promise<void> {
  // Load job (check not cancelled)
  const jobRow = await env.DB.prepare("SELECT * FROM jobs WHERE id = ?").bind(jobId).first<JobRow>();
  if (!jobRow) {
    console.warn(`[queue] job ${jobId} not found, skipping`);
    return;
  }
  if (jobRow.status === "cancelled_by_user") {
    // Mark page skipped
    await env.DB.prepare(
      "UPDATE pages SET status = 'skipped', http_status = 0 WHERE job_id = ? AND url = ?"
    ).bind(jobId, url).run();
    return;
  }

  const config: JobConfig = JSON.parse(jobRow.config);

  // Fetch the URL
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS);

  let html = "";
  let httpStatus = 0;
  let fetchError = false;

  try {
    const headers: Record<string, string> = {
      "User-Agent": config.userAgent,
      Accept: "text/html,application/xhtml+xml,*/*",
      "Accept-Language": "en-US,en;q=0.9",
      ...config.extraHeaders,
    };

    const resp = await fetch(url, {
      headers,
      redirect: "follow",
      signal: controller.signal,
    });

    httpStatus = resp.status;
    const ct = resp.headers.get("content-type") ?? "";
    if (ct.includes("text/html") || ct.includes("application/xhtml")) {
      html = await resp.text();
    }
  } catch (err) {
    fetchError = true;
    console.error(`[queue] fetch error for ${url}:`, err);
  } finally {
    clearTimeout(timer);
  }

  const now = Date.now();
  const title = html ? extractTitle(html) : "";
  const markdown = html && config.formats.includes("markdown") ? htmlToMarkdown(html) : null;
  const htmlContent = html && config.formats.includes("html") ? html : null;
  const pageStatus = fetchError ? "errored" : "completed";

  // Update page record
  await env.DB.prepare(
    `UPDATE pages
     SET status = ?, http_status = ?, title = ?, html = ?, markdown = ?, created_at = ?
     WHERE job_id = ? AND url = ?`
  )
    .bind(pageStatus, httpStatus, title, htmlContent, markdown, now, jobId, url)
    .run();

  // Increment finished count
  await env.DB.prepare(
    "UPDATE jobs SET finished = finished + 1, updated_at = ? WHERE id = ?"
  ).bind(now, jobId).run();

  // Discover and enqueue new links (only if fetch succeeded and within depth)
  if (!fetchError && html && depth < config.depth) {
    const rawLinks = await extractLinks(html, url);
    const filtered = filterLinks(rawLinks, url, config.url, config.options);

    for (const linkUrl of filtered) {
      // Atomically insert + check limit
      const currentJob = await env.DB.prepare("SELECT total FROM jobs WHERE id = ?")
        .bind(jobId)
        .first<{ total: number }>();

      if (!currentJob || currentJob.total >= config.limit) break;

      // Insert page row (ignore if already exists due to UNIQUE index)
      const result = await env.DB.prepare(
        `INSERT OR IGNORE INTO pages (job_id, url, status, depth, created_at)
         VALUES (?, ?, 'queued', ?, ?)`
      )
        .bind(jobId, linkUrl, depth + 1, now)
        .run();

      if (result.meta.changes > 0) {
        // New URL inserted — increment total and enqueue
        await env.DB.prepare(
          "UPDATE jobs SET total = total + 1, updated_at = ? WHERE id = ?"
        ).bind(now, jobId).run();

        await env.CRAWL_QUEUE.send({ jobId, url: linkUrl, depth: depth + 1 });
      }
    }
  }

  // Check if job is complete (all pages finished)
  const finalJob = await env.DB.prepare(
    "SELECT total, finished FROM jobs WHERE id = ?"
  ).bind(jobId).first<{ total: number; finished: number }>();

  if (finalJob && finalJob.total > 0 && finalJob.finished >= finalJob.total) {
    await env.DB.prepare(
      "UPDATE jobs SET status = 'completed', updated_at = ? WHERE id = ?"
    ).bind(Date.now(), jobId).run();
  }
}
