import type { Context } from "hono";

/**
 * Extract the wildcard portion of a URL path after a given prefix.
 * e.g. for URL "/files/docs/readme.md" with prefix "/files/" → "docs/readme.md"
 */
export function wildcardPath(c: Context, prefix: string): string {
  const url = new URL(c.req.url);
  const raw = url.pathname;
  const idx = raw.indexOf(prefix);
  if (idx === -1) return "";
  return decodeURIComponent(raw.slice(idx + prefix.length));
}
