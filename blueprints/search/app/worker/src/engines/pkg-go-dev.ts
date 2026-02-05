/**
 * pkg.go.dev Search Engine adapter.
 * Searches for Go packages on pkg.go.dev.
 * Note: pkg.go.dev doesn't have an official public API,
 * so we scrape the search results page.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { decodeHtmlEntities } from '../lib/html-parser';

// ========== pkg.go.dev Types ==========

interface GoPackage {
  name: string;
  path: string;
  modulePath?: string;
  synopsis?: string;
  version?: string;
  publishedAt?: string;
  importedBy?: number;
  license?: string;
}

const PKG_GO_DEV_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

export class PkgGoDevEngine implements OnlineEngine {
  name = 'pkg.go.dev';
  shortcut = 'go';
  categories: Category[] = ['it'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  // Reserved for pagination calculation: private resultsPerPage = 25;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('m', 'package'); // Search for packages (not modules)

    // pkg.go.dev uses limit and page (0-indexed internally, but displayed 1-indexed)
    if (params.page > 1) {
      // They use a cursor-based approach, but page param works for basic pagination
      searchParams.set('page', String(params.page));
    }

    return {
      url: `https://pkg.go.dev/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': PKG_GO_DEV_USER_AGENT,
        Accept: 'text/html,application/xhtml+xml',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const packages = this.parseSearchHtml(body);

      for (const pkg of packages) {
        const pkgUrl = `https://pkg.go.dev/${pkg.path}`;

        // Build content with synopsis and metadata
        let content = pkg.synopsis || '';
        const meta: string[] = [];

        if (pkg.version) {
          meta.push(pkg.version);
        }
        if (pkg.importedBy !== undefined && pkg.importedBy > 0) {
          meta.push(`${formatCount(pkg.importedBy)} imports`);
        }
        if (pkg.license) {
          meta.push(pkg.license);
        }

        if (meta.length > 0) {
          content = content
            ? `${content} | ${meta.join(' | ')}`
            : meta.join(' | ');
        }

        let publishedAt = '';
        if (pkg.publishedAt) {
          try {
            publishedAt = new Date(pkg.publishedAt).toISOString();
          } catch {
            publishedAt = pkg.publishedAt;
          }
        }

        results.results.push({
          url: pkgUrl,
          title: pkg.path,
          content,
          engine: this.name,
          score: this.weight,
          category: 'it',
          template: 'packages',
          publishedAt,
          language: 'Go',
          metadata: {
            modulePath: pkg.modulePath,
            version: pkg.version,
            importedBy: pkg.importedBy,
            license: pkg.license,
          },
        });
      }
    } catch {
      // Parse error
    }

    return results;
  }

  private parseSearchHtml(html: string): GoPackage[] {
    const packages: GoPackage[] = [];

    // pkg.go.dev uses SearchSnippet divs with class "SearchSnippet"
    // The structure is:
    // <div class="SearchSnippet">
    //   <div class="SearchSnippet-header">
    //     <a href="/path" data-test-id="snippet-title">path</a>
    //     <span class="SearchSnippet-header-version">v1.0.0</span>
    //   </div>
    //   <p class="SearchSnippet-synopsis">Description</p>
    //   <div class="SearchSnippet-infoLabel">
    //     <span>Imported by: 1234</span>
    //     <span>License</span>
    //     <span>Published: date</span>
    //   </div>
    // </div>

    // Match search result snippets
    const snippetRegex = /<div\s+class="SearchSnippet"[^>]*>([\s\S]*?)<\/div>\s*(?=<div\s+class="SearchSnippet"|<div\s+class="Pagination"|<footer|$)/gi;
    let snippetMatch;

    while ((snippetMatch = snippetRegex.exec(html)) !== null) {
      const content = snippetMatch[1];

      // Extract path from the link
      const pathMatch = content.match(/<a[^>]*href="\/([^"]+)"[^>]*data-test-id="snippet-title"[^>]*>([^<]*)<\/a>/i);
      if (!pathMatch) {
        // Try alternative pattern
        const altPathMatch = content.match(/<a[^>]*href="\/([^"]+)"[^>]*class="[^"]*SearchSnippet-header-path[^"]*"[^>]*>/i);
        if (!altPathMatch) continue;
      }
      const path = pathMatch ? pathMatch[1] : '';
      if (!path) continue;

      // Extract version
      const versionMatch = content.match(/<span[^>]*class="[^"]*SearchSnippet-header-version[^"]*"[^>]*>([^<]+)<\/span>/i);
      const version = versionMatch ? versionMatch[1].trim() : '';

      // Extract synopsis/description
      const synopsisMatch = content.match(/<p[^>]*class="[^"]*SearchSnippet-synopsis[^"]*"[^>]*>([^<]*)<\/p>/i);
      const synopsis = synopsisMatch ? decodeHtmlEntities(synopsisMatch[1].trim()) : '';

      // Extract imported by count
      const importedByMatch = content.match(/Imported by[:\s]*(\d[\d,]*)/i);
      const importedBy = importedByMatch
        ? parseInt(importedByMatch[1].replace(/,/g, ''), 10)
        : undefined;

      // Extract license
      const licenseMatch = content.match(/<span[^>]*class="[^"]*go-textSubtle[^"]*"[^>]*>([A-Z][A-Za-z0-9-]+(?:, [A-Z][A-Za-z0-9-]+)*)<\/span>/i);
      const license = licenseMatch ? licenseMatch[1].trim() : '';

      // Extract published date
      const publishedMatch = content.match(/Published[:\s]*([A-Za-z]+\s+\d+,\s+\d{4}|\d{4}-\d{2}-\d{2})/i);
      const publishedAt = publishedMatch ? publishedMatch[1] : '';

      packages.push({
        name: path.split('/').pop() || path,
        path,
        synopsis,
        version,
        importedBy,
        license,
        publishedAt,
      });
    }

    // If no snippets found with the first regex, try simpler extraction
    if (packages.length === 0) {
      // Fallback: look for any links to packages
      const linkRegex = /<a[^>]*href="\/((?:github\.com|golang\.org|google\.golang\.org)[^"]+)"[^>]*>([^<]+)<\/a>/gi;
      let linkMatch;
      const seen = new Set<string>();

      while ((linkMatch = linkRegex.exec(html)) !== null) {
        const path = linkMatch[1];
        if (seen.has(path)) continue;
        seen.add(path);

        // Try to find description near the link
        const pos = linkMatch.index;
        const nearbyHtml = html.slice(pos, pos + 500);
        const descMatch = nearbyHtml.match(/synopsis[^>]*>([^<]+)</i);
        const synopsis = descMatch ? decodeHtmlEntities(descMatch[1].trim()) : '';

        packages.push({
          name: path.split('/').pop() || path,
          path,
          synopsis,
        });

        if (packages.length >= 25) break;
      }
    }

    return packages;
  }
}

/**
 * Format count with K/M suffixes.
 */
function formatCount(count: number): string {
  if (count >= 1_000_000) {
    return (count / 1_000_000).toFixed(1) + 'M';
  }
  if (count >= 1_000) {
    return (count / 1_000).toFixed(1) + 'k';
  }
  return count.toString();
}
