import type { Context } from "hono";
import type { Env, ContentRequest } from "./types";
import { cacheGet, cacheSet } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { extractTitle } from "./markdown";

// Pure helper — exported for testing
export function buildContentResult(html: string): { html: string; title: string } {
  return { html, title: extractTitle(html) };
}

export async function handleContent(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: ContentRequest;
  try {
    body = await c.req.json<ContentRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  if (body.url) {
    try { new URL(body.url); } catch {
      return c.json({ success: false, errors: [{ code: 1001, message: "url is not a valid URL" }], result: null }, 400);
    }
  }

  // If raw html provided without url, skip cache and return immediately
  if (body.html && !body.url) {
    return c.json({ success: true, result: body.html });
  }

  const url = body.url!;
  const paramsHash = "";  // content caches by URL only

  // L1 + L2 cache
  const cached = await cacheGet(c.env.DB, url, "content", paramsHash);
  if (cached?.html != null) {
    return c.json({ success: true, result: cached.html });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("content", body, c.env as any);
    if (cf.ok) {
      const result = (cf.body as any)?.result ?? "";
      const title = extractTitle(typeof result === "string" ? result : "");
      await cacheSet(c.env.DB, url, "content", paramsHash, { html: result, markdown: null, result: null, title });
      const res = c.json({ success: true, result });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
    // rate limited → fall through to L4
  }

  // L4: own fallback — plain fetch
  try {
    const resp = await fetch(url, {
      headers: {
        "User-Agent": body.userAgent ?? "mizu-browser/1.0",
        Accept: "text/html,*/*",
        ...(body.setExtraHTTPHeaders ?? {}),
      },
      redirect: "follow",
    });
    const html = await resp.text();
    const { title } = buildContentResult(html);
    await cacheSet(c.env.DB, url, "content", paramsHash, { html, markdown: null, result: null, title });
    return c.json({ success: true, result: html });
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
