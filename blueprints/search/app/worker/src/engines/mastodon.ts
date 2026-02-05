/**
 * Mastodon Federated Search Engine adapter.
 *
 * Uses the Mastodon API for searching across federated instances.
 * Default instance: mastodon.social (largest public instance)
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { decodeHtmlEntities, extractText } from '../lib/html-parser';

// ========== Mastodon API Types ==========

interface MastodonAccount {
  id: string;
  username: string;
  acct: string;
  display_name: string;
  url: string;
  avatar: string;
  avatar_static: string;
  followers_count?: number;
  following_count?: number;
  statuses_count?: number;
}

interface MastodonStatus {
  id: string;
  created_at: string;
  url: string;
  uri: string;
  content: string;
  account: MastodonAccount;
  reblogs_count: number;
  favourites_count: number;
  replies_count?: number;
  media_attachments?: Array<{
    type: string;
    url: string;
    preview_url: string;
    description?: string;
  }>;
  spoiler_text?: string;
  sensitive?: boolean;
  card?: {
    url: string;
    title: string;
    description: string;
    image?: string;
  };
}

interface MastodonHashtag {
  name: string;
  url: string;
  history?: Array<{
    day: string;
    uses: string;
    accounts: string;
  }>;
}

interface MastodonSearchResponse {
  accounts?: MastodonAccount[];
  statuses?: MastodonStatus[];
  hashtags?: MastodonHashtag[];
}

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Default instance to search
const DEFAULT_INSTANCE = 'mastodon.social';

export class MastodonEngine implements OnlineEngine {
  name = 'mastodon';
  shortcut = 'mst';
  categories: Category[] = ['social'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.8;
  disabled = false;

  private instance: string;

  constructor(options?: { instance?: string }) {
    this.instance = options?.instance ?? DEFAULT_INSTANCE;
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('type', 'statuses'); // Search statuses (toots)
    searchParams.set('limit', '20');

    // Pagination via offset
    if (params.page > 1) {
      searchParams.set('offset', ((params.page - 1) * 20).toString());
    }

    // Resolve remote accounts/statuses if needed
    searchParams.set('resolve', 'false');

    return {
      url: `https://${this.instance}/api/v2/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': USER_AGENT,
      },
      cookies: [],
    };
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data: MastodonSearchResponse = JSON.parse(body);

      // Parse statuses (toots)
      if (data.statuses && Array.isArray(data.statuses)) {
        for (const status of data.statuses) {
          // Skip sensitive content if safe search is enabled
          if (params.safeSearch >= 1 && status.sensitive) {
            continue;
          }

          const result = this.parseStatus(status);
          if (result) {
            results.results.push(result);
          }
        }
      }

      // Extract hashtag suggestions
      if (data.hashtags && Array.isArray(data.hashtags)) {
        for (const tag of data.hashtags.slice(0, 5)) {
          results.suggestions.push(`#${tag.name}`);
        }
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }

  private parseStatus(status: MastodonStatus): EngineResults['results'][0] | null {
    if (!status.url || !status.content) return null;

    // Strip HTML from content
    const textContent = extractText(status.content);
    const displayName = status.account.display_name || status.account.username;

    // Build title from display name and truncated content
    let title = `${displayName}: `;
    const contentPreview = textContent.slice(0, 100);
    title += contentPreview.length < textContent.length
      ? contentPreview + '...'
      : contentPreview;

    // Get thumbnail from media or card
    let thumbnailUrl = '';
    let imageUrl = '';
    if (status.media_attachments && status.media_attachments.length > 0) {
      const media = status.media_attachments[0];
      thumbnailUrl = media.preview_url || media.url;
      if (media.type === 'image') {
        imageUrl = media.url;
      }
    } else if (status.card?.image) {
      thumbnailUrl = status.card.image;
    }

    // Format stats
    const stats: string[] = [];
    if (status.reblogs_count > 0) {
      stats.push(`${status.reblogs_count} boosts`);
    }
    if (status.favourites_count > 0) {
      stats.push(`${status.favourites_count} favorites`);
    }
    if (status.replies_count && status.replies_count > 0) {
      stats.push(`${status.replies_count} replies`);
    }

    let content = textContent;
    if (textContent.length > 300) {
      content = textContent.slice(0, 297) + '...';
    }
    if (stats.length > 0) {
      content += ` | ${stats.join(' | ')}`;
    }

    return {
      url: status.url,
      title: decodeHtmlEntities(title),
      content,
      engine: this.name,
      score: this.weight,
      category: 'social',
      template: thumbnailUrl ? 'images' : undefined,
      publishedAt: status.created_at,
      source: `@${status.account.acct}`,
      channel: displayName,
      thumbnailUrl: thumbnailUrl || undefined,
      imageUrl: imageUrl || undefined,
      metadata: {
        statusId: status.id,
        accountId: status.account.id,
        accountUrl: status.account.url,
        avatar: status.account.avatar,
        reblogsCount: status.reblogs_count,
        favouritesCount: status.favourites_count,
        repliesCount: status.replies_count,
        instance: this.instance,
      },
    };
  }
}

/**
 * Mastodon Account Search Engine.
 * Searches for user accounts instead of statuses.
 */
export class MastodonAccountsEngine implements OnlineEngine {
  name = 'mastodon accounts';
  shortcut = 'msta';
  categories: Category[] = ['social'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.75;
  disabled = false;

  private instance: string;

  constructor(options?: { instance?: string }) {
    this.instance = options?.instance ?? DEFAULT_INSTANCE;
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('type', 'accounts');
    searchParams.set('limit', '20');

    if (params.page > 1) {
      searchParams.set('offset', ((params.page - 1) * 20).toString());
    }

    return {
      url: `https://${this.instance}/api/v2/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': USER_AGENT,
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data: MastodonSearchResponse = JSON.parse(body);

      if (data.accounts && Array.isArray(data.accounts)) {
        for (const account of data.accounts) {
          results.results.push({
            url: account.url,
            title: account.display_name || account.username,
            content: `@${account.acct} | ${account.followers_count || 0} followers | ${account.statuses_count || 0} posts`,
            engine: this.name,
            score: this.weight,
            category: 'social',
            thumbnailUrl: account.avatar_static || account.avatar,
            template: 'images',
            source: this.instance,
            metadata: {
              accountId: account.id,
              username: account.username,
              acct: account.acct,
              followersCount: account.followers_count,
              followingCount: account.following_count,
              statusesCount: account.statuses_count,
            },
          });
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}
