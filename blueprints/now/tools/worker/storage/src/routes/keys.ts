import { createRoute, z } from "@hono/zod-openapi";
import type { App } from "../types";
import { auth } from "../middleware/auth";
import { apiKeyId, apiKeyToken } from "../lib/id";
import { sha256 } from "../middleware/auth";
import { audit } from "../lib/audit";
import { errRes } from "../schema";

const createKeyRoute = createRoute({
  method: "post",
  path: "/auth/keys",
  summary: "Create an API key",
  tags: ["keys"],
  security: [{ bearer: [] }],
  request: {
    body: {
      content: {
        "application/json": {
          schema: z.object({
            name: z.string().default("").optional().openapi({ example: "deploy-bot" }),
            prefix: z
              .string()
              .default("")
              .optional()
              .openapi({ description: "Restrict to paths starting with this prefix" }),
            expires_in: z
              .number()
              .int()
              .positive()
              .optional()
              .openapi({ description: "Seconds until expiry" }),
          }),
        },
      },
    },
  },
  responses: {
    201: {
      description: "API key (token shown once)",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string(),
            token: z.string().openapi({ description: "Store securely — not shown again" }),
            name: z.string(),
            prefix: z.string(),
            expires_at: z.number().int().nullable(),
          }),
        },
      },
    },
  },
});

const listKeysRoute = createRoute({
  method: "get",
  path: "/auth/keys",
  summary: "List API keys",
  tags: ["keys"],
  security: [{ bearer: [] }],
  responses: {
    200: {
      description: "API keys",
      content: {
        "application/json": {
          schema: z.object({
            keys: z.array(
              z.object({
                id: z.string(),
                name: z.string(),
                prefix: z.string(),
                expires_at: z.number().int().nullable(),
                created_at: z.number().int(),
              }),
            ),
          }),
        },
      },
    },
  },
});

const deleteKeyRoute = createRoute({
  method: "delete",
  path: "/auth/keys/{id}",
  summary: "Delete an API key",
  tags: ["keys"],
  security: [{ bearer: [] }],
  request: {
    params: z.object({ id: z.string() }),
  },
  responses: {
    200: {
      description: "Deleted",
      content: {
        "application/json": {
          schema: z.object({ deleted: z.boolean() }),
        },
      },
    },
    404: errRes("Not found"),
  },
});

export function register(app: App) {
  app.use("/auth/keys", auth);
  app.use("/auth/keys/*", auth);

  app.openapi(createKeyRoute, async (c) => {
    const actor = c.get("actor");
    const body = c.req.valid("json");

    const id = apiKeyId();
    const token = apiKeyToken();
    const hash = await sha256(token);
    const name = body.name || "";
    const prefix = body.prefix || "";
    const expiresAt = body.expires_in ? Date.now() + body.expires_in * 1000 : null;

    await c.env.DB.prepare(
      "INSERT INTO api_keys (id, actor, token_hash, name, prefix, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
    )
      .bind(id, actor, hash, name, prefix, expiresAt, Date.now())
      .run();

    audit(c, "key.create", name);
    return c.json({ id, token, name, prefix, expires_at: expiresAt }, 201);
  });

  app.openapi(listKeysRoute, async (c) => {
    const actor = c.get("actor");
    const { results } = await c.env.DB.prepare(
      "SELECT id, name, prefix, expires_at, created_at FROM api_keys WHERE actor = ? ORDER BY created_at DESC",
    )
      .bind(actor)
      .all();

    return c.json({ keys: (results || []) as any }, 200);
  });

  app.openapi(deleteKeyRoute, async (c) => {
    const actor = c.get("actor");
    const { id } = c.req.valid("param");

    const key = await c.env.DB.prepare(
      "SELECT 1 FROM api_keys WHERE id = ? AND actor = ?",
    )
      .bind(id, actor)
      .first();

    if (!key) return c.json({ error: "not_found", message: "API key not found" }, 404);

    await c.env.DB.prepare("DELETE FROM api_keys WHERE id = ?").bind(id).run();
    audit(c, "key.delete", id);
    return c.json({ deleted: true }, 200);
  });
}
