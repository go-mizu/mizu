import { createRoute, z } from "@hono/zod-openapi";
import type { App } from "../types";
import { auth } from "../middleware/auth";

const route = createRoute({
  method: "get",
  path: "/stat",
  tags: ["management"],
  security: [{ bearer: [] }],
  responses: {
    200: {
      description: "Storage usage",
      content: {
        "application/json": {
          schema: z.object({
            files: z.number().int().openapi({ example: 42 }),
            bytes: z.number().int().openapi({ example: 1048576 }),
          }),
        },
      },
    },
  },
});

export function register(app: App) {
  app.use("/stat", auth);

  app.openapi(route, async (c) => {
    const actor = c.get("actor");
    const row = await c.env.DB.prepare(
      "SELECT COUNT(*) as files, COALESCE(SUM(size), 0) as bytes FROM files WHERE owner = ?",
    )
      .bind(actor)
      .first<{ files: number; bytes: number }>();

    return c.json({ files: row?.files || 0, bytes: row?.bytes || 0 }, 200);
  });
}
