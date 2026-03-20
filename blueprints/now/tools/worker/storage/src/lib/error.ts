import type { Context } from "hono";

const STATUS: Record<string, number> = {
  bad_request: 400,
  unauthorized: 401,
  forbidden: 403,
  not_found: 404,
  conflict: 409,
  too_large: 413,
  internal: 500,
};

export function err(c: Context, code: string, message: string) {
  return c.json({ error: code, message }, (STATUS[code] || 500) as any);
}
