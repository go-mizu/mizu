/**
 * Mojeek Search Engine adapter.
 * Mojeek is an independent UK-based search engine with its own crawler and index.
 * https://www.mojeek.com
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

const MOJEEK_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// ========== Time Range Maps ==========

const timeRangeMap: Record<string, string> = {
  day: 'day',
  week: 'week',
  month: 'month',
  year: 'year',
};

// ========== Safe Search Maps ==========

const safeSearchMap: Record<number, string> = {
  0: '0',   // Off
  1: '1',   // Moderate (default)
  2: '2',   // Strict
};

// ========== MojeekEngine ==========

export class MojeekEngine implements OnlineEngine {
  name = 'mojeek';
  shortcut = 'mj';
  categories: Category[] = ['general'];
  supportsPaging = true;
  maxPage = 100;
  timeout = 10_000;
  weight = 0.8;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);

    // Pagination - Mojeek uses 's' parameter for start offset (0-indexed, 10 per page)
    if (params.page > 1) {
      const offset = (params.page - 1) * 10;
      searchParams.set('s', offset.toString());
    }

    // Time range filter
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('date', timeRangeMap[params.timeRange]);
    }

    // Safe search
    const safeValue = safeSearchMap[params.safeSearch];
    if (safeValue !== undefined) {
      searchParams.set('safe', safeValue);
    }

    // Language/region filtering
    const locale = params.locale || 'en-US';
    const parts = locale.split('-');
    const lang = (parts[0] || 'en').toLowerCase();
    const region = (parts[1] || '').toLowerCase();

    // Mojeek supports language filtering
    if (lang && lang !== 'all') {
      searchParams.set('lb', lang); // Language bias
    }

    // Regional bias (country code)
    if (region) {
      searchParams.set('arc', region.toUpperCase()); // Region code
    }

    // Request format - plain HTML
    searchParams.set('fmt', 'html');

    return {
      url: `https://www.mojeek.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': MOJEEK_USER_AGENT,
        Accept:
          'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'Accept-Language': `${lang},en;q=0.9`,
        DNT: '1',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Check for no results message
    if (
      body.includes('No results found') ||
      body.includes('did not match any documents')
    ) {
      return results;
    }

    // Mojeek results are in ul.results-standard > li elements
    const resultItems = findElements(body, 'li.result');

    for (const item of resultItems) {
      const result = this.parseResult(item);
      if (result) {
        results.results.push(result);
      }
    }

    // Fallback: try class results-standard children
    if (results.results.length === 0) {
      const standardResults = findElements(body, 'ul.results-standard');
      if (standardResults.length > 0) {
        const liPattern = /<li[^>]*>([\s\S]*?)<\/li>/gi;
        let match: RegExpExecArray | null;
        while ((match = liPattern.exec(standardResults[0])) !== null) {
          const result = this.parseResult(match[1]);
          if (result) {
            results.results.push(result);
          }
        }
      }
    }

    // Another fallback using search-results div
    if (results.results.length === 0) {
      const searchResultsDiv = findElements(body, 'div.search-results');
      if (searchResultsDiv.length > 0) {
        this.parseFallbackResults(searchResultsDiv[0], results);
      }
    }

    // Parse related searches
    const relatedElements = findElements(body, 'div.related-searches');
    for (const el of relatedElements) {
      const linkPattern = /<a[^>]*>([^<]+)<\/a>/gi;
      let m: RegExpExecArray | null;
      while ((m = linkPattern.exec(el)) !== null) {
        const term = extractText(m[1]).trim();
        if (term && term.length < 100 && !results.suggestions.includes(term)) {
          results.suggestions.push(term);
        }
      }
    }

    // Also check for "Also try" section
    const alsoTryMatch = body.match(/Also try:[\s\S]*?<\/div>/i);
    if (alsoTryMatch) {
      const linkPattern = /<a[^>]*>([^<]+)<\/a>/gi;
      let m: RegExpExecArray | null;
      while ((m = linkPattern.exec(alsoTryMatch[0])) !== null) {
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

    // Mojeek result structure:
    // <a class="ob" href="..."><h2>Title</h2></a>
    // <p class="s">Description</p>
    // <p class="i">URL display</p>

    // Find title and URL in the anchor with class "ob" or containing h2
    const titleLinkMatch = html.match(
      /<a[^>]*class="[^"]*ob[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/i
    );
    if (titleLinkMatch) {
      url = decodeHtmlEntities(titleLinkMatch[1]);
      // Title might be in h2 inside the link
      const h2Match = titleLinkMatch[2].match(/<h2[^>]*>([\s\S]*?)<\/h2>/i);
      if (h2Match) {
        title = extractText(h2Match[1]).trim();
      } else {
        title = extractText(titleLinkMatch[2]).trim();
      }
    }

    // Fallback: look for any link with h2
    if (!url || !title) {
      const h2LinkMatch = html.match(
        /<a[^>]*href="(https?:\/\/[^"]+)"[^>]*>[\s\S]*?<h2[^>]*>([\s\S]*?)<\/h2>/i
      );
      if (h2LinkMatch) {
        url = decodeHtmlEntities(h2LinkMatch[1]);
        title = extractText(h2LinkMatch[2]).trim();
      }
    }

    // Another fallback: separate h2 and link
    if (!title) {
      const h2Match = html.match(/<h2[^>]*>([\s\S]*?)<\/h2>/i);
      if (h2Match) {
        title = extractText(h2Match[1]).trim();
      }
    }

    if (!url) {
      const linkMatch = html.match(/<a[^>]*href="(https?:\/\/[^"]+)"/i);
      if (linkMatch) {
        url = decodeHtmlEntities(linkMatch[1]);
      }
    }

    if (!url || !title) return null;

    // Skip Mojeek internal links
    if (url.includes('mojeek.com')) {
      return null;
    }

    // Find content/snippet in p.s or similar
    const snippetPatterns = [
      /<p[^>]*class="[^"]*s[^"]*"[^>]*>([\s\S]*?)<\/p>/i,
      /<p[^>]*class="[^"]*snippet[^"]*"[^>]*>([\s\S]*?)<\/p>/i,
      /<span[^>]*class="[^"]*desc[^"]*"[^>]*>([\s\S]*?)<\/span>/i,
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

    // Fallback: any p tag that looks like content
    if (!content) {
      const pMatch = html.match(/<p[^>]*>([^<]{20,})<\/p>/i);
      if (pMatch) {
        content = extractText(pMatch[1]).trim();
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

  private parseFallbackResults(html: string, results: EngineResults): void {
    // Find all links that look like search results
    const linkPattern = /<a[^>]*href="(https?:\/\/(?!mojeek\.com)[^"]+)"[^>]*>([\s\S]*?)<\/a>/gi;
    let match: RegExpExecArray | null;
    const seen = new Set<string>();

    while ((match = linkPattern.exec(html)) !== null) {
      const url = decodeHtmlEntities(match[1]);
      if (seen.has(url)) continue;

      const titleRaw = extractText(match[2]).trim();
      // Filter out navigation links and short text
      if (titleRaw.length > 10 && !titleRaw.includes('Next') && !titleRaw.includes('Previous')) {
        seen.add(url);
        results.results.push({
          url,
          title: titleRaw,
          content: '',
          engine: this.name,
          score: this.weight,
          category: 'general',
        });
      }
    }
  }
}
