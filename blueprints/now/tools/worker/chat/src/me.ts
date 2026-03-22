import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { getSessionActor } from "./session";
import { detectPlatform, parseLinks } from "./social-icons";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

/** GET /me — own profile data */
export async function getMe(c: AppContext) {
  const actor = await getSessionActor(c);
  if (!actor) return c.json({ error: { code: "unauthorized", message: "Not signed in" } }, 401);

  const row = await c.env.DB.prepare(
    "SELECT actor, bio, links, created_at FROM actors WHERE actor = ?"
  ).bind(actor).first<{ actor: string; bio: string | null; links: string | null; created_at: number }>();

  if (!row) return c.json({ error: { code: "not_found", message: "Actor not found" } }, 404);

  return c.json({
    actor: row.actor,
    bio: row.bio || "",
    links: parseLinks(row.links),
    created_at: row.created_at,
  });
}

/** PUT /me/bio — update own bio */
export async function updateBio(c: AppContext) {
  const actor = await getSessionActor(c);
  if (!actor) return c.json({ error: { code: "unauthorized", message: "Not signed in" } }, 401);

  let body: { bio?: string };
  try {
    body = await c.req.json();
  } catch {
    return c.json({ error: { code: "invalid_request", message: "Invalid JSON" } }, 400);
  }

  const bio = typeof body.bio === "string" ? body.bio.trim().slice(0, 280) : "";

  await c.env.DB.prepare("UPDATE actors SET bio = ? WHERE actor = ?").bind(bio, actor).run();
  return c.json({ bio });
}

/** PUT /me/links — update own social links */
export async function updateLinks(c: AppContext) {
  const actor = await getSessionActor(c);
  if (!actor) return c.json({ error: { code: "unauthorized", message: "Not signed in" } }, 401);

  let body: { links?: { url: string }[] };
  try {
    body = await c.req.json();
  } catch {
    return c.json({ error: { code: "invalid_request", message: "Invalid JSON" } }, 400);
  }

  if (!Array.isArray(body.links)) {
    return c.json({ error: { code: "invalid_request", message: "links must be an array" } }, 400);
  }

  // Validate and normalize
  const links = body.links
    .filter((l): l is { url: string } => !!l && typeof l.url === "string")
    .filter(l => /^https?:\/\//.test(l.url) && l.url.length <= 500)
    .slice(0, 6)
    .map(l => ({ platform: detectPlatform(l.url), url: l.url }));

  const json = JSON.stringify(links);
  await c.env.DB.prepare("UPDATE actors SET links = ? WHERE actor = ?").bind(json, actor).run();
  return c.json({ links });
}
