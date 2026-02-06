/**
 * Reuters News Engine.
 * Fetches news from Reuters search API.
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

const REUTERS_SEARCH_URL = 'https://www.reuters.com/pf/api/v3/content/fetch/articles-by-search-v2';

const REUTERS_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Results per page
const RESULTS_PER_PAGE = 20;

// Time range in seconds mapping
const timeRangeSeconds: Record<string, number> = {
  day: 86400,
  week: 604800,
  month: 2592000,
  year: 31536000,
};

interface ReutersArticle {
  id?: string;
  canonical_url?: string;
  title?: string;
  description?: string;
  published_time?: string;
  updated_time?: string;
  authors?: Array<{ name?: string }>;
  thumbnail?: {
    url?: string;
    width?: number;
    height?: number;
  };
  source?: {
    name?: string;
  };
  kicker?: {
    name?: string;
  };
}

interface ReutersSearchResponse {
  result?: {
    articles?: ReutersArticle[];
  };
  statusCode?: number;
}

export class ReutersEngine implements OnlineEngine {
  name = 'reuters';
  shortcut = 'rtr';
  categories: Category[] = ['news'];
  supportsPaging = true;
  maxPage = 20;
  timeout = 15_000;
  weight = 1.1; // Higher weight for authoritative news source
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const offset = (params.page - 1) * RESULTS_PER_PAGE;

    // Build the query object for Reuters API
    const queryObj: Record<string, unknown> = {
      keyword: query,
      offset,
      size: RESULTS_PER_PAGE,
      sort: 'relevance',
      website: 'reuters',
    };

    // Time range filter
    if (params.timeRange && timeRangeSeconds[params.timeRange]) {
      const now = new Date();
      const startDate = new Date(now.getTime() - timeRangeSeconds[params.timeRange] * 1000);
      queryObj.date_range = {
        start_date: startDate.toISOString().split('T')[0],
        end_date: now.toISOString().split('T')[0],
      };
    }

    const encodedQuery = encodeURIComponent(JSON.stringify(queryObj));

    return {
      url: `${REUTERS_SEARCH_URL}?query=${encodedQuery}&_website=reuters`,
      method: 'GET',
      headers: {
        'User-Agent': REUTERS_USER_AGENT,
        Accept: 'application/json',
        'Accept-Language': params.locale || 'en-US,en;q=0.9',
        Referer: 'https://www.reuters.com/',
        Origin: 'https://www.reuters.com',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as ReutersSearchResponse;

      if (!data.result?.articles || !Array.isArray(data.result.articles)) {
        return results;
      }

      for (const article of data.result.articles) {
        if (!article.canonical_url || !article.title) continue;

        // Build full URL
        const url = article.canonical_url.startsWith('http')
          ? article.canonical_url
          : `https://www.reuters.com${article.canonical_url}`;

        // Parse published time
        const publishedAt = article.published_time
          ? new Date(article.published_time).toISOString()
          : new Date().toISOString();

        // Get thumbnail URL
        let thumbnailUrl: string | undefined;
        if (article.thumbnail?.url) {
          thumbnailUrl = article.thumbnail.url;
        }

        // Get source - typically Reuters, but can have kicker/category
        const source = article.kicker?.name || 'Reuters';

        // Extract author names
        const authorNames = article.authors?.map(a => a.name).filter(Boolean) as string[] | undefined;
        const author = authorNames?.join(', ') || undefined;

        results.results.push({
          url,
          title: decodeHtmlEntities(article.title),
          content: decodeHtmlEntities(article.description || ''),
          engine: this.name,
          score: this.weight,
          category: 'news',
          template: 'news',
          source,
          thumbnailUrl,
          publishedAt,
          author,
          metadata: {
            articleId: article.id,
            authors: authorNames,
            updatedAt: article.updated_time,
          },
        });
      }
    } catch {
      // JSON parse failed - try fallback HTML parsing
      return this.parseHtmlFallback(body);
    }

    return results;
  }

  /**
   * Fallback HTML parser in case the API returns HTML instead of JSON.
   */
  private parseHtmlFallback(body: string): EngineResults {
    const results = newEngineResults();

    try {
      // Look for JSON-LD structured data
      const jsonLdMatch = body.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/g);
      if (jsonLdMatch) {
        for (const match of jsonLdMatch) {
          const jsonContent = match.replace(/<script[^>]*>/, '').replace(/<\/script>/, '');
          try {
            const ldData = JSON.parse(jsonContent);
            if (ldData['@type'] === 'NewsArticle' || ldData['@type'] === 'Article') {
              const url = ldData.url || ldData.mainEntityOfPage?.['@id'];
              if (url) {
                results.results.push({
                  url,
                  title: decodeHtmlEntities(ldData.headline || ''),
                  content: decodeHtmlEntities(ldData.description || ''),
                  engine: this.name,
                  score: this.weight,
                  category: 'news',
                  template: 'news',
                  source: ldData.publisher?.name || 'Reuters',
                  thumbnailUrl: ldData.image?.url || ldData.thumbnailUrl,
                  publishedAt: ldData.datePublished
                    ? new Date(ldData.datePublished).toISOString()
                    : new Date().toISOString(),
                });
              }
            }
          } catch {
            // Individual JSON-LD block parse failed
          }
        }
      }
    } catch {
      // HTML fallback failed
    }

    return results;
  }
}

/**
 * Reuters RSS Engine for specific topics.
 * Uses Reuters RSS feeds for category-based news.
 */
export class ReutersRSSEngine implements OnlineEngine {
  name = 'reuters rss';
  shortcut = 'rtrss';
  categories: Category[] = ['news'];
  supportsPaging = false; // RSS doesn't support pagination
  maxPage = 1;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  private topic: string;

  private static readonly TOPICS: Record<string, string> = {
    world: 'world',
    business: 'business',
    technology: 'technology',
    markets: 'markets',
    politics: 'politics',
    science: 'science',
    health: 'health',
    sports: 'sports',
    entertainment: 'lifestyle',
  };

  constructor(topic = 'world') {
    this.topic = topic;
    this.name = `reuters rss (${topic})`;
  }

  buildRequest(_query: string, params: EngineParams): RequestConfig {
    const topicSlug = ReutersRSSEngine.TOPICS[this.topic] || this.topic;

    return {
      url: `https://www.reuters.com/rssfeed/${topicSlug}`,
      method: 'GET',
      headers: {
        'User-Agent': REUTERS_USER_AGENT,
        Accept: 'application/rss+xml, application/xml, text/xml',
        'Accept-Language': params.locale || 'en-US,en;q=0.9',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      // Simple RSS item extraction
      const itemRegex = /<item>([\s\S]*?)<\/item>/gi;
      let match: RegExpExecArray | null;

      while ((match = itemRegex.exec(body)) !== null) {
        const itemXml = match[1];

        // Extract fields using simple regex
        const title = this.extractTag(itemXml, 'title');
        const link = this.extractTag(itemXml, 'link');
        const description = this.extractTag(itemXml, 'description');
        const pubDate = this.extractTag(itemXml, 'pubDate');

        if (!link || !title) continue;

        // Extract thumbnail from media:content or enclosure
        let thumbnailUrl: string | undefined;
        const mediaMatch = itemXml.match(/<media:content[^>]+url=["']([^"']+)["']/i);
        if (mediaMatch) {
          thumbnailUrl = mediaMatch[1];
        } else {
          const enclosureMatch = itemXml.match(/<enclosure[^>]+url=["']([^"']+)["']/i);
          if (enclosureMatch) {
            thumbnailUrl = enclosureMatch[1];
          }
        }

        const publishedAt = pubDate
          ? new Date(pubDate).toISOString()
          : new Date().toISOString();

        results.results.push({
          url: link,
          title: decodeHtmlEntities(title),
          content: decodeHtmlEntities(description.replace(/<[^>]+>/g, '')),
          engine: this.name,
          score: this.weight,
          category: 'news',
          template: 'news',
          source: 'Reuters',
          thumbnailUrl,
          publishedAt,
          metadata: {
            topic: this.topic,
          },
        });
      }
    } catch {
      // RSS parse failed
    }

    return results;
  }

  private extractTag(xml: string, tagName: string): string {
    const match = xml.match(new RegExp(`<${tagName}[^>]*><!\\[CDATA\\[([\\s\\S]*?)\\]\\]><\\/${tagName}>`, 'i'))
      || xml.match(new RegExp(`<${tagName}[^>]*>([\\s\\S]*?)<\\/${tagName}>`, 'i'));
    return match ? match[1].trim() : '';
  }
}
