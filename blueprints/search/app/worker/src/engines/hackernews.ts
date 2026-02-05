/**
 * Hacker News Engine.
 * Fetches news from Hacker News via the Algolia API.
 * https://hn.algolia.com/api
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

const HN_ALGOLIA_SEARCH_URL = 'https://hn.algolia.com/api/v1/search';
const HN_ALGOLIA_SEARCH_DATE_URL = 'https://hn.algolia.com/api/v1/search_by_date';

const HN_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Results per page (Algolia default is 20, max is 1000)
const RESULTS_PER_PAGE = 30;

// Time range in seconds mapping for numericFilters
const timeRangeSeconds: Record<string, number> = {
  day: 86400,
  week: 604800,
  month: 2592000,
  year: 31536000,
};

// Filter types - story types on HN
type HNStoryType = 'story' | 'comment' | 'poll' | 'pollopt' | 'show_hn' | 'ask_hn' | 'front_page';

interface HNAlgoliaHit {
  objectID: string;
  title?: string;
  url?: string;
  author: string;
  points?: number | null;
  story_text?: string;
  comment_text?: string;
  num_comments?: number;
  story_id?: number;
  story_title?: string;
  story_url?: string;
  created_at: string;
  created_at_i: number;
  _tags?: string[];
  _highlightResult?: {
    title?: { value?: string };
    url?: { value?: string };
    story_text?: { value?: string };
    author?: { value?: string };
  };
}

interface HNAlgoliaResponse {
  hits?: HNAlgoliaHit[];
  nbHits?: number;
  page?: number;
  nbPages?: number;
  hitsPerPage?: number;
  query?: string;
}

export class HackerNewsEngine implements OnlineEngine {
  name = 'hacker news';
  shortcut = 'hn';
  categories: Category[] = ['news', 'it'];
  supportsPaging = true;
  maxPage = 30;
  timeout = 10_000;
  weight = 0.95;
  disabled = false;

  private sortByDate: boolean;
  private storyType: HNStoryType | null;

  constructor(options?: { sortByDate?: boolean; storyType?: HNStoryType }) {
    this.sortByDate = options?.sortByDate ?? false;
    this.storyType = options?.storyType ?? null;

    if (this.sortByDate) {
      this.name = 'hacker news (recent)';
    }
    if (this.storyType) {
      this.name = `hacker news (${this.storyType.replace('_', ' ')})`;
    }
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('query', query);
    searchParams.set('page', String(params.page - 1)); // Algolia is 0-indexed
    searchParams.set('hitsPerPage', String(RESULTS_PER_PAGE));

    // Build tags filter
    const tags: string[] = [];

    // Filter by story type
    if (this.storyType) {
      tags.push(this.storyType);
    } else {
      // Default: search stories only (not comments)
      tags.push('story');
    }

    if (tags.length > 0) {
      searchParams.set('tags', tags.join(','));
    }

    // Time range filter using numericFilters
    const numericFilters: string[] = [];
    if (params.timeRange && timeRangeSeconds[params.timeRange]) {
      const cutoff = Math.floor(Date.now() / 1000) - timeRangeSeconds[params.timeRange];
      numericFilters.push(`created_at_i>${cutoff}`);
    }

    if (numericFilters.length > 0) {
      searchParams.set('numericFilters', numericFilters.join(','));
    }

    // Choose endpoint based on sort preference
    const baseUrl = this.sortByDate ? HN_ALGOLIA_SEARCH_DATE_URL : HN_ALGOLIA_SEARCH_URL;

    return {
      url: `${baseUrl}?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': HN_USER_AGENT,
        Accept: 'application/json',
        'Accept-Language': params.locale || 'en-US,en;q=0.9',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as HNAlgoliaResponse;

      if (!data.hits || !Array.isArray(data.hits)) {
        return results;
      }

      for (const hit of data.hits) {
        const result = this.parseHit(hit);
        if (result) {
          results.results.push(result);
        }
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }

  private parseHit(hit: HNAlgoliaHit): EngineResults['results'][0] | null {
    // Skip items without a title
    if (!hit.title) return null;

    // Determine the URL - use external URL if available, otherwise HN discussion
    const externalUrl = hit.url;
    const hnDiscussionUrl = `https://news.ycombinator.com/item?id=${hit.objectID}`;
    const url = externalUrl || hnDiscussionUrl;

    // Parse published time
    const publishedAt = hit.created_at
      ? new Date(hit.created_at).toISOString()
      : new Date(hit.created_at_i * 1000).toISOString();

    // Build content/snippet
    let content = '';
    if (hit.story_text) {
      // For Ask HN, Show HN, etc.
      content = this.stripHtml(hit.story_text);
      if (content.length > 300) {
        content = content.slice(0, 297) + '...';
      }
    } else if (externalUrl) {
      // For external links, show domain and stats
      try {
        const domain = new URL(externalUrl).hostname.replace(/^www\./, '');
        content = `${domain}`;
      } catch {
        content = '';
      }
    }

    // Add points and comments info
    const stats: string[] = [];
    if (hit.points !== null && hit.points !== undefined) {
      stats.push(`${hit.points} points`);
    }
    if (hit.num_comments !== undefined) {
      stats.push(`${hit.num_comments} comments`);
    }
    if (stats.length > 0) {
      if (content) {
        content += ` | ${stats.join(' | ')}`;
      } else {
        content = stats.join(' | ');
      }
    }

    // Determine source
    let source = 'Hacker News';
    if (externalUrl) {
      try {
        source = new URL(externalUrl).hostname.replace(/^www\./, '');
      } catch {
        // Keep as Hacker News
      }
    }

    return {
      url,
      title: decodeHtmlEntities(hit.title),
      content,
      engine: this.name,
      score: this.weight,
      category: 'news',
      template: 'news',
      source,
      publishedAt,
      metadata: {
        hnId: hit.objectID,
        hnUrl: hnDiscussionUrl,
        externalUrl,
        author: hit.author,
        points: hit.points,
        numComments: hit.num_comments,
        tags: hit._tags,
      },
    };
  }

  private stripHtml(html: string): string {
    return html
      .replace(/<[^>]+>/g, ' ')
      .replace(/\s+/g, ' ')
      .trim();
  }
}

/**
 * Hacker News Front Page Engine.
 * Fetches the current front page stories.
 */
export class HackerNewsFrontPageEngine implements OnlineEngine {
  name = 'hacker news front page';
  shortcut = 'hnfp';
  categories: Category[] = ['news', 'it'];
  supportsPaging = false;
  maxPage = 1;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(_query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('tags', 'front_page');
    searchParams.set('hitsPerPage', '30');

    return {
      url: `${HN_ALGOLIA_SEARCH_URL}?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': HN_USER_AGENT,
        Accept: 'application/json',
        'Accept-Language': params.locale || 'en-US,en;q=0.9',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as HNAlgoliaResponse;

      if (!data.hits || !Array.isArray(data.hits)) {
        return results;
      }

      for (const hit of data.hits) {
        if (!hit.title) continue;

        const externalUrl = hit.url;
        const hnUrl = `https://news.ycombinator.com/item?id=${hit.objectID}`;
        const url = externalUrl || hnUrl;

        const publishedAt = hit.created_at
          ? new Date(hit.created_at).toISOString()
          : new Date(hit.created_at_i * 1000).toISOString();

        let source = 'Hacker News';
        if (externalUrl) {
          try {
            source = new URL(externalUrl).hostname.replace(/^www\./, '');
          } catch {
            // Keep default
          }
        }

        let content = '';
        if (hit.points !== null && hit.points !== undefined) {
          content = `${hit.points} points`;
        }
        if (hit.num_comments !== undefined) {
          content += content ? ` | ${hit.num_comments} comments` : `${hit.num_comments} comments`;
        }

        results.results.push({
          url,
          title: decodeHtmlEntities(hit.title),
          content,
          engine: this.name,
          score: this.weight,
          category: 'news',
          template: 'news',
          source,
          publishedAt,
          metadata: {
            hnId: hit.objectID,
            hnUrl,
            externalUrl,
            author: hit.author,
            points: hit.points,
            numComments: hit.num_comments,
          },
        });
      }
    } catch {
      // Parse failed
    }

    return results;
  }
}

/**
 * Hacker News Show HN Engine.
 * Fetches Show HN posts.
 */
export class HackerNewsShowHNEngine extends HackerNewsEngine {
  constructor() {
    super({ storyType: 'show_hn' });
  }
}

/**
 * Hacker News Ask HN Engine.
 * Fetches Ask HN posts.
 */
export class HackerNewsAskHNEngine extends HackerNewsEngine {
  constructor() {
    super({ storyType: 'ask_hn' });
  }
}
