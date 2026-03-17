import type { Context } from "hono";
import type { Env, Variables, RegisterActorRequest } from "./types";
import { isValidActor, prefixMatchesType } from "./actor";
import { importEd25519PublicKey } from "./crypto";
import { errorResponse } from "./error";

export async function registerActor(c: Context<{ Bindings: Env; Variables: Variables }>) {
  let body: RegisterActorRequest;
  try {
    body = await c.req.json<RegisterActorRequest>();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.actor || typeof body.actor !== "string") {
    return errorResponse(c, "invalid_request", "actor is required");
  }

  if (!isValidActor(body.actor)) {
    return errorResponse(c, "invalid_request", "Invalid actor format (use u/<name> or a/<name>, max 64 chars)");
  }

  if (!body.type || (body.type !== "human" && body.type !== "agent")) {
    return errorResponse(c, "invalid_request", "type must be 'human' or 'agent'");
  }

  if (!prefixMatchesType(body.actor, body.type)) {
    return errorResponse(c, "invalid_request", "Actor prefix must match type (u/ for human, a/ for agent)");
  }

  if (!body.public_key || typeof body.public_key !== "string") {
    return errorResponse(c, "invalid_request", "public_key is required");
  }

  try {
    await importEd25519PublicKey(body.public_key);
  } catch {
    return errorResponse(c, "invalid_request", "Invalid public key format (expected base64url Ed25519 public key)");
  }

  const now = Date.now();

  try {
    await c.env.DB.prepare(
      "INSERT INTO actors (actor, type, public_key, created_at) VALUES (?, ?, ?, ?)"
    ).bind(body.actor, body.type, body.public_key, now).run();
  } catch (e: unknown) {
    if (e instanceof Error && e.message.includes("UNIQUE")) {
      // Idempotent: same actor + same key = success
      const existing = await c.env.DB.prepare("SELECT public_key FROM actors WHERE actor = ?")
        .bind(body.actor).first<{ public_key: string }>();
      if (existing && existing.public_key === body.public_key) {
        return c.json({ actor: body.actor, created: false }, 200);
      }
      return errorResponse(c, "conflict", "Actor name already taken");
    }
    throw e;
  }

  return c.json({ actor: body.actor, created: true }, 201);
}
