import type { Context } from "hono";
import type { Env, SnapshotRequest, SnapshotResult } from "./types";
import { cacheGet, cacheSet } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { extractTitle } from "./markdown";

export function buildSnapshotFallback(html: string): SnapshotResult {
  return { content: html, screenshot: null };
}

export async function handleSnapshot(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: SnapshotRequest;
  try {
    body = await c.req.json<SnapshotRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  if (body.html && !body.url) {
    return c.json({ success: true, result: buildSnapshotFallback(body.html) });
  }

  const url = body.url!;
  const paramsHash = "";

  // L1 + L2 — html col = content, result col = {"screenshot":"..."}
  const cached = await cacheGet(c.env.DB, url, "snapshot", paramsHash);
  if (cached?.html != null) {
    const screenshot = cached.result ? (JSON.parse(cached.result) as { screenshot: string | null }).screenshot : null;
    return c.json({ success: true, result: { content: cached.html, screenshot } });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("snapshot", body, c.env as any);
    if (cf.ok) {
      const cfResult = (cf.body as any)?.result as SnapshotResult;
      const title = extractTitle(cfResult.content ?? "");
      await cacheSet(c.env.DB, url, "snapshot", paramsHash, {
        html: cfResult.content ?? null,
        markdown: null,
        result: JSON.stringify({ screenshot: cfResult.screenshot }),
        title,
      });
      const res = c.json({ success: true, result: cfResult });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
  }

  // L4: own fallback — fetch HTML, screenshot = null
  try {
    const resp = await fetch(url, {
      headers: { "User-Agent": body.userAgent ?? "mizu-browser/1.0", Accept: "text/html,*/*", ...(body.setExtraHTTPHeaders ?? {}) },
      redirect: "follow",
    });
    const html = await resp.text();
    const title = extractTitle(html);
    const result = buildSnapshotFallback(html);
    await cacheSet(c.env.DB, url, "snapshot", paramsHash, {
      html,
      markdown: null,
      result: JSON.stringify({ screenshot: null }),
      title,
    });
    const res = c.json({ success: true, result });
    res.headers.set("X-Fallback", "true");
    return res;
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
