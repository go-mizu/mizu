import { z } from "@hono/zod-openapi";

export const ErrorSchema = z
  .object({
    error: z.string().openapi({ example: "not_found" }),
    message: z.string().openapi({ example: "File not found" }),
  })
  .openapi("Error");

export const errRes = (desc: string) =>
  ({
    description: desc,
    content: { "application/json": { schema: ErrorSchema } },
  }) as const;
