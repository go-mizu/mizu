import type { Context, Next } from "hono";
import type { Env } from "./types";

export async function authMiddleware(c: Context<{ Bindings: Env }>, next: Next) {
  const header = c.req.header("Authorization");
  if (!header) {
    return c.json({ success: false, errors: [{ code: 1000, message: "Unauthorized" }], result: null }, 401);
  }

  const [scheme, token] = header.split(" ", 2);
  if (scheme !== "Bearer" || !token || token !== c.env.AUTH_TOKEN) {
    return c.json({ success: false, errors: [{ code: 1000, message: "Unauthorized" }], result: null }, 401);
  }

  await next();
}
