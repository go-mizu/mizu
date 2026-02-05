/**
 * Startpage Search Engine adapter.
 * Startpage is a privacy-focused search engine that proxies Google results.
 * https://www.startpage.com
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

const STARTPAGE_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// ========== Time Range Maps ==========

const timeRangeMap: Record<string, string> = {
  day: 'd',
  week: 'w',
  month: 'm',
  year: 'y',
};

// ========== StartpageEngine ==========

export class StartpageEngine implements OnlineEngine {
  name = 'startpage';
  shortcut = 'sp';
  categories: Category[] = ['general'];
  supportsPaging = true;
  maxPage = 20;
  timeout = 12_000;
  weight = 0.95; // High weight since it uses Google's index
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('query', query);
    searchParams.set('cat', 'web'); // Category: web search

    // Pagination - Startpage uses 'page' parameter (1-indexed)
    if (params.page > 1) {
      searchParams.set('page', params.page.toString());
    }

    // Time range filter - uses 'with_date' parameter
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('with_date', timeRangeMap[params.timeRange]);
    }

    // Locale/language settings
    const locale = params.locale || 'en-US';
    const parts = locale.split('-');
    const lang = (parts[0] || 'en').toLowerCase();
    const region = (parts[1] || 'us').toLowerCase();

    // Set language
    searchParams.set('language', lang);

    // Set region/locale for UI
    searchParams.set('lui', lang);
    searchParams.set('sc', this.getStartpageLocale(locale));

    // Safe search - Startpage uses 'qadf' parameter
    // Note: Startpage doesn't have explicit safe search toggle in API,
    // but we can use prfh (preferences hash) cookie for this

    return {
      url: `https://www.startpage.com/sp/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': STARTPAGE_USER_AGENT,
        Accept:
          'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'Accept-Language': `${lang}-${region.toUpperCase()},${lang};q=0.9,en;q=0.8`,
        DNT: '1',
        'Upgrade-Insecure-Requests': '1',
      },
      cookies: this.buildCookies(params),
    };
  }

  /**
   * Get Startpage locale code from standard locale.
   */
  private getStartpageLocale(locale: string): string {
    const parts = locale.split('-');
    const lang = (parts[0] || 'en').toLowerCase();
    const region = (parts[1] || 'us').toLowerCase();

    // Startpage uses format like "XuR9hBLzJN1VN" for preferences
    // For simplicity, we'll use the region code
    const regionMap: Record<string, string> = {
      us: 'en-US',
      gb: 'en-GB',
      uk: 'en-GB',
      de: 'de-DE',
      fr: 'fr-FR',
      es: 'es-ES',
      it: 'it-IT',
      nl: 'nl-NL',
      pt: 'pt-PT',
      pl: 'pl-PL',
      ru: 'ru-RU',
      jp: 'ja-JP',
      cn: 'zh-CN',
    };

    return regionMap[region] || `${lang}-${region.toUpperCase()}`;
  }

  /**
   * Build cookies for Startpage requests.
   */
  private buildCookies(params: EngineParams): string[] {
    const cookies: string[] = [];

    // Set preferences cookie for safe search and other settings
    // Startpage uses base64-encoded JSON preferences
    // For simplicity, we'll use known preference strings
    if (params.safeSearch === 2) {
      // Strict safe search
      cookies.push('preferences=EAAAiRO7kn');
    } else if (params.safeSearch === 0) {
      // Off
      cookies.push('preferences=EAAAhA');
    }

    return cookies;
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Check for no results or error
    if (
      body.includes('No results found') ||
      body.includes('did not match any documents') ||
      body.includes('error-message')
    ) {
      return results;
    }

    // Check for CAPTCHA
    if (body.includes('captcha') || body.includes('g-recaptcha')) {
      return results;
    }

    // Startpage results are in <div class="w-gl__result"> elements
    const resultElements = findElements(body, 'div.w-gl__result');

    for (const el of resultElements) {
      const result = this.parseResult(el);
      if (result) {
        results.results.push(result);
      }
    }

    // Fallback: try result-title class
    if (results.results.length === 0) {
      const resultTitles = findElements(body, 'div.result');
      for (const el of resultTitles) {
        const result = this.parseResult(el);
        if (result) {
          results.results.push(result);
        }
      }
    }

    // Another fallback: search for result patterns
    if (results.results.length === 0) {
      this.parseFallbackResults(body, results);
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

    // Also check for suggest2 class
    const suggestElements = findElements(body, 'div.suggest2');
    for (const el of suggestElements) {
      const linkPattern = /<a[^>]*>([^<]+)<\/a>/gi;
      let m: RegExpExecArray | null;
      while ((m = linkPattern.exec(el)) !== null) {
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

    // Startpage result structure:
    // <a class="w-gl__result-url" href="...">
    // <h3 class="w-gl__result-title">Title</h3>
    // <p class="w-gl__description">Description</p>

    // Find title in h3.w-gl__result-title
    const titleElements = findElements(html, 'h3.w-gl__result-title');
    if (titleElements.length > 0) {
      title = extractText(titleElements[0]).trim();
    }

    // Find URL in a.w-gl__result-url
    const urlElements = findElements(html, 'a.w-gl__result-url');
    if (urlElements.length > 0) {
      const hrefMatch = urlElements[0].match(/href="([^"]+)"/i);
      if (hrefMatch) {
        url = decodeHtmlEntities(hrefMatch[1]);
      }
    }

    // Fallback: find any link with title
    if (!url || !title) {
      const linkMatch = html.match(
        /<a[^>]*href="(https?:\/\/[^"]+)"[^>]*>([\s\S]*?)<\/a>/i
      );
      if (linkMatch) {
        if (!url) {
          url = decodeHtmlEntities(linkMatch[1]);
        }
        if (!title) {
          // Check if there's an h3 inside
          const h3Match = linkMatch[2].match(/<h3[^>]*>([\s\S]*?)<\/h3>/i);
          if (h3Match) {
            title = extractText(h3Match[1]).trim();
          } else {
            title = extractText(linkMatch[2]).trim();
          }
        }
      }
    }

    // Another title fallback
    if (!title) {
      const h3Match = html.match(/<h3[^>]*>([\s\S]*?)<\/h3>/i);
      if (h3Match) {
        title = extractText(h3Match[1]).trim();
      }
    }

    if (!url || !title) return null;

    // Unwrap Startpage proxy URLs if present (do this BEFORE checking for internal links)
    if (url.includes('startpage.com/do/proxy') || url.includes('/proxy?') || url.includes('proxy?u=')) {
      const realUrlMatch = url.match(/[?&]u=([^&]+)/);
      if (realUrlMatch) {
        try {
          url = decodeURIComponent(realUrlMatch[1]);
        } catch {
          // Keep original URL
        }
      }
    }

    // Skip Startpage internal links (after unwrapping proxy URLs)
    if (url.includes('startpage.com') || url.includes('startmail.com')) {
      return null;
    }

    // Find description in p.w-gl__description
    const descElements = findElements(html, 'p.w-gl__description');
    if (descElements.length > 0) {
      content = extractText(descElements[0]).trim();
    }

    // Fallback: find any p that looks like content
    if (!content) {
      const pMatch = html.match(/<p[^>]*class="[^"]*(?:desc|snippet|text)[^"]*"[^>]*>([\s\S]*?)<\/p>/i);
      if (pMatch) {
        content = extractText(pMatch[1]).trim();
      }
    }

    // Another content fallback
    if (!content) {
      const pMatch = html.match(/<p[^>]*>([^<]{30,})<\/p>/i);
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

  private parseFallbackResults(body: string, results: EngineResults): void {
    // Find result blocks with title and URL pattern
    const resultPattern = /<div[^>]*class="[^"]*result[^"]*"[^>]*>([\s\S]*?)<\/div>\s*<\/div>/gi;
    let match: RegExpExecArray | null;
    const seen = new Set<string>();

    while ((match = resultPattern.exec(body)) !== null) {
      const block = match[1];
      const linkMatch = block.match(/<a[^>]*href="(https?:\/\/(?!startpage)[^"]+)"[^>]*>/i);
      const titleMatch = block.match(/<h[23][^>]*>([\s\S]*?)<\/h[23]>/i);

      if (linkMatch && titleMatch) {
        const url = decodeHtmlEntities(linkMatch[1]);
        if (!seen.has(url) && !url.includes('startpage.com')) {
          seen.add(url);
          results.results.push({
            url,
            title: extractText(titleMatch[1]).trim(),
            content: '',
            engine: this.name,
            score: this.weight,
            category: 'general',
          });
        }
      }
    }

    // Additional fallback: look for result URLs with associated h3
    if (results.results.length === 0) {
      const h3Pattern = /<h3[^>]*>[\s\S]*?<a[^>]*href="(https?:\/\/(?!startpage)[^"]+)"[^>]*>([\s\S]*?)<\/a>/gi;
      while ((match = h3Pattern.exec(body)) !== null) {
        const url = decodeHtmlEntities(match[1]);
        const title = extractText(match[2]).trim();
        if (!seen.has(url) && title.length > 5) {
          seen.add(url);
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
}
