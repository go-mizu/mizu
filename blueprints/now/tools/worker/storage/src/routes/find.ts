import { createRoute, z } from "@hono/zod-openapi";
import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { errRes } from "../schema";

type C = Context<{ Bindings: Env; Variables: Variables }>;

// ── In-memory cache per isolate (30s TTL) ───────────────────────────
const cache = new Map<string, { rows: { path: string; name: string }[]; ts: number }>();
const TTL = 30_000;

export function invalidateCache(owner: string) {
  cache.delete(owner);
}

async function getCachedNames(
  db: D1Database,
  owner: string,
): Promise<{ path: string; name: string }[]> {
  const hit = cache.get(owner);
  if (hit && Date.now() - hit.ts < TTL) return hit.rows;

  const { results } = await db
    .prepare("SELECT path, name FROM files WHERE owner = ?")
    .bind(owner)
    .all();
  const rows = (results || []).map((r) => ({
    path: r.path as string,
    name: r.name as string,
  }));
  cache.set(owner, { rows, ts: Date.now() });
  return rows;
}

// ── Route definition (single source of truth for validation + docs) ─
const route = createRoute({
  method: "get",
  path: "/find",
  tags: ["search"],
  security: [{ bearer: [] }],
  request: {
    query: z.object({
      q: z.string().min(1).openapi({ description: "Search query", example: "readme" }),
      limit: z.coerce
        .number()
        .int()
        .min(1)
        .max(200)
        .default(50)
        .openapi({ description: "Max results" }),
    }),
  },
  responses: {
    200: {
      description: "Search results",
      content: {
        "application/json": {
          schema: z.object({
            query: z.string(),
            results: z.array(
              z.object({ path: z.string(), name: z.string() }),
            ),
          }),
        },
      },
    },
    400: errRes("Bad request"),
  },
});

export function register(app: App) {
  app.use("/find", auth);

  app.openapi(route, async (c) => {
    const { q, limit } = c.req.valid("query");
    const query = q.trim().toLowerCase();

    const actor = c.get("actor");
    const prefix = c.get("prefix");

    const names = await getCachedNames(c.env.DB, actor);
    const tokens = query.split(/\s+/);

    const hits: { path: string; name: string; score: number }[] = [];
    for (const { path, name } of names) {
      if (prefix && !path.startsWith(prefix)) continue;

      const lp = path.toLowerCase();
      const ln = name.toLowerCase();
      if (!tokens.every((t) => lp.includes(t))) continue;

      let score = 0;
      for (const t of tokens) {
        if (ln === t) score += 10;
        else if (ln.includes(t)) score += 5;
        else score += 1;
      }
      hits.push({ path, name, score });
    }

    hits.sort((a, b) => b.score - a.score);
    return c.json(
      { query, results: hits.slice(0, limit).map(({ path, name }) => ({ path, name })) },
      200,
    );
  });
}
