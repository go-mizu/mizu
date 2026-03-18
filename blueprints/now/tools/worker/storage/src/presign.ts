import { AwsClient } from "aws4fetch";
import type { Context } from "hono";
import type { Env, Variables, ObjectRow } from "./types";
import { objectId } from "./id";
import { mimeFromName } from "./mime";
import { errorResponse } from "./error";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const DEFAULT_EXPIRES = 3600; // 1 hour
const MAX_EXPIRES = 86400; // 24 hours

function getR2Client(c: AppContext): AwsClient | null {
  const { R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY } = c.env;
  if (!R2_ACCESS_KEY_ID || !R2_SECRET_ACCESS_KEY) return null;
  return new AwsClient({
    accessKeyId: R2_ACCESS_KEY_ID,
    secretAccessKey: R2_SECRET_ACCESS_KEY,
    service: "s3",
    region: "auto",
  });
}

function getS3Url(c: AppContext, key: string): string {
  const accountId = c.env.CF_ACCOUNT_ID;
  const bucket = c.env.R2_BUCKET_NAME || "storage-files";
  return `https://${accountId}.r2.cloudflarestorage.com/${bucket}/${key}`;
}

/**
 * Ensure all parent folders exist in D1 for a given path.
 */
async function ensureParentFolders(db: D1Database, owner: string, filePath: string) {
  const parts = filePath.split("/");
  parts.pop();
  let current = "";
  for (const part of parts) {
    current = current ? `${current}/${part}` : part;
    const folderPath = current + "/";
    const existing = await db
      .prepare("SELECT 1 FROM objects WHERE owner = ? AND path = ?")
      .bind(owner, folderPath)
      .first();
    if (!existing) {
      const now = Date.now();
      await db
        .prepare(
          "INSERT OR IGNORE INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, 1, '', 0, '', ?, ?)",
        )
        .bind(objectId(), owner, folderPath, part, now, now)
        .run();
    }
  }
}

// POST /presign/upload
export async function presignUpload(c: AppContext) {
  const r2 = getR2Client(c);
  if (!r2) {
    return errorResponse(c, "not_configured", "Presigned URLs are not configured. Use PUT /files/* instead.");
  }

  let body: { path?: string; content_type?: string; expires?: number };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  const filePath = body.path?.replace(/^\/+/, "");
  if (!filePath || filePath.endsWith("/")) {
    return errorResponse(c, "invalid_request", "File path is required (not a folder)");
  }

  const actor = c.get("actor");
  const contentType = body.content_type || mimeFromName(filePath.split("/").pop() || filePath);
  const expires = Math.min(Math.max(body.expires || DEFAULT_EXPIRES, 1), MAX_EXPIRES);
  const r2Key = `${actor}/${filePath}`;

  const url = new URL(getS3Url(c, r2Key));
  url.searchParams.set("X-Amz-Expires", expires.toString());

  const signed = await r2.sign(
    new Request(url, { method: "PUT" }),
    { aws: { signQuery: true } },
  );

  return c.json({
    upload_url: signed.url,
    r2_key: r2Key,
    path: filePath,
    content_type: contentType,
    expires_in: expires,
    method: "PUT",
    headers: {
      "Content-Type": contentType,
    },
  });
}

// POST /presign/download
export async function presignDownload(c: AppContext) {
  const r2 = getR2Client(c);
  if (!r2) {
    return errorResponse(c, "not_configured", "Presigned URLs are not configured. Use GET /files/* instead.");
  }

  let body: { path?: string; expires?: number };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  const filePath = body.path?.replace(/^\/+/, "");
  if (!filePath) {
    return errorResponse(c, "invalid_request", "File path is required");
  }

  const actor = c.get("actor");
  const expires = Math.min(Math.max(body.expires || DEFAULT_EXPIRES, 1), MAX_EXPIRES);

  // Verify file exists in D1
  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND is_folder = 0",
  )
    .bind(actor, filePath)
    .first<ObjectRow>();

  if (!obj) {
    return errorResponse(c, "not_found", "File not found");
  }

  const url = new URL(getS3Url(c, obj.r2_key));
  url.searchParams.set("X-Amz-Expires", expires.toString());

  const signed = await r2.sign(
    new Request(url, { method: "GET" }),
    { aws: { signQuery: true } },
  );

  return c.json({
    download_url: signed.url,
    path: filePath,
    name: obj.name,
    content_type: obj.content_type,
    size: obj.size,
    expires_in: expires,
  });
}

// POST /presign/complete
export async function presignComplete(c: AppContext) {
  let body: { path?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  const filePath = body.path?.replace(/^\/+/, "");
  if (!filePath || filePath.endsWith("/")) {
    return errorResponse(c, "invalid_request", "File path is required (not a folder)");
  }

  const actor = c.get("actor");
  const r2Key = `${actor}/${filePath}`;

  // Verify the object actually exists in R2
  const r2Head = await c.env.BUCKET.head(r2Key);
  if (!r2Head) {
    return errorResponse(c, "not_found", "File not found in storage. Upload may have failed.");
  }

  const name = filePath.split("/").pop() || filePath;
  const contentType = r2Head.httpMetadata?.contentType || mimeFromName(name);
  const size = r2Head.size;

  // Ensure parent folders exist
  await ensureParentFolders(c.env.DB, actor, filePath);

  // Upsert metadata in D1
  const now = Date.now();
  const existing = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ?",
  )
    .bind(actor, filePath)
    .first<{ id: string }>();

  let id: string;
  if (existing) {
    id = existing.id;
    await c.env.DB.prepare(
      "UPDATE objects SET content_type = ?, size = ?, r2_key = ?, updated_at = ? WHERE id = ?",
    )
      .bind(contentType, size, r2Key, now, id)
      .run();
  } else {
    id = objectId();
    await c.env.DB.prepare(
      "INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?, ?)",
    )
      .bind(id, actor, filePath, name, contentType, size, r2Key, now, now)
      .run();
  }

  return c.json({
    id,
    path: filePath,
    name,
    content_type: contentType,
    size,
    created_at: now,
  });
}
