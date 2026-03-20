import { AwsClient } from "aws4fetch";
import type { Context } from "hono";
import type { Env, Variables } from "../types";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const MAX_EXPIRES = 86400;

export function getR2Client(c: AppContext): AwsClient | null {
  const { R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY } = c.env;
  if (!R2_ACCESS_KEY_ID || !R2_SECRET_ACCESS_KEY) return null;
  return new AwsClient({
    accessKeyId: R2_ACCESS_KEY_ID,
    secretAccessKey: R2_SECRET_ACCESS_KEY,
    service: "s3",
    region: "auto",
  });
}

export function getS3Url(c: AppContext, key: string): string {
  const accountId = c.env.CF_ACCOUNT_ID;
  const bucket = c.env.R2_BUCKET_NAME || "storage-files";
  return `https://${accountId}.r2.cloudflarestorage.com/${bucket}/${key}`;
}

/** Generate a presigned GET URL. Returns null if R2 credentials are not configured. */
export async function presignGet(c: AppContext, r2Key: string, expiresIn = 3600): Promise<string | null> {
  const r2 = getR2Client(c);
  if (!r2) return null;
  const secs = Math.min(Math.max(expiresIn, 1), MAX_EXPIRES);
  const url = new URL(getS3Url(c, r2Key));
  url.searchParams.set("X-Amz-Expires", secs.toString());
  const signed = await r2.sign(new Request(url, { method: "GET" }), { aws: { signQuery: true } });
  return signed.url;
}

/** Generate a presigned PUT URL. Returns null if R2 credentials are not configured. */
export async function presignPut(c: AppContext, r2Key: string, expiresIn = 3600): Promise<string | null> {
  const r2 = getR2Client(c);
  if (!r2) return null;
  const secs = Math.min(Math.max(expiresIn, 1), MAX_EXPIRES);
  const url = new URL(getS3Url(c, r2Key));
  url.searchParams.set("X-Amz-Expires", secs.toString());
  const signed = await r2.sign(new Request(url, { method: "PUT" }), { aws: { signQuery: true } });
  return signed.url;
}
