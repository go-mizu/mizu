import type { Context } from "hono";
import type { Env, JsonRequest } from "./types";
import { cacheGet, cacheSet, hashParams } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { htmlToMarkdown } from "./markdown";

export function jsonParamsObj(req: Partial<JsonRequest>) {
  return {
    prompt: req.prompt,
    schema: req.response_format?.schema,
  };
}

export async function handleJson(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: JsonRequest;
  try {
    body = await c.req.json<JsonRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  const paramsHash = await hashParams(jsonParamsObj(body));

  // Raw html: skip cache and CF, try AI fallback directly
  if (body.html && !body.url) {
    return await runJsonFallback(c, body.html, body);
  }

  const url = body.url!;

  // L1 + L2
  const cached = await cacheGet(c.env.DB, url, "json", paramsHash);
  if (cached?.result != null) {
    return c.json({ success: true, result: JSON.parse(cached.result) });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("json", body, c.env as any);
    if (cf.ok) {
      const result = (cf.body as any)?.result ?? {};
      await cacheSet(c.env.DB, url, "json", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
      const res = c.json({ success: true, result });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
  }

  // L4: own fallback — fetch HTML, run Workers AI if binding available
  try {
    const resp = await fetch(url, {
      headers: { "User-Agent": body.userAgent ?? "mizu-browser/1.0", Accept: "text/html,*/*", ...(body.setExtraHTTPHeaders ?? {}) },
      redirect: "follow",
    });
    const html = await resp.text();
    return await runJsonFallback(c, html, body, url, paramsHash);
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}

async function runJsonFallback(
  c: Context<{ Bindings: Env }>,
  html: string,
  body: JsonRequest,
  url?: string,
  paramsHash?: string
): Promise<Response> {
  const env = c.env as any;

  // Try Workers AI if binding is available
  if (env.AI && (body.prompt || body.response_format)) {
    try {
      const md = htmlToMarkdown(html);
      const systemPrompt = body.response_format
        ? `Extract data matching this JSON schema: ${JSON.stringify(body.response_format.schema)}. Return only valid JSON.`
        : "Extract structured data from the content. Return only valid JSON.";
      const userPrompt = body.prompt ? `${body.prompt}\n\n${md}` : md;

      const aiResp: any = await env.AI.run("@cf/meta/llama-3.1-8b-instruct-fast", {
        messages: [
          { role: "system", content: systemPrompt },
          { role: "user", content: userPrompt },
        ],
      });

      const text: string = aiResp?.response ?? "";
      const jsonMatch = text.match(/\{[\s\S]*\}|\[[\s\S]*\]/);
      if (jsonMatch) {
        const result = JSON.parse(jsonMatch[0]);
        if (url && paramsHash !== undefined) {
          await cacheSet(c.env.DB, url, "json", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
        }
        const res = c.json({ success: true, result });
        res.headers.set("X-Fallback", "true");
        return res;
      }
    } catch {
      // AI failed — fall through to error
    }
  }

  // No AI binding or AI failed
  return c.json({
    success: false,
    errors: [{ code: 503, message: "AI extraction unavailable; CF rate limited and AI binding not configured" }],
    result: null,
  }, 503);
}
