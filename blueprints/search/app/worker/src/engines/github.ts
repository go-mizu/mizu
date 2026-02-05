/**
 * GitHub Search Engine adapter.
 * Ported from Go: pkg/engine/local/engines/github.go
 *
 * Uses api.github.com/search/repositories, sorted by stars.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== GitHub API Types ==========

interface GitHubRepo {
  id?: number;
  full_name?: string;
  html_url?: string;
  description?: string;
  fork?: boolean;
  language?: string;
  stargazers_count?: number;
  watchers_count?: number;
  forks_count?: number;
  open_issues_count?: number;
  license?: {
    name?: string;
    spdx_id?: string;
  };
  topics?: string[];
  updated_at?: string;
  created_at?: string;
  homepage?: string;
  clone_url?: string;
  owner?: {
    login?: string;
    avatar_url?: string;
  };
}

interface GitHubSearchResponse {
  total_count?: number;
  items?: GitHubRepo[];
}

export class GitHubEngine implements OnlineEngine {
  name = 'github';
  shortcut = 'gh';
  categories: Category[] = ['it'];
  supportsPaging = false;
  maxPage = 1;
  timeout = 5_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, _params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('sort', 'stars');
    searchParams.set('order', 'desc');

    return {
      url: `https://api.github.com/search/repositories?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/vnd.github.preview.text-match+json',
        'User-Agent': 'SearXNG',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as GitHubSearchResponse;

      if (!data.items) return results;

      for (const item of data.items) {
        if (!item.html_url || !item.full_name) continue;

        // Build content with description and metadata
        let content = item.description || '';
        const meta: string[] = [];

        if (item.stargazers_count !== undefined && item.stargazers_count > 0) {
          meta.push(`${formatStars(item.stargazers_count)} stars`);
        }
        if (item.language) {
          meta.push(item.language);
        }
        if (item.topics && item.topics.length > 0) {
          meta.push(item.topics.slice(0, 5).join(', '));
        }

        if (meta.length > 0) {
          content = content
            ? `${content} | ${meta.join(' | ')}`
            : meta.join(' | ');
        }

        let publishedAt = '';
        if (item.updated_at) {
          try {
            publishedAt = new Date(item.updated_at).toISOString();
          } catch {
            publishedAt = item.updated_at;
          }
        }

        results.results.push({
          url: item.html_url,
          title: item.full_name,
          content,
          engine: this.name,
          score: this.weight,
          category: 'it',
          template: 'packages',
          thumbnailUrl: item.owner?.avatar_url || '',
          publishedAt,
          stars: item.stargazers_count || 0,
          language: item.language || '',
          topics: item.topics || [],
        });
      }
    } catch {
      // Parse error
    }

    return results;
  }
}

/**
 * Format star count with K/M suffixes for readability.
 */
function formatStars(count: number): string {
  if (count >= 1_000_000) {
    return (count / 1_000_000).toFixed(1) + 'M';
  }
  if (count >= 1_000) {
    return (count / 1_000).toFixed(1) + 'k';
  }
  return count.toString();
}
