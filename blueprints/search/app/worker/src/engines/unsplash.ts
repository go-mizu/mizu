/**
 * Unsplash Image Search Engine adapter.
 *
 * Uses Unsplash's napi (Next.js API) endpoints for searching photos.
 * No API key required for basic public searches.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== Unsplash API Types ==========

interface UnsplashPhoto {
  id: string;
  slug?: string;
  created_at?: string;
  width: number;
  height: number;
  color?: string;
  blur_hash?: string;
  description?: string;
  alt_description?: string;
  urls: {
    raw?: string;
    full?: string;
    regular?: string;
    small?: string;
    thumb?: string;
  };
  links?: {
    self?: string;
    html?: string;
    download?: string;
    download_location?: string;
  };
  user?: {
    id?: string;
    username?: string;
    name?: string;
    portfolio_url?: string;
    profile_image?: {
      small?: string;
      medium?: string;
      large?: string;
    };
  };
  tags?: Array<{
    type?: string;
    title?: string;
  }>;
}

interface UnsplashSearchResponse {
  total: number;
  total_pages: number;
  results: UnsplashPhoto[];
}

// ========== Color Filter Mapping ==========

const unsplashColorMap: Record<string, string> = {
  color: '', // Any color (default)
  gray: 'black_and_white',
  black: 'black',
  white: 'white',
  yellow: 'yellow',
  orange: 'orange',
  red: 'red',
  purple: 'purple',
  green: 'green',
  teal: 'teal',
  blue: 'blue',
};

// ========== Orientation (Aspect) Filter Mapping ==========

const unsplashOrientationMap: Record<string, string> = {
  tall: 'portrait',
  wide: 'landscape',
  square: 'squarish',
};

export class UnsplashEngine implements OnlineEngine {
  name = 'unsplash';
  shortcut = 'us';
  categories: Category[] = ['images'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 8000;
  weight = 0.9;
  disabled = false;

  private readonly perPage = 20;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();

    searchParams.set('query', query);
    searchParams.set('per_page', this.perPage.toString());
    searchParams.set('page', params.page.toString());

    // Apply image filters
    const filters = params.imageFilters;
    if (filters) {
      // Color filter
      if (filters.color && filters.color !== 'any' && unsplashColorMap[filters.color]) {
        searchParams.set('color', unsplashColorMap[filters.color]);
      }

      // Orientation (aspect ratio)
      if (filters.aspect && filters.aspect !== 'any' && unsplashOrientationMap[filters.aspect]) {
        searchParams.set('orientation', unsplashOrientationMap[filters.aspect]);
      }
    }

    // Sort order - Unsplash defaults to relevance
    // Time range not directly supported but we can use order_by
    if (params.timeRange === 'day' || params.timeRange === 'week') {
      searchParams.set('order_by', 'latest');
    }

    // Unsplash's napi endpoint
    return {
      url: `https://unsplash.com/napi/search/photos?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'Accept-Language': params.locale || 'en-US',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        Referer: 'https://unsplash.com/',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();
    const filters = params.imageFilters;

    try {
      const data: UnsplashSearchResponse = JSON.parse(body);

      if (!data.results || data.results.length === 0) {
        return results;
      }

      for (const photo of data.results) {
        const width = photo.width;
        const height = photo.height;

        // Client-side size filtering
        if (filters) {
          if (filters.minWidth && width < filters.minWidth) continue;
          if (filters.minHeight && height < filters.minHeight) continue;
          if (filters.maxWidth && width > filters.maxWidth) continue;
          if (filters.maxHeight && height > filters.maxHeight) continue;

          // Size category filter
          if (filters.size && filters.size !== 'any') {
            const maxDim = Math.max(width, height);
            if (filters.size === 'large' && maxDim < 2000) continue;
            if (filters.size === 'medium' && (maxDim < 1000 || maxDim > 2000)) continue;
            if (filters.size === 'small' && (maxDim < 400 || maxDim > 1000)) continue;
            if (filters.size === 'icon' && maxDim > 400) continue;
          }
        }

        // Get best image URL
        const imageUrl = photo.urls.full || photo.urls.regular || photo.urls.raw || '';
        const thumbnailUrl = photo.urls.small || photo.urls.thumb || '';

        // Build page URL
        const pageUrl = photo.links?.html || `https://unsplash.com/photos/${photo.slug || photo.id}`;

        // Build title/description
        const title = photo.alt_description || photo.description || `Photo by ${photo.user?.name || 'Unknown'}`;
        const content = photo.description || '';

        results.results.push({
          url: pageUrl,
          title,
          content,
          engine: this.name,
          score: this.weight,
          category: 'images',
          template: 'images',
          imageUrl,
          thumbnailUrl,
          resolution: `${width}x${height}`,
          source: photo.user?.name || 'Unsplash',
        });
      }
    } catch {
      // JSON parse failed, try HTML extraction
      this.extractFromHtml(body, results);
    }

    return results;
  }

  private extractFromHtml(body: string, results: EngineResults): void {
    // Fallback: try to extract from HTML/SSR data if JSON fails
    const dataRegex = /<script[^>]*id="__NEXT_DATA__"[^>]*>([^<]+)<\/script>/;
    const match = body.match(dataRegex);

    if (match) {
      try {
        const nextData = JSON.parse(match[1]);
        const photos = this.findPhotosInNextData(nextData);

        for (const photo of photos) {
          if (photo.urls?.regular) {
            results.results.push({
              url: `https://unsplash.com/photos/${photo.id}`,
              title: photo.alt_description || 'Unsplash Photo',
              content: photo.description || '',
              engine: this.name,
              score: this.weight,
              category: 'images',
              template: 'images',
              imageUrl: photo.urls.full || photo.urls.regular,
              thumbnailUrl: photo.urls.small || photo.urls.thumb,
              resolution: photo.width && photo.height ? `${photo.width}x${photo.height}` : '',
              source: photo.user?.name || 'Unsplash',
            });
          }
        }
      } catch {
        // Parsing failed
      }
    }
  }

  private findPhotosInNextData(data: unknown): UnsplashPhoto[] {
    const photos: UnsplashPhoto[] = [];

    const traverse = (obj: unknown): void => {
      if (!obj || typeof obj !== 'object') return;

      if (Array.isArray(obj)) {
        for (const item of obj) {
          // Check if item looks like a photo
          if (
            item &&
            typeof item === 'object' &&
            'urls' in item &&
            'width' in item &&
            'height' in item
          ) {
            photos.push(item as UnsplashPhoto);
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
    return photos;
  }
}
