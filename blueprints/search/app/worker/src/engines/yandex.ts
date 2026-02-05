/**
 * Yandex Search Engine adapter.
 * Parses Yandex web search results from HTML.
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

const YANDEX_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// ========== Time Range Maps ==========

const timeRangeMap: Record<string, string> = {
  day: '77',   // Last 24 hours
  week: '1',   // Last week
  month: '2',  // Last month
  year: '3',   // Last year
};

// ========== Safe Search Maps ==========

const safeSearchMap: Record<number, string> = {
  0: '0',   // Off
  1: '1',   // Moderate
  2: '2',   // Strict
};

// ========== YandexEngine ==========

export class YandexEngine implements OnlineEngine {
  name = 'yandex';
  shortcut = 'ya';
  categories: Category[] = ['general'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('text', query);

    // Pagination - Yandex uses 'p' parameter (0-indexed page number)
    if (params.page > 1) {
      searchParams.set('p', (params.page - 1).toString());
    }

    // Time range filter
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('within', timeRangeMap[params.timeRange]);
      // For custom time ranges, Yandex uses 'from_day', 'from_month', etc.
    }

    // Safe search (family filter)
    const safeValue = safeSearchMap[params.safeSearch];
    if (safeValue !== undefined) {
      searchParams.set('family', safeValue);
    }

    // Locale/region
    const locale = params.locale || 'en-US';
    const parts = locale.split('-');
    const lang = (parts[0] || 'en').toLowerCase();

    // Set language parameter
    searchParams.set('lr', this.getRegionCode(locale));

    // Use English interface for international users
    if (lang !== 'ru') {
      searchParams.set('lang', lang);
    }

    // Request more results per page
    searchParams.set('numdoc', '10');

    return {
      url: `https://yandex.com/search/?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': YANDEX_USER_AGENT,
        Accept:
          'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'Accept-Language': `${lang},en;q=0.9`,
        DNT: '1',
      },
      cookies: [],
    };
  }

  /**
   * Get Yandex region code from locale.
   * Region codes determine search locality.
   */
  private getRegionCode(locale: string): string {
    const parts = locale.split('-');
    const region = (parts[1] || 'us').toLowerCase();

    const regionCodes: Record<string, string> = {
      us: '84',   // USA
      gb: '102',  // UK
      uk: '102',  // UK
      de: '96',   // Germany
      fr: '124',  // France
      ru: '225',  // Russia
      ua: '187',  // Ukraine
      by: '149',  // Belarus
      kz: '159',  // Kazakhstan
      tr: '983',  // Turkey
    };

    return regionCodes[region] || '84'; // Default to USA
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Check for CAPTCHA
    if (
      body.includes('captcha') ||
      body.includes('showcaptcha') ||
      body.includes('robot')
    ) {
      return results;
    }

    // Yandex results are in li.serp-item elements
    const serpItems = findElements(body, 'li.serp-item');

    for (const item of serpItems) {
      const result = this.parseResult(item);
      if (result) {
        results.results.push(result);
      }
    }

    // Fallback: try organic results with different class
    if (results.results.length === 0) {
      const organicItems = findElements(body, 'div.organic');
      for (const item of organicItems) {
        const result = this.parseResult(item);
        if (result) {
          results.results.push(result);
        }
      }
    }

    // Try another fallback pattern for newer Yandex layout
    if (results.results.length === 0) {
      this.parseFallbackResults(body, results);
    }

    // Parse related searches
    const relatedElements = findElements(body, 'div.misspell');
    for (const el of relatedElements) {
      const text = extractText(el).trim();
      if (text && text.length < 100) {
        results.suggestions.push(text);
      }
    }

    // Related queries in suggest block
    const suggestElements = findElements(body, 'li.suggest2-item');
    for (const el of suggestElements) {
      const text = extractText(el).trim();
      if (text && text.length < 100 && !results.suggestions.includes(text)) {
        results.suggestions.push(text);
      }
    }

    return results;
  }

  private parseResult(html: string): EngineResults['results'][number] | null {
    let url = '';
    let title = '';
    let content = '';

    // Find title in h2 with OrganicTitle class or similar
    const titleElements = findElements(html, 'h2.OrganicTitle');
    if (titleElements.length > 0) {
      const linkMatch = titleElements[0].match(
        /<a\b[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/i
      );
      if (linkMatch) {
        url = decodeHtmlEntities(linkMatch[1]);
        title = extractText(linkMatch[2]).trim();
      }
    }

    // Fallback: find title in data-cid element
    if (!title) {
      const cidMatch = html.match(/data-cid="[^"]+"/);
      if (cidMatch) {
        const linkMatch = html.match(
          /<a\b[^>]*class="[^"]*(?:Link|OrganicTitle)[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/i
        );
        if (linkMatch) {
          url = decodeHtmlEntities(linkMatch[1]);
          title = extractText(linkMatch[2]).trim();
        }
      }
    }

    // Another fallback: look for path element which contains URL
    if (!url) {
      const pathElements = findElements(html, 'div.Path');
      if (pathElements.length > 0) {
        const linkMatch = pathElements[0].match(/href="([^"]+)"/i);
        if (linkMatch) {
          url = decodeHtmlEntities(linkMatch[1]);
        }
      }
    }

    // Get title from any h2 if not found
    if (!title) {
      const h2Match = html.match(/<h2[^>]*>([\s\S]*?)<\/h2>/i);
      if (h2Match) {
        const linkMatch = h2Match[1].match(/<a[^>]*>([\s\S]*?)<\/a>/i);
        if (linkMatch) {
          title = extractText(linkMatch[1]).trim();
        } else {
          title = extractText(h2Match[1]).trim();
        }
      }
    }

    // Get URL from first <a> with http if not found
    if (!url) {
      const linkMatch = html.match(/<a\b[^>]*href="(https?:\/\/[^"]+)"/i);
      if (linkMatch) {
        url = decodeHtmlEntities(linkMatch[1]);
      }
    }

    if (!url || !title) return null;

    // Skip Yandex internal links
    if (url.includes('yandex.') || url.includes('yabs.')) {
      return null;
    }

    // Find content/snippet
    const snippetPatterns = [
      /<div[^>]*class="[^"]*(?:OrganicText|TextContainer|text-container)[^"]*"[^>]*>([\s\S]*?)<\/div>/i,
      /<span[^>]*class="[^"]*(?:OrganicText|extended-text)[^"]*"[^>]*>([\s\S]*?)<\/span>/i,
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

  private parseFallbackResults(body: string, results: EngineResults): void {
    // Try to find results using URL pattern and surrounding context
    const urlPattern = /<a[^>]*href="(https?:\/\/(?!yandex\.)[^"]+)"[^>]*class="[^"]*(?:link|organic)[^"]*"[^>]*>([\s\S]*?)<\/a>/gi;
    let match: RegExpExecArray | null;
    const seen = new Set<string>();

    while ((match = urlPattern.exec(body)) !== null) {
      const url = decodeHtmlEntities(match[1]);
      if (seen.has(url) || url.includes('yandex.')) continue;
      seen.add(url);

      const title = extractText(match[2]).trim();
      if (title && title.length > 5) {
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
