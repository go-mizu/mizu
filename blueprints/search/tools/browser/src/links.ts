import { matchesAnyPattern } from "./patterns";
import type { CrawlOptions } from "./types";

/**
 * Extract all href links from HTML using HTMLRewriter.
 */
export async function extractLinks(html: string, baseUrl: string): Promise<string[]> {
  const links: string[] = [];

  // HTMLRewriter works on Response objects
  const fakeResponse = new Response(html, {
    headers: { "Content-Type": "text/html" },
  });

  const extractor = new HTMLRewriter()
    .on("a[href]", {
      element(el) {
        const href = el.getAttribute("href");
        if (href) links.push(href);
      },
    })
    .transform(fakeResponse);

  // Drain the body to trigger rewriting
  await extractor.text();

  return links;
}

/**
 * Resolve, normalize, and filter discovered links.
 * Returns absolute URLs that pass all filters.
 */
export function filterLinks(
  rawLinks: string[],
  pageUrl: string,
  seedUrl: string,
  options: {
    includeSubdomains: boolean;
    includeExternalLinks: boolean;
    includePatterns: string[];
    excludePatterns: string[];
  }
): string[] {
  const seedHost = new URL(seedUrl).hostname;
  const seedOrigin = new URL(seedUrl).origin;

  const seen = new Set<string>();
  const result: string[] = [];

  for (const raw of rawLinks) {
    let abs: URL;
    try {
      abs = new URL(raw, pageUrl);
    } catch {
      continue;
    }

    // Only http/https
    if (abs.protocol !== "http:" && abs.protocol !== "https:") continue;

    // Normalize: remove fragment, trailing slash on path (except root)
    abs.hash = "";
    if (abs.pathname.length > 1 && abs.pathname.endsWith("/")) {
      abs.pathname = abs.pathname.slice(0, -1);
    }

    const url = abs.toString();
    if (seen.has(url)) continue;
    seen.add(url);

    const host = abs.hostname;

    // Domain filtering
    if (host === seedHost) {
      // Same domain — always allowed
    } else if (options.includeSubdomains && host.endsWith("." + seedHost)) {
      // Subdomain allowed
    } else if (options.includeExternalLinks) {
      // External allowed
    } else {
      continue;
    }

    // Exclude patterns take precedence
    if (options.excludePatterns.length > 0 && matchesAnyPattern(url, options.excludePatterns)) {
      continue;
    }

    // Include patterns — if specified, URL must match at least one
    if (options.includePatterns.length > 0 && !matchesAnyPattern(url, options.includePatterns)) {
      continue;
    }

    // Skip common non-content paths
    if (/\.(jpg|jpeg|png|gif|svg|webp|ico|pdf|zip|gz|tar|mp4|mp3|woff|woff2|ttf|eot)(\?|$)/i.test(abs.pathname)) {
      continue;
    }

    result.push(url);
  }

  return result;
}
