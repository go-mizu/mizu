import type { OpenAPIHono } from "@hono/zod-openapi";
import type { StorageEngine } from "./storage/engine";
import type { StorageDO } from "./storage/do_driver";
import type { StorageDOv2 } from "./storage/do_v2_driver";

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
  STORAGE_DO_V2?: DurableObjectNamespace<StorageDOv2>;
  STORAGE_DRIVER?: string; // "d1" | "d1v2" | "do" | "dov2" | "hyperdrive" | "neon"
  HYPERDRIVE?: Hyperdrive; // Cloudflare Hyperdrive binding (PostgreSQL proxy)
  POSTGRES_DSN?: string; // Direct PostgreSQL connection string (for Neon driver)
  POSTGRES_EC1_DSN?: string; // Neon EU (eu-central-1) connection string
  DEV_MODE?: string; // "1" to return magic link in response (for testing)
}

export interface Variables {
  actor: string;
  prefix: string;
  engine: StorageEngine;
}

export type App = OpenAPIHono<{ Bindings: Env; Variables: Variables }>;
