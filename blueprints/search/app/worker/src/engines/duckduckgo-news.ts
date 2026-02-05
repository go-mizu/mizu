/**
 * DuckDuckGo News Engine.
 * Fetches news from DuckDuckGo's news search.
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

const DDG_NEWS_URL = 'https://duckduckgo.com/news.js';

const DDG_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Time range mapping for DuckDuckGo
const timeRangeMap: Record<string, string> = {
  day: 'd',
  week: 'w',
  month: 'm',
};

interface DDGNewsResult {
  date: number; // Unix timestamp
  excerpt: string;
  image?: string;
  relative_time: string;
  source: string;
  title: string;
  url: string;
}

interface DDGNewsResponse {
  results?: DDGNewsResult[];
}

export class DuckDuckGoNewsEngine implements OnlineEngine {
  name = 'duckduckgo news';
  shortcut = 'ddgn';
  categories: Category[] = ['news'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('l', params.locale || 'us-en');
    searchParams.set('o', 'json');
    searchParams.set('noamp', '1');
    searchParams.set('df', ''); // Date filter

    // Pagination (DDG uses 's' offset parameter)
    if (params.page > 1) {
      const offset = (params.page - 1) * 30;
      searchParams.set('s', offset.toString());
    }

    // Time range
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('df', timeRangeMap[params.timeRange]);
    }

    // Safe search
    if (params.safeSearch === 0) {
      searchParams.set('kp', '-2'); // Off
    } else if (params.safeSearch === 2) {
      searchParams.set('kp', '1'); // Strict
    } else {
      searchParams.set('kp', '-1'); // Moderate
    }

    return {
      url: `${DDG_NEWS_URL}?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': DDG_USER_AGENT,
        Accept: 'application/json',
        'Accept-Language': 'en-US,en;q=0.9',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as DDGNewsResponse;

      if (!data.results || !Array.isArray(data.results)) {
        return results;
      }

      for (const item of data.results) {
        if (!item.url || !item.title) continue;

        // Convert Unix timestamp to ISO string
        const publishedAt = item.date
          ? new Date(item.date * 1000).toISOString()
          : new Date().toISOString();

        results.results.push({
          url: item.url,
          title: decodeHtmlEntities(item.title),
          content: decodeHtmlEntities(item.excerpt || ''),
          engine: this.name,
          score: this.weight,
          category: 'news',
          template: 'news',
          source: item.source || '',
          thumbnailUrl: item.image,
          publishedAt,
        });
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }
}
