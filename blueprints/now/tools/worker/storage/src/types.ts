import type { OpenAPIHono } from "@hono/zod-openapi";

export interface Env {
  DB: D1Database;
  BUCKET: R2Bucket;
  SIGNING_KEY: string;
  CLI_UPLOAD_KEY?: string;
  RESEND_API_KEY?: string;
}

export interface Variables {
  actor: string;
  prefix: string;
}

export type App = OpenAPIHono<{ Bindings: Env; Variables: Variables }>;
