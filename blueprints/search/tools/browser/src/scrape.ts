import type { Context } from "hono";
import type { Env, ScrapeRequest, ScrapeSelectorResult, ScrapeNodeResult } from "./types";
import { cacheGet, cacheSet, hashParams } from "./cache";
import { cfAvailable, proxyCF } from "./cf";

/**
 * Fallback scraper using HTMLRewriter.
 * Returns text, html (reconstructed), and attributes for each matched element.
 * Bounding box (height, width, top, left) = 0 — no layout engine available.
 */
export async function scrapeHtml(
  html: string,
  elements: Array<{ selector: string }>
): Promise<ScrapeSelectorResult[]> {
  return Promise.all(
    elements.map(async ({ selector }) => {
      const nodes: ScrapeNodeResult[] = [];

      // We collect per-element data using a stack approach
      // HTMLRewriter processes elements in document order
      const nodeData: Array<{ attrs: Array<{ name: string; value: string }>; texts: string[]; tagName: string }> = [];
      let depth = 0;

      const rw = new HTMLRewriter()
        .on(selector, {
          element(el) {
            const attrs: Array<{ name: string; value: string }> = [];
            for (const [name, value] of el.attributes) {
              attrs.push({ name, value });
            }
            const tagName = el.tagName;
            nodeData.push({ attrs, texts: [], tagName });
            depth++;
            el.onEndTag(() => {
              depth--;
            });
          },
          text(chunk) {
            if (nodeData.length > 0 && depth > 0) {
              nodeData[nodeData.length - 1].texts.push(chunk.text);
            }
          },
        })
        .transform(new Response(html, { headers: { "Content-Type": "text/html" } }));

      await rw.text();

      for (const nd of nodeData) {
        const text = nd.texts.join("").trim();
        const attrStr = nd.attrs.map(a => ` ${a.name}="${a.value}"`).join("");
        const htmlStr = `<${nd.tagName}${attrStr}>${text}</${nd.tagName}>`;
        nodes.push({
          text,
          html: htmlStr,
          attributes: nd.attrs,
          height: 0,
          width: 0,
          top: 0,
          left: 0,
        });
      }

      return { selector, results: nodes };
    })
  );
}

export async function handleScrape(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: ScrapeRequest;
  try {
    body = await c.req.json<ScrapeRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }
  if (!body.elements || body.elements.length === 0) {
    return c.json({ success: false, errors: [{ code: 1001, message: "elements is required" }], result: null }, 400);
  }

  // Sort selectors for deterministic hashing
  const sortedSelectors = [...body.elements].sort((a, b) => a.selector.localeCompare(b.selector));
  const paramsHash = await hashParams(sortedSelectors.map(e => e.selector));

  if (body.html && !body.url) {
    const result = await scrapeHtml(body.html, body.elements);
    return c.json({ success: true, result });
  }

  const url = body.url!;

  // L1 + L2
  const cached = await cacheGet(c.env.DB, url, "scrape", paramsHash);
  if (cached?.result != null) {
    return c.json({ success: true, result: JSON.parse(cached.result) });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("scrape", body, c.env as any);
    if (cf.ok) {
      const result = (cf.body as any)?.result ?? [];
      await cacheSet(c.env.DB, url, "scrape", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
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
    const result = await scrapeHtml(html, body.elements);
    await cacheSet(c.env.DB, url, "scrape", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
    const res = c.json({ success: true, result });
    res.headers.set("X-Fallback", "true");
    return res;
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
