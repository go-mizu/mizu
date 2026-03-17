import type { Context } from "hono";
import type { Env, Variables, RotateKeyRequest, RotateRecoveryRequest, DeleteActorRequest } from "./types";
import { sha256hex, generateRecoveryCode, importEd25519PublicKey } from "./crypto";
import { invalidateKeyCache } from "./auth";

async function verifyRecoveryCode(
  db: D1Database,
  actor: string,
  code: string
): Promise<{ valid: boolean; found: boolean }> {
  const row = await db.prepare("SELECT recovery_hash FROM actors WHERE actor = ?")
    .bind(actor)
    .first<{ recovery_hash: string }>();
  if (!row) return { valid: false, found: false };
  const hash = await sha256hex(code);
  return { valid: hash === row.recovery_hash, found: true };
}

// Rate limit: 5 failed recovery attempts per actor per hour
const ROTATION_FAIL_LIMIT = 5;
const ROTATION_WINDOW_MS = 3600_000;
const rotationFailures = new Map<string, { count: number; firstAt: number }>();

function checkRotationRateLimit(actor: string): boolean {
  const entry = rotationFailures.get(actor);
  if (!entry) return true;
  if (Date.now() - entry.firstAt > ROTATION_WINDOW_MS) {
    rotationFailures.delete(actor);
    return true;
  }
  return entry.count < ROTATION_FAIL_LIMIT;
}

function recordRotationFailure(actor: string): void {
  const entry = rotationFailures.get(actor);
  if (!entry || Date.now() - entry.firstAt > ROTATION_WINDOW_MS) {
    rotationFailures.set(actor, { count: 1, firstAt: Date.now() });
  } else {
    entry.count++;
  }
}

function clearRotationFailures(actor: string): void {
  rotationFailures.delete(actor);
}

export async function rotateKey(c: Context<{ Bindings: Env; Variables: Variables }>) {
  let body: RotateKeyRequest;
  try {
    body = await c.req.json<RotateKeyRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.actor || !body.recovery_code || !body.new_public_key) {
    return c.json({ error: "actor, recovery_code, and new_public_key are required" }, 400);
  }

  if (!checkRotationRateLimit(body.actor)) {
    return c.json({ error: "Too many failed attempts, try again later" }, 429);
  }

  // Validate new key format
  try {
    await importEd25519PublicKey(body.new_public_key);
  } catch {
    return c.json({ error: "Invalid public key format" }, 400);
  }

  const { valid, found } = await verifyRecoveryCode(c.env.DB, body.actor, body.recovery_code);
  if (!found) return c.json({ error: "Actor not found" }, 404);
  if (!valid) {
    recordRotationFailure(body.actor);
    return c.json({ error: "Invalid recovery code" }, 401);
  }

  clearRotationFailures(body.actor);

  await c.env.DB.prepare("UPDATE actors SET public_key = ? WHERE actor = ?")
    .bind(body.new_public_key, body.actor).run();

  invalidateKeyCache(body.actor);

  return c.json({ actor: body.actor });
}

export async function rotateRecovery(c: Context<{ Bindings: Env; Variables: Variables }>) {
  let body: RotateRecoveryRequest;
  try {
    body = await c.req.json<RotateRecoveryRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.actor || !body.recovery_code) {
    return c.json({ error: "actor and recovery_code are required" }, 400);
  }

  if (!checkRotationRateLimit(body.actor)) {
    return c.json({ error: "Too many failed attempts, try again later" }, 429);
  }

  const { valid, found } = await verifyRecoveryCode(c.env.DB, body.actor, body.recovery_code);
  if (!found) return c.json({ error: "Actor not found" }, 404);
  if (!valid) {
    recordRotationFailure(body.actor);
    return c.json({ error: "Invalid recovery code" }, 401);
  }

  clearRotationFailures(body.actor);

  const newCode = generateRecoveryCode();
  const newHash = await sha256hex(newCode);

  await c.env.DB.prepare("UPDATE actors SET recovery_hash = ? WHERE actor = ?")
    .bind(newHash, body.actor).run();

  return c.json({ recovery_code: newCode });
}

export async function deleteActor(c: Context<{ Bindings: Env; Variables: Variables }>) {
  let body: DeleteActorRequest;
  try {
    body = await c.req.json<DeleteActorRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.actor || !body.recovery_code) {
    return c.json({ error: "actor and recovery_code are required" }, 400);
  }

  if (!checkRotationRateLimit(body.actor)) {
    return c.json({ error: "Too many failed attempts, try again later" }, 429);
  }

  const { valid, found } = await verifyRecoveryCode(c.env.DB, body.actor, body.recovery_code);
  if (!found) return c.json({ error: "Actor not found" }, 404);
  if (!valid) {
    recordRotationFailure(body.actor);
    return c.json({ error: "Invalid recovery code" }, 401);
  }

  clearRotationFailures(body.actor);

  await c.env.DB.prepare("DELETE FROM actors WHERE actor = ?")
    .bind(body.actor).run();

  invalidateKeyCache(body.actor);

  return c.json({ deleted: body.actor });
}
