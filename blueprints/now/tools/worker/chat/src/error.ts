import type { Context } from "hono";

type ErrorCode = "invalid_request" | "unauthorized" | "forbidden" | "not_found" | "conflict" | "rate_limited";

const STATUS: Record<ErrorCode, number> = {
  invalid_request: 400,
  unauthorized: 401,
  forbidden: 403,
  not_found: 404,
  conflict: 409,
  rate_limited: 429,
};

export function errorResponse(c: Context, code: ErrorCode, message: string) {
  return c.json({ error: { code, message } }, STATUS[code] as any);
}
