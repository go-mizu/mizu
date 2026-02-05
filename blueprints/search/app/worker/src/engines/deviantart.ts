/**
 * DeviantArt Image Search Engine adapter.
 *
 * Scrapes DeviantArt's search results for artwork images.
 * Uses the public search endpoint without API key.
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

// ========== DeviantArt Data Types ==========

interface DeviantArtDeviation {
  deviationId?: number;
  url?: string;
  title?: string;
  publishedTime?: string;
  isDownloadable?: boolean;
  media?: {
    baseUri?: string;
    prettyName?: string;
    types?: Array<{
      t?: string;
      h?: number;
      w?: number;
      c?: string;
    }>;
    token?: string[];
  };
  author?: {
    username?: string;
    usericon?: string;
  };
  stats?: {
    favourites?: number;
    comments?: number;
  };
  extended?: {
    originalFile?: {
      width?: number;
      height?: number;
    };
  };
}

// Note: DeviantArt API types reserved for future OAuth implementation
// interface DeviantArtSearchResult {
//   results?: DeviantArtDeviation[];
//   hasMore?: boolean;
//   nextOffset?: number;
// }

// ========== Order/Sort Mapping ==========

// Reserved for future authenticated API access
// const deviantartOrderMap: Record<string, string> = {
//   relevance: 'most-relevant',
//   newest: 'newest',
//   popular: 'popular-all-time',
// };

export class DeviantArtEngine implements OnlineEngine {
  name = 'deviantart';
  shortcut = 'da';
  categories: Category[] = ['images'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10000;
  weight = 0.8;
  disabled = false;

  private readonly perPage = 24;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const offset = (params.page - 1) * this.perPage;

    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('offset', offset.toString());

    // Apply time range filter
    if (params.timeRange) {
      const timeMap: Record<string, string> = {
        day: '24hr',
        week: '1week',
        month: '1month',
        year: '1year',
      };
      if (timeMap[params.timeRange]) {
        searchParams.set('order', `popular-${timeMap[params.timeRange]}`);
      }
    }

    // Mature content filter based on safe search level
    if (params.safeSearch >= 1) {
      searchParams.set('mature_content', 'false');
    }

    return {
      url: `https://www.deviantart.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
        'Accept-Language': params.locale || 'en-US',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      },
      cookies: params.safeSearch >= 1 ? ['agegate_state=1'] : [],
    };
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();
    const filters = params.imageFilters;

    // Try to extract __INITIAL_STATE__ JSON data
    const stateMatch = body.match(/window\.__INITIAL_STATE__\s*=\s*JSON\.parse\(["'](.+?)["']\);/);
    if (stateMatch) {
      try {
        // The state is JSON-stringified twice, so we need to parse and unescape
        const escaped = stateMatch[1];
        const unescaped = escaped
          .replace(/\\"/g, '"')
          .replace(/\\'/g, "'")
          .replace(/\\\\/g, '\\')
          .replace(/\\n/g, '\n')
          .replace(/\\r/g, '\r')
          .replace(/\\t/g, '\t');
        const state = JSON.parse(unescaped);

        this.extractFromState(state, results, filters);
        if (results.results.length > 0) return results;
      } catch {
        // Continue to other extraction methods
      }
    }

    // Try Apollo state
    const apolloMatch = body.match(/window\.__APOLLO_STATE__\s*=\s*(\{[\s\S]*?\});\s*(?:window\.|<\/script>)/);
    if (apolloMatch) {
      try {
        const state = JSON.parse(apolloMatch[1]);
        this.extractFromApollo(state, results, filters);
        if (results.results.length > 0) return results;
      } catch {
        // Continue
      }
    }

    // Fallback: parse HTML directly
    this.extractFromHtml(body, results, filters);

    return results;
  }

  private extractFromState(
    state: unknown,
    results: EngineResults,
    filters?: EngineParams['imageFilters']
  ): void {
    const deviations = this.findDeviations(state);

    for (const deviation of deviations) {
      this.processDeviation(deviation, results, filters);
    }
  }

  private extractFromApollo(
    state: Record<string, unknown>,
    results: EngineResults,
    filters?: EngineParams['imageFilters']
  ): void {
    // Apollo state stores deviations as keyed objects
    for (const key of Object.keys(state)) {
      if (key.startsWith('Deviation:')) {
        const deviation = state[key] as DeviantArtDeviation;
        this.processDeviation(deviation, results, filters);
      }
    }
  }

  private findDeviations(data: unknown): DeviantArtDeviation[] {
    const deviations: DeviantArtDeviation[] = [];

    const traverse = (obj: unknown): void => {
      if (!obj || typeof obj !== 'object') return;

      if (Array.isArray(obj)) {
        for (const item of obj) {
          if (
            item &&
            typeof item === 'object' &&
            ('deviationId' in item || 'deviation' in item)
          ) {
            const dev = 'deviation' in item ? (item as { deviation: DeviantArtDeviation }).deviation : item;
            if (dev && typeof dev === 'object') {
              deviations.push(dev as DeviantArtDeviation);
            }
          } else {
            traverse(item);
          }
        }
      } else {
        // Check if this object is a deviation
        if ('deviationId' in obj && 'media' in obj) {
          deviations.push(obj as DeviantArtDeviation);
        }

        for (const value of Object.values(obj)) {
          traverse(value);
        }
      }
    };

    traverse(data);
    return deviations;
  }

  private processDeviation(
    deviation: DeviantArtDeviation,
    results: EngineResults,
    filters?: EngineParams['imageFilters']
  ): void {
    if (!deviation.media?.baseUri) return;

    // Build image URLs from media object
    const media = deviation.media;
    const baseUri = media.baseUri || '';
    const token = media.token?.[0] ? `?token=${media.token[0]}` : '';

    // Find the best quality type
    const types = media.types || [];
    let bestType = types.find((t) => t.t === 'fullview') ||
                   types.find((t) => t.t === 'preview') ||
                   types[0];

    if (!bestType) return;

    // Build image URL with token
    let imageUrl = baseUri;
    if (bestType.c) {
      imageUrl = baseUri.replace('<prettyName>', media.prettyName || '');
      imageUrl = imageUrl.replace(/\/v1\/.*$/, '') + bestType.c;
    }
    imageUrl += token;

    // Get thumbnail
    const thumbType = types.find((t) => t.t === '150') ||
                      types.find((t) => t.t === 'preview') ||
                      bestType;
    let thumbnailUrl = baseUri;
    if (thumbType?.c) {
      thumbnailUrl = baseUri.replace('<prettyName>', media.prettyName || '');
      thumbnailUrl = thumbnailUrl.replace(/\/v1\/.*$/, '') + thumbType.c;
    }
    thumbnailUrl += token;

    // Get dimensions
    const width = bestType.w || deviation.extended?.originalFile?.width || 0;
    const height = bestType.h || deviation.extended?.originalFile?.height || 0;

    // Client-side filtering
    if (filters && width && height) {
      if (filters.minWidth && width < filters.minWidth) return;
      if (filters.minHeight && height < filters.minHeight) return;
      if (filters.maxWidth && width > filters.maxWidth) return;
      if (filters.maxHeight && height > filters.maxHeight) return;

      // Size category filter
      if (filters.size && filters.size !== 'any') {
        const maxDim = Math.max(width, height);
        if (filters.size === 'large' && maxDim < 1920) return;
        if (filters.size === 'medium' && (maxDim < 800 || maxDim > 1920)) return;
        if (filters.size === 'small' && (maxDim < 300 || maxDim > 800)) return;
        if (filters.size === 'icon' && maxDim > 300) return;
      }

      // Aspect ratio filter
      if (filters.aspect && filters.aspect !== 'any') {
        const ratio = width / height;
        if (filters.aspect === 'tall' && ratio > 0.9) return;
        if (filters.aspect === 'wide' && ratio < 1.1) return;
        if (filters.aspect === 'square' && (ratio < 0.8 || ratio > 1.2)) return;
        if (filters.aspect === 'panoramic' && ratio < 2.0) return;
      }
    }

    results.results.push({
      url: deviation.url || `https://www.deviantart.com/deviation/${deviation.deviationId}`,
      title: deviation.title || 'DeviantArt Artwork',
      content: '',
      engine: this.name,
      score: this.weight,
      category: 'images',
      template: 'images',
      imageUrl,
      thumbnailUrl,
      resolution: width && height ? `${width}x${height}` : '',
      source: deviation.author?.username || 'DeviantArt',
    });
  }

  private extractFromHtml(
    body: string,
    results: EngineResults,
    _filters?: EngineParams['imageFilters']
  ): void {
    // Look for deviation links and images in HTML
    const deviationRegex = /<a[^>]+href="(https:\/\/www\.deviantart\.com\/[^\/]+\/art\/[^"]+)"[^>]*>[\s\S]*?<img[^>]+src="([^"]+)"[^>]*>/g;
    let match: RegExpExecArray | null;
    const seen = new Set<string>();

    while ((match = deviationRegex.exec(body)) !== null) {
      const pageUrl = match[1];
      if (seen.has(pageUrl)) continue;
      seen.add(pageUrl);

      let imageUrl = decodeHtmlEntities(match[2]);
      // Try to get higher quality version
      imageUrl = imageUrl.replace(/\/v1\/fill\/w_\d+,h_\d+[^/]*\//, '/v1/fill/w_1200/');

      // Extract title from URL
      const titleMatch = pageUrl.match(/\/art\/([^?]+)/);
      const title = titleMatch
        ? decodeURIComponent(titleMatch[1]).replace(/-\d+$/, '').replace(/-/g, ' ')
        : 'DeviantArt Artwork';

      // Extract username from URL
      const userMatch = pageUrl.match(/deviantart\.com\/([^\/]+)\/art/);
      const source = userMatch ? userMatch[1] : 'DeviantArt';

      results.results.push({
        url: pageUrl,
        title,
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'images',
        template: 'images',
        imageUrl,
        thumbnailUrl: imageUrl,
        source,
      });
    }

    // Also look for data-super-img or data-src attributes
    const imgRegex = /data-(?:super-img|src)="([^"]+)"/g;
    while ((match = imgRegex.exec(body)) !== null) {
      const imageUrl = decodeHtmlEntities(match[1]);
      if (!imageUrl.includes('deviantart') && !imageUrl.includes('wixmp')) continue;
      if (seen.has(imageUrl)) continue;
      seen.add(imageUrl);

      results.results.push({
        url: imageUrl,
        title: 'DeviantArt Artwork',
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'images',
        template: 'images',
        imageUrl,
        thumbnailUrl: imageUrl,
        source: 'DeviantArt',
      });
    }
  }
}
