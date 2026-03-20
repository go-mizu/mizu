import { z } from "@hono/zod-openapi";
import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { err } from "../lib/error";
import { wildcardPath } from "../lib/path";

type C = Context<{ Bindings: Env; Variables: Variables }>;

async function ls(c: C) {
  const actor = c.get("actor");
  const prefix = wildcardPath(c, "/ls/") || "";

  const pfx = c.get("prefix");
  if (pfx && !prefix.startsWith(pfx)) return err(c, "forbidden", "Path not allowed");

  const limit = Math.min(parseInt(c.req.query("limit") || "200", 10), 1000);
  const offset = parseInt(c.req.query("offset") || "0", 10);

  const { results } = await c.env.DB.prepare(
    "SELECT path, name, size, type, updated_at FROM files WHERE owner = ? AND path LIKE ? ORDER BY path LIMIT ? OFFSET ?",
  )
    .bind(actor, `${prefix}%`, limit + 1, offset)
    .all();

  const rows = results || [];
  const truncated = rows.length > limit;
  if (truncated) rows.pop();

  const entries: { name: string; type: string; size?: number; updated_at?: number }[] = [];
  const dirs = new Set<string>();

  for (const row of rows) {
    const relative = (row.path as string).slice(prefix.length);
    const slash = relative.indexOf("/");
    if (slash === -1) {
      entries.push({
        name: relative,
        type: row.type as string,
        size: row.size as number,
        updated_at: row.updated_at as number,
      });
    } else {
      const dir = relative.slice(0, slash + 1);
      if (!dirs.has(dir)) {
        dirs.add(dir);
        entries.push({ name: dir, type: "directory" });
      }
    }
  }

  return c.json({ prefix: prefix || "/", entries, truncated });
}

export function register(app: App) {
  app.get("/ls", auth, ls);
  app.get("/ls/*", auth, ls);

  // Manual OpenAPI registration
  app.openAPIRegistry.registerPath({
    method: "get",
    path: "/ls/{prefix}",
    summary: "List directory entries",
    tags: ["listing"],
    security: [{ bearer: [] }],
    request: {
      params: z.object({ prefix: z.string().openapi({ description: "Directory prefix" }) }),
      query: z.object({
        limit: z.coerce.number().int().default(200).optional(),
        offset: z.coerce.number().int().default(0).optional(),
      }),
    },
    responses: {
      200: {
        description: "Directory listing",
        content: {
          "application/json": {
            schema: z.object({
              prefix: z.string(),
              entries: z.array(
                z.object({
                  name: z.string(),
                  type: z.string().openapi({ description: "MIME type or 'directory'" }),
                  size: z.number().int().optional(),
                  updated_at: z.number().int().optional(),
                }),
              ),
              truncated: z.boolean(),
            }),
          },
        },
      },
    },
  });
}
