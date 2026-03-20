import type { OpenAPIHono } from "@hono/zod-openapi";
import type { StorageEngine } from "./storage/engine";

export interface Env {
  DB: D1Database;
  BUCKET: R2Bucket;
  SIGNING_KEY: string;
  CLI_UPLOAD_KEY?: string;
  RESEND_API_KEY?: string;
  R2_ENDPOINT?: string; // https://<ACCOUNT_ID>.r2.cloudflarestorage.com
  R2_ACCESS_KEY_ID?: string;
  R2_SECRET_ACCESS_KEY?: string;
  R2_BUCKET_NAME?: string; // defaults to "storage-files"
}

export interface Variables {
  actor: string;
  prefix: string;
  engine: StorageEngine;
}

export type App = OpenAPIHono<{ Bindings: Env; Variables: Variables }>;
