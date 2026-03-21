import type { OpenAPIHono } from "@hono/zod-openapi";
import type { StorageEngine } from "./storage/engine";
import type { StorageDO } from "./storage/do_driver";

export interface Env {
  DB: D1Database;
  BUCKET: R2Bucket;
  SIGNING_KEY: string;
  BENCHMARK_KEY?: string;
  CLI_UPLOAD_KEY?: string;
  RESEND_API_KEY?: string;
  R2_ENDPOINT?: string; // https://<ACCOUNT_ID>.r2.cloudflarestorage.com
  R2_ACCESS_KEY_ID?: string;
  R2_SECRET_ACCESS_KEY?: string;
  R2_BUCKET_NAME?: string; // defaults to "storage-files"
  STORAGE_DO?: DurableObjectNamespace<StorageDO>;
  STORAGE_DRIVER?: string; // "d1" (default) | "do" | "hyperdrive" | "neon"
  HYPERDRIVE?: Hyperdrive; // Cloudflare Hyperdrive binding (PostgreSQL proxy)
  POSTGRES_DSN?: string; // Direct PostgreSQL connection string (for Neon driver)
  DEV_MODE?: string; // "1" to return magic link in response (for testing)
}

export interface Variables {
  actor: string;
  prefix: string;
  engine: StorageEngine;
}

export type App = OpenAPIHono<{ Bindings: Env; Variables: Variables }>;
