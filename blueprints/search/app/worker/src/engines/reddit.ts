/**
 * Reddit Search Engine adapter.
 * Ported from Go: pkg/engine/local/engines/reddit.go
 *
 * Uses reddit.com/search.json for JSON results.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// Invalid thumbnail values to filter out
const INVALID_THUMBNAILS = new Set(['self', 'default', 'nsfw', 'spoiler', '']);

/**
 * Check if a URL is valid (has scheme and host).
 */
function isValidUrl(s: string): boolean {
  if (!s || INVALID_THUMBNAILS.has(s)) return false;
  try {
    const url = new URL(s);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

/**
 * Truncate text to maxLen characters with ellipsis.
 */
function truncateText(s: string, maxLen: number): string {
  if (s.length <= maxLen) return s;
  return s.slice(0, maxLen) + '...';
}

export class RedditEngine implements OnlineEngine {
  name = 'reddit';
  shortcut = 're';
  categories: Category[] = ['social'];
  supportsPaging = false;
  maxPage = 1;
  timeout = 5_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, _params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('limit', '25');

    return {
      url: `https://www.reddit.com/search.json?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': 'Mozilla/5.0 (compatible; SearXNG)',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as {
        data?: {
          children?: Array<{
            data?: {
              title?: string;
              selftext?: string;
              permalink?: string;
              url?: string;
              thumbnail?: string;
              created_utc?: number;
              subreddit?: string;
              author?: string;
              score?: number;
              num_comments?: number;
            };
          }>;
        };
      };

      if (!data.data?.children) return results;

      for (const child of data.data.children) {
        const post = child.data;
        if (!post?.permalink || !post.title) continue;

        const url = `https://www.reddit.com${post.permalink}`;
        const content = truncateText(post.selftext || '', 500);

        let publishedAt = '';
        if (post.created_utc && post.created_utc > 0) {
          publishedAt = new Date(post.created_utc * 1000).toISOString();
        }

        let thumbnailUrl = '';
        let imageUrl = '';
        if (isValidUrl(post.thumbnail || '')) {
          thumbnailUrl = post.thumbnail!;
          if (isValidUrl(post.url || '')) {
            imageUrl = post.url!;
          }
        }

        results.results.push({
          url,
          title: post.title,
          content,
          engine: this.name,
          score: this.weight,
          category: 'social',
          publishedAt,
          thumbnailUrl,
          imageUrl: imageUrl || undefined,
          template: thumbnailUrl ? 'images' : undefined,
          source: post.subreddit ? `r/${post.subreddit}` : undefined,
          channel: post.author || undefined,
          metadata: {
            upvotes: post.score ?? 0,
            comments: post.num_comments ?? 0,
            author: post.author,
            subreddit: post.subreddit,
            published: publishedAt,
          },
        });
      }
    } catch {
      // Parse error
    }

    return results;
  }
}
