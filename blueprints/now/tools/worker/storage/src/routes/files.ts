import { z } from "@hono/zod-openapi";
import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { mimeFromName, isInlineType } from "../lib/mime";
import { err } from "../lib/error";
import { wildcardPath, validatePath } from "../lib/path";
import { audit } from "../lib/audit";
import { invalidateCache } from "./find";
import { ErrorSchema, errRes } from "../schema";

type C = Context<{ Bindings: Env; Variables: Variables }>;

const MAX_SIZE = 100 * 1024 * 1024;

function checkPrefix(c: C, path: string): Response | null {
  const pfx = c.get("prefix");
  if (pfx && !path.startsWith(pfx)) return err(c, "forbidden", "Path not allowed for this token");
  return null;
}

function safeName(name: string): string {
  return name.replace(/["\\\r\n]/g, "_").replace(/[^\x20-\x7E]/g, "_");
}

// ── Handlers ────────────────────────────────────────────────────────

async function writeFile(c: C) {
  const path = wildcardPath(c, "/f/");
  if (!path || path.endsWith("/")) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const prefixErr = checkPrefix(c, path);
  if (prefixErr) return prefixErr;

  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_SIZE) return err(c, "too_large", "File exceeds 100 MB");
  const body = await c.req.arrayBuffer();
  if (body.byteLength > MAX_SIZE) return err(c, "too_large", "File exceeds 100 MB");

  const actor = c.get("actor");
  const name = path.split("/").pop()!;
  const type = c.req.header("Content-Type") || mimeFromName(name);
  const now = Date.now();

  await c.env.BUCKET.put(`${actor}/${path}`, body, { httpMetadata: { contentType: type } });
  await c.env.DB.prepare(
    "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT (owner, path) DO UPDATE SET size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
  )
    .bind(actor, path, name, body.byteLength, type, now)
    .run();

  invalidateCache(actor);
  audit(c, "write", path);
  return c.json({ path, name, size: body.byteLength, type, updated_at: now });
}

async function readFile(c: C) {
  const path = wildcardPath(c, "/f/");
  if (!path) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const prefixErr = checkPrefix(c, path);
  if (prefixErr) return prefixErr;

  const actor = c.get("actor");
  const obj = await c.env.BUCKET.get(`${actor}/${path}`);
  if (!obj) return err(c, "not_found", "File not found");

  audit(c, "read", path);
  const ct = obj.httpMetadata?.contentType || "application/octet-stream";
  const name = path.split("/").pop()!;
  const headers = new Headers();
  headers.set("Content-Type", ct);
  headers.set("Content-Length", obj.size.toString());
  headers.set("ETag", obj.etag);
  headers.set("Content-Disposition", `${isInlineType(ct) ? "inline" : "attachment"}; filename="${safeName(name)}"`);
  return new Response(obj.body, { headers });
}

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

// ── Registration (routes + OpenAPI metadata) ────────────────────────

const FileInfo = z.object({
  path: z.string(),
  name: z.string(),
  size: z.number().int(),
  type: z.string(),
  updated_at: z.number().int(),
}).openapi("FileInfo");

export function register(app: App) {
  // Plain Hono routes (wildcard paths — can't use createRoute)
  app.put("/f/*", auth, writeFile);
  app.get("/f/*", auth, readFile);
  app.delete("/f/*", auth, deleteFile);
  app.on("HEAD", "/f/*", auth, statFile);

  // Manual OpenAPI registration for wildcard routes
  const pathParam = {
    name: "path" as const,
    in: "path" as const,
    required: true,
    schema: { type: "string" as const },
    description: "File path (e.g. docs/readme.md)",
  };

  const params = z.object({ path: z.string() });
  const base = { path: "/f/{path}" as const, tags: ["files"], security: [{ bearer: [] as string[] }], request: { params } };

  app.openAPIRegistry.registerPath({ ...base, method: "put", summary: "Write a file (create or overwrite)", responses: { 200: { description: "File metadata" }, 400: { description: "Bad request" }, 413: { description: "Too large" } } });
  app.openAPIRegistry.registerPath({ ...base, method: "get", summary: "Read a file", responses: { 200: { description: "File content (binary stream)" }, 404: { description: "Not found" } } });
  app.openAPIRegistry.registerPath({ ...base, method: "delete", summary: "Delete file or directory (trailing /)", responses: { 200: { description: "Delete count" } } });
  app.openAPIRegistry.registerPath({ ...base, method: "head", summary: "Stat file (metadata in headers)", responses: { 200: { description: "Headers: Content-Type, Content-Length, ETag" }, 404: { description: "Not found" } } });
}
