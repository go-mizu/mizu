import { createRoute, z } from "@hono/zod-openapi";
import type { App } from "../types";
import { presignUrl } from "../lib/presign";
import { errRes } from "../schema";

// ── GET /s/:token ───────────────────────────────────────────────────
// Public share link access — redirects to presigned R2 URL. No auth required.
const accessRoute = createRoute({
  method: "get",
  path: "/s/{token}",
  summary: "Access a shared file",
  tags: ["sharing"],
  request: {
    params: z.object({
      token: z.string().openapi({ description: "Share token" }),
    }),
  },
  responses: {
    302: { description: "Redirect to presigned R2 URL" },
    401: errRes("Invalid or expired token"),
  },
});

export function register(app: App) {
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
