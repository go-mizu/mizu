import type { Context } from "hono";
import type { Env, Variables, BucketRow } from "../types";
import { objectId } from "../lib/id";
import { mimeFromName } from "../lib/mime";
import { errorResponse } from "../lib/error";
import { validatePath } from "../lib/path";
import { requireScope } from "../middleware/authorize";
import { audit } from "../lib/audit";
import { resolveBucket } from "./buckets";
import {
  TUS_VERSION,
  TUS_EXTENSIONS,
  TUS_EXPIRY_MS,
  tusUploadId,
  parseUploadMetadata,
  formatExpires,
} from "../lib/tus";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const DEFAULT_MAX_SIZE = 5 * 1024 * 1024 * 1024; // 5 GB

interface TusUploadRow {
  id: string;
  owner: string;
  bucket_id: string;
  path: string;
  upload_length: number;
  upload_offset: number;
  part_count: number;
  content_type: string;
  metadata: string;
  upsert: number;
  expires_at: number;
  created_at: number;
}

// ── Helpers ─────────────────────────────────────────────────────────

function tusHeaders(extra?: Record<string, string>): Record<string, string> {
  return { "Tus-Resumable": TUS_VERSION, ...extra };
}

function tusError(c: AppContext, status: number, message: string): Response {
  return c.json({ error: { code: "tus_error", message } }, status as any, tusHeaders());
}

function checkVersion(c: AppContext): Response | null {
  const v = c.req.header("Tus-Resumable");
  if (v !== TUS_VERSION) {
    return c.json(
      { error: { code: "tus_error", message: `Unsupported TUS version: ${v}. Required: ${TUS_VERSION}` } },
      412 as any,
      tusHeaders(),
    );
  }
  return null;
}

/** R2 key for a TUS part chunk */
function partKey(uploadId: string, partNum: number): string {
  return `__tus/${uploadId}/${partNum}`;
}

/**
 * Return the base URL for TUS uploads, mirroring whichever path prefix the
 * client used (/upload/resumable or /storage/v1/upload/resumable).
 */
function tusBaseUrl(c: AppContext): string {
  const url = new URL(c.req.url);
  // Strip any trailing slash or upload ID from the path
  const pathname = url.pathname.replace(/\/$/, "");
  // The POST handler is always at the root of the TUS base path
  return `${url.origin}${pathname}`;
}

/** Probabilistic cleanup of expired uploads (~1% chance per request) */
function maybeCleanup(c: AppContext) {
  if (Math.random() > 0.01) return;
  c.executionCtx.waitUntil(
    (async () => {
      try {
        const now = Date.now();
        const { results } = await c.env.DB
          .prepare("SELECT id, part_count FROM tus_uploads WHERE expires_at < ? LIMIT 10")
          .bind(now)
          .all<{ id: string; part_count: number }>();

        await Promise.all((results || []).map(async (row) => {
          await Promise.all(
            Array.from({ length: row.part_count }, (_, i) => c.env.BUCKET.delete(partKey(row.id, i))),
          );
          await c.env.DB.prepare("DELETE FROM tus_uploads WHERE id = ?").bind(row.id).run();
        }));
      } catch {
        // Cleanup should never break requests
      }
    })(),
  );
}

// ── OPTIONS /upload/resumable ───────────────────────────────────────

export async function tusOptions(c: AppContext) {
  return new Response(null, {
    status: 204,
    headers: {
      "Tus-Resumable": TUS_VERSION,
      "Tus-Version": TUS_VERSION,
      "Tus-Extension": TUS_EXTENSIONS,
      "Tus-Max-Size": DEFAULT_MAX_SIZE.toString(),
    },
  });
}

// ── POST /upload/resumable ──────────────────────────────────────────

export async function tusCreate(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const versionErr = checkVersion(c);
  if (versionErr) return versionErr;

  const actor = c.get("actor");

  // Parse Upload-Length
  const lengthHeader = c.req.header("Upload-Length");
  if (!lengthHeader) {
    return tusError(c, 400, "Upload-Length header is required");
  }
  const uploadLength = parseInt(lengthHeader, 10);
  if (isNaN(uploadLength) || uploadLength < 0) {
    return tusError(c, 400, "Upload-Length must be a non-negative integer");
  }

  // Parse Upload-Metadata
  const metadataHeader = c.req.header("Upload-Metadata") || "";
  const meta = parseUploadMetadata(metadataHeader);

  const bucketName = meta.bucketName;
  const objectName = meta.objectName;
  if (!bucketName) return tusError(c, 400, "Upload-Metadata must include bucketName");
  if (!objectName) return tusError(c, 400, "Upload-Metadata must include objectName");

  const pathErr = validatePath(objectName);
  if (pathErr) return tusError(c, 400, pathErr);

  // Resolve bucket
  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return tusError(c, 404, "Bucket not found");

  // Check bucket size limit
  const maxSize = bucket.file_size_limit || DEFAULT_MAX_SIZE;
  if (uploadLength > maxSize) {
    return c.json(
      { error: { code: "tus_error", message: `Upload exceeds size limit of ${maxSize} bytes` } },
      413 as any,
      tusHeaders(),
    );
  }

  // Check MIME type restrictions
  const contentType = meta.contentType || mimeFromName(objectName.split("/").pop() || objectName);
  if (bucket.allowed_mime_types) {
    const allowed: string[] = JSON.parse(bucket.allowed_mime_types);
    if (allowed.length > 0 && !allowed.includes(contentType)) {
      return tusError(c, 400, `MIME type ${contentType} not allowed in this bucket`);
    }
  }

  // Check for existing object (unless upsert)
  const upsert = c.req.header("X-Upsert") === "true";
  if (!upsert) {
    const existing = await c.env.DB
      .prepare("SELECT 1 FROM objects WHERE bucket_id = ? AND path = ?")
      .bind(bucket.id, objectName)
      .first();
    if (existing) {
      return tusError(c, 409, "Object already exists — set X-Upsert: true to overwrite");
    }
  }

  // Create upload record
  const id = tusUploadId();
  const now = Date.now();
  const expiresAt = now + TUS_EXPIRY_MS;
  const customMeta = meta.metadata || "{}";

  await c.env.DB
    .prepare(
      "INSERT INTO tus_uploads (id, owner, bucket_id, path, upload_length, upload_offset, part_count, content_type, metadata, upsert, expires_at, created_at) VALUES (?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?, ?)",
    )
    .bind(id, actor, bucket.id, objectName, uploadLength, contentType, customMeta, upsert ? 1 : 0, expiresAt, now)
    .run();

  let offset = 0;
  let partCount = 0;

  // creation-with-upload: if body is provided, store first chunk
  const ct = c.req.header("Content-Type");
  if (ct === "application/offset+octet-stream") {
    const body = await c.req.arrayBuffer();
    if (body.byteLength > 0) {
      if (body.byteLength > uploadLength) {
        return tusError(c, 413, "Chunk exceeds declared Upload-Length");
      }
      await c.env.BUCKET.put(partKey(id, 0), body);
      offset = body.byteLength;
      partCount = 1;
      await c.env.DB
        .prepare("UPDATE tus_uploads SET upload_offset = ?, part_count = 1 WHERE id = ?")
        .bind(offset, id)
        .run();

      // Check if complete (small file in single POST)
      if (offset === uploadLength) {
        await assembleUpload(c, id, actor, bucket, objectName, contentType, uploadLength, customMeta, upsert, partCount);
        const base = tusBaseUrl(c);
        return c.body(null, 201, tusHeaders({
          Location: `${base}/${id}`,
          "Upload-Offset": offset.toString(),
          "Upload-Expires": formatExpires(expiresAt),
          "Tus-Complete": "1",
        }));
      }
    }
  }

  const base = tusBaseUrl(c);

  maybeCleanup(c);

  return c.body(null, 201, tusHeaders({
    Location: `${base}/${id}`,
    "Upload-Offset": offset.toString(),
    "Upload-Expires": formatExpires(expiresAt),
  }));
}

// ── GET|HEAD /upload/resumable/:id ───────────────────────────────────

export async function tusHead(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const versionErr = checkVersion(c);
  if (versionErr) return versionErr;

  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const upload = await c.env.DB
    .prepare("SELECT * FROM tus_uploads WHERE id = ? AND owner = ?")
    .bind(id, actor)
    .first<TusUploadRow>();

  if (!upload) {
    return new Response(null, { status: 404, headers: tusHeaders() });
  }

  if (upload.expires_at < Date.now()) {
    return new Response(null, {
      status: 410,
      headers: tusHeaders({ "Upload-Offset": upload.upload_offset.toString() }),
    });
  }

  return new Response(null, {
    status: 200,
    headers: tusHeaders({
      "Upload-Offset": upload.upload_offset.toString(),
      "Upload-Length": upload.upload_length.toString(),
      "Upload-Expires": formatExpires(upload.expires_at),
      "Cache-Control": "no-store",
    }),
  });
}

// ── PATCH /upload/resumable/:id ─────────────────────────────────────

export async function tusPatch(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const versionErr = checkVersion(c);
  if (versionErr) return versionErr;

  // Content-Type must be application/offset+octet-stream
  const ct = c.req.header("Content-Type");
  if (ct !== "application/offset+octet-stream") {
    return c.json(
      { error: { code: "tus_error", message: "Content-Type must be application/offset+octet-stream" } },
      415 as any,
      tusHeaders(),
    );
  }

  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const upload = await c.env.DB
    .prepare("SELECT * FROM tus_uploads WHERE id = ? AND owner = ?")
    .bind(id, actor)
    .first<TusUploadRow>();

  if (!upload) return tusError(c, 404, "Upload not found");

  if (upload.expires_at < Date.now()) {
    return c.json(
      { error: { code: "tus_error", message: "Upload has expired" } },
      410 as any,
      tusHeaders(),
    );
  }

  // Validate offset
  const offsetHeader = c.req.header("Upload-Offset");
  if (!offsetHeader) return tusError(c, 400, "Upload-Offset header is required");

  const clientOffset = parseInt(offsetHeader, 10);
  if (isNaN(clientOffset) || clientOffset < 0) {
    return tusError(c, 400, "Upload-Offset must be a non-negative integer");
  }

  if (clientOffset !== upload.upload_offset) {
    return c.json(
      { error: { code: "tus_error", message: `Offset mismatch: server has ${upload.upload_offset}, client sent ${clientOffset}` } },
      409 as any,
      tusHeaders({ "Upload-Offset": upload.upload_offset.toString() }),
    );
  }

  // Read chunk
  const body = await c.req.arrayBuffer();
  if (body.byteLength === 0) {
    return tusError(c, 400, "Empty PATCH body");
  }

  const newOffset = upload.upload_offset + body.byteLength;
  if (newOffset > upload.upload_length) {
    return tusError(c, 413, "Chunk would exceed declared Upload-Length");
  }

  // Store part using D1 part_count as the part number (no R2.list())
  const partNum = upload.part_count;
  const newPartCount = partNum + 1;
  await c.env.BUCKET.put(partKey(id, partNum), body);

  // Update offset and part_count atomically
  await c.env.DB
    .prepare("UPDATE tus_uploads SET upload_offset = ?, part_count = ? WHERE id = ?")
    .bind(newOffset, newPartCount, id)
    .run();

  // Check if upload is complete
  if (newOffset === upload.upload_length) {
    const bucket = await c.env.DB
      .prepare("SELECT * FROM buckets WHERE id = ?")
      .bind(upload.bucket_id)
      .first<BucketRow>();

    if (!bucket) return tusError(c, 404, "Bucket not found");

    await assembleUpload(
      c, id, actor, bucket, upload.path, upload.content_type,
      upload.upload_length, upload.metadata, !!upload.upsert, newPartCount,
    );

    return c.body(null, 204, tusHeaders({
      "Upload-Offset": newOffset.toString(),
      "Tus-Complete": "1",
    }));
  }

  maybeCleanup(c);

  return c.body(null, 204, tusHeaders({
    "Upload-Offset": newOffset.toString(),
    "Upload-Expires": formatExpires(upload.expires_at),
  }));
}

// ── DELETE /upload/resumable/:id ────────────────────────────────────

export async function tusDelete(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const versionErr = checkVersion(c);
  if (versionErr) return versionErr;

  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const upload = await c.env.DB
    .prepare("SELECT * FROM tus_uploads WHERE id = ? AND owner = ?")
    .bind(id, actor)
    .first<TusUploadRow>();

  if (!upload) return tusError(c, 404, "Upload not found");

  // Delete all part objects in parallel (use D1 part_count — no R2.list())
  await Promise.all(
    Array.from({ length: upload.part_count }, (_, i) => c.env.BUCKET.delete(partKey(id, i))),
  );

  // Delete the upload record
  await c.env.DB.prepare("DELETE FROM tus_uploads WHERE id = ?").bind(id).run();

  audit(c, "tus.cancel", `${upload.path}`);

  return c.body(null, 204, tusHeaders());
}

// ── Assembly: concatenate parts into final object ───────────────────

async function assembleUpload(
  c: AppContext,
  uploadId: string,
  owner: string,
  bucket: BucketRow,
  filePath: string,
  contentType: string,
  totalSize: number,
  customMetadata: string,
  upsert: boolean,
  partCount: number,
) {
  const r2Key = `${owner}/${bucket.name}/${filePath}`;

  if (totalSize === 0) {
    // Empty file
    await c.env.BUCKET.put(r2Key, new ArrayBuffer(0), {
      httpMetadata: { contentType },
    });
  } else if (partCount === 1) {
    // Single part — move directly to final key (streaming, no buffer)
    const part = await c.env.BUCKET.get(partKey(uploadId, 0));
    if (part) {
      await c.env.BUCKET.put(r2Key, part.body, {
        httpMetadata: { contentType },
      });
    }
  } else {
    // Multiple parts — fetch all in parallel, then concatenate
    const partKeys = Array.from({ length: partCount }, (_, i) => partKey(uploadId, i));
    const partObjects = await Promise.all(partKeys.map((k) => c.env.BUCKET.get(k)));

    const chunks = await Promise.all(
      partObjects.map((obj) => (obj ? obj.arrayBuffer() : Promise.resolve(new ArrayBuffer(0)))),
    );

    const combined = new Uint8Array(totalSize);
    let pos = 0;
    for (const chunk of chunks) {
      combined.set(new Uint8Array(chunk), pos);
      pos += chunk.byteLength;
    }
    await c.env.BUCKET.put(r2Key, combined.buffer, {
      httpMetadata: { contentType },
    });
  }

  // Upsert object metadata in D1
  const now = Date.now();
  const name = filePath.split("/").pop() || filePath;

  const existing = await c.env.DB
    .prepare("SELECT id FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first<{ id: string }>();

  if (existing && upsert) {
    await c.env.DB
      .prepare("UPDATE objects SET content_type = ?, size = ?, r2_key = ?, metadata = ?, updated_at = ? WHERE id = ?")
      .bind(contentType, totalSize, r2Key, customMetadata, now, existing.id)
      .run();
  } else {
    const id = objectId();
    await c.env.DB
      .prepare(
        "INSERT INTO objects (id, owner, bucket_id, path, name, content_type, size, r2_key, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
      )
      .bind(id, owner, bucket.id, filePath, name, contentType, totalSize, r2Key, customMetadata, now, now)
      .run();
  }

  // Cleanup: delete TUS parts in parallel and upload record
  const partKeys = Array.from({ length: partCount }, (_, i) => partKey(uploadId, i));
  await Promise.all(partKeys.map((k) => c.env.BUCKET.delete(k)));
  await c.env.DB.prepare("DELETE FROM tus_uploads WHERE id = ?").bind(uploadId).run();

  audit(c, "tus.complete", `${bucket.name}/${filePath}`, { size: totalSize });
}
