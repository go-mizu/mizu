import { z } from "@hono/zod-openapi";
import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { err } from "../lib/error";
import { wildcardPath, validatePath } from "../lib/path";
import { audit } from "../lib/audit";
import { invalidateCache } from "./find";

type C = Context<{ Bindings: Env; Variables: Variables }>;

function checkPrefix(c: C, path: string): Response | null {
  const pfx = c.get("prefix");
  if (pfx && !path.startsWith(pfx)) return err(c, "forbidden", "Path not allowed for this token");
  return null;
}

// ── Handlers ────────────────────────────────────────────────────────

async function deleteFile(c: C) {
  const path = wildcardPath(c, "/f/");
  if (!path) return err(c, "bad_request", "Path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const prefixErr = checkPrefix(c, path);
  if (prefixErr) return prefixErr;

  const actor = c.get("actor");

  if (path.endsWith("/")) {
    const prefix = `${actor}/${path}`;
    let cursor: string | undefined;
    let deleted = 0;
    do {
      const list = await c.env.BUCKET.list({ prefix, cursor, limit: 1000 });
      if (list.objects.length) {
        await c.env.BUCKET.delete(list.objects.map((o) => o.key));
        deleted += list.objects.length;
      }
      cursor = list.truncated ? list.cursor : undefined;
    } while (cursor);

    await c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path LIKE ?")
      .bind(actor, `${path}%`).run();
    invalidateCache(actor);
    audit(c, "rm", path);
    return c.json({ deleted });
  }

  await c.env.BUCKET.delete(`${actor}/${path}`);
  await c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path = ?")
    .bind(actor, path).run();
  invalidateCache(actor);
  audit(c, "rm", path);
  return c.json({ deleted: 1 });
}

async function statFile(c: C) {
  const path = wildcardPath(c, "/f/");
  if (!path) return c.body(null, 400);
  const pathErr = validatePath(path);
  if (pathErr) return c.body(null, 400);
  const prefixErr = checkPrefix(c, path);
  if (prefixErr) return prefixErr;

  const actor = c.get("actor");
  const obj = await c.env.BUCKET.head(`${actor}/${path}`);
  if (!obj) return c.body(null, 404);

  return c.body(null, 200, {
    "Content-Type": obj.httpMetadata?.contentType || "application/octet-stream",
    "Content-Length": obj.size.toString(),
    "ETag": obj.etag,
  });
}

// Route handler for /f/* — dispatches by method.
// No file data flows through the Worker: PUT/GET return 410 Gone.
async function fileHandler(c: C) {
  const method = c.req.method;
  if (method === "DELETE") return deleteFile(c);
  if (method === "HEAD") return statFile(c);
  // PUT and GET are gone — use presigned URLs
  return err(c, "gone", "Use presigned URLs for file read/write. See POST /presign/upload and GET /presign/read/*");
}

// ── Registration (routes + OpenAPI metadata) ────────────────────────

export function register(app: App) {
  // Single route for /f/* — dispatches by method internally.
  // Data read/write uses presigned URLs — no file data flows through the Worker.
  app.all("/f/*", auth, fileHandler);

  // Manual OpenAPI registration for wildcard routes
  const params = z.object({ path: z.string() });
  const base = { path: "/f/{path}" as const, tags: ["files"], security: [{ bearer: [] as string[] }], request: { params } };

  app.openAPIRegistry.registerPath({ ...base, method: "delete", summary: "Delete file or directory (trailing /)", responses: { 200: { description: "Delete count" } } });
  app.openAPIRegistry.registerPath({ ...base, method: "head", summary: "Stat file (metadata in headers)", responses: { 200: { description: "Headers: Content-Type, Content-Length, ETag" }, 404: { description: "Not found" } } });
}
