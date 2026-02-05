/**
 * Pixabay Image Search Engine adapter.
 *
 * Scrapes Pixabay's search page for image results.
 * All Pixabay images are free to use under the Pixabay License.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { findElements, decodeHtmlEntities } from '../lib/html-parser';

// ========== Size Filter Mapping ==========

const pixabaySizeMap: Record<string, string> = {
  large: 'large', // > 3000px
  medium: 'medium', // 1000-3000px
  small: 'small', // < 1000px
};

// ========== Color Filter Mapping ==========

const pixabayColorMap: Record<string, string> = {
  transparent: 'transparent',
  gray: 'grayscale',
  red: 'red',
  orange: 'orange',
  yellow: 'yellow',
  green: 'green',
  teal: 'turquoise',
  blue: 'blue',
  purple: 'lilac',
  pink: 'pink',
  white: 'white',
  black: 'black',
  brown: 'brown',
};

// ========== Image Type Filter Mapping ==========

const pixabayTypeMap: Record<string, string> = {
  photo: 'photo',
  clipart: 'illustration',
  lineart: 'vector',
};

// ========== Orientation (Aspect) Filter Mapping ==========

const pixabayOrientationMap: Record<string, string> = {
  tall: 'vertical',
  wide: 'horizontal',
};

export class PixabayEngine implements OnlineEngine {
  name = 'pixabay';
  shortcut = 'px';
  categories: Category[] = ['images'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 8000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();

    searchParams.set('q', query);
    searchParams.set('pagi', params.page.toString());

    // Apply image filters
    const filters = params.imageFilters;
    if (filters) {
      // Size filter
      if (filters.size && filters.size !== 'any' && pixabaySizeMap[filters.size]) {
        searchParams.set('min_width', filters.size === 'large' ? '3000' : filters.size === 'medium' ? '1000' : '0');
      }

      // Color filter
      if (filters.color && filters.color !== 'any' && pixabayColorMap[filters.color]) {
        searchParams.set('colors', pixabayColorMap[filters.color]);
      }

      // Image type filter
      if (filters.type && filters.type !== 'any' && pixabayTypeMap[filters.type]) {
        searchParams.set('image_type', pixabayTypeMap[filters.type]);
      }

      // Orientation filter
      if (filters.aspect && filters.aspect !== 'any' && pixabayOrientationMap[filters.aspect]) {
        searchParams.set('orientation', pixabayOrientationMap[filters.aspect]);
      }

      // Minimum dimensions
      if (filters.minWidth) {
        searchParams.set('min_width', filters.minWidth.toString());
      }
      if (filters.minHeight) {
        searchParams.set('min_height', filters.minHeight.toString());
      }
    }

    // Safe search
    if (params.safeSearch >= 1) {
      searchParams.set('safesearch', 'true');
    }

    return {
      url: `https://pixabay.com/images/search/${encodeURIComponent(query)}/?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html,application/xhtml+xml',
        'Accept-Language': params.locale || 'en-US',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();
    const filters = params.imageFilters;

    // Try to find JSON data embedded in the page
    const jsonMatch = body.match(/<script[^>]*type="application\/json"[^>]*>(\{[^<]+\})<\/script>/);
    if (jsonMatch) {
      try {
        const data = JSON.parse(jsonMatch[1]);
        if (data.images || data.hits) {
          this.extractFromJson(data.images || data.hits, results, filters);
          if (results.results.length > 0) return results;
        }
      } catch {
        // Continue to HTML parsing
      }
    }

    // Try to find __NUXT__ data
    const nuxtMatch = body.match(/window\.__NUXT__\s*=\s*(\{[\s\S]*?\});\s*<\/script>/);
    if (nuxtMatch) {
      try {
        // Use safer eval-free parsing
        const data = this.parseNuxtData(nuxtMatch[1]);
        if (data) {
          this.extractFromNuxt(data, results, filters);
          if (results.results.length > 0) return results;
        }
      } catch {
        // Continue to HTML parsing
      }
    }

    // Parse HTML directly
    this.extractFromHtml(body, results, filters);

    return results;
  }

  private parseNuxtData(str: string): unknown {
    // Basic JSON-like parsing for Nuxt data
    // This is limited but safer than eval
    try {
      // Replace function calls and undefined values
      const cleaned = str
        .replace(/\bfunction\s*\([^)]*\)\s*\{[^}]*\}/g, '""')
        .replace(/\bundefined\b/g, 'null')
        .replace(/\bvoid 0\b/g, 'null');
      return JSON.parse(cleaned);
    } catch {
      return null;
    }
  }

  private extractFromJson(
    items: unknown[],
    results: EngineResults,
    filters?: EngineParams['imageFilters']
  ): void {
    for (const item of items) {
      const photo = item as Record<string, unknown>;
      if (!photo.largeImageURL && !photo.webformatURL) continue;

      const width = (photo.imageWidth || photo.webformatWidth || 0) as number;
      const height = (photo.imageHeight || photo.webformatHeight || 0) as number;

      // Client-side filtering
      if (filters) {
        if (filters.minWidth && width < filters.minWidth) continue;
        if (filters.minHeight && height < filters.minHeight) continue;
        if (filters.maxWidth && width > filters.maxWidth) continue;
        if (filters.maxHeight && height > filters.maxHeight) continue;
      }

      results.results.push({
        url: (photo.pageURL || `https://pixabay.com/photos/id-${photo.id}/`) as string,
        title: ((photo.tags as string) || 'Pixabay Image').replace(/,\s*/g, ', '),
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'images',
        template: 'images',
        imageUrl: (photo.largeImageURL || photo.webformatURL) as string,
        thumbnailUrl: (photo.previewURL || photo.webformatURL) as string,
        resolution: width && height ? `${width}x${height}` : '',
        source: (photo.user || 'Pixabay') as string,
      });
    }
  }

  private extractFromNuxt(
    data: unknown,
    results: EngineResults,
    filters?: EngineParams['imageFilters']
  ): void {
    const images: unknown[] = [];

    const traverse = (obj: unknown): void => {
      if (!obj || typeof obj !== 'object') return;

      if (Array.isArray(obj)) {
        for (const item of obj) {
          if (
            item &&
            typeof item === 'object' &&
            ('largeImageURL' in item || 'previewURL' in item || 'srcset' in item)
          ) {
            images.push(item);
          } else {
            traverse(item);
          }
        }
      } else {
        for (const value of Object.values(obj)) {
          traverse(value);
        }
      }
    };

    traverse(data);
    this.extractFromJson(images, results, filters);
  }

  private extractFromHtml(
    body: string,
    results: EngineResults,
    filters?: EngineParams['imageFilters']
  ): void {
    // Find image containers
    const imageContainers = findElements(body, 'div.container--wYO8e');
    if (imageContainers.length === 0) {
      // Try alternate class
      this.extractFromImageLinks(body, results, filters);
      return;
    }

    for (const container of imageContainers) {
      // Extract image source
      const srcsetMatch = container.match(/srcset="([^"]+)"/);
      const srcMatch = container.match(/src="([^"]+)"/);
      const imgSrc = srcsetMatch || srcMatch;

      if (!imgSrc) continue;

      // Parse srcset to get best quality
      let imageUrl = '';
      let thumbnailUrl = '';

      if (srcsetMatch) {
        const srcset = srcsetMatch[1];
        const sources = srcset.split(',').map((s) => {
          const parts = s.trim().split(/\s+/);
          const url = parts[0];
          const width = parseInt(parts[1]?.replace('w', '') || '0', 10);
          return { url, width };
        });
        sources.sort((a, b) => b.width - a.width);
        imageUrl = sources[0]?.url || '';
        thumbnailUrl = sources[sources.length - 1]?.url || imageUrl;
      } else if (srcMatch) {
        imageUrl = srcMatch[1];
        thumbnailUrl = imageUrl;
      }

      if (!imageUrl) continue;

      // Decode HTML entities
      imageUrl = decodeHtmlEntities(imageUrl);
      thumbnailUrl = decodeHtmlEntities(thumbnailUrl);

      // Extract page URL
      const linkMatch = container.match(/href="(\/photos\/[^"]+)"/);
      const pageUrl = linkMatch ? `https://pixabay.com${linkMatch[1]}` : '';

      // Extract dimensions from URL or data attributes
      let width = 0;
      let height = 0;
      const dimMatch = imageUrl.match(/_(\d+)x(\d+)\./);
      if (dimMatch) {
        width = parseInt(dimMatch[1], 10);
        height = parseInt(dimMatch[2], 10);
      }

      // Client-side filtering
      if (filters && width && height) {
        if (filters.minWidth && width < filters.minWidth) continue;
        if (filters.minHeight && height < filters.minHeight) continue;
        if (filters.maxWidth && width > filters.maxWidth) continue;
        if (filters.maxHeight && height > filters.maxHeight) continue;
      }

      // Extract alt text for title
      const altMatch = container.match(/alt="([^"]+)"/);
      const title = altMatch ? decodeHtmlEntities(altMatch[1]) : 'Pixabay Image';

      results.results.push({
        url: pageUrl || imageUrl,
        title,
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'images',
        template: 'images',
        imageUrl,
        thumbnailUrl,
        resolution: width && height ? `${width}x${height}` : '',
        source: 'Pixabay',
      });
    }
  }

  private extractFromImageLinks(
    body: string,
    results: EngineResults,
    _filters?: EngineParams['imageFilters']
  ): void {
    // Fallback: look for image links directly
    const imgRegex = /<a[^>]+href="(\/photos\/[^"]+)"[^>]*>\s*<img[^>]+src="([^"]+)"[^>]*alt="([^"]*)"[^>]*>/g;
    let match: RegExpExecArray | null;
    const seen = new Set<string>();

    while ((match = imgRegex.exec(body)) !== null) {
      const pageUrl = `https://pixabay.com${match[1]}`;
      if (seen.has(pageUrl)) continue;
      seen.add(pageUrl);

      const thumbnailUrl = decodeHtmlEntities(match[2]);
      const title = decodeHtmlEntities(match[3]) || 'Pixabay Image';

      // Get larger image from thumbnail URL
      const imageUrl = thumbnailUrl.replace(/_\d+\./, '_1280.');

      results.results.push({
        url: pageUrl,
        title,
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'images',
        template: 'images',
        imageUrl,
        thumbnailUrl,
        source: 'Pixabay',
      });
    }
  }
}
