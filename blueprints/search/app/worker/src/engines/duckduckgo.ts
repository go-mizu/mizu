/**
 * DuckDuckGo Search Engine adapters.
 * Ported from Go: pkg/engine/local/engines/duckduckgo.go
 *
 * Uses JSON APIs (i.js, v.js, news.js) which work without CAPTCHA.
 * VQD token fetching with 1-hour caching.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

const DDG_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36';

// ========== VQD Token Cache ==========

interface VqdEntry {
  value: string;
  expires: number;
}

const vqdCache = new Map<string, VqdEntry>();

/**
 * Fetch or return cached VQD token for a query.
 * VQD is required for DDG's bot protection on JSON APIs.
 */
async function getVqd(query: string, region: string): Promise<string> {
  const cacheKey = `${query}//${region}`;

  // Check cache
  const cached = vqdCache.get(cacheKey);
  if (cached && Date.now() < cached.expires) {
    return cached.value;
  }

  // Fetch VQD from DuckDuckGo
  const reqUrl = `https://duckduckgo.com/?q=${encodeURIComponent(query)}`;

  const response = await fetch(reqUrl, {
    headers: {
      'User-Agent': DDG_USER_AGENT,
      Accept:
        'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
      'Accept-Language': 'en-US,en;q=0.9',
    },
    redirect: 'manual',
  });

  if (!response.ok && response.status !== 301 && response.status !== 302) {
    throw new Error(`DDG VQD request failed: ${response.status}`);
  }

  const body = await response.text();

  // Extract VQD using regex
  let vqd = '';
  const doubleQuoteMatch = body.match(/vqd="([^"]+)"/);
  if (doubleQuoteMatch) {
    vqd = doubleQuoteMatch[1];
  } else {
    const singleQuoteMatch = body.match(/vqd='([^']+)'/);
    if (singleQuoteMatch) {
      vqd = singleQuoteMatch[1];
    }
  }

  if (!vqd) {
    throw new Error('Could not extract VQD from DDG response');
  }

  // Cache for 1 hour
  vqdCache.set(cacheKey, {
    value: vqd,
    expires: Date.now() + 3600_000,
  });

  return vqd;
}

/**
 * Store a VQD token in cache (e.g., from a response).
 */
function setVqd(query: string, region: string, vqd: string): void {
  const cacheKey = `${query}//${region}`;
  vqdCache.set(cacheKey, {
    value: vqd,
    expires: Date.now() + 3600_000,
  });
}

// ========== Region Mapping ==========

const ddgRegions: Record<string, string> = {
  'en-US': 'us-en',
  'en-GB': 'uk-en',
  'de-DE': 'de-de',
  'fr-FR': 'fr-fr',
  'es-ES': 'es-es',
  'it-IT': 'it-it',
  'ja-JP': 'jp-jp',
  'ko-KR': 'kr-kr',
  'zh-CN': 'cn-zh',
  'ru-RU': 'ru-ru',
};

function getDdgRegion(locale: string): string {
  return ddgRegions[locale] || 'wt-wt';
}

// ========== Sec-Fetch Headers ==========

const secFetchHeaders: Record<string, string> = {
  'Sec-Fetch-Dest': 'document',
  'Sec-Fetch-Mode': 'navigate',
  'Sec-Fetch-Site': 'same-origin',
  'Sec-Fetch-User': '?1',
  'Upgrade-Insecure-Requests': '1',
};

// ========== DuckDuckGoImagesEngine ==========

export class DuckDuckGoImagesEngine implements OnlineEngine {
  name = 'duckduckgo images';
  shortcut = 'ddi';
  categories: Category[] = ['images'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  /**
   * NOTE: This engine requires an async VQD fetch before building the request.
   * The caller should pre-fetch VQD and pass it in params.engineData['vqd'].
   * If not provided, executeEngine will fail. Use the companion helper
   * `prepareVqd(query, locale)` before calling executeEngine.
   */
  buildRequest(query: string, params: EngineParams): RequestConfig {
    const region = getDdgRegion(params.locale);
    const vqd = params.engineData['vqd'] || '';

    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('o', 'json');
    searchParams.set('l', region);
    searchParams.set('f', ',,,,,');
    searchParams.set('vqd', vqd);

    if (params.page > 1) {
      searchParams.set('s', ((params.page - 1) * 100).toString());
    }

    // SafeSearch
    if (params.safeSearch === 0) {
      searchParams.set('p', '-1');
    } else if (params.safeSearch === 2) {
      searchParams.set('p', '1');
    }

    return {
      url: `https://duckduckgo.com/i.js?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': DDG_USER_AGENT,
        Accept: 'application/json, text/javascript, */*; q=0.01',
        Referer: 'https://duckduckgo.com/',
        'X-Requested-With': 'XMLHttpRequest',
        ...secFetchHeaders,
      },
      cookies: [`l=${region}`, `ah=${region}`],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    const jsonStart = body.indexOf('{');
    if (jsonStart === -1) return results;

    try {
      const data = JSON.parse(body.slice(jsonStart)) as {
        results?: Array<{
          image?: string;
          thumbnail?: string;
          title?: string;
          url?: string;
          source?: string;
          width?: number;
          height?: number;
        }>;
      };

      if (data.results) {
        for (const item of data.results) {
          if (item.image) {
            results.results.push({
              url: item.url || '',
              title: item.title || '',
              content: '',
              engine: this.name,
              score: this.weight,
              category: 'images',
              template: 'images',
              imageUrl: item.image,
              thumbnailUrl: item.thumbnail || '',
              source: item.source || '',
              resolution: `${item.width || 0}x${item.height || 0}`,
            });
          }
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}

// ========== DuckDuckGoVideosEngine ==========

export class DuckDuckGoVideosEngine implements OnlineEngine {
  name = 'duckduckgo videos';
  shortcut = 'ddv';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const region = getDdgRegion(params.locale);
    const vqd = params.engineData['vqd'] || '';

    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('o', 'json');
    searchParams.set('l', region);
    searchParams.set('f', ',,,,,');
    searchParams.set('vqd', vqd);

    if (params.page > 1) {
      searchParams.set('s', ((params.page - 1) * 60).toString());
    }

    if (params.safeSearch === 0) {
      searchParams.set('p', '-1');
    } else if (params.safeSearch === 2) {
      searchParams.set('p', '1');
    }

    return {
      url: `https://duckduckgo.com/v.js?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': DDG_USER_AGENT,
        Accept: 'application/json, text/javascript, */*; q=0.01',
        Referer: 'https://duckduckgo.com/',
        'X-Requested-With': 'XMLHttpRequest',
        ...secFetchHeaders,
      },
      cookies: [`l=${region}`],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    const jsonStart = body.indexOf('{');
    if (jsonStart === -1) return results;

    try {
      const data = JSON.parse(body.slice(jsonStart)) as {
        results?: Array<{
          content?: string;
          title?: string;
          description?: string;
          duration?: string;
          provider?: string;
          uploader?: string;
          published?: string;
          images?: { small?: string; medium?: string; large?: string };
          embed_url?: string;
        }>;
      };

      if (data.results) {
        for (const item of data.results) {
          let thumbnail =
            item.images?.medium ||
            item.images?.small ||
            item.images?.large ||
            '';

          let content = item.description || '';
          if (item.uploader && content) {
            content = `by ${item.uploader} - ${content}`;
          } else if (item.uploader) {
            content = `by ${item.uploader}`;
          }

          results.results.push({
            url: item.content || '',
            title: item.title || '',
            content,
            engine: this.name,
            score: this.weight,
            category: 'videos',
            template: 'videos',
            duration: item.duration || '',
            source: item.provider || '',
            thumbnailUrl: thumbnail,
            embedUrl: item.embed_url || '',
          });
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}

// ========== DuckDuckGoNewsEngine ==========

export class DuckDuckGoNewsEngine implements OnlineEngine {
  name = 'duckduckgo news';
  shortcut = 'ddn';
  categories: Category[] = ['news'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const region = getDdgRegion(params.locale);
    const vqd = params.engineData['vqd'] || '';

    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('o', 'json');
    searchParams.set('l', region);
    searchParams.set('f', ',,,,,');
    searchParams.set('vqd', vqd);

    if (params.page > 1) {
      searchParams.set('s', ((params.page - 1) * 30).toString());
    }

    return {
      url: `https://duckduckgo.com/news.js?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': DDG_USER_AGENT,
        Accept: 'application/json, text/javascript, */*; q=0.01',
        Referer: 'https://duckduckgo.com/',
        'X-Requested-With': 'XMLHttpRequest',
        ...secFetchHeaders,
      },
      cookies: [`l=${region}`],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    const jsonStart = body.indexOf('{');
    if (jsonStart === -1) return results;

    try {
      const data = JSON.parse(body.slice(jsonStart)) as {
        results?: Array<{
          url?: string;
          title?: string;
          excerpt?: string;
          source?: string;
          image?: string;
          date?: number;
          relative_time?: string;
        }>;
      };

      if (data.results) {
        for (const item of data.results) {
          let publishedAt = '';
          if (item.date && item.date > 0) {
            publishedAt = new Date(item.date * 1000).toISOString();
          }

          results.results.push({
            url: item.url || '',
            title: item.title || '',
            content: item.excerpt || '',
            engine: this.name,
            score: this.weight,
            category: 'news',
            template: 'news',
            source: item.source || '',
            thumbnailUrl: item.image || '',
            publishedAt,
          });
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}

// ========== VQD Helper ==========

/**
 * Pre-fetch VQD token for DuckDuckGo engines.
 * Call this before executeEngine for any DDG engine.
 * Returns the VQD token to set in params.engineData['vqd'].
 */
export async function prepareVqd(
  query: string,
  locale: string
): Promise<string> {
  const region = getDdgRegion(locale);
  return getVqd(query, region);
}

export { setVqd };
