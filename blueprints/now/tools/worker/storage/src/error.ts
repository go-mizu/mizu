import type { Context } from "hono";

const STATUS_MAP: Record<string, number> = {
  invalid_request: 400,
  unauthorized: 401,
  forbidden: 403,
  not_found: 404,
  conflict: 409,
  too_large: 413,
  internal: 500,
};

export function errorResponse(c: Context, code: string, message: string) {
  const status = STATUS_MAP[code] || 500;
  return c.json({ error: { code, message } }, status as any);
}
