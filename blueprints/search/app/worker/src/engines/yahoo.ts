/**
 * Yahoo Search Engine adapter.
 * Parses Yahoo web search results from HTML.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { extractText, findElements, decodeHtmlEntities } from '../lib/html-parser';

const YAHOO_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// ========== Time Range Maps ==========

const timeRangeMap: Record<string, string> = {
  day: '1d',
  week: '1w',
  month: '1m',
  year: '1y',
};

// ========== URL Unwrapper ==========

/**
 * Unwrap Yahoo redirect URLs to get the real destination.
 * Yahoo uses /rrq/... redirects with RU parameter containing the actual URL.
 */
function unwrapYahooUrl(href: string): string {
  // Pattern: https://r.search.yahoo.com/_ylt=...;_ylu=.../RU=<encoded_url>/...
  const ruMatch = href.match(/\/RU=([^/]+)/);
  if (ruMatch) {
    try {
      return decodeURIComponent(ruMatch[1]);
    } catch {
      // Decode failed
    }
  }

  // Alternative pattern with u= parameter
  const uMatch = href.match(/[?&]u=([^&]+)/);
  if (uMatch) {
    try {
      return decodeURIComponent(uMatch[1]);
    } catch {
      // Decode failed
    }
  }

  return href;
}

// ========== YahooEngine ==========

export class YahooEngine implements OnlineEngine {
  name = 'yahoo';
  shortcut = 'yh';
  categories: Category[] = ['general'];
  supportsPaging = true;
  maxPage = 20;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('p', query);
    searchParams.set('ei', 'UTF-8');

    // Pagination - Yahoo uses 'b' parameter (1-indexed, 10 per page)
    if (params.page > 1) {
      const offset = (params.page - 1) * 10 + 1;
      searchParams.set('b', offset.toString());
    }

    // Time range filter
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('fr2', 'time');
      searchParams.set('btf', timeRangeMap[params.timeRange]);
    }

    // Safe search (vl=lang_en restricts adult content when combined with vm=r)
    if (params.safeSearch === 2) {
      searchParams.set('vm', 'r');
    } else if (params.safeSearch === 0) {
      searchParams.set('vm', 'i'); // Images/videos unrestricted
    }

    // Locale/region
    const locale = params.locale || 'en-US';
    const parts = locale.split('-');
    const lang = (parts[0] || 'en').toLowerCase();
    const region = (parts[1] || 'us').toLowerCase();

    // Set language parameter
    searchParams.set('vl', `lang_${lang}`);

    // Use regional Yahoo domain or default to .com
    let domain = 'search.yahoo.com';
    if (region === 'uk' || region === 'gb') {
      domain = 'uk.search.yahoo.com';
    } else if (region === 'de') {
      domain = 'de.search.yahoo.com';
    } else if (region === 'fr') {
      domain = 'fr.search.yahoo.com';
    } else if (region === 'jp') {
      domain = 'search.yahoo.co.jp';
    }

    return {
      url: `https://${domain}/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': YAHOO_USER_AGENT,
        Accept:
          'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'Accept-Language': `${lang}-${region.toUpperCase()},${lang};q=0.9,en;q=0.8`,
        DNT: '1',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Check for CAPTCHA or error
    if (
      body.includes('captcha') ||
      body.includes('unusual traffic') ||
      body.includes('robot')
    ) {
      return results;
    }

    // Yahoo search results are in <li> elements with class containing "algo"
    // Primary selector: div.algo or li with dd class
    const algoElements = findElements(body, 'div.algo');
    const ddElements = findElements(body, 'div.dd');

    // Combine both result types
    const resultElements = [...algoElements, ...ddElements];

    // Also try the newer result format with compTitle
    if (resultElements.length === 0) {
      const compTitleElements = findElements(body, 'div.compTitle');
      resultElements.push(...compTitleElements);
    }

    for (const el of resultElements) {
      const result = this.parseResult(el);
      if (result) {
        results.results.push(result);
      }
    }

    // Fallback: try to find results using h3 pattern
    if (results.results.length === 0) {
      const h3Pattern = /<h3[^>]*class="[^"]*title[^"]*"[^>]*>([\s\S]*?)<\/h3>/gi;
      let match: RegExpExecArray | null;
      while ((match = h3Pattern.exec(body)) !== null) {
        const h3Html = match[0];
        const linkMatch = h3Html.match(/<a\b[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/i);
        if (linkMatch) {
          let url = decodeHtmlEntities(linkMatch[1]);
          const title = extractText(linkMatch[2]).trim();

          if (url.includes('yahoo.com') && url.includes('/RU=')) {
            url = unwrapYahooUrl(url);
          }

          if (url && title && url.startsWith('http') && !url.includes('yahoo.com')) {
            results.results.push({
              url,
              title,
              content: '',
              engine: this.name,
              score: this.weight,
              category: 'general',
            });
          }
        }
      }
    }

    // Parse related searches / suggestions
    const relatedElements = findElements(body, 'div.compDlink');
    for (const el of relatedElements) {
      const text = extractText(el).trim();
      if (text && text.length < 100) {
        results.suggestions.push(text);
      }
    }

    // Alternative related searches pattern
    const altRelated = body.match(/Also try:[\s\S]*?<ul[^>]*>([\s\S]*?)<\/ul>/i);
    if (altRelated) {
      const linkPattern = /<a[^>]*>([^<]+)<\/a>/gi;
      let m: RegExpExecArray | null;
      while ((m = linkPattern.exec(altRelated[1])) !== null) {
        const term = extractText(m[1]).trim();
        if (term && term.length < 100 && !results.suggestions.includes(term)) {
          results.suggestions.push(term);
        }
      }
    }

    return results;
  }

  private parseResult(html: string): EngineResults['results'][number] | null {
    let url = '';
    let title = '';
    let content = '';

    // Find the title link (usually in h3 > a or just a with specific class)
    const titleLinkMatch = html.match(
      /<a\b[^>]*class="[^"]*(?:ac-algo|fz-ms|d-ib)[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/i
    );
    if (titleLinkMatch) {
      url = decodeHtmlEntities(titleLinkMatch[1]);
      title = extractText(titleLinkMatch[2]).trim();
    } else {
      // Alternative: find first meaningful link
      const linkMatch = html.match(
        /<a\b[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/i
      );
      if (linkMatch) {
        url = decodeHtmlEntities(linkMatch[1]);
        title = extractText(linkMatch[2]).trim();
      }
    }

    // Handle h3 inside the element
    if (!title) {
      const h3Match = html.match(/<h3[^>]*>([\s\S]*?)<\/h3>/i);
      if (h3Match) {
        title = extractText(h3Match[1]).trim();
      }
    }

    if (!url || !title) return null;

    // Unwrap Yahoo redirect URLs
    if (url.includes('yahoo.com') || url.includes('/RU=')) {
      url = unwrapYahooUrl(url);
    }

    // Skip internal Yahoo links and non-http URLs
    if (!url.startsWith('http') || url.includes('yahoo.com/search')) {
      return null;
    }

    // Find content/description
    const snippetPatterns = [
      /<span[^>]*class="[^"]*(?:fc-falcon|s-desc)[^"]*"[^>]*>([\s\S]*?)<\/span>/i,
      /<p[^>]*class="[^"]*(?:lh-1|s-desc)[^"]*"[^>]*>([\s\S]*?)<\/p>/i,
      /<div[^>]*class="[^"]*compText[^"]*"[^>]*>([\s\S]*?)<\/div>/i,
    ];

    for (const pattern of snippetPatterns) {
      const match = html.match(pattern);
      if (match) {
        const text = extractText(match[1]).trim();
        if (text.length > content.length) {
          content = text;
        }
      }
    }

    return {
      url,
      title,
      content,
      engine: this.name,
      score: this.weight,
      category: 'general',
    };
  }
}
