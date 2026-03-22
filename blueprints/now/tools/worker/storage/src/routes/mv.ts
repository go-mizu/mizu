import { createRoute, z } from "@hono/zod-openapi";
import type { App } from "../types";
import { auth } from "../middleware/auth";
import { validatePath } from "../lib/path";
import { mimeFromName } from "../lib/mime";
import { audit } from "../lib/audit";
import { invalidateCache } from "./find";
import { errRes } from "../schema";

const route = createRoute({
  method: "post",
  path: "/mv",
  tags: ["management"],
  security: [{ bearer: [] }],
  request: {
    body: {
      content: {
        "application/json": {
          schema: z.object({
            from: z.string().openapi({ example: "docs/old.md" }),
            to: z.string().openapi({ example: "docs/new.md" }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Moved",
      content: {
        "application/json": {
          schema: z.object({ from: z.string(), to: z.string() }),
        },
      },
    },
    400: errRes("Bad request"),
    403: errRes("Forbidden"),
    404: errRes("Not found"),
  },
});

export function register(app: App) {
  app.use("/mv", auth);

  app.openapi(route, async (c) => {
    const { from, to } = c.req.valid("json");

    const fromErr = validatePath(from);
    if (fromErr) return c.json({ error: "bad_request", message: `from: ${fromErr}` }, 400);

    const toErr = validatePath(to);
    if (toErr) return c.json({ error: "bad_request", message: `to: ${toErr}` }, 400);

    const actor = c.get("actor");
    const prefix = c.get("prefix");
    if (prefix && (!from.startsWith(prefix) || !to.startsWith(prefix))) {
      return c.json({ error: "forbidden", message: "Path not allowed" }, 403);
    }

    const fromKey = `${actor}/${from}`;
    const obj = await c.env.BUCKET.get(fromKey);
    if (!obj) return c.json({ error: "not_found", message: "Source not found" }, 404);

    await c.env.BUCKET.put(`${actor}/${to}`, obj.body, {
      httpMetadata: obj.httpMetadata,
      customMetadata: obj.customMetadata,
    });
    await c.env.BUCKET.delete(fromKey);

    const name = to.split("/").pop()!;
    const type = obj.httpMetadata?.contentType || mimeFromName(name);
    const now = Date.now();

    await c.env.DB.batch([
      c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path = ?").bind(actor, from),
      c.env.DB.prepare(
        "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
          "ON CONFLICT (owner, path) DO UPDATE SET name = excluded.name, size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
      ).bind(actor, to, name, obj.size, type, now),
    ]);

    invalidateCache(actor);
    audit(c, "mv", `${from} → ${to}`);
    return c.json({ from, to }, 200);
  });
}
