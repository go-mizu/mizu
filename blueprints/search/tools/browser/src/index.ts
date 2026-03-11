import { Hono } from "hono";
import { cors } from "hono/cors";
import { authMiddleware } from "./auth";
import { handlePost, handleGet, handleDelete } from "./crawl";
import { handleCrawlQueue } from "./queue";
import type { Env, CrawlMessage } from "./types";

const app = new Hono<{ Bindings: Env }>();

app.use("*", cors());

// Health check (no auth)
app.get("/", (c) => c.json({ ok: true, service: "browser-worker" }));

// All /api/* routes require auth
app.use("/api/*", authMiddleware);

app.post("/api/crawl", handlePost);
app.get("/api/crawl/:id", handleGet);
app.delete("/api/crawl/:id", handleDelete);

// 404 fallback
app.notFound((c) => c.json({ success: false, errors: [{ code: 404, message: "Not found" }], result: null }, 404));

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
