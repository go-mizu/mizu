/**
 * Lemmy Federated Search Engine adapter.
 *
 * Uses the Lemmy API for searching posts and communities across the fediverse.
 * Default instance: lemmy.ml
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

// ========== Lemmy API Types ==========

interface LemmyCommunity {
  id: number;
  name: string;
  title: string;
  description?: string;
  icon?: string;
  banner?: string;
  actor_id: string;
  local: boolean;
  nsfw: boolean;
  subscribers: number;
  posts: number;
  comments: number;
}

interface LemmyCreator {
  id: number;
  name: string;
  display_name?: string;
  avatar?: string;
  actor_id: string;
  local: boolean;
}

interface LemmyPost {
  id: number;
  name: string;
  url?: string;
  body?: string;
  creator_id: number;
  community_id: number;
  nsfw: boolean;
  embed_title?: string;
  embed_description?: string;
  thumbnail_url?: string;
  ap_id: string;
  local: boolean;
  published: string;
  updated?: string;
}

interface LemmyPostView {
  post: LemmyPost;
  creator: LemmyCreator;
  community: LemmyCommunity;
  counts: {
    id: number;
    post_id: number;
    comments: number;
    score: number;
    upvotes: number;
    downvotes: number;
    published: string;
    newest_comment_time?: string;
  };
}

interface LemmyCommunityView {
  community: LemmyCommunity;
  counts: {
    id: number;
    community_id: number;
    subscribers: number;
    posts: number;
    comments: number;
    published: string;
    users_active_day: number;
    users_active_week: number;
    users_active_month: number;
    users_active_half_year: number;
  };
}

interface LemmySearchResponse {
  type_: string;
  posts?: LemmyPostView[];
  communities?: LemmyCommunityView[];
  comments?: unknown[];
  users?: unknown[];
}

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Default Lemmy instance
const DEFAULT_INSTANCE = 'lemmy.ml';

// Mapping of time range to Lemmy sort types
type LemmySortType = 'Active' | 'Hot' | 'New' | 'Old' | 'TopDay' | 'TopWeek' | 'TopMonth' | 'TopYear' | 'TopAll' | 'MostComments' | 'NewComments';

const timeRangeSort: Record<string, LemmySortType> = {
  day: 'TopDay',
  week: 'TopWeek',
  month: 'TopMonth',
  year: 'TopYear',
};

export class LemmyEngine implements OnlineEngine {
  name = 'lemmy';
  shortcut = 'lm';
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
    searchParams.set('type_', 'Posts'); // Search posts
    searchParams.set('limit', '20');
    searchParams.set('page', params.page.toString());

    // Sort based on time range
    const sort = params.timeRange && timeRangeSort[params.timeRange]
      ? timeRangeSort[params.timeRange]
      : 'TopAll';
    searchParams.set('sort', sort);

    // Listing type - search across all federated instances
    searchParams.set('listing_type', 'All');

    return {
      url: `https://${this.instance}/api/v3/search?${searchParams.toString()}`,
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
      const data: LemmySearchResponse = JSON.parse(body);

      if (data.posts && Array.isArray(data.posts)) {
        for (const postView of data.posts) {
          // Skip NSFW content if safe search is enabled
          if (params.safeSearch >= 1 && (postView.post.nsfw || postView.community.nsfw)) {
            continue;
          }

          const result = this.parsePostView(postView);
          if (result) {
            results.results.push(result);
          }
        }
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }

  private parsePostView(postView: LemmyPostView): EngineResults['results'][0] | null {
    const post = postView.post;
    const community = postView.community;
    const creator = postView.creator;
    const counts = postView.counts;

    if (!post.name) return null;

    // Use the Lemmy post URL or external URL
    const url = post.url || post.ap_id;

    // Build content from post body or embed description
    let content = '';
    if (post.body) {
      content = extractText(post.body);
      if (content.length > 300) {
        content = content.slice(0, 297) + '...';
      }
    } else if (post.embed_description) {
      content = post.embed_description;
    }

    // Add stats
    const stats: string[] = [];
    stats.push(`${counts.score} points`);
    stats.push(`${counts.comments} comments`);

    if (content) {
      content += ` | ${stats.join(' | ')}`;
    } else {
      content = stats.join(' | ');
    }

    // Get creator display name
    const authorName = creator.display_name || creator.name;

    return {
      url,
      title: decodeHtmlEntities(post.name),
      content,
      engine: this.name,
      score: this.weight,
      category: 'social',
      template: post.thumbnail_url ? 'images' : undefined,
      publishedAt: post.published,
      source: `!${community.name}@${this.extractInstance(community.actor_id)}`,
      channel: authorName,
      thumbnailUrl: post.thumbnail_url || community.icon || undefined,
      metadata: {
        postId: post.id,
        communityId: community.id,
        communityName: community.name,
        communityTitle: community.title,
        creatorId: creator.id,
        creatorName: creator.name,
        upvotes: counts.upvotes,
        downvotes: counts.downvotes,
        score: counts.score,
        comments: counts.comments,
        isLocal: post.local,
        instance: this.instance,
        externalUrl: post.url,
        apId: post.ap_id,
      },
    };
  }

  private extractInstance(actorId: string): string {
    try {
      return new URL(actorId).host;
    } catch {
      return this.instance;
    }
  }
}

/**
 * Lemmy Community Search Engine.
 * Searches for communities instead of posts.
 */
export class LemmyCommunitiesEngine implements OnlineEngine {
  name = 'lemmy communities';
  shortcut = 'lmc';
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
    searchParams.set('type_', 'Communities');
    searchParams.set('limit', '20');
    searchParams.set('page', params.page.toString());
    searchParams.set('sort', 'TopAll');
    searchParams.set('listing_type', 'All');

    return {
      url: `https://${this.instance}/api/v3/search?${searchParams.toString()}`,
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
      const data: LemmySearchResponse = JSON.parse(body);

      if (data.communities && Array.isArray(data.communities)) {
        for (const communityView of data.communities) {
          // Skip NSFW if safe search is enabled
          if (params.safeSearch >= 1 && communityView.community.nsfw) {
            continue;
          }

          const community = communityView.community;
          const counts = communityView.counts;

          const description = community.description
            ? extractText(community.description)
            : '';

          results.results.push({
            url: community.actor_id,
            title: community.title || community.name,
            content: `!${community.name} | ${counts.subscribers} subscribers | ${counts.posts} posts | ${description.slice(0, 200)}`,
            engine: this.name,
            score: this.weight,
            category: 'social',
            thumbnailUrl: community.icon || community.banner || undefined,
            template: community.icon ? 'images' : undefined,
            source: this.extractInstance(community.actor_id),
            metadata: {
              communityId: community.id,
              name: community.name,
              title: community.title,
              subscribers: counts.subscribers,
              posts: counts.posts,
              comments: counts.comments,
              usersActiveMonth: counts.users_active_month,
              isLocal: community.local,
            },
          });
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }

  private extractInstance(actorId: string): string {
    try {
      return new URL(actorId).host;
    } catch {
      return this.instance;
    }
  }
}
