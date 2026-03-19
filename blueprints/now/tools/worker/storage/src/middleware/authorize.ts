import type { Context } from "hono";
import type { Env, Variables } from "../types";
import { errorResponse } from "../lib/error";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

export type Role = "owner" | "editor" | "uploader" | "viewer";

/**
 * Capability sets per role.
 *
 * Uploader can write new files but NOT read/delete — it's not a superset of viewer.
 * The hierarchy is: owner ⊃ editor ⊃ viewer, with uploader as a separate branch.
 */
const ROLE_CAPS: Record<Role, Set<Role>> = {
  owner: new Set(["owner", "editor", "uploader", "viewer"]),
  editor: new Set(["editor", "uploader", "viewer"]),
  uploader: new Set(["uploader"]),
  viewer: new Set(["viewer"]),
};

/**
 * Resolve the permission an actor has on a file/folder.
 *
 * Uses Box-style folder inheritance:
 *   1. Direct share on the exact path → use it
 *   2. Walk up parent paths → first match wins (most specific)
 *   3. No match → null (no access)
 *
 * Returns 'owner' if actor === fileOwner.
 */
export async function resolvePermission(
  db: D1Database,
  actor: string,
  fileOwner: string,
  filePath: string,
): Promise<Role | null> {
  if (actor === fileOwner) return "owner";

  // Build list of candidate paths: the file itself + all parent folders
  // e.g. "docs/reports/q1.pdf" → ["docs/reports/q1.pdf", "docs/reports/", "docs/"]
  const candidates: string[] = [filePath];
  const parts = filePath.replace(/\/$/, "").split("/");
  for (let i = parts.length - 1; i > 0; i--) {
    candidates.push(parts.slice(0, i).join("/") + "/");
  }

  // Single query: find the most specific share
  // We order by path length descending → the first result is the most specific match
  const placeholders = candidates.map(() => "?").join(",");
  const sql = `
    SELECT s.permission
    FROM shares s
    JOIN objects o ON s.object_id = o.id
    WHERE s.grantee = ?
      AND o.owner = ?
      AND o.path IN (${placeholders})
    ORDER BY LENGTH(o.path) DESC
    LIMIT 1
  `;

  const binds = [actor, fileOwner, ...candidates];
  const row = await db.prepare(sql).bind(...binds).first<{ permission: string }>();

  if (!row) return null;
  return normalizeRole(row.permission);
}

/**
 * Check if a role includes the required capability.
 */
export function hasPermission(role: Role | null, required: Role): boolean {
  if (!role) return false;
  return ROLE_CAPS[role].has(required);
}

/**
 * Normalize legacy permission names to roles.
 */
export function normalizeRole(permission: string): Role {
  switch (permission) {
    case "read":
      return "viewer";
    case "write":
      return "editor";
    case "owner":
    case "editor":
    case "uploader":
    case "viewer":
      return permission as Role;
    default:
      return "viewer";
  }
}

/**
 * Normalize input permission to a valid share role.
 */
export function normalizeSharePermission(input: string): "viewer" | "editor" | "uploader" {
  switch (input) {
    case "read":
    case "viewer":
      return "viewer";
    case "write":
    case "editor":
      return "editor";
    case "uploader":
      return "uploader";
    default:
      return "viewer";
  }
}

// ── Scope enforcement ────────────────────────────────────────────────

const ALL_SCOPES = "*";

export function requireScope(c: AppContext, scope: string): Response | null {
  const scopes = c.get("scopes") || ALL_SCOPES;
  if (scopes === ALL_SCOPES) return null;

  const allowed = scopes.split(",").map((s: string) => s.trim());
  if (allowed.includes(scope) || allowed.includes(ALL_SCOPES)) return null;

  return errorResponse(c, "forbidden", `Token lacks required scope: ${scope}`);
}

/**
 * Check path prefix restriction for scoped API keys.
 */
export function checkPathPrefix(c: AppContext, path: string): Response | null {
  const prefix = c.get("pathPrefix") || "";
  if (!prefix) return null;
  if (path.startsWith(prefix)) return null;
  return errorResponse(c, "forbidden", "Path not allowed for this token");
}

// ── Filename sanitization ────────────────────────────────────────────

/**
 * Sanitize filename for Content-Disposition header.
 * Prevents header injection via quotes, newlines, or non-ASCII.
 */
export function sanitizeFilename(name: string): string {
  return name.replace(/["\\\r\n]/g, "_").replace(/[^\x20-\x7E]/g, "_");
}
