import { createRoute, z } from "@hono/zod-openapi";
import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { validatePath } from "../lib/path";
import { audit } from "../lib/audit";
import { shareToken } from "../lib/id";
import { presignUrl } from "../lib/presign";
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

    const key = `${row.actor}/${row.path}`;

    // Redirect directly to R2 via presigned URL (no proxy)
    const endpoint = c.env.R2_ENDPOINT;
    const accessKeyId = c.env.R2_ACCESS_KEY_ID;
    const secretAccessKey = c.env.R2_SECRET_ACCESS_KEY;
    if (!endpoint || !accessKeyId || !secretAccessKey) {
      return c.json({ error: "not_configured", message: "Storage not configured" }, 500);
    }

    const ttl = Math.max(1, Math.floor((row.expires_at - Date.now()) / 1000));
    const url = await presignUrl({
      method: "GET",
      key,
      bucket: c.env.R2_BUCKET_NAME || "storage-files",
      endpoint,
      accessKeyId,
      secretAccessKey,
      expiresIn: Math.min(ttl, 3600),
    });
    return c.redirect(url, 302) as any;
  });
}
