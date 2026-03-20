/**
 * S3 V4 presigned URL generator for Cloudflare R2.
 * Uses Web Crypto API — no external dependencies.
 */

export interface PresignOpts {
  method: "GET" | "PUT" | "POST" | "HEAD" | "DELETE";
  key: string; // R2 object key (e.g. "actor/path/file.txt")
  bucket: string;
  endpoint: string; // https://<ACCOUNT_ID>.r2.cloudflarestorage.com
  accessKeyId: string;
  secretAccessKey: string;
  expiresIn?: number; // seconds, default 3600
  contentType?: string; // for PUT — included in signed headers if provided
  queryParams?: Record<string, string>; // extra query params (e.g. partNumber, uploadId)
}

const REGION = "auto"; // R2 always uses "auto"

/** Generate a presigned URL for direct R2 access. */
export async function presignUrl(opts: PresignOpts): Promise<string> {
  const expiresIn = opts.expiresIn ?? 3600;
  const now = new Date();
  const amzDate = amzTimestamp(now);
  const dateStamp = amzDate.slice(0, 8);
  const host = new URL(opts.endpoint).host;
  const scope = `${dateStamp}/${REGION}/s3/aws4_request`;
  const credential = `${opts.accessKeyId}/${scope}`;

  // URI-encode each segment of the key (but not the slashes)
  const canonicalUri = "/" + opts.bucket + "/" + uriEncodePath(opts.key);

  // Determine signed headers
  const signedHeaderNames = ["host"];
  let canonicalHeaders = `host:${host}\n`;
  if (opts.method === "PUT" && opts.contentType) {
    signedHeaderNames.push("content-type");
    // Must be alphabetical
    signedHeaderNames.sort();
    canonicalHeaders = "";
    for (const h of signedHeaderNames) {
      canonicalHeaders += h === "host" ? `host:${host}\n` : `content-type:${opts.contentType}\n`;
    }
  }
  const signedHeaders = signedHeaderNames.join(";");

  // Build query string (must be sorted by param name)
  const params: [string, string][] = [
    ["X-Amz-Algorithm", "AWS4-HMAC-SHA256"],
    ["X-Amz-Credential", credential],
    ["X-Amz-Date", amzDate],
    ["X-Amz-Expires", String(expiresIn)],
    ["X-Amz-SignedHeaders", signedHeaders],
  ];
  // Include extra query params (e.g. partNumber, uploadId for multipart)
  if (opts.queryParams) {
    for (const [k, v] of Object.entries(opts.queryParams)) {
      params.push([k, v]);
    }
  }
  params.sort((a, b) => a[0].localeCompare(b[0]));
  const canonicalQueryString = params.map(([k, v]) => `${rfc3986(k)}=${rfc3986(v)}`).join("&");

  // Canonical request
  const canonicalRequest = [
    opts.method,
    canonicalUri,
    canonicalQueryString,
    canonicalHeaders,
    signedHeaders,
    "UNSIGNED-PAYLOAD",
  ].join("\n");

  // String to sign
  const stringToSign = [
    "AWS4-HMAC-SHA256",
    amzDate,
    scope,
    await sha256Hex(canonicalRequest),
  ].join("\n");

  // Signing key
  const signingKey = await deriveKey(opts.secretAccessKey, dateStamp);

  // Signature
  const signature = await hmacHex(signingKey, stringToSign);

  return `${opts.endpoint}/${opts.bucket}/${uriEncodePath(opts.key)}?${canonicalQueryString}&X-Amz-Signature=${signature}`;
}

// ── Helpers ───────────────────────────────────────────────────────────

function amzTimestamp(d: Date): string {
  return d.toISOString().replace(/[:-]|\.\d{3}/g, "");
}

/** URI-encode each path segment but preserve slashes. */
function uriEncodePath(path: string): string {
  return path.split("/").map((s) => rfc3986(s)).join("/");
}

/** RFC 3986 URI encoding (stricter than encodeURIComponent). */
function rfc3986(s: string): string {
  return encodeURIComponent(s).replace(
    /[!'()*]/g,
    (c) => "%" + c.charCodeAt(0).toString(16).toUpperCase(),
  );
}

async function sha256Hex(data: string): Promise<string> {
  const hash = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(data));
  return hex(hash);
}

async function hmac(key: ArrayBuffer, data: string): Promise<ArrayBuffer> {
  const k = await crypto.subtle.importKey("raw", key, { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  return crypto.subtle.sign("HMAC", k, new TextEncoder().encode(data));
}

async function hmacHex(key: ArrayBuffer, data: string): Promise<string> {
  return hex(await hmac(key, data));
}

async function deriveKey(secret: string, dateStamp: string): Promise<ArrayBuffer> {
  let key = await hmac(new TextEncoder().encode("AWS4" + secret).buffer as ArrayBuffer, dateStamp);
  key = await hmac(key, REGION);
  key = await hmac(key, "s3");
  key = await hmac(key, "aws4_request");
  return key;
}

function hex(buf: ArrayBuffer): string {
  return Array.from(new Uint8Array(buf), (b) => b.toString(16).padStart(2, "0")).join("");
}
