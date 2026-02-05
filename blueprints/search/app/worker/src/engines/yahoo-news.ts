/**
 * Yahoo News Engine.
 * Fetches news from Yahoo News search API.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { decodeHtmlEntities, findElements, extractText, extractAttribute } from '../lib/html-parser';

const YAHOO_NEWS_URL = 'https://news.search.yahoo.com/search';

const YAHOO_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Time range mapping for Yahoo News
const timeRangeMap: Record<string, string> = {
  day: '1d',
  week: '1w',
  month: '1m',
  year: '',
};

export class YahooNewsEngine implements OnlineEngine {
  name = 'yahoo news';
  shortcut = 'yhn';
  categories: Category[] = ['news'];
  supportsPaging = true;
  maxPage = 20;
  timeout = 15_000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('p', query);

    // Pagination (Yahoo uses 'b' offset parameter, 10 results per page)
    if (params.page > 1) {
      const offset = (params.page - 1) * 10 + 1;
      searchParams.set('b', offset.toString());
    }

    // Time range filter
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('age', timeRangeMap[params.timeRange]);
      searchParams.set('btf', timeRangeMap[params.timeRange]);
    }

    return {
      url: `${YAHOO_NEWS_URL}?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': YAHOO_USER_AGENT,
        Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
        'Accept-Language': params.locale || 'en-US,en;q=0.9',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      // Find news article containers
      const newsItems = findElements(body, 'div.NewsArticle');

      // Fallback: try alternative selectors
      const items = newsItems.length > 0
        ? newsItems
        : findElements(body, 'li.ov-a');

      for (const item of items) {
        const result = this.parseNewsItem(item);
        if (result) {
          results.results.push(result);
        }
      }

      // If still no results, try parsing the generic search results format
      if (results.results.length === 0) {
        const searchResults = findElements(body, 'div.dd');
        for (const item of searchResults) {
          const result = this.parseSearchResult(item);
          if (result) {
            results.results.push(result);
          }
        }
      }
    } catch {
      // Parse failed silently
    }

    return results;
  }

  private parseNewsItem(html: string): EngineResults['results'][0] | null {
    // Extract URL from the main link
    const url = extractAttribute(html, 'a', 'href');
    if (!url || url.startsWith('javascript:')) return null;

    // Clean up Yahoo redirect URLs
    const cleanUrl = this.cleanYahooUrl(url);

    // Extract title
    const titleEl = findElements(html, 'h4');
    const title = titleEl.length > 0
      ? extractText(titleEl[0])
      : extractText(findElements(html, 'a')[0] || '');

    if (!title) return null;

    // Extract snippet/description
    const snippetEl = findElements(html, 'p');
    const snippet = snippetEl.length > 0 ? extractText(snippetEl[0]) : '';

    // Extract source
    const sourceEl = findElements(html, 'span.s-source');
    let source = sourceEl.length > 0 ? extractText(sourceEl[0]) : '';

    // Fallback source extraction
    if (!source) {
      const sourceAlt = findElements(html, 'span.source');
      source = sourceAlt.length > 0 ? extractText(sourceAlt[0]) : '';
    }

    // Extract published time
    const timeEl = findElements(html, 'span.s-time');
    let publishedAt = '';
    if (timeEl.length > 0) {
      const timeText = extractText(timeEl[0]);
      publishedAt = this.parseRelativeTime(timeText);
    }

    // Fallback time extraction
    if (!publishedAt) {
      const timeAlt = findElements(html, 'span.fc-2nd');
      if (timeAlt.length > 0) {
        const timeText = extractText(timeAlt[0]);
        publishedAt = this.parseRelativeTime(timeText);
      }
    }

    if (!publishedAt) {
      publishedAt = new Date().toISOString();
    }

    // Extract thumbnail
    const thumbnailUrl = extractAttribute(html, 'img', 'src') || undefined;

    return {
      url: cleanUrl,
      title: decodeHtmlEntities(title),
      content: decodeHtmlEntities(snippet),
      engine: this.name,
      score: this.weight,
      category: 'news',
      template: 'news',
      source: decodeHtmlEntities(source),
      thumbnailUrl: thumbnailUrl && !thumbnailUrl.includes('data:') ? thumbnailUrl : undefined,
      publishedAt,
    };
  }

  private parseSearchResult(html: string): EngineResults['results'][0] | null {
    // Extract URL
    const url = extractAttribute(html, 'a', 'href');
    if (!url || url.startsWith('javascript:')) return null;

    const cleanUrl = this.cleanYahooUrl(url);

    // Extract title from h4 or first anchor
    const titleEl = findElements(html, 'h4');
    let title = titleEl.length > 0 ? extractText(titleEl[0]) : '';
    if (!title) {
      const linkEl = findElements(html, 'a');
      title = linkEl.length > 0 ? extractText(linkEl[0]) : '';
    }
    if (!title) return null;

    // Extract snippet
    const snippetEl = findElements(html, 'p');
    const snippet = snippetEl.length > 0 ? extractText(snippetEl[0]) : '';

    // Try to extract source from cite or span elements
    const citeEl = findElements(html, 'cite');
    let source = citeEl.length > 0 ? extractText(citeEl[0]) : '';
    if (source) {
      // Often the cite contains the full URL, extract domain
      try {
        source = new URL(source.startsWith('http') ? source : `https://${source}`).hostname;
      } catch {
        // Keep as-is
      }
    }

    return {
      url: cleanUrl,
      title: decodeHtmlEntities(title),
      content: decodeHtmlEntities(snippet),
      engine: this.name,
      score: this.weight,
      category: 'news',
      template: 'news',
      source: decodeHtmlEntities(source),
      publishedAt: new Date().toISOString(),
    };
  }

  private cleanYahooUrl(url: string): string {
    // Yahoo often wraps URLs in their redirect service
    // Pattern: https://r.search.yahoo.com/.../**https://actual-url.com...
    const actualUrlMatch = url.match(/\*\*(.+)$/);
    if (actualUrlMatch) {
      try {
        return decodeURIComponent(actualUrlMatch[1]);
      } catch {
        return actualUrlMatch[1];
      }
    }

    // Alternative pattern with RU= parameter
    const ruMatch = url.match(/RU=([^/]+)/);
    if (ruMatch) {
      try {
        return decodeURIComponent(ruMatch[1]);
      } catch {
        return ruMatch[1];
      }
    }

    return url;
  }

  private parseRelativeTime(timeText: string): string {
    const now = new Date();
    const text = timeText.toLowerCase().trim();

    // Match patterns like "5 hours ago", "2 days ago", etc.
    const match = text.match(/(\d+)\s*(second|minute|hour|day|week|month|year)s?\s*ago/i);
    if (match) {
      const value = parseInt(match[1], 10);
      const unit = match[2].toLowerCase();

      switch (unit) {
        case 'second':
          now.setSeconds(now.getSeconds() - value);
          break;
        case 'minute':
          now.setMinutes(now.getMinutes() - value);
          break;
        case 'hour':
          now.setHours(now.getHours() - value);
          break;
        case 'day':
          now.setDate(now.getDate() - value);
          break;
        case 'week':
          now.setDate(now.getDate() - value * 7);
          break;
        case 'month':
          now.setMonth(now.getMonth() - value);
          break;
        case 'year':
          now.setFullYear(now.getFullYear() - value);
          break;
      }
      return now.toISOString();
    }

    // Handle "yesterday"
    if (text.includes('yesterday')) {
      now.setDate(now.getDate() - 1);
      return now.toISOString();
    }

    // Try parsing as a date string
    try {
      const parsed = new Date(timeText);
      if (!isNaN(parsed.getTime())) {
        return parsed.toISOString();
      }
    } catch {
      // Ignore parse errors
    }

    return new Date().toISOString();
  }
}
