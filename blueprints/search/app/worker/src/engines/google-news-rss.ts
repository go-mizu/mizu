/**
 * Google News RSS Engine.
 * Fetches news from Google News RSS feeds for various categories and topics.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import {
  getElementsByTagName,
  getTextContent,
  getElementAttribute,
} from '../lib/xml-parser';
import { decodeHtmlEntities } from '../lib/html-parser';
import type { NewsCategory } from '../types';

// Google News RSS feed base URL
const GNEWS_RSS_BASE = 'https://news.google.com/rss';

// Category to Google News topic mapping
const CATEGORY_TOPICS: Record<NewsCategory, string> = {
  top: 'headlines',
  world: 'WORLD',
  nation: 'NATION',
  business: 'BUSINESS',
  technology: 'TECHNOLOGY',
  science: 'SCIENCE',
  health: 'HEALTH',
  sports: 'SPORTS',
  entertainment: 'ENTERTAINMENT',
};

// User agent for Google News requests
const GNEWS_USER_AGENT =
  'Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)';

/**
 * Build Google News RSS URL for a category.
 */
export function buildCategoryFeedUrl(
  category: NewsCategory,
  language = 'en',
  region = 'US'
): string {
  const topic = CATEGORY_TOPICS[category];
  if (category === 'top') {
    return `${GNEWS_RSS_BASE}?hl=${language}&gl=${region}&ceid=${region}:${language}`;
  }
  return `${GNEWS_RSS_BASE}/topics/${topic}?hl=${language}&gl=${region}&ceid=${region}:${language}`;
}

/**
 * Build Google News RSS URL for search.
 */
export function buildSearchFeedUrl(
  query: string,
  language = 'en',
  region = 'US'
): string {
  const encodedQuery = encodeURIComponent(query);
  return `${GNEWS_RSS_BASE}/search?q=${encodedQuery}&hl=${language}&gl=${region}&ceid=${region}:${language}`;
}

/**
 * Build Google News RSS URL for a specific topic/cluster.
 */
export function buildTopicFeedUrl(
  topicId: string,
  language = 'en',
  region = 'US'
): string {
  return `${GNEWS_RSS_BASE}/stories/${topicId}?hl=${language}&gl=${region}&ceid=${region}:${language}`;
}

/**
 * Build Google News RSS URL for local news.
 */
export function buildLocalFeedUrl(
  location: string,
  language = 'en',
  region = 'US'
): string {
  const encodedLocation = encodeURIComponent(location);
  return `${GNEWS_RSS_BASE}/search?q=${encodedLocation}&hl=${language}&gl=${region}&ceid=${region}:${language}`;
}

/**
 * Parse a Google News RSS item.
 */
interface ParsedNewsItem {
  url: string;
  title: string;
  source: string;
  sourceUrl: string;
  snippet: string;
  publishedAt: string;
  imageUrl?: string;
  clusterId?: string;
}

function parseRssItem(itemXml: string): ParsedNewsItem | null {
  // Get title - remove source suffix if present
  let title = getTextContent(itemXml, 'title');
  let source = '';

  // Google News titles often end with " - Source Name"
  const sourceSeparator = title.lastIndexOf(' - ');
  if (sourceSeparator > 0) {
    source = title.slice(sourceSeparator + 3).trim();
    title = title.slice(0, sourceSeparator).trim();
  }

  // Get link
  const link = getTextContent(itemXml, 'link');
  if (!link) return null;

  // Get publication date
  const pubDate = getTextContent(itemXml, 'pubDate');

  // Get description (often contains more source info and snippet)
  let snippet = '';
  const description = getTextContent(itemXml, 'description');
  if (description) {
    // Description often contains HTML with source links
    // Extract plain text
    snippet = description.replace(/<[^>]+>/g, '').trim();
    // Limit snippet length
    if (snippet.length > 300) {
      snippet = snippet.slice(0, 297) + '...';
    }
  }

  // Try to get source from source element or dc:creator
  if (!source) {
    source = getTextContent(itemXml, 'source') || getTextContent(itemXml, 'dc:creator') || '';
  }

  // Get source URL from source element's url attribute
  const sourceUrls = getElementAttribute(itemXml, 'source', 'url');
  const sourceUrl = sourceUrls[0] || '';

  // Try to extract image from media:content or enclosure
  let imageUrl: string | undefined;
  const mediaUrls = getElementAttribute(itemXml, 'media:content', 'url');
  if (mediaUrls.length > 0) {
    imageUrl = mediaUrls[0];
  } else {
    const enclosureUrls = getElementAttribute(itemXml, 'enclosure', 'url');
    if (enclosureUrls.length > 0) {
      imageUrl = enclosureUrls[0];
    }
  }

  // Extract cluster ID from GUID if present (for Full Coverage)
  let clusterId: string | undefined;
  const guid = getTextContent(itemXml, 'guid');
  if (guid) {
    // Google News GUIDs often contain the story cluster ID
    const clusterMatch = guid.match(/stories\/([A-Za-z0-9_-]+)/);
    if (clusterMatch) {
      clusterId = clusterMatch[1];
    }
  }

  return {
    url: link,
    title: decodeHtmlEntities(title),
    source: decodeHtmlEntities(source),
    sourceUrl,
    snippet: decodeHtmlEntities(snippet),
    publishedAt: pubDate ? new Date(pubDate).toISOString() : new Date().toISOString(),
    imageUrl,
    clusterId,
  };
}

/**
 * Parse Google News RSS feed.
 */
export function parseGoogleNewsRss(rssXml: string): ParsedNewsItem[] {
  const items = getElementsByTagName(rssXml, 'item');
  const results: ParsedNewsItem[] = [];

  for (const itemXml of items) {
    const parsed = parseRssItem(itemXml);
    if (parsed) {
      results.push(parsed);
    }
  }

  return results;
}

/**
 * Google News RSS Engine for category feeds.
 */
export class GoogleNewsRSSEngine implements OnlineEngine {
  name = 'google news rss';
  shortcut = 'gnr';
  categories: Category[] = ['news'];
  supportsPaging = false; // RSS doesn't support pagination
  maxPage = 1;
  timeout = 10_000;
  weight = 1.5; // Higher weight for primary source
  disabled = false;

  private newsCategory: NewsCategory = 'top';

  constructor(category: NewsCategory = 'top') {
    this.newsCategory = category;
    this.name = `google news rss (${category})`;
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const language = params.locale?.split('-')[0] || 'en';
    const region = params.locale?.split('-')[1]?.toUpperCase() || 'US';

    let url: string;
    if (query) {
      // Search query
      url = buildSearchFeedUrl(query, language, region);
    } else {
      // Category feed
      url = buildCategoryFeedUrl(this.newsCategory, language, region);
    }

    return {
      url,
      method: 'GET',
      headers: {
        'User-Agent': GNEWS_USER_AGENT,
        Accept: 'application/rss+xml, application/xml, text/xml',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();
    const items = parseGoogleNewsRss(body);

    for (const item of items) {
      const id = this.generateArticleId(item.url);

      results.results.push({
        url: item.url,
        title: item.title,
        content: item.snippet,
        engine: this.name,
        score: this.weight,
        category: 'news',
        template: 'news',
        source: item.source,
        thumbnailUrl: item.imageUrl,
        publishedAt: item.publishedAt,
        // Store extra data for news-specific processing
        metadata: {
          sourceUrl: item.sourceUrl,
          clusterId: item.clusterId,
          articleId: id,
          newsCategory: this.newsCategory,
        },
      });
    }

    return results;
  }

  private generateArticleId(url: string): string {
    // Create a simple hash from URL
    let hash = 0;
    for (let i = 0; i < url.length; i++) {
      const char = url.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(36);
  }
}

/**
 * Google News Topic Engine for Full Coverage.
 * Fetches articles for a specific story cluster.
 */
export class GoogleNewsTopicEngine implements OnlineEngine {
  name = 'google news topic';
  shortcut = 'gnt';
  categories: Category[] = ['news'];
  supportsPaging = false;
  maxPage = 1;
  timeout = 10_000;
  weight = 1.5;
  disabled = false;

  private topicId: string;

  constructor(topicId: string) {
    this.topicId = topicId;
    this.name = `google news topic (${topicId})`;
  }

  buildRequest(_query: string, params: EngineParams): RequestConfig {
    const language = params.locale?.split('-')[0] || 'en';
    const region = params.locale?.split('-')[1]?.toUpperCase() || 'US';

    return {
      url: buildTopicFeedUrl(this.topicId, language, region),
      method: 'GET',
      headers: {
        'User-Agent': GNEWS_USER_AGENT,
        Accept: 'application/rss+xml, application/xml, text/xml',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();
    const items = parseGoogleNewsRss(body);

    for (const item of items) {
      results.results.push({
        url: item.url,
        title: item.title,
        content: item.snippet,
        engine: this.name,
        score: this.weight,
        category: 'news',
        template: 'news',
        source: item.source,
        thumbnailUrl: item.imageUrl,
        publishedAt: item.publishedAt,
        metadata: {
          sourceUrl: item.sourceUrl,
          clusterId: this.topicId,
        },
      });
    }

    return results;
  }
}

/**
 * Factory function to create engines for all news categories.
 */
export function createAllCategoryEngines(): GoogleNewsRSSEngine[] {
  const categories: NewsCategory[] = [
    'top',
    'world',
    'nation',
    'business',
    'technology',
    'science',
    'health',
    'sports',
    'entertainment',
  ];

  return categories.map((cat) => new GoogleNewsRSSEngine(cat));
}
