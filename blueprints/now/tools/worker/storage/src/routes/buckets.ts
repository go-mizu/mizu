import type { Context } from "hono";
import type { Env, Variables, BucketRow } from "../types";
import { bucketId } from "../lib/id";
import { errorResponse } from "../lib/error";
import { requireScope } from "../middleware/authorize";
import { audit } from "../lib/audit";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// POST /bucket — create a new bucket
export async function createBucket(c: AppContext) {
  const scopeErr = requireScope(c, "bucket:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const body = await c.req.json<{
    name: string;
    public?: boolean;
    file_size_limit?: number;
    allowed_mime_types?: string[];
  }>();

  if (!body.name || typeof body.name !== "string") {
    return errorResponse(c, "invalid_request", "Bucket name is required");
  }

  const name = body.name.trim().toLowerCase();
  if (!/^[a-z0-9][a-z0-9._-]{1,62}$/.test(name)) {
    return errorResponse(c, "invalid_request", "Bucket name must be 2-63 chars, lowercase alphanumeric with . - _");
  }

  const existing = await c.env.DB
    .prepare("SELECT 1 FROM buckets WHERE owner = ? AND name = ?")
    .bind(actor, name)
    .first();

  if (existing) {
    return errorResponse(c, "conflict", "Bucket already exists");
  }

  const id = bucketId();
  const now = Date.now();
  const isPublic = body.public ? 1 : 0;
  const mimeTypes = body.allowed_mime_types ? JSON.stringify(body.allowed_mime_types) : null;

  await c.env.DB
    .prepare(
      "INSERT INTO buckets (id, owner, name, public, file_size_limit, allowed_mime_types, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
    )
    .bind(id, actor, name, isPublic, body.file_size_limit || null, mimeTypes, now, now)
    .run();

  audit(c, "bucket.create", name);

  return c.json({
    id,
    name,
    owner: actor,
    public: !!body.public,
    file_size_limit: body.file_size_limit || null,
    allowed_mime_types: body.allowed_mime_types || null,
    created_at: now,
  }, 201);
}

// GET /bucket — list all buckets
export async function listBuckets(c: AppContext) {
  const scopeErr = requireScope(c, "bucket:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { results } = await c.env.DB
    .prepare("SELECT * FROM buckets WHERE owner = ? ORDER BY name")
    .bind(actor)
    .all<BucketRow>();

  return c.json(
    (results || []).map((b) => ({
      id: b.id,
      name: b.name,
      public: !!b.public,
      file_size_limit: b.file_size_limit,
      allowed_mime_types: b.allowed_mime_types ? JSON.parse(b.allowed_mime_types) : null,
      created_at: b.created_at,
      updated_at: b.updated_at,
    })),
  );
}

// GET /bucket/:id — get bucket details
export async function getBucket(c: AppContext) {
  const scopeErr = requireScope(c, "bucket:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const bucket = await c.env.DB
    .prepare("SELECT * FROM buckets WHERE id = ? AND owner = ?")
    .bind(id, actor)
    .first<BucketRow>();

  if (!bucket) {
    return errorResponse(c, "not_found", "Bucket not found");
  }

  // Get object count and total size
  const stats = await c.env.DB
    .prepare("SELECT COUNT(*) as count, COALESCE(SUM(size), 0) as total_size FROM objects WHERE bucket_id = ?")
    .bind(id)
    .first<{ count: number; total_size: number }>();

  return c.json({
    id: bucket.id,
    name: bucket.name,
    owner: bucket.owner,
    public: !!bucket.public,
    file_size_limit: bucket.file_size_limit,
    allowed_mime_types: bucket.allowed_mime_types ? JSON.parse(bucket.allowed_mime_types) : null,
    object_count: stats?.count || 0,
    total_size: stats?.total_size || 0,
    created_at: bucket.created_at,
    updated_at: bucket.updated_at,
  });
}

// PATCH /bucket/:id — update bucket config
export async function updateBucket(c: AppContext) {
  const scopeErr = requireScope(c, "bucket:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const bucket = await c.env.DB
    .prepare("SELECT * FROM buckets WHERE id = ? AND owner = ?")
    .bind(id, actor)
    .first<BucketRow>();

  if (!bucket) {
    return errorResponse(c, "not_found", "Bucket not found");
  }

  const body = await c.req.json<{
    public?: boolean;
    file_size_limit?: number | null;
    allowed_mime_types?: string[] | null;
  }>();

  const now = Date.now();
  const isPublic = body.public !== undefined ? (body.public ? 1 : 0) : bucket.public;
  const sizeLimit = body.file_size_limit !== undefined ? body.file_size_limit : bucket.file_size_limit;
  const mimeTypes = body.allowed_mime_types !== undefined
    ? (body.allowed_mime_types ? JSON.stringify(body.allowed_mime_types) : null)
    : bucket.allowed_mime_types;

  await c.env.DB
    .prepare("UPDATE buckets SET public = ?, file_size_limit = ?, allowed_mime_types = ?, updated_at = ? WHERE id = ?")
    .bind(isPublic, sizeLimit, mimeTypes, now, id)
    .run();

  audit(c, "bucket.update", bucket.name);

  return c.json({ updated: true });
}

// DELETE /bucket/:id — delete bucket (must be empty)
export async function deleteBucket(c: AppContext) {
  const scopeErr = requireScope(c, "bucket:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const bucket = await c.env.DB
    .prepare("SELECT * FROM buckets WHERE id = ? AND owner = ?")
    .bind(id, actor)
    .first<BucketRow>();

  if (!bucket) {
    return errorResponse(c, "not_found", "Bucket not found");
  }

  const count = await c.env.DB
    .prepare("SELECT COUNT(*) as count FROM objects WHERE bucket_id = ?")
    .bind(id)
    .first<{ count: number }>();

  if (count && count.count > 0) {
    return errorResponse(c, "invalid_request", "Bucket is not empty — use POST /bucket/:id/empty first");
  }

  await c.env.DB.prepare("DELETE FROM signed_urls WHERE bucket_id = ?").bind(id).run();
  await c.env.DB.prepare("DELETE FROM buckets WHERE id = ?").bind(id).run();

  audit(c, "bucket.delete", bucket.name);

  return c.json({ deleted: true });
}

// POST /bucket/:id/empty — remove all objects from bucket
export async function emptyBucket(c: AppContext) {
  const scopeErr = requireScope(c, "bucket:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const bucket = await c.env.DB
    .prepare("SELECT * FROM buckets WHERE id = ? AND owner = ?")
    .bind(id, actor)
    .first<BucketRow>();

  if (!bucket) {
    return errorResponse(c, "not_found", "Bucket not found");
  }

  // Get all R2 keys to delete
  const { results } = await c.env.DB
    .prepare("SELECT r2_key FROM objects WHERE bucket_id = ?")
    .bind(id)
    .all<{ r2_key: string }>();

  if (results && results.length > 0) {
    const keys = results.map((r) => r.r2_key).filter(Boolean);
    // R2 delete supports batch of up to 1000
    for (let i = 0; i < keys.length; i += 1000) {
      const batch = keys.slice(i, i + 1000);
      await c.env.BUCKET.delete(batch);
    }
  }

  await c.env.DB.prepare("DELETE FROM signed_urls WHERE bucket_id = ?").bind(id).run();
  await c.env.DB.prepare("DELETE FROM objects WHERE bucket_id = ?").bind(id).run();

  audit(c, "bucket.empty", bucket.name);

  return c.json({ emptied: true });
}

/** Resolve a bucket by name for the given owner. Shared helper for object routes. */
export async function resolveBucket(
  db: D1Database,
  owner: string,
  name: string,
): Promise<BucketRow | null> {
  return db
    .prepare("SELECT * FROM buckets WHERE owner = ? AND name = ?")
    .bind(owner, name)
    .first<BucketRow>();
}

/** Resolve a bucket by ID. */
export async function resolveBucketById(
  db: D1Database,
  id: string,
): Promise<BucketRow | null> {
  return db.prepare("SELECT * FROM buckets WHERE id = ?").bind(id).first<BucketRow>();
}

/** Get or create a default bucket for the given owner. Used by legacy drive routes. */
export async function ensureDefaultBucket(db: D1Database, owner: string): Promise<string> {
  const existing = await db
    .prepare("SELECT id FROM buckets WHERE owner = ? AND name = 'default'")
    .bind(owner)
    .first<{ id: string }>();
  if (existing) return existing.id;

  const id = bucketId();
  const now = Date.now();
  await db
    .prepare("INSERT INTO buckets (id, owner, name, public, created_at, updated_at) VALUES (?, ?, 'default', 0, ?, ?)")
    .bind(id, owner, now, now)
    .run();
  return id;
}
