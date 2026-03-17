import type { Context } from "hono";
import type { Env, Variables, RegisterRequest } from "./types";
import { isValidActor } from "./actor";
import { sha256hex, generateRecoveryCode, importEd25519PublicKey } from "./crypto";

export async function registerActor(c: Context<{ Bindings: Env; Variables: Variables }>) {
  let body: RegisterRequest;
  try {
    body = await c.req.json<RegisterRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.actor || typeof body.actor !== "string") {
    return c.json({ error: "actor is required" }, 400);
  }

  if (!isValidActor(body.actor)) {
    return c.json({ error: "Invalid actor format (use u/<name> or a/<name>, max 64 chars)" }, 400);
  }

  if (!body.public_key || typeof body.public_key !== "string") {
    return c.json({ error: "public_key is required" }, 400);
  }

  // Validate public key format by attempting import
  try {
    await importEd25519PublicKey(body.public_key);
  } catch {
    return c.json({ error: "Invalid public key format (expected base64url Ed25519 public key)" }, 400);
  }

  // Rate limit: 5 registrations per IP per hour
  const ip = c.req.header("CF-Connecting-IP") || "unknown";
  const ipHash = await sha256hex(ip);
  const oneHourAgo = Date.now() - 3600_000;

  const rateCheck = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM actors WHERE created_ip_hash = ? AND created_at > ?"
  ).bind(ipHash, oneHourAgo).first<{ count: number }>();

  if ((rateCheck?.count ?? 0) >= 20) {
    return c.json({ error: "Rate limit exceeded (max 20 registrations per hour)" }, 429);
  }

  // Generate recovery code
  const recoveryCode = generateRecoveryCode();
  const recoveryHash = await sha256hex(recoveryCode);
  const now = Date.now();

  // Use INSERT and catch UNIQUE constraint violation for race-safe 409
  try {
    await c.env.DB.prepare(
      "INSERT INTO actors (actor, public_key, recovery_hash, created_at, created_ip_hash) VALUES (?, ?, ?, ?, ?)"
    ).bind(body.actor, body.public_key, recoveryHash, now, ipHash).run();
  } catch (e: unknown) {
    if (e instanceof Error && e.message.includes("UNIQUE")) {
      return c.json({ error: "Actor name already taken" }, 409);
    }
    throw e;
  }

  return c.json({ actor: body.actor, recovery_code: recoveryCode }, 201);
}
