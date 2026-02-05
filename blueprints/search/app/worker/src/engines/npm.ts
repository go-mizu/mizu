/**
 * NPM Package Search Engine adapter.
 * Uses NPM Registry API to search for packages.
 * https://github.com/npm/registry/blob/master/docs/REGISTRY-API.md
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== NPM API Types ==========

interface NpmPackage {
  name: string;
  scope?: string;
  version?: string;
  description?: string;
  keywords?: string[];
  date?: string;
  links?: {
    npm?: string;
    homepage?: string;
    repository?: string;
    bugs?: string;
  };
  author?: {
    name?: string;
    email?: string;
    username?: string;
  };
  publisher?: {
    username?: string;
    email?: string;
  };
  maintainers?: Array<{
    username?: string;
    email?: string;
  }>;
}

interface NpmSearchObject {
  package: NpmPackage;
  score?: {
    final?: number;
    detail?: {
      quality?: number;
      popularity?: number;
      maintenance?: number;
    };
  };
  searchScore?: number;
  flags?: {
    unstable?: boolean;
    deprecated?: string;
  };
  downloads?: {
    weekly?: number;
    monthly?: number;
  };
}

interface NpmSearchResponse {
  objects?: NpmSearchObject[];
  total?: number;
  time?: string;
}

const NPM_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

export class NpmEngine implements OnlineEngine {
  name = 'npm';
  shortcut = 'npm';
  categories: Category[] = ['it'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.95;
  disabled = false;

  private resultsPerPage = 20;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('text', query);
    searchParams.set('size', String(this.resultsPerPage));
    searchParams.set('from', String((params.page - 1) * this.resultsPerPage));

    return {
      url: `https://registry.npmjs.org/-/v1/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': NPM_USER_AGENT,
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as NpmSearchResponse;

      if (!data.objects || !Array.isArray(data.objects)) return results;

      for (const obj of data.objects) {
        const pkg = obj.package;
        if (!pkg || !pkg.name) continue;

        const npmUrl = pkg.links?.npm || `https://www.npmjs.com/package/${pkg.name}`;

        // Build content with description and metadata
        let content = pkg.description || '';
        const meta: string[] = [];

        if (pkg.version) {
          meta.push(`v${pkg.version}`);
        }
        if (obj.score?.detail?.popularity !== undefined) {
          const popularity = Math.round(obj.score.detail.popularity * 100);
          meta.push(`${popularity}% popularity`);
        }
        if (obj.flags?.deprecated) {
          meta.push('DEPRECATED');
        }
        if (pkg.keywords && pkg.keywords.length > 0) {
          meta.push(pkg.keywords.slice(0, 5).join(', '));
        }

        if (meta.length > 0) {
          content = content
            ? `${content} | ${meta.join(' | ')}`
            : meta.join(' | ');
        }

        let publishedAt = '';
        if (pkg.date) {
          try {
            publishedAt = new Date(pkg.date).toISOString();
          } catch {
            publishedAt = pkg.date;
          }
        }

        results.results.push({
          url: npmUrl,
          title: pkg.name,
          content,
          engine: this.name,
          score: this.weight,
          category: 'it',
          template: 'packages',
          publishedAt,
          topics: pkg.keywords || [],
          metadata: {
            version: pkg.version,
            scope: pkg.scope,
            homepage: pkg.links?.homepage,
            repository: pkg.links?.repository,
            author: pkg.author?.name || pkg.publisher?.username,
            maintainers: pkg.maintainers?.map(m => m.username).filter(Boolean),
            quality: obj.score?.detail?.quality,
            popularity: obj.score?.detail?.popularity,
            maintenance: obj.score?.detail?.maintenance,
            finalScore: obj.score?.final,
            deprecated: obj.flags?.deprecated,
            weeklyDownloads: obj.downloads?.weekly,
            monthlyDownloads: obj.downloads?.monthly,
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
 * Format download count with K/M suffixes.
 */
export function formatDownloads(count: number): string {
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
