import puppeteer from '@cloudflare/puppeteer';

export type ConversionMethod = 'primary' | 'ai' | 'browser';

export interface ConversionResult {
  markdown: string;
  method: ConversionMethod;
  durationMs: number;
  title: string;
  tokens?: number;
  sourceUrl: string;
  fetchFailed?: boolean; // true when all tiers failed to retrieve content
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Env = { AI: any; BROWSER: Fetcher };

const UA = 'Mozilla/5.0 (compatible; go-mizu-markdown/1.0; +https://markdown.go-mizu.workers.dev)';
const FETCH_TIMEOUT_MS = 10_000;
const BROWSER_TIMEOUT_MS = 20_000;

interface CachedResult {
  result: ConversionResult;
}

const resultCache = new Map<string, CachedResult>();

export async function convert(url: string, env: Env): Promise<ConversionResult> {
  // Validate URL
  const parsed = new URL(url); // throws TypeError if invalid
  if (!['http:', 'https:'].includes(parsed.protocol)) {
    throw new Error('Only http and https URLs are supported');
  }

  // Check in-memory cache
  const cached = resultCache.get(url);
  if (cached) {
    return { ...cached.result, durationMs: 0 };
  }

  const result = await doConvert(url, env);

  // Cache the result (no TTL — keep until isolate recycles)
  // Only cache successful conversions (not error fallbacks)
  if (!result.fetchFailed) {
    resultCache.set(url, { result });
  }

  return result;
}

async function doConvert(url: string, env: Env): Promise<ConversionResult> {
  const start = Date.now();

  // Tier 1: Native Markdown negotiation
  const nativeResult = await tryNativeMarkdown(url);
  if (nativeResult !== null) {
    return {
      markdown: nativeResult,
      method: 'primary',
      durationMs: Date.now() - start,
      title: extractTitleFromMarkdown(nativeResult),
      sourceUrl: url,
    };
  }

  // Fetch HTML (shared between tiers 2 and 3)
  const html = await fetchHTML(url);

  if (html !== null) {
    // Tier 2: Workers AI toMarkdown
    const aiResult = await tryWorkersAI(html, env).catch(() => null);
    if (aiResult !== null) {
      return {
        markdown: aiResult.markdown,
        method: 'ai',
        durationMs: Date.now() - start,
        title: extractTitleFromHTML(html),
        tokens: aiResult.tokens,
        sourceUrl: url,
      };
    }
  }

  // Tier 3: Browser Rendering via Puppeteer → AI
  let browserHtml = '';
  try {
    browserHtml = await tryBrowserRendering(url, env);
  } catch {
    // Browser binding unavailable or navigation failed; fall through to stripHtml
  }
  const aiFromBrowser = browserHtml ? await tryWorkersAI(browserHtml, env).catch(() => null) : null;
  const fetchFailed = !browserHtml && !aiFromBrowser;
  const markdown = aiFromBrowser?.markdown ?? (browserHtml ? stripHtml(browserHtml) : 'Unable to retrieve page content.');

  return {
    markdown,
    method: 'browser',
    durationMs: Date.now() - start,
    title: browserHtml ? extractTitleFromHTML(browserHtml) : 'Untitled',
    tokens: aiFromBrowser?.tokens,
    fetchFailed: fetchFailed || undefined,
    sourceUrl: url,
  };
}

// Tier 1: Accept: text/markdown negotiation
async function tryNativeMarkdown(url: string): Promise<string | null> {
  try {
    const resp = await fetch(url, {
      headers: {
        Accept: 'text/markdown, text/x-markdown, text/plain;q=0.9, */*;q=0.1',
        'User-Agent': UA,
      },
      redirect: 'follow',
      signal: AbortSignal.timeout(FETCH_TIMEOUT_MS),
    });
    if (!resp.ok) return null;
    const ct = resp.headers.get('content-type') ?? '';
    if (ct.includes('text/markdown') || ct.includes('text/x-markdown')) {
      return await resp.text();
    }
    return null;
  } catch {
    return null;
  }
}

// Fetch raw HTML for tier 2
async function fetchHTML(url: string): Promise<string | null> {
  try {
    const resp = await fetch(url, {
      headers: {
        Accept: 'text/html,application/xhtml+xml,*/*;q=0.8',
        'User-Agent': UA,
      },
      redirect: 'follow',
      signal: AbortSignal.timeout(FETCH_TIMEOUT_MS),
    });
    if (!resp.ok) return null;
    const contentLength = parseInt(resp.headers.get('content-length') ?? '0', 10);
    if (contentLength > 5_000_000) return null; // skip responses > 5MB
    return await resp.text();
  } catch {
    return null;
  }
}

// Tier 2: Cloudflare Workers AI toMarkdown()
async function tryWorkersAI(
  html: string,
  env: Env
): Promise<{ markdown: string; tokens: number } | null> {
  const blob = new Blob([html], { type: 'text/html' });
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const results: any[] = await env.AI.toMarkdown([{ name: 'page.html', blob }]);
  if (!results || results.length === 0) return null;
  const r = results[0];
  const markdown = (r.data ?? r.result ?? '') as string;
  if (!markdown.trim()) return null;
  return { markdown, tokens: r.tokens ?? 0 };
}

// Tier 3: Browser Rendering via @cloudflare/puppeteer
async function tryBrowserRendering(url: string, env: Env): Promise<string> {
  const browser = await puppeteer.launch(env.BROWSER);
  try {
    const page = await browser.newPage();
    await page.setUserAgent(UA);
    await page.goto(url, {
      waitUntil: 'networkidle0',
      timeout: BROWSER_TIMEOUT_MS,
    });
    return await page.content();
  } finally {
    await browser.close();
  }
}

// Extract <title> from HTML
function extractTitleFromHTML(html: string): string {
  const m = html.match(/<title[^>]*>([^<]+)<\/title>/i);
  return m ? m[1].trim() : 'Untitled';
}

// Extract first H1 from Markdown
function extractTitleFromMarkdown(md: string): string {
  const m = md.match(/^#{1,2}\s+(.+)$/m);
  return m ? m[1].trim() : 'Untitled';
}

// Last-resort HTML strip (fallback if AI unavailable)
function stripHtml(html: string): string {
  return html
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/\s{3,}/g, '\n\n')
    .trim();
}
