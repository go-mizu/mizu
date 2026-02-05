/**
 * GitLab Search Engine adapter.
 * Uses GitLab API to search for projects.
 * https://docs.gitlab.com/ee/api/search.html
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== GitLab API Types ==========

interface GitLabProject {
  id: number;
  name: string;
  name_with_namespace?: string;
  path_with_namespace?: string;
  description?: string;
  web_url?: string;
  avatar_url?: string;
  star_count?: number;
  forks_count?: number;
  open_issues_count?: number;
  last_activity_at?: string;
  created_at?: string;
  default_branch?: string;
  topics?: string[];
  readme_url?: string;
  namespace?: {
    name?: string;
    path?: string;
    avatar_url?: string;
  };
}

// GitLab API returns an array directly
type GitLabSearchResponse = GitLabProject[];

const GITLAB_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

export class GitLabEngine implements OnlineEngine {
  name = 'gitlab';
  shortcut = 'gl';
  categories: Category[] = ['it'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  private baseUrl: string;
  private resultsPerPage = 20;

  constructor(options?: { baseUrl?: string }) {
    this.baseUrl = options?.baseUrl || 'https://gitlab.com';
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('search', query);
    searchParams.set('per_page', String(this.resultsPerPage));
    searchParams.set('page', String(params.page));
    // Note: order_by=stars is not supported in GitLab's projects API
    // Use last_activity_at for relevance instead
    searchParams.set('order_by', 'last_activity_at');
    searchParams.set('sort', 'desc');

    return {
      url: `${this.baseUrl}/api/v4/projects?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': GITLAB_USER_AGENT,
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const projects = JSON.parse(body) as GitLabSearchResponse;

      if (!Array.isArray(projects)) return results;

      for (const project of projects) {
        if (!project.web_url || !project.name) continue;

        // Build content with description and metadata
        let content = project.description || '';
        const meta: string[] = [];

        if (project.star_count !== undefined && project.star_count > 0) {
          meta.push(`${formatCount(project.star_count)} stars`);
        }
        if (project.forks_count !== undefined && project.forks_count > 0) {
          meta.push(`${formatCount(project.forks_count)} forks`);
        }
        if (project.topics && project.topics.length > 0) {
          meta.push(project.topics.slice(0, 5).join(', '));
        }

        if (meta.length > 0) {
          content = content
            ? `${content} | ${meta.join(' | ')}`
            : meta.join(' | ');
        }

        let publishedAt = '';
        if (project.last_activity_at) {
          try {
            publishedAt = new Date(project.last_activity_at).toISOString();
          } catch {
            publishedAt = project.last_activity_at;
          }
        }

        results.results.push({
          url: project.web_url,
          title: project.path_with_namespace || project.name,
          content,
          engine: this.name,
          score: this.weight,
          category: 'it',
          template: 'packages',
          thumbnailUrl: project.avatar_url || project.namespace?.avatar_url || '',
          publishedAt,
          stars: project.star_count || 0,
          topics: project.topics || [],
          metadata: {
            id: project.id,
            forks: project.forks_count,
            openIssues: project.open_issues_count,
            defaultBranch: project.default_branch,
            createdAt: project.created_at,
            namespace: project.namespace?.name,
          },
        });
      }
    } catch {
      // Parse error
    }

    return results;
  }
}

/**
 * Format count with K/M suffixes for readability.
 */
function formatCount(count: number): string {
  if (count >= 1_000_000) {
    return (count / 1_000_000).toFixed(1) + 'M';
  }
  if (count >= 1_000) {
    return (count / 1_000).toFixed(1) + 'k';
  }
  return count.toString();
}
