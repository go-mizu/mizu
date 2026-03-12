import type { Context } from "hono";
import type { Env, MarkdownRequest } from "./types";
import { cacheGet, cacheSet } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { htmlToMarkdown, extractTitle } from "./markdown";

export async function handleMarkdown(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: MarkdownRequest;
  try {
    body = await c.req.json<MarkdownRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  // Raw html → convert immediately, no cache
  if (body.html && !body.url) {
    return c.json({ success: true, result: htmlToMarkdown(body.html) });
  }

  const url = body.url!;
  const paramsHash = "";

  // L1 + L2
  const cached = await cacheGet(c.env.DB, url, "markdown", paramsHash);
  if (cached?.markdown != null) {
    return c.json({ success: true, result: cached.markdown });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("markdown", body, c.env as any);
    if (cf.ok) {
      const md = (cf.body as any)?.result ?? "";
      await cacheSet(c.env.DB, url, "markdown", paramsHash, { html: null, markdown: md, result: null, title: null });
      const res = c.json({ success: true, result: md });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
  }

  // L4: own fallback
  try {
    const resp = await fetch(url, {
      headers: { "User-Agent": body.userAgent ?? "mizu-browser/1.0", Accept: "text/html,*/*", ...(body.setExtraHTTPHeaders ?? {}) },
      redirect: "follow",
    });
    const html = await resp.text();
    const md = htmlToMarkdown(html);
    const title = extractTitle(html);
    await cacheSet(c.env.DB, url, "markdown", paramsHash, { html: null, markdown: md, result: null, title });
    return c.json({ success: true, result: md });
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
