import { Hono } from "hono";
import { cors } from "hono/cors";
import { authMiddleware } from "./auth";
import { handlePost, handleGet, handleDelete } from "./crawl";
import { handleCrawlQueue } from "./queue";
import { handleContent } from "./content";
import { handleScreenshot } from "./screenshot";
import { handlePdf } from "./pdf";
import { handleMarkdown } from "./markdown-ep";
import { handleSnapshot } from "./snapshot";
import { handleScrape } from "./scrape";
import { handleJson } from "./json-ep";
import { handleLinks } from "./links-ep";
import type { Env, CrawlMessage } from "./types";

const app = new Hono<{ Bindings: Env }>();

app.use("*", cors());

// Health check (no auth)
app.get("/", (c) => c.json({ ok: true, service: "browser-worker" }));

// All /api/* routes require auth
app.use("/api/*", authMiddleware);

// Crawl endpoints (existing)
app.post("/api/crawl", handlePost);
app.get("/api/crawl/:id", handleGet);
app.delete("/api/crawl/:id", handleDelete);

// Single-URL rendering endpoints (new)
app.post("/api/content",    handleContent);
app.post("/api/screenshot", handleScreenshot);
app.post("/api/pdf",        handlePdf);
app.post("/api/markdown",   handleMarkdown);
app.post("/api/snapshot",   handleSnapshot);
app.post("/api/scrape",     handleScrape);
app.post("/api/json",       handleJson);
app.post("/api/links",      handleLinks);

// 404 fallback
app.notFound((c) =>
  c.json({ success: false, errors: [{ code: 404, message: "Not found" }], result: null }, 404)
);

// Error handler
app.onError((err, c) => {
  console.error("[worker] unhandled error:", err);
  return c.json({ success: false, errors: [{ code: 500, message: "Internal server error" }], result: null }, 500);
});

export default {
  fetch: app.fetch,

  async queue(batch: MessageBatch<unknown>, env: Env): Promise<void> {
    await handleCrawlQueue(batch as MessageBatch<CrawlMessage>, env);
  },
} satisfies ExportedHandler<Env>;
