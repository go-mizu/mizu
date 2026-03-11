import type { Context } from "hono";
import type { Env, LinksRequest } from "./types";
import { cacheGet, cacheSet } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { extractLinks } from "./links";

export function filterLinksForEndpoint(
  links: string[],
  pageUrl: string,
  _visibleLinksOnly: boolean,
  excludeExternalLinks: boolean
): string[] {
  if (!excludeExternalLinks) return links;
  const host = new URL(pageUrl).hostname;
  return links.filter((l) => {
    try { return new URL(l).hostname === host; } catch { return false; }
  });
}

export async function handleLinks(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: LinksRequest;
  try {
    body = await c.req.json<LinksRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  // Raw html: extract immediately, no cache
  if (body.html && !body.url) {
    const links = await extractLinks(body.html, "https://localhost");
    return c.json({ success: true, result: links });
  }

  const url = body.url!;
  const visibleLinksOnly = body.visibleLinksOnly ?? false;
  const excludeExternalLinks = body.excludeExternalLinks ?? false;
  const paramsHash = `${visibleLinksOnly}:${excludeExternalLinks}`;

  // L1 + L2
  const cached = await cacheGet(c.env.DB, url, "links", paramsHash);
  if (cached?.result != null) {
    return c.json({ success: true, result: JSON.parse(cached.result) });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("links", body, c.env as any);
    if (cf.ok) {
      const result: string[] = (cf.body as any)?.result ?? [];
      await cacheSet(c.env.DB, url, "links", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
      const res = c.json({ success: true, result });
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
    const rawLinks = await extractLinks(html, url);
    const result = filterLinksForEndpoint(rawLinks, url, visibleLinksOnly, excludeExternalLinks);
    await cacheSet(c.env.DB, url, "links", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
    const res = c.json({ success: true, result });
    res.headers.set("X-Fallback", "true");
    return res;
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
