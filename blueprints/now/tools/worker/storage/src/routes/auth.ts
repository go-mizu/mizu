import { createRoute, z } from "@hono/zod-openapi";
import type { App } from "../types";
import { challengeId, nonce, sessionToken } from "../lib/id";
import { audit } from "../lib/audit";
import { errRes } from "../schema";

// ── Route definitions (single source of truth) ─────────────────────

const registerRoute = createRoute({
  method: "post",
  path: "/auth/register",
  tags: ["auth"],
  request: {
    body: {
      content: {
        "application/json": {
          schema: z.object({
            actor: z
              .string()
              .regex(/^[a-zA-Z0-9_-]{1,64}$/)
              .openapi({ example: "alice" }),
            type: z.enum(["human", "agent"]).default("human").optional(),
            public_key: z
              .string()
              .openapi({ description: "Base64 Ed25519 public key" }),
          }),
        },
      },
    },
  },
  responses: {
    201: {
      description: "Created",
      content: {
        "application/json": {
          schema: z.object({
            actor: z.string(),
            type: z.string(),
          }),
        },
      },
    },
    400: errRes("Bad request"),
    409: errRes("Actor already exists"),
  },
});

const challengeRoute = createRoute({
  method: "post",
  path: "/auth/challenge",
  tags: ["auth"],
  request: {
    body: {
      content: {
        "application/json": {
          schema: z.object({
            actor: z.string().openapi({ example: "alice" }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Challenge issued",
      content: {
        "application/json": {
          schema: z.object({
            challenge_id: z.string(),
            nonce: z.string(),
            expires_at: z.number().int(),
          }),
        },
      },
    },
    400: errRes("Bad request"),
    404: errRes("Actor not found"),
  },
});

const verifyRoute = createRoute({
  method: "post",
  path: "/auth/verify",
  tags: ["auth"],
  request: {
    body: {
      content: {
        "application/json": {
          schema: z.object({
            challenge_id: z.string(),
            actor: z.string(),
            signature: z
              .string()
              .openapi({ description: "Base64 Ed25519 signature of the nonce" }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Session token",
      content: {
        "application/json": {
          schema: z.object({
            token: z.string(),
            actor: z.string(),
            expires_at: z.number().int(),
          }),
        },
      },
    },
    400: errRes("Bad request"),
    401: errRes("Invalid signature"),
    404: errRes("Challenge not found"),
  },
});

const logoutRoute = createRoute({
  method: "post",
  path: "/auth/logout",
  tags: ["auth"],
  responses: {
    200: {
      description: "OK",
      content: {
        "application/json": {
          schema: z.object({ ok: z.boolean() }),
        },
      },
    },
  },
});

// ── Handlers ────────────────────────────────────────────────────────

export function register(app: App) {
  app.openapi(registerRoute, async (c) => {
    const { actor: name, type, public_key } = c.req.valid("json");

    const existing = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
      .bind(name)
      .first();
    if (existing) return c.json({ error: "conflict", message: "Actor already exists" }, 409);

    const actorType = type === "agent" ? "agent" : "human";
    await c.env.DB.prepare(
      "INSERT INTO actors (actor, type, public_key, created_at) VALUES (?, ?, ?, ?)",
    )
      .bind(name, actorType, public_key, Date.now())
      .run();

    audit(c, "register", name);
    return c.json({ actor: name, type: actorType }, 201);
  });

  app.openapi(challengeRoute, async (c) => {
    const { actor: name } = c.req.valid("json");

    const row = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
      .bind(name)
      .first();
    if (!row) return c.json({ error: "not_found", message: "Actor not found" }, 404);

    c.executionCtx.waitUntil(
      c.env.DB.prepare("DELETE FROM challenges WHERE expires_at < ?")
        .bind(Date.now())
        .run(),
    );

    const id = challengeId();
    const n = nonce();
    const expiresAt = Date.now() + 300_000;

    await c.env.DB.prepare(
      "INSERT INTO challenges (id, actor, nonce, expires_at) VALUES (?, ?, ?, ?)",
    )
      .bind(id, name, n, expiresAt)
      .run();

    return c.json({ challenge_id: id, nonce: n, expires_at: expiresAt }, 200);
  });

  app.openapi(verifyRoute, async (c) => {
    const { challenge_id, actor: name, signature } = c.req.valid("json");

    const ch = await c.env.DB.prepare(
      "SELECT nonce, expires_at FROM challenges WHERE id = ? AND actor = ?",
    )
      .bind(challenge_id, name)
      .first<{ nonce: string; expires_at: number }>();

    if (!ch) return c.json({ error: "not_found", message: "Challenge not found" }, 404);
    if (Date.now() > ch.expires_at)
      return c.json({ error: "unauthorized", message: "Challenge expired" }, 401);

    const actor = await c.env.DB.prepare(
      "SELECT public_key FROM actors WHERE actor = ?",
    )
      .bind(name)
      .first<{ public_key: string }>();

    if (!actor?.public_key)
      return c.json({ error: "unauthorized", message: "No public key" }, 401);

    const valid = await verifyEd25519(actor.public_key, ch.nonce, signature);
    if (!valid)
      return c.json({ error: "unauthorized", message: "Invalid signature" }, 401);

    await c.env.DB.prepare("DELETE FROM challenges WHERE id = ?")
      .bind(challenge_id)
      .run();

    const token = sessionToken();
    const expiresAt = Date.now() + 30 * 86400000;

    await c.env.DB.prepare(
      "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
    )
      .bind(token, name, expiresAt)
      .run();

    audit(c, "login", name);
    return c.json({ token, actor: name, expires_at: expiresAt }, 200);
  });

  app.openapi(logoutRoute, async (c) => {
    let token: string | undefined;

    const h = c.req.header("Authorization");
    if (h?.startsWith("Bearer ")) token = h.slice(7).trim();

    if (!token) {
      const cookie = c.req.header("Cookie");
      if (cookie) {
        const m = cookie.match(/(?:^|;\s*)session=([^;]+)/);
        if (m) token = m[1];
      }
    }

    if (token) {
      await c.env.DB.prepare("DELETE FROM sessions WHERE token = ?").bind(token).run();
    }

    return c.json({ ok: true }, 200);
  });
}

// ── Ed25519 verification ────────────────────────────────────────────

async function verifyEd25519(
  publicKeyB64: string,
  message: string,
  signatureB64: string,
): Promise<boolean> {
  try {
    const decode = (s: string) =>
      Uint8Array.from(atob(s.replace(/-/g, "+").replace(/_/g, "/")), (c) =>
        c.charCodeAt(0),
      );

    const key = await crypto.subtle.importKey(
      "raw",
      decode(publicKeyB64),
      { name: "Ed25519" },
      false,
      ["verify"],
    );

    return await crypto.subtle.verify(
      "Ed25519",
      key,
      decode(signatureB64),
      new TextEncoder().encode(message),
    );
  } catch {
    return false;
  }
}
