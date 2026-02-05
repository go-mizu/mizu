/**
 * PyPI Package Search Engine adapter.
 * Uses PyPI JSON API and search page to search for packages.
 * Note: PyPI doesn't have an official search API, so we use their simple search endpoint.
 * https://warehouse.pypa.io/api-reference/
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== PyPI API Types ==========

interface PyPIPackageInfo {
  author?: string;
  author_email?: string;
  bugtrack_url?: string;
  classifiers?: string[];
  description?: string;
  description_content_type?: string;
  docs_url?: string;
  download_url?: string;
  downloads?: {
    last_day?: number;
    last_month?: number;
    last_week?: number;
  };
  home_page?: string;
  keywords?: string;
  license?: string;
  maintainer?: string;
  maintainer_email?: string;
  name: string;
  package_url?: string;
  platform?: string;
  project_url?: string;
  project_urls?: Record<string, string>;
  release_url?: string;
  requires_dist?: string[];
  requires_python?: string;
  summary?: string;
  version?: string;
  yanked?: boolean;
  yanked_reason?: string;
}

interface PyPIRelease {
  upload_time?: string;
  upload_time_iso_8601?: string;
  filename?: string;
  size?: number;
  packagetype?: string;
  python_version?: string;
  requires_python?: string;
}

interface PyPIPackageResponse {
  info: PyPIPackageInfo;
  releases?: Record<string, PyPIRelease[]>;
  urls?: PyPIRelease[];
}

// For the search results from pypi.org/search (HTML scraping as fallback)
interface PyPISearchResult {
  name: string;
  version: string;
  description: string;
  url: string;
  uploadTime?: string;
}

const PYPI_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

export class PyPIEngine implements OnlineEngine {
  name = 'pypi';
  shortcut = 'pip';
  categories: Category[] = ['it'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  private resultsPerPage = 20;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Use PyPI search page (HTML)
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('page', String(params.page));

    return {
      url: `https://pypi.org/search/?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': PYPI_USER_AGENT,
        Accept: 'text/html,application/xhtml+xml',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      // Parse HTML response from PyPI search
      const packages = this.parseSearchHtml(body);

      for (const pkg of packages) {
        let content = pkg.description || '';
        const meta: string[] = [];

        if (pkg.version) {
          meta.push(`v${pkg.version}`);
        }

        if (meta.length > 0) {
          content = content
            ? `${content} | ${meta.join(' | ')}`
            : meta.join(' | ');
        }

        let publishedAt = '';
        if (pkg.uploadTime) {
          try {
            publishedAt = new Date(pkg.uploadTime).toISOString();
          } catch {
            publishedAt = pkg.uploadTime;
          }
        }

        results.results.push({
          url: pkg.url,
          title: pkg.name,
          content,
          engine: this.name,
          score: this.weight,
          category: 'it',
          template: 'packages',
          publishedAt,
          language: 'Python',
          metadata: {
            version: pkg.version,
          },
        });
      }
    } catch {
      // Parse error
    }

    return results;
  }

  private parseSearchHtml(html: string): PyPISearchResult[] {
    const results: PyPISearchResult[] = [];

    // Match package snippets: <a class="package-snippet" href="/project/xxx/">
    // Each snippet contains: name, version, description
    const snippetRegex = /<a\s+class="package-snippet"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/gi;
    let match;

    while ((match = snippetRegex.exec(html)) !== null) {
      const href = match[1];
      const content = match[2];

      // Extract package name
      const nameMatch = content.match(/<span\s+class="package-snippet__name"[^>]*>([^<]+)<\/span>/i);
      const name = nameMatch ? nameMatch[1].trim() : '';

      // Extract version
      const versionMatch = content.match(/<span\s+class="package-snippet__version"[^>]*>([^<]+)<\/span>/i);
      const version = versionMatch ? versionMatch[1].trim() : '';

      // Extract description
      const descMatch = content.match(/<p\s+class="package-snippet__description"[^>]*>([^<]*)<\/p>/i);
      const description = descMatch ? descMatch[1].trim() : '';

      // Extract upload time (if available)
      const timeMatch = content.match(/<time[^>]*datetime="([^"]+)"[^>]*>/i);
      const uploadTime = timeMatch ? timeMatch[1] : '';

      if (name) {
        results.push({
          name,
          version,
          description,
          url: `https://pypi.org${href}`,
          uploadTime,
        });
      }
    }

    return results;
  }
}

/**
 * PyPI Package Detail fetcher (for getting full package info).
 * This is a helper function, not an engine.
 */
export async function fetchPyPIPackageInfo(packageName: string): Promise<PyPIPackageResponse | null> {
  try {
    const response = await fetch(`https://pypi.org/pypi/${packageName}/json`, {
      headers: {
        'User-Agent': PYPI_USER_AGENT,
        Accept: 'application/json',
      },
    });

    if (!response.ok) return null;

    return await response.json() as PyPIPackageResponse;
  } catch {
    return null;
  }
}
