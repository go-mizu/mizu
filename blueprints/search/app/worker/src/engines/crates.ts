/**
 * crates.io Search Engine adapter.
 * Uses crates.io API to search for Rust packages (crates).
 * https://crates.io/docs/api
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== crates.io API Types ==========

interface CrateInfo {
  id: string;
  name: string;
  description?: string;
  documentation?: string;
  homepage?: string;
  repository?: string;
  max_version?: string;
  max_stable_version?: string;
  newest_version?: string;
  downloads: number;
  recent_downloads?: number;
  created_at?: string;
  updated_at?: string;
  exact_match?: boolean;
  keywords?: string[];
  categories?: string[];
  links?: {
    version_downloads?: string;
    versions?: string;
    owners?: string;
    owner_team?: string;
    owner_user?: string;
    reverse_dependencies?: string;
  };
}

interface CrateMeta {
  total?: number;
  next_page?: string;
  prev_page?: string;
}

interface CratesSearchResponse {
  crates?: CrateInfo[];
  meta?: CrateMeta;
}

const CRATES_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Sort options
type CratesSort = 'relevance' | 'downloads' | 'recent-downloads' | 'recent-updates' | 'new';

export class CratesEngine implements OnlineEngine {
  name = 'crates.io';
  shortcut = 'crate';
  categories: Category[] = ['it'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  private resultsPerPage = 20;
  private sort: CratesSort;

  constructor(options?: { sort?: CratesSort }) {
    this.sort = options?.sort || 'relevance';
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('per_page', String(this.resultsPerPage));
    searchParams.set('page', String(params.page));
    searchParams.set('sort', this.sort);

    return {
      url: `https://crates.io/api/v1/crates?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': CRATES_USER_AGENT,
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as CratesSearchResponse;

      if (!data.crates || !Array.isArray(data.crates)) return results;

      for (const crate of data.crates) {
        if (!crate.name) continue;

        const crateUrl = `https://crates.io/crates/${crate.name}`;

        // Build content with description and metadata
        let content = crate.description || '';
        const meta: string[] = [];

        if (crate.max_stable_version || crate.newest_version) {
          meta.push(`v${crate.max_stable_version || crate.newest_version}`);
        }
        if (crate.downloads !== undefined && crate.downloads > 0) {
          meta.push(`${formatDownloads(crate.downloads)} downloads`);
        }
        if (crate.recent_downloads !== undefined && crate.recent_downloads > 0) {
          meta.push(`${formatDownloads(crate.recent_downloads)} recent`);
        }

        if (meta.length > 0) {
          content = content
            ? `${content} | ${meta.join(' | ')}`
            : meta.join(' | ');
        }

        let publishedAt = '';
        if (crate.updated_at) {
          try {
            publishedAt = new Date(crate.updated_at).toISOString();
          } catch {
            publishedAt = crate.updated_at;
          }
        }

        results.results.push({
          url: crateUrl,
          title: crate.name,
          content,
          engine: this.name,
          score: this.weight,
          category: 'it',
          template: 'packages',
          publishedAt,
          language: 'Rust',
          topics: [...(crate.keywords || []), ...(crate.categories || [])],
          metadata: {
            id: crate.id,
            version: crate.max_stable_version || crate.newest_version,
            maxVersion: crate.max_version,
            downloads: crate.downloads,
            recentDownloads: crate.recent_downloads,
            documentation: crate.documentation,
            homepage: crate.homepage,
            repository: crate.repository,
            createdAt: crate.created_at,
            keywords: crate.keywords,
            categories: crate.categories,
            exactMatch: crate.exact_match,
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
 * Format download count with K/M/B suffixes.
 */
function formatDownloads(count: number): string {
  if (count >= 1_000_000_000) {
    return (count / 1_000_000_000).toFixed(1) + 'B';
  }
  if (count >= 1_000_000) {
    return (count / 1_000_000).toFixed(1) + 'M';
  }
  if (count >= 1_000) {
    return (count / 1_000).toFixed(1) + 'k';
  }
  return count.toString();
}

/**
 * Crates sorted by downloads.
 */
export class CratesPopularEngine extends CratesEngine {
  constructor() {
    super({ sort: 'downloads' });
    this.name = 'crates.io (popular)';
  }
}

/**
 * Crates sorted by recent downloads.
 */
export class CratesTrendingEngine extends CratesEngine {
  constructor() {
    super({ sort: 'recent-downloads' });
    this.name = 'crates.io (trending)';
  }
}

/**
 * Crates sorted by recently updated.
 */
export class CratesRecentEngine extends CratesEngine {
  constructor() {
    super({ sort: 'recent-updates' });
    this.name = 'crates.io (recent)';
  }
}

/**
 * Newly published crates.
 */
export class CratesNewEngine extends CratesEngine {
  constructor() {
    super({ sort: 'new' });
    this.name = 'crates.io (new)';
  }
}
