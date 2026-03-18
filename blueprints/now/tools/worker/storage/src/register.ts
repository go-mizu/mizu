import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { errorResponse } from "./error";

const ACTOR_RE = /^[ua]\/[\w.@-]{1,64}$/;

export async function registerActor(
  c: Context<{ Bindings: Env; Variables: Variables }>,
) {
  let body: { actor?: string; type?: string; public_key?: string; email?: string; bio?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.actor || !ACTOR_RE.test(body.actor)) {
    return errorResponse(c, "invalid_request", "Invalid actor format (u/name or a/name)");
  }

  const expectedType = body.actor.startsWith("u/") ? "human" : "agent";
  if (body.type && body.type !== expectedType) {
    return errorResponse(c, "invalid_request", `Actor prefix doesn't match type`);
  }

  if (expectedType === "agent" && !body.public_key) {
    return errorResponse(c, "invalid_request", "Agents require a public_key");
  }

  const existing = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
    .bind(body.actor)
    .first();
  if (existing) {
    return c.json({ actor: body.actor, created: false });
  }

  const now = Date.now();
  await c.env.DB.prepare(
    "INSERT INTO actors (actor, type, public_key, email, bio, created_at) VALUES (?, ?, ?, ?, ?, ?)",
  )
    .bind(body.actor, expectedType, body.public_key || null, body.email || null, body.bio || "", now)
    .run();

  return c.json({ actor: body.actor, created: true }, 201);
}
