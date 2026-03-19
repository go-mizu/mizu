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

const MAX_PATH_LENGTH = 1024;
const MAX_NAME_LENGTH = 255;

/**
 * Validate a file/folder path. Returns null if valid, error message if invalid.
 *
 * Rules:
 *  - No empty paths
 *  - No ".." segments (directory traversal)
 *  - No leading "/" (absolute paths)
 *  - No "//" (double slashes)
 *  - No null bytes
 *  - Max 1024 chars total, 255 chars per segment
 */
export function validatePath(path: string): string | null {
  if (!path) return "Path is required";
  if (path.length > MAX_PATH_LENGTH) return "Path exceeds 1024 characters";
  if (path.startsWith("/")) return "Path must not start with /";
  if (path.includes("\0")) return "Path must not contain null bytes";
  if (path.includes("//")) return "Path must not contain //";

  // Check each segment for traversal and length
  const segments = path.replace(/\/$/, "").split("/");
  for (const seg of segments) {
    if (seg === "..") return "Path must not contain ..";
    if (seg === ".") return "Path must not contain .";
    if (seg.length > MAX_NAME_LENGTH) return "Filename exceeds 255 characters";
    if (seg.length === 0) return "Path must not contain empty segments";
  }

  return null;
}
