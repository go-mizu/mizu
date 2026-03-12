import type { Context } from "hono";
import type { Env, ScreenshotRequest } from "./types";
import { cfAvailable, proxyCF } from "./cf";

export async function handleScreenshot(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: ScreenshotRequest;
  try {
    body = await c.req.json<ScreenshotRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  if (!cfAvailable(c.env)) {
    return c.json(
      { success: false, errors: [{ code: 503, message: "CF credentials not configured; screenshot requires a real browser" }], result: null },
      503
    );
  }

  const type = body.screenshotOptions?.type ?? "png";
  const contentType = type === "jpeg" ? "image/jpeg" : "image/png";

  const cf = await proxyCF("screenshot", body, c.env as any, true);

  if (cf.ok && cf.blob) {
    const headers = new Headers({ "Content-Type": contentType });
    if (cf.browserMsUsed) headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
    return new Response(cf.blob, { headers });
  }

  if (cf.rateLimited) {
    return c.json(
      { success: false, errors: [{ code: 429, message: "CF rate limited; screenshot requires a real browser" }], result: null },
      503
    );
  }

  return c.json(
    { success: false, errors: [{ code: cf.status, message: "CF error" }], result: null },
    cf.status as any
  );
}
