import { createRoute, z } from "@hono/zod-openapi";
import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { validatePath } from "../lib/path";
import { isInlineType } from "../lib/mime";
import { audit } from "../lib/audit";
import { shareToken } from "../lib/id";
import { errRes } from "../schema";

type C = Context<{ Bindings: Env; Variables: Variables }>;

const DEFAULT_TTL = 3600;
const MAX_TTL = 7 * 86400;

// ── POST /share ─────────────────────────────────────────────────────
const shareRoute = createRoute({
  method: "post",
  path: "/share",
  tags: ["sharing"],
  security: [{ bearer: [] }],
  request: {
    body: {
      content: {
        "application/json": {
          schema: z.object({
            path: z.string().openapi({ example: "docs/report.pdf" }),
            ttl: z
              .number()
              .int()
              .min(60)
              .max(604800)
              .default(3600)
              .optional()
              .openapi({ description: "Seconds (default 3600, max 7 days)" }),
          }),
        },
      },
    },
  },
  responses: {
    201: {
      description: "Share URL created",
      content: {
        "application/json": {
          schema: z.object({
            url: z.string(),
            token: z.string(),
            expires_at: z.number().int(),
            ttl: z.number().int(),
          }),
        },
      },
    },
    400: errRes("Bad request"),
    403: errRes("Forbidden"),
    404: errRes("Not found"),
  },
});

// ── GET /s/:token ───────────────────────────────────────────────────
const accessRoute = createRoute({
  method: "get",
  path: "/s/{token}",
  tags: ["sharing"],
  request: {
    params: z.object({
      token: z.string().openapi({ description: "Share token" }),
    }),
  },
  responses: {
    200: { description: "File content (binary stream)" },
    401: errRes("Invalid or expired token"),
    404: errRes("Not found"),
  },
});

export function register(app: App) {
  app.use("/share", auth);

  app.openapi(shareRoute, async (c) => {
    const { path, ttl: rawTtl } = c.req.valid("json");

    const pathErr = validatePath(path);
    if (pathErr) return c.json({ error: "bad_request", message: pathErr }, 400);

    const actor = c.get("actor");
    const prefix = c.get("prefix");
    if (prefix && !path.startsWith(prefix)) {
      return c.json({ error: "forbidden", message: "Path not allowed" }, 403);
    }

    const head = await c.env.BUCKET.head(`${actor}/${path}`);
    if (!head) return c.json({ error: "not_found", message: "File not found" }, 404);

    const ttl = Math.min(Math.max(rawTtl || DEFAULT_TTL, 60), MAX_TTL);
    const now = Date.now();
    const expiresAt = now + ttl * 1000;
    const token = shareToken();

    await c.env.DB.prepare(
      "INSERT INTO share_links (token, actor, path, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
    )
      .bind(token, actor, path, expiresAt, now)
      .run();

    const origin = new URL(c.req.url).origin;
    audit(c, "share", path);

    return c.json({ url: `${origin}/s/${token}`, token, expires_at: expiresAt, ttl }, 201);
  });

  app.openapi(accessRoute, async (c) => {
    const { token } = c.req.valid("param");

    const row = await c.env.DB.prepare(
      "SELECT actor, path, expires_at FROM share_links WHERE token = ?",
    )
      .bind(token)
      .first<{ actor: string; path: string; expires_at: number }>();

    if (!row || row.expires_at < Date.now()) {
      return c.json({ error: "unauthorized", message: "Invalid or expired share link" }, 401);
    }

    const obj = await c.env.BUCKET.get(`${row.actor}/${row.path}`);
    if (!obj) return c.json({ error: "not_found", message: "File not found" }, 404);

    const name = row.path.split("/").pop()!;
    const ct = obj.httpMetadata?.contentType || "application/octet-stream";
    const safe = name.replace(/["\\\r\n]/g, "_").replace(/[^\x20-\x7E]/g, "_");

    const headers = new Headers();
    headers.set("Content-Type", ct);
    headers.set("Content-Length", obj.size.toString());
    headers.set("ETag", obj.etag);
    headers.set(
      "Content-Disposition",
      `${isInlineType(ct) ? "inline" : "attachment"}; filename="${safe}"`,
    );
    headers.set("Cache-Control", "private, max-age=3600");

    return new Response(obj.body, { headers }) as any;
  });
}
